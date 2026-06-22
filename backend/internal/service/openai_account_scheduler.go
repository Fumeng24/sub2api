package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
	openAIAccountScheduleLayerCacheAffinity    = "cache_affinity"
	openAIAccountScheduleLayerLoadBalance      = "load_balance"
	openAIAdvancedSchedulerSettingKey          = "openai_advanced_scheduler_enabled"
	openAIAccountWeakFallbackReason            = "same_group_soft_filter_relaxed"
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
	GroupID                           *int64
	SessionHash                       string
	CacheAffinityHash                 string
	StickyAccountID                   int64
	CacheAffinityAccountID            int64
	CacheAffinityGroup                string
	PreserveStickyBinding             bool
	PreviousResponseID                string
	RequestedModel                    string
	SchedulerEndpoint                 string
	RequiredTransport                 OpenAIUpstreamTransport
	RequiredCapability                OpenAIEndpointCapability
	RequiredImageCapability           OpenAIImagesCapability
	RequireCompact                    bool
	RequireCodexImageGenerationBridge bool
	ExcludedIDs                       map[int64]struct{}
}

type OpenAIAccountScheduleOptions struct {
	RequireCodexImageGenerationBridge bool
	AllowCompactModelMapping          bool
}

type OpenAIAccountScheduleDecision struct {
	Layer               string
	StickyPreviousHit   bool
	StickySessionHit    bool
	CacheAffinityHit    bool
	CandidateCount      int
	TopK                int
	LatencyMs           int64
	LoadSkew            float64
	SelectedAccountID   int64
	SelectedAccountType string
	Diagnostics         OpenAIAccountSelectionDiagnostics
}

type OpenAIAccountSelectionDiagnostics struct {
	Collected                                  bool
	GroupID                                    int64
	Model                                      string
	Endpoint                                   string
	RequireCompact                             bool
	CompactStrictSupportedOnly                 bool
	RequiredTransport                          string
	RequiredCapability                         string
	RequiredImageCapability                    string
	RequireCodexImageGenerationBridge          bool
	GroupBindingAccountCount                   int
	ActiveSchedulableCount                     int
	ExcludedAccountCount                       int
	AfterExcludedCount                         int
	ModelSupportedCount                        int
	EndpointSupportedCount                     int
	ImageGenerationBridgeSupportedCount        int
	CompactSupportedCount                      int
	StateAllowedCount                          int
	CircuitAllowedCount                        int
	ConcurrencySlotAllowedCount                int
	FinalCandidateCount                        int
	StateFilteredCount                         int
	CircuitFilteredCount                       int
	ConcurrencySlotFilteredCount               int
	HalfOpenFilteredCount                      int
	CompactUnsupportedCount                    int
	ImageGenerationBridgeUnsupportedCount      int
	StatusFilteredCount                        int
	TempUnschedulableFilteredCount             int
	OverloadFilteredCount                      int
	RateLimitFilteredCount                     int
	ModelRateLimitFilteredCount                int
	ChannelRestrictionFilteredCount            int
	GroupScopeFilteredCount                    int
	ExcludedAccountIDs                         []int64
	CandidateAccountIDs                        []int64
	OrderedCandidateAccountIDs                 []int64
	CacheAffinityGroupCandidateIDs             []int64
	CacheAffinityGroupFallbackAccountIDs       []int64
	ActiveSchedulableAccountIDs                []int64
	ModelUnsupportedAccountIDs                 []int64
	EndpointUnsupportedAccountIDs              []int64
	ImageGenerationBridgeUnsupportedAccountIDs []int64
	ChannelRestrictionAccountIDs               []int64
	CompactUnsupportedAccountIDs               []int64
	StateFilteredAccountIDs                    []int64
	CircuitFilteredAccountIDs                  []int64
	ConcurrencySlotFilteredAccountIDs          []int64
	HalfOpenFilteredAccountIDs                 []int64
	FilterReasonCounts                         map[string]int
	GroupBindingAccountIDs                     []int64
	AfterExcludedAccountIDs                    []int64
	ModelSupportedAccountIDs                   []int64
	EndpointSupportedAccountIDs                []int64
	ImageGenerationBridgeSupportedAccountIDs   []int64
	CompactSupportedAccountIDs                 []int64
	StateAllowedAccountIDs                     []int64
	CircuitAllowedAccountIDs                   []int64
	EarliestRetryAt                            time.Time
	EarliestRetryReason                        string
	EarliestRetryAccountID                     int64
	RetryAfterSeconds                          int
}

type OpenAIAccountSchedulerMetricsSnapshot struct {
	SelectTotal              int64   `json:"select_total"`
	StickyPreviousHitTotal   int64   `json:"sticky_previous_hit_total"`
	StickySessionHitTotal    int64   `json:"sticky_session_hit_total"`
	CacheAffinityHitTotal    int64   `json:"cache_affinity_hit_total"`
	LoadBalanceSelectTotal   int64   `json:"load_balance_select_total"`
	AccountSwitchTotal       int64   `json:"account_switch_total"`
	SchedulerLatencyMsTotal  int64   `json:"scheduler_latency_ms_total"`
	SchedulerLatencyMsAvg    float64 `json:"scheduler_latency_ms_avg"`
	StickyHitRatio           float64 `json:"sticky_hit_ratio"`
	AccountSwitchRate        float64 `json:"account_switch_rate"`
	LoadSkewAvg              float64 `json:"load_skew_avg"`
	RuntimeStatsAccountCount int     `json:"runtime_stats_account_count"`
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
	cacheAffinityHitTotal  atomic.Int64
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
	stableLowTTFT             bool
	stableLowTTFTSeq          uint64
}

type openAISelectionPlan struct {
	accounts          []*Account
	circuitAllowed    []*Account
	candidateAccounts []*Account
	loadReq           []AccountWithConcurrency
	loadMap           map[int64]*AccountLoadInfo
	loadPlan          openAIAccountLoadPlan
	diagnostics       OpenAIAccountSelectionDiagnostics
	compactBlocked    bool
	schedGroup        *Group
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
	if decision.CacheAffinityHit {
		m.cacheAffinityHitTotal.Add(1)
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

	stableLowTTFTSeq atomic.Uint64
}

type openAIStickyEscapeConfig struct {
	enabled   bool
	ttftMs    float64
	errorRate float64
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
	var schedGroupForDiagnostics *Group
	attachDiagnostics := func() {
		if decision.Diagnostics.Collected || s == nil || s.service == nil {
			return
		}
		diagCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIAdvancedSchedulerSettingDBTimeout)
		defer cancel()
		if schedGroupForDiagnostics == nil {
			schedGroupForDiagnostics = s.service.openAISchedulerGroupForFallback(diagCtx, req.GroupID)
		}
		decision.Diagnostics = s.buildOpenAISelectionDiagnostics(diagCtx, req, schedGroupForDiagnostics)
	}
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
			openAIAccountScheduleOptionsFromRequest(req),
		)
		if err != nil {
			attachDiagnostics()
			return nil, decision, attachOpenAINoAvailableDiagnostics(err, req.RequestedModel, decision.Diagnostics)
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
			s.bindOpenAISelectedAccount(ctx, req, selection.Account.ID)
			return selection, decision, nil
		}
	}

	sessionReq := req
	// Ordinary session stickiness is stronger than cross-session cache affinity:
	// if the current session still has a usable account, keep it there.
	sessionReq.CacheAffinityGroup = ""
	selection, escapedStickyID, err := s.selectBySessionHash(ctx, sessionReq)
	if err != nil {
		attachDiagnostics()
		return nil, decision, attachOpenAINoAvailableDiagnostics(err, req.RequestedModel, decision.Diagnostics)
	}
	if selection != nil && selection.Account != nil {
		decision.Layer = openAIAccountScheduleLayerSessionSticky
		decision.StickySessionHit = true
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
		s.bindOpenAISelectedAccount(ctx, req, selection.Account.ID)
		return selection, decision, nil
	}
	if escapedStickyID > 0 {
		req.PreserveStickyBinding = true
		req.ExcludedIDs = cloneOpenAIExcludedIDsWith(req.ExcludedIDs, escapedStickyID)
	}

	selection, escapedAffinityID, err := s.selectByCacheAffinity(ctx, req)
	if err != nil {
		attachDiagnostics()
		return nil, decision, attachOpenAINoAvailableDiagnostics(err, req.RequestedModel, decision.Diagnostics)
	}
	if selection != nil && selection.Account != nil {
		decision.Layer = openAIAccountScheduleLayerCacheAffinity
		decision.CacheAffinityHit = true
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
		s.bindOpenAISelectedAccount(ctx, req, selection.Account.ID)
		return selection, decision, nil
	}
	if escapedAffinityID > 0 {
		req.ExcludedIDs = cloneOpenAIExcludedIDsWith(req.ExcludedIDs, escapedAffinityID)
	}

	selection, candidateCount, topK, loadSkew, err := s.selectByLoadBalance(ctx, req)
	decision.Layer = openAIAccountScheduleLayerLoadBalance
	decision.CandidateCount = candidateCount
	decision.TopK = topK
	decision.LoadSkew = loadSkew
	if err != nil {
		attachDiagnostics()
		err = attachOpenAINoAvailableDiagnostics(err, req.RequestedModel, decision.Diagnostics)
		s.service.emitOpenAISelectionEmptyAlert(ctx, req, decision, err)
		return nil, decision, err
	}
	if selection != nil && selection.Account != nil {
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
	}
	return selection, decision, nil
}

func (s *defaultOpenAIAccountScheduler) selectByCacheAffinity(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, int64, error) {
	affinityHash := strings.TrimSpace(req.CacheAffinityHash)
	if affinityHash == "" || s == nil || s.service == nil || s.service.cache == nil {
		return nil, 0, nil
	}

	affinityReq := req
	affinityReq.SessionHash = affinityHash
	affinityReq.StickyAccountID = req.CacheAffinityAccountID
	affinityReq.PreserveStickyBinding = false
	return s.selectBySessionHash(ctx, affinityReq)
}

func cloneOpenAIExcludedIDsWith(ids map[int64]struct{}, accountID int64) map[int64]struct{} {
	if accountID <= 0 {
		return ids
	}
	next := make(map[int64]struct{}, len(ids)+1)
	for id := range ids {
		next[id] = struct{}{}
	}
	next[accountID] = struct{}{}
	return next
}

