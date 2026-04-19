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

	shortenCalled        bool
	shortenSubID         int64
	shortenNewExpiresAt  time.Time
	shortenWindowStart   time.Time
	shortenMinRemaining  time.Duration
	shortenReturnUpdated bool
	shortenReturnErr     error
}

func (r *resetWithCostStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *resetWithCostStub) ShortenExpiryAndResetDaily(_ context.Context, _ int64, _ time.Time, _ time.Time, _ time.Time) (bool, error) {
	// TODO: Hotfix 2 will rewrite test assertions
	return false, nil
}

func newResetWithCostSvc(stub *resetWithCostStub) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, nil, nil, nil)
}

func TestResetSubscriptionWithCost_Success(t *testing.T) {
	futureExpiry := time.Now().Add(30 * 24 * time.Hour)
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:            1,
			UserID:        10,
			GroupID:       20,
			Status:        SubscriptionStatusActive,
			ExpiresAt:     futureExpiry,
			DailyUsageUSD: 50.0,
		},
		shortenReturnUpdated: true,
	}
	svc := newResetWithCostSvc(stub)

	result, err := svc.ResetSubscriptionWithCost(context.Background(), 1, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.shortenCalled)
	require.Equal(t, int64(1), stub.shortenSubID)
	require.WithinDuration(t, futureExpiry.AddDate(0, 0, -1), stub.shortenNewExpiresAt, time.Second)
	require.Equal(t, 24*time.Hour, stub.shortenMinRemaining)
	require.Equal(t, float64(0), result.DailyUsageUSD)
}

func TestResetSubscriptionWithCost_AdminCall_SkipsOwnershipCheck(t *testing.T) {
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:        2,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(5 * 24 * time.Hour),
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

func TestResetSubscriptionWithCost_TimeInsufficientBoundary_ExactOneDay(t *testing.T) {
	// 剩余正好 24h → 应拒绝（边界严格 > 1 day）
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:        5,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 5, 10)

	require.ErrorIs(t, err, ErrSubscriptionTimeInsufficient)
	require.False(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_TimeInsufficientBoundary_OneDayPlusOneSecond(t *testing.T) {
	// 剩余 24h + 1s → 应成功（严格 > 1 day）
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:        6,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(24*time.Hour + 1*time.Second),
		},
		shortenReturnUpdated: true,
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 6, 10)

	require.NoError(t, err)
	require.True(t, stub.shortenCalled)
}

func TestResetSubscriptionWithCost_ConcurrentLoses(t *testing.T) {
	// 预检通过，但原子 UPDATE 返回 0 行（并发竞争失败者）
	stub := &resetWithCostStub{
		sub: &UserSubscription{
			ID:        7,
			UserID:    10,
			GroupID:   20,
			Status:    SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(5 * 24 * time.Hour),
		},
		shortenReturnUpdated: false, // 模拟 0 行影响
	}
	svc := newResetWithCostSvc(stub)

	_, err := svc.ResetSubscriptionWithCost(context.Background(), 7, 10)

	require.ErrorIs(t, err, ErrSubscriptionTimeInsufficient)
	require.True(t, stub.shortenCalled)
}
