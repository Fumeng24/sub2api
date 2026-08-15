package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

type groupAutoSortAdminStub struct {
	groups       []Group
	entries      map[int64][]AccountSchedulingEntry
	updated      []int64
	priority     map[int64]int
	groupUpdates map[int64][]AccountSchedulingConfig
}

func (s *groupAutoSortAdminStub) GetAllGroups(ctx context.Context) ([]Group, error) {
	return s.groups, nil
}

func (s *groupAutoSortAdminStub) GetGroupAccountScheduling(ctx context.Context, groupID int64) ([]AccountSchedulingEntry, error) {
	return s.entries[groupID], nil
}

func (s *groupAutoSortAdminStub) UpdateGroupAccountScheduling(ctx context.Context, groupID int64, configs []AccountSchedulingConfig) error {
	if s.groupUpdates == nil {
		s.groupUpdates = make(map[int64][]AccountSchedulingConfig)
	}
	s.groupUpdates[groupID] = append([]AccountSchedulingConfig(nil), configs...)
	for _, config := range configs {
		s.updated = append(s.updated, config.AccountID)
		if s.priority == nil {
			s.priority = make(map[int64]int)
		}
		s.priority[config.AccountID] = config.SortOrder
	}
	return nil
}

type groupAutoSortAvailabilityStub struct {
	status map[int64]*AccountMonitorStatus
}

type groupAutoSortExperienceStub struct {
	byGroup map[int64]map[int64]*groupAutoSortExperienceStats
}

type groupAutoSortRateStub struct {
	rates map[int64]float64
}

func (s *groupAutoSortRateStub) RatesByAccountID(_ context.Context, _ []int64, _ time.Time) (map[int64]float64, error) {
	return s.rates, nil
}

func (s *groupAutoSortExperienceStub) StatsByAccountID(_ context.Context, groupID int64, _ []int64, _ time.Time) (map[int64]*groupAutoSortExperienceStats, error) {
	return s.byGroup[groupID], nil
}

func (s *groupAutoSortAvailabilityStub) StatusByAccountID(ctx context.Context) (map[int64]*AccountMonitorStatus, error) {
	return s.status, nil
}

