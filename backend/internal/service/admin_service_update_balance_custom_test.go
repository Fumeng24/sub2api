//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type failingBalanceRedeemRepoStub struct {
	*balanceRedeemRepoStub
	err error
}

func (s *failingBalanceRedeemRepoStub) Create(context.Context, *RedeemCode) error { return s.err }

func TestAdminService_UpdateUserBalance_ReturnsErrorWhenAdjustmentRecordFails(t *testing.T) {
	repo := &balanceUserRepoStub{userRepoStub: &userRepoStub{user: &User{ID: 7, Balance: 10}}}
	createErr := errors.New("insert adjustment failed")
	redeemRepo := &failingBalanceRedeemRepoStub{balanceRedeemRepoStub: &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}, err: createErr}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: redeemRepo, authCacheInvalidator: invalidator}

	user, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "")
	require.Nil(t, user)
	require.ErrorIs(t, err, createErr)
	require.Empty(t, invalidator.userIDs)
}
