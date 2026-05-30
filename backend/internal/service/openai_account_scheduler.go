package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	openAIAccountScheduleLayerPreviousResponse = "previous_response_id"
	openAIAccountScheduleLayerSessionSticky    = "session_hash"
	openAIAccountScheduleLayerLoadBalance      = "load_balance"
	openAIAdvancedSchedulerSettingKey          = "openai_advanced_scheduler_enabled"
)

const (
	openAIAdvancedSchedulerSettingCacheTTL  = 5 * time.Second
	openAIAdvancedSchedulerSettingDBTimeout = 2 * time.Second
)

type cachedOpenAIAdvancedSchedulerSetting struct {
	enabled   bool
	expiresAt int64
}

var openAIAdvancedSchedulerSettingCache atomic.Value // *cachedOpenAIAdvancedSchedulerSetting
var openAIAdvancedSchedulerSettingSF singleflight.Group

type OpenAIAccountScheduleRequest struct {
	GroupID                 *int64
	SessionHash             string
	StickyAccountID         int64
	PreviousResponseID      string
	RequestedModel          string
	SchedulerEndpoint       string
	RequiredTransport       OpenAIUpstreamTransport
	RequiredCapability      OpenAIEndpointCapability
	RequiredImageCapability OpenAIImagesCapability
	RequireCompact          bool
	ExcludedIDs             map[int64]struct{}
}

type OpenAIAccountScheduleDecision struct {
	Layer               string
	StickyPreviousHit   bool
	StickySessionHit    bool
	CandidateCount      int
	TopK                int
	LatencyMs           int64
	LoadSkew            float64
	SelectedAccountID   int64
	SelectedAccountType string
}

type OpenAIAccountSchedulerMetricsSnapshot struct {
	SelectTotal              int64
	StickyPreviousHitTotal   int64
	StickySessionHitTotal    int64
	LoadBalanceSelectTotal   int64
	AccountSwitchTotal       int64
	SchedulerLatencyMsTotal  int64
	SchedulerLatencyMsAvg    float64
	StickyHitRatio           float64
	AccountSwitchRate        float64
	LoadSkewAvg              float64
	RuntimeStatsAccountCount int
}

type OpenAIAccountScheduler interface {
	Select(ctx context.Context, req OpenAIAccountScheduleRequest) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error)
	ReportResult(accountID int64, success bool, firstTokenMs *int)
	ReportSwitch()
	SnapshotMetrics() OpenAIAccountSchedulerMetricsSnapshot
}

type openAIAccountSchedulerMetrics struct {
	selectTotal            atomic.Int64
	stickyPreviousHitTotal atomic.Int64
	stickySessionHitTotal  atomic.Int64
	loadBalanceSelectTotal atomic.Int64
	accountSwitchTotal     atomic.Int64
	latencyMsTotal         atomic.Int64
	loadSkewMilliTotal     atomic.Int64
}

type openAIAccountLoadPlan struct {
	allCandidates             []openAIAccountCandidateScore
	candidates                []openAIAccountCandidateScore
	staleSnapshotCompactRetry []openAIAccountCandidateScore
	selectionOrder            []openAIAccountCandidateScore
	waitOrder                 []openAIAccountCandidateScore
	candidateCount            int
	topK                      int
	loadSkew                  float64
}

func (m *openAIAccountSchedulerMetrics) recordSelect(decision OpenAIAccountScheduleDecision) {
	if m == nil {
		return
	}
	m.selectTotal.Add(1)
	m.latencyMsTotal.Add(decision.LatencyMs)
	m.loadSkewMilliTotal.Add(int64(math.Round(decision.LoadSkew * 1000)))
	if decision.StickyPreviousHit {
		m.stickyPreviousHitTotal.Add(1)
	}
	if decision.StickySessionHit {
		m.stickySessionHitTotal.Add(1)
	}
	if decision.Layer == openAIAccountScheduleLayerLoadBalance {
		m.loadBalanceSelectTotal.Add(1)
	}
}

func (m *openAIAccountSchedulerMetrics) recordSwitch() {
	if m == nil {
		return
	}
	m.accountSwitchTotal.Add(1)
}

type openAIAccountRuntimeStats struct {
	accounts     sync.Map
	accountCount atomic.Int64
}

type openAIAccountRuntimeStat struct {
	errorRateEWMABits atomic.Uint64
	ttftEWMABits      atomic.Uint64
}

func newOpenAIAccountRuntimeStats() *openAIAccountRuntimeStats {
	return &openAIAccountRuntimeStats{}
}

func (s *openAIAccountRuntimeStats) loadOrCreate(accountID int64) *openAIAccountRuntimeStat {
	if value, ok := s.accounts.Load(accountID); ok {
		stat, _ := value.(*openAIAccountRuntimeStat)
		if stat != nil {
			return stat
		}
	}

	stat := &openAIAccountRuntimeStat{}
	stat.ttftEWMABits.Store(math.Float64bits(math.NaN()))
	actual, loaded := s.accounts.LoadOrStore(accountID, stat)
	if !loaded {
		s.accountCount.Add(1)
		return stat
	}
	existing, _ := actual.(*openAIAccountRuntimeStat)
	if existing != nil {
		return existing
	}
	return stat
}

func updateEWMAAtomic(target *atomic.Uint64, sample float64, alpha float64) {
	for {
		oldBits := target.Load()
		oldValue := math.Float64frombits(oldBits)
		newValue := alpha*sample + (1-alpha)*oldValue
		if target.CompareAndSwap(oldBits, math.Float64bits(newValue)) {
			return
		}
	}
}

func (s *openAIAccountRuntimeStats) report(accountID int64, success bool, firstTokenMs *int) {
	if s == nil || accountID <= 0 {
		return
	}
	const alpha = 0.2
	stat := s.loadOrCreate(accountID)

	errorSample := 1.0
	if success {
		errorSample = 0.0
	}
	updateEWMAAtomic(&stat.errorRateEWMABits, errorSample, alpha)

	if firstTokenMs != nil && *firstTokenMs > 0 {
		ttft := float64(*firstTokenMs)
		ttftBits := math.Float64bits(ttft)
		for {
			oldBits := stat.ttftEWMABits.Load()
			oldValue := math.Float64frombits(oldBits)
			if math.IsNaN(oldValue) {
				if stat.ttftEWMABits.CompareAndSwap(oldBits, ttftBits) {
					break
				}
				continue
			}
			newValue := alpha*ttft + (1-alpha)*oldValue
			if stat.ttftEWMABits.CompareAndSwap(oldBits, math.Float64bits(newValue)) {
				break
			}
		}
	}
}

