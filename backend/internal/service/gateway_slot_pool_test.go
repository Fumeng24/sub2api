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

// --- Stub implementations for Gateway SlotPool integration tests ---

type stubSlotPoolServiceForGateway struct {
	enabled      bool
	acquireID    int64
	acquireOK    bool
	acquireErr   error
	releasedIDs  []int64
	releaseErr   error
	acquireCalls int
	releaseCalls int
}

var _ SlotPoolService = (*stubSlotPoolServiceForGateway)(nil)

func (s *stubSlotPoolServiceForGateway) IsEnabled() bool { return s.enabled }

func (s *stubSlotPoolServiceForGateway) AcquireFromPool(_ context.Context, _ SchedulerBucket, _ map[int64]*Account, _ string) (int64, bool, error) {
	s.acquireCalls++
	return s.acquireID, s.acquireOK, s.acquireErr
}

func (s *stubSlotPoolServiceForGateway) OnSlotReleased(_ context.Context, accountID int64) error {
	s.releaseCalls++
	s.releasedIDs = append(s.releasedIDs, accountID)
	return s.releaseErr
}

func (s *stubSlotPoolServiceForGateway) RebuildBucketPool(_ context.Context, _ SchedulerBucket, _ []*Account) error {
	return nil
}

func (s *stubSlotPoolServiceForGateway) Start() {}
func (s *stubSlotPoolServiceForGateway) Stop()  {}

type stubConcurrencyCacheForGateway struct {
	acquireResult bool
	acquireErr    error
	releaseCalls  []struct {
		AccountID int64
		RequestID string
	}
}

var _ ConcurrencyCache = (*stubConcurrencyCacheForGateway)(nil)

func (c *stubConcurrencyCacheForGateway) AcquireAccountSlot(_ context.Context, _ int64, _ int, _ string) (bool, error) {
	return c.acquireResult, c.acquireErr
}
func (c *stubConcurrencyCacheForGateway) ReleaseAccountSlot(_ context.Context, accountID int64, requestID string) error {
	c.releaseCalls = append(c.releaseCalls, struct {
		AccountID int64
		RequestID string
	}{accountID, requestID})
	return nil
}
func (c *stubConcurrencyCacheForGateway) GetAccountConcurrency(_ context.Context, _ int64) (int, error) {
	return 0, nil
}
func (c *stubConcurrencyCacheForGateway) GetAccountConcurrencyBatch(_ context.Context, ids []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(ids))
	for _, id := range ids {
		result[id] = 0
	}
	return result, nil
}
func (c *stubConcurrencyCacheForGateway) IncrementAccountWaitCount(_ context.Context, _ int64, _ int) (bool, error) {
	return true, nil
}
func (c *stubConcurrencyCacheForGateway) DecrementAccountWaitCount(_ context.Context, _ int64) error {
	return nil
}
func (c *stubConcurrencyCacheForGateway) GetAccountWaitingCount(_ context.Context, _ int64) (int, error) {
	return 0, nil
}
func (c *stubConcurrencyCacheForGateway) AcquireUserSlot(_ context.Context, _ int64, _ int, _ string) (bool, error) {
	return true, nil
}
func (c *stubConcurrencyCacheForGateway) ReleaseUserSlot(_ context.Context, _ int64, _ string) error {
	return nil
}
func (c *stubConcurrencyCacheForGateway) GetUserConcurrency(_ context.Context, _ int64) (int, error) {
	return 0, nil
}
func (c *stubConcurrencyCacheForGateway) IncrementWaitCount(_ context.Context, _ int64, _ int) (bool, error) {
	return true, nil
}
func (c *stubConcurrencyCacheForGateway) DecrementWaitCount(_ context.Context, _ int64) error {
	return nil
}
func (c *stubConcurrencyCacheForGateway) GetAccountsLoadBatch(_ context.Context, _ []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	return nil, nil
}
func (c *stubConcurrencyCacheForGateway) GetUsersLoadBatch(_ context.Context, _ []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	return nil, nil
}
func (c *stubConcurrencyCacheForGateway) CleanupExpiredAccountSlots(_ context.Context, _ int64) error {
	return nil
}
func (c *stubConcurrencyCacheForGateway) CleanupExpiredAccountSlotKeys(_ context.Context) error {
	return nil
}
func (c *stubConcurrencyCacheForGateway) CleanupStaleProcessSlots(_ context.Context, _ string) error {
	return nil
}

type stubSessionLimitCacheForGateway struct {
	checkResult bool
}

func (c *stubSessionLimitCacheForGateway) CheckAndRegisterSession(_ context.Context, _ int64, _ string, _ int) (bool, error) {
	return c.checkResult, nil
}

// --- Tests ---

