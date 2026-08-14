package service

import "context"

type userGroupRateEntryCustom struct {
	DiscountMultiplier *float64 `json:"discount_multiplier,omitempty"`
}

// UserGroupRateConfig is the hot-path fixed-rate/relative-discount pair.
type UserGroupRateConfig struct {
	RateMultiplier     *float64
	DiscountMultiplier *float64
}

type userGroupRateRepositoryCustom interface {
	GetDiscountsByUserID(ctx context.Context, userID int64) (map[int64]float64, error)
	GetEffectiveByUserID(ctx context.Context, userID int64) (map[int64]float64, error)
	GetRateConfigByUserAndGroup(ctx context.Context, userID, groupID int64) (*UserGroupRateConfig, error)
	SyncUserGroupDiscounts(ctx context.Context, userID int64, discounts map[int64]*float64) error
}