func (s *openAIAccountRuntimeStats) snapshot(accountID int64) (errorRate float64, ttft float64, hasTTFT bool) {
	if s == nil || accountID <= 0 {
		return 0, 0, false
	}
	value, ok := s.accounts.Load(accountID)
	if !ok {
		return 0, 0, false
	}
	stat, _ := value.(*openAIAccountRuntimeStat)
	if stat == nil {
		return 0, 0, false
	}
	errorRate = clamp01(math.Float64frombits(stat.errorRateEWMABits.Load()))
	ttftValue := math.Float64frombits(stat.ttftEWMABits.Load())
	if math.IsNaN(ttftValue) {
		return errorRate, 0, false
	}
	return errorRate, ttftValue, true
}

func (s *openAIAccountRuntimeStats) size() int {
	if s == nil {
		return 0
	}
	return int(s.accountCount.Load())
}

type defaultOpenAIAccountScheduler struct {
	service *OpenAIGatewayService
	metrics openAIAccountSchedulerMetrics
	stats   *openAIAccountRuntimeStats
}

func newDefaultOpenAIAccountScheduler(service *OpenAIGatewayService, stats *openAIAccountRuntimeStats) OpenAIAccountScheduler {
	if stats == nil {
		stats = newOpenAIAccountRuntimeStats()
	}
	return &defaultOpenAIAccountScheduler{
		service: service,
		stats:   stats,
	}
}

func (s *defaultOpenAIAccountScheduler) Select(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	decision := OpenAIAccountScheduleDecision{}
	start := time.Now()
	defer func() {
		decision.LatencyMs = time.Since(start).Milliseconds()
		s.metrics.recordSelect(decision)
	}()

	previousResponseID := strings.TrimSpace(req.PreviousResponseID)
	if previousResponseID != "" {
		selection, err := s.service.selectAccountByPreviousResponseIDForCapability(
			ctx,
			req.GroupID,
			previousResponseID,
			req.RequestedModel,
			req.ExcludedIDs,
			req.RequiredCapability,
			req.RequireCompact,
			schedulerEndpointFromOpenAIRequest(req),
		)
		if err != nil {
			return nil, decision, err
		}
		if selection != nil && selection.Account != nil {
			if !s.isAccountTransportCompatible(selection.Account, req.RequiredTransport) {
				if selection.ReleaseFunc != nil {
					selection.ReleaseFunc()
				}
				selection = nil
			}
		}
		if selection != nil && selection.Account != nil {
			decision.Layer = openAIAccountScheduleLayerPreviousResponse
			decision.StickyPreviousHit = true
			decision.SelectedAccountID = selection.Account.ID
			decision.SelectedAccountType = selection.Account.Type
			if req.SessionHash != "" {
				_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, selection.Account.ID)
			}
			return selection, decision, nil
		}
	}

	selection, err := s.selectBySessionHash(ctx, req)
	if err != nil {
		return nil, decision, err
	}
	if selection != nil && selection.Account != nil {
		decision.Layer = openAIAccountScheduleLayerSessionSticky
		decision.StickySessionHit = true
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
		return selection, decision, nil
	}

	selection, candidateCount, topK, loadSkew, err := s.selectByLoadBalance(ctx, req)
	decision.Layer = openAIAccountScheduleLayerLoadBalance
	decision.CandidateCount = candidateCount
	decision.TopK = topK
	decision.LoadSkew = loadSkew
	if err != nil {
		return nil, decision, err
	}
	if selection != nil && selection.Account != nil {
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
	}
	return selection, decision, nil
}

func (s *defaultOpenAIAccountScheduler) selectBySessionHash(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, error) {
	sessionHash := strings.TrimSpace(req.SessionHash)
	if sessionHash == "" || s == nil || s.service == nil || s.service.cache == nil {
		return nil, nil
	}

	accountID := req.StickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.service.getStickySessionAccountID(ctx, req.GroupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil, nil
		}
	}
	if accountID <= 0 {
		return nil, nil
	}
	if req.ExcludedIDs != nil {
		if _, excluded := req.ExcludedIDs[accountID]; excluded {
			return nil, nil
		}
	}

	account, err := s.service.getSchedulableAccount(ctx, accountID)
	if err != nil || account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	if shouldClearStickySession(account, req.RequestedModel) || !account.IsOpenAI() || !account.IsSchedulable() {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	if !s.isAccountRequestCompatible(ctx, account, req) {
		return nil, nil
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	account = s.service.recheckSelectedOpenAIAccountFromDB(ctx, account, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
	if account == nil || !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	if !s.service.isOpenAIAccountSchedulerHealthAllowed(account.ID, req.RequestedModel, schedulerEndpointFromOpenAIRequest(req)) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}

	result, acquireErr := s.service.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
	if acquireErr == nil && result != nil && result.Acquired {
		_ = s.service.refreshStickySessionTTL(ctx, req.GroupID, sessionHash, s.service.openAIWSSessionStickyTTL())
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, nil
	}

	cfg := s.service.schedulingConfig()
	// WaitPlan.MaxConcurrency 使用 Concurrency（非 EffectiveLoadFactor），因为 WaitPlan 控制的是 Redis 实际并发槽位等待。
	if s.service.concurrencyService != nil {
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      accountID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.StickySessionWaitTimeout,
				MaxWaiting:     cfg.StickySessionMaxWaiting,
			},
		}, nil
	}
	return nil, nil
}

type openAIAccountCandidateScore struct {
	account    *Account
	loadInfo   *AccountLoadInfo
	score      float64
	errorRate  float64
	ttft       float64
	hasTTFT    bool
	sortOrder  int
	health     schedulerHealthSnapshot
	halfOpen   bool
	cooldown   bool
	cooldownAt time.Time
}

func isOpenAIAccountCandidateBetter(left openAIAccountCandidateScore, right openAIAccountCandidateScore) bool {
	if left.cooldown != right.cooldown {
		return !left.cooldown
	}
	if left.halfOpen != right.halfOpen {
		return !left.halfOpen
	}
	if left.sortOrder != right.sortOrder {
		return left.sortOrder < right.sortOrder
	}
	if left.account.Priority != right.account.Priority {
		return left.account.Priority < right.account.Priority
	}
	if left.cooldown && right.cooldown && !left.cooldownAt.Equal(right.cooldownAt) {
		if left.cooldownAt.IsZero() {
			return false
		}
		if right.cooldownAt.IsZero() {
			return true
		}
		return left.cooldownAt.Before(right.cooldownAt)
	}
	if left.score != right.score {
		return left.score > right.score
	}
	leftLoad := openAIAccountCandidateLoadInfo(left)
	rightLoad := openAIAccountCandidateLoadInfo(right)
	if leftLoad.LoadRate != rightLoad.LoadRate {
		return leftLoad.LoadRate < rightLoad.LoadRate
	}
	if leftLoad.WaitingCount != rightLoad.WaitingCount {
		return leftLoad.WaitingCount < rightLoad.WaitingCount
	}
	return left.account.ID < right.account.ID
}

