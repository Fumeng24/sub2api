package service

import (
	"context"
	"database/sql"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// groupAutoSortLeaderLockKey 保证多实例下只有一个实例执行分组自动排序，避免重复重排。
	groupAutoSortLeaderLockKey = "group:autosort:leader"
	// groupAutoSortLeaderLockTTL 略大于一个周期，给崩溃恢复留窗口。
	groupAutoSortLeaderLockTTL = 3 * time.Minute
	// groupAutoSortDefaultInterval 分组自动排序默认执行间隔。
	groupAutoSortDefaultInterval = time.Minute
	// groupAutoSortRunTimeout 单轮全量重排的超时。
	groupAutoSortRunTimeout = 2 * time.Minute
	// groupAutoSortProbeWindow is the primary health window. Account probes run
	// every 60 seconds in production, so this gives roughly fifteen fresh
	// observations without retaining stale upstream behavior for half an hour.
	groupAutoSortProbeWindow = 15 * time.Minute
	// A few successful probe samples are enough to leave probation, but a
	// missing/stale probe never masquerades as a healthy account.
	groupAutoSortProbeMinSamples = 3
	// Real user traffic is a model-specific overlay. Keep it on the same short
	// window so an unstable upstream is not hidden by old traffic history.
	groupAutoSortExperienceWindow = groupAutoSortProbeWindow
	// groupAutoSortExperienceMinAttempts 达到该样本量后才使用完整失败率分层。
	groupAutoSortExperienceMinAttempts int64 = 20
	// groupAutoSortExperienceMinModelAttempts 模型级异常的最小判定样本。
	// Model-level rates are noisier than account-level rates, so require ten
	// requests before a model can move an account into a degraded tier.
	groupAutoSortExperienceMinModelAttempts int64 = 10
	// groupAutoSortRateMinFirstTokenSamples 避免少量慢样本触发长期降级。
	groupAutoSortRateMinFirstTokenSamples int64 = 8
	// groupAutoSortExperienceMaxP95FirstTokenMs 是体验分组健康主力层的首字上限。
	groupAutoSortExperienceMaxP95FirstTokenMs = 25000.0
	// groupAutoSortExperienceScoreDeadband 同层分差不足时保持原顺序，避免周期性抖动。
	groupAutoSortExperienceScoreDeadband = 0.1
	// groupAutoSortScoreImprovement is the minimum relative improvement needed
	// to move an account within the same health tier. A small absolute floor is
	// useful when scores are close to zero.
	groupAutoSortScoreImprovement = 0.10
	groupAutoSortMinResidence     = 10 * time.Minute
	// A stale/disabled probe is not health evidence. The normal production
	// interval is 60s; the per-monitor interval is used when available and is
	// clamped so an accidentally huge interval cannot keep stale green state
	// forever.
	groupAutoSortMonitorMinFreshness = 2 * time.Minute
	groupAutoSortMonitorMaxFreshness = 10 * time.Minute
	// groupAutoSortUpstreamRateFreshness 超过三个上游刷新周期的数据不参与倍率排序。
	groupAutoSortUpstreamRateFreshness = 15 * time.Minute
)

// groupAutoSortAdmin 是 GroupAutoSortService 依赖的 AdminService 子集，便于测试替换。
type groupAutoSortAdmin interface {
	GetAllGroups(ctx context.Context) ([]Group, error)
	GetGroupAccountScheduling(ctx context.Context, groupID int64) ([]AccountSchedulingEntry, error)
	UpdateGroupAccountScheduling(ctx context.Context, groupID int64, configs []AccountSchedulingConfig) error
}

// groupAutoSortAvailabilityProvider 提供按 account_id 索引的探针状态和时间线。
// 自动排序从时间线计算最近 15 分钟窗口，旧的 1 小时聚合只用于兼容没有时间线的状态提供者。
type groupAutoSortAvailabilityProvider interface {
	StatusByAccountID(ctx context.Context) (map[int64]*AccountMonitorStatus, error)
}

