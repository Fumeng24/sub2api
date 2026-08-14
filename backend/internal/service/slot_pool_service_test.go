//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// --- Stub implementations ---

type stubSlotPoolCache struct {
	popResults   []int64 // sequential pop results
	popIndex     int
	popErr       error
	addCalls     []slotPoolAddCall
	addErr       error
	rebuildCalls []slotPoolRebuildCall
	rebuildErr   error
}

type slotPoolAddCall struct {
	Bucket    SchedulerBucket
	AccountID int64
	Priority  int
}

type slotPoolRebuildCall struct {
	Bucket  SchedulerBucket
	Entries []SlotPoolEntry
}

var _ SlotPoolCache = (*stubSlotPoolCache)(nil)

func (c *stubSlotPoolCache) Pop(_ context.Context, _ SchedulerBucket) (int64, error) {
	if c.popErr != nil {
		return 0, c.popErr
	}
	if c.popIndex >= len(c.popResults) {
		return 0, nil // pool empty
	}
	result := c.popResults[c.popIndex]
	c.popIndex++
	return result, nil
}

func (c *stubSlotPoolCache) Add(_ context.Context, bucket SchedulerBucket, accountID int64, priority int, _ time.Time) error {
	c.addCalls = append(c.addCalls, slotPoolAddCall{Bucket: bucket, AccountID: accountID, Priority: priority})
	return c.addErr
}

func (c *stubSlotPoolCache) Remove(_ context.Context, _ SchedulerBucket, _ int64) error {
	return nil
}

func (c *stubSlotPoolCache) Rebuild(_ context.Context, bucket SchedulerBucket, entries []SlotPoolEntry) error {
	c.rebuildCalls = append(c.rebuildCalls, slotPoolRebuildCall{Bucket: bucket, Entries: entries})
	return c.rebuildErr
}

func (c *stubSlotPoolCache) Size(_ context.Context, _ SchedulerBucket) (int64, error) {
	return int64(len(c.popResults) - c.popIndex), nil
}

type stubSchedulerCacheForSlotPool struct {
	account    *Account
	accountErr error
	buckets    []SchedulerBucket
	bucketsErr error
	snapshot   []*Account
	snapshotOK bool
}

var _ SchedulerCache = (*stubSchedulerCacheForSlotPool)(nil)

func (c *stubSchedulerCacheForSlotPool) GetSnapshot(_ context.Context, _ SchedulerBucket) ([]*Account, bool, error) {
	return c.snapshot, c.snapshotOK, nil
}
func (c *stubSchedulerCacheForSlotPool) CaptureBucketWriteToken(_ context.Context, bucket SchedulerBucket) (SchedulerBucketWriteToken, error) {
	return SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}
func (c *stubSchedulerCacheForSlotPool) SetSnapshot(_ context.Context, _ SchedulerBucket, _ SchedulerBucketWriteToken, _ []Account) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) RetireBucket(_ context.Context, _ SchedulerBucket) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) ReopenBucket(_ context.Context, bucket SchedulerBucket) (SchedulerBucketWriteToken, error) {
	return SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}
func (c *stubSchedulerCacheForSlotPool) TryAcquireGroupLifecycleLease(_ context.Context, groupID int64, _ time.Duration) (SchedulerGroupLifecycleLease, bool, error) {
	return SchedulerGroupLifecycleLease{GroupID: groupID, OwnerToken: "test"}, true, nil
}
func (c *stubSchedulerCacheForSlotPool) ReleaseGroupLifecycleLease(_ context.Context, _ SchedulerGroupLifecycleLease) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) GetAccount(_ context.Context, _ int64) (*Account, error) {
	return c.account, c.accountErr
}
func (c *stubSchedulerCacheForSlotPool) SetAccount(_ context.Context, _ *Account) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) DeleteAccount(_ context.Context, _ int64) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) UpdateLastUsed(_ context.Context, _ map[int64]time.Time) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) TryLockBucket(_ context.Context, _ SchedulerBucket, _ time.Duration) (bool, error) {
	return true, nil
}
func (c *stubSchedulerCacheForSlotPool) UnlockBucket(_ context.Context, _ SchedulerBucket) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) ListBuckets(_ context.Context) ([]SchedulerBucket, error) {
	return c.buckets, c.bucketsErr
}
func (c *stubSchedulerCacheForSlotPool) GetOutboxWatermark(_ context.Context) (int64, error) {
	return 0, nil
}
func (c *stubSchedulerCacheForSlotPool) SetOutboxWatermark(_ context.Context, _ int64) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) SetBucketMembers(_ context.Context, _ SchedulerBucket, _ []int64) error {
	return nil
}
func (c *stubSchedulerCacheForSlotPool) RemoveAccountFromBuckets(_ context.Context, _ int64) error {
	return nil
}

