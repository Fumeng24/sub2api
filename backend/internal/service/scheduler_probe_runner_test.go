package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type schedulerProbeRunnerAdapterStub struct {
	results       []schedulerProbeRunnerResult
	recovered     bool
	unschedulable bool
	continueCount int
}

type schedulerProbeRunnerResult struct {
	statusCode int
	body       []byte
	ttftMs     int
	err        error
}

func (s *schedulerProbeRunnerAdapterStub) Probe(ctx context.Context, key schedulerProbeKey) (int, []byte, int, error) {
	if len(s.results) == 0 {
		return 0, nil, 0, errors.New("no probe result")
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result.statusCode, result.body, result.ttftMs, result.err
}

func (s *schedulerProbeRunnerAdapterStub) OnRecovered(key schedulerProbeKey) {
	s.recovered = true
}

func (s *schedulerProbeRunnerAdapterStub) OnUnschedulable(key schedulerProbeKey) {
	s.unschedulable = true
}

func (s *schedulerProbeRunnerAdapterStub) ShouldContinue(key schedulerProbeKey, category string) bool {
	s.continueCount++
	return false
}

func (s *schedulerProbeRunnerAdapterStub) LogAttrs(key schedulerProbeKey) []any {
	return []any{"account_id", key.AccountID, "model", key.Model, "endpoint", key.Endpoint}
}

func TestSchedulerProbeRunner_HealthyTTFTRecovers(t *testing.T) {
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	schedulerHealthyTTFTThreshold = 15 * time.Millisecond
	defer func() { schedulerHealthyTTFTThreshold = oldHealthyTTFT }()

	health := newAccountSchedulerHealthStats()
	key := makeAccountSchedulerHealthKey(1, "model", "endpoint")
	health.reportFailure(key.AccountID, key.Model, key.Endpoint, schedulerSlowTTFTCategory, time.Minute)
	adapter := &schedulerProbeRunnerAdapterStub{results: []schedulerProbeRunnerResult{
		{statusCode: 200, ttftMs: 10},
	}}

	schedulerProbeRunner{health: health, adapter: adapter}.run(context.Background(), key, schedulerSlowTTFTCategory)

	require.True(t, adapter.recovered)
	snap := health.snapshot(key.AccountID, key.Model, key.Endpoint, false)
	require.Equal(t, schedulerCircuitClosed, snap.CircuitState)
}

func TestSchedulerProbeRunner_SlowTTFTRetriesWithoutRecovering(t *testing.T) {
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	schedulerHealthyTTFTThreshold = 15 * time.Millisecond
	defer func() { schedulerHealthyTTFTThreshold = oldHealthyTTFT }()

	health := newAccountSchedulerHealthStats()
	key := makeAccountSchedulerHealthKey(2, "model", "endpoint")
	adapter := &schedulerProbeRunnerAdapterStub{results: []schedulerProbeRunnerResult{
		{statusCode: 200, ttftMs: 20},
	}}

	schedulerProbeRunner{health: health, adapter: adapter}.run(context.Background(), key, "transient_timeout")

	require.False(t, adapter.recovered)
	require.Equal(t, 1, adapter.continueCount)
	snap := health.snapshot(key.AccountID, key.Model, key.Endpoint, false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
	require.Equal(t, schedulerSlowTTFTCategory, snap.LastFailureReason)
}

func TestSchedulerProbeRunner_EmptySuccessStreamClassifiedAsTimeout(t *testing.T) {
	health := newAccountSchedulerHealthStats()
	key := makeAccountSchedulerHealthKey(3, "model", "endpoint")
	adapter := &schedulerProbeRunnerAdapterStub{results: []schedulerProbeRunnerResult{
		{statusCode: 200, err: errors.New("probe response did not produce first token")},
	}}

	schedulerProbeRunner{health: health, adapter: adapter}.run(context.Background(), key, "transient_timeout")

	require.False(t, adapter.recovered)
	snap := health.snapshot(key.AccountID, key.Model, key.Endpoint, false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
	require.Equal(t, schedulerSlowTTFTCategory, snap.LastFailureReason)
}

func TestSchedulerProbeRunnerConfigFromConfig(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			OpenAIScheduler: config.GatewayOpenAISchedulerConfig{
				RuntimeCooldowns: config.GatewayOpenAIRuntimeCooldownsConfig{
					ProbeTimeoutSeconds:    7,
					ProbeRetryDelaySeconds: 2,
					ProbeMaxConcurrency:    3,
				},
			},
		},
	}

	got := schedulerProbeRunnerConfigFromConfig(cfg, 30*time.Second, 5*time.Second)

	require.Equal(t, 7*time.Second, got.timeout)
	require.Equal(t, 2*time.Second, got.retryDelay)
	require.Equal(t, 3, got.maxConcurrency)
}

func TestSchedulerProbeRunner_ProbeTimeoutCancelsAttempt(t *testing.T) {
	health := newAccountSchedulerHealthStats()
	key := makeAccountSchedulerHealthKey(4, "model", "endpoint")
	adapter := &schedulerProbeTimeoutAdapter{}
	before := GetSchedulerProbeRunnerMetricsSnapshot()

	started := time.Now()
	schedulerProbeRunner{
		health:     health,
		adapter:    adapter,
		timeout:    20 * time.Millisecond,
		retryDelay: time.Hour,
	}.run(context.Background(), key, "transient_timeout")

	require.GreaterOrEqual(t, time.Since(started), 20*time.Millisecond)
	require.Less(t, time.Since(started), 200*time.Millisecond)
	require.Equal(t, 1, adapter.continueCount)
	after := GetSchedulerProbeRunnerMetricsSnapshot()
	require.Equal(t, before.Timeouts+1, after.Timeouts)
	snap := health.snapshot(key.AccountID, key.Model, key.Endpoint, false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
}

func TestSchedulerProbeRunner_ConcurrencyLimitWaitsForSlot(t *testing.T) {
	key := makeAccountSchedulerHealthKey(5, "model", "endpoint")
	limiter := newSchedulerProbeLimiter(1)
	first := &schedulerProbeBlockingAdapter{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	second := &schedulerProbeCountingAdapter{}
	before := GetSchedulerProbeRunnerMetricsSnapshot()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		schedulerProbeRunner{
			adapter: first,
			timeout: time.Second,
			limiter: limiter,
		}.run(context.Background(), key, "transient_timeout")
	}()

	select {
	case <-first.started:
	case <-time.After(time.Second):
		t.Fatal("first probe did not acquire slot")
	}

	schedulerProbeRunner{
		adapter: second,
		timeout: 20 * time.Millisecond,
		limiter: limiter,
	}.run(context.Background(), key, "transient_timeout")

	require.Equal(t, 0, second.calls)
	after := GetSchedulerProbeRunnerMetricsSnapshot()
	require.Equal(t, before.ConcurrencyWaitTimeout+1, after.ConcurrencyWaitTimeout)

	close(first.release)
	wg.Wait()
}

type schedulerProbeTimeoutAdapter struct {
	continueCount int
}

func (a *schedulerProbeTimeoutAdapter) Probe(ctx context.Context, key schedulerProbeKey) (int, []byte, int, error) {
	<-ctx.Done()
	return 0, nil, 0, ctx.Err()
}

func (a *schedulerProbeTimeoutAdapter) OnRecovered(key schedulerProbeKey) {}

func (a *schedulerProbeTimeoutAdapter) OnUnschedulable(key schedulerProbeKey) {}

func (a *schedulerProbeTimeoutAdapter) ShouldContinue(key schedulerProbeKey, category string) bool {
	a.continueCount++
	return false
}

func (a *schedulerProbeTimeoutAdapter) LogAttrs(key schedulerProbeKey) []any {
	return []any{"account_id", key.AccountID, "model", key.Model, "endpoint", key.Endpoint}
}

type schedulerProbeBlockingAdapter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (a *schedulerProbeBlockingAdapter) Probe(ctx context.Context, key schedulerProbeKey) (int, []byte, int, error) {
	a.once.Do(func() { close(a.started) })
	select {
	case <-ctx.Done():
		return 0, nil, 0, ctx.Err()
	case <-a.release:
		return 200, nil, 1, nil
	}
}

func (a *schedulerProbeBlockingAdapter) OnRecovered(key schedulerProbeKey) {}

func (a *schedulerProbeBlockingAdapter) OnUnschedulable(key schedulerProbeKey) {}

func (a *schedulerProbeBlockingAdapter) ShouldContinue(key schedulerProbeKey, category string) bool {
	return false
}

func (a *schedulerProbeBlockingAdapter) LogAttrs(key schedulerProbeKey) []any {
	return []any{"account_id", key.AccountID, "model", key.Model, "endpoint", key.Endpoint}
}

type schedulerProbeCountingAdapter struct {
	calls int
}

func (a *schedulerProbeCountingAdapter) Probe(ctx context.Context, key schedulerProbeKey) (int, []byte, int, error) {
	a.calls++
	return 200, nil, 1, nil
}

func (a *schedulerProbeCountingAdapter) OnRecovered(key schedulerProbeKey) {}

func (a *schedulerProbeCountingAdapter) OnUnschedulable(key schedulerProbeKey) {}

func (a *schedulerProbeCountingAdapter) ShouldContinue(key schedulerProbeKey, category string) bool {
	return false
}

func (a *schedulerProbeCountingAdapter) LogAttrs(key schedulerProbeKey) []any {
	return []any{"account_id", key.AccountID, "model", key.Model, "endpoint", key.Endpoint}
}