func TestGroupAutoSortLatency_PutsFailedAccountsAfterHealthyOnes(t *testing.T) {
	admin := &groupAutoSortAdminStub{
		groups: []Group{
			{
				ID: 1,
				AutoSortConfig: GroupAutoSortConfig{
					Enabled: true,
					Basis:   domain.GroupAutoSortBasisLatency,
				},
			},
		},
		entries: map[int64][]AccountSchedulingEntry{
			1: {
				{Account: groupAutoSortAccount(101, "healthy-slower")},
				{Account: groupAutoSortAccount(102, "failed-fast")},
				{Account: groupAutoSortAccount(103, "healthy-faster")},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{
		status: map[int64]*AccountMonitorStatus{
			101: {
				AccountID:      101,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 100,
				AvgLatency1h:   groupAutoSortFloat64Ptr(300),
			},
			102: {
				AccountID:      102,
				LatestStatus:   MonitorStatusFailed,
				Availability1h: 0,
				AvgLatency1h:   groupAutoSortFloat64Ptr(50),
			},
			103: {
				AccountID:      103,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 100,
				AvgLatency1h:   groupAutoSortFloat64Ptr(120),
			},
		},
	}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.runOnce()

	// A small latency improvement within the same health tier is held by the
	// deadband; the failed account is still moved out of the serving path.
	require.Equal(t, []int64{101, 103, 102}, admin.updated)
	require.Equal(t, 1, admin.priority[101])
	require.Equal(t, 2, admin.priority[103])
	require.Equal(t, 3, admin.priority[102])
}

func TestGroupAutoSortLatency_PrioritizesAvailabilityBeforeLatency(t *testing.T) {
	admin := &groupAutoSortAdminStub{
		groups: []Group{
			{
				ID: 1,
				AutoSortConfig: GroupAutoSortConfig{
					Enabled: true,
					Basis:   domain.GroupAutoSortBasisLatency,
				},
			},
		},
		entries: map[int64][]AccountSchedulingEntry{
			1: {
				{Account: groupAutoSortAccount(201, "healthy-slow")},
				{Account: groupAutoSortAccount(202, "flaky-fast")},
				{Account: groupAutoSortAccount(203, "healthy-fast")},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{
		status: map[int64]*AccountMonitorStatus{
			201: {
				AccountID:      201,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 100,
				AvgLatency1h:   groupAutoSortFloat64Ptr(3000),
			},
			202: {
				AccountID:      202,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 45,
				AvgLatency1h:   groupAutoSortFloat64Ptr(100),
			},
			203: {
				AccountID:      203,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 100,
				AvgLatency1h:   groupAutoSortFloat64Ptr(250),
			},
		},
	}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.runOnce()

	// Availability/latency is secondary to the same-tier residence deadband;
	// the degraded account is nevertheless kept behind both healthy accounts.
	require.Equal(t, []int64{201, 203, 202}, admin.updated)
	require.Equal(t, 1, admin.priority[201])
	require.Equal(t, 2, admin.priority[203])
	require.Equal(t, 3, admin.priority[202])
}

func TestGroupAutoSortAvailability_PutsFailedAndUnschedulableAfterHealthyOnes(t *testing.T) {
	blocked := groupAutoSortAccount(302, "blocked-oauth")
	blocked.Type = AccountTypeOAuth
	blocked.Schedulable = false
	admin := &groupAutoSortAdminStub{
		groups: []Group{
			{
				ID: 1,
				AutoSortConfig: GroupAutoSortConfig{
					Enabled: true,
					Basis:   domain.GroupAutoSortBasisAvailability,
				},
			},
		},
		entries: map[int64][]AccountSchedulingEntry{
			1: {
				{Account: blocked},
				{Account: groupAutoSortAccount(301, "healthy-lower-availability")},
				{Account: groupAutoSortAccount(303, "failed-high-availability")},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{
		status: map[int64]*AccountMonitorStatus{
			301: {
				AccountID:      301,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 90,
			},
			302: {
				AccountID:      302,
				LatestStatus:   MonitorStatusOperational,
				Availability1h: 100,
			},
			303: {
				AccountID:      303,
				LatestStatus:   MonitorStatusFailed,
				Availability1h: 100,
			},
		},
	}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.runOnce()

	require.Equal(t, []int64{301, 303, 302}, admin.updated)
	require.Equal(t, 1, admin.priority[301])
	require.Equal(t, 2, admin.priority[303])
	require.Equal(t, 3, admin.priority[302])
}

func TestFinalRateForAccount_PrefersLocalOverridesBeforeUpstreamCached(t *testing.T) {
	t.Run("manual rate wins", func(t *testing.T) {
		acc := &Account{
			RateMultiplier: groupAutoSortFloat64Ptr(1),
			Extra: map[string]any{
				"manual_rate":          0.15,
				"upstream_rate_cached": 1.0,
				"rate_scale":           1.0,
			},
		}
		got, ok := finalRateForAccount(acc)
		require.True(t, ok)
		require.InDelta(t, 0.15, got, 1e-12)
	})

	t.Run("unknown default rate does not masquerade as verified", func(t *testing.T) {
		acc := &Account{
			RateMultiplier: groupAutoSortFloat64Ptr(1),
			Extra: map[string]any{
				"upstream_rate_cached": 0.15,
				"rate_scale":           1.0,
			},
		}
		_, ok := finalRateForAccount(acc)
		require.False(t, ok)
	})

	t.Run("billing rate is fallback", func(t *testing.T) {
		acc := &Account{RateMultiplier: groupAutoSortFloat64Ptr(0.2)}
		got, ok := finalRateForAccount(acc)
		require.True(t, ok)
		require.InDelta(t, 0.2, got, 1e-12)
	})
}

func TestGroupAutoSortExperience_PenalizesFastButUnstableAccount(t *testing.T) {
	admin := &groupAutoSortAdminStub{
		groups: []Group{{
			ID:             7,
			AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisExperience},
		}},
		entries: map[int64][]AccountSchedulingEntry{
			7: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 702, Priority: 1, Weight: 100, SortOrder: 1}, Account: groupAutoSortAccount(702, "fast-but-flaky")},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 701, Priority: 2, Weight: 100, SortOrder: 2}, Account: groupAutoSortAccount(701, "stable")},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		701: {AccountID: 701, LatestStatus: MonitorStatusOperational, Availability1h: 100, AvgLatency1h: groupAutoSortFloat64Ptr(8000)},
		702: {AccountID: 702, LatestStatus: MonitorStatusOperational, Availability1h: 100, AvgLatency1h: groupAutoSortFloat64Ptr(900)},
	}}
	experience := &groupAutoSortExperienceStub{byGroup: map[int64]map[int64]*groupAutoSortExperienceStats{
		7: {
			701: {
				SuccessCount: 100, FirstTokenSamples: 100, DurationSamples: 100,
				P95FirstTokenMs: 8000, P95DurationMs: 20000, CacheReadTokens: 950, CacheEligibleTokens: 1000,
			},
			702: {
				SuccessCount: 40, FailureCount: 20, FailoverCount: 20,
				FirstTokenSamples: 40, DurationSamples: 40, P95FirstTokenMs: 900,
				P95DurationMs: 4000, CacheReadTokens: 390, CacheEligibleTokens: 400,
			},
		},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.SetExperienceProvider(experience)
	svc.runOnce()

	require.Equal(t, []int64{701, 702}, admin.updated)
	require.Equal(t, 1, admin.priority[701])
	require.Equal(t, 2, admin.priority[702])
}

func TestGroupAutoSortExperience_DeadbandPreservesIncumbent(t *testing.T) {
	items := []groupAutoSortRanked{
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 1, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          20,
			hasKey:       true,
			currentOrder: 0,
		},
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 2, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          18,
			hasKey:       true,
			currentOrder: 1,
		},
	}

	groupAutoSortExperienceWithHysteresis(items)

	require.Equal(t, int64(1), items[0].entry.AccountID)
	require.Equal(t, int64(2), items[1].entry.AccountID)
}

func TestGroupAutoSortExperience_SignificantImprovementPromotesChallenger(t *testing.T) {
	items := []groupAutoSortRanked{
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 1, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          20,
			hasKey:       true,
			currentOrder: 0,
		},
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 2, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          16,
			hasKey:       true,
			currentOrder: 1,
		},
	}

	groupAutoSortExperienceWithHysteresis(items)

	require.Equal(t, int64(2), items[0].entry.AccountID)
	require.Equal(t, int64(1), items[1].entry.AccountID)
}

func TestGroupAutoSortExperience_BestChallengerIsNotBlockedByDeadbandNeighbor(t *testing.T) {
	items := []groupAutoSortRanked{
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 1, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          20,
			hasKey:       true,
			currentOrder: 0,
		},
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 2, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          18,
			hasKey:       true,
			currentOrder: 1,
		},
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 3, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          16,
			hasKey:       true,
			currentOrder: 2,
		},
	}

	groupAutoSortExperienceWithHysteresis(items)

	require.Equal(t, int64(3), items[0].entry.AccountID)
	require.Equal(t, int64(1), items[1].entry.AccountID)
	require.Equal(t, int64(2), items[2].entry.AccountID)
}