type stubConcurrencyCacheForSlotPool struct {
	acquireResult  bool
	acquireErr     error
	concurrency    int
	concurrencyErr error
}

var _ ConcurrencyCache = (*stubConcurrencyCacheForSlotPool)(nil)

func (c *stubConcurrencyCacheForSlotPool) AcquireAccountSlot(_ context.Context, _ int64, _ int, _ string) (bool, error) {
	return c.acquireResult, c.acquireErr
}
func (c *stubConcurrencyCacheForSlotPool) ReleaseAccountSlot(_ context.Context, _ int64, _ string) error {
	return nil
}
func (c *stubConcurrencyCacheForSlotPool) GetAccountConcurrency(_ context.Context, _ int64) (int, error) {
	return c.concurrency, c.concurrencyErr
}
func (c *stubConcurrencyCacheForSlotPool) GetAccountConcurrencyBatch(_ context.Context, ids []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(ids))
	for _, id := range ids {
		result[id] = c.concurrency
	}
	return result, c.concurrencyErr
}
func (c *stubConcurrencyCacheForSlotPool) IncrementAccountWaitCount(_ context.Context, _ int64, _ int) (bool, error) {
	return true, nil
}
func (c *stubConcurrencyCacheForSlotPool) DecrementAccountWaitCount(_ context.Context, _ int64) error {
	return nil
}
func (c *stubConcurrencyCacheForSlotPool) GetAccountWaitingCount(_ context.Context, _ int64) (int, error) {
	return 0, nil
}
func (c *stubConcurrencyCacheForSlotPool) AcquireUserSlot(_ context.Context, _ int64, _ int, _ string) (bool, error) {
	return true, nil
}
func (c *stubConcurrencyCacheForSlotPool) ReleaseUserSlot(_ context.Context, _ int64, _ string) error {
	return nil
}
func (c *stubConcurrencyCacheForSlotPool) GetUserConcurrency(_ context.Context, _ int64) (int, error) {
	return 0, nil
}
func (c *stubConcurrencyCacheForSlotPool) IncrementWaitCount(_ context.Context, _ int64, _ int) (bool, error) {
	return true, nil
}
func (c *stubConcurrencyCacheForSlotPool) DecrementWaitCount(_ context.Context, _ int64) error {
	return nil
}
func (c *stubConcurrencyCacheForSlotPool) GetAccountsLoadBatch(_ context.Context, _ []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return nil, nil
}
func (c *stubConcurrencyCacheForSlotPool) GetUsersLoadBatch(_ context.Context, _ []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	return nil, nil
}
func (c *stubConcurrencyCacheForSlotPool) CleanupExpiredAccountSlots(_ context.Context, _ int64) error {
	return nil
}
func (c *stubConcurrencyCacheForSlotPool) CleanupExpiredAccountSlotKeys(_ context.Context) error {
	return nil
}
func (c *stubConcurrencyCacheForSlotPool) CleanupStaleProcessSlots(_ context.Context, _ string) error {
	return nil
}

// --- Helper functions ---

func newTestSlotPoolService(
	cache *stubSlotPoolCache,
	schedulerCache *stubSchedulerCacheForSlotPool,
	concurrencyCache *stubConcurrencyCacheForSlotPool,
	cfg *config.Config,
) *slotPoolService {
	return &slotPoolService{
		cache:            cache,
		schedulerCache:   schedulerCache,
		concurrencyCache: concurrencyCache,
		cfg:              cfg,
		stopCh:           make(chan struct{}),
	}
}

func testConfigEnabled() *config.Config {
	return &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				GatewaySchedulingConfigCustom: config.GatewaySchedulingConfigCustom{
					SlotPool: config.SlotPoolConfig{
						Enabled:    true,
						MaxRetries: 10,
					},
				},
			},
		},
	}
}

func testConfigSimpleMode() *config.Config {
	cfg := testConfigEnabled()
	cfg.RunMode = config.RunModeSimple
	return cfg
}

func testConfigDisabled() *config.Config {
	return &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				GatewaySchedulingConfigCustom: config.GatewaySchedulingConfigCustom{
					SlotPool: config.SlotPoolConfig{
						Enabled: false,
					},
				},
			},
		},
	}
}

