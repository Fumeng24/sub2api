//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// --- stubs ---

type accountMonitorRepoStub struct {
	byID      map[int64]*AccountMonitor
	byAccount map[int64]*AccountMonitor
	created   *AccountMonitor
	checks    []*AccountMonitorCheck
	recent    map[int64][]*AccountMonitorCheck
}

func (s *accountMonitorRepoStub) Create(_ context.Context, m *AccountMonitor) error {
	m.ID = 1
	s.created = m
	return nil
}
func (s *accountMonitorRepoStub) Update(_ context.Context, _ *AccountMonitor) error { return nil }
func (s *accountMonitorRepoStub) Delete(_ context.Context, _ int64) error           { return nil }
func (s *accountMonitorRepoStub) GetByID(_ context.Context, id int64) (*AccountMonitor, error) {
	if m, ok := s.byID[id]; ok {
		return m, nil
	}
	return nil, ErrAccountMonitorNotFound
}
func (s *accountMonitorRepoStub) GetByAccountID(_ context.Context, accountID int64) (*AccountMonitor, error) {
	if m, ok := s.byAccount[accountID]; ok {
		return m, nil
	}
	return nil, ErrAccountMonitorNotFound
}
func (s *accountMonitorRepoStub) List(_ context.Context) ([]*AccountMonitor, error) { return nil, nil }
func (s *accountMonitorRepoStub) ListEnabled(_ context.Context) ([]*AccountMonitor, error) {
	return nil, nil
}
func (s *accountMonitorRepoStub) UpdateLastCheckedAt(_ context.Context, _ int64, _ time.Time) error {
	return nil
}
func (s *accountMonitorRepoStub) InsertChecks(_ context.Context, checks []*AccountMonitorCheck) error {
	s.checks = append(s.checks, checks...)
	return nil
}
func (s *accountMonitorRepoStub) LatestChecks(_ context.Context, _ []int64) (map[int64]*AccountMonitorCheck, error) {
	return nil, nil
}
func (s *accountMonitorRepoStub) Availability1h(_ context.Context, _ []int64) (map[int64]float64, error) {
	return nil, nil
}
func (s *accountMonitorRepoStub) AvgLatency1h(_ context.Context, _ []int64) (map[int64]float64, error) {
	return nil, nil
}
func (s *accountMonitorRepoStub) RecentChecks(_ context.Context, _ []int64, _ int) (map[int64][]*AccountMonitorCheck, error) {
	if s.recent != nil {
		return s.recent, nil
	}
	return nil, nil
}
func (s *accountMonitorRepoStub) DeleteChecksOlderThan(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

type accountReaderStub struct {
	byID map[int64]*Account
	err  error
}

func (s *accountReaderStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.err != nil {
		return nil, s.err
	}
	if a, ok := s.byID[id]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func apiKeyAccount() *Account {
	return &Account{
		ID:          7,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Platform:    "openai",
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.example.com"},
	}
}

type accountMonitorRecoveryStub struct {
	calls      []int64
	modelCalls []string
	err        error
}

func (s *accountMonitorRecoveryStub) RecoverAccountAfterSuccessfulTest(_ context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error) {
	s.calls = append(s.calls, accountID)
	if s.err != nil {
		return nil, s.err
	}
	return &SuccessfulTestRecoveryResult{ClearedError: true}, nil
}

func (s *accountMonitorRecoveryStub) RecoverAccountModelAfterSuccessfulTest(_ context.Context, accountID int64, model string) (*SuccessfulTestRecoveryResult, error) {
	s.modelCalls = append(s.modelCalls, fmt.Sprintf("%d:%s", accountID, model))
	if s.err != nil {
		return nil, s.err
	}
	return &SuccessfulTestRecoveryResult{ClearedRateLimit: true}, nil
}

type accountMonitorFailureBlockerStub struct {
	schedulableByGroup map[int64][]Account
	blockedAccountID   int64
	blockedUntil       time.Time
	blockedReason      string
	modelRateLimitID   int64
	modelRateLimitKey  string
	modelRateLimitAt   time.Time
	modelRateLimitWhy  string
	groupBlocked       []accountMonitorGroupBlock
	outboxEvents       []accountMonitorOutboxEvent
}

type accountMonitorGroupBlock struct {
	accountID int64
	groupID   int64
	until     time.Time
	reason    string
}

type accountMonitorOutboxEvent struct {
	eventType string
	accountID *int64
	groupID   *int64
	payload   map[string]any
}

func (s *accountMonitorFailureBlockerStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	s.blockedAccountID = id
	s.blockedUntil = until
	s.blockedReason = reason
	return nil
}

func (s *accountMonitorFailureBlockerStub) SetModelRateLimit(_ context.Context, id int64, scope string, resetAt time.Time, reason ...string) error {
	s.modelRateLimitID = id
	s.modelRateLimitKey = scope
	s.modelRateLimitAt = resetAt
	if len(reason) > 0 {
		s.modelRateLimitWhy = reason[0]
	}
	return nil
}

func (s *accountMonitorFailureBlockerStub) SetGroupTempUnschedulable(_ context.Context, id int64, groupID int64, until time.Time, reason string) error {
	s.groupBlocked = append(s.groupBlocked, accountMonitorGroupBlock{
		accountID: id,
		groupID:   groupID,
		until:     until,
		reason:    reason,
	})
	return nil
}

func (s *accountMonitorFailureBlockerStub) ListSchedulableByGroupID(_ context.Context, groupID int64) ([]Account, error) {
	return s.schedulableByGroup[groupID], nil
}

func (s *accountMonitorFailureBlockerStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]Account, error) {
	accounts := s.schedulableByGroup[groupID]
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out, nil
}

func (s *accountMonitorFailureBlockerStub) AppendSchedulerOutboxEvent(_ context.Context, eventType string, accountID *int64, groupID *int64, payload map[string]any) error {
	s.outboxEvents = append(s.outboxEvents, accountMonitorOutboxEvent{
		eventType: eventType,
		accountID: accountID,
		groupID:   groupID,
		payload:   payload,
	})
	return nil
}

// --- tests ---

func TestAccountMonitorCreateRejectsNonAPIKeyAccount(t *testing.T) {
	t.Parallel()
	oauthAcc := &Account{ID: 7, Type: "oauth", Status: StatusActive}
	svc := NewAccountMonitorService(
		&accountMonitorRepoStub{byID: map[int64]*AccountMonitor{}, byAccount: map[int64]*AccountMonitor{}},
		&accountReaderStub{byID: map[int64]*Account{7: oauthAcc}},
	)
	_, err := svc.Create(context.Background(), AccountMonitorCreateParams{AccountID: 7, CreatedBy: 1})
	require.ErrorIs(t, err, ErrAccountMonitorNotEligible)
}

func TestAccountMonitorCreateRejectsInactiveAccount(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.Status = "disabled"
	svc := NewAccountMonitorService(
		&accountMonitorRepoStub{byID: map[int64]*AccountMonitor{}, byAccount: map[int64]*AccountMonitor{}},
		&accountReaderStub{byID: map[int64]*Account{7: acc}},
	)
	_, err := svc.Create(context.Background(), AccountMonitorCreateParams{AccountID: 7, CreatedBy: 1})
	require.ErrorIs(t, err, ErrAccountMonitorNotEligible)
}

func TestAccountMonitorCreateAppliesDefaults(t *testing.T) {
	t.Parallel()
	repo := &accountMonitorRepoStub{byID: map[int64]*AccountMonitor{}, byAccount: map[int64]*AccountMonitor{}}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{7: apiKeyAccount()}})
	m, err := svc.Create(context.Background(), AccountMonitorCreateParams{AccountID: 7, Enabled: true, CreatedBy: 1})
	require.NoError(t, err)
	require.Equal(t, accountMonitorDefaultModel, m.Model)
	require.Equal(t, accountMonitorDefaultInterval, m.IntervalSeconds)
	require.Equal(t, MonitorProviderOpenAI, m.Provider)
}