func (s *defaultOpenAIAccountScheduler) selectBySessionHash(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, int64, error) {
	sessionHash := strings.TrimSpace(req.SessionHash)
	if sessionHash == "" || s == nil || s.service == nil || s.service.cache == nil {
		return nil, 0, nil
	}

	accountID := req.StickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.service.getStickySessionAccountID(ctx, req.GroupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil, 0, nil
		}
	}
	if accountID <= 0 {
		return nil, 0, nil
	}
	if req.ExcludedIDs != nil {
		if _, excluded := req.ExcludedIDs[accountID]; excluded {
			return nil, 0, nil
		}
	}

	account, err := s.service.getSchedulableAccount(ctx, accountID)
	if err != nil || account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, 0, nil
	}
	if shouldClearStickySession(account, req.RequestedModel) || !account.IsOpenAI() || !account.IsSchedulable() {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, 0, nil
	}
	if !openAICompactStickyHitAllowed(req.RequireCompact, account, req.RequestedModel) {
		return nil, 0, nil
	}
	if !s.isAccountRequestCompatible(ctx, account, req) {
		return nil, 0, nil
	}
	if req.CacheAffinityGroup != "" && openAIAccountCacheAffinityGroup(account) != req.CacheAffinityGroup {
		return nil, 0, nil
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, 0, nil
	}
	account = s.service.recheckSelectedOpenAIAccountFromDBForSelection(ctx, account, req.RequestedModel, req.RequireCompact, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
	if account == nil || !openAIStickyAccountMatchesGroup(account, req.GroupID) || !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, 0, nil
	}
	if req.CacheAffinityGroup != "" && openAIAccountCacheAffinityGroup(account) != req.CacheAffinityGroup {
		return nil, 0, nil
	}
	escapeCfg := s.service.openAIStickyEscapeConfig()
	if reason, errorRate, ttft, shouldEscape := s.shouldEscapeStickyAccount(accountID, escapeCfg); shouldEscape {
		slog.Info("sticky_escape_triggered",
			"account_id", accountID,
			"reason", reason,
			"error_rate", errorRate,
			"ttft", ttft,
		)
		return nil, accountID, nil
	}
	if !openAICompactStickyHitAllowed(req.RequireCompact, account, req.RequestedModel) {
		return nil, 0, nil
	}
	if !s.service.isOpenAIAccountSchedulerHealthAllowed(account.ID, req.RequestedModel, schedulerEndpointFromOpenAIRequest(req)) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, 0, nil
	}

	result, acquireErr := s.service.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
	if acquireErr == nil && result != nil && result.Acquired {
		_ = s.service.refreshStickySessionTTL(ctx, req.GroupID, sessionHash, s.service.openAIWSSessionStickyTTL())
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, 0, nil
	}

	cfg := s.service.schedulingConfig()
	// WaitPlan.MaxConcurrency 使用 Concurrency（非 EffectiveLoadFactor），因为 WaitPlan 控制的是 Redis 实际并发槽位等待。
	if s.service.concurrencyService != nil {
		if escapeCfg.enabled && acquireErr == nil && result != nil && !result.Acquired {
			errorRate, ttft, _ := s.stats.snapshot(accountID)
			slog.Info("sticky_escape_triggered",
				"account_id", accountID,
				"reason", "concurrency_full",
				"error_rate", errorRate,
				"ttft", ttft,
			)
			return nil, accountID, nil
		}
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      accountID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.StickySessionWaitTimeout,
				MaxWaiting:     cfg.StickySessionMaxWaiting,
			},
		}, 0, nil
	}
	return nil, 0, nil
}

func openAIStickyAccountMatchesGroup(account *Account, groupID *int64) bool {
	if account == nil {
		return false
	}
	if groupID == nil {
		return len(account.AccountGroups) == 0 && len(account.GroupIDs) == 0
	}
	for _, accountGroupID := range account.GroupIDs {
		if accountGroupID == *groupID {
			return true
		}
	}
	for _, accountGroup := range account.AccountGroups {
		if accountGroup.GroupID == *groupID {
			return true
		}
	}
	return false
}

func (s *defaultOpenAIAccountScheduler) shouldEscapeStickyAccount(accountID int64, cfg openAIStickyEscapeConfig) (reason string, errorRate float64, ttft float64, shouldEscape bool) {
	if !cfg.enabled || s == nil || s.stats == nil || accountID <= 0 {
		return "", 0, 0, false
	}
	errorRate, ttft, hasTTFT := s.stats.snapshot(accountID)
	if hasTTFT && ttft > cfg.ttftMs {
		return "ttft", errorRate, ttft, true
	}
	if errorRate > cfg.errorRate {
		return "error_rate", errorRate, ttft, true
	}
	return "", errorRate, ttft, false
}

func (s *defaultOpenAIAccountScheduler) bindOpenAISelectedAccount(ctx context.Context, req OpenAIAccountScheduleRequest, accountID int64) {
	if s == nil || s.service == nil || accountID <= 0 {
		return
	}
	if req.SessionHash != "" && !req.PreserveStickyBinding {
		_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, accountID)
	}
	if req.CacheAffinityHash != "" {
		_ = s.service.BindStickySession(ctx, req.GroupID, req.CacheAffinityHash, accountID)
	}
}

type openAIAccountCandidateScore struct {
	account    *Account
	loadInfo   *AccountLoadInfo
	score      float64
	errorRate  float64
	ttft       float64
	hasTTFT    bool
	sortOrder  int
	groupOrder bool
	groupPrio  int
	health     schedulerHealthSnapshot
	halfOpen   bool
	cooldown   bool
	cooldownAt time.Time
	excluded   bool
}

func isOpenAIAccountCandidateBetter(left openAIAccountCandidateScore, right openAIAccountCandidateScore) bool {
	return isOpenAIAccountCandidateBetterWithSeed(left, right, schedulerSeededOrder{})
}

func isOpenAIAccountCandidateBetterWithSeed(left openAIAccountCandidateScore, right openAIAccountCandidateScore, seed schedulerSeededOrder) bool {
	if left.groupOrder || right.groupOrder {
		return isOpenAIAccountGroupOrderCandidateBetterWithSeed(left, right, seed)
	}
	if left.excluded != right.excluded {
		return !left.excluded
	}
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
	if seed.enabled {
		return openAIAccountCandidateSeededTieBreakLess(seed, left, right)
	}
	if openAIAccountCandidateLastUsedLess(left, right) {
		return true
	}
	if openAIAccountCandidateLastUsedLess(right, left) {
		return false
	}
	return openAIAccountCandidateSeededTieBreakLess(seed, left, right)
}

func isOpenAIAccountGroupOrderCandidateBetter(left openAIAccountCandidateScore, right openAIAccountCandidateScore) bool {
	return isOpenAIAccountGroupOrderCandidateBetterWithSeed(left, right, schedulerSeededOrder{})
}

func isOpenAIAccountGroupOrderCandidateBetterWithSeed(left openAIAccountCandidateScore, right openAIAccountCandidateScore, seed schedulerSeededOrder) bool {
	if left.excluded != right.excluded {
		return !left.excluded
	}
	if left.cooldown != right.cooldown {
		return !left.cooldown
	}
	if left.groupOrder != right.groupOrder {
		return left.groupOrder
	}
	if left.sortOrder != right.sortOrder {
		return left.sortOrder < right.sortOrder
	}
	if left.groupPrio != right.groupPrio {
		return left.groupPrio < right.groupPrio
	}
	if left.halfOpen != right.halfOpen {
		return !left.halfOpen
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
	return openAIAccountCandidateSeededTieBreakLess(seed, left, right)
}

func openAIAccountCandidateSeededTieBreakLess(seed schedulerSeededOrder, left, right openAIAccountCandidateScore) bool {
	return schedulerSeededTieBreakLess(seed,
		schedulerAccountScore{Account: left.account},
		schedulerAccountScore{Account: right.account},
	)
}

func openAIAccountCandidateID(candidate openAIAccountCandidateScore) int64 {
	if candidate.account == nil {
		return 0
	}
	return candidate.account.ID
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

func openAIAccountCandidateLastUsedLess(left openAIAccountCandidateScore, right openAIAccountCandidateScore) bool {
	if left.account == nil || right.account == nil {
		return left.account != nil
	}
	switch {
	case left.account.LastUsedAt == nil && right.account.LastUsedAt != nil:
		return true
	case left.account.LastUsedAt != nil && right.account.LastUsedAt == nil:
		return false
	case left.account.LastUsedAt == nil && right.account.LastUsedAt == nil:
		return false
	default:
		return left.account.LastUsedAt.Before(*right.account.LastUsedAt)
	}
}

func openAIAccountCacheAffinityGroup(account *Account) string {
	if account == nil {
		return ""
	}
	group := account.GetCacheAffinityGroup()
	if group != "" {
		return group
	}
	if account.ID > 0 {
		return fmt.Sprintf("account:%d", account.ID)
	}
	return ""
}

func splitOpenAICacheAffinityGroupCandidates(candidates []openAIAccountCandidateScore, group string) (compatible []openAIAccountCandidateScore, fallback []openAIAccountCandidateScore) {
	group = strings.TrimSpace(group)
	if group == "" || len(candidates) == 0 {
		return nil, candidates
	}
	compatible = make([]openAIAccountCandidateScore, 0, len(candidates))
	fallback = make([]openAIAccountCandidateScore, 0, len(candidates))
	for _, candidate := range candidates {
		if openAIAccountCacheAffinityGroup(candidate.account) == group {
			compatible = append(compatible, candidate)
			continue
		}
		fallback = append(fallback, candidate)
	}
	return compatible, fallback
}

func schedulerEndpointFromOpenAIRequest(req OpenAIAccountScheduleRequest) string {
	if endpoint := strings.TrimSpace(req.SchedulerEndpoint); endpoint != "" {
		return endpoint
	}
	if req.RequireCompact {
		return "/v1/responses/compact"
	}
	if req.RequireCodexImageGenerationBridge {
		if req.RequiredTransport == OpenAIUpstreamTransportResponsesWebsocketV2 {
			return OpenAIResponsesSchedulerEndpointForIntent("/v1/responses/ws", true)
		}
		return OpenAIResponsesSchedulerEndpointForIntent("/v1/responses", true)
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

func openAICompactStrictSupportedOnly(req OpenAIAccountScheduleRequest) bool {
	return openAICompactStrictSupportedOnlyForSelection(req.RequireCompact, req.ExcludedIDs)
}

func openAICompactStrictSupportedOnlyForSelection(requireCompact bool, excludedIDs map[int64]struct{}) bool {
	return requireCompact && len(excludedIDs) > 0
}

func openAICompactAccountAllowed(requireCompact bool, strictSupportedOnly bool, account *Account, requestedModel string) bool {
	if !requireCompact {
		return true
	}
	tier := openAICompactSupportTierForModel(account, requestedModel)
	if strictSupportedOnly {
		return tier == 2
	}
	return tier > 0
}

func openAICompactAccountAllowedForRequest(req OpenAIAccountScheduleRequest, account *Account) bool {
	return openAICompactAccountAllowed(req.RequireCompact, openAICompactStrictSupportedOnly(req), account, req.RequestedModel)
}

func openAICompactAccountAllowedForSelection(requireCompact bool, excludedIDs map[int64]struct{}, account *Account, requestedModel string) bool {
	return openAICompactAccountAllowed(requireCompact, openAICompactStrictSupportedOnlyForSelection(requireCompact, excludedIDs), account, requestedModel)
}

func openAIAccountScheduleOptionsFromRequest(req OpenAIAccountScheduleRequest) OpenAIAccountScheduleOptions {
	return normalizeOpenAIAccountScheduleOptions(req.RequireCompact, OpenAIAccountScheduleOptions{
		RequireCodexImageGenerationBridge: req.RequireCodexImageGenerationBridge,
	})
}

func normalizeOpenAIAccountScheduleOptions(requireCompact bool, options OpenAIAccountScheduleOptions) OpenAIAccountScheduleOptions {
	if requireCompact {
		options.AllowCompactModelMapping = true
	}
	return options
}

func accountSatisfiesOpenAIScheduleRequest(account *Account, req OpenAIAccountScheduleRequest) bool {
	return accountSatisfiesOpenAIScheduleOptions(account, openAIAccountScheduleOptionsFromRequest(req))
}

func openAIAccountSupportsModelForSchedule(account *Account, requestedModel string, requireCompact bool, options OpenAIAccountScheduleOptions) bool {
	if strings.TrimSpace(requestedModel) == "" {
		return true
	}
	if account == nil {
		return false
	}
	if account.IsModelSupported(requestedModel) {
		return true
	}
	return (requireCompact || options.AllowCompactModelMapping) && openAICompactMappingMatchesRequest(account, requestedModel)
}

func openAIAccountIDsFromMap(ids map[int64]struct{}) []int64 {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func appendOpenAIAccountID(ids []int64, account *Account) []int64 {
	if account == nil || account.ID <= 0 {
		return ids
	}
	return append(ids, account.ID)
}

func openAIAccountsWithAvailableConcurrency(accounts []*Account, loadMap map[int64]*AccountLoadInfo) []*Account {
	if len(accounts) == 0 {
		return nil
	}
	out := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		loadInfo := loadMap[account.ID]
		if loadInfo != nil && loadInfo.LoadRate >= 100 {
			continue
		}
		out = append(out, account)
	}
	return out
}

func openAIAccountStatusFilterReason(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest, schedGroup *Group) string {
	if account == nil {
		return "nil_account"
	}
	if !account.IsOpenAI() {
		return "not_openai"
	}
	if !account.IsActive() {
		return "inactive"
	}
	if !account.Schedulable {
		return "manual_unschedulable"
	}
	now := time.Now()
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		return "expired"
	}
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		return "overloaded"
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		return "rate_limited"
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		return "temp_unschedulable"
	}
	if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		return "quota_exceeded"
	}
	if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
		return "quota_auto_paused"
	}
	if account.GetRateLimitRemainingTimeWithContext(ctx, req.RequestedModel) > 0 {
		return "model_rate_limited"
	}
	if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
		return "privacy_not_set"
	}
	return ""
}