func activeSchedulableAccount(id int64, platform string, concurrency int, groupIDs []int64) *Account {
	return &Account{
		ID:          id,
		Platform:    platform,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: concurrency,
		GroupIDs:    groupIDs,
		Priority:    1,
	}
}

// --- Tests ---

func TestSlotPoolService_IsEnabled(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())
		require.True(t, svc.IsEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigDisabled())
		require.False(t, svc.IsEnabled())
	})

	t.Run("nil_config", func(t *testing.T) {
		svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, nil)
		require.False(t, svc.IsEnabled())
	})
}

func TestSlotPoolService_AcquireFromPool_PoolEmpty(t *testing.T) {
	cache := &stubSlotPoolCache{popResults: []int64{}} // empty pool
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	candidates := map[int64]*Account{1: activeSchedulableAccount(1, PlatformAnthropic, 1, nil)}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.False(t, acquired)
	require.Zero(t, accountID)
}

func TestSlotPoolService_AcquireFromPool_PopError(t *testing.T) {
	cache := &stubSlotPoolCache{popErr: errors.New("redis error")}
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	candidates := map[int64]*Account{1: activeSchedulableAccount(1, PlatformAnthropic, 1, nil)}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	_, _, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis error")
}

func TestSlotPoolService_AcquireFromPool_AccountNotInCandidates(t *testing.T) {
	// Pool returns account 99, but candidates only has account 1
	cache := &stubSlotPoolCache{popResults: []int64{99}}
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	candidates := map[int64]*Account{1: activeSchedulableAccount(1, PlatformAnthropic, 1, nil)}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.False(t, acquired)
	require.Zero(t, accountID)
}

func TestSlotPoolService_AcquireFromPool_AcquireSuccess_WithRemainingSlots(t *testing.T) {
	cache := &stubSlotPoolCache{popResults: []int64{1}}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: true, concurrency: 0} // 0 current, has remaining
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAnthropic, 2, nil) // concurrency=2
	candidates := map[int64]*Account{1: account}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)

	// Should add back to pool since there are remaining slots
	require.Len(t, cache.addCalls, 1)
	require.Equal(t, int64(1), cache.addCalls[0].AccountID)
}

func TestSlotPoolService_AcquireFromPool_AcquireSuccess_NoRemainingSlots(t *testing.T) {
	cache := &stubSlotPoolCache{popResults: []int64{1}}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: true, concurrency: 1} // 1 current, 1 max = no remaining
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAnthropic, 1, nil) // concurrency=1
	candidates := map[int64]*Account{1: account}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)

	// Should NOT add back to pool since no remaining slots
	require.Empty(t, cache.addCalls)
}

func TestSlotPoolService_AcquireFromPool_AcquireFailure_RetryNextAccount(t *testing.T) {
	// First account fails acquire, second succeeds
	cache := &stubSlotPoolCache{popResults: []int64{1, 2}}
	acquireCount := 0
	concurrencyCache := &stubConcurrencyCacheForSlotPool{}
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, testConfigEnabled())

	// Override acquire behavior
	originalAcquire := concurrencyCache.acquireResult
	concurrencyCache.acquireResult = false // first will fail

	account1 := activeSchedulableAccount(1, PlatformAnthropic, 1, nil)
	account2 := activeSchedulableAccount(2, PlatformAnthropic, 1, nil)
	candidates := map[int64]*Account{1: account1, 2: account2}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	// Use a custom cache that tracks acquire attempts
	type trackingConcurrencyCache struct {
		stubConcurrencyCacheForSlotPool
		attempts []int64
	}
	trackCache := &trackingConcurrencyCache{}
	trackCache.acquireResult = false
	svc.concurrencyCache = trackCache

	// Override AcquireAccountSlot to succeed on second attempt
	svc.concurrencyCache = &dynamicConcurrencyCache{
		acquireFn: func(accountID int64) (bool, error) {
			acquireCount++
			return acquireCount == 2, nil // succeed on second attempt
		},
		concurrency: 0,
	}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(2), accountID)
	_ = originalAcquire
}

func TestSlotPoolService_AcquireFromPool_MaxRetriesExhausted(t *testing.T) {
	// All accounts fail acquire
	cache := &stubSlotPoolCache{popResults: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: false}
	cfg := testConfigEnabled()
	cfg.Gateway.Scheduling.SlotPool.MaxRetries = 3
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, cfg)

	candidates := make(map[int64]*Account)
	for i := int64(1); i <= 11; i++ {
		candidates[i] = activeSchedulableAccount(i, PlatformAnthropic, 1, nil)
	}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.False(t, acquired)
	require.Zero(t, accountID)
	// Should have popped exactly MaxRetries times
	require.Equal(t, 3, cache.popIndex)
}