func TestAccountMonitorCreateRejectsDuplicate(t *testing.T) {
	t.Parallel()
	repo := &accountMonitorRepoStub{
		byID:      map[int64]*AccountMonitor{},
		byAccount: map[int64]*AccountMonitor{7: {ID: 99, AccountID: 7}},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{7: apiKeyAccount()}})
	_, err := svc.Create(context.Background(), AccountMonitorCreateParams{AccountID: 7, CreatedBy: 1})
	require.ErrorIs(t, err, ErrAccountMonitorExists)
}

// RunCheck on an ineligible account records an error check and returns the eligibility error.
func TestAccountMonitorRunCheckIneligibleRecordsErrorCheck(t *testing.T) {
	t.Parallel()
	mon := &AccountMonitor{ID: 1, AccountID: 7, Model: "gpt-5.4-mini", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{byID: map[int64]*AccountMonitor{1: mon}}
	svc := NewAccountMonitorService(repo, &accountReaderStub{err: errors.New("boom")})
	check, err := svc.RunCheck(context.Background(), 1)
	require.ErrorIs(t, err, ErrAccountMonitorNotEligible)
	require.NotNil(t, check)
	require.Equal(t, MonitorStatusError, check.Status)
	require.Len(t, repo.checks, 1)
}

func TestAccountMonitorProbeAllowsErrorAccount(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.Status = StatusError
	svc := NewAccountMonitorService(
		&accountMonitorRepoStub{byID: map[int64]*AccountMonitor{}, byAccount: map[int64]*AccountMonitor{}},
		&accountReaderStub{byID: map[int64]*Account{7: acc}},
	)

	got, err := svc.eligibleAccountForProbe(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, int64(7), got.ID)
}

func TestAccountMonitorCreateStillRejectsErrorAccount(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.Status = StatusError
	svc := NewAccountMonitorService(
		&accountMonitorRepoStub{byID: map[int64]*AccountMonitor{}, byAccount: map[int64]*AccountMonitor{}},
		&accountReaderStub{byID: map[int64]*Account{7: acc}},
	)

	_, err := svc.Create(context.Background(), AccountMonitorCreateParams{AccountID: 7, CreatedBy: 1})

	require.ErrorIs(t, err, ErrAccountMonitorNotEligible)
}

func TestAccountMonitorRecoverAfterSuccessfulStatus(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	recovery := &accountMonitorRecoveryStub{}
	svc := NewAccountMonitorService(
		&accountMonitorRepoStub{byID: map[int64]*AccountMonitor{}, byAccount: map[int64]*AccountMonitor{}},
		&accountReaderStub{byID: map[int64]*Account{7: acc}},
	)
	svc.SetRecovery(recovery)

	svc.recoverAccountAfterSuccessfulMonitor(context.Background(), acc, &AccountMonitorCheck{Status: MonitorStatusOperational, Model: "gpt-5.5"})

	require.Empty(t, recovery.calls)
	require.Equal(t, []string{"7:gpt-5.5"}, recovery.modelCalls)
}

func TestAccountMonitorBlocksOpenAIModelAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusFailed},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: "upstream HTTP 503",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Equal(t, acc.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5", blocker.modelRateLimitKey)
	require.True(t, blocker.modelRateLimitAt.After(time.Now()))
	require.Contains(t, blocker.modelRateLimitWhy, "upstream HTTP 503")
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, SchedulerOutboxEventSchedulingBlocked, blocker.outboxEvents[0].eventType)
	require.Equal(t, "account_monitor_consecutive_failures", blocker.outboxEvents[0].payload["reason"])
	require.Equal(t, "account_monitor", blocker.outboxEvents[0].payload["source"])
	require.Equal(t, accountMonitorFailureBlockThreshold, blocker.outboxEvents[0].payload["failure_threshold"])
	require.Equal(t, "model", blocker.outboxEvents[0].payload["block_granularity"])
}

