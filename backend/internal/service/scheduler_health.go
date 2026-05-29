package service

import (
	"hash/fnv"
	"math"
	mathrand "math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	schedulerCircuitClosed   = "closed"
	schedulerCircuitOpen     = "open"
	schedulerCircuitHalfOpen = "half_open"

	defaultSchedulerEndpoint = "default"
	defaultSchedulerModel    = "default"
)

type accountSchedulerHealthStats struct {
	entries sync.Map
	count   atomic.Int64
}

type accountSchedulerHealthKey struct {
	AccountID int64
	Model     string
	Endpoint  string
}

type accountSchedulerHealthEntry struct {
	mu                 sync.Mutex
	successEWMA        float64
	errorEWMA          float64
	ttftEWMA           float64
	hasTTFT            bool
	consecutiveFailure int
	recent             []bool
	recentPos          int
	recentFilled       bool
	circuitState       string
	cooldownUntil      time.Time
	halfOpenInFlight   bool
	lastFailureReason  string
	updatedAt          time.Time
}

type schedulerHealthSnapshot struct {
	Key               accountSchedulerHealthKey
	HealthScore       float64
	ModelScore        float64
	LatencyScore      float64
	ErrorRate         float64
	TTFTEWMA          float64
	HasTTFT           bool
	CircuitState      string
	CooldownUntil     time.Time
	HalfOpenProbe     bool
	ConsecutiveFailed int
	LastFailureReason string
}

type schedulerAccountScore struct {
	Account    *Account
	Config     AccountGroup
	LoadInfo   *AccountLoadInfo
	Health     schedulerHealthSnapshot
	Score      float64
	Role       string
	HalfOpen   bool
	SortOrder  int
	BaseWeight int
}

func newAccountSchedulerHealthStats() *accountSchedulerHealthStats {
	return &accountSchedulerHealthStats{}
}

func normalizeSchedulerDimension(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}
	return value
}

func makeAccountSchedulerHealthKey(accountID int64, model, endpoint string) accountSchedulerHealthKey {
	return accountSchedulerHealthKey{
		AccountID: accountID,
		Model:     normalizeSchedulerDimension(model, defaultSchedulerModel),
		Endpoint:  normalizeSchedulerDimension(endpoint, defaultSchedulerEndpoint),
	}
}

func (s *accountSchedulerHealthStats) loadOrCreate(key accountSchedulerHealthKey) *accountSchedulerHealthEntry {
	if s == nil || key.AccountID <= 0 {
		return nil
	}
	if value, ok := s.entries.Load(key); ok {
		entry, _ := value.(*accountSchedulerHealthEntry)
		if entry != nil {
			return entry
		}
	}
	entry := &accountSchedulerHealthEntry{
		successEWMA:  1,
		circuitState: schedulerCircuitClosed,
		recent:       make([]bool, 20),
	}
	actual, loaded := s.entries.LoadOrStore(key, entry)
	if !loaded {
		s.count.Add(1)
		return entry
	}
	existing, _ := actual.(*accountSchedulerHealthEntry)
	if existing != nil {
		return existing
	}
	return entry
}

func (s *accountSchedulerHealthStats) size() int {
	if s == nil {
		return 0
	}
	return int(s.count.Load())
}