func (s *defaultOpenAIAccountScheduler) openAIAccountCircuitFilterReason(account *Account, req OpenAIAccountScheduleRequest, endpoint string) string {
	reason, _ := s.openAIAccountCircuitFilterState(account, req, endpoint)
	return reason
}

func (d *OpenAIAccountSelectionDiagnostics) addReason(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	if d.FilterReasonCounts == nil {
		d.FilterReasonCounts = make(map[string]int)
	}
	d.FilterReasonCounts[reason]++
}

func (d *OpenAIAccountSelectionDiagnostics) considerRetryAt(account *Account, reason string, retryAt time.Time) {
	if d == nil || account == nil || retryAt.IsZero() {
		return
	}
	now := time.Now()
	if !retryAt.After(now) {
		return
	}
	if d.EarliestRetryAt.IsZero() || retryAt.Before(d.EarliestRetryAt) {
		d.EarliestRetryAt = retryAt
		d.EarliestRetryReason = strings.TrimSpace(reason)
		d.EarliestRetryAccountID = account.ID
		d.RetryAfterSeconds = retryAfterSecondsUntil(retryAt, now)
	}
}

func retryAfterSecondsUntil(retryAt time.Time, now time.Time) int {
	if retryAt.IsZero() || !retryAt.After(now) {
		return 0
	}
	seconds := int(math.Ceil(retryAt.Sub(now).Seconds()))
	if seconds < 1 {
		return 1
	}
	return seconds
}

func openAIAccountStatusRetryAt(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest, reason string) time.Time {
	if account == nil {
		return time.Time{}
	}
	switch reason {
	case "overloaded":
		if account.OverloadUntil != nil {
			return *account.OverloadUntil
		}
	case "rate_limited":
		if account.RateLimitResetAt != nil {
			return *account.RateLimitResetAt
		}
	case "temp_unschedulable":
		if account.TempUnschedulableUntil != nil {
			return *account.TempUnschedulableUntil
		}
	case "model_rate_limited":
		return openAIAccountModelRateLimitRetryAt(ctx, account, req)
	}
	return time.Time{}
}

func openAIAccountModelRateLimitRetryAt(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) time.Time {
	if account == nil {
		return time.Time{}
	}
	now := time.Now()
	var retryAt time.Time
	for _, key := range account.modelRateLimitKeysForRequest(ctx, req.RequestedModel) {
		resetAt := account.modelRateLimitResetAt(key)
		if resetAt == nil || !resetAt.After(now) {
			continue
		}
		retryAt = earliestNonZeroTime(retryAt, *resetAt)
	}
	if retryAt.IsZero() {
		if remaining := account.GetRateLimitRemainingTimeWithContext(ctx, req.RequestedModel); remaining > 0 {
			retryAt = now.Add(remaining)
		}
	}
	return retryAt
}

func (s *defaultOpenAIAccountScheduler) openAIAccountCircuitFilterState(account *Account, req OpenAIAccountScheduleRequest, endpoint string) (string, time.Time) {
	if s == nil || s.service == nil || account == nil {
		return "", time.Time{}
	}
	now := time.Now()
	if until, ok := s.service.openAIAccountRuntimeBlockUntil(account.ID); ok && until.After(now) {
		return "runtime_circuit_open", until
	}
	if s.service.isOpenAIAccountCircuitHalfOpenInFlight(account.ID, now) {
		return "runtime_half_open_in_flight", time.Time{}
	}
	if s.service.schedulerHealth != nil {
		allowHalfOpen := openAIRequestRequiresImageGenerationBridge(req)
		// Most OpenAI dimensions recover through the background probe runner.
		// Responses image_generation uses real user requests as the half-open
		// probe because a text-only probe does not validate that tool chain.
		snap := s.service.schedulerHealth.snapshot(account.ID, req.RequestedModel, endpoint, allowHalfOpen)
		switch snap.CircuitState {
		case schedulerCircuitOpen:
			if snap.CooldownUntil.IsZero() || snap.CooldownUntil.After(now) {
				return "scheduler_circuit_open", snap.CooldownUntil
			}
		case schedulerCircuitHalfOpen:
			if allowHalfOpen && snap.HalfOpenProbe {
				return "", time.Time{}
			}
			return "scheduler_probe_pending", snap.CooldownUntil
		}
	}
	return "", time.Time{}
}

func (s *defaultOpenAIAccountScheduler) buildOpenAISelectionPlan(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
) openAISelectionPlan {
	if s == nil || s.service == nil {
		return openAISelectionPlan{
			diagnostics: OpenAIAccountSelectionDiagnostics{
				Collected: false,
			},
		}
	}
	accounts, err := s.service.listOpenAICooldownFallbackAccounts(ctx, req.GroupID)
	if err != nil {
		plan := s.buildOpenAISelectionPlanFromAccounts(ctx, req, schedGroup, nil)
		plan.diagnostics.addReason("diagnostics_account_list_failed")
		return plan
	}
	return s.buildOpenAISelectionPlanFromAccounts(ctx, req, schedGroup, accounts)
}