func TestAccountMonitorDoesNotRepeatTransientModelBlockEventDuringActiveCooldown(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	acc.Extra = map[string]any{
		modelRateLimitsKey: map[string]any{
			"gpt-5.5": map[string]any{
				"rate_limit_reset_at": time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: "upstream HTTP 503",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Zero(t, blocker.modelRateLimitID)
	require.Empty(t, blocker.outboxEvents)
}

func TestAccountMonitorBlocksUngroupedOpenAIModelAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = nil
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: "upstream HTTP 503",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Equal(t, acc.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5", blocker.modelRateLimitKey)
	require.True(t, blocker.modelRateLimitAt.After(time.Now()))
	require.Empty(t, blocker.groupBlocked)
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, SchedulerOutboxEventSchedulingBlocked, blocker.outboxEvents[0].eventType)
	require.Nil(t, blocker.outboxEvents[0].groupID)
	require.Equal(t, "model", blocker.outboxEvents[0].payload["block_granularity"])
}

func TestAccountMonitorDoesNotBlockAfterConsecutiveDegradedChecks(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusDegraded},
				{Status: MonitorStatusDegraded},
				{Status: MonitorStatusDegraded},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusDegraded,
		Message: "slow response: 9000ms",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Zero(t, blocker.modelRateLimitID)
	require.Empty(t, blocker.outboxEvents)
}

func TestAccountMonitorHardFailureUsesLongCooldown(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: `upstream HTTP 403: {"code":"INSUFFICIENT_BALANCE","message":"Insufficient account balance"}`,
	})

	require.Equal(t, acc.ID, blocker.blockedAccountID)
	require.True(t, blocker.blockedUntil.After(time.Now().Add(23*time.Hour)))
	require.Contains(t, blocker.blockedReason, "account_monitor_insufficient_balance")
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, accountMonitorHardFailureCooldownHours*60, blocker.outboxEvents[0].payload["cooldown_minutes"])
	require.Equal(t, "account_monitor_insufficient_balance", blocker.outboxEvents[0].payload["failure_category"])
}

