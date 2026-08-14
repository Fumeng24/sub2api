//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type userDiscountRepoStub struct {
	userGroupRateRepoStubForGroupRate
	rates     map[int64]float64
	discounts map[int64]float64
	synced    map[int64]*float64
}

func (s *userDiscountRepoStub) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	return s.rates, nil
}

func (s *userDiscountRepoStub) GetDiscountsByUserID(context.Context, int64) (map[int64]float64, error) {
	return s.discounts, nil
}

func (s *userDiscountRepoStub) SyncUserGroupDiscounts(_ context.Context, _ int64, discounts map[int64]*float64) error {
	s.synced = discounts
	s.discounts = make(map[int64]float64)
	for groupID, discount := range discounts {
		if discount != nil {
			s.discounts[groupID] = *discount
		}
	}
	return nil
}

func TestAdminService_UpdateUserSyncsGroupDiscounts(t *testing.T) {
	discount := 0.75
	userRepo := &rpmUserRepoStub{userRepoStub: &userRepoStub{user: &User{ID: 42, Role: RoleUser}}}
	discountRepo := &userDiscountRepoStub{
		rates:     map[int64]float64{},
		discounts: map[int64]float64{9: 0.8},
	}
	groupRepo := &groupRepoStubForAdmin{getByIDByID: map[int64]*Group{
		9: {ID: 9, SubscriptionType: SubscriptionTypeStandard},
	}}
	svc := &adminServiceImpl{
		userRepo:          userRepo,
		groupRepo:         groupRepo,
		userGroupRateRepo: discountRepo,
		redeemCodeRepo:    &redeemRepoStub{},
	}

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		GroupDiscounts: map[int64]*float64{9: &discount},
	})

	require.NoError(t, err)
	require.Equal(t, &discount, discountRepo.synced[9])
	require.Equal(t, map[int64]float64{9: 0.75}, updated.GroupDiscounts)
}

func TestAdminService_UpdateUserRejectsDiscountWithFixedRate(t *testing.T) {
	discount := 0.75
	userRepo := &rpmUserRepoStub{userRepoStub: &userRepoStub{user: &User{ID: 42, Role: RoleUser}}}
	discountRepo := &userDiscountRepoStub{
		rates:     map[int64]float64{9: 1.4},
		discounts: map[int64]float64{},
	}
	groupRepo := &groupRepoStubForAdmin{getByIDByID: map[int64]*Group{
		9: {ID: 9, SubscriptionType: SubscriptionTypeStandard},
	}}
	svc := &adminServiceImpl{
		userRepo:          userRepo,
		groupRepo:         groupRepo,
		userGroupRateRepo: discountRepo,
		redeemCodeRepo:    &redeemRepoStub{},
	}

	_, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		GroupDiscounts: map[int64]*float64{9: &discount},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "fixed rate")
	require.Empty(t, discountRepo.synced)
	require.Empty(t, userRepo.lastUpdated)
}
