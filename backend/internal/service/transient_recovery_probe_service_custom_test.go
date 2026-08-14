package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type transientRecoveryAccountStub struct {
	mu       sync.Mutex
	accounts map[int64]*Account
}

func (s *transientRecoveryAccountStub) GetByID(_ context.Context, id int64) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	clone := *account
	return &clone, nil
}

func (s *transientRecoveryAccountStub) ListActive(_ context.Context) ([]Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := make([]Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		clone := *account
		accounts = append(accounts, clone)
	}
	return accounts, nil
}

type transientRecoveryProberStub struct {
	mu      sync.Mutex
	calls   int
	models  []string
	result  *CheckResult
	started chan struct{}
	release <-chan struct{}
}

func (s *transientRecoveryProberStub) ProbeTransientRecovery(_ context.Context, _ int64, requestedModel string) (*CheckResult, error) {
	s.mu.Lock()
	s.calls++
	s.models = append(s.models, requestedModel)
	started := s.started
	release := s.release
	result := s.result
	s.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if release != nil {
		<-release
	}
	return result, nil
}

func (s *transientRecoveryProberStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type transientRecoveryCooldownStub struct {
	mu          sync.Mutex
	renewCalls  int
	clearCalls  int
	renewResult bool
	clearResult bool
	cleared     chan struct{}
}

func (s *transientRecoveryCooldownStub) RenewTransient5xxCooldown(_ context.Context, _ int64, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.renewCalls++
	return s.renewResult, nil
}

func (s *transientRecoveryCooldownStub) ClearTransient5xxCooldown(_ context.Context, _ int64) (bool, error) {
	s.mu.Lock()
	s.clearCalls++
	cleared := s.cleared
	result := s.clearResult
	s.mu.Unlock()
	if cleared != nil {
		close(cleared)
	}
	return result, nil
}

func (s *transientRecoveryCooldownStub) counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.renewCalls, s.clearCalls
}

func transientRecoveryTestAccount(until time.Time) *Account {
	return &Account{
		ID:                      42,
		Status:                  StatusActive,
		Schedulable:             true,
		Type:                    AccountTypeAPIKey,
		Platform:                PlatformAnthropic,
		Credentials:             map[string]any{"api_key": "test"},
		TempUnschedulableUntil:  &until,
		TempUnschedulableReason: PoolTransient5xxCooldownReasonPrefix + ": temporary upstream failure",
	}
}

func startTransientRecoveryForTest(svc *TransientRecoveryProbeService) {
	svc.mu.Lock()
	svc.started = true
	svc.mu.Unlock()
}

func TestTransientRecoveryProbe_SuccessClearsOnlyTransientCooldown(t *testing.T) {
	until := time.Now().Add(5 * time.Minute)
	accounts := &transientRecoveryAccountStub{accounts: map[int64]*Account{42: transientRecoveryTestAccount(until)}}
	prober := &transientRecoveryProberStub{result: &CheckResult{Model: "claude-test", Status: MonitorStatusOperational}}
	cooldowns := &transientRecoveryCooldownStub{clearResult: true, cleared: make(chan struct{})}
	svc := newTransientRecoveryProbeService(accounts, prober, cooldowns)
	startTransientRecoveryForTest(svc)
	defer svc.Stop()

	require.True(t, svc.schedule(42, "claude-requested", 0))
	select {
	case <-cooldowns.cleared:
	case <-time.After(time.Second):
		t.Fatal("expected successful probe to clear cooldown")
	}

	renewCalls, clearCalls := cooldowns.counts()
	require.Zero(t, renewCalls)
	require.Equal(t, 1, clearCalls)
	require.Equal(t, 1, prober.callCount())
	prober.mu.Lock()
	require.Equal(t, []string{"claude-requested"}, prober.models)
	prober.mu.Unlock()
}