// GroupAutoSortService periodically reorders enabled groups with one canonical
// comparator: hard scheduling gates, real-request/model stability, latency and
// cache signals, then (only for a near-tie) upstream rate.  The legacy
// primary/backup role is metadata only and never changes the order.
// The legacy auto_sort_config basis is accepted for API compatibility but does
// not select a different ranking policy.
//
// 仅 enabled=true 的分组参与。结果只写 account_groups，不修改账号全局 priority。
type GroupAutoSortService struct {
	admin        groupAutoSortAdmin
	availability groupAutoSortAvailabilityProvider
	experience   groupAutoSortExperienceProvider
	rateProvider groupAutoSortRateProvider
	interval     time.Duration

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup

	lockCache     LeaderLockCache
	db            *sql.DB
	instanceID    string
	orderMu       sync.Mutex
	lastReorderAt map[int64]time.Time
}

func (s *GroupAutoSortService) SetRateProvider(provider groupAutoSortRateProvider) {
	if s != nil {
		s.rateProvider = provider
	}
}

// SetExperienceProvider injects real user-request statistics. A nil provider
// keeps legacy bases operational and makes experience fall back to probes.
func (s *GroupAutoSortService) SetExperienceProvider(provider groupAutoSortExperienceProvider) {
	if s != nil {
		s.experience = provider
	}
}

// NewGroupAutoSortService 构造分组自动排序服务。availability 可为 nil（此时 availability 依据的分组被跳过）。
func NewGroupAutoSortService(admin groupAutoSortAdmin, availability groupAutoSortAvailabilityProvider, interval time.Duration) *GroupAutoSortService {
	if interval <= 0 {
		interval = groupAutoSortDefaultInterval
	}
	return &GroupAutoSortService{
		admin:         admin,
		availability:  availability,
		interval:      interval,
		stopCh:        make(chan struct{}),
		instanceID:    uuid.NewString(),
		lastReorderAt: make(map[int64]time.Time),
	}
}

// SetLeaderLock 注入 leader-lock，多实例时只有 leader 执行重排。两者为 nil 时不做选主（单实例/测试）。
func (s *GroupAutoSortService) SetLeaderLock(lockCache LeaderLockCache, db *sql.DB) {
	if s == nil {
		return
	}
	s.lockCache = lockCache
	s.db = db
	s.loadPersistedResidence(context.Background())
}

