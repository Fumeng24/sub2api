package service

import "time"

type accountAvailabilityFlagSet struct {
	BlockReason         string
	IsAvailable         bool
	IsRateLimited       bool
	IsOverloaded        bool
	IsTempUnschedulable bool
	HasError            bool
}

func accountAvailabilityFlags(account *Account, now time.Time) accountAvailabilityFlagSet {
	flags, _ := accountAvailabilityFlagsCustom(account, now)
	return flags
}

func accountAvailabilityFlagsCustom(account *Account, now time.Time) (accountAvailabilityFlagSet, bool) {
	class := account.SchedulabilityClassAt(now)
	return accountAvailabilityFlagSet{
		BlockReason: class.Reason.String(), IsAvailable: class.Schedulable,
		IsRateLimited: class.RateLimited, IsOverloaded: class.Overloaded,
		IsTempUnschedulable: class.TempUnschedulable, HasError: class.StatusError,
	}, true
}

func (f accountAvailabilityFlagSet) values() (string, bool, bool, bool, bool, bool) {
	return f.BlockReason, f.IsAvailable, f.IsRateLimited, f.IsOverloaded, f.IsTempUnschedulable, f.HasError
}
