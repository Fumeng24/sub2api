package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type openAISchedulerRuntimeStatsStoreStub struct {
	records []OpenAIAccountRuntimeStatRecord
}

func (s *openAISchedulerRuntimeStatsStoreStub) LoadOpenAIAccountRuntimeStats(context.Context) ([]OpenAIAccountRuntimeStatRecord, error) {
	return append([]OpenAIAccountRuntimeStatRecord(nil), s.records...), nil
}

func (s *openAISchedulerRuntimeStatsStoreStub) SaveOpenAIAccountRuntimeStats(_ context.Context, records []OpenAIAccountRuntimeStatRecord) error {
	s.records = append([]OpenAIAccountRuntimeStatRecord(nil), records...)
	return nil
}

func TestOpenAIAccountRuntimeStats_LoadPersistedRestoresSafetyState(t *testing.T) {
	updatedAt := time.Now().UTC().Add(-2 * time.Minute)
	lastFailure := updatedAt.Add(30 * time.Second)
	blockUntil := time.Now().Add(20 * time.Second)
	markedAt := updatedAt.Add(45 * time.Second)
	lastTouched := time.Now().UTC()
	expiresAt := time.Now().Add(3 * time.Minute)
	ttft := 17200.0
	store := &openAISchedulerRuntimeStatsStoreStub{records: []OpenAIAccountRuntimeStatRecord{{
		AccountID:                9001,
		CanonicalModel:           "GPT-5.6-SOL",
		ErrorRateEWMA:            0.75,
		TTFTEWMA:                 &ttft,
		SampleCount:              24,
		TTFTSamples:              18,
		UpdatedAt:                updatedAt,
		TransientFailureStreak:   3,
		TransientLastFailureAt:   &lastFailure,
		TransientBlockUntil:      &blockUntil,
		SlowReserveMarkedAt:      &markedAt,
		SlowReserveLastTouchedAt: &lastTouched,
		SlowReserveExpiresAt:     &expiresAt,
		SlowReserveReason:        "ttft",
		SlowReserveTTFTMs:        17200,
	}}}

	stats := newOpenAIAccountRuntimeStats()
	transient := newOpenAIAccountModelTransientState(16)
	slowReserve := newOpenAIAccountSlowReserveState()
	stats.attachSafetyStateSources(transient, slowReserve)
	require.NoError(t, stats.loadPersisted(context.Background(), store))

	errorRate, gotTTFT, hasTTFT := stats.snapshot(9001, "gpt-5.6-sol")
	require.InDelta(t, 0.75, errorRate, 1e-9)
	require.True(t, hasTTFT)
	require.InDelta(t, 17200.0, gotTTFT, 1e-9)
	require.Equal(t, 1, stats.size())
	require.Equal(t, 3, transient.activeFailureStreak(9001, "gpt-5.6-sol", time.Now()))
	require.True(t, slowReserve.isReserved(9001, "gpt-5.6-sol", time.Now()))
}