func openAIAccountCandidateLoadInfo(candidate openAIAccountCandidateScore) *AccountLoadInfo {
	if candidate.loadInfo != nil {
		return candidate.loadInfo
	}
	accountID := int64(0)
	if candidate.account != nil {
		accountID = candidate.account.ID
	}
	return &AccountLoadInfo{AccountID: accountID}
}

func schedulerEndpointFromOpenAIRequest(req OpenAIAccountScheduleRequest) string {
	if endpoint := strings.TrimSpace(req.SchedulerEndpoint); endpoint != "" {
		return endpoint
	}
	if req.RequireCompact {
		return "/v1/responses/compact"
	}
	if req.RequiredImageCapability != "" {
		return "images:" + string(req.RequiredImageCapability)
	}
	switch req.RequiredCapability {
	case OpenAIEndpointCapabilityEmbeddings:
		return "/v1/embeddings"
	case OpenAIEndpointCapabilityChatCompletions:
		if req.RequiredTransport == OpenAIUpstreamTransportResponsesWebsocketV2 {
			return "/v1/responses/ws"
		}
		return "/v1/responses"
	default:
		if req.RequiredTransport == OpenAIUpstreamTransportResponsesWebsocketV2 {
			return "/v1/responses/ws"
		}
		return "/v1/responses"
	}
}

func schedulerEndpointFromOpenAIContext(ctx context.Context, requireCompact bool, requiredCapability OpenAIEndpointCapability) string {
	fallback := "/v1/responses"
	if requireCompact {
		fallback = "/v1/responses/compact"
	} else if requiredCapability == OpenAIEndpointCapabilityEmbeddings {
		fallback = "/v1/embeddings"
	}
	return schedulerEndpointFromContext(ctx, fallback)
}

func openAIAccountCandidateFromSchedulerScore(score schedulerAccountScore) openAIAccountCandidateScore {
	return openAIAccountCandidateScore{
		account:   score.Account,
		loadInfo:  score.LoadInfo,
		score:     score.Score,
		errorRate: score.Health.ErrorRate,
		ttft:      score.Health.TTFTEWMA,
		hasTTFT:   score.Health.HasTTFT,
		sortOrder: score.SortOrder,
		health:    score.Health,
		halfOpen:  score.HalfOpen,
	}
}

func openAIAccountCandidatesFromSchedulerScores(scores []schedulerAccountScore) []openAIAccountCandidateScore {
	if len(scores) == 0 {
		return nil
	}
	out := make([]openAIAccountCandidateScore, 0, len(scores))
	for _, score := range scores {
		if score.Account == nil {
			continue
		}
		out = append(out, openAIAccountCandidateFromSchedulerScore(score))
	}
	return out
}

func buildOpenAIOrderedSelectionOrder(candidates []openAIAccountCandidateScore) []openAIAccountCandidateScore {
	order := append([]openAIAccountCandidateScore(nil), candidates...)
	sort.SliceStable(order, func(i, j int) bool {
		return isOpenAIAccountCandidateBetter(order[i], order[j])
	})
	return order
}

func (s *defaultOpenAIAccountScheduler) buildOpenAIAccountLoadPlan(
	req OpenAIAccountScheduleRequest,
	filtered []*Account,
	loadMap map[int64]*AccountLoadInfo,
) openAIAccountLoadPlan {
	endpoint := schedulerEndpointFromOpenAIRequest(req)
	var health *accountSchedulerHealthStats
	if s != nil && s.service != nil {
		health = s.service.schedulerHealth
	}
	allScores := buildSchedulerAccountScores(filtered, req.GroupID, req.RequestedModel, endpoint, loadMap, health, true)
	allCandidates := openAIAccountCandidatesFromSchedulerScores(allScores)

	candidates := allCandidates
	staleSnapshotCompactRetry := make([]openAIAccountCandidateScore, 0, len(allCandidates))
	if req.RequireCompact {
		candidates = make([]openAIAccountCandidateScore, 0, len(allCandidates))
		for _, candidate := range allCandidates {
			if openAICompactSupportTier(candidate.account) == 0 {
				staleSnapshotCompactRetry = append(staleSnapshotCompactRetry, candidate)
				continue
			}
			candidates = append(candidates, candidate)
		}
	}

	plan := openAIAccountLoadPlan{
		allCandidates:             allCandidates,
		candidates:                candidates,
		staleSnapshotCompactRetry: staleSnapshotCompactRetry,
		candidateCount:            len(candidates),
	}
	plan.topK = len(candidates)
	if plan.topK <= 0 {
		plan.topK = len(staleSnapshotCompactRetry)
	}
	waitScores := buildSchedulerAccountWaitScores(filtered, req.GroupID, req.RequestedModel, endpoint, loadMap, health)
	waitCandidates := openAIAccountCandidatesFromSchedulerScores(waitScores)
	if len(candidates) == 0 {
		plan.selectionOrder = s.buildOpenAISelectionOrder(req, plan)
		waitPlan := plan
		waitPlan.candidates = waitCandidates
		waitPlan.candidateCount = len(waitCandidates)
		waitPlan.staleSnapshotCompactRetry = nil
		if req.RequireCompact {
			waitPlan.candidates, waitPlan.staleSnapshotCompactRetry = splitOpenAICompactCandidates(waitCandidates)
			waitPlan.candidateCount = len(waitPlan.candidates)
		}
		waitPlan.topK = len(waitPlan.candidates)
		if waitPlan.topK <= 0 {
			waitPlan.topK = len(waitPlan.staleSnapshotCompactRetry)
		}
		plan.waitOrder = s.buildOpenAISelectionOrder(req, waitPlan)
		return plan
	}

	loadRateSum := 0.0
	loadRateSumSquares := 0.0
	for _, candidate := range candidates {
		loadInfo := openAIAccountCandidateLoadInfo(candidate)
		loadRate := float64(loadInfo.LoadRate)
		loadRateSum += loadRate
		loadRateSumSquares += loadRate * loadRate
	}
	plan.loadSkew = calcLoadSkewByMoments(loadRateSum, loadRateSumSquares, len(candidates))
	plan.candidates = candidates

	plan.topK = len(candidates)

	plan.selectionOrder = s.buildOpenAISelectionOrder(req, plan)
	waitPlan := plan
	waitPlan.candidates = waitCandidates
	waitPlan.candidateCount = len(waitCandidates)
	waitPlan.staleSnapshotCompactRetry = nil
	if req.RequireCompact {
		waitPlan.candidates, waitPlan.staleSnapshotCompactRetry = splitOpenAICompactCandidates(waitCandidates)
		waitPlan.candidateCount = len(waitPlan.candidates)
	}
	waitPlan.topK = len(waitPlan.candidates)
	if waitPlan.topK <= 0 {
		waitPlan.topK = len(waitPlan.staleSnapshotCompactRetry)
	}
	plan.waitOrder = s.buildOpenAISelectionOrder(req, waitPlan)
	return plan
}

