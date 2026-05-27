package service

import "context"

// UserGroupRateEntry 分组下用户专属倍率/RPM 条目。
// RateMultiplier、DiscountMultiplier 与 RPMOverride 均为指针以支持"未设置"语义（NULL）。
type UserGroupRateEntry struct {
	UserID             int64    `json:"user_id"`
	UserName           string   `json:"user_name"`
	UserEmail          string   `json:"user_email"`
	UserNotes          string   `json:"user_notes"`
	UserStatus         string   `json:"user_status"`
	RateMultiplier     *float64 `json:"rate_multiplier,omitempty"`
	DiscountMultiplier *float64 `json:"discount_multiplier,omitempty"`
	RPMOverride        *int     `json:"rpm_override,omitempty"`
}

// UserGroupRateConfig 是热路径读取的单个用户-分组计费配置。
// RateMultiplier 非 nil 时为固定倍率，优先级高于 DiscountMultiplier。
// DiscountMultiplier 非 nil 时表示相对于当前分组默认倍率的折扣系数。
type UserGroupRateConfig struct {
	RateMultiplier     *float64
	DiscountMultiplier *float64
}

// GroupRateMultiplierInput 批量设置分组倍率的输入条目
type GroupRateMultiplierInput struct {
	UserID         int64   `json:"user_id"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

// GroupRPMOverrideInput 批量设置分组 RPM override 的输入条目。
// RPMOverride 为 *int 以支持清除（nil）语义。
type GroupRPMOverrideInput struct {
	UserID      int64 `json:"user_id"`
	RPMOverride *int  `json:"rpm_override"`
}

// UserGroupRateRepository 用户专属分组倍率/折扣/RPM 仓储接口。
// 允许管理员为特定用户设置分组的固定倍率、折扣与 RPM 上限，覆盖分组默认值。
type UserGroupRateRepository interface {
	// GetByUserID 获取用户所有专属分组 rate_multiplier（仅返回非 NULL 的条目）
	GetByUserID(ctx context.Context, userID int64) (map[int64]float64, error)

	// GetDiscountsByUserID 获取用户所有专属分组 discount_multiplier（仅返回非 NULL 的条目）
	GetDiscountsByUserID(ctx context.Context, userID int64) (map[int64]float64, error)

	// GetEffectiveByUserID 获取用户所有专属分组的最终有效倍率。
	// 固定倍率优先；否则按当前分组倍率 * discount_multiplier 计算。
	GetEffectiveByUserID(ctx context.Context, userID int64) (map[int64]float64, error)

	// GetRateConfigByUserAndGroup 获取用户在特定分组的固定倍率/折扣配置（全 NULL 或无行返回 nil）
	GetRateConfigByUserAndGroup(ctx context.Context, userID, groupID int64) (*UserGroupRateConfig, error)

	// GetRPMOverrideByUserAndGroup 获取用户在特定分组的 rpm_override（NULL 返回 nil）
	GetRPMOverrideByUserAndGroup(ctx context.Context, userID, groupID int64) (*int, error)

	// GetByGroupID 获取指定分组下所有用户的专属配置（rate 与 rpm_override 任一非 NULL 即返回）
	GetByGroupID(ctx context.Context, groupID int64) ([]UserGroupRateEntry, error)

	// SyncUserGroupRates 同步用户的分组专属倍率；nil 表示清空该分组的 rate_multiplier
	SyncUserGroupRates(ctx context.Context, userID int64, rates map[int64]*float64) error

	// SyncUserGroupDiscounts 同步用户的分组专属折扣；nil 表示清空该分组的 discount_multiplier
	SyncUserGroupDiscounts(ctx context.Context, userID int64, discounts map[int64]*float64) error

	// SyncGroupRateMultipliers 批量同步分组的用户专属倍率（替换整组 rate 部分）
	SyncGroupRateMultipliers(ctx context.Context, groupID int64, entries []GroupRateMultiplierInput) error

	// SyncGroupRPMOverrides 批量同步分组的用户专属 RPM（替换整组 rpm_override 部分）。
	// 条目中 RPMOverride 为 nil 时清空对应行的 rpm_override；非 nil 时 upsert。
	SyncGroupRPMOverrides(ctx context.Context, groupID int64, entries []GroupRPMOverrideInput) error

	// ClearGroupRPMOverrides 清空指定分组的所有 rpm_override（整组 rpm 部分归 NULL）
	ClearGroupRPMOverrides(ctx context.Context, groupID int64) error

	// DeleteByGroupID 删除指定分组的所有用户专属条目（分组删除时调用）
	DeleteByGroupID(ctx context.Context, groupID int64) error

	// DeleteByUserID 删除指定用户的所有专属条目（用户删除时调用）
	DeleteByUserID(ctx context.Context, userID int64) error
}
