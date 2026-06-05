package service

import (
	"math"
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

	schedulerFailureRateMinSamples = 5
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
	entry.lastFailureReason = ""
	entry.updatedAt = time.Now()
}

func (s *accountSchedulerHealthStats) clear(accountID int64, model, endpoint string) {
	if s == nil || accountID <= 0 {
		return
	}
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	if _, deleted := s.entries.LoadAndDelete(key); deleted {
		s.count.Add(-1)
	}
}

func (s *accountSchedulerHealthStats) reportNeutral(accountID int64, model, endpoint string) {
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	if s == nil || accountID <= 0 {
		return
	}
	value, ok := s.entries.Load(key)
	if !ok {
		return
	}
	entry, _ := value.(*accountSchedulerHealthEntry)
	if entry == nil {
		return
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.circuitState == schedulerCircuitHalfOpen {
		entry.halfOpenInFlight = false
		entry.updatedAt = time.Now()
	}
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
	now := time.Now()
	entry.updatedAt = now

	recentFailureRate := entry.recentFailureRate()
	recentSampleCount := entry.recentSampleCount()
	shouldOpen := entry.consecutiveFailure >= 3 ||
		(recentSampleCount >= schedulerFailureRateMinSamples && recentFailureRate > 0.20)
	if category == "auth" ||
		category == "forbidden" ||
		category == "balance" ||
		category == "rate_limit" ||
		category == "transient" ||
		category == "transient_transport" ||
		category == "transient_timeout" ||
		category == "compact_bad_output" ||
		category == "model_unsupported" {
		shouldOpen = true
	}
	if shouldOpen {
		if cooldown <= 0 {
			cooldown = schedulerCooldownForCategory(category, nil)
		}
		newUntil := now.Add(cooldown)
		if entry.circuitState == schedulerCircuitOpen && entry.cooldownUntil.After(now) {
			return
		}
		entry.circuitState = schedulerCircuitOpen
		entry.cooldownUntil = newUntil
		entry.halfOpenInFlight = false
	}
}

func (s *accountSchedulerHealthStats) tryBeginHalfOpenProbe(accountID int64, model, endpoint string) bool {
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	if s == nil || accountID <= 0 {
		return true
	}
	value, ok := s.entries.Load(key)
	if !ok {
		return true
	}
	entry, _ := value.(*accountSchedulerHealthEntry)
	if entry == nil {
		return true
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	if entry.circuitState == schedulerCircuitOpen && !entry.cooldownUntil.IsZero() && now.After(entry.cooldownUntil) {
		entry.circuitState = schedulerCircuitHalfOpen
		entry.halfOpenInFlight = false
	}
	if entry.circuitState == schedulerCircuitOpen {
		return false
	}
	if entry.circuitState != schedulerCircuitHalfOpen {
		return true
	}
	if entry.halfOpenInFlight {
		return false
	}
	entry.halfOpenInFlight = true
	return true
}

func (e *accountSchedulerHealthEntry) recordRecent(failed bool) {
	if len(e.recent) == 0 {
		e.recent = make([]bool, 20)
	}
	e.recent[e.recentPos] = failed
	e.recentPos = (e.recentPos + 1) % len(e.recent)
	if e.recentPos == 0 {
		e.recentFilled = true
	}
}

func (e *accountSchedulerHealthEntry) recentFailureRate() float64 {
	count := e.recentSampleCount()
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

func (e *accountSchedulerHealthEntry) recentSampleCount() int {
	if e == nil || len(e.recent) == 0 {
		return 0
	}
	count := len(e.recent)
	if !e.recentFilled {
		count = e.recentPos
	}
	return count
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
	if statusCode == 0 {
		return schedulerStatusZeroFailureCategory(body)
	}
	if isOpenAICompactBadOutputBody(body) {
		return "compact_bad_output"
	}
	if class := classifyOpenAIUpstreamError(statusCode, "", body); class != openAIUpstreamErrorUnknown {
		return openAIUpstreamErrorClassSchedulerCategory(class)
	}
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
	return "error"
}

func isOpenAICompactBadOutputBody(body []byte) bool {
	text := strings.ToLower(strings.TrimSpace(string(body)))
	if text == "" {
		return false
	}
	if strings.Contains(text, openAICompactBadOutputCode) {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(extractUpstreamErrorCode(body)))
	return code == openAICompactBadOutputCode
}

func schedulerStatusZeroFailureCategory(body []byte) string {
	text := strings.ToLower(strings.TrimSpace(string(body)))
	if text == "" {
		return "error"
	}
	normalized := strings.NewReplacer("_", " ", "-", " ", "\n", " ", "\r", " ", "\t", " ").Replace(text)
	normalized = strings.Join(strings.Fields(normalized), " ")
	combined := text + " " + normalized
	if containsAnySchedulerText(combined,
		"timeout",
		"deadline exceeded",
		"awaiting response headers",
		"stream data interval timeout",
	) {
		return "transient_timeout"
	}
	if containsAnySchedulerText(combined,
		"openai_request_error",
		"openai request error",
		"transport_closed",
		"transport closed",
		"account_circuit_transport_closed",
		"account circuit transport closed",
		"context canceled",
		"context cancelled",
		"connection reset by peer",
		"connection refused",
		"use of closed network connection",
		"client connection force closed",
		"clientconn.close",
		"http2:",
		"goaway",
		"dial tcp",
		"dial udp",
		"network is unreachable",
		"no such host",
		"broken pipe",
		"unexpected eof",
		" eof",
		"stream error",
		"request failed",
		"upstream connection error",
	) {
		return "transient_transport"
	}
	if strings.Contains(combined, "overload") {
		return "transient"
	}
	return "error"
}

func containsAnySchedulerText(text string, markers ...string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	for _, marker := range markers {
		marker = strings.ToLower(strings.TrimSpace(marker))
		if marker != "" && strings.Contains(text, marker) {
			return true
		}
	}
	return false
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
	case "transient_transport", "transient_timeout":
		return openAIRequestErrorCooldown
	case "compact_bad_output":
		return 30 * time.Second
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

func isAccountBetterByCurrentGroupOrder(candidate, current *Account, groupID *int64) bool {
	candidateKey, candidateOK := accountCurrentGroupOrderKey(candidate, groupID)
	currentKey, currentOK := accountCurrentGroupOrderKey(current, groupID)
	if candidateOK != currentOK {
		return candidateOK
	}
	if candidateOK && currentOK {
		if candidateKey.sortOrder != currentKey.sortOrder {
			return candidateKey.sortOrder < currentKey.sortOrder
		}
		if candidateKey.priority != currentKey.priority {
			return candidateKey.priority < currentKey.priority
		}
		return candidateKey.accountID < currentKey.accountID
	}
	if candidate == nil || current == nil {
		return candidate != nil
	}
	return candidate.ID < current.ID
}

type accountGroupOrderKey struct {
	sortOrder int
	priority  int
	accountID int64
}

func accountCurrentGroupOrderKey(account *Account, groupID *int64) (accountGroupOrderKey, bool) {
	if account == nil || groupID == nil {
		return accountGroupOrderKey{}, false
	}
	cfg := accountGroupConfigFor(account, groupID)
	if cfg.GroupID != *groupID {
		return accountGroupOrderKey{}, false
	}
	return accountGroupOrderKey{
		sortOrder: cfg.SortOrder,
		priority:  cfg.Priority,
		accountID: account.ID,
	}, true
}

func accountsContainCurrentGroupBinding(accounts []*Account, groupID *int64) bool {
	if groupID == nil {
		return false
	}
	for _, account := range accounts {
		if _, ok := accountCurrentGroupOrderKey(account, groupID); ok {
			return true
		}
	}
	return false
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
	return orderedScores
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
	if schedulerScoresUseGroupOrder(scores) {
		sortSchedulerScoresByGroupOrder(scores)
		return
	}
	sort.SliceStable(scores, func(i, j int) bool {
		a, b := scores[i], scores[j]
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		if a.Account.Priority != b.Account.Priority {
			return a.Account.Priority < b.Account.Priority
		}
		if preferOAuth && a.Account.Type != b.Account.Type {
			return a.Account.Type == AccountTypeOAuth
		}
		if a.Score != b.Score {
			return a.Score > b.Score
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

func schedulerScoresUseGroupOrder(scores []schedulerAccountScore) bool {
	for _, score := range scores {
		if score.Config.GroupID > 0 {
			return true
		}
	}
	return false
}

func sortSchedulerScoresByGroupOrder(scores []schedulerAccountScore) {
	sort.SliceStable(scores, func(i, j int) bool {
		return schedulerScoreGroupOrderLess(scores[i], scores[j])
	})
}

func schedulerScoreGroupOrderLess(a, b schedulerAccountScore) bool {
	if a.Config.GroupID > 0 && b.Config.GroupID == 0 {
		return true
	}
	if a.Config.GroupID == 0 && b.Config.GroupID > 0 {
		return false
	}
	if a.SortOrder != b.SortOrder {
		return a.SortOrder < b.SortOrder
	}
	if a.Config.Priority != b.Config.Priority {
		return a.Config.Priority < b.Config.Priority
	}
	return accountIDForSchedulerScore(a) < accountIDForSchedulerScore(b)
}

func accountIDForSchedulerScore(score schedulerAccountScore) int64 {
	if score.Account == nil {
		return 0
	}
	return score.Account.ID
}