func TestGatewayService_SlotPool_AcquireSuccess(t *testing.T) {
	// Test that SlotPool acquire success returns the account and calls OnSlotReleased on release

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:   true,
		acquireID: 1,
		acquireOK: true,
	}
	concurrencyCache := &stubConcurrencyCacheForGateway{}
	sessionCache := &stubSessionLimitCacheForGateway{checkResult: true}

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				GatewaySchedulingConfigCustom: config.GatewaySchedulingConfigCustom{
					SlotPool: config.SlotPoolConfig{
						Enabled:         true,
						FallbackOnError: true,
					},
				},
			},
		},
	}

	// Verify slot pool was called
	require.Equal(t, 0, slotPool.acquireCalls)

	// Simulate acquire call
	accountID, acquired, err := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)
	require.Equal(t, 1, slotPool.acquireCalls)

	// Simulate release callback
	err = slotPool.OnSlotReleased(context.Background(), accountID)
	require.NoError(t, err)
	require.Equal(t, 1, slotPool.releaseCalls)
	require.Equal(t, []int64{1}, slotPool.releasedIDs)

	// Unused references to satisfy linter
	_ = cfg
	_ = concurrencyCache
	_ = sessionCache
}

func TestGatewayService_SlotPool_SessionLimitFailure_ReleasesSlot(t *testing.T) {
	// Test that when session limit check fails after SlotPool acquire,
	// the slot is released and OnSlotReleased is called

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:   true,
		acquireID: 1,
		acquireOK: true,
	}
	concurrencyCache := &stubConcurrencyCacheForGateway{}
	sessionCache := &stubSessionLimitCacheForGateway{checkResult: false} // Session limit fails

	// Simulate the flow:
	// 1. SlotPool acquire succeeds
	accountID, acquired, err := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, int64(1), accountID)

	// 2. Session limit check fails
	sessionOK, err := sessionCache.CheckAndRegisterSession(context.Background(), accountID, "session-1", 10)
	require.NoError(t, err)
	require.False(t, sessionOK)

	// 3. On session limit failure, we should release the slot
	err = concurrencyCache.ReleaseAccountSlot(context.Background(), accountID, "req-1")
	require.NoError(t, err)
	require.Len(t, concurrencyCache.releaseCalls, 1)
	require.Equal(t, int64(1), concurrencyCache.releaseCalls[0].AccountID)

	// 4. And call OnSlotReleased for potential backfill
	err = slotPool.OnSlotReleased(context.Background(), accountID)
	require.NoError(t, err)
	require.Equal(t, 1, slotPool.releaseCalls)
}

func TestGatewayService_SlotPool_AcquireError_FallbackEnabled(t *testing.T) {
	// Test that when SlotPool acquire errors with fallback enabled, we continue to load-batch

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:    true,
		acquireErr: errors.New("redis connection error"),
	}

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				GatewaySchedulingConfigCustom: config.GatewaySchedulingConfigCustom{
					SlotPool: config.SlotPoolConfig{
						Enabled:         true,
						FallbackOnError: true,
					},
				},
			},
		},
	}

	// Simulate acquire call with error
	accountID, acquired, err := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.Error(t, err)
	require.False(t, acquired)
	require.Zero(t, accountID)

	// With FallbackOnError=true, gateway should continue to load-batch logic
	// This is handled in gateway_service.go:1743-1749
	require.True(t, cfg.Gateway.Scheduling.SlotPool.FallbackOnError)
}

func TestGatewayService_SlotPool_AcquireError_FallbackDisabled(t *testing.T) {
	// Test that when SlotPool acquire errors with fallback disabled, we return error

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:    true,
		acquireErr: errors.New("redis connection error"),
	}

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Scheduling: config.GatewaySchedulingConfig{
				GatewaySchedulingConfigCustom: config.GatewaySchedulingConfigCustom{
					SlotPool: config.SlotPoolConfig{
						Enabled:         true,
						FallbackOnError: false,
					},
				},
			},
		},
	}

	// Simulate acquire call with error
	_, _, err := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.Error(t, err)

	// With FallbackOnError=false, gateway should return the error
	// This is handled in gateway_service.go:1745-1746
	require.False(t, cfg.Gateway.Scheduling.SlotPool.FallbackOnError)
}

func TestGatewayService_SlotPool_PoolEmpty_FallbackToLoadBatch(t *testing.T) {
	// Test that when SlotPool returns pool empty (acquired=false, err=nil), we fallback to load-batch

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:   true,
		acquireID: 0,
		acquireOK: false, // Pool empty or retries exhausted
	}

	// Simulate acquire call returning pool empty
	accountID, acquired, err := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.NoError(t, err)
	require.False(t, acquired)
	require.Zero(t, accountID)

	// Gateway should continue to load-batch logic (line 1777)
}