func TestGroupAutoSortExperience_IgnoresPrimaryBackupRole(t *testing.T) {
	items := []groupAutoSortRanked{
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 1, Role: AccountGroupRolePrimary}},
			tier:         2,
			key:          80,
			hasKey:       true,
			currentOrder: 0,
		},
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 2, Role: AccountGroupRoleBackup}},
			tier:         0,
			key:          1,
			hasKey:       true,
			currentOrder: 1,
		},
	}

	groupAutoSortExperienceWithHysteresis(items)

	// A materially healthier account outranks a degraded account regardless of
	// the legacy primary/backup label.
	require.Equal(t, int64(2), items[0].entry.AccountID)
	require.Equal(t, int64(1), items[1].entry.AccountID)
}

func TestGroupAutoSortExperience_DoesNotUseRoleForEqualHealth(t *testing.T) {
	items := []groupAutoSortRanked{
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 1, Role: AccountGroupRoleBackup}},
			tier:         0,
			key:          10,
			hasKey:       true,
			currentOrder: 0,
		},
		{
			entry:        AccountSchedulingEntry{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 2, Role: AccountGroupRolePrimary}},
			tier:         0,
			key:          10,
			hasKey:       true,
			currentOrder: 1,
		},
	}

	groupAutoSortWithHysteresis(items)

	require.Equal(t, int64(1), items[0].entry.AccountID)
	require.Equal(t, int64(2), items[1].entry.AccountID)
}

