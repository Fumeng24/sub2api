package service

import "context"

// schedulerCacheAccountBucketRemover is an optional scheduler cache extension.
// The upstream cache contract does not require bucket membership mutation.
type schedulerCacheAccountBucketRemover interface {
	RemoveAccountFromBuckets(ctx context.Context, accountID int64) error
}

// RemoveSchedulerAccountFromBuckets invokes the local bucket-membership
// extension when the configured cache supports it.
func RemoveSchedulerAccountFromBuckets(ctx context.Context, cache SchedulerCache, accountID int64) error {
	if cache == nil || accountID <= 0 {
		return nil
	}
	remover, ok := cache.(schedulerCacheAccountBucketRemover)
	if !ok {
		return nil
	}
	return remover.RemoveAccountFromBuckets(ctx, accountID)
}
