//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type autoResetStub struct {
	userSubRepoNoop

	sub *UserSubscription

	updateCalled  bool
	updateSubID   int64
	updateEnabled bool
	updateErr     error
}

func (r *autoResetStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *autoResetStub) UpdateAutoResetDaily(_ context.Context, subID int64, enabled bool) error {
	r.updateCalled = true
	r.updateSubID = subID
	r.updateEnabled = enabled
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.sub != nil {
		r.sub.AutoResetDaily = enabled
	}
	return nil
}

func newAutoResetSvc(stub *autoResetStub) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, nil, nil, nil)
}

func TestSetAutoResetDaily_Enable(t *testing.T) {
	stub := &autoResetStub{
		sub: &UserSubscription{ID: 1, UserID: 10, GroupID: 20, AutoResetDaily: false},
	}
	svc := newAutoResetSvc(stub)

	result, err := svc.SetAutoResetDaily(context.Background(), 1, 10, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.updateCalled)
	require.Equal(t, int64(1), stub.updateSubID)
	require.True(t, stub.updateEnabled)
	require.True(t, result.AutoResetDaily)
}

func TestSetAutoResetDaily_Disable(t *testing.T) {
	stub := &autoResetStub{
		sub: &UserSubscription{ID: 2, UserID: 10, GroupID: 20, AutoResetDaily: true},
	}
	svc := newAutoResetSvc(stub)

	result, err := svc.SetAutoResetDaily(context.Background(), 2, 10, false)

	require.NoError(t, err)
	require.False(t, result.AutoResetDaily)
	require.False(t, stub.updateEnabled)
}

func TestSetAutoResetDaily_AdminCall_SkipsOwnershipCheck(t *testing.T) {
	stub := &autoResetStub{
		sub: &UserSubscription{ID: 3, UserID: 10, GroupID: 20},
	}
	svc := newAutoResetSvc(stub)

	_, err := svc.SetAutoResetDaily(context.Background(), 3, 0, true)

	require.NoError(t, err)
	require.True(t, stub.updateCalled, "admin call should skip ownership and proceed")
}

func TestSetAutoResetDaily_OwnershipMismatch(t *testing.T) {
	stub := &autoResetStub{
		sub: &UserSubscription{ID: 4, UserID: 10, GroupID: 20},
	}
	svc := newAutoResetSvc(stub)

	_, err := svc.SetAutoResetDaily(context.Background(), 4, 999, true)

	require.ErrorIs(t, err, ErrSubscriptionNotOwned)
	require.False(t, stub.updateCalled)
}

func TestSetAutoResetDaily_NotFound(t *testing.T) {
	stub := &autoResetStub{sub: nil}
	svc := newAutoResetSvc(stub)

	_, err := svc.SetAutoResetDaily(context.Background(), 999, 10, true)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.False(t, stub.updateCalled)
}