func TestAccountMonitorClassifiesUpstreamGroupDeletedAsHardFailure(t *testing.T) {
	t.Parallel()
	got := classifyAccountMonitorFailure(`upstream HTTP 403: {"code":"GROUP_DELETED","message":"API Key 所属分组已删除"}`)

	require.True(t, got.Hard)
	require.Equal(t, "account_monitor_upstream_group_unavailable", got.Reason)
	require.Equal(t, accountMonitorHardFailureCooldownHours*60, got.CooldownMinutes)
}

func TestAccountMonitorClassifiesUnauthorizedAsHardFailure(t *testing.T) {
	t.Parallel()
	got := classifyAccountMonitorFailure(`upstream HTTP 401: {"error":{"message":"Unauthorized"}}`)

	require.True(t, got.Hard)
	require.Equal(t, "account_monitor_auth_failed", got.Reason)
}

func TestAccountMonitorClassifiesEndpointMigratedAsHardFailure(t *testing.T) {
	t.Parallel()
	got := classifyAccountMonitorFailure(`upstream HTTP 410: {"error":{"type":"endpoint_migrated","message":"The API endpoint is not served from the panel domain. Please use the published API endpoint for this service."}}`)

	require.True(t, got.Hard)
	require.Equal(t, "account_monitor_endpoint_migrated", got.Reason)
	require.Equal(t, accountMonitorHardFailureCooldownHours*60, got.CooldownMinutes)
}

func TestAccountMonitorModelNotFoundBlocksOnlyModel(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	acc.Credentials["model_mapping"] = map[string]any{"gpt-5.5": "gpt-5.5-upstream", "gpt-5.4": "gpt-5.4-upstream"}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: `upstream HTTP 503: {"error":{"code":"model_not_found","message":"No available channel for model gpt-5.5"}}`,
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Equal(t, acc.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5-upstream", blocker.modelRateLimitKey)
	require.Equal(t, upstreamModelNotFoundReason, blocker.modelRateLimitWhy)
	require.True(t, blocker.modelRateLimitAt.After(time.Now()))
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, SchedulerOutboxEventSchedulingBlocked, blocker.outboxEvents[0].eventType)
	require.Equal(t, "account_monitor_model_unsupported", blocker.outboxEvents[0].payload["reason"])
	require.Equal(t, "model", blocker.outboxEvents[0].payload["block_granularity"])
	require.Equal(t, "gpt-5.5-upstream", blocker.outboxEvents[0].payload["model_rate_limit"])
}