func (s *defaultOpenAIAccountScheduler) buildOpenAISelectionPlanFromAccounts(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
	accounts []*Account,
) openAISelectionPlan {
	diag := OpenAIAccountSelectionDiagnostics{
		Collected:                         true,
		GroupID:                           derefGroupID(req.GroupID),
		Model:                             req.RequestedModel,
		Endpoint:                          schedulerEndpointFromOpenAIRequest(req),
		RequireCompact:                    req.RequireCompact,
		CompactStrictSupportedOnly:        openAICompactStrictSupportedOnly(req),
		RequiredTransport:                 string(req.RequiredTransport),
		RequiredCapability:                string(req.RequiredCapability),
		RequiredImageCapability:           string(req.RequiredImageCapability),
		RequireCodexImageGenerationBridge: req.RequireCodexImageGenerationBridge,
		ExcludedAccountCount:              len(req.ExcludedIDs),
		ExcludedAccountIDs:                openAIAccountIDsFromMap(req.ExcludedIDs),
		FilterReasonCounts:                make(map[string]int),
	}
	plan := openAISelectionPlan{diagnostics: diag, schedGroup: schedGroup}
	if s == nil || s.service == nil {
		plan.diagnostics.Collected = false
		return plan
	}

	plan.accounts = accounts
	diag.GroupBindingAccountCount = len(accounts)
	circuitAllowed := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			diag.addReason("nil_account")
			continue
		}
		diag.GroupBindingAccountIDs = appendOpenAIAccountID(diag.GroupBindingAccountIDs, account)
		if account.IsOpenAI() && account.Status == StatusActive && account.Schedulable {
			diag.ActiveSchedulableCount++
			diag.ActiveSchedulableAccountIDs = appendOpenAIAccountID(diag.ActiveSchedulableAccountIDs, account)
		}
		if req.ExcludedIDs != nil {
			if _, excluded := req.ExcludedIDs[account.ID]; excluded {
				diag.addReason("excluded")
				continue
			}
		}
		diag.AfterExcludedCount++
		diag.AfterExcludedAccountIDs = appendOpenAIAccountID(diag.AfterExcludedAccountIDs, account)

		if !openAIAccountSupportsModelForSchedule(account, req.RequestedModel, req.RequireCompact, openAIAccountScheduleOptionsFromRequest(req)) {
			diag.ModelUnsupportedAccountIDs = appendOpenAIAccountID(diag.ModelUnsupportedAccountIDs, account)
			diag.addReason("model_unsupported")
			continue
		}
		diag.ModelSupportedCount++
		diag.ModelSupportedAccountIDs = appendOpenAIAccountID(diag.ModelSupportedAccountIDs, account)

		if req.GroupID != nil && s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
			s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
			diag.ChannelRestrictionFilteredCount++
			diag.ChannelRestrictionAccountIDs = appendOpenAIAccountID(diag.ChannelRestrictionAccountIDs, account)
			diag.addReason("channel_pricing_restricted")
			continue
		}

		if !accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability) ||
			!s.isAccountTransportCompatible(account, req.RequiredTransport) {
			diag.EndpointUnsupportedAccountIDs = appendOpenAIAccountID(diag.EndpointUnsupportedAccountIDs, account)
			diag.addReason("endpoint_unsupported")
			continue
		}
		diag.EndpointSupportedCount++
		diag.EndpointSupportedAccountIDs = appendOpenAIAccountID(diag.EndpointSupportedAccountIDs, account)

		if req.RequireCodexImageGenerationBridge && !accountSatisfiesOpenAIScheduleOptions(account, OpenAIAccountScheduleOptions{RequireCodexImageGenerationBridge: true}) {
			diag.ImageGenerationBridgeUnsupportedCount++
			diag.ImageGenerationBridgeUnsupportedAccountIDs = appendOpenAIAccountID(diag.ImageGenerationBridgeUnsupportedAccountIDs, account)
			diag.addReason("image_generation_bridge_unsupported")
			continue
		}
		if req.RequireCodexImageGenerationBridge {
			diag.ImageGenerationBridgeSupportedCount++
			diag.ImageGenerationBridgeSupportedAccountIDs = appendOpenAIAccountID(diag.ImageGenerationBridgeSupportedAccountIDs, account)
		}

		if req.RequireCompact && !openAICompactAccountAllowedForRequest(req, account) {
			diag.CompactUnsupportedCount++
			diag.CompactUnsupportedAccountIDs = appendOpenAIAccountID(diag.CompactUnsupportedAccountIDs, account)
			if openAICompactStrictSupportedOnly(req) {
				diag.addReason("compact_not_explicitly_supported")
			} else {
				diag.addReason("compact_unsupported")
			}
			continue
		}
		diag.CompactSupportedCount++
		diag.CompactSupportedAccountIDs = appendOpenAIAccountID(diag.CompactSupportedAccountIDs, account)

		if reason := openAIAccountStatusFilterReason(ctx, account, req, schedGroup); reason != "" {
			diag.StateFilteredCount++
			diag.StateFilteredAccountIDs = appendOpenAIAccountID(diag.StateFilteredAccountIDs, account)
			switch reason {
			case "inactive", "manual_unschedulable", "expired", "quota_exceeded", "quota_auto_paused":
				diag.StatusFilteredCount++
			case "temp_unschedulable":
				diag.TempUnschedulableFilteredCount++
			case "overloaded":
				diag.OverloadFilteredCount++
			case "rate_limited":
				diag.RateLimitFilteredCount++
			case "model_rate_limited":
				diag.ModelRateLimitFilteredCount++
			case "privacy_not_set":
				diag.GroupScopeFilteredCount++
			}
			diag.addReason(reason)
			diag.considerRetryAt(account, reason, openAIAccountStatusRetryAt(ctx, account, req, reason))
			continue
		}
		diag.StateAllowedCount++
		diag.StateAllowedAccountIDs = appendOpenAIAccountID(diag.StateAllowedAccountIDs, account)

		if reason, retryAt := s.openAIAccountCircuitFilterState(account, req, diag.Endpoint); reason != "" {
			diag.CircuitFilteredCount++
			diag.CircuitFilteredAccountIDs = appendOpenAIAccountID(diag.CircuitFilteredAccountIDs, account)
			if strings.Contains(reason, "half_open") {
				diag.HalfOpenFilteredCount++
				diag.HalfOpenFilteredAccountIDs = appendOpenAIAccountID(diag.HalfOpenFilteredAccountIDs, account)
			}
			diag.addReason(reason)
			diag.considerRetryAt(account, reason, retryAt)
			continue
		}
		diag.CircuitAllowedCount++
		diag.CircuitAllowedAccountIDs = appendOpenAIAccountID(diag.CircuitAllowedAccountIDs, account)
		circuitAllowed = append(circuitAllowed, account)
		plan.circuitAllowed = append(plan.circuitAllowed, account)
	}

	loadMap := map[int64]*AccountLoadInfo{}
	loadReq := make([]AccountWithConcurrency, 0, len(circuitAllowed))
	if s.service.concurrencyService != nil && len(circuitAllowed) > 0 {
		for _, account := range circuitAllowed {
			loadReq = append(loadReq, AccountWithConcurrency{
				ID:             account.ID,
				MaxConcurrency: account.EffectiveLoadFactor(),
			})
		}
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		}
	}
	candidateAccounts := make([]*Account, 0, len(circuitAllowed))
	for _, account := range circuitAllowed {
		loadInfo := loadMap[account.ID]
		if loadInfo != nil && loadInfo.LoadRate >= 100 {
			diag.ConcurrencySlotFilteredCount++
			diag.ConcurrencySlotFilteredAccountIDs = appendOpenAIAccountID(diag.ConcurrencySlotFilteredAccountIDs, account)
			diag.addReason("concurrency_full")
			continue
		}
		diag.ConcurrencySlotAllowedCount++
		diag.CandidateAccountIDs = appendOpenAIAccountID(diag.CandidateAccountIDs, account)
		candidateAccounts = append(candidateAccounts, account)
	}
	diag.FinalCandidateCount = len(diag.CandidateAccountIDs)
	plan.candidateAccounts = candidateAccounts
	plan.loadReq = loadReq
	plan.loadMap = loadMap
	if req.RequireCompact {
		plan.compactBlocked = diag.CompactUnsupportedCount > 0 && diag.CompactSupportedCount == 0
	}
	plan.loadPlan = s.buildOpenAIAccountLoadPlan(req, schedGroup, candidateAccounts, circuitAllowed, loadMap)
	for _, candidate := range plan.loadPlan.selectionOrder {
		diag.OrderedCandidateAccountIDs = appendOpenAIAccountID(diag.OrderedCandidateAccountIDs, candidate.account)
		if req.CacheAffinityGroup != "" {
			if openAIAccountCacheAffinityGroup(candidate.account) == req.CacheAffinityGroup {
				diag.CacheAffinityGroupCandidateIDs = appendOpenAIAccountID(diag.CacheAffinityGroupCandidateIDs, candidate.account)
			} else {
				diag.CacheAffinityGroupFallbackAccountIDs = appendOpenAIAccountID(diag.CacheAffinityGroupFallbackAccountIDs, candidate.account)
			}
		}
		if len(diag.OrderedCandidateAccountIDs) >= 10 {
			break
		}
	}
	plan.diagnostics = diag
	return plan
}

func (s *defaultOpenAIAccountScheduler) buildOpenAISelectionDiagnostics(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
) OpenAIAccountSelectionDiagnostics {
	return s.buildOpenAISelectionPlan(ctx, req, schedGroup).diagnostics
}