func (s *defaultOpenAIAccountScheduler) buildOpenAICooldownFallbackOrder(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
) ([]openAIAccountCandidateScore, int) {
	if s == nil || s.service == nil {
		return nil, 0
	}
	accounts, err := s.service.listOpenAICooldownFallbackAccounts(ctx, req.GroupID)
	if err != nil || len(accounts) == 0 {
		return nil, 0
	}
	endpoint := schedulerEndpointFromOpenAIRequest(req)
	loadReq := make([]AccountWithConcurrency, 0, len(accounts))
	candidates := make([]openAIAccountCandidateScore, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if req.ExcludedIDs != nil {
			if _, excluded := req.ExcludedIDs[account.ID]; excluded {
				continue
			}
		}
		if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
			continue
		}
		if !s.isCooldownFallbackCandidate(ctx, account, req) {
			continue
		}
		cooldownUntil, ok := s.cooldownFallbackUntil(account, req, endpoint)
		if !ok {
			continue
		}
		cfg := accountGroupConfigFor(account, req.GroupID)
		candidates = append(candidates, openAIAccountCandidateScore{
			account:    account,
			loadInfo:   &AccountLoadInfo{AccountID: account.ID},
			score:      0.01,
			sortOrder:  cfg.SortOrder,
			cooldown:   true,
			cooldownAt: cooldownUntil,
		})
		loadReq = append(loadReq, AccountWithConcurrency{
			ID:             account.ID,
			MaxConcurrency: account.EffectiveLoadFactor(),
		})
	}
	if len(candidates) == 0 {
		return nil, 0
	}
	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil && len(loadReq) > 0 {
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		}
	}
	for i := range candidates {
		if loadInfo := loadMap[candidates[i].account.ID]; loadInfo != nil {
			candidates[i].loadInfo = loadInfo
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return isOpenAIAccountCandidateBetter(candidates[i], candidates[j])
	})
	return candidates, len(candidates)
}

func (s *defaultOpenAIAccountScheduler) trySelectOpenAICooldownFallback(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
	allowWaitPlan bool,
) (*AccountSelectionResult, int, bool, error) {
	order, count := s.buildOpenAICooldownFallbackOrder(ctx, req, schedGroup)
	if count == 0 {
		return nil, 0, false, nil
	}

	result, compactBlocked, acquireErr := s.tryAcquireOpenAISelectionOrder(ctx, req, order, true)
	if acquireErr != nil || result != nil || !allowWaitPlan || s.service.concurrencyService == nil {
		return result, count, compactBlocked, acquireErr
	}

	cfg := s.service.schedulingConfig()
	for _, candidate := range order {
		fresh := s.service.resolveFreshOpenAIAccountIgnoringCooldown(ctx, candidate.account, req.RequestedModel, false, req.RequiredCapability)
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !s.isAccountRequestCompatibleIgnoringCooldown(ctx, fresh, req) {
			continue
		}
		fresh = s.service.recheckSelectedOpenAIAccountFromDBIgnoringCooldown(ctx, fresh, req.RequestedModel, false, req.RequiredCapability)
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !s.isAccountRequestCompatibleIgnoringCooldown(ctx, fresh, req) {
			continue
		}
		if req.RequireCompact && openAICompactSupportTier(fresh) == 0 {
			compactBlocked = true
			continue
		}
		return &AccountSelectionResult{
			Account: fresh,
			WaitPlan: &AccountWaitPlan{
				AccountID:      fresh.ID,
				MaxConcurrency: fresh.Concurrency,
				Timeout:        cfg.FallbackWaitTimeout,
				MaxWaiting:     cfg.FallbackMaxWaiting,
			},
		}, count, compactBlocked, nil
	}

	return nil, count, compactBlocked, nil
}

func (s *defaultOpenAIAccountScheduler) isCooldownFallbackCandidate(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	if account == nil || !account.IsOpenAI() || account.Status != StatusActive || !account.Schedulable {
		return false
	}
	now := time.Now()
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		return false
	}
	if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		return false
	}
	if req.RequestedModel != "" && !account.IsModelSupported(req.RequestedModel) {
		return false
	}
	if !accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability) {
		return false
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		return false
	}
	if req.GroupID != nil && s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
		s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
		return false
	}
	if req.RequireCompact && openAICompactSupportTier(account) == 0 {
		return false
	}
	return true
}

func (s *defaultOpenAIAccountScheduler) cooldownFallbackUntil(account *Account, req OpenAIAccountScheduleRequest, endpoint string) (time.Time, bool) {
	if account == nil {
		return time.Time{}, false
	}
	now := time.Now()
	var until time.Time
	extend := func(candidate *time.Time) {
		if candidate != nil && candidate.After(now) && (until.IsZero() || candidate.After(until)) {
			until = *candidate
		}
	}
	extend(account.RateLimitResetAt)
	extend(account.OverloadUntil)
	extend(account.TempUnschedulableUntil)
	if remaining := account.GetRateLimitRemainingTimeWithContext(context.Background(), req.RequestedModel); remaining > 0 {
		modelUntil := now.Add(remaining)
		extend(&modelUntil)
	}
	if s != nil && s.service != nil {
		if runtimeUntil, ok := s.service.openAIAccountRuntimeBlockUntil(account.ID); ok && runtimeUntil.After(now) {
			extend(&runtimeUntil)
		}
		if s.service.schedulerHealth != nil {
			snap := s.service.schedulerHealth.snapshot(account.ID, req.RequestedModel, endpoint, false)
			if snap.CircuitState == schedulerCircuitOpen && snap.CooldownUntil.After(now) {
				extend(&snap.CooldownUntil)
			}
		}
	}
	if until.IsZero() {
		return time.Time{}, false
	}
	return until, true
}