func TestSlotPoolService_AcquireFromPool_Disabled(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigDisabled())

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.NoError(t, err)
	require.False(t, acquired)
	require.Zero(t, accountID)
}

func TestSlotPoolService_OnSlotReleased_AccountNotSchedulable(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
	}{
		{
			name: "inactive_status",
			account: &Account{
				ID:          1,
				Status:      StatusDisabled,
				Schedulable: true,
				Concurrency: 1,
			},
		},
		{
			name: "schedulable_false",
			account: &Account{
				ID:          1,
				Status:      StatusActive,
				Schedulable: false,
				Concurrency: 1,
			},
		},
		{
			name: "expired",
			account: func() *Account {
				past := time.Now().Add(-time.Hour)
				return &Account{
					ID:                 1,
					Status:             StatusActive,
					Schedulable:        true,
					Concurrency:        1,
					AutoPauseOnExpired: true,
					ExpiresAt:          &past,
				}
			}(),
		},
		{
			name: "overloaded",
			account: func() *Account {
				future := time.Now().Add(time.Hour)
				return &Account{
					ID:            1,
					Status:        StatusActive,
					Schedulable:   true,
					Concurrency:   1,
					OverloadUntil: &future,
				}
			}(),
		},
		{
			name: "rate_limited",
			account: func() *Account {
				future := time.Now().Add(time.Hour)
				return &Account{
					ID:               1,
					Status:           StatusActive,
					Schedulable:      true,
					Concurrency:      1,
					RateLimitResetAt: &future,
				}
			}(),
		},
		{
			name: "temp_unschedulable",
			account: func() *Account {
				future := time.Now().Add(time.Hour)
				return &Account{
					ID:                     1,
					Status:                 StatusActive,
					Schedulable:            true,
					Concurrency:            1,
					TempUnschedulableUntil: &future,
				}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &stubSlotPoolCache{}
			schedulerCache := &stubSchedulerCacheForSlotPool{account: tt.account}
			concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 0}
			svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

			err := svc.OnSlotReleased(context.Background(), 1)
			require.NoError(t, err)

			// Should NOT add to pool since account is not schedulable
			require.Empty(t, cache.addCalls)
		})
	}
}

func TestSlotPoolService_OnSlotReleased_Backfill(t *testing.T) {
	cache := &stubSlotPoolCache{}
	account := activeSchedulableAccount(1, PlatformAnthropic, 2, []int64{100})
	schedulerCache := &stubSchedulerCacheForSlotPool{account: account}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 1} // 1 < 2, has free slot
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	err := svc.OnSlotReleased(context.Background(), 1)
	require.NoError(t, err)

	// Should add to pool for all buckets
	require.NotEmpty(t, cache.addCalls)
	// Check that account was added to single and forced buckets
	var hasSingle, hasForced bool
	for _, call := range cache.addCalls {
		if call.Bucket.Mode == SchedulerModeSingle {
			hasSingle = true
		}
		if call.Bucket.Mode == SchedulerModeForced {
			hasForced = true
		}
	}
	require.True(t, hasSingle)
	require.True(t, hasForced)
}

func TestSlotPoolService_OnSlotReleased_UsesPerGroupPriority(t *testing.T) {
	cache := &stubSlotPoolCache{}
	account := activeSchedulableAccount(1, PlatformOpenAI, 2, []int64{100, 200})
	account.Priority = 99
	account.AccountGroups = []AccountGroup{
		{AccountID: account.ID, GroupID: 100, Priority: 8, SortOrder: 2, SchedulingConfigured: true},
		{AccountID: account.ID, GroupID: 200, Priority: 3, SortOrder: 7, SchedulingConfigured: true},
	}
	schedulerCache := &stubSchedulerCacheForSlotPool{account: account}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 0}
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	err := svc.OnSlotReleased(context.Background(), account.ID)
	require.NoError(t, err)
	require.Len(t, cache.addCalls, 4)
	for _, call := range cache.addCalls {
		switch call.Bucket.GroupID {
		case 100:
			require.Equal(t, 2, call.Priority)
		case 200:
			require.Equal(t, 7, call.Priority)
		default:
			t.Fatalf("unexpected group %d", call.Bucket.GroupID)
		}
	}
}