func openAIAccountCandidateFromSchedulerScore(score schedulerAccountScore) openAIAccountCandidateScore {
	return openAIAccountCandidateScore{
		account:    score.Account,
		loadInfo:   score.LoadInfo,
		score:      score.Score,
		errorRate:  score.Health.ErrorRate,
		ttft:       score.Health.TTFTEWMA,
		hasTTFT:    score.Health.HasTTFT,
		sortOrder:  score.SortOrder,
		groupOrder: score.Config.GroupID > 0 && score.Config.SchedulingConfigured,
		groupPrio:  score.Config.Priority,
		health:     score.Health,
		halfOpen:   score.HalfOpen,
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
	return buildOpenAIOrderedSelectionOrderWithSeed(candidates, schedulerSeededOrder{})
}

func buildOpenAIOrderedSelectionOrderWithSeed(candidates []openAIAccountCandidateScore, seed schedulerSeededOrder) []openAIAccountCandidateScore {
	order := append([]openAIAccountCandidateScore(nil), candidates...)
	sort.SliceStable(order, func(i, j int) bool {
		return isOpenAIAccountCandidateBetterWithSeed(order[i], order[j], seed)
	})
	return order
}

func (s *defaultOpenAIAccountScheduler) buildOpenAIAccountLoadPlan(
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
	candidatePool []*Account,
	waitPool []*Account,
	loadMap map[int64]*AccountLoadInfo,
) openAIAccountLoadPlan {
	endpoint := schedulerEndpointFromOpenAIRequest(req)
	var health *accountSchedulerHealthStats
	if s != nil && s.service != nil {
		health = s.service.schedulerHealth
	}
	allowHalfOpen := openAIRequestRequiresImageGenerationBridge(req)
	allScores := buildSchedulerAccountScores(candidatePool, req.GroupID, req.RequestedModel, endpoint, loadMap, health, allowHalfOpen)
	allCandidates := openAIAccountCandidatesFromSchedulerScores(allScores)

	candidates := allCandidates
	staleSnapshotCompactRetry := make([]openAIAccountCandidateScore, 0, len(allCandidates))
	if req.RequireCompact {
		candidates = make([]openAIAccountCandidateScore, 0, len(allCandidates))
		for _, candidate := range allCandidates {
			if !openAICompactAccountAllowedForRequest(req, candidate.account) {
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
		stableLowTTFT:             isOpenAIStableLowTTFTGroup(schedGroup),
	}
	if plan.stableLowTTFT {
		plan.stableLowTTFTSeq = s.stableLowTTFTSeq.Add(1)
	}
	plan.topK = len(candidates)
	if plan.topK <= 0 {
		plan.topK = len(staleSnapshotCompactRetry)
	}
	waitScores := buildSchedulerAccountWaitScores(waitPool, req.GroupID, req.RequestedModel, endpoint, loadMap, health)
	waitCandidates := openAIAccountCandidatesFromSchedulerScores(waitScores)
	if len(candidates) == 0 {
		plan.selectionOrder = s.buildOpenAISelectionOrder(req, plan)
		waitPlan := plan
		waitPlan.candidates = waitCandidates
		waitPlan.candidateCount = len(waitCandidates)
		waitPlan.staleSnapshotCompactRetry = nil
		if req.RequireCompact {
			waitPlan.candidates, waitPlan.staleSnapshotCompactRetry = splitOpenAICompactCandidates(waitCandidates, req.RequestedModel)
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
		waitPlan.candidates, waitPlan.staleSnapshotCompactRetry = splitOpenAICompactCandidates(waitCandidates, req.RequestedModel)
		waitPlan.candidateCount = len(waitPlan.candidates)
	}
	waitPlan.topK = len(waitPlan.candidates)
	if waitPlan.topK <= 0 {
		waitPlan.topK = len(waitPlan.staleSnapshotCompactRetry)
	}
	plan.waitOrder = s.buildOpenAISelectionOrder(req, waitPlan)
	return plan
}

func openAIAccountSoftCooldownState(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest, schedGroup *Group, health schedulerHealthSnapshot, loadInfo *AccountLoadInfo) (cooldown bool, cooldownAt time.Time, reasons []string) {
	if account == nil {
		return false, time.Time{}, nil
	}
	now := time.Now()
	addReason := func(reason string) {
		if reason == "" {
			return
		}
		reasons = append(reasons, reason)
	}
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		cooldown = true
		cooldownAt = earliestNonZeroTime(cooldownAt, *account.OverloadUntil)
		addReason("overloaded")
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		cooldown = true
		cooldownAt = earliestNonZeroTime(cooldownAt, *account.RateLimitResetAt)
		addReason("rate_limited")
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		cooldown = true
		cooldownAt = earliestNonZeroTime(cooldownAt, *account.TempUnschedulableUntil)
		addReason("temp_unschedulable")
	}
	if account.GetRateLimitRemainingTimeWithContext(ctx, req.RequestedModel) > 0 {
		cooldown = true
		addReason("model_rate_limited")
	}
	if health.CircuitState == schedulerCircuitOpen || health.CircuitState == schedulerCircuitHalfOpen {
		cooldown = true
		cooldownAt = earliestNonZeroTime(cooldownAt, health.CooldownUntil)
		addReason("scheduler_" + health.CircuitState)
	}
	if loadInfo != nil && loadInfo.LoadRate >= 100 {
		cooldown = true
		addReason("concurrency_full")
	}
	_ = schedGroup
	return cooldown, cooldownAt, reasons
}

func earliestNonZeroTime(current time.Time, candidate time.Time) time.Time {
	if candidate.IsZero() {
		return current
	}
	if current.IsZero() || candidate.Before(current) {
		return candidate
	}
	return current
}

func (s *defaultOpenAIAccountScheduler) isOpenAIWeakFallbackHardCompatible(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest, schedGroup *Group) bool {
	if account == nil || !account.IsOpenAI() {
		return false
	}
	if account.Status != StatusActive || !account.Schedulable {
		return false
	}
	now := time.Now()
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		return false
	}
	if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		return false
	}
	if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
		return false
	}
	if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
		return false
	}
	if !openAIAccountSupportsModelForSchedule(account, req.RequestedModel, req.RequireCompact, openAIAccountScheduleOptionsFromRequest(req)) {
		return false
	}
	if req.GroupID != nil && s != nil && s.service != nil &&
		s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
		s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
		return false
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		return false
	}
	if !accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability) {
		return false
	}
	if !accountSatisfiesOpenAIScheduleRequest(account, req) {
		return false
	}
	if req.RequireCompact && !openAICompactAccountAllowedForRequest(req, account) {
		return false
	}
	return true
}

func (s *defaultOpenAIAccountScheduler) isOpenAIWeakFallbackHealthEligible(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest, health schedulerHealthSnapshot, loadInfo *AccountLoadInfo) bool {
	if account == nil {
		return false
	}
	if s != nil && s.service != nil {
		if s.service.isOpenAIAccountRuntimeBlocked(account) {
			return false
		}
		if !s.service.isOpenAIAccountSchedulerHealthAllowedForSelection(account.ID, req.RequestedModel, schedulerEndpointFromOpenAIRequest(req), false) {
			return false
		}
	}
	if health.CircuitState != "" && health.CircuitState != schedulerCircuitClosed {
		return false
	}
	now := time.Now()
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		return false
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		return false
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		return false
	}
	if account.GetRateLimitRemainingTimeWithContext(ctx, req.RequestedModel) > 0 {
		return false
	}
	_ = loadInfo
	return true
}

func (s *defaultOpenAIAccountScheduler) buildOpenAIWeakFallbackOrder(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
	accounts []*Account,
	loadMap map[int64]*AccountLoadInfo,
) ([]openAIAccountCandidateScore, bool, int) {
	if len(accounts) == 0 {
		return nil, false, 0
	}
	endpoint := schedulerEndpointFromOpenAIRequest(req)
	var healthStats *accountSchedulerHealthStats
	if s != nil && s.service != nil {
		healthStats = s.service.schedulerHealth
	}
	candidates := make([]openAIAccountCandidateScore, 0, len(accounts))
	compactBlocked := false
	excludedCount := 0
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if req.RequireCompact && !openAICompactAccountAllowedForRequest(req, account) {
			compactBlocked = true
		}
		if !s.isOpenAIWeakFallbackHardCompatible(ctx, account, req, schedGroup) {
			continue
		}
		excluded := false
		if req.ExcludedIDs != nil {
			_, excluded = req.ExcludedIDs[account.ID]
		}
		if excluded {
			excludedCount++
			continue
		}
		health := schedulerHealthSnapshot{HealthScore: 1, ModelScore: 1, LatencyScore: 1, CircuitState: schedulerCircuitClosed}
		if healthStats != nil {
			health = healthStats.snapshot(account.ID, req.RequestedModel, endpoint, true)
		}
		loadInfo := loadMap[account.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: account.ID}
		}
		if !s.isOpenAIWeakFallbackHealthEligible(ctx, account, req, health, loadInfo) {
			continue
		}
		cfg := accountGroupConfigFor(account, req.GroupID)
		cooldown, cooldownAt, _ := openAIAccountSoftCooldownState(ctx, account, req, schedGroup, health, loadInfo)
		score := 1.0
		if health.HealthScore > 0 {
			score *= health.HealthScore
		}
		if health.ModelScore > 0 {
			score *= health.ModelScore
		}
		if health.LatencyScore > 0 {
			score *= health.LatencyScore
		}
		candidates = append(candidates, openAIAccountCandidateScore{
			account:    account,
			loadInfo:   loadInfo,
			score:      score,
			errorRate:  health.ErrorRate,
			ttft:       health.TTFTEWMA,
			hasTTFT:    health.HasTTFT,
			sortOrder:  cfg.SortOrder,
			groupOrder: cfg.GroupID > 0 && cfg.SchedulingConfigured,
			groupPrio:  cfg.Priority,
			health:     health,
			halfOpen:   health.CircuitState == schedulerCircuitHalfOpen || health.HalfOpenProbe,
			cooldown:   cooldown,
			cooldownAt: cooldownAt,
			excluded:   excluded,
		})
	}
	return buildOpenAIOrderedSelectionOrder(candidates), compactBlocked, excludedCount
}

func (s *defaultOpenAIAccountScheduler) trySelectOpenAICooldownFallback(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
	allowWaitPlan bool,
) (*AccountSelectionResult, int, bool, error) {
	if s == nil || s.service == nil {
		return nil, 0, false, nil
	}
	if openAIRequestRequiresImageGenerationBridge(req) {
		return nil, 0, false, nil
	}
	accounts, err := s.service.listOpenAICooldownFallbackAccounts(ctx, req.GroupID)
	if err != nil {
		return nil, 0, false, err
	}
	if len(accounts) == 0 {
		return nil, 0, false, nil
	}
	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil {
		loadReq := make([]AccountWithConcurrency, 0, len(accounts))
		for _, account := range accounts {
			if account == nil || !account.IsOpenAI() {
				continue
			}
			loadReq = append(loadReq, AccountWithConcurrency{
				ID:             account.ID,
				MaxConcurrency: account.EffectiveLoadFactor(),
			})
		}
		if len(loadReq) > 0 {
			if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatchFresh(ctx, loadReq); loadErr == nil {
				loadMap = batchLoad
			} else if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
				loadMap = batchLoad
			}
		}
	}
	order, compactBlocked, excludedCount := s.buildOpenAIWeakFallbackOrder(ctx, req, schedGroup, accounts, loadMap)
	if len(order) == 0 {
		return nil, 0, compactBlocked, nil
	}
	result, orderCompactBlocked, acquireErr := s.tryAcquireOpenAISelectionOrder(ctx, req, order, true, true)
	if acquireErr != nil {
		return nil, len(order), compactBlocked || orderCompactBlocked, acquireErr
	}
	if result == nil {
		if allowWaitPlan && s.service.concurrencyService != nil {
			cfg := s.service.schedulingConfig()
			for _, candidate := range order {
				fresh := s.service.recheckSelectedOpenAIAccountFromDBIgnoringCooldownForSelection(ctx, candidate.account, req.RequestedModel, req.RequireCompact, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
				if fresh == nil || !s.isOpenAIWeakFallbackHardCompatible(ctx, fresh, req, schedGroup) ||
					!s.isOpenAIWeakFallbackHealthEligible(ctx, fresh, req, schedulerHealthSnapshot{}, nil) {
					continue
				}
				return &AccountSelectionResult{
					Account:              fresh,
					WeakFallback:         true,
					WeakFallbackReason:   openAIAccountWeakFallbackReason,
					BypassOpenAIHeaderTO: true,
					WaitPlan: &AccountWaitPlan{
						AccountID:      fresh.ID,
						MaxConcurrency: fresh.Concurrency,
						Timeout:        cfg.FallbackWaitTimeout,
						MaxWaiting:     cfg.FallbackMaxWaiting,
					},
				}, len(order), compactBlocked || orderCompactBlocked, nil
			}
		}
		return nil, len(order), compactBlocked || orderCompactBlocked, nil
	}
	result.WeakFallback = true
	result.WeakFallbackReason = openAIAccountWeakFallbackReason
	result.BypassOpenAIHeaderTO = true
	slog.Warn("openai_account_weak_fallback_selected",
		"group_id", derefGroupID(req.GroupID),
		"model", req.RequestedModel,
		"endpoint", schedulerEndpointFromOpenAIRequest(req),
		"account_id", result.Account.ID,
		"group_binding_count", len(accounts),
		"candidate_count", len(order),
		"excluded_candidate_count", excludedCount,
	)
	return result, len(order), compactBlocked || orderCompactBlocked, nil
}