// loadPersistedResidence restores the anti-thrashing clock from successful
// group_changed events. The target account order is already persisted in
// account_groups; this timestamp prevents a restart from immediately writing
// another order before the configured minimum residence has elapsed.
func (s *GroupAutoSortService) loadPersistedResidence(parent context.Context) {
	if s == nil || s.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
		SELECT group_id, MAX(created_at)
		FROM scheduler_history
		WHERE event_type = 'group_changed'
		  AND group_id IS NOT NULL
		  AND created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY group_id
	`)
	if err != nil {
		log.Printf("[GroupAutoSort] load persisted residence failed: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()
	s.orderMu.Lock()
	defer s.orderMu.Unlock()
	for rows.Next() {
		var groupID int64
		var changedAt time.Time
		if err := rows.Scan(&groupID, &changedAt); err != nil {
			log.Printf("[GroupAutoSort] scan persisted residence failed: %v", err)
			return
		}
		if groupID > 0 && !changedAt.IsZero() {
			s.lastReorderAt[groupID] = changedAt
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[GroupAutoSort] read persisted residence failed: %v", err)
	}
}

func (s *GroupAutoSortService) Start() {
	if s == nil || s.admin == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *GroupAutoSortService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *GroupAutoSortService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), groupAutoSortRunTimeout)
	defer cancel()

	groups, err := s.admin.GetAllGroups(ctx)
	if err != nil {
		log.Printf("[GroupAutoSort] list groups failed: %v", err)
		return
	}

	// 先判断有没有需要排序的分组，没有就不去抢锁。
	hasWork := false
	for i := range groups {
		if groups[i].AutoSortConfig.Enabled {
			hasWork = true
			break
		}
	}
	if !hasWork {
		return
	}

	// 多实例护栏：只有 leader 执行重排，避免 N× 重复写。
	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, groupAutoSortLeaderLockKey, s.instanceID, groupAutoSortLeaderLockTTL)
	if !ok {
		return
	}
	defer release()

	// 所有策略都先执行健康分层；按需取一次聚合快照供各分组复用。
	var monitorByAccount map[int64]*AccountMonitorStatus
	for i := range groups {
		g := groups[i]
		if !g.AutoSortConfig.Enabled {
			continue
		}
		basis := g.AutoSortConfig.NormalizedBasis()
		if monitorByAccount == nil && s.availability != nil {
			monitorByAccount, err = s.availability.StatusByAccountID(ctx)
			if err != nil {
				log.Printf("[GroupAutoSort] load monitor status failed: %v", err)
				monitorByAccount = map[int64]*AccountMonitorStatus{}
			}
		}
		s.sortGroup(ctx, g, basis, monitorByAccount)
	}
}

type groupAutoSortRanked struct {
	entry        AccountSchedulingEntry
	tier         int
	key          float64
	hasKey       bool
	rate         float64
	hasRate      bool
	currentOrder int
}

// sortGroup 计算单个分组的目标顺序并仅更新该分组的成员关系。
func (s *GroupAutoSortService) sortGroup(ctx context.Context, g Group, _ string, monitorByAccount map[int64]*AccountMonitorStatus) {
	entries, err := s.admin.GetGroupAccountScheduling(ctx, g.ID)
	if err != nil {
		log.Printf("[GroupAutoSort] list scheduling for group=%d failed: %v", g.ID, err)
		return
	}
	if len(entries) <= 1 {
		return
	}

	accountIDs := make([]int64, 0, len(entries))
	for i := range entries {
		if entries[i].Account != nil {
			accountIDs = append(accountIDs, entries[i].Account.ID)
		}
	}
	experienceByAccount := map[int64]*groupAutoSortExperienceStats{}
	if s.experience != nil {
		now := time.Now()
		experienceSince := now.Add(-groupAutoSortExperienceWindow)
		experienceByAccount, err = s.experience.StatsByAccountID(ctx, g.ID, accountIDs, experienceSince)
		if err != nil {
			log.Printf("[GroupAutoSort] load experience stats group=%d failed: %v", g.ID, err)
			experienceByAccount = map[int64]*groupAutoSortExperienceStats{}
		}
	}
	rateByAccount := map[int64]float64{}
	if s.rateProvider != nil {
		rateByAccount, err = s.rateProvider.RatesByAccountID(ctx, accountIDs, time.Now())
		if err != nil {
			log.Printf("[GroupAutoSort] load upstream rates group=%d failed: %v", g.ID, err)
			rateByAccount = map[int64]float64{}
		}
	}

	items := make([]groupAutoSortRanked, 0, len(entries))
	for i := range entries {
		acc := entries[i].Account
		if acc == nil {
			continue
		}
		r := groupAutoSortRanked{entry: entries[i], currentOrder: i}
		if r.entry.AccountID <= 0 {
			r.entry.AccountID = acc.ID
		}
		monitor := monitorByAccount[acc.ID]
		experience := experienceByAccount[acc.ID]
		// Every group uses the same canonical health comparator. The legacy
		// basis remains accepted for API compatibility, but no basis can make a
		// cheap/fast account outrank a materially more stable account.
		r.tier = groupAutoSortHealthTier(acc, monitor, experience)
		r.key = groupAutoSortCanonicalScore(monitor, experience)
		r.hasKey = monitor != nil || experience != nil
		upstreamRate, hasUpstreamRate := rateByAccount[acc.ID]
		if rate, ok := finalRateForAccountWithUpstreamRateAt(acc, upstreamRate, hasUpstreamRate, time.Now()); ok {
			r.rate = rate
			r.hasRate = true
		}
		items = append(items, r)
	}
	if len(items) <= 1 {
		return
	}

	// One comparator and one deadband are used for every group/basis. This
	// avoids the old frontend/backend divergence between rate, latency and
	// experience modes.
	groupAutoSortWithHysteresis(items)

	configs := make([]AccountSchedulingConfig, 0, len(items))
	changed := false
	for idx := range items {
		order := idx + 1
		entry := items[idx].entry
		if entry.SortOrder != order || entry.Priority != order {
			changed = true
		}
		weight := entry.Weight
		if weight <= 0 {
			weight = 1
		}
		configs = append(configs, AccountSchedulingConfig{
			AccountID:            entry.AccountID,
			Priority:             order,
			Role:                 entry.Role,
			Weight:               weight,
			SortOrder:            order,
			SchedulingConfigured: true,
		})
	}
	if !changed {
		return
	}
	reorderNow := time.Now()
	if !s.canReorderGroup(g.ID, reorderNow) && !groupAutoSortHasUrgentTierMove(items) {
		return
	}
	if err := s.admin.UpdateGroupAccountScheduling(ctx, g.ID, configs); err != nil {
		log.Printf("[GroupAutoSort] update group order group=%d failed: %v", g.ID, err)
		return
	}
	s.markReorderedGroup(g.ID, reorderNow)
}

func (s *GroupAutoSortService) canReorderGroup(groupID int64, now time.Time) bool {
	if s == nil || groupID <= 0 {
		return true
	}
	s.orderMu.Lock()
	defer s.orderMu.Unlock()
	if previous, ok := s.lastReorderAt[groupID]; ok && now.Sub(previous) < groupAutoSortMinResidence {
		return false
	}
	return true
}

func (s *GroupAutoSortService) markReorderedGroup(groupID int64, now time.Time) {
	if s == nil || groupID <= 0 {
		return
	}
	s.orderMu.Lock()
	defer s.orderMu.Unlock()
	s.lastReorderAt[groupID] = now
}

// groupAutoSortHasUrgentTierMove bypasses the minimum-residence window only
// when a hard scheduling failure changed an account's position. This keeps
// normal score jitter sticky while allowing a fresh probe failure or an
// account-level scheduling block to leave the serving path immediately.
func groupAutoSortHasUrgentTierMove(items []groupAutoSortRanked) bool {
	for desiredPosition, item := range items {
		if item.tier >= 3 && item.currentOrder != desiredPosition {
			return true
		}
	}
	return false
}

func groupAutoSortWithHysteresis(items []groupAutoSortRanked) {
	for position := 0; position < len(items)-1; position++ {
		best := position
		for candidate := position + 1; candidate < len(items); candidate++ {
			if groupAutoSortStrictlyBefore(items[candidate], items[best]) {
				best = candidate
			}
		}
		if best == position || !groupAutoSortExperienceShouldPromote(items[best], items[position]) {
			continue
		}
		candidate := items[best]
		copy(items[position+1:best+1], items[position:best])
		items[position] = candidate
	}
}

// Kept as a compatibility helper for package tests and older callers.
func groupAutoSortExperienceWithHysteresis(items []groupAutoSortRanked) {
	groupAutoSortWithHysteresis(items)
}

func groupAutoSortExperienceShouldPromote(candidate, incumbent groupAutoSortRanked) bool {
	if candidate.tier != incumbent.tier {
		return candidate.tier < incumbent.tier
	}
	if candidate.hasKey != incumbent.hasKey {
		return candidate.hasKey
	}
	if !candidate.hasKey {
		return false
	}
	// A material health improvement is required to move an account within the
	// same tier.  Role is intentionally not considered: primary/backup is a
	// legacy label, not a quality signal.
	threshold := math.Max(groupAutoSortExperienceScoreDeadband, math.Abs(incumbent.key)*groupAutoSortScoreImprovement)
	if candidate.key+threshold < incumbent.key {
		return true
	}
	// Cost is a tie-breaker only while the two health scores are within the
	// deadband. It can never override a material stability difference.
	if math.Abs(candidate.key-incumbent.key) <= threshold && candidate.hasRate && incumbent.hasRate && candidate.rate < incumbent.rate {
		return true
	}
	return false
}

func groupAutoSortStrictlyBefore(a, b groupAutoSortRanked) bool {
	if a.tier != b.tier {
		return a.tier < b.tier
	}
	if a.hasKey != b.hasKey {
		return a.hasKey
	}
	if a.hasKey && a.key != b.key {
		return a.key < b.key
	}
	if a.hasRate != b.hasRate {
		return a.hasRate
	}
	if a.hasRate && a.rate != b.rate {
		return a.rate < b.rate
	}
	return a.currentOrder < b.currentOrder
}

// groupAutoSortProbeStats is the short-lived account-level health baseline
// derived from the configured account probe. It deliberately describes the
// upstream account as a whole; model-specific user traffic is layered on top
// only when enough samples exist.
type groupAutoSortProbeStats struct {
	known           bool
	samples         int
	failures        int
	degraded        int
	availability    float64
	p95LatencyMs    float64
	latestStatus    string
	latestIsFailure bool
}

func (p groupAutoSortProbeStats) failureRate() float64 {
	if p.samples <= 0 {
		return 0
	}
	return float64(p.failures) / float64(p.samples)
}

// groupAutoSortProbeWindowStats converts the account monitor timeline into a
// fresh 15-minute window. The status aggregate's one-hour fields are retained
// only as a compatibility fallback for synthetic/legacy callers that do not
// supply a timeline.
func groupAutoSortProbeWindowStats(status *AccountMonitorStatus, now time.Time) groupAutoSortProbeStats {
	if status == nil || !groupAutoSortMonitorKnown(status, now) {
		return groupAutoSortProbeStats{}
	}
	windowStart := now.Add(-groupAutoSortProbeWindow)
	latencies := make([]float64, 0, len(status.Timeline))
	stats := groupAutoSortProbeStats{latestStatus: status.LatestStatus}
	for _, check := range status.Timeline {
		if check == nil || check.CheckedAt.IsZero() || check.CheckedAt.Before(windowStart) || check.CheckedAt.After(now.Add(groupAutoSortMonitorMinFreshness)) {
			continue
		}
		if stats.latestStatus == "" {
			stats.latestStatus = check.Status
		}
		switch check.Status {
		case MonitorStatusOperational:
			stats.samples++
		case MonitorStatusDegraded:
			stats.samples++
			stats.degraded++
		case MonitorStatusFailed, MonitorStatusError:
			stats.samples++
			stats.failures++
		default:
			continue
		}
		if check.LatencyMs != nil && *check.LatencyMs > 0 {
			latencies = append(latencies, float64(*check.LatencyMs))
		} else if check.PingLatencyMs != nil && *check.PingLatencyMs > 0 {
			latencies = append(latencies, float64(*check.PingLatencyMs))
		}
	}
	if stats.samples == 0 {
		// Unit tests and older status providers may only expose the aggregate.
		// Use one fresh synthetic sample rather than treating a known-good probe
		// as absent, while production timelines always take the path above.
		if status.LatestStatus == MonitorStatusOperational || status.LatestStatus == MonitorStatusDegraded ||
			status.LatestStatus == MonitorStatusFailed || status.LatestStatus == MonitorStatusError {
			stats.samples = 1
			stats.availability = status.Availability1h
			if status.LatestStatus == MonitorStatusFailed || status.LatestStatus == MonitorStatusError {
				stats.failures = 1
			} else if status.LatestStatus == MonitorStatusDegraded {
				stats.degraded = 1
			}
			if status.AvgLatency1h != nil && *status.AvgLatency1h > 0 {
				latencies = append(latencies, *status.AvgLatency1h)
			}
		}
	}
	if stats.samples == 0 {
		return groupAutoSortProbeStats{}
	}
	if stats.availability == 0 || len(status.Timeline) > 0 {
		stats.availability = float64(stats.samples-stats.failures) * 100 / float64(stats.samples)
	}
	stats.known = true
	stats.latestIsFailure = stats.latestStatus == MonitorStatusFailed || stats.latestStatus == MonitorStatusError
	if len(latencies) > 0 {
		sort.Float64s(latencies)
		index := int(math.Ceil(float64(len(latencies))*0.95)) - 1
		if index < 0 {
			index = 0
		}
		if index >= len(latencies) {
			index = len(latencies) - 1
		}
		stats.p95LatencyMs = latencies[index]
	}
	return stats
}

func groupAutoSortHealthTier(acc *Account, monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) int {
	if acc == nil || acc.SchedulingBlockReasonAt(time.Now()) != AccountSchedulingBlockNone {
		return 4
	}
	probe := groupAutoSortProbeWindowStats(monitor, time.Now())
	// A current probe failure is an immediate account-level demotion. It is not
	// diluted by a good model-specific traffic sample.
	if probe.known && (probe.latestIsFailure || probe.availability < 80) {
		return 3
	}
	if experience != nil {
		failureRate := experience.failureRate()
		worstModelRate := experience.worstModelFailureRate(groupAutoSortExperienceMinModelAttempts)
		if (experience.attempts() >= groupAutoSortExperienceMinAttempts && failureRate >= 0.20) || worstModelRate >= 0.30 {
			return 3
		}
		if (experience.attempts() >= groupAutoSortExperienceMinAttempts && failureRate >= 0.05) ||
			worstModelRate >= 0.12 ||
			(experience.FirstTokenSamples >= groupAutoSortRateMinFirstTokenSamples && experience.P95FirstTokenMs > groupAutoSortExperienceMaxP95FirstTokenMs) {
			return 2
		}
	}
	if probe.known {
		if probe.latestStatus == MonitorStatusDegraded || probe.degraded > 0 || probe.availability < 95 ||
			probe.p95LatencyMs > groupAutoSortExperienceMaxP95FirstTokenMs {
			return 2
		}
		if probe.samples < groupAutoSortProbeMinSamples || experience == nil || experience.attempts() < groupAutoSortExperienceMinAttempts {
			return 1
		}
		return 0
	}
	// No fresh probe means the account is probationary, even when it has old
	// traffic data. This keeps stopped probes from leaving a stale green rank.
	return 1
}

// groupAutoSortRateHealthTier is retained for compatibility with older tests
// and callers; all group bases now use the same probe-first policy.
func groupAutoSortRateHealthTier(acc *Account, monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) int {
	return groupAutoSortHealthTier(acc, monitor, experience)
}

func groupAutoSortExperienceScore(monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) float64 {
	score := 0.0
	probe := groupAutoSortProbeWindowStats(monitor, time.Now())
	if !probe.known {
		score += 40
	} else {
		// Probe health is the dominant account/site baseline.
		score += 1200 * probe.failureRate()
		score += math.Max(0, 100-probe.availability) * 2
		if probe.latestIsFailure {
			score += 100
		}
		if probe.latestStatus == MonitorStatusDegraded || probe.degraded > 0 {
			score += 8
		}
		if probe.p95LatencyMs > 0 {
			score += math.Min(probe.p95LatencyMs, 120000) / 1000
		}
		if probe.samples < groupAutoSortProbeMinSamples {
			score += 20
		}
	}
	if experience == nil || experience.attempts() == 0 {
		score += 25
	} else {
		// Real traffic is a model-specific overlay, deliberately weaker than the
		// account-level probe so it cannot hide a failing upstream endpoint.
		score += 350 * experience.failureRate()
		score += 150 * experience.failoverRate()
		score += 200 * experience.worstModelFailureRate(groupAutoSortExperienceMinModelAttempts)
		if experience.attempts() < groupAutoSortExperienceMinAttempts {
			score += 15
		}
		if experience.FirstTokenSamples > 0 {
			score += math.Min(experience.P95FirstTokenMs, 120000) / 2000
		} else {
			score += 15
		}
		if experience.DurationSamples > 0 {
			score += math.Min(experience.P95DurationMs, 600000) / 20000
		}
		if experience.SuccessCount >= 4 {
			score += 2 * (1 - experience.cacheHitRate())
		} else {
			score += 1
		}
	}
	return score
}

// groupAutoSortCanonicalScore is deliberately basis-independent. Stability and
// model-level success dominate; latency and cache are secondary signals, while
// rate is kept out of the score and used only as a final tie-breaker.
func groupAutoSortCanonicalScore(monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) float64 {
	return groupAutoSortExperienceScore(monitor, experience)
}

// groupAutoSortMonitorKnown reports whether a probe can be trusted as current
// health evidence. AccountMonitorService aggregates historical checks, so a
// disabled or stopped monitor can otherwise leave an account looking green
// to the auto-sort worker indefinitely. Tests and legacy callers that provide
// a synthetic status without monitor metadata retain the old behavior.
func groupAutoSortMonitorKnown(st *AccountMonitorStatus, now time.Time) bool {
	if st == nil {
		return false
	}
	if !st.Enabled && (st.MonitorID > 0 || st.LastCheckedAt != nil || len(st.Timeline) > 0) {
		return false
	}
	if st.LastCheckedAt == nil {
		return true
	}
	interval := time.Duration(st.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	maxAge := 2 * interval
	if maxAge < groupAutoSortMonitorMinFreshness {
		maxAge = groupAutoSortMonitorMinFreshness
	}
	if maxAge > groupAutoSortMonitorMaxFreshness {
		maxAge = groupAutoSortMonitorMaxFreshness
	}
	age := now.Sub(*st.LastCheckedAt)
	return age >= -groupAutoSortMonitorMinFreshness && age <= maxAge
}

func monitorSortTier(acc *Account, st *AccountMonitorStatus, requireLatency bool) int {
	if acc == nil || acc.SchedulingBlockReasonAt(time.Now()) != AccountSchedulingBlockNone {
		return 3
	}
	if !groupAutoSortMonitorKnown(st, time.Now()) {
		return 3
	}
	switch st.LatestStatus {
	case MonitorStatusFailed, MonitorStatusError:
		return 2
	case MonitorStatusOperational, MonitorStatusDegraded:
	default:
		return 3
	}
	if requireLatency && st.AvgLatency1h == nil {
		return 3
	}
	if st.Availability1h < 80 {
		return 2
	}
	if st.Availability1h < 95 {
		return 1
	}
	return 0
}

// finalRateForAccount 复刻后台当前倍率语义：
//  1. manual_rate
//  2. 未过期的 upstream_billing_probe 动态倍率 x rate_scale
//  3. 明确配置的非默认 account.rate_multiplier
//
// 上游倍率探测失败时必须按“无数据”处理，不能使用历史缓存参与排序。
//
// 返回 (倍率, 是否可得)。
func finalRateForAccount(acc *Account) (float64, bool) {
	return finalRateForAccountAt(acc, time.Now())
}

func finalRateForAccountAt(acc *Account, now time.Time) (float64, bool) {
	return finalRateForAccountWithUpstreamRateAt(acc, 0, false, now)
}

func finalRateForAccountWithUpstreamRateAt(acc *Account, upstreamRate float64, hasUpstreamRate bool, now time.Time) (float64, bool) {
	if acc == nil {
		return 0, false
	}
	if manual, ok := extraFloat(acc.Extra, "manual_rate"); ok {
		return validGroupAutoSortRate(manual)
	}
	scale := 1.0
	if configuredScale, exists := extraFloat(acc.Extra, "rate_scale"); exists {
		scale = configuredScale
	}
	if hasUpstreamRate {
		return validGroupAutoSortRate(upstreamRate * scale)
	}
	// 绑定上游管理的账号必须以其最新探测为准；失败时不回落到历史同步值。
	if acc.UpstreamID != nil {
		return 0, false
	}
	if upstream, ok := openAIFreshUpstreamBillingRate(acc, now); ok {
		return validGroupAutoSortRate(upstream * scale)
	}
	// 1.0 是账号字段的数据库默认值，无法证明上游倍率；只有显式的非默认值可兜底。
	if acc.RateMultiplier != nil && *acc.RateMultiplier != 1 {
		return validGroupAutoSortRate(*acc.RateMultiplier)
	}
	return 0, false
}

func validGroupAutoSortRate(rate float64) (float64, bool) {
	if rate < 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0, false
	}
	return rate, true
}

// extraFloat 从 extra(JSONB) 读取数值字段；JSON 反序列化后数值统一是 float64，
// 但也容忍 int/json.Number 之外的少数情形。
func extraFloat(extra map[string]any, key string) (float64, bool) {
	if extra == nil {
		return 0, false
	}
	v, ok := extra[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