func TestGroupAutoSortStrategiesRemainIndependentPerGroup(t *testing.T) {
	sharedA := groupAutoSortAccount(801, "shared-a")
	sharedA.RateMultiplier = groupAutoSortFloat64Ptr(0.5)
	sharedB := groupAutoSortAccount(802, "shared-b")
	sharedB.RateMultiplier = groupAutoSortFloat64Ptr(0.1)
	admin := &groupAutoSortAdminStub{
		groups: []Group{
			{ID: 81, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisExperience}},
			{ID: 82, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisRate}},
		},
		entries: map[int64][]AccountSchedulingEntry{
			81: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 802, Priority: 1, Weight: 100, SortOrder: 1}, Account: sharedB},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 801, Priority: 2, Weight: 100, SortOrder: 2}, Account: sharedA},
			},
			82: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 801, Priority: 1, Weight: 100, SortOrder: 1}, Account: sharedA},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 802, Priority: 2, Weight: 100, SortOrder: 2}, Account: sharedB},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		801: {AccountID: 801, LatestStatus: MonitorStatusOperational, Availability1h: 100, AvgLatency1h: groupAutoSortFloat64Ptr(1000)},
		802: {AccountID: 802, LatestStatus: MonitorStatusOperational, Availability1h: 100, AvgLatency1h: groupAutoSortFloat64Ptr(1000)},
	}}
	experience := &groupAutoSortExperienceStub{byGroup: map[int64]map[int64]*groupAutoSortExperienceStats{
		81: {
			801: {SuccessCount: 100, FirstTokenSamples: 100, P95FirstTokenMs: 1000, CacheReadTokens: 900, CacheEligibleTokens: 1000},
			802: {SuccessCount: 20, FailureCount: 20, FailoverCount: 20, FirstTokenSamples: 20, P95FirstTokenMs: 500, CacheReadTokens: 200, CacheEligibleTokens: 200},
		},
		82: {
			801: {SuccessCount: 100, FirstTokenSamples: 100, P95FirstTokenMs: 1000, CacheReadTokens: 900, CacheEligibleTokens: 1000},
			802: {SuccessCount: 100, FirstTokenSamples: 100, P95FirstTokenMs: 1000, CacheReadTokens: 900, CacheEligibleTokens: 1000},
		},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.SetExperienceProvider(experience)
	svc.runOnce()

	require.Equal(t, int64(801), admin.groupUpdates[81][0].AccountID, "Pro-style experience order must prefer the stable account")
	require.Equal(t, int64(802), admin.groupUpdates[82][0].AccountID, "rate order must independently prefer the cheaper account")
}

func TestAccountSchedulingPriorityForGroup(t *testing.T) {
	account := &Account{
		Priority: 50,
		AccountGroups: []AccountGroup{
			{GroupID: 1, Priority: 9, SortOrder: 2},
			{GroupID: 2, Priority: 3, SortOrder: 7},
		},
	}

	require.Equal(t, 2, account.SchedulingPriorityForGroup(1))
	require.Equal(t, 7, account.SchedulingPriorityForGroup(2))
	require.Equal(t, 50, account.SchedulingPriorityForGroup(3))
}

func TestFilterSchedulableAccountsForGroupProjectsCurrentGroupOrder(t *testing.T) {
	groupID := int64(2)
	accounts := []Account{{
		ID:          801,
		Priority:    99,
		Status:      StatusActive,
		Schedulable: true,
		AccountGroups: []AccountGroup{
			{GroupID: 1, SortOrder: 8, SchedulingConfigured: true},
			{GroupID: 2, SortOrder: 3, SchedulingConfigured: true},
		},
	}}

	filtered := filterSchedulableAccountsForGroup(accounts, &groupID)

	require.Len(t, filtered, 1)
	require.Equal(t, 3, filtered[0].Priority)
	require.Equal(t, 99, accounts[0].Priority, "projection must not mutate the shared cached account")
}

func TestGroupAutoSortHealthTier_DetectsModelSpecificFailure(t *testing.T) {
	account := groupAutoSortAccount(901, "model-flaky")
	monitor := &AccountMonitorStatus{
		AccountID:      account.ID,
		LatestStatus:   MonitorStatusOperational,
		Availability1h: 100,
	}
	stats := &groupAutoSortExperienceStats{
		SuccessCount: 100,
		FailureCount: 2,
		Models: map[string]*groupAutoSortModelExperienceStats{
			"gpt-5.6-terra": {SuccessCount: 5, FailureCount: 5, FailoverCount: 5},
		},
	}

	require.Equal(t, 3, groupAutoSortHealthTier(account, monitor, stats))
}

func TestGroupAutoSortMonitorKnownRejectsDisabledAndStaleEvidence(t *testing.T) {
	now := time.Now()

	stale := &AccountMonitorStatus{
		MonitorID:       1,
		Enabled:         true,
		IntervalSeconds: 60,
		LastCheckedAt:   groupAutoSortTimePtr(now.Add(-3 * time.Minute)),
		LatestStatus:    MonitorStatusOperational,
	}
	require.False(t, groupAutoSortMonitorKnown(stale, now))

	disabled := &AccountMonitorStatus{
		MonitorID:       1,
		Enabled:         false,
		IntervalSeconds: 60,
		LastCheckedAt:   groupAutoSortTimePtr(now.Add(-time.Second)),
		LatestStatus:    MonitorStatusOperational,
	}
	require.False(t, groupAutoSortMonitorKnown(disabled, now))

	fresh := *stale
	fresh.Enabled = true
	fresh.LastCheckedAt = groupAutoSortTimePtr(now.Add(-time.Minute))
	require.True(t, groupAutoSortMonitorKnown(&fresh, now))
}

func TestGroupAutoSortProbeWindowStatsUsesFresh15MinuteTimeline(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	status := &AccountMonitorStatus{
		MonitorID:       1,
		AccountID:       1001,
		Enabled:         true,
		IntervalSeconds: 60,
		LatestStatus:    MonitorStatusOperational,
		LastCheckedAt:   groupAutoSortTimePtr(now),
		Timeline: []*AccountMonitorCheck{
			{Status: MonitorStatusOperational, LatencyMs: groupAutoSortIntPtr(100), CheckedAt: now.Add(-time.Minute)},
			{Status: MonitorStatusDegraded, LatencyMs: groupAutoSortIntPtr(200), CheckedAt: now.Add(-2 * time.Minute)},
			{Status: MonitorStatusFailed, LatencyMs: groupAutoSortIntPtr(300), CheckedAt: now.Add(-3 * time.Minute)},
			// Older probe evidence must not dilute the current 15-minute result.
			{Status: MonitorStatusOperational, LatencyMs: groupAutoSortIntPtr(1), CheckedAt: now.Add(-16 * time.Minute)},
		},
	}

	stats := groupAutoSortProbeWindowStats(status, now)
	require.True(t, stats.known)
	require.Equal(t, 3, stats.samples)
	require.Equal(t, 1, stats.failures)
	require.Equal(t, 1, stats.degraded)
	require.InDelta(t, 66.666, stats.availability, 0.01)
	require.InDelta(t, 300, stats.p95LatencyMs, 0.01)
}

func TestGroupAutoSortHealthTierLatestProbeFailureOverridesGoodTraffic(t *testing.T) {
	now := time.Now()
	account := groupAutoSortAccount(1002, "probe-failed")
	monitor := &AccountMonitorStatus{
		MonitorID:       2,
		AccountID:       account.ID,
		Enabled:         true,
		IntervalSeconds: 60,
		LatestStatus:    MonitorStatusError,
		LastCheckedAt:   groupAutoSortTimePtr(now),
		Timeline:        []*AccountMonitorCheck{{Status: MonitorStatusError, CheckedAt: now}},
	}
	traffic := &groupAutoSortExperienceStats{SuccessCount: 200, FirstTokenSamples: 200, P95FirstTokenMs: 500}

	require.Equal(t, 3, groupAutoSortHealthTier(account, monitor, traffic))
}

func TestGroupAutoSortLatestProbeFailureBypassesMinimumResidence(t *testing.T) {
	now := time.Now()
	healthy := groupAutoSortAccount(1003, "healthy")
	failed := groupAutoSortAccount(1004, "probe-failed")
	admin := &groupAutoSortAdminStub{
		groups: []Group{{ID: 100, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisExperience}}},
		entries: map[int64][]AccountSchedulingEntry{
			100: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: failed.ID, Priority: 1, SortOrder: 1}, Account: failed},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: healthy.ID, Priority: 2, SortOrder: 2}, Account: healthy},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		healthy.ID: {
			AccountID: healthy.ID, Enabled: true, IntervalSeconds: 60,
			LatestStatus: MonitorStatusOperational, LastCheckedAt: groupAutoSortTimePtr(now),
			Timeline: []*AccountMonitorCheck{{Status: MonitorStatusOperational, CheckedAt: now}},
		},
		failed.ID: {
			AccountID: failed.ID, Enabled: true, IntervalSeconds: 60,
			LatestStatus: MonitorStatusError, LastCheckedAt: groupAutoSortTimePtr(now),
			Timeline: []*AccountMonitorCheck{{Status: MonitorStatusError, CheckedAt: now}},
		},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.lastReorderAt[100] = now.Add(-time.Minute)
	svc.runOnce()

	require.Equal(t, []int64{healthy.ID, failed.ID}, admin.updated)
}