func TestSlotPoolService_OnSlotReleased_NoFreeSlots(t *testing.T) {
	cache := &stubSlotPoolCache{}
	account := activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100})
	schedulerCache := &stubSchedulerCacheForSlotPool{account: account}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 1} // 1 >= 1, no free slot
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	err := svc.OnSlotReleased(context.Background(), 1)
	require.NoError(t, err)

	// Should NOT add to pool since no free slots
	require.Empty(t, cache.addCalls)
}

func TestSlotPoolService_RebuildBucketPool_FiltersNonSchedulable(t *testing.T) {
	cache := &stubSlotPoolCache{}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 0}
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, testConfigEnabled())

	future := time.Now().Add(time.Hour)
	accounts := []*Account{
		activeSchedulableAccount(1, PlatformAnthropic, 1, nil),                                   // schedulable
		{ID: 2, Status: StatusDisabled, Schedulable: true, Concurrency: 1},                       // inactive
		{ID: 3, Status: StatusActive, Schedulable: false, Concurrency: 1},                        // not schedulable
		{ID: 4, Status: StatusActive, Schedulable: true, Concurrency: 1, OverloadUntil: &future}, // overloaded
		activeSchedulableAccount(5, PlatformAnthropic, 1, nil),                                   // schedulable
	}

	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}
	err := svc.RebuildBucketPool(context.Background(), bucket, accounts)
	require.NoError(t, err)

	require.Len(t, cache.rebuildCalls, 1)
	entries := cache.rebuildCalls[0].Entries
	require.Len(t, entries, 2) // only accounts 1 and 5
	require.Equal(t, int64(1), entries[0].AccountID)
	require.Equal(t, int64(5), entries[1].AccountID)
}

func TestSlotPoolService_RebuildBucketPool_UsesBucketGroupPriority(t *testing.T) {
	cache := &stubSlotPoolCache{}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 0}
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformOpenAI, 1, []int64{100, 200})
	account.Priority = 99
	account.AccountGroups = []AccountGroup{
		{AccountID: account.ID, GroupID: 100, Priority: 8, SortOrder: 2, SchedulingConfigured: true},
		{AccountID: account.ID, GroupID: 200, Priority: 3, SortOrder: 7, SchedulingConfigured: true},
	}
	bucket := SchedulerBucket{GroupID: 200, Platform: PlatformOpenAI, Mode: SchedulerModeSingle}

	err := svc.RebuildBucketPool(context.Background(), bucket, []*Account{account})
	require.NoError(t, err)
	require.Len(t, cache.rebuildCalls, 1)
	require.Len(t, cache.rebuildCalls[0].Entries, 1)
	require.Equal(t, 7, cache.rebuildCalls[0].Entries[0].Priority)
}

func TestSlotPoolService_getAccountBuckets_SinglePlatform(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformOpenAI, 1, []int64{100})
	buckets := svc.getAccountBuckets(account)

	// OpenAI should have single and forced buckets, no mixed
	require.Len(t, buckets, 2)
	var modes []string
	for _, b := range buckets {
		modes = append(modes, b.Mode)
		require.Equal(t, int64(100), b.GroupID)
		require.Equal(t, PlatformOpenAI, b.Platform)
	}
	require.Contains(t, modes, SchedulerModeSingle)
	require.Contains(t, modes, SchedulerModeForced)
}

func TestSlotPoolService_getAccountBuckets_MixedPlatform_Anthropic(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100})
	buckets := svc.getAccountBuckets(account)

	// Anthropic should have single, forced, and mixed buckets
	require.Len(t, buckets, 3)
	var modes []string
	for _, b := range buckets {
		modes = append(modes, b.Mode)
		require.Equal(t, int64(100), b.GroupID)
		require.Equal(t, PlatformAnthropic, b.Platform)
	}
	require.Contains(t, modes, SchedulerModeSingle)
	require.Contains(t, modes, SchedulerModeForced)
	require.Contains(t, modes, SchedulerModeMixed)
}

func TestSlotPoolService_getAccountBuckets_MixedPlatform_Gemini(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformGemini, 1, []int64{100})
	buckets := svc.getAccountBuckets(account)

	// Gemini should have single, forced, and mixed buckets
	require.Len(t, buckets, 3)
	var modes []string
	for _, b := range buckets {
		modes = append(modes, b.Mode)
	}
	require.Contains(t, modes, SchedulerModeSingle)
	require.Contains(t, modes, SchedulerModeForced)
	require.Contains(t, modes, SchedulerModeMixed)
}