func (s *accountSchedulerHealthStats) snapshot(accountID int64, model, endpoint string, allowHalfOpen bool) schedulerHealthSnapshot {
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	snap := schedulerHealthSnapshot{
		Key:           key,
		HealthScore:   1,
		ModelScore:    1,
		LatencyScore:  1,
		CircuitState:  schedulerCircuitClosed,
		HalfOpenProbe: false,
	}
	if s == nil || accountID <= 0 {
		return snap
	}
	value, ok := s.entries.Load(key)
	if !ok {
		return snap
	}
	entry, _ := value.(*accountSchedulerHealthEntry)
	if entry == nil {
		return snap
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	if entry.circuitState == schedulerCircuitOpen && !entry.cooldownUntil.IsZero() && now.After(entry.cooldownUntil) {
		entry.circuitState = schedulerCircuitHalfOpen
		entry.halfOpenInFlight = false
	}
	if entry.circuitState == schedulerCircuitHalfOpen {
		snap.CircuitState = schedulerCircuitHalfOpen
		snap.CooldownUntil = entry.cooldownUntil
		snap.HealthScore = clampSchedulerScore(entry.successEWMA)
		snap.ModelScore = 1 - clamp01(entry.errorEWMA)
		snap.LatencyScore = latencyScoreFromTTFT(entry.ttftEWMA, entry.hasTTFT)
		snap.ErrorRate = clamp01(entry.errorEWMA)
		snap.TTFTEWMA = entry.ttftEWMA
		snap.HasTTFT = entry.hasTTFT
		snap.ConsecutiveFailed = entry.consecutiveFailure
		snap.LastFailureReason = entry.lastFailureReason
		if allowHalfOpen && !entry.halfOpenInFlight {
			entry.halfOpenInFlight = true
			snap.HalfOpenProbe = true
		}
		return snap
	}
	if entry.circuitState == schedulerCircuitOpen && entry.cooldownUntil.After(now) {
		snap.CircuitState = schedulerCircuitOpen
		snap.CooldownUntil = entry.cooldownUntil
		snap.HealthScore = 0
		snap.ModelScore = 0
		snap.LatencyScore = latencyScoreFromTTFT(entry.ttftEWMA, entry.hasTTFT)
		snap.ErrorRate = clamp01(entry.errorEWMA)
		snap.TTFTEWMA = entry.ttftEWMA
		snap.HasTTFT = entry.hasTTFT
		snap.ConsecutiveFailed = entry.consecutiveFailure
		snap.LastFailureReason = entry.lastFailureReason
		return snap
	}
	if entry.circuitState == schedulerCircuitOpen {
		entry.circuitState = schedulerCircuitClosed
		entry.cooldownUntil = time.Time{}
		entry.halfOpenInFlight = false
	}

	snap.CircuitState = entry.circuitState
	snap.HealthScore = clampSchedulerScore(entry.successEWMA)
	snap.ModelScore = 1 - clamp01(entry.errorEWMA)
	snap.LatencyScore = latencyScoreFromTTFT(entry.ttftEWMA, entry.hasTTFT)
	snap.ErrorRate = clamp01(entry.errorEWMA)
	snap.TTFTEWMA = entry.ttftEWMA
	snap.HasTTFT = entry.hasTTFT
	snap.ConsecutiveFailed = entry.consecutiveFailure
	snap.LastFailureReason = entry.lastFailureReason
	return snap
}

func (s *accountSchedulerHealthStats) reportSuccess(accountID int64, model, endpoint string, firstTokenMs *int) {
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	entry := s.loadOrCreate(key)
	if entry == nil {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	entry.successEWMA = ewma(entry.successEWMA, 1, 0.12)
	entry.errorEWMA = ewma(entry.errorEWMA, 0, 0.12)
	if firstTokenMs != nil && *firstTokenMs > 0 {
		ttft := float64(*firstTokenMs)
		if !entry.hasTTFT {
			entry.ttftEWMA = ttft
			entry.hasTTFT = true
		} else {
			entry.ttftEWMA = ewma(entry.ttftEWMA, ttft, 0.2)
		}
	}
	entry.consecutiveFailure = 0
	entry.recordRecent(false)
	entry.circuitState = schedulerCircuitClosed
	entry.cooldownUntil = time.Time{}
	entry.halfOpenInFlight = false
	entry.updatedAt = time.Now()
}

func (s *accountSchedulerHealthStats) reportFailure(accountID int64, model, endpoint string, category string, cooldown time.Duration) {
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	entry := s.loadOrCreate(key)
	if entry == nil {
		return
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.successEWMA = ewma(entry.successEWMA, 0, 0.45)
	entry.errorEWMA = ewma(entry.errorEWMA, 1, 0.45)
	entry.consecutiveFailure++
	entry.recordRecent(true)
	entry.lastFailureReason = strings.TrimSpace(category)
	entry.updatedAt = time.Now()

	recentFailureRate := entry.recentFailureRate()
	shouldOpen := entry.consecutiveFailure >= 3 || recentFailureRate > 0.20
	if category == "auth" || category == "forbidden" || category == "balance" {
		shouldOpen = true
	}
	if shouldOpen {
		if cooldown <= 0 {
			cooldown = schedulerCooldownForCategory(category, nil)
		}
		entry.circuitState = schedulerCircuitOpen
		entry.cooldownUntil = time.Now().Add(cooldown)
		entry.halfOpenInFlight = false
	}
}

func (e *accountSchedulerHealthEntry) recordRecent(failed bool) {
	if e.recent == nil || len(e.recent) == 0 {
		e.recent = make([]bool, 20)
	}
	e.recent[e.recentPos] = failed
	e.recentPos = (e.recentPos + 1) % len(e.recent)
	if e.recentPos == 0 {
		e.recentFilled = true
	}
}

func (e *accountSchedulerHealthEntry) recentFailureRate() float64 {
	if e == nil || len(e.recent) == 0 {
		return 0
	}
	count := len(e.recent)
	if !e.recentFilled {
		count = e.recentPos
	}
	if count == 0 {
		return 0
	}
	failures := 0
	for i := 0; i < count; i++ {
		if e.recent[i] {
			failures++
		}
	}
	return float64(failures) / float64(count)
}

func ewma(oldValue, sample, alpha float64) float64 {
	if math.IsNaN(oldValue) || math.IsInf(oldValue, 0) {
		return sample
	}
	return alpha*sample + (1-alpha)*oldValue
}

func clampSchedulerScore(v float64) float64 {
	if v <= 0 {
		return 0.05
	}
	return clamp01(v)
}

func latencyScoreFromTTFT(ttft float64, hasTTFT bool) float64 {
	if !hasTTFT || ttft <= 0 {
		return 1
	}
	switch {
	case ttft <= 1500:
		return 1
	case ttft >= 20000:
		return 0.2
	default:
		return 1 - ((ttft - 1500) / (20000 - 1500) * 0.8)
	}
}

func schedulerFailureCategory(statusCode int, body []byte) string {
	text := strings.ToLower(string(body))
	switch statusCode {
	case http.StatusUnauthorized:
		return "auth"
	case http.StatusForbidden:
		if strings.Contains(text, "balance") || strings.Contains(text, "quota") || strings.Contains(text, "insufficient") {
			return "balance"
		}
		return "forbidden"
	case http.StatusTooManyRequests:
		return "rate_limit"
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "transient"
	}
	if statusCode >= 500 {
		return "transient"
	}
	if strings.Contains(text, "validate api key failed") || strings.Contains(text, "invalid api key") || strings.Contains(text, "invalid_api_key") {
		return "auth"
	}
	if strings.Contains(text, "insufficient account balance") || strings.Contains(text, "insufficient balance") {
		return "balance"
	}
	if strings.Contains(text, "rate limit") || strings.Contains(text, "too many requests") {
		return "rate_limit"
	}
	if strings.Contains(text, "request failed") || strings.Contains(text, "timeout") || strings.Contains(text, "overload") {
		return "transient"
	}
	if statusCode > 0 {
		return "error"
	}
	return "unknown"
}

func schedulerCooldownForCategory(category string, headers http.Header) time.Duration {
	switch category {
	case "auth":
		return time.Hour
	case "forbidden", "balance":
		return 6 * time.Hour
	case "rate_limit":
		if headers != nil {
			if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
					return time.Duration(seconds) * time.Second
				}
				if t, err := http.ParseTime(retryAfter); err == nil {
					if d := time.Until(t); d > 0 {
						return d
					}
				}
			}
		}
		return time.Minute
	case "transient", "unknown":
		return 90 * time.Second
	default:
		return 2 * time.Minute
	}
}

func accountGroupConfigFor(account *Account, groupID *int64) AccountGroup {
	if account == nil {
		return AccountGroup{Role: AccountGroupRolePrimary, Weight: 100, SortOrder: 50, Priority: 50}
	}
	if groupID != nil {
		for _, ag := range account.AccountGroups {
			if ag.GroupID == *groupID {
				return normalizeAccountGroupConfig(ag, account)
			}
		}
	}
	priority := account.Priority
	return AccountGroup{
		AccountID: account.ID,
		Priority:  account.Priority,
		Role:      AccountGroupRolePrimary,
		Weight:    100,
		SortOrder: priority,
	}
}

func normalizeAccountGroupConfig(ag AccountGroup, account *Account) AccountGroup {
	if ag.AccountID == 0 && account != nil {
		ag.AccountID = account.ID
	}
	ag.Role = ag.NormalizedRole()
	ag.Weight = ag.EffectiveWeight()
	ag.SortOrder = ag.EffectiveSortOrder()
	if ag.Priority == 0 && account != nil {
		ag.Priority = account.Priority
	}
	if ag.SortOrder == 0 {
		ag.SortOrder = ag.Priority
	}
	return ag
}

func schedulerRoleRank(role string) int {
	if role == AccountGroupRoleBackup {
		return 1
	}
	return 0
}

func splitSchedulerScoresByRole(scores []schedulerAccountScore) (primary []schedulerAccountScore, backup []schedulerAccountScore) {
	for _, score := range scores {
		if score.Role == AccountGroupRoleBackup {
			backup = append(backup, score)
		} else {
			primary = append(primary, score)
		}
	}
	return primary, backup
}

func buildSchedulerAccountScores(
	accounts []*Account,
	groupID *int64,
	model string,
	endpoint string,
	loadMap map[int64]*AccountLoadInfo,
	health *accountSchedulerHealthStats,
	allowHalfOpen bool,
) []schedulerAccountScore {
	return buildSchedulerAccountScoresWithOptions(accounts, groupID, model, endpoint, loadMap, health, allowHalfOpen, false)
}

func buildSchedulerAccountWaitScores(
	accounts []*Account,
	groupID *int64,
	model string,
	endpoint string,
	loadMap map[int64]*AccountLoadInfo,
	health *accountSchedulerHealthStats,
) []schedulerAccountScore {
	return buildSchedulerAccountScoresWithOptions(accounts, groupID, model, endpoint, loadMap, health, false, true)
}

func buildSchedulerAccountScoresWithOptions(
	accounts []*Account,
	groupID *int64,
	model string,
	endpoint string,
	loadMap map[int64]*AccountLoadInfo,
	health *accountSchedulerHealthStats,
	allowHalfOpen bool,
	includeFullLoad bool,
) []schedulerAccountScore {
	scores := make([]schedulerAccountScore, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		cfg := accountGroupConfigFor(account, groupID)
		snap := schedulerHealthSnapshot{HealthScore: 1, ModelScore: 1, LatencyScore: 1, CircuitState: schedulerCircuitClosed}
		if health != nil {
			snap = health.snapshot(account.ID, model, endpoint, allowHalfOpen)
		}
		if snap.CircuitState == schedulerCircuitOpen {
			continue
		}
		if snap.CircuitState == schedulerCircuitHalfOpen && !snap.HalfOpenProbe {
			continue
		}
		loadInfo := loadMap[account.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: account.ID}
		}
		if loadInfo.LoadRate >= 100 && !snap.HalfOpenProbe && !includeFullLoad {
			continue
		}
		inflight := loadInfo.CurrentConcurrency
		if inflight < 0 {
			inflight = 0
		}
		baseWeight := cfg.EffectiveWeight()
		loadScore := 1 - clamp01(float64(loadInfo.LoadRate)/100.0)
		if snap.HalfOpenProbe {
			loadScore = math.Max(loadScore, 0.2)
		}
		score := float64(baseWeight) * snap.HealthScore * snap.ModelScore * snap.LatencyScore * loadScore / float64(1+inflight)
		if score <= 0 {
			score = 0.01
		}
		scores = append(scores, schedulerAccountScore{
			Account:    account,
			Config:     cfg,
			LoadInfo:   loadInfo,
			Health:     snap,
			Score:      score,
			Role:       cfg.NormalizedRole(),
			HalfOpen:   snap.HalfOpenProbe,
			SortOrder:  cfg.SortOrder,
			BaseWeight: baseWeight,
		})
	}
	return scores
}