func groupAutoSortIntPtr(v int) *int {
	return &v
}

func TestFinalRateForAccount_UsesOnlyFreshProbeAndAppliesScale(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"rate_scale": 0.5,
			UpstreamBillingProbeExtraKey: &UpstreamBillingProbeSnapshot{
				Status:     UpstreamBillingProbeStatusOK,
				ReceivedAt: groupAutoSortTimePtr(now.Add(-time.Minute)),
				FreshUntil: groupAutoSortTimePtr(now.Add(time.Minute)),
				Data: map[string]any{
					"billing_scope":            "token",
					"resolved_rate_multiplier": 0.4,
					"peak_rate_enabled":        false,
				},
			},
		},
	}

	rate, ok := finalRateForAccountAt(account, now)
	require.True(t, ok)
	require.InDelta(t, 0.2, rate, 1e-12)

	_, ok = finalRateForAccountAt(account, now.Add(2*time.Minute))
	require.False(t, ok, "expired probe data must be treated as unknown")
}

func TestGroupAutoSortDoesNotWriteWhenGroupOrderIsUnchanged(t *testing.T) {
	first := groupAutoSortAccount(911, "cheap")
	first.RateMultiplier = groupAutoSortFloat64Ptr(0.1)
	second := groupAutoSortAccount(912, "expensive")
	second.RateMultiplier = groupAutoSortFloat64Ptr(0.3)
	admin := &groupAutoSortAdminStub{
		groups: []Group{{ID: 91, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisRate}}},
		entries: map[int64][]AccountSchedulingEntry{
			91: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 911, Priority: 1, Weight: 100, SortOrder: 1}, Account: first},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 912, Priority: 2, Weight: 100, SortOrder: 2}, Account: second},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		911: {AccountID: 911, LatestStatus: MonitorStatusOperational, Availability1h: 100},
		912: {AccountID: 912, LatestStatus: MonitorStatusOperational, Availability1h: 100},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.runOnce()

	require.Empty(t, admin.groupUpdates)
}

