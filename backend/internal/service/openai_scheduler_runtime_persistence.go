package service

import (
	"context"
	"time"
)

// OpenAIAccountRuntimeStatRecord is the durable form of the model-scoped
// scheduler health snapshot. Runtime selection keeps atomics in memory for the
// hot path, while this record lets a restart resume from the recent history.
type OpenAIAccountRuntimeStatRecord struct {
	AccountID      int64
	CanonicalModel string
	ErrorRateEWMA  float64
	TTFTEWMA       *float64
	SampleCount    int64
	TTFTSamples    int64
	UpdatedAt      time.Time

	// The two short-lived safety states are persisted alongside the EWMA so a
	// restart does not immediately send traffic back to an account that was
	// just failing or timing out. Their own TTLs are still authoritative when
	// the process restores them.
	TransientFailureStreak   int
	TransientLastFailureAt   *time.Time
	TransientBlockUntil      *time.Time
	SlowReserveMarkedAt      *time.Time
	SlowReserveLastTouchedAt *time.Time
	SlowReserveExpiresAt     *time.Time
	SlowReserveReason        string
	SlowReserveTTFTMs        int
}

// OpenAIAccountRuntimeStatsStore is intentionally optional. Production's
// account repository implements it; unit-test repositories and embedded users
// that do not provide the store continue to use the in-memory scheduler.
type OpenAIAccountRuntimeStatsStore interface {
	LoadOpenAIAccountRuntimeStats(ctx context.Context) ([]OpenAIAccountRuntimeStatRecord, error)
	SaveOpenAIAccountRuntimeStats(ctx context.Context, records []OpenAIAccountRuntimeStatRecord) error
}

// OpenAISchedulerSafetyStateRecord persists only the short-lived breaker and
// slow-reserve fields. Keeping this update separate prevents a safety-state
// notification from overwriting a newer EWMA snapshot that is still queued.
type OpenAISchedulerSafetyStateRecord struct {
	AccountID                int64
	CanonicalModel           string
	TransientFailureStreak   int
	TransientLastFailureAt   *time.Time
	TransientBlockUntil      *time.Time
	SlowReserveMarkedAt      *time.Time
	SlowReserveLastTouchedAt *time.Time
	SlowReserveExpiresAt     *time.Time
	SlowReserveReason        string
	SlowReserveTTFTMs        int
}

type OpenAISchedulerSafetyStateStore interface {
	SaveOpenAISchedulerSafetyState(ctx context.Context, record OpenAISchedulerSafetyStateRecord) error
}