func TestSlotPoolService_getAccountBuckets_Antigravity_WithMixedScheduling(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAntigravity, 1, []int64{100})
	account.Extra = map[string]any{"mixed_scheduling": true}
	buckets := svc.getAccountBuckets(account)

	// Antigravity with mixed_scheduling should have:
	// - single (antigravity)
	// - forced (antigravity)
	// - mixed (anthropic)
	// - mixed (gemini)
	require.Len(t, buckets, 4)

	var antigravitySingle, antigravityForced, anthropicMixed, geminiMixed bool
	for _, b := range buckets {
		if b.Platform == PlatformAntigravity && b.Mode == SchedulerModeSingle {
			antigravitySingle = true
		}
		if b.Platform == PlatformAntigravity && b.Mode == SchedulerModeForced {
			antigravityForced = true
		}
		if b.Platform == PlatformAnthropic && b.Mode == SchedulerModeMixed {
			anthropicMixed = true
		}
		if b.Platform == PlatformGemini && b.Mode == SchedulerModeMixed {
			geminiMixed = true
		}
	}
	require.True(t, antigravitySingle)
	require.True(t, antigravityForced)
	require.True(t, anthropicMixed)
	require.True(t, geminiMixed)
}

func TestSlotPoolService_getAccountBuckets_Antigravity_WithoutMixedScheduling(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAntigravity, 1, []int64{100})
	// No mixed_scheduling setting
	buckets := svc.getAccountBuckets(account)

	// Should only have single and forced for antigravity
	require.Len(t, buckets, 2)
	for _, b := range buckets {
		require.Equal(t, PlatformAntigravity, b.Platform)
	}
}

func TestSlotPoolService_getAccountBuckets_Ungrouped(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAnthropic, 1, nil) // empty GroupIDs
	buckets := svc.getAccountBuckets(account)

	// All buckets should have groupID=0
	for _, b := range buckets {
		require.Equal(t, int64(0), b.GroupID)
	}
}

func TestSlotPoolService_getAccountBuckets_MultipleGroups(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	account := activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100, 200})
	buckets := svc.getAccountBuckets(account)

	// Should have buckets for both groups: (single, forced, mixed) * 2 = 6
	require.Len(t, buckets, 6)

	groupIDs := make(map[int64]int)
	for _, b := range buckets {
		groupIDs[b.GroupID]++
	}
	require.Equal(t, 3, groupIDs[100])
	require.Equal(t, 3, groupIDs[200])
}

func TestSlotPoolService_normalizeGroupIDs_SimpleMode(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigSimpleMode())

	// Any group IDs should be normalized to [0]
	result := svc.normalizeGroupIDs([]int64{100, 200, 300})
	require.Equal(t, []int64{0}, result)

	// Empty should also return [0]
	result = svc.normalizeGroupIDs(nil)
	require.Equal(t, []int64{0}, result)
}

func TestSlotPoolService_normalizeGroupIDs_NormalMode(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	// Should preserve group IDs
	result := svc.normalizeGroupIDs([]int64{100, 200})
	require.Equal(t, []int64{100, 200}, result)

	// Empty should return [0]
	result = svc.normalizeGroupIDs(nil)
	require.Equal(t, []int64{0}, result)
}

func TestSlotPoolService_getAccountBuckets_SimpleMode(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigSimpleMode())

	account := activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100, 200})
	buckets := svc.getAccountBuckets(account)

	// In simple mode, all buckets should have groupID=0
	for _, b := range buckets {
		require.Equal(t, int64(0), b.GroupID, "simple mode should normalize all groupIDs to 0")
	}
	// Should only have 3 buckets (not 6) since groups are deduplicated to [0]
	require.Len(t, buckets, 3)
}

func TestSlotPoolService_AcquireFromPool_NotInCandidates_DeferredPutBack(t *testing.T) {
	// Test that when popped account is not in candidates:
	// 1. It's tracked to avoid re-popping in same loop
	// 2. It's put back AFTER the loop completes (deferred)
	// 3. Capacity and bucket membership are checked before putting back

	// Account 99 belongs to group 100 (matches bucket)
	cachedAccount := activeSchedulableAccount(99, PlatformAnthropic, 2, []int64{100})
	cache := &stubSlotPoolCache{popResults: []int64{99, 1}} // First returns 99 (not in candidates), then 1
	schedulerCache := &stubSchedulerCacheForSlotPool{account: cachedAccount}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: true, concurrency: 0} // Has capacity
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	// Only account 1 is in candidates, not 99
	// Account 99 might be filtered by model, but still belongs to the bucket
	account1 := activeSchedulableAccount(1, PlatformAnthropic, 1, nil)
	candidates := map[int64]*Account{1: account1}
	// Bucket matches account 99's group
	bucket := SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)

	// Account 99 should have been put back into the pool (deferred)
	// because it belongs to the bucket and has capacity
	var foundPutBack bool
	for _, call := range cache.addCalls {
		if call.AccountID == 99 {
			foundPutBack = true
			break
		}
	}
	require.True(t, foundPutBack, "account 99 should be put back into pool after loop completes")
}

