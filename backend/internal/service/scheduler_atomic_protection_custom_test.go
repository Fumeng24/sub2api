package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type transientRecoverySchedulerRecorder struct {
	accountID int64
	model     string
	calls     int
}

func (s *transientRecoverySchedulerRecorder) ScheduleTransientRecoveryProbe(accountID int64, requestedModel string) {
	s.accountID = accountID
	s.model = requestedModel
	s.calls++
}

func TestPoolTransientCooldownUsesAccountProtection(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &RateLimitService{accountRepo: repo}
	account := &Account{ID: 38805, Platform: PlatformOpenAI, GroupIDs: []int64{10}}
	until := time.Now().Add(10 * time.Minute)

	applied := svc.setPoolTransientTempUnschedulable(context.Background(), account, until, "pool_transient_5xx: upstream_503", 503)

	require.True(t, applied)
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, account.ID, repo.lastTempID)
	require.Len(t, repo.outboxEvents, 1)
	require.Equal(t, "account", repo.outboxEvents[0].payload["block_granularity"])
	require.Equal(t, "pool_transient_5xx", repo.outboxEvents[0].payload["source"])
}

func TestNonPoolTransientCooldownUsesAccountProtection(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &RateLimitService{accountRepo: repo}
	account := &Account{ID: 38839, Platform: PlatformOpenAI, GroupIDs: []int64{10}}

	applied := svc.handleTransient5xx(context.Background(), account, 503, "temporarily unavailable")

	require.True(t, applied)
	require.Equal(t, 1, repo.tempCalls)
	require.Len(t, repo.outboxEvents, 1)
	require.Equal(t, "upstream_transient_5xx: temporarily unavailable", repo.outboxEvents[0].payload["reason"])
	require.Equal(t, "account", repo.outboxEvents[0].payload["block_granularity"])
}

func TestTransientCooldownSchedulesActiveRecoveryWithoutBlockingRequestPath(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	recovery := &transientRecoverySchedulerRecorder{}
	svc := &RateLimitService{accountRepo: repo, transientRecovery: recovery}
	account := &Account{ID: 38840, Platform: PlatformAnthropic, GroupIDs: []int64{10}}

	applied := svc.handleTransient5xx(context.Background(), account, 503, "temporarily unavailable", "claude-sonnet-4-5")

	require.True(t, applied)
	require.Equal(t, 1, recovery.calls)
	require.Equal(t, account.ID, recovery.accountID)
	require.Equal(t, "claude-sonnet-4-5", recovery.model)
}
