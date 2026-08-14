package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func NormalizeBalanceBusinessCategory(category string) string {
	return strings.TrimSpace(category)
}

func IsKnownBalanceBusinessCategory(category string) bool {
	switch NormalizeBalanceBusinessCategory(category) {
	case BalanceBusinessCategoryUnclassified,
		BalanceBusinessCategoryRecharge,
		BalanceBusinessCategoryManualCollection,
		BalanceBusinessCategoryManualRefund,
		BalanceBusinessCategoryGiftCompensation,
		BalanceBusinessCategoryGiftReversal,
		BalanceBusinessCategorySystemServiceFee,
		BalanceBusinessCategoryAffiliateReward:
		return true
	default:
		return false
	}
}

func DefaultAdminBalanceBusinessCategory(operation string, diff float64) string {
	if diff == 0 {
		return BalanceBusinessCategoryUnclassified
	}
	switch operation {
	case "add":
		return BalanceBusinessCategoryManualCollection
	case "subtract":
		return BalanceBusinessCategoryManualRefund
	case "set":
		if diff > 0 {
			return BalanceBusinessCategoryManualCollection
		}
		return BalanceBusinessCategoryManualRefund
	default:
		if diff > 0 {
			return BalanceBusinessCategoryManualCollection
		}
		return BalanceBusinessCategoryManualRefund
	}
}

func NormalizeAdminBalanceBusinessCategory(operation string, diff float64, category string) (string, error) {
	category = NormalizeBalanceBusinessCategory(category)
	if category == "" {
		category = DefaultAdminBalanceBusinessCategory(operation, diff)
	}
	if err := ValidateRedeemCodeBusinessCategory(AdjustmentTypeAdminBalance, diff, category); err != nil {
		return "", err
	}
	return category, nil
}

func ValidateRedeemCodeBusinessCategory(codeType string, value float64, category string) error {
	category = NormalizeBalanceBusinessCategory(category)
	if category != "" && !IsKnownBalanceBusinessCategory(category) {
		return infraerrors.BadRequest("BALANCE_BUSINESS_CATEGORY_INVALID", "business_category is invalid")
	}

	switch codeType {
	case RedeemTypeBalance:
		switch category {
		case BalanceBusinessCategoryUnclassified, BalanceBusinessCategoryRecharge:
			return nil
		default:
			return infraerrors.BadRequest("BALANCE_BUSINESS_CATEGORY_TYPE_INVALID", "balance redeem codes can only be classified as recharge")
		}
	case AdjustmentTypeAdminBalance:
		switch category {
		case BalanceBusinessCategoryUnclassified,
			BalanceBusinessCategoryManualCollection,
			BalanceBusinessCategoryManualRefund,
			BalanceBusinessCategoryGiftCompensation,
			BalanceBusinessCategoryGiftReversal,
			BalanceBusinessCategorySystemServiceFee:
		default:
			return infraerrors.BadRequest("BALANCE_BUSINESS_CATEGORY_TYPE_INVALID", "admin balance adjustments cannot use this business_category")
		}
		if value > 0 && (category == BalanceBusinessCategoryManualRefund || category == BalanceBusinessCategoryGiftReversal || category == BalanceBusinessCategorySystemServiceFee) {
			return infraerrors.BadRequest("BALANCE_BUSINESS_CATEGORY_DIRECTION_INVALID", "business_category does not match a positive balance adjustment")
		}
		if value < 0 && (category == BalanceBusinessCategoryManualCollection || category == BalanceBusinessCategoryGiftCompensation) {
			return infraerrors.BadRequest("BALANCE_BUSINESS_CATEGORY_DIRECTION_INVALID", "business_category does not match a negative balance adjustment")
		}
		return nil
	default:
		if category != BalanceBusinessCategoryUnclassified {
			return infraerrors.BadRequest("BALANCE_BUSINESS_CATEGORY_TYPE_INVALID", "this redeem code type cannot use a balance business_category")
		}
		return nil
	}
}
