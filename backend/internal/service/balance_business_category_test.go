package service

import (
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateRedeemCodeBusinessCategoryRejectsInvalidTypeCategory(t *testing.T) {
	err := ValidateRedeemCodeBusinessCategory(RedeemTypeBalance, 100, BalanceBusinessCategoryManualRefund)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}

func TestValidateRedeemCodeBusinessCategoryRejectsInvalidAdminDirection(t *testing.T) {
	err := ValidateRedeemCodeBusinessCategory(AdjustmentTypeAdminBalance, 100, BalanceBusinessCategoryManualRefund)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))

	err = ValidateRedeemCodeBusinessCategory(AdjustmentTypeAdminBalance, -100, BalanceBusinessCategoryManualCollection)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}

func TestValidateRedeemCodeBusinessCategoryAllowsValidAdminDirection(t *testing.T) {
	require.NoError(t, ValidateRedeemCodeBusinessCategory(AdjustmentTypeAdminBalance, 100, BalanceBusinessCategoryManualCollection))
	require.NoError(t, ValidateRedeemCodeBusinessCategory(AdjustmentTypeAdminBalance, -100, BalanceBusinessCategoryManualRefund))
	require.NoError(t, ValidateRedeemCodeBusinessCategory(AdjustmentTypeAdminBalance, -10, BalanceBusinessCategorySystemServiceFee))
}

func TestNormalizeAdminBalanceBusinessCategoryDefaultsByOperation(t *testing.T) {
	category, err := NormalizeAdminBalanceBusinessCategory("add", 50, "")
	require.NoError(t, err)
	require.Equal(t, BalanceBusinessCategoryManualCollection, category)

	category, err = NormalizeAdminBalanceBusinessCategory("subtract", -50, "")
	require.NoError(t, err)
	require.Equal(t, BalanceBusinessCategoryManualRefund, category)
}