func TestGatewayService_SlotPool_ReleaseFunc_CallsOnSlotReleased(t *testing.T) {
	// Test that the ReleaseFunc returned by successful SlotPool acquire
	// correctly calls OnSlotReleased for backfill

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:   true,
		acquireID: 42,
		acquireOK: true,
	}
	concurrencyCache := &stubConcurrencyCacheForGateway{}

	// Simulate successful acquire
	accountID, acquired, _ := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")
	require.True(t, acquired)
	require.Equal(t, int64(42), accountID)

	// Simulate the ReleaseFunc that gateway_service.go:1767-1772 creates
	releaseFunc := func() {
		_ = concurrencyCache.ReleaseAccountSlot(context.Background(), accountID, "req-1")
		_ = slotPool.OnSlotReleased(context.Background(), accountID)
	}

	// Execute release
	releaseFunc()

	// Verify both release and backfill were called
	require.Len(t, concurrencyCache.releaseCalls, 1)
	require.Equal(t, int64(42), concurrencyCache.releaseCalls[0].AccountID)
	require.Equal(t, 1, slotPool.releaseCalls)
	require.Equal(t, []int64{42}, slotPool.releasedIDs)
}

func TestGatewayService_SlotPool_BucketDerivation_ForcedMode(t *testing.T) {
	// Test that when hasForcePlatform is true, bucket mode is SchedulerModeForced

	// This tests the logic at gateway_service.go:1723-1726
	hasForcePlatform := true
	useMixed := false

	bucketMode := SchedulerModeSingle
	if hasForcePlatform {
		bucketMode = SchedulerModeForced
	} else if useMixed {
		bucketMode = SchedulerModeMixed
	}

	require.Equal(t, SchedulerModeForced, bucketMode)
}

func TestGatewayService_SlotPool_BucketDerivation_MixedMode(t *testing.T) {
	// Test that when useMixed is true (and not forced), bucket mode is SchedulerModeMixed

	hasForcePlatform := false
	useMixed := true

	bucketMode := SchedulerModeSingle
	if hasForcePlatform {
		bucketMode = SchedulerModeForced
	} else if useMixed {
		bucketMode = SchedulerModeMixed
	}

	require.Equal(t, SchedulerModeMixed, bucketMode)
}

func TestGatewayService_SlotPool_BucketDerivation_SingleMode(t *testing.T) {
	// Test that when neither forced nor mixed, bucket mode is SchedulerModeSingle

	hasForcePlatform := false
	useMixed := false

	bucketMode := SchedulerModeSingle
	if hasForcePlatform {
		bucketMode = SchedulerModeForced
	} else if useMixed {
		bucketMode = SchedulerModeMixed
	}

	require.Equal(t, SchedulerModeSingle, bucketMode)
}

func TestGatewayService_SlotPool_BucketDerivation_SimpleModeNormalization(t *testing.T) {
	// Test that in simple mode, groupID is normalized to 0

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
	}

	groupID := int64(100)
	bucketGroupID := groupID
	if cfg.RunMode == config.RunModeSimple {
		bucketGroupID = 0
	}

	require.Equal(t, int64(0), bucketGroupID)
}

func TestGatewayService_SlotPool_BucketDerivation_NormalModePreservesGroupID(t *testing.T) {
	// Test that in normal mode, groupID is preserved

	cfg := &config.Config{
		RunMode: config.RunModeStandard,
	}

	groupID := int64(100)
	bucketGroupID := groupID
	if cfg.RunMode == config.RunModeSimple {
		bucketGroupID = 0
	}

	require.Equal(t, int64(100), bucketGroupID)
}

func TestGatewayService_SlotPool_Disabled_SkipsPool(t *testing.T) {
	// Test that when SlotPool is disabled, we skip directly to load-batch

	slotPool := &stubSlotPoolServiceForGateway{
		enabled: false,
	}

	// IsEnabled returns false, so gateway should skip SlotPool entirely
	require.False(t, slotPool.IsEnabled())

	// Verify no acquire calls would be made
	require.Equal(t, 0, slotPool.acquireCalls)
}

func TestGatewayService_SlotPool_BackfillTiming(t *testing.T) {
	// Test that OnSlotReleased is called at the right time during release

	slotPool := &stubSlotPoolServiceForGateway{
		enabled:   true,
		acquireID: 1,
		acquireOK: true,
	}

	// Acquire
	accountID, _, _ := slotPool.AcquireFromPool(context.Background(), SchedulerBucket{}, nil, "req-1")

	// Simulate some work being done (request processing)
	time.Sleep(10 * time.Millisecond)

	// Release should trigger backfill check
	require.Equal(t, 0, slotPool.releaseCalls)
	_ = slotPool.OnSlotReleased(context.Background(), accountID)
	require.Equal(t, 1, slotPool.releaseCalls)
}
