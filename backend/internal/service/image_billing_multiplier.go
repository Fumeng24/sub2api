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

func resolveImageRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64, activeDiscountMultiplier float64) float64 {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.ImageRateIndependent {
		if apiKey.Group.ImageRateMultiplier < 0 {
			return 0
		}
		if activeDiscountMultiplier <= 0 {
			activeDiscountMultiplier = 1
		}
		return apiKey.Group.ImageRateMultiplier * activeDiscountMultiplier
	}
	return effectiveGroupMultiplier
}