func TestGroupAutoSortRate_UsesUpstreamManagementRate(t *testing.T) {
	upstreamID := int64(5)
	first := groupAutoSortAccount(921, "locally-cheap-upstream-expensive")
	first.UpstreamID = &upstreamID
	first.RateMultiplier = groupAutoSortFloat64Ptr(0.01)
	second := groupAutoSortAccount(922, "locally-expensive-upstream-cheap")
	second.UpstreamID = &upstreamID
	second.RateMultiplier = groupAutoSortFloat64Ptr(0.9)
	admin := &groupAutoSortAdminStub{
		groups: []Group{{ID: 92, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisRate}}},
		entries: map[int64][]AccountSchedulingEntry{
			92: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 921, Priority: 1, Weight: 100, SortOrder: 1}, Account: first},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 922, Priority: 2, Weight: 100, SortOrder: 2}, Account: second},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		921: {AccountID: 921, LatestStatus: MonitorStatusOperational, Availability1h: 100},
		922: {AccountID: 922, LatestStatus: MonitorStatusOperational, Availability1h: 100},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.SetRateProvider(&groupAutoSortRateStub{rates: map[int64]float64{921: 0.2, 922: 0.05}})
	svc.runOnce()

	require.Equal(t, int64(922), admin.groupUpdates[92][0].AccountID)
}

