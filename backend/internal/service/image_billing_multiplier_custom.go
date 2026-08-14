package service

import (
	"context"
	"time"
)

func resolveAPIKeyBillingGroupID(apiKey *APIKey) (int64, bool) {
	if apiKey == nil {
		return 0, false
	}
	if apiKey.GroupID != nil && *apiKey.GroupID > 0 {
		return *apiKey.GroupID, true
	}
	if apiKey.Group != nil && apiKey.Group.ID > 0 {
		return apiKey.Group.ID, true
	}
	return 0, false
}

func activeGroupRateDiscountMultiplierAt(ctx context.Context, settingService *SettingService, groupID int64, now time.Time) float64 {
	if settingService == nil || groupID <= 0 {
		return 1
	}
	discount := settingService.ActiveGroupRateDiscount(ctx, now)
	if discount == nil || !discount.AppliesToGroup(groupID) || discount.DiscountMultiplier <= 0 {
		return 1
	}
	return discount.DiscountMultiplier
}

func resolveIndependentImageRateMultiplierCustom(apiKey *APIKey, activeDiscountMultiplier []float64) (float64, bool) {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.ImageRateIndependent {
		return 0, false
	}
	return applyIndependentRateDiscountCustom(apiKey.Group.ImageRateMultiplier, activeDiscountMultiplier), true
}

func resolveIndependentVideoRateMultiplierCustom(apiKey *APIKey, activeDiscountMultiplier []float64) (float64, bool) {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.VideoRateIndependent {
		return 0, false
	}
	return applyIndependentRateDiscountCustom(apiKey.Group.VideoRateMultiplier, activeDiscountMultiplier), true
}

func applyIndependentRateDiscountCustom(multiplier float64, activeDiscountMultiplier []float64) float64 {
	if multiplier < 0 {
		return 0
	}
	discount := 1.0
	if len(activeDiscountMultiplier) > 0 && activeDiscountMultiplier[0] > 0 {
		discount = activeDiscountMultiplier[0]
	}
	return multiplier * discount
}
