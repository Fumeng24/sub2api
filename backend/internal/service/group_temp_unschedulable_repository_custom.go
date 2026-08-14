package service

import (
	"context"
	"time"
)

// GroupTempUnschedulableRepository is the optional persistence capability used
// by deployment scheduling policies without widening AccountRepository.
type GroupTempUnschedulableRepository interface {
	SetGroupTempUnschedulable(ctx context.Context, accountID, groupID int64, until time.Time, reason string) error
}

// AtomicGroupTempUnschedulableRepository keeps the last-account check and the
// group cooldown write in one database transaction. It remains optional so the
// official AccountRepository contract does not need deployment-specific APIs.
type AtomicGroupTempUnschedulableRepository interface {
	TrySetGroupTempUnschedulableUnlessLast(ctx context.Context, accountID, groupID int64, platform string, until time.Time, reason string) (bool, error)
}