func TestSlotPoolService_AcquireFromPool_NotInCandidates_NoCapacity_NotPutBack(t *testing.T) {
	// Test that non-candidate accounts are NOT put back if they're at capacity

	cachedAccount := activeSchedulableAccount(99, PlatformAnthropic, 1, []int64{100})
	cache := &stubSlotPoolCache{popResults: []int64{99, 1}}
	schedulerCache := &stubSchedulerCacheForSlotPool{account: cachedAccount}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: true, concurrency: 1} // At capacity (1 >= 1)
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	account1 := activeSchedulableAccount(1, PlatformAnthropic, 1, nil)
	candidates := map[int64]*Account{1: account1}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)

	// Account 99 should NOT have been put back since it's at capacity
	for _, call := range cache.addCalls {
		require.NotEqual(t, int64(99), call.AccountID, "account 99 should not be put back when at capacity")
	}
}

func TestSlotPoolService_AcquireFromPool_RepeatedNonCandidate_ConsumesBudget(t *testing.T) {
	// Test that repeated pops of the same non-candidate account DO consume retry budget
	// This prevents livelock from concurrent reinserts

	cachedAccount := activeSchedulableAccount(99, PlatformAnthropic, 2, []int64{0})

	// Simulate ZSET that keeps returning account 99 repeatedly (simulating concurrent reinserts)
	// Each pop should consume retry budget to prevent infinite loop
	popResults := []int64{99, 99, 99, 99, 99, 99} // More than MaxRetries
	cache := &stubSlotPoolCache{popResults: popResults}
	schedulerCache := &stubSchedulerCacheForSlotPool{account: cachedAccount}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: true, concurrency: 0}

	cfg := testConfigEnabled()
	cfg.Gateway.Scheduling.SlotPool.MaxRetries = 3 // Only 3 retries
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, cfg)

	// No valid candidates, only account 99 which isn't in candidates
	candidates := map[int64]*Account{}
	bucket := SchedulerBucket{GroupID: 0, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	// Should exhaust retries and return gracefully (not infinite loop)
	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.False(t, acquired, "should exhaust retries and return false")
	require.Zero(t, accountID)

	// Should have consumed exactly MaxRetries pops (first pop + 2 more retries = 3 iterations)
	// But skipped pops still count, so we pop exactly MaxRetries times
	require.Equal(t, 3, cache.popIndex, "should consume exactly MaxRetries pops")
}

func TestSlotPoolService_AcquireFromPool_NotInCandidates_WrongBucket_NotPutBack(t *testing.T) {
	// Test that non-candidate accounts that no longer belong to the bucket are NOT put back
	// (e.g., group/platform/mode drift)

	// Account belongs to group 200, but we're querying bucket for group 100
	cachedAccount := activeSchedulableAccount(99, PlatformAnthropic, 2, []int64{200})
	cache := &stubSlotPoolCache{popResults: []int64{99, 1}}
	schedulerCache := &stubSchedulerCacheForSlotPool{account: cachedAccount}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{acquireResult: true, concurrency: 0}
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	account1 := activeSchedulableAccount(1, PlatformAnthropic, 1, nil)
	candidates := map[int64]*Account{1: account1}
	// Bucket for group 100 - account 99 belongs to group 200, so it shouldn't be put back
	bucket := SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	accountID, acquired, err := svc.AcquireFromPool(context.Background(), bucket, candidates, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)

	// Account 99 should NOT be put back since it doesn't belong to bucket group 100
	for _, call := range cache.addCalls {
		require.NotEqual(t, int64(99), call.AccountID,
			"account 99 should not be put back - it belongs to group 200, not bucket's group 100")
	}
}

