package domain

// GroupAutoSortBasis is retained for configuration/API compatibility. The
// scheduler now uses one canonical health comparator for every basis; Basis
// no longer allows cost or latency to outrank stability/model success. It may
// still be shown as a legacy label until the admin UI is migrated.
const (
	// GroupAutoSortBasisRate 按最终倍率升序（低倍率账号优先调度）。
	GroupAutoSortBasisRate = "rate"
	// GroupAutoSortBasisAvailability 按近 1 小时可用率降序（高可用账号优先调度）。
	GroupAutoSortBasisAvailability = "availability"
	// GroupAutoSortBasisLatency 按账号监控近 1 小时平均探测响应耗时升序（响应快的账号优先调度）。
	GroupAutoSortBasisLatency = "latency"
	// GroupAutoSortBasisExperience 按真实用户请求的稳定性与长尾延迟综合排序。
	GroupAutoSortBasisExperience = "experience"
)

// GroupAutoSortConfig 是分组级「持续自动排序」配置，存于 groups.auto_sort_config (jsonb)。
// 后端定时任务按 Basis 周期性重排该分组成员账号的 priority。
type GroupAutoSortConfig struct {
	// Enabled 为 true 时该分组参与自动排序。
	Enabled bool `json:"enabled,omitempty"`
	// Basis 排序依据：rate、experience、availability 或 latency。空值视为 rate。
	Basis string `json:"basis,omitempty"`
}

// NormalizedBasis returns a valid legacy label. The backend comparator is
// basis-independent, so callers should not use this value to select a
// different ranking algorithm.
func (c GroupAutoSortConfig) NormalizedBasis() string {
	switch c.Basis {
	case GroupAutoSortBasisAvailability:
		return GroupAutoSortBasisAvailability
	case GroupAutoSortBasisLatency:
		return GroupAutoSortBasisLatency
	case GroupAutoSortBasisExperience:
		return GroupAutoSortBasisExperience
	default:
		return GroupAutoSortBasisRate
	}
}