func (s *defaultOpenAIAccountScheduler) buildOpenAISelectionOrder(
	req OpenAIAccountScheduleRequest,
	plan openAIAccountLoadPlan,
) []openAIAccountCandidateScore {
	buildSelectionOrder := func(pool []openAIAccountCandidateScore) []openAIAccountCandidateScore {
		if len(pool) == 0 {
			return nil
		}
		return buildOpenAIOrderedSelectionOrder(pool)
	}

	if req.RequireCompact {
		selectionOrder := make([]openAIAccountCandidateScore, 0, len(plan.allCandidates))
		supported, unknown := splitOpenAICompactCandidates(plan.candidates)
		selectionOrder = append(selectionOrder, buildSelectionOrder(supported)...)
		selectionOrder = append(selectionOrder, buildSelectionOrder(unknown)...)
		if len(plan.staleSnapshotCompactRetry) > 0 && s.service.schedulerSnapshot != nil {
			selectionOrder = append(selectionOrder, sortOpenAICompactRetryCandidates(plan.staleSnapshotCompactRetry)...)
		}
		return selectionOrder
	}

	selectionOrder := make([]openAIAccountCandidateScore, 0, len(plan.candidates))
	selectionOrder = append(selectionOrder, buildSelectionOrder(plan.candidates)...)
	return selectionOrder
}

func splitOpenAICompactCandidates(candidates []openAIAccountCandidateScore) (supported []openAIAccountCandidateScore, unknown []openAIAccountCandidateScore) {
	for _, candidate := range candidates {
		switch openAICompactSupportTier(candidate.account) {
		case 2:
			supported = append(supported, candidate)
		case 1:
			unknown = append(unknown, candidate)
		}
	}
	return supported, unknown
}

func sortOpenAICompactRetryCandidates(pool []openAIAccountCandidateScore) []openAIAccountCandidateScore {
	if len(pool) == 0 {
		return nil
	}
	ordered := append([]openAIAccountCandidateScore(nil), pool...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return isOpenAIAccountCandidateBetter(ordered[i], ordered[j])
	})
	return ordered
}

func (s *defaultOpenAIAccountScheduler) tryAcquireOpenAISelectionOrder(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	selectionOrder []openAIAccountCandidateScore,
	allowCooldownFallback bool,
) (*AccountSelectionResult, bool, error) {
	compactBlocked := false
	for i := 0; i < len(selectionOrder); i++ {
		candidate := selectionOrder[i]
		var fresh *Account
		compatible := s.isAccountRequestCompatible
		if allowCooldownFallback {
			compatible = s.isAccountRequestCompatibleIgnoringCooldown
			fresh = s.service.resolveFreshOpenAIAccountIgnoringCooldown(ctx, candidate.account, req.RequestedModel, false, req.RequiredCapability)
		} else {
			fresh = s.service.resolveFreshSchedulableOpenAIAccount(ctx, candidate.account, req.RequestedModel, false, req.RequiredCapability)
		}
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !compatible(ctx, fresh, req) {
			continue
		}
		if allowCooldownFallback {
			fresh = s.service.recheckSelectedOpenAIAccountFromDBIgnoringCooldown(ctx, fresh, req.RequestedModel, false, req.RequiredCapability)
		} else {
			fresh = s.service.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.RequestedModel, false, req.RequiredCapability)
		}
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !compatible(ctx, fresh, req) {
			continue
		}
		if req.RequireCompact && openAICompactSupportTier(fresh) == 0 {
			compactBlocked = true
			continue
		}
		result, acquireErr := s.service.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
		if acquireErr != nil {
			return nil, compactBlocked, acquireErr
		}
		if result != nil && result.Acquired {
			if req.SessionHash != "" && !allowCooldownFallback {
				_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, fresh.ID)
			}
			return &AccountSelectionResult{
				Account:     fresh,
				Acquired:    true,
				ReleaseFunc: result.ReleaseFunc,
			}, compactBlocked, nil
		}
	}
	return nil, compactBlocked, nil
}

