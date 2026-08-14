package service

import (
	"context"
	"database/sql"
	"log"
	"math"
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
	// groupAutoSortExperienceWindow 使用足够短的真实请求窗口保持实时性，同时避免被单次抖动支配。
	groupAutoSortExperienceWindow = 30 * time.Minute
	// groupAutoSortExperienceLongWindow supplies a decayed baseline so a quiet
	// account is not promoted (or quarantined) from a handful of recent calls.
	groupAutoSortExperienceLongWindow = 24 * time.Hour
	// groupAutoSortExperienceMinAttempts 达到该样本量后才使用完整失败率分层。
	groupAutoSortExperienceMinAttempts int64 = 20
	// groupAutoSortExperienceMinModelAttempts 模型级异常的最小判定样本。
	// Model-level rates are noisier than account-level rates, so require ten
	// requests before a model can move an account into a degraded tier.
	groupAutoSortExperienceMinModelAttempts int64 = 10
	// groupAutoSortRateMaxFailureRate 是低价分组进入已验证主力层的最大上游尝试失败率。
	groupAutoSortRateMaxFailureRate = 0.10
	// groupAutoSortRateSevereFailureRate 将持续失败账号直接降到故障层。
	groupAutoSortRateSevereFailureRate = 0.20
	// groupAutoSortRateMaxP95FirstTokenMs 是低价分组已验证主力层的首字上限。
	groupAutoSortRateMaxP95FirstTokenMs = 30000.0
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

// groupAutoSortAvailabilityProvider 提供按 account_id 索引的近 1 小时可用率，用于 availability 依据排序。
type groupAutoSortAvailabilityProvider interface {
	StatusByAccountID(ctx context.Context) (map[int64]*AccountMonitorStatus, error)
}

// GroupAutoSortService periodically reorders enabled groups with one canonical
// comparator: hard scheduling gates, real-request/model stability, latency and
// cache signals, then reserve role and (only for a near-tie) upstream rate.
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
		recentSince := now.Add(-groupAutoSortExperienceWindow)
		longSince := now.Add(-groupAutoSortExperienceLongWindow)
		if weighted, ok := s.experience.(groupAutoSortWeightedExperienceProvider); ok {
			experienceByAccount, err = weighted.StatsByAccountIDWeighted(ctx, g.ID, accountIDs, recentSince, longSince)
		} else {
			experienceByAccount, err = s.experience.StatsByAccountID(ctx, g.ID, accountIDs, recentSince)
		}
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
	if !s.canReorderGroup(g.ID, time.Now()) {
		return
	}
	if err := s.admin.UpdateGroupAccountScheduling(ctx, g.ID, configs); err != nil {
		log.Printf("[GroupAutoSort] update group order group=%d failed: %v", g.ID, err)
		return
	}
	s.markReorderedGroup(g.ID, time.Now())
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
	// A role is only a tie-breaker. A backup may promote over a primary when it
	// has a materially better health score, but not for a small fluctuation.
	threshold := math.Max(groupAutoSortExperienceScoreDeadband, math.Abs(incumbent.key)*groupAutoSortScoreImprovement)
	if candidate.key+threshold < incumbent.key {
		return true
	}
	// Cost is a tie-breaker only while the two health scores are within the
	// deadband. It can never override a material stability difference.
	if math.Abs(candidate.key-incumbent.key) <= threshold && candidate.hasRate && incumbent.hasRate && candidate.rate < incumbent.rate {
		return true
	}
	return schedulerRoleRankForAutoSort(candidate.entry.Role) < schedulerRoleRankForAutoSort(incumbent.entry.Role)
}

func groupAutoSortStrictlyBefore(a, b groupAutoSortRanked) bool {
	aRole := schedulerRoleRankForAutoSort(a.entry.Role)
	bRole := schedulerRoleRankForAutoSort(b.entry.Role)
	if a.tier != b.tier {
		return a.tier < b.tier
	}
	if a.hasKey != b.hasKey {
		return a.hasKey
	}
	if a.hasKey && a.key != b.key {
		return a.key < b.key
	}
	if aRole != bRole {
		return aRole < bRole
	}
	if a.hasRate != b.hasRate {
		return a.hasRate
	}
	if a.hasRate && a.rate != b.rate {
		return a.rate < b.rate
	}
	return a.currentOrder < b.currentOrder
}

func schedulerRoleRankForAutoSort(role string) int {
	if role == AccountGroupRoleBackup {
		return 1
	}
	return 0
}