func TestTransientRecoveryProbe_FailureRenewsBeforeShortCooldownExpires(t *testing.T) {
	until := time.Now().Add(time.Second)
	accounts := &transientRecoveryAccountStub{accounts: map[int64]*Account{42: transientRecoveryTestAccount(until)}}
	prober := &transientRecoveryProberStub{
		result:  &CheckResult{Model: "claude-test", Status: MonitorStatusError},
		started: make(chan struct{}, 1),
	}
	cooldowns := &transientRecoveryCooldownStub{renewResult: true}
	svc := newTransientRecoveryProbeService(accounts, prober, cooldowns)
	startTransientRecoveryForTest(svc)

	require.True(t, svc.schedule(42, "", 0))
	select {
	case <-prober.started:
	case <-time.After(time.Second):
		t.Fatal("expected recovery probe to run")
	}
	svc.Stop()

	renewCalls, clearCalls := cooldowns.counts()
	require.Equal(t, 1, renewCalls)
	require.Zero(t, clearCalls)
}

func TestTransientRecoveryProbe_ManualDisableNeverProbesOrClears(t *testing.T) {
	until := time.Now().Add(time.Minute)
	account := transientRecoveryTestAccount(until)
	account.Schedulable = false
	accounts := &transientRecoveryAccountStub{accounts: map[int64]*Account{42: account}}
	prober := &transientRecoveryProberStub{result: &CheckResult{Status: MonitorStatusOperational}}
	cooldowns := &transientRecoveryCooldownStub{clearResult: true}
	svc := newTransientRecoveryProbeService(accounts, prober, cooldowns)
	startTransientRecoveryForTest(svc)
	defer svc.Stop()

	require.True(t, svc.schedule(42, "claude-requested", 0))
	require.Eventually(t, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		return len(svc.tasks) == 0
	}, time.Second, 10*time.Millisecond)
	renewCalls, clearCalls := cooldowns.counts()
	require.Zero(t, renewCalls)
	require.Zero(t, clearCalls)
	require.Zero(t, prober.callCount())
}

func TestTransientRecoveryProbe_DeduplicatesAccountTask(t *testing.T) {
	until := time.Now().Add(5 * time.Minute)
	accounts := &transientRecoveryAccountStub{accounts: map[int64]*Account{42: transientRecoveryTestAccount(until)}}
	release := make(chan struct{})
	prober := &transientRecoveryProberStub{
		result:  &CheckResult{Model: "claude-test", Status: MonitorStatusOperational},
		started: make(chan struct{}, 1),
		release: release,
	}
	cooldowns := &transientRecoveryCooldownStub{clearResult: true, cleared: make(chan struct{})}
	svc := newTransientRecoveryProbeService(accounts, prober, cooldowns)
	startTransientRecoveryForTest(svc)
	defer svc.Stop()

	require.True(t, svc.schedule(42, "first", 0))
	select {
	case <-prober.started:
	case <-time.After(time.Second):
		t.Fatal("expected first probe to start")
	}
	require.False(t, svc.schedule(42, "second", 0))
	close(release)
	select {
	case <-cooldowns.cleared:
	case <-time.After(time.Second):
		t.Fatal("expected first probe to finish")
	}
	require.Equal(t, 1, prober.callCount())
}

func TestTransientRecoveryProbe_StartRestoresPersistedEpisode(t *testing.T) {
	expired := time.Now().Add(-time.Second)
	accounts := &transientRecoveryAccountStub{accounts: map[int64]*Account{42: transientRecoveryTestAccount(expired)}}
	prober := &transientRecoveryProberStub{result: &CheckResult{Model: "claude-test", Status: MonitorStatusOperational}}
	cooldowns := &transientRecoveryCooldownStub{renewResult: true, clearResult: true, cleared: make(chan struct{})}
	svc := newTransientRecoveryProbeService(accounts, prober, cooldowns)
	defer svc.Stop()

	svc.Start()
	select {
	case <-cooldowns.cleared:
	case <-time.After(time.Second):
		t.Fatal("expected startup scan to probe persisted transient cooldown")
	}
	renewCalls, clearCalls := cooldowns.counts()
	require.Equal(t, 1, renewCalls)
	require.Equal(t, 1, clearCalls)
	require.Equal(t, 1, prober.callCount())
}

func TestIsTransient5xxCooldownReason(t *testing.T) {
	require.True(t, isTransient5xxCooldownReason(Transient5xxCooldownReasonPrefix+": 503"))
	require.True(t, isTransient5xxCooldownReason(PoolTransient5xxCooldownReasonPrefix+": 502"))
	require.False(t, isTransient5xxCooldownReason("account_monitor_auth_failed"))
}