func (s *defaultOpenAIAccountScheduler) selectByLoadBalance(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, int, int, float64, error) {
	// require_privacy_set: 获取分组信息
	var schedGroup *Group
	if req.GroupID != nil && s.service.schedulerSnapshot != nil {
		schedGroup, _ = s.service.schedulerSnapshot.GetGroupByID(ctx, *req.GroupID)
	}

	accounts, err := s.service.listSchedulableAccounts(ctx, req.GroupID)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if len(accounts) == 0 {
		if fallback, fallbackCount, _, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
			return nil, fallbackCount, fallbackCount, 0, fallbackErr
		} else if fallback != nil {
			return fallback, fallbackCount, fallbackCount, 0, nil
		}
		return nil, 0, 0, 0, noAvailableOpenAISelectionError(req.RequestedModel, false)
	}

	filtered := make([]*Account, 0, len(accounts))
	loadReq := make([]AccountWithConcurrency, 0, len(accounts))
	for _, account := range accounts {
		if req.ExcludedIDs != nil {
			if _, excluded := req.ExcludedIDs[account.ID]; excluded {
				continue
			}
		}
		if !account.IsSchedulable() || !account.IsOpenAI() {
			continue
		}
		if s.service.isOpenAIAccountRuntimeBlocked(account) {
			continue
		}
		// require_privacy_set: 跳过 privacy 未设置的账号并标记异常
		if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
			s.service.BlockAccountScheduling(account, time.Time{}, "privacy_not_set")
			_ = s.service.accountRepo.SetError(ctx, account.ID,
				fmt.Sprintf("Privacy not set, required by group [%s]", schedGroup.Name))
			continue
		}
		if !s.isAccountRequestCompatible(ctx, account, req) {
			continue
		}
		if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
			continue
		}
		filtered = append(filtered, account)
		loadReq = append(loadReq, AccountWithConcurrency{
			ID:             account.ID,
			MaxConcurrency: account.EffectiveLoadFactor(),
		})
	}
	if len(filtered) == 0 {
		if fallback, fallbackCount, _, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
			return nil, fallbackCount, fallbackCount, 0, fallbackErr
		} else if fallback != nil {
			return fallback, fallbackCount, fallbackCount, 0, nil
		}
		return nil, 0, 0, 0, noAvailableOpenAISelectionError(req.RequestedModel, false)
	}

	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil {
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		}
	}

	plan := s.buildOpenAIAccountLoadPlan(req, filtered, loadMap)
	candidateCount := plan.candidateCount
	topK := plan.topK
	loadSkew := plan.loadSkew
	selectionOrder := plan.selectionOrder
	if req.RequireCompact && len(plan.candidates) == 0 && len(plan.staleSnapshotCompactRetry) == 0 {
		if fallback, fallbackCount, _, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
			return nil, fallbackCount, fallbackCount, loadSkew, fallbackErr
		} else if fallback != nil {
			return fallback, fallbackCount, fallbackCount, loadSkew, nil
		}
		return nil, 0, 0, 0, ErrNoAvailableCompactAccounts
	}
	if req.RequireCompact && len(selectionOrder) == 0 && s.service.schedulerSnapshot == nil {
		if fallback, fallbackCount, _, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
			return nil, fallbackCount, fallbackCount, loadSkew, fallbackErr
		} else if fallback != nil {
			return fallback, fallbackCount, fallbackCount, loadSkew, nil
		}
		return nil, candidateCount, topK, loadSkew, ErrNoAvailableCompactAccounts
	}
	if len(selectionOrder) == 0 {
		if fallback, fallbackCount, _, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
			return nil, fallbackCount, fallbackCount, loadSkew, fallbackErr
		} else if fallback != nil {
			return fallback, fallbackCount, fallbackCount, loadSkew, nil
		}
		return nil, candidateCount, topK, loadSkew, noAvailableOpenAISelectionError(req.RequestedModel, req.RequireCompact && len(plan.allCandidates) > 0)
	}

	result, compactBlocked, acquireErr := s.tryAcquireOpenAISelectionOrder(ctx, req, selectionOrder, false)
	if acquireErr != nil {
		return nil, candidateCount, topK, loadSkew, acquireErr
	}
	if result != nil {
		return result, candidateCount, topK, loadSkew, nil
	}

	if s.service.concurrencyService != nil {
		if freshLoadMap, loadErr := s.service.concurrencyService.GetAccountsLoadBatchFresh(ctx, loadReq); loadErr == nil {
			freshPlan := s.buildOpenAIAccountLoadPlan(req, filtered, freshLoadMap)
			if len(freshPlan.selectionOrder) > 0 {
				freshResult, freshCompactBlocked, freshAcquireErr := s.tryAcquireOpenAISelectionOrder(ctx, req, freshPlan.selectionOrder, false)
				if freshAcquireErr != nil {
					return nil, candidateCount, topK, loadSkew, freshAcquireErr
				}
				if freshResult != nil {
					return freshResult, freshPlan.candidateCount, freshPlan.topK, freshPlan.loadSkew, nil
				}
				compactBlocked = compactBlocked || freshCompactBlocked
				selectionOrder = freshPlan.selectionOrder
				candidateCount = freshPlan.candidateCount
				topK = freshPlan.topK
				loadSkew = freshPlan.loadSkew
			}
		}
	}

	if fallback, fallbackCount, fallbackCompactBlocked, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, false); fallbackErr != nil {
		return nil, candidateCount, topK, loadSkew, fallbackErr
	} else if fallback != nil {
		return fallback, fallbackCount, fallbackCount, loadSkew, nil
	} else {
		compactBlocked = compactBlocked || fallbackCompactBlocked
	}

	cfg := s.service.schedulingConfig()
	// WaitPlan.MaxConcurrency 使用 Concurrency（非 EffectiveLoadFactor），因为 WaitPlan 控制的是 Redis 实际并发槽位等待。
	waitOrder := plan.waitOrder
	if len(waitOrder) == 0 {
		waitOrder = selectionOrder
	}
	for _, candidate := range waitOrder {
		fresh := s.service.resolveFreshSchedulableOpenAIAccount(ctx, candidate.account, req.RequestedModel, false, req.RequiredCapability)
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !s.isAccountRequestCompatible(ctx, fresh, req) {
			continue
		}
		fresh = s.service.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.RequestedModel, false, req.RequiredCapability)
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !s.isAccountRequestCompatible(ctx, fresh, req) {
			continue
		}
		if req.RequireCompact && openAICompactSupportTier(fresh) == 0 {
			compactBlocked = true
			continue
		}
		return &AccountSelectionResult{
			Account: fresh,
			WaitPlan: &AccountWaitPlan{
				AccountID:      fresh.ID,
				MaxConcurrency: fresh.Concurrency,
				Timeout:        cfg.FallbackWaitTimeout,
				MaxWaiting:     cfg.FallbackMaxWaiting,
			},
		}, candidateCount, topK, loadSkew, nil
	}

	if fallback, fallbackCount, fallbackCompactBlocked, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
		return nil, candidateCount, topK, loadSkew, fallbackErr
	} else if fallback != nil {
		return fallback, fallbackCount, fallbackCount, loadSkew, nil
	} else {
		compactBlocked = compactBlocked || fallbackCompactBlocked
	}

	return nil, candidateCount, topK, loadSkew, noAvailableOpenAISelectionError(req.RequestedModel, compactBlocked)
}

func (s *defaultOpenAIAccountScheduler) isAccountTransportCompatible(account *Account, requiredTransport OpenAIUpstreamTransport) bool {
	if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
		return true
	}
	if s == nil || s.service == nil {
		return false
	}
	return s.service.isOpenAIAccountTransportCompatible(account, requiredTransport)
}

func (s *defaultOpenAIAccountScheduler) isAccountRequestCompatible(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	if account == nil {
		return false
	}
	if s != nil && s.service != nil && s.service.isOpenAIAccountRuntimeBlocked(account) {
		return false
	}
	if req.RequestedModel != "" && !account.IsModelSupported(req.RequestedModel) {
		return false
	}
	if req.GroupID != nil && s != nil && s.service != nil &&
		s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
		s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
		return false
	}
	return accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability)
}

func (s *defaultOpenAIAccountScheduler) isAccountRequestCompatibleIgnoringCooldown(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	if account == nil {
		return false
	}
	if req.RequestedModel != "" && !account.IsModelSupported(req.RequestedModel) {
		return false
	}
	if req.GroupID != nil && s != nil && s.service != nil &&
		s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
		s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
		return false
	}
	return accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability)
}

func (s *defaultOpenAIAccountScheduler) ReportResult(accountID int64, success bool, firstTokenMs *int) {
	if s == nil || s.stats == nil {
		return
	}
	s.stats.report(accountID, success, firstTokenMs)
}

func (s *defaultOpenAIAccountScheduler) ReportSwitch() {
	if s == nil {
		return
	}
	s.metrics.recordSwitch()
}

