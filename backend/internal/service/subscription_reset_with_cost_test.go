//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// resetWithCostStub 支持 GetByID、ShortenExpiryAndResetDaily；其余方法继承 userSubRepoNoop。
type resetWithCostStub struct {
	userSubRepoNoop

	sub *UserSubscription

	shortenCalled              bool
	shortenSubID               int64
	shortenOriginalWindowStart time.Time
	shortenNewExpiresAt        time.Time
	shortenNow                 time.Time
	shortenReturnUpdated       bool
	shortenReturnErr           error
}

func (r *resetWithCostStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *resetWithCostStub) ShortenExpiryAndResetDaily(
	_ context.Context,
	subID int64,
	originalWindowStart time.Time,
	newExpiresAt time.Time,
	now time.Time,
) (bool, error) {
	r.shortenCalled = true
	r.shortenSubID = subID
	r.shortenOriginalWindowStart = originalWindowStart
	r.shortenNewExpiresAt = newExpiresAt
	r.shortenNow = now
	if r.shortenReturnErr != nil {
		return false, r.shortenReturnErr
	}
	if r.shortenReturnUpdated && r.sub != nil {
		r.sub.ExpiresAt = newExpiresAt
		r.sub.DailyUsageUSD = 0
		r.sub.DailyWindowStart = &now
	}
	return r.shortenReturnUpdated, nil
}

func newResetWithCostSvc(stub *resetWithCostStub) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, nil, nil, nil)
}

func TestResetSubscriptionWithCost_Success(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-10 * time.Hour)
	futureExpiry := now.Add(30 * 24 * time.Hour)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               1,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        futureExpiry,
			DailyUsageUSD:    50.0,
			DailyWindowStart: &windowStart,
		},
		shortenReturnUpdated: true,
	}
	svc := newResetWithCostSvc(stub)

	result, err := svc.ResetSubscriptionWithCost(context.Background(), 1, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.shortenCalled)
	require.Equal(t, int64(1), stub.shortenSubID)
	require.WithinDuration(t, windowStart, stub.shortenOriginalWindowStart, time.Second)
	// cost ≈ 14h (24h - 10h since window start)
	expectedNewExpiry := futureExpiry.Add(-14 * time.Hour)
	require.WithinDuration(t, expectedNewExpiry, stub.shortenNewExpiresAt, 2*time.Second)
	require.Equal(t, float64(0), result.DailyUsageUSD)
}

func TestResetSubscriptionWithCost_AdminCall_SkipsOwnershipCheck(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-10 * time.Hour)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               2,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        now.Add(5 * 24 * time.Hour),
			DailyWindowStart: &windowStart,
		},
		shortenReturnUpdated: true,
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 2, 0)

	require.NoError(t, err)
	require.True(t, stub.shortenCalled, "admin 调用应继续执行")
}

func TestResetSubscriptionWithCost_OwnershipMismatch(t *testing.T) {
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:        3,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(5 * 24 * time.Hour),
		},
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 3, 999)

	require.ErrorIs(t, err, ErrSubscriptionNotOwned)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_NotFound(t *testing.T) {
	stub := &resetWithCostStub{sub: nil}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 999, 10)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_InactiveStatus(t *testing.T) {
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:        4,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusExpired,
			ExpiresAt: time.Now().Add(5 * 24 * time.Hour),
		},
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 4, 10)

	require.ErrorIs(t, err, ErrSubscriptionInactive)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_RemainingExactly24h(t *testing.T) {
	// 剩余正好 24h → 应拒绝（边界严格 > 24h）
	now := time.Now()
	windowStart := now.Add(-10 * time.Hour)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               5,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        now.Add(24 * time.Hour),
			DailyWindowStart: &windowStart,
		},
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 5, 10)

	require.ErrorIs(t, err, ErrSubscriptionTimeInsufficient)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_RemainingBarelyOver24h(t *testing.T) {
	// 剩余 24h + 1s → 应成功（严格 > 24h），cost ≈ 14h
	now := time.Now()
	windowStart := now.Add(-10 * time.Hour)
	expiry := now.Add(24*time.Hour + time.Second)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               6,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        expiry,
			DailyWindowStart: &windowStart,
		},
		shortenReturnUpdated: true,
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 6, 10)

	require.NoError(t, err)
	require.True(t, stub.shortenCalled)
	// cost ≈ 14h，newExpiresAt ≈ expiry - 14h
	expectedNewExpiry := expiry.Add(-14 * time.Hour)
	require.WithinDuration(t, expectedNewExpiry, stub.shortenNewExpiresAt, 2*time.Second)
}

func TestResetSubscriptionWithCost_DailyWindowNil(t *testing.T) {
	// DailyWindowStart 为 nil → 不可重置
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               8,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        time.Now().Add(5 * 24 * time.Hour),
			DailyWindowStart: nil,
		},
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 8, 10)

	require.ErrorIs(t, err, ErrSubscriptionTimeInsufficient)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_DailyWindowExpired(t *testing.T) {
	// DailyWindowStart 已过期（> 24h 之前）→ 自然滚动会处理，拒绝手动重置
	now := time.Now()
	windowStart := now.Add(-25 * time.Hour)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               9,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        now.Add(5 * 24 * time.Hour),
			DailyWindowStart: &windowStart,
		},
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 9, 10)

	require.ErrorIs(t, err, ErrSubscriptionTimeInsufficient)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_ConcurrentLoses(t *testing.T) {
	// 预检通过，但原子 UPDATE 返回 0 行（并发竞争失败者）
	now := time.Now()
	windowStart := now.Add(-10 * time.Hour)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:               7,
			UserID:           10,
			GroupID:          20,
			Status:           SubscriptionStatusActive,
			ExpiresAt:        now.Add(5 * 24 * time.Hour),
			DailyWindowStart: &windowStart,
		},
		shortenReturnUpdated: false, // 模拟 0 行影响
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 7, 10)

	require.ErrorIs(t, err, ErrSubscriptionTimeInsufficient)
	require.True(t, stub.shortenCalled)
}