func TestAccountMonitorDoesNotRepeatModelBlockEventDuringActiveCooldown(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	acc.Credentials["model_mapping"] = map[string]any{"gpt-5.5": "gpt-5.5-upstream"}
	acc.Extra = map[string]any{
		modelRateLimitsKey: map[string]any{
			"gpt-5.5-upstream": map[string]any{
				"rate_limit_reset_at": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: `upstream HTTP 503: {"error":{"code":"model_not_found","message":"No available channel for model gpt-5.5"}}`,
	})

	require.Equal(t, acc.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5-upstream", blocker.modelRateLimitKey)
	require.Empty(t, blocker.outboxEvents)
}

func TestAccountMonitorDoesNotRepeatModelBlockEventWithStaleAccountSnapshot(t *testing.T) {
	t.Parallel()
	stale := apiKeyAccount()
	stale.GroupIDs = []int64{10}
	stale.Credentials["model_mapping"] = map[string]any{"gpt-5.5": "gpt-5.5-upstream"}
	fresh := apiKeyAccount()
	fresh.GroupIDs = []int64{10}
	fresh.Credentials["model_mapping"] = map[string]any{"gpt-5.5": "gpt-5.5-upstream"}
	fresh.Extra = map[string]any{
		modelRateLimitsKey: map[string]any{
			"gpt-5.5-upstream": map[string]any{
				"rate_limit_reset_at": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	mon := &AccountMonitor{ID: 1, AccountID: stale.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*fresh, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{fresh.ID: fresh}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, stale, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: `upstream HTTP 503: {"error":{"code":"model_not_found","message":"No available channel for model gpt-5.5"}}`,
	})

	require.Equal(t, stale.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5-upstream", blocker.modelRateLimitKey)
	require.Empty(t, blocker.outboxEvents)
}

func TestAccountMonitorModelNotFoundWaitsForConsecutiveFailures(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: `upstream HTTP 503: {"error":{"code":"model_not_found","message":"No available channel for model gpt-5.5"}}`,
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Zero(t, blocker.modelRateLimitID)
	require.Empty(t, blocker.outboxEvents)
}

func TestAccountMonitorModelBlockAppliesToLastGroupAccount(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusFailed},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{10: {*acc}},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: "upstream HTTP 503",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Equal(t, acc.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5", blocker.modelRateLimitKey)
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, SchedulerOutboxEventSchedulingBlocked, blocker.outboxEvents[0].eventType)
	require.Equal(t, "model", blocker.outboxEvents[0].payload["block_granularity"])
}

func TestAccountMonitorModelBlockIsSharedAcrossAccountGroups(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10, 20}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusFailed},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc},
			20: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: "upstream HTTP 503",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Equal(t, acc.ID, blocker.modelRateLimitID)
	require.Equal(t, "gpt-5.5", blocker.modelRateLimitKey)
	require.Empty(t, blocker.groupBlocked)
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, SchedulerOutboxEventSchedulingBlocked, blocker.outboxEvents[0].eventType)
	require.Equal(t, "model", blocker.outboxEvents[0].payload["block_granularity"])
}

func TestAccountMonitorDoesNotRepeatModelBlockAcrossMultipleGroups(t *testing.T) {
	t.Parallel()
	acc := apiKeyAccount()
	acc.GroupIDs = []int64{10, 20}
	acc.Extra = map[string]any{
		modelRateLimitsKey: map[string]any{
			"gpt-5.5": map[string]any{
				"until":               time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
				"rate_limit_reset_at": time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	mon := &AccountMonitor{ID: 1, AccountID: acc.ID, Model: "gpt-5.5", Provider: MonitorProviderOpenAI, Enabled: true}
	repo := &accountMonitorRepoStub{
		byID: map[int64]*AccountMonitor{1: mon},
		recent: map[int64][]*AccountMonitorCheck{
			1: {
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
				{Status: MonitorStatusError},
			},
		},
	}
	blocker := &accountMonitorFailureBlockerStub{
		schedulableByGroup: map[int64][]Account{
			10: {*acc},
			20: {*acc, {ID: 8, Platform: PlatformOpenAI, Status: StatusActive}},
		},
	}
	svc := NewAccountMonitorService(repo, &accountReaderStub{byID: map[int64]*Account{acc.ID: acc}})
	svc.SetFailureBlocker(blocker)

	svc.blockAccountAfterConsecutiveMonitorFailures(context.Background(), mon, acc, &AccountMonitorCheck{
		Status:  MonitorStatusError,
		Message: "upstream HTTP 503",
	})

	require.Zero(t, blocker.blockedAccountID)
	require.Zero(t, blocker.modelRateLimitID)
	require.Empty(t, blocker.groupBlocked)
	require.Empty(t, blocker.outboxEvents)
}