func (s *defaultOpenAIAccountScheduler) SnapshotMetrics() OpenAIAccountSchedulerMetricsSnapshot {
	if s == nil {
		return OpenAIAccountSchedulerMetricsSnapshot{}
	}

	selectTotal := s.metrics.selectTotal.Load()
	prevHit := s.metrics.stickyPreviousHitTotal.Load()
	sessionHit := s.metrics.stickySessionHitTotal.Load()
	switchTotal := s.metrics.accountSwitchTotal.Load()
	latencyTotal := s.metrics.latencyMsTotal.Load()
	loadSkewTotal := s.metrics.loadSkewMilliTotal.Load()

	snapshot := OpenAIAccountSchedulerMetricsSnapshot{
		SelectTotal:              selectTotal,
		StickyPreviousHitTotal:   prevHit,
		StickySessionHitTotal:    sessionHit,
		LoadBalanceSelectTotal:   s.metrics.loadBalanceSelectTotal.Load(),
		AccountSwitchTotal:       switchTotal,
		SchedulerLatencyMsTotal:  latencyTotal,
		RuntimeStatsAccountCount: s.stats.size(),
	}
	if selectTotal > 0 {
		snapshot.SchedulerLatencyMsAvg = float64(latencyTotal) / float64(selectTotal)
		snapshot.StickyHitRatio = float64(prevHit+sessionHit) / float64(selectTotal)
		snapshot.AccountSwitchRate = float64(switchTotal) / float64(selectTotal)
		snapshot.LoadSkewAvg = float64(loadSkewTotal) / 1000 / float64(selectTotal)
	}
	return snapshot
}

func (s *OpenAIGatewayService) openAIAdvancedSchedulerSettingRepo() SettingRepository {
	if s == nil || s.rateLimitService == nil || s.rateLimitService.settingService == nil {
		return nil
	}
	return s.rateLimitService.settingService.settingRepo
}

func (s *OpenAIGatewayService) isOpenAIAdvancedSchedulerEnabled(ctx context.Context) bool {
	if cached, ok := openAIAdvancedSchedulerSettingCache.Load().(*cachedOpenAIAdvancedSchedulerSetting); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.enabled
		}
	}

	result, _, _ := openAIAdvancedSchedulerSettingSF.Do(openAIAdvancedSchedulerSettingKey, func() (any, error) {
		if cached, ok := openAIAdvancedSchedulerSettingCache.Load().(*cachedOpenAIAdvancedSchedulerSetting); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.enabled, nil
			}
		}

		enabled := false
		if repo := s.openAIAdvancedSchedulerSettingRepo(); repo != nil {
			dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIAdvancedSchedulerSettingDBTimeout)
			defer cancel()

			value, err := repo.GetValue(dbCtx, openAIAdvancedSchedulerSettingKey)
			if err == nil {
				enabled = strings.EqualFold(strings.TrimSpace(value), "true")
			}
		}

		openAIAdvancedSchedulerSettingCache.Store(&cachedOpenAIAdvancedSchedulerSetting{
			enabled:   enabled,
			expiresAt: time.Now().Add(openAIAdvancedSchedulerSettingCacheTTL).UnixNano(),
		})
		return enabled, nil
	})

	enabled, _ := result.(bool)
	return enabled
}

func (s *OpenAIGatewayService) getOpenAIAccountScheduler(ctx context.Context) OpenAIAccountScheduler {
	if s == nil {
		return nil
	}
	if !s.isOpenAIAdvancedSchedulerEnabled(ctx) {
		return nil
	}
	s.openaiSchedulerOnce.Do(func() {
		if s.openaiAccountStats == nil {
			s.openaiAccountStats = newOpenAIAccountRuntimeStats()
		}
		if s.openaiScheduler == nil {
			s.openaiScheduler = newDefaultOpenAIAccountScheduler(s, s.openaiAccountStats)
		}
	})
	return s.openaiScheduler
}

func resetOpenAIAdvancedSchedulerSettingCacheForTest() {
	openAIAdvancedSchedulerSettingCache = atomic.Value{}
	openAIAdvancedSchedulerSettingSF = singleflight.Group{}
}

func (s *OpenAIGatewayService) SelectAccountWithScheduler(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requireCompact bool,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	return s.selectAccountWithScheduler(ctx, groupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport, "", "", requireCompact)
}

func (s *OpenAIGatewayService) SelectAccountWithSchedulerForCapability(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requiredCapability OpenAIEndpointCapability,
	requireCompact bool,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	return s.selectAccountWithScheduler(ctx, groupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport, requiredCapability, "", requireCompact)
}

func (s *OpenAIGatewayService) SelectAccountWithSchedulerForImages(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredCapability OpenAIImagesCapability,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	selection, decision, err := s.selectAccountWithScheduler(ctx, groupID, "", sessionHash, requestedModel, excludedIDs, OpenAIUpstreamTransportHTTPSSE, "", requiredCapability, false)
	if err == nil && selection != nil && selection.Account != nil {
		return selection, decision, nil
	}
	// 如果要求 native 能力（如指定了模型）但没有可用的 APIKey 账号，回退到 basic（OAuth 账号）
	if requiredCapability == OpenAIImagesCapabilityNative {
		return s.selectAccountWithScheduler(ctx, groupID, "", sessionHash, requestedModel, excludedIDs, OpenAIUpstreamTransportHTTPSSE, "", OpenAIImagesCapabilityBasic, false)
	}
	return selection, decision, err
}

func (s *OpenAIGatewayService) selectAccountWithScheduler(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requiredCapability OpenAIEndpointCapability,
	requiredImageCapability OpenAIImagesCapability,
	requireCompact bool,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	decision := OpenAIAccountScheduleDecision{}
	scheduler := s.getOpenAIAccountScheduler(ctx)
	if scheduler == nil {
		decision.Layer = openAIAccountScheduleLayerLoadBalance
		if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
			effectiveExcludedIDs := cloneExcludedAccountIDs(excludedIDs)
			for {
				selection, err := s.selectAccountWithLoadAwareness(ctx, groupID, sessionHash, requestedModel, effectiveExcludedIDs, requireCompact, requiredCapability)
				if err != nil {
					return nil, decision, err
				}
				if selection == nil || selection.Account == nil {
					return selection, decision, nil
				}
				if accountSupportsOpenAICapabilities(selection.Account, requiredCapability, requiredImageCapability) {
					return selection, decision, nil
				}
				if selection.ReleaseFunc != nil {
					selection.ReleaseFunc()
				}
				if effectiveExcludedIDs == nil {
					effectiveExcludedIDs = make(map[int64]struct{})
				}
				if _, exists := effectiveExcludedIDs[selection.Account.ID]; exists {
					return nil, decision, ErrNoAvailableAccounts
				}
				effectiveExcludedIDs[selection.Account.ID] = struct{}{}
			}
		}

		effectiveExcludedIDs := cloneExcludedAccountIDs(excludedIDs)
		for {
			selection, err := s.selectAccountWithLoadAwareness(ctx, groupID, sessionHash, requestedModel, effectiveExcludedIDs, requireCompact, requiredCapability)
			if err != nil {
				return nil, decision, err
			}
			if selection == nil || selection.Account == nil {
				return selection, decision, nil
			}
			if s.isOpenAIAccountTransportCompatible(selection.Account, requiredTransport) &&
				accountSupportsOpenAICapabilities(selection.Account, requiredCapability, requiredImageCapability) {
				return selection, decision, nil
			}
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			if effectiveExcludedIDs == nil {
				effectiveExcludedIDs = make(map[int64]struct{})
			}
			if _, exists := effectiveExcludedIDs[selection.Account.ID]; exists {
				return nil, decision, ErrNoAvailableAccounts
			}
			effectiveExcludedIDs[selection.Account.ID] = struct{}{}
		}
	}

	if s.checkChannelPricingRestriction(ctx, groupID, requestedModel) {
		slog.Warn("channel pricing restriction blocked request",
			"group_id", derefGroupID(groupID),
			"model", requestedModel)
		return nil, decision, fmt.Errorf("%w supporting model: %s (channel pricing restriction)", ErrNoAvailableAccounts, requestedModel)
	}

	var stickyAccountID int64
	if sessionHash != "" && s.cache != nil {
		if accountID, err := s.getStickySessionAccountID(ctx, groupID, sessionHash); err == nil && accountID > 0 {
			stickyAccountID = accountID
		}
	}

	req := OpenAIAccountScheduleRequest{
		GroupID:                 groupID,
		SessionHash:             sessionHash,
		StickyAccountID:         stickyAccountID,
		PreviousResponseID:      previousResponseID,
		RequestedModel:          requestedModel,
		RequiredTransport:       requiredTransport,
		RequiredCapability:      requiredCapability,
		RequiredImageCapability: requiredImageCapability,
		RequireCompact:          requireCompact,
		ExcludedIDs:             excludedIDs,
	}
	req.SchedulerEndpoint = schedulerEndpointFromContext(ctx, schedulerEndpointFromOpenAIRequest(req))
	return scheduler.Select(ctx, req)
}