func TestGroupAutoSortRate_HealthGatePrecedesRate(t *testing.T) {
	cheap := groupAutoSortAccount(931, "cheap-but-degraded")
	cheap.RateMultiplier = groupAutoSortFloat64Ptr(0.05)
	expensive := groupAutoSortAccount(932, "expensive-and-healthy")
	expensive.RateMultiplier = groupAutoSortFloat64Ptr(0.9)
	admin := &groupAutoSortAdminStub{
		groups: []Group{{ID: 93, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisRate}}},
		entries: map[int64][]AccountSchedulingEntry{
			93: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 931, Priority: 1, Weight: 100, SortOrder: 1}, Account: cheap},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 932, Priority: 2, Weight: 100, SortOrder: 2}, Account: expensive},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		931: {AccountID: 931, LatestStatus: MonitorStatusFailed, Availability1h: 0},
		932: {AccountID: 932, LatestStatus: MonitorStatusOperational, Availability1h: 100},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.SetExperienceProvider(&groupAutoSortExperienceStub{byGroup: map[int64]map[int64]*groupAutoSortExperienceStats{
		93: {
			931: {SuccessCount: 1, FailureCount: 20},
			932: {SuccessCount: 100},
		},
	}})
	svc.runOnce()

	require.Equal(t, int64(932), admin.groupUpdates[93][0].AccountID)
}

