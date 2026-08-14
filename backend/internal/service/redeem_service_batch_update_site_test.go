//go:build unit

package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRedeemService_BatchUpdate_ValidatesBusinessCategoryAgainstCodeType(t *testing.T) {
	category := BalanceBusinessCategoryManualRefund
	repo := &redeemRepoStub{
		listByIDsResult: []RedeemCode{{
			ID:    42,
			Type:  RedeemTypeBalance,
			Value: 100,
		}},
	}
	svc := &RedeemService{redeemRepo: repo}

	result, err := svc.BatchUpdate(context.Background(), &RedeemCodeBatchUpdateInput{
		IDs: []int64{42},
		Fields: RedeemCodeBatchUpdateFields{
			RedeemCodeBatchUpdateFieldsCustom: RedeemCodeBatchUpdateFieldsCustom{BusinessCategory: &category},
		},
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.False(t, repo.batchUpdateCalled)
}

func TestRedeemService_BatchUpdate_ValidatesAdminBalanceCategoryDirection(t *testing.T) {
	category := BalanceBusinessCategoryManualRefund
	repo := &redeemRepoStub{
		listByIDsResult: []RedeemCode{{
			ID:    42,
			Type:  AdjustmentTypeAdminBalance,
			Value: 100,
		}},
	}
	svc := &RedeemService{redeemRepo: repo}

	result, err := svc.BatchUpdate(context.Background(), &RedeemCodeBatchUpdateInput{
		IDs: []int64{42},
		Fields: RedeemCodeBatchUpdateFields{
			RedeemCodeBatchUpdateFieldsCustom: RedeemCodeBatchUpdateFieldsCustom{BusinessCategory: &category},
		},
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.False(t, repo.batchUpdateCalled)
}

func TestRedeemService_BatchUpdate_AllowsValidAdminBalanceCategory(t *testing.T) {
	category := BalanceBusinessCategoryManualRefund
	repo := &redeemRepoStub{
		listByIDsResult: []RedeemCode{{
			ID:    42,
			Type:  AdjustmentTypeAdminBalance,
			Value: -100,
		}},
	}
	svc := &RedeemService{redeemRepo: repo}

	result, err := svc.BatchUpdate(context.Background(), &RedeemCodeBatchUpdateInput{
		IDs: []int64{42},
		Fields: RedeemCodeBatchUpdateFields{
			RedeemCodeBatchUpdateFieldsCustom: RedeemCodeBatchUpdateFieldsCustom{BusinessCategory: &category},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Updated)
	require.True(t, repo.batchUpdateCalled)
	require.Equal(t, &category, repo.batchUpdateFields.BusinessCategory)
}

func TestRedeemService_CreateCode_RejectsInvalidBusinessCategory(t *testing.T) {
	repo := &redeemRepoStub{}
	svc := &RedeemService{redeemRepo: repo}

	err := svc.CreateCode(context.Background(), &RedeemCode{
		Code:   "BAD-CATEGORY",
		Type:   RedeemTypeBalance,
		Value:  100,
		Status: StatusUnused,
		RedeemCodeCustom: RedeemCodeCustom{
			BusinessCategory: BalanceBusinessCategoryManualRefund,
		},
	})

	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Empty(t, repo.created)
}

func TestRedeemService_CreateCode_DefaultsBalanceBusinessCategory(t *testing.T) {
	repo := &redeemRepoStub{}
	svc := &RedeemService{redeemRepo: repo}

	err := svc.CreateCode(context.Background(), &RedeemCode{
		Code:   "BALANCE-CODE",
		Type:   RedeemTypeBalance,
		Value:  100,
		Status: StatusUnused,
	})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	require.Equal(t, BalanceBusinessCategoryRecharge, repo.created[0].BusinessCategory)
}