func accountSupportsOpenAICapabilities(account *Account, requiredCapability OpenAIEndpointCapability, requiredImageCapability OpenAIImagesCapability) bool {
	if account == nil {
		return false
	}
	return account.SupportsOpenAIEndpointCapability(requiredCapability) &&
		account.SupportsOpenAIImageCapability(requiredImageCapability)
}

func cloneExcludedAccountIDs(excludedIDs map[int64]struct{}) map[int64]struct{} {
	if len(excludedIDs) == 0 {
		return nil
	}
	cloned := make(map[int64]struct{}, len(excludedIDs))
	for id := range excludedIDs {
		cloned[id] = struct{}{}
	}
	return cloned
}

func (s *OpenAIGatewayService) isOpenAIAccountTransportCompatible(account *Account, requiredTransport OpenAIUpstreamTransport) bool {
	if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
		return true
	}
	if s == nil || account == nil {
		return false
	}
	return s.getOpenAIWSProtocolResolver().Resolve(account).Transport == requiredTransport
}

func (s *OpenAIGatewayService) ReportOpenAIAccountScheduleResult(accountID int64, success bool, firstTokenMs *int) {
	s.ReportOpenAIAccountScheduleResultForRequest(accountID, "", "", success, firstTokenMs)
}

func (s *OpenAIGatewayService) ReportOpenAIAccountScheduleResultForRequest(accountID int64, model, endpoint string, success bool, firstTokenMs *int) {
	if s == nil || accountID <= 0 {
		return
	}
	if s.schedulerHealth != nil {
		if success {
			s.schedulerHealth.reportSuccess(accountID, model, endpoint, firstTokenMs)
		} else {
			category := schedulerFailureCategory(0, nil)
			s.schedulerHealth.reportFailure(accountID, model, endpoint, category, schedulerCooldownForCategory(category, nil))
		}
	}
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler != nil {
		scheduler.ReportResult(accountID, success, firstTokenMs)
	}
}

func (s *OpenAIGatewayService) ReportOpenAIAccountScheduleFailure(accountID int64, model, endpoint string, failoverErr *UpstreamFailoverError) {
	if s == nil || accountID <= 0 {
		return
	}
	statusCode := 0
	var body []byte
	var headers http.Header
	if failoverErr != nil {
		statusCode = failoverErr.StatusCode
		body = failoverErr.ResponseBody
		headers = failoverErr.ResponseHeaders
	}
	category := schedulerFailureCategory(statusCode, body)
	cooldown := schedulerCooldownForCategory(category, headers)
	if s.schedulerHealth != nil {
		s.schedulerHealth.reportFailure(accountID, model, endpoint, category, cooldown)
	}
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler != nil {
		scheduler.ReportResult(accountID, false, nil)
	}
}

func (s *OpenAIGatewayService) RecordOpenAIAccountSwitch() {
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler == nil {
		return
	}
	scheduler.ReportSwitch()
}

func (s *OpenAIGatewayService) MaxOpenAIAccountSwitches(ctx context.Context, configured int, groupID *int64) int {
	if configured <= 0 {
		configured = 3
	}
	if s == nil || !s.isOpenAIAdvancedSchedulerEnabled(ctx) {
		return configured
	}
	accounts, err := s.listOpenAICooldownFallbackAccounts(ctx, groupID)
	if err == nil && len(accounts) > configured {
		return len(accounts)
	}
	if configured < 50 {
		return 50
	}
	return configured
}

func (s *OpenAIGatewayService) SnapshotOpenAIAccountSchedulerMetrics() OpenAIAccountSchedulerMetricsSnapshot {
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler == nil {
		return OpenAIAccountSchedulerMetricsSnapshot{}
	}
	return scheduler.SnapshotMetrics()
}

func (s *OpenAIGatewayService) openAIWSSessionStickyTTL() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.StickySessionTTLSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.StickySessionTTLSeconds) * time.Second
	}
	return openaiStickySessionTTL
}

func (s *OpenAIGatewayService) openAIWSSchedulerWeights() GatewayOpenAIWSSchedulerScoreWeightsView {
	if s != nil && s.cfg != nil {
		return GatewayOpenAIWSSchedulerScoreWeightsView{
			Priority:  s.cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority,
			Load:      s.cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load,
			Queue:     s.cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue,
			ErrorRate: s.cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate,
			TTFT:      s.cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT,
		}
	}
	return GatewayOpenAIWSSchedulerScoreWeightsView{
		Priority:  1.0,
		Load:      1.0,
		Queue:     0.7,
		ErrorRate: 0.8,
		TTFT:      0.5,
	}
}

type GatewayOpenAIWSSchedulerScoreWeightsView struct {
	Priority  float64
	Load      float64
	Queue     float64
	ErrorRate float64
	TTFT      float64
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func calcLoadSkewByMoments(sum float64, sumSquares float64, count int) float64 {
	if count <= 1 {
		return 0
	}
	mean := sum / float64(count)
	variance := sumSquares/float64(count) - mean*mean
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}