func TestGroupAutoSortRate_UsesLowestRateWithinQualifiedTier(t *testing.T) {
	cheap := groupAutoSortAccount(941, "cheap-qualified")
	cheap.RateMultiplier = groupAutoSortFloat64Ptr(0.09)
	expensive := groupAutoSortAccount(942, "expensive-qualified")
	expensive.RateMultiplier = groupAutoSortFloat64Ptr(0.15)
	admin := &groupAutoSortAdminStub{
		groups: []Group{{ID: 94, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisRate}}},
		entries: map[int64][]AccountSchedulingEntry{
			94: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 942, Priority: 1, Weight: 100, SortOrder: 1}, Account: expensive},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 941, Priority: 2, Weight: 100, SortOrder: 2}, Account: cheap},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		941: {AccountID: 941, LatestStatus: MonitorStatusOperational, Availability1h: 100},
		942: {AccountID: 942, LatestStatus: MonitorStatusOperational, Availability1h: 100},
	}}
	experience := &groupAutoSortExperienceStub{byGroup: map[int64]map[int64]*groupAutoSortExperienceStats{
		94: {
			941: {SuccessCount: 95, FailureCount: 5, FirstTokenSamples: 95, P95FirstTokenMs: 12000},
			942: {SuccessCount: 100, FirstTokenSamples: 100, P95FirstTokenMs: 9000},
		},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.SetExperienceProvider(experience)
	svc.runOnce()

	// Rate is only a same-health tie-breaker. The more stable account remains
	// first even when the other account is cheaper.
	require.Empty(t, admin.groupUpdates)
}

func TestGroupAutoSortRate_ProbationaryCheapAccountDoesNotDisplaceQualifiedPrimary(t *testing.T) {
	probationary := groupAutoSortAccount(951, "cheap-probationary")
	probationary.RateMultiplier = groupAutoSortFloat64Ptr(0.05)
	qualified := groupAutoSortAccount(952, "qualified")
	qualified.RateMultiplier = groupAutoSortFloat64Ptr(0.15)
	admin := &groupAutoSortAdminStub{
		groups: []Group{{ID: 95, AutoSortConfig: GroupAutoSortConfig{Enabled: true, Basis: domain.GroupAutoSortBasisRate}}},
		entries: map[int64][]AccountSchedulingEntry{
			95: {
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 951, Priority: 1, Weight: 100, SortOrder: 1}, Account: probationary},
				{AccountSchedulingConfig: AccountSchedulingConfig{AccountID: 952, Priority: 2, Weight: 100, SortOrder: 2}, Account: qualified},
			},
		},
	}
	availability := &groupAutoSortAvailabilityStub{status: map[int64]*AccountMonitorStatus{
		951: {AccountID: 951, LatestStatus: MonitorStatusOperational, Availability1h: 100},
		952: {AccountID: 952, LatestStatus: MonitorStatusOperational, Availability1h: 100},
	}}
	experience := &groupAutoSortExperienceStub{byGroup: map[int64]map[int64]*groupAutoSortExperienceStats{
		95: {
			951: {SuccessCount: 5},
			952: {SuccessCount: 100, FirstTokenSamples: 100, P95FirstTokenMs: 10000},
		},
	}}

	svc := NewGroupAutoSortService(admin, availability, 0)
	svc.SetExperienceProvider(experience)
	svc.runOnce()

	require.Equal(t, int64(952), admin.groupUpdates[95][0].AccountID)
}

func TestFinalRateForBoundAccountDoesNotReuseLocalSyncedRateWhenProbeIsMissing(t *testing.T) {
	upstreamID := int64(6)
	account := &Account{
		UpstreamID:     &upstreamID,
		RateMultiplier: groupAutoSortFloat64Ptr(0.02),
	}

	_, ok := finalRateForAccountWithUpstreamRateAt(account, 0, false, time.Now())
	require.False(t, ok)
}

func groupAutoSortAccount(id int64, name string) *Account {
	return &Account{
		ID:          id,
		Name:        name,
		Status:      StatusActive,
		Schedulable: true,
	}
}

func groupAutoSortFloat64Ptr(v float64) *float64 {
	return &v
}

func groupAutoSortTimePtr(v time.Time) *time.Time {
	return &v
}
