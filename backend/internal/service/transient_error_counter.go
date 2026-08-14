package service

import "context"

// TransientErrorCounterCache tracks repeated transient upstream failures before
// temporarily removing pool-mode accounts from scheduling.
type TransientErrorCounterCache interface {
	IncrementTransientErrorCount(ctx context.Context, accountID int64, windowMinutes int) (int64, error)
	ResetTransientErrorCount(ctx context.Context, accountID int64) error
}

// TransientErrorRequestCounter is an optional extension for counters backed by
// a shared store. Pool mode can retry the same account several times for one
// logical request; those retries must not inflate the account's 5xx streak.
// Implementations return incremented=false when this request/account pair was
// already counted during the active window.
type TransientErrorRequestCounter interface {
	IncrementTransientErrorCountOnce(ctx context.Context, accountID int64, requestID string, windowMinutes int) (count int64, incremented bool, err error)
}