func (s *defaultOpenAIAccountScheduler) buildOpenAISelectionPlanFromSchedulableAccounts(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
	fresh bool,
) (openAISelectionPlan, error) {
	if fresh {
		accounts, err := s.service.listFreshSchedulableAccounts(ctx, req.GroupID)
		if err != nil {
			return openAISelectionPlan{}, err
		}
		return s.buildOpenAISelectionPlanFromAccounts(ctx, req, schedGroup, accounts), nil
	}
	accounts, err := s.service.listSchedulableAccounts(ctx, req.GroupID)
	if err != nil {
		return openAISelectionPlan{}, err
	}
	return s.buildOpenAISelectionPlanFromAccounts(ctx, req, schedGroup, accounts), nil
}

func (s *defaultOpenAIAccountScheduler) refreshOpenAISelectionPlanLoad(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	base openAISelectionPlan,
) (openAISelectionPlan, bool) {
	if s == nil || s.service == nil || s.service.concurrencyService == nil || len(base.loadReq) == 0 {
		return base, false
	}
	freshLoadMap, loadErr := s.service.concurrencyService.GetAccountsLoadBatchFresh(ctx, base.loadReq)
	if loadErr != nil {
		return base, false
	}
	base.loadMap = freshLoadMap
	base.candidateAccounts = openAIAccountsWithAvailableConcurrency(base.circuitAllowed, freshLoadMap)
	base.loadPlan = s.buildOpenAIAccountLoadPlan(req, base.schedGroup, base.candidateAccounts, base.circuitAllowed, freshLoadMap)
	return base, true
}

func (s *defaultOpenAIAccountScheduler) tryAcquireOpenAISelectionPlan(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	plan openAISelectionPlan,
	allowCooldownFallback bool,
	bypassSnapshot bool,
) (*AccountSelectionResult, bool, error) {
	if len(plan.loadPlan.selectionOrder) == 0 {
		return nil, plan.compactBlocked, nil
	}
	selection, compactBlocked, err := s.tryAcquireOpenAISelectionOrder(ctx, req, plan.loadPlan.selectionOrder, allowCooldownFallback, bypassSnapshot)
	return selection, plan.compactBlocked || compactBlocked, err
}