func TestSlotPoolService_accountBelongsToBucket(t *testing.T) {
	svc := newTestSlotPoolService(&stubSlotPoolCache{}, &stubSchedulerCacheForSlotPool{}, &stubConcurrencyCacheForSlotPool{}, testConfigEnabled())

	tests := []struct {
		name     string
		account  *Account
		bucket   SchedulerBucket
		expected bool
	}{
		{
			name:     "matches_single_bucket",
			account:  activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100}),
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle},
			expected: true,
		},
		{
			name:     "matches_forced_bucket",
			account:  activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100}),
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeForced},
			expected: true,
		},
		{
			name:     "matches_mixed_bucket",
			account:  activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{100}),
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeMixed},
			expected: true,
		},
		{
			name:     "wrong_group",
			account:  activeSchedulableAccount(1, PlatformAnthropic, 1, []int64{200}),
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle},
			expected: false,
		},
		{
			name:     "wrong_platform",
			account:  activeSchedulableAccount(1, PlatformGemini, 1, []int64{100}),
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle},
			expected: false,
		},
		{
			name:     "openai_no_mixed",
			account:  activeSchedulableAccount(1, PlatformOpenAI, 1, []int64{100}),
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformOpenAI, Mode: SchedulerModeMixed},
			expected: false, // OpenAI doesn't have mixed bucket
		},
		{
			name:     "nil_account",
			account:  nil,
			bucket:   SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.accountBelongsToBucket(tt.account, tt.bucket)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSlotPoolService_UnlimitedConcurrency_AlwaysInPool(t *testing.T) {
	// Test that accounts with unlimited concurrency (Concurrency <= 0) are always considered to have available slots

	cache := &stubSlotPoolCache{}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 100} // High load, but shouldn't matter
	svc := newTestSlotPoolService(cache, &stubSchedulerCacheForSlotPool{}, concurrencyCache, testConfigEnabled())

	// Account with Concurrency=0 (unlimited)
	unlimitedAccount := activeSchedulableAccount(1, PlatformAnthropic, 0, []int64{100})
	unlimitedAccount.Concurrency = 0 // Explicitly set to 0 for unlimited

	accounts := []*Account{unlimitedAccount}
	bucket := SchedulerBucket{GroupID: 100, Platform: PlatformAnthropic, Mode: SchedulerModeSingle}

	err := svc.RebuildBucketPool(context.Background(), bucket, accounts)
	require.NoError(t, err)

	// Unlimited account should be in the pool despite high currentLoad
	require.Len(t, cache.rebuildCalls, 1)
	require.Len(t, cache.rebuildCalls[0].Entries, 1, "unlimited concurrency account should be included in pool")
	require.Equal(t, int64(1), cache.rebuildCalls[0].Entries[0].AccountID)
}

func TestSlotPoolService_OnSlotReleased_UnlimitedConcurrency_Backfills(t *testing.T) {
	// Test that unlimited concurrency accounts are always backfilled on release

	cache := &stubSlotPoolCache{}
	account := activeSchedulableAccount(1, PlatformAnthropic, 0, []int64{100}) // Concurrency=0 (unlimited)
	schedulerCache := &stubSchedulerCacheForSlotPool{account: account}
	concurrencyCache := &stubConcurrencyCacheForSlotPool{concurrency: 999} // Very high load, shouldn't matter
	svc := newTestSlotPoolService(cache, schedulerCache, concurrencyCache, testConfigEnabled())

	err := svc.OnSlotReleased(context.Background(), 1)
	require.NoError(t, err)

	// Should still add to pool since unlimited concurrency always has capacity
	require.NotEmpty(t, cache.addCalls, "unlimited concurrency account should be backfilled")
}

func TestHasAvailableSlot(t *testing.T) {
	tests := []struct {
		name           string
		currentLoad    int
		maxConcurrency int
		expected       bool
	}{
		{"unlimited_zero", 100, 0, true},
		{"unlimited_negative", 100, -1, true},
		{"has_capacity", 5, 10, true},
		{"at_capacity", 10, 10, false},
		{"over_capacity", 15, 10, false},
		{"zero_load_limited", 0, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAvailableSlot(tt.currentLoad, tt.maxConcurrency)
			require.Equal(t, tt.expected, result)
		})
	}
}

// --- Dynamic stub for complex acquire scenarios ---

type dynamicConcurrencyCache struct {
	stubConcurrencyCacheForSlotPool
	acquireFn   func(accountID int64) (bool, error)
	concurrency int
}

func (c *dynamicConcurrencyCache) AcquireAccountSlot(_ context.Context, accountID int64, _ int, _ string) (bool, error) {
	if c.acquireFn != nil {
		return c.acquireFn(accountID)
	}
	return c.acquireResult, c.acquireErr
}

func (c *dynamicConcurrencyCache) GetAccountConcurrency(_ context.Context, _ int64) (int, error) {
	return c.concurrency, nil
}
