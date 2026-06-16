package service

import (
	"context"
	"errors"
	"testing"
	"time"

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