func (s *defaultOpenAIAccountScheduler) waitPlanFromOpenAISelectionPlan(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	plan openAISelectionPlan,
	allowCooldownFallback bool,
) (*AccountSelectionResult, bool) {
	if s == nil || s.service == nil || s.service.concurrencyService == nil {
		return nil, plan.compactBlocked
	}
	cfg := s.service.schedulingConfig()
	waitOrder := plan.loadPlan.waitOrder
	if len(waitOrder) == 0 {
		waitOrder = plan.loadPlan.selectionOrder
	}
	compatible := s.isAccountRequestCompatible
	for _, candidate := range waitOrder {
		var fresh *Account
		if allowCooldownFallback {
			compatible = s.isAccountRequestCompatibleIgnoringCooldown
			fresh = s.service.recheckSelectedOpenAIAccountFromDBIgnoringCooldownForSelection(ctx, candidate.account, req.RequestedModel, req.RequireCompact, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
		} else {
			fresh = s.service.resolveFreshSchedulableOpenAIAccountForSelection(ctx, candidate.account, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
			if fresh != nil {
				fresh = s.service.recheckSelectedOpenAIAccountFromDBForSelection(ctx, fresh, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
			}
		}
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !compatible(ctx, fresh, req) {
			continue
		}
		if req.RequireCompact && !openAICompactAccountAllowedForRequest(req, fresh) {
			return nil, true
		}
		return &AccountSelectionResult{
			Account:              fresh,
			WeakFallback:         allowCooldownFallback,
			WeakFallbackReason:   openAIAccountWeakFallbackReason,
			BypassOpenAIHeaderTO: allowCooldownFallback,
			WaitPlan: &AccountWaitPlan{
				AccountID:      fresh.ID,
				MaxConcurrency: fresh.Concurrency,
				Timeout:        cfg.FallbackWaitTimeout,
				MaxWaiting:     cfg.FallbackMaxWaiting,
			},
		}, plan.compactBlocked
	}
	return nil, plan.compactBlocked
}

func (s *defaultOpenAIAccountScheduler) buildOpenAISelectionOrder(
	req OpenAIAccountScheduleRequest,
	plan openAIAccountLoadPlan,
) []openAIAccountCandidateScore {
	seedHash := strings.TrimSpace(req.SessionHash)
	if seedHash == "" {
		seedHash = strings.TrimSpace(req.CacheAffinityHash)
	}
	seed := newSchedulerSessionSeededOrder(
		req.GroupID,
		req.RequestedModel,
		schedulerEndpointFromOpenAIRequest(req),
		seedHash,
	)
	buildSelectionOrder := func(pool []openAIAccountCandidateScore) []openAIAccountCandidateScore {
		if len(pool) == 0 {
			return nil
		}
		if plan.stableLowTTFT {
			return buildOpenAIStableLowTTFTSelectionOrder(pool, plan.stableLowTTFTSeq)
		}
		return buildOpenAIOrderedSelectionOrderWithSeed(pool, seed)
	}
	buildCacheAwareSelectionOrder := func(pool []openAIAccountCandidateScore) []openAIAccountCandidateScore {
		compatible, fallback := splitOpenAICacheAffinityGroupCandidates(pool, req.CacheAffinityGroup)
		selectionOrder := make([]openAIAccountCandidateScore, 0, len(pool))
		selectionOrder = append(selectionOrder, buildSelectionOrder(compatible)...)
		selectionOrder = append(selectionOrder, buildSelectionOrder(fallback)...)
		return selectionOrder
	}

	if req.RequireCompact {
		selectionOrder := make([]openAIAccountCandidateScore, 0, len(plan.allCandidates))
		supported, unknown := splitOpenAICompactCandidates(plan.candidates, req.RequestedModel)
		selectionOrder = append(selectionOrder, buildCacheAwareSelectionOrder(supported)...)
		if !openAICompactStrictSupportedOnly(req) {
			selectionOrder = append(selectionOrder, buildCacheAwareSelectionOrder(unknown)...)
		}
		if len(plan.staleSnapshotCompactRetry) > 0 && s.service.schedulerSnapshot != nil && !openAICompactStrictSupportedOnly(req) {
			selectionOrder = append(selectionOrder, buildCacheAwareSelectionOrder(sortOpenAICompactRetryCandidates(plan.staleSnapshotCompactRetry))...)
		}
		return selectionOrder
	}

	selectionOrder := make([]openAIAccountCandidateScore, 0, len(plan.candidates))
	selectionOrder = append(selectionOrder, buildCacheAwareSelectionOrder(plan.candidates)...)
	return selectionOrder
}

func isOpenAIStableLowTTFTGroup(group *Group) bool {
	if group == nil || !strings.EqualFold(strings.TrimSpace(group.Platform), PlatformOpenAI) {
		return false
	}
	return group.OpenAIStableLowTTFT
}

func buildOpenAIStableLowTTFTSelectionOrder(pool []openAIAccountCandidateScore, seq uint64) []openAIAccountCandidateScore {
	known := make([]openAIAccountCandidateScore, 0, len(pool))
	unknown := make([]openAIAccountCandidateScore, 0, len(pool))
	for _, candidate := range pool {
		if candidate.hasTTFT && candidate.ttft > 0 {
			known = append(known, candidate)
			continue
		}
		unknown = append(unknown, candidate)
	}

	sortOpenAIStableLowTTFTKnown(known)
	unknown = rotateOpenAIStableLowTTFTUnknown(unknown, seq)
	if len(known) == 0 {
		return unknown
	}
	return append(known, unknown...)
}

func sortOpenAIStableLowTTFTKnown(scores []openAIAccountCandidateScore) {
	sort.SliceStable(scores, func(i, j int) bool {
		a, b := scores[i], scores[j]
		if a.cooldown != b.cooldown {
			return !a.cooldown
		}
		if a.halfOpen != b.halfOpen {
			return !a.halfOpen
		}
		if a.ttft != b.ttft {
			return a.ttft < b.ttft
		}
		aLoad := openAIAccountCandidateLoadInfo(a)
		bLoad := openAIAccountCandidateLoadInfo(b)
		if aLoad.LoadRate != bLoad.LoadRate {
			return aLoad.LoadRate < bLoad.LoadRate
		}
		if aLoad.WaitingCount != bLoad.WaitingCount {
			return aLoad.WaitingCount < bLoad.WaitingCount
		}
		if a.score != b.score {
			return a.score > b.score
		}
		if a.sortOrder != b.sortOrder {
			return a.sortOrder < b.sortOrder
		}
		if a.groupPrio != b.groupPrio {
			return a.groupPrio < b.groupPrio
		}
		return openAIAccountCandidateID(a) < openAIAccountCandidateID(b)
	})
}

func rotateOpenAIStableLowTTFTUnknown(scores []openAIAccountCandidateScore, seq uint64) []openAIAccountCandidateScore {
	ordered := buildOpenAIOrderedSelectionOrder(scores)
	if len(ordered) <= 1 {
		return ordered
	}
	shift := int(seq % uint64(len(ordered)))
	if shift == 0 {
		return ordered
	}
	return append(ordered[shift:], ordered[:shift]...)
}

func splitOpenAICompactCandidates(candidates []openAIAccountCandidateScore, requestedModel string) (supported []openAIAccountCandidateScore, unknown []openAIAccountCandidateScore) {
	for _, candidate := range candidates {
		switch openAICompactSupportTierForModel(candidate.account, requestedModel) {
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
	bypassSnapshot bool,
) (*AccountSelectionResult, bool, error) {
	compactBlocked := false
	for i := 0; i < len(selectionOrder); i++ {
		candidate := selectionOrder[i]
		var fresh *Account
		compatible := s.isAccountRequestCompatible
		if allowCooldownFallback {
			compatible = s.isAccountRequestCompatibleIgnoringCooldown
			fresh = s.service.resolveFreshOpenAIAccountIgnoringCooldownForSelection(ctx, candidate.account, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
			if fresh != nil && !s.isOpenAIWeakFallbackHealthEligible(ctx, fresh, req, schedulerHealthSnapshot{}, nil) {
				fresh = nil
			}
		} else if bypassSnapshot {
			fresh = s.service.recheckSelectedOpenAIAccountFromDBForSelection(ctx, candidate.account, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
		} else {
			fresh = s.service.resolveFreshSchedulableOpenAIAccountForSelection(ctx, candidate.account, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
		}
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !compatible(ctx, fresh, req) {
			continue
		}
		if allowCooldownFallback {
			fresh = s.service.recheckSelectedOpenAIAccountFromDBIgnoringCooldownForSelection(ctx, fresh, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
			if fresh != nil && !s.isOpenAIWeakFallbackHealthEligible(ctx, fresh, req, schedulerHealthSnapshot{}, nil) {
				fresh = nil
			}
		} else {
			fresh = s.service.recheckSelectedOpenAIAccountFromDBForSelection(ctx, fresh, req.RequestedModel, false, req.ExcludedIDs, req.RequiredCapability, openAIAccountScheduleOptionsFromRequest(req))
		}
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) || !compatible(ctx, fresh, req) {
			continue
		}
		if req.RequireCompact && !openAICompactAccountAllowedForRequest(req, fresh) {
			compactBlocked = true
			continue
		}
		result, acquireErr := s.service.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
		if acquireErr != nil {
			return nil, compactBlocked, acquireErr
		}
		if result != nil && result.Acquired {
			if !allowCooldownFallback && candidate.halfOpen && s.service.schedulerHealth != nil &&
				!s.service.schedulerHealth.tryBeginHalfOpenProbe(fresh.ID, req.RequestedModel, schedulerEndpointFromOpenAIRequest(req)) {
				if result.ReleaseFunc != nil {
					result.ReleaseFunc()
				}
				continue
			}
			s.bindOpenAISelectedAccount(ctx, req, fresh.ID)
			selection := &AccountSelectionResult{
				Account:              fresh,
				Acquired:             true,
				ReleaseFunc:          result.ReleaseFunc,
				WeakFallback:         allowCooldownFallback,
				BypassOpenAIHeaderTO: allowCooldownFallback,
			}
			if allowCooldownFallback {
				selection.WeakFallbackReason = openAIAccountWeakFallbackReason
			}
			return selection, compactBlocked, nil
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

	plan, err := s.buildOpenAISelectionPlanFromSchedulableAccounts(ctx, req, schedGroup, false)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	candidateCount := plan.loadPlan.candidateCount
	topK := plan.loadPlan.topK
	loadSkew := plan.loadPlan.loadSkew
	compactBlocked := plan.compactBlocked || (req.RequireCompact && len(plan.loadPlan.allCandidates) > 0 && len(plan.loadPlan.candidates) == 0)

	if result, blocked, acquireErr := s.tryAcquireOpenAISelectionPlan(ctx, req, plan, false, false); acquireErr != nil {
		return nil, candidateCount, topK, loadSkew, acquireErr
	} else if result != nil {
		return result, candidateCount, topK, loadSkew, nil
	} else {
		compactBlocked = compactBlocked || blocked
	}

	if freshLoadPlan, refreshed := s.refreshOpenAISelectionPlanLoad(ctx, req, plan); refreshed {
		plan = freshLoadPlan
		candidateCount = plan.loadPlan.candidateCount
		topK = plan.loadPlan.topK
		loadSkew = plan.loadPlan.loadSkew
		compactBlocked = compactBlocked || plan.compactBlocked || (req.RequireCompact && len(plan.loadPlan.allCandidates) > 0 && len(plan.loadPlan.candidates) == 0)
		if result, blocked, acquireErr := s.tryAcquireOpenAISelectionPlan(ctx, req, plan, false, false); acquireErr != nil {
			return nil, candidateCount, topK, loadSkew, acquireErr
		} else if result != nil {
			return result, candidateCount, topK, loadSkew, nil
		} else {
			compactBlocked = compactBlocked || blocked
		}
	}

	if s.service.schedulerSnapshot != nil {
		freshPlan, freshErr := s.buildOpenAISelectionPlanFromSchedulableAccounts(ctx, req, schedGroup, true)
		if freshErr != nil {
			return nil, candidateCount, topK, loadSkew, freshErr
		}
		if len(freshPlan.accounts) > 0 || len(plan.accounts) == 0 {
			candidateCount = freshPlan.loadPlan.candidateCount
			topK = freshPlan.loadPlan.topK
			loadSkew = freshPlan.loadPlan.loadSkew
			compactBlocked = compactBlocked || freshPlan.compactBlocked || (req.RequireCompact && len(freshPlan.loadPlan.allCandidates) > 0 && len(freshPlan.loadPlan.candidates) == 0)
			if result, blocked, acquireErr := s.tryAcquireOpenAISelectionPlan(ctx, req, freshPlan, false, true); acquireErr != nil {
				return nil, candidateCount, topK, loadSkew, acquireErr
			} else if result != nil {
				return result, candidateCount, topK, loadSkew, nil
			} else {
				compactBlocked = compactBlocked || blocked
			}
			if len(freshPlan.loadPlan.selectionOrder) > 0 || len(freshPlan.loadPlan.waitOrder) > 0 {
				plan = freshPlan
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

	if wait, waitCompactBlocked := s.waitPlanFromOpenAISelectionPlan(ctx, req, plan, false); wait != nil {
		return wait, candidateCount, topK, loadSkew, nil
	} else {
		compactBlocked = compactBlocked || waitCompactBlocked
	}

	if fallback, fallbackCount, fallbackCompactBlocked, fallbackErr := s.trySelectOpenAICooldownFallback(ctx, req, schedGroup, true); fallbackErr != nil {
		return nil, candidateCount, topK, loadSkew, fallbackErr
	} else if fallback != nil {
		return fallback, fallbackCount, fallbackCount, loadSkew, nil
	} else {
		compactBlocked = compactBlocked || fallbackCompactBlocked
	}

	if req.RequireCompact && compactBlocked {
		return nil, candidateCount, topK, loadSkew, ErrNoAvailableCompactAccounts
	}
	return nil, candidateCount, topK, loadSkew, noAvailableOpenAISelectionError(req.RequestedModel, compactBlocked, plan.diagnostics)
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
	// Quota auto-pause must be evaluated during the initial filter too. Without it the
	// TopK candidate pool can be filled with paused accounts and the later fresh/DB
	// rechecks won't reach healthy accounts that fell outside TopK — manifesting as
	// "no available accounts" even though healthy ones exist.
	if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
		return false
	}
	if !openAIAccountSupportsModelForSchedule(account, req.RequestedModel, req.RequireCompact, openAIAccountScheduleOptionsFromRequest(req)) {
		return false
	}
	if req.GroupID != nil && s != nil && s.service != nil &&
		s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
		s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
		return false
	}
	return accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability) &&
		accountSatisfiesOpenAIScheduleRequest(account, req)
}

func (s *defaultOpenAIAccountScheduler) isAccountRequestCompatibleIgnoringCooldown(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	if account == nil {
		return false
	}
	if !openAIAccountSupportsModelForSchedule(account, req.RequestedModel, req.RequireCompact, openAIAccountScheduleOptionsFromRequest(req)) {
		return false
	}
	if req.GroupID != nil && s != nil && s.service != nil &&
		s.service.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID) &&
		s.service.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, account, req.RequestedModel, req.RequireCompact) {
		return false
	}
	return accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability) &&
		accountSatisfiesOpenAIScheduleRequest(account, req)
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
	affinityHit := s.metrics.cacheAffinityHitTotal.Load()
	switchTotal := s.metrics.accountSwitchTotal.Load()
	latencyTotal := s.metrics.latencyMsTotal.Load()
	loadSkewTotal := s.metrics.loadSkewMilliTotal.Load()

	snapshot := OpenAIAccountSchedulerMetricsSnapshot{
		SelectTotal:              selectTotal,
		StickyPreviousHitTotal:   prevHit,
		StickySessionHitTotal:    sessionHit,
		CacheAffinityHitTotal:    affinityHit,
		LoadBalanceSelectTotal:   s.metrics.loadBalanceSelectTotal.Load(),
		AccountSwitchTotal:       switchTotal,
		SchedulerLatencyMsTotal:  latencyTotal,
		RuntimeStatsAccountCount: s.stats.size(),
	}
	if selectTotal > 0 {
		snapshot.SchedulerLatencyMsAvg = float64(latencyTotal) / float64(selectTotal)
		snapshot.StickyHitRatio = float64(prevHit+sessionHit+affinityHit) / float64(selectTotal)
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
	return s.selectAccountWithScheduler(ctx, groupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport, "", "", requireCompact, OpenAIAccountScheduleOptions{})
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
	return s.SelectAccountWithSchedulerForCapabilityAndOptions(ctx, groupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport, requiredCapability, requireCompact, OpenAIAccountScheduleOptions{})
}

func (s *OpenAIGatewayService) SelectAccountWithSchedulerForCapabilityAndOptions(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requiredCapability OpenAIEndpointCapability,
	requireCompact bool,
	options OpenAIAccountScheduleOptions,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	return s.selectAccountWithScheduler(ctx, groupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport, requiredCapability, "", requireCompact, options)
}

func (s *OpenAIGatewayService) SelectAccountWithSchedulerForImages(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredCapability OpenAIImagesCapability,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	selection, decision, err := s.selectAccountWithScheduler(ctx, groupID, "", sessionHash, requestedModel, excludedIDs, OpenAIUpstreamTransportHTTPSSE, "", requiredCapability, false, OpenAIAccountScheduleOptions{})
	if err == nil && selection != nil && selection.Account != nil {
		return selection, decision, nil
	}
	// 如果要求 native 能力（如指定了模型）但没有可用的 APIKey 账号，回退到 basic（OAuth 账号）
	if requiredCapability == OpenAIImagesCapabilityNative {
		return s.selectAccountWithScheduler(ctx, groupID, "", sessionHash, requestedModel, excludedIDs, OpenAIUpstreamTransportHTTPSSE, "", OpenAIImagesCapabilityBasic, false, OpenAIAccountScheduleOptions{})
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
	options OpenAIAccountScheduleOptions,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	ctx = s.withOpenAIQuotaAutoPauseContext(ctx)
	decision := OpenAIAccountScheduleDecision{}
	scheduler := s.getOpenAIAccountScheduler(ctx)
	if scheduler == nil {
		decision.Layer = openAIAccountScheduleLayerLoadBalance
		attachFallbackDiagnostics := func(effectiveExcludedIDs map[int64]struct{}) {
			if decision.Diagnostics.Collected || s == nil {
				return
			}
			diagReq := OpenAIAccountScheduleRequest{
				GroupID:                           groupID,
				SessionHash:                       sessionHash,
				CacheAffinityHash:                 openAICacheAffinityHashFromContext(ctx),
				PreviousResponseID:                previousResponseID,
				RequestedModel:                    requestedModel,
				RequiredTransport:                 requiredTransport,
				RequiredCapability:                requiredCapability,
				RequiredImageCapability:           requiredImageCapability,
				RequireCompact:                    requireCompact,
				RequireCodexImageGenerationBridge: options.RequireCodexImageGenerationBridge,
				ExcludedIDs:                       effectiveExcludedIDs,
			}
			diagReq.SchedulerEndpoint = schedulerEndpointFromContext(ctx, schedulerEndpointFromOpenAIRequest(diagReq))
			diagCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIAdvancedSchedulerSettingDBTimeout)
			defer cancel()
			diagnosticScheduler := &defaultOpenAIAccountScheduler{service: s}
			decision.Diagnostics = diagnosticScheduler.buildOpenAISelectionDiagnostics(
				diagCtx,
				diagReq,
				s.openAISchedulerGroupForFallback(diagCtx, groupID),
			)
		}
		if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
			effectiveExcludedIDs := cloneExcludedAccountIDs(excludedIDs)
			for {
				selection, err := s.selectAccountWithLoadAwareness(ctx, groupID, sessionHash, requestedModel, effectiveExcludedIDs, requireCompact, requiredCapability, options)
				if err != nil {
					attachFallbackDiagnostics(effectiveExcludedIDs)
					return nil, decision, attachOpenAINoAvailableDiagnostics(err, requestedModel, decision.Diagnostics)
				}
				if selection == nil || selection.Account == nil {
					return selection, decision, nil
				}
				if accountSupportsOpenAICapabilities(selection.Account, requiredCapability, requiredImageCapability) &&
					accountSatisfiesOpenAIScheduleOptions(selection.Account, options) {
					return selection, decision, nil
				}
				if selection.ReleaseFunc != nil {
					selection.ReleaseFunc()
				}
				if effectiveExcludedIDs == nil {
					effectiveExcludedIDs = make(map[int64]struct{})
				}
				if _, exists := effectiveExcludedIDs[selection.Account.ID]; exists {
					attachFallbackDiagnostics(effectiveExcludedIDs)
					return nil, decision, attachOpenAINoAvailableDiagnostics(ErrNoAvailableAccounts, requestedModel, decision.Diagnostics)
				}
				effectiveExcludedIDs[selection.Account.ID] = struct{}{}
			}
		}

		effectiveExcludedIDs := cloneExcludedAccountIDs(excludedIDs)
		for {
			selection, err := s.selectAccountWithLoadAwareness(ctx, groupID, sessionHash, requestedModel, effectiveExcludedIDs, requireCompact, requiredCapability, options)
			if err != nil {
				attachFallbackDiagnostics(effectiveExcludedIDs)
				return nil, decision, attachOpenAINoAvailableDiagnostics(err, requestedModel, decision.Diagnostics)
			}
			if selection == nil || selection.Account == nil {
				return selection, decision, nil
			}
			if s.isOpenAIAccountTransportCompatible(selection.Account, requiredTransport) &&
				accountSupportsOpenAICapabilities(selection.Account, requiredCapability, requiredImageCapability) &&
				accountSatisfiesOpenAIScheduleOptions(selection.Account, options) {
				return selection, decision, nil
			}
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			if effectiveExcludedIDs == nil {
				effectiveExcludedIDs = make(map[int64]struct{})
			}
			if _, exists := effectiveExcludedIDs[selection.Account.ID]; exists {
				attachFallbackDiagnostics(effectiveExcludedIDs)
				return nil, decision, attachOpenAINoAvailableDiagnostics(ErrNoAvailableAccounts, requestedModel, decision.Diagnostics)
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
	cacheAffinityHash := openAICacheAffinityHashFromContext(ctx)
	var cacheAffinityAccountID int64
	if cacheAffinityHash != "" && s.cache != nil {
		if accountID, err := s.getStickySessionAccountID(ctx, groupID, cacheAffinityHash); err == nil && accountID > 0 {
			cacheAffinityAccountID = accountID
		}
	}
	cacheAffinityGroup := ""
	if cacheAffinityAccountID > 0 {
		if account, err := s.getSchedulableAccount(ctx, cacheAffinityAccountID); err == nil && account != nil {
			cacheAffinityGroup = openAIAccountCacheAffinityGroup(account)
		}
	}

	req := OpenAIAccountScheduleRequest{
		GroupID:                           groupID,
		SessionHash:                       sessionHash,
		CacheAffinityHash:                 cacheAffinityHash,
		StickyAccountID:                   stickyAccountID,
		CacheAffinityAccountID:            cacheAffinityAccountID,
		CacheAffinityGroup:                cacheAffinityGroup,
		PreviousResponseID:                previousResponseID,
		RequestedModel:                    requestedModel,
		RequiredTransport:                 requiredTransport,
		RequiredCapability:                requiredCapability,
		RequiredImageCapability:           requiredImageCapability,
		RequireCompact:                    requireCompact,
		RequireCodexImageGenerationBridge: options.RequireCodexImageGenerationBridge,
		ExcludedIDs:                       excludedIDs,
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
	outcome := schedulerResultReportOutcome{}
	slowStreamingSuccess := success && schedulerStreamingTTFTIsSlow(endpoint, firstTokenMs)
	if success {
		if !slowStreamingSuccess {
			s.recoverOpenAIAccountCircuit(accountID)
			s.stopOpenAIAccountCircuitProbe(accountID, model, endpoint)
		}
	} else {
		if s.isOpenAIAccountCircuitHalfOpenInFlight(accountID, time.Now()) {
			// Non-failover errors (for example a user 400) still prove the
			// upstream account is reachable. Network/5xx probe failures are
			// reported through ReportOpenAIAccountScheduleFailure and reopen
			// the circuit there.
			s.recoverOpenAIAccountCircuit(accountID)
		}
	}
	if s.schedulerHealth != nil {
		outcome = s.openAISchedulerReporter().report(schedulerResultReport{
			AccountID:    accountID,
			Model:        model,
			Endpoint:     endpoint,
			Success:      success,
			FirstTokenMs: firstTokenMs,
		})
	}
	if outcome.SlowStreamingSuccess {
		s.maybeStartOpenAIAccountCircuitProbe(accountID, model, endpoint, schedulerSlowTTFTCategory)
	}
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler != nil {
		scheduler.ReportResult(accountID, success, firstTokenMs)
	}
}

func (s *OpenAIGatewayService) ReportOpenAIAccountScheduleTerminal(accountID int64, model, endpoint string) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiAccountCircuitHalfOpen.Delete(accountID)
	if s.schedulerHealth != nil {
		s.schedulerHealth.reportNeutral(accountID, model, endpoint)
	}
}

func (s *OpenAIGatewayService) ReportOpenAIAccountScheduleFailure(accountID int64, model, endpoint string, failoverErr *UpstreamFailoverError) {
	if s == nil || accountID <= 0 {
		return
	}
	statusCode, _, _ := schedulerFailureInputs(failoverErr)
	outcome := s.openAISchedulerReporter().report(schedulerResultReport{
		AccountID:   accountID,
		Model:       model,
		Endpoint:    endpoint,
		Success:     false,
		FailoverErr: failoverErr,
	})
	category := outcome.FailureCategory
	cooldown := outcome.Cooldown
	if category == "auth" {
		reason := "openai_auth_error"
		s.reopenOpenAIAccountCircuit(accountID, reason, cooldown)
	}
	s.maybeStartOpenAIAccountCircuitProbe(accountID, model, endpoint, category)
	if category == "transient" || category == "transient_transport" || category == "transient_timeout" || category == "compact_bad_output" || category == "empty_output" || category == "rate_limit" || category == "model_unsupported" {
		reason := "openai_request_error"
		if category == "transient_transport" {
			reason = "openai_transport_error"
		} else if category == "transient_timeout" {
			reason = "openai_timeout"
		} else if category == "compact_bad_output" {
			reason = "openai_compact_bad_output"
		} else if category == "empty_output" {
			reason = "openai_empty_output"
		} else if category == "transient" && isOpenAITransient5xxStatus(statusCode) {
			reason = "openai_transient_5xx"
		} else if category == "rate_limit" {
			reason = "openai_rate_limit"
		} else if category == "model_unsupported" {
			reason = "openai_model_unsupported"
		}
		slog.Info("account_circuit_open",
			"account_id", accountID,
			"model", model,
			"endpoint", endpoint,
			"status_code", statusCode,
			"reason", reason,
			"until", time.Now().Add(cooldown),
			"scope", "account_model_endpoint",
			"persisted", false,
		)
	}
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler != nil {
		scheduler.ReportResult(accountID, false, nil)
	}
}

func (s *OpenAIGatewayService) openAISchedulerReporter() schedulerResultReporter {
	if s == nil {
		return schedulerResultReporter{source: PlatformOpenAI}
	}
	return schedulerResultReporter{health: s.schedulerHealth, source: PlatformOpenAI}
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
		return 3
	}
	_ = ctx
	_ = groupID
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

func (s *OpenAIGatewayService) openAIStickyEscapeConfig() openAIStickyEscapeConfig {
	if s != nil && s.cfg != nil {
		cfg := s.cfg.Gateway.OpenAIScheduler
		enabled := cfg.StickyEscapeEnabled
		if !enabled && cfg.StickyEscapeTTFTMs == 0 && cfg.StickyEscapeErrorRate == 0 {
			enabled = true
		}
		ttftMs := float64(cfg.StickyEscapeTTFTMs)
		if ttftMs <= 0 {
			ttftMs = 15000
		}
		errorRate := cfg.StickyEscapeErrorRate
		if errorRate < 0 || errorRate > 1 {
			errorRate = 0.5
		}
		if errorRate == 0 && cfg.StickyEscapeTTFTMs == 0 && cfg.StickyEscapeErrorRate == 0 {
			errorRate = 0.5
		}
		return openAIStickyEscapeConfig{
			enabled:   enabled,
			ttftMs:    ttftMs,
			errorRate: errorRate,
		}
	}
	return openAIStickyEscapeConfig{
		enabled:   true,
		ttftMs:    15000,
		errorRate: 0.5,
	}
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
