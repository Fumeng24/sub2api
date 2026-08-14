package service

import "time"

func isAvailabilityStatusError(acc *AccountAvailability) bool {
	if acc == nil {
		return false
	}
	if acc.BlockReason != "" {
		return acc.BlockReason == AccountSchedulingBlockInactive.String() && acc.HasError
	}
	return acc.HasError && acc.TempUnschedulableUntil == nil
}

func isAvailabilityTempUnschedulableAt(acc *AccountAvailability, now time.Time) bool {
	if acc == nil {
		return false
	}
	if acc.BlockReason != "" {
		return acc.BlockReason == AccountSchedulingBlockTempUnschedulable.String()
	}
	return acc.TempUnschedulableUntil != nil && now.Before(*acc.TempUnschedulableUntil)
}