func buildRoleAwareSchedulerOrder(scores []schedulerAccountScore, preferOAuth bool, seedParts ...string) []schedulerAccountScore {
	if len(scores) == 0 {
		return nil
	}
	orderedScores := append([]schedulerAccountScore(nil), scores...)
	sortSchedulerScores(orderedScores, preferOAuth)
	primary, backup := splitSchedulerScoresByRole(orderedScores)
	order := make([]schedulerAccountScore, 0, len(orderedScores))
	if !hasExplicitSchedulerGroupConfig(orderedScores) {
		order = append(order, primary...)
		order = append(order, backup...)
		return order
	}
	order = append(order, buildWeightedSchedulerOrder(primary, append(seedParts, AccountGroupRolePrimary)...)...)
	order = append(order, buildWeightedSchedulerOrder(backup, append(seedParts, AccountGroupRoleBackup)...)...)
	return order
}

func hasExplicitSchedulerGroupConfig(scores []schedulerAccountScore) bool {
	for _, score := range scores {
		if score.Config.GroupID > 0 && score.Config.SchedulingConfigured {
			return true
		}
	}
	return false
}

func sortSchedulerScores(scores []schedulerAccountScore, preferOAuth bool) {
	sort.SliceStable(scores, func(i, j int) bool {
		a, b := scores[i], scores[j]
		if schedulerRoleRank(a.Role) != schedulerRoleRank(b.Role) {
			return schedulerRoleRank(a.Role) < schedulerRoleRank(b.Role)
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		if a.Account.Priority != b.Account.Priority {
			return a.Account.Priority < b.Account.Priority
		}
		if preferOAuth && a.Account.Type != b.Account.Type {
			return a.Account.Type == AccountTypeOAuth
		}
		switch {
		case a.Account.LastUsedAt == nil && b.Account.LastUsedAt != nil:
			return true
		case a.Account.LastUsedAt != nil && b.Account.LastUsedAt == nil:
			return false
		case a.Account.LastUsedAt == nil && b.Account.LastUsedAt == nil:
			return a.Account.ID < b.Account.ID
		default:
			if !a.Account.LastUsedAt.Equal(*b.Account.LastUsedAt) {
				return a.Account.LastUsedAt.Before(*b.Account.LastUsedAt)
			}
			return a.Account.ID < b.Account.ID
		}
	})
}

func buildWeightedSchedulerOrder(scores []schedulerAccountScore, seedParts ...string) []schedulerAccountScore {
	if len(scores) <= 1 {
		return append([]schedulerAccountScore(nil), scores...)
	}
	pool := append([]schedulerAccountScore(nil), scores...)
	order := make([]schedulerAccountScore, 0, len(pool))
	rng := mathrand.New(mathrand.NewSource(int64(schedulerSelectionSeed(seedParts...))))
	for len(pool) > 0 {
		total := 0.0
		for _, item := range pool {
			total += math.Max(item.Score, 0.01)
		}
		idx := 0
		if total > 0 {
			r := rng.Float64() * total
			accum := 0.0
			for i, item := range pool {
				accum += math.Max(item.Score, 0.01)
				if r <= accum {
					idx = i
					break
				}
			}
		}
		order = append(order, pool[idx])
		pool = append(pool[:idx], pool[idx+1:]...)
	}
	return order
}

func schedulerSelectionSeed(parts ...string) uint64 {
	h := fnv.New64a()
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	seed := h.Sum64() ^ uint64(time.Now().UnixNano())
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	return seed
}