func groupAutoSortHealthTier(acc *Account, monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) int {
	if acc == nil || acc.SchedulingBlockReasonAt(time.Now()) != AccountSchedulingBlockNone {
		return 4
	}
	monitorKnown := groupAutoSortMonitorKnown(monitor, time.Now()) && (monitor.LatestStatus == MonitorStatusOperational ||
		monitor.LatestStatus == MonitorStatusDegraded ||
		monitor.LatestStatus == MonitorStatusFailed ||
		monitor.LatestStatus == MonitorStatusError)
	if monitorKnown {
		if monitor.LatestStatus == MonitorStatusFailed || monitor.LatestStatus == MonitorStatusError || monitor.Availability1h < 80 {
			return 3
		}
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
	if monitorKnown && (monitor.LatestStatus == MonitorStatusDegraded || monitor.Availability1h < 95) {
		return 2
	}
	// Tier 0 is proven only after enough real traffic. Probe-only or low-sample
	// accounts stay in the fresh/probation tier until user traffic validates
	// their model success rate.
	if experience == nil || experience.attempts() < groupAutoSortExperienceMinAttempts {
		return 1
	}
	return 0
}

// groupAutoSortRateHealthTier keeps cost-oriented groups cheap without letting
// an inexpensive but failing account become their primary. A proven account
// must have enough real attempts, at least 90% upstream-attempt success, and a
// bounded P95 first-token latency. New accounts remain usable as probationary
// overflow so they can collect samples without displacing a proven primary.
func groupAutoSortRateHealthTier(acc *Account, monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) int {
	if acc == nil || acc.SchedulingBlockReasonAt(time.Now()) != AccountSchedulingBlockNone {
		return 4
	}
	monitorKnown := groupAutoSortMonitorKnown(monitor, time.Now()) && (monitor.LatestStatus == MonitorStatusOperational ||
		monitor.LatestStatus == MonitorStatusDegraded ||
		monitor.LatestStatus == MonitorStatusFailed ||
		monitor.LatestStatus == MonitorStatusError)
	if monitorKnown && (monitor.LatestStatus == MonitorStatusFailed ||
		monitor.LatestStatus == MonitorStatusError || monitor.Availability1h < 80) {
		return 3
	}

	if experience != nil && experience.attempts() >= groupAutoSortExperienceMinAttempts {
		failureRate := experience.failureRate()
		if failureRate >= groupAutoSortRateSevereFailureRate {
			return 3
		}
		if failureRate > groupAutoSortRateMaxFailureRate ||
			(experience.FirstTokenSamples >= groupAutoSortRateMinFirstTokenSamples && experience.P95FirstTokenMs > groupAutoSortRateMaxP95FirstTokenMs) {
			return 2
		}
		if monitorKnown && (monitor.LatestStatus == MonitorStatusDegraded || monitor.Availability1h < 95) {
			return 1
		}
		return 0
	}

	if monitorKnown {
		if monitor.LatestStatus == MonitorStatusOperational && monitor.Availability1h >= 95 {
			return 1
		}
		return 2
	}
	return 2
}

func groupAutoSortExperienceScore(monitor *AccountMonitorStatus, experience *groupAutoSortExperienceStats) float64 {
	score := 0.0
	if experience == nil || experience.attempts() == 0 {
		score += 50
	} else {
		score += 700 * experience.failureRate()
		score += 300 * experience.failoverRate()
		score += 500 * experience.worstModelFailureRate(groupAutoSortExperienceMinModelAttempts)
		if experience.attempts() < groupAutoSortExperienceMinAttempts {
			score += 25
		}
		if experience.FirstTokenSamples > 0 {
			score += math.Min(experience.P95FirstTokenMs, 120000) / 1000
		} else {
			score += 30
		}
		if experience.DurationSamples > 0 {
			score += math.Min(experience.P95DurationMs, 600000) / 10000
		}
		if experience.SuccessCount >= 4 {
			score += 5 * (1 - experience.cacheHitRate())
		} else {
			score += 2.5
		}
	}
	monitorKnown := groupAutoSortMonitorKnown(monitor, time.Now()) && (monitor.LatestStatus == MonitorStatusOperational || monitor.LatestStatus == MonitorStatusDegraded)
	if monitorKnown {
		score += math.Max(0, 100-monitor.Availability1h) * 0.5
		if (experience == nil || experience.FirstTokenSamples == 0) && monitor.AvgLatency1h != nil {
			score += math.Min(*monitor.AvgLatency1h, 120000) / 1000
		}
	} else {
		score += 20
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
