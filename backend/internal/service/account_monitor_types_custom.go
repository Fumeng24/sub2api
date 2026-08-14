package service

import "time"

// 账号监控（account-monitor）：与 channel-monitor 完全独立的一套。
// 为单个 api_key 类上游账号配置探针，运行时按 account_id 实时读账号凭证打心跳。
// 纯管理员视角，无任何用户可达路由。

// AccountMonitor 账号监控配置（service 层视图）。
type AccountMonitor struct {
	ID              int64
	AccountID       int64
	Provider        string
	Model           string
	Enabled         bool
	IntervalSeconds int
	JitterSeconds   int
	LastCheckedAt   *time.Time
	CreatedBy       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AccountMonitorCheck 一次探测记录（service 层视图）。
type AccountMonitorCheck struct {
	ID               int64
	AccountMonitorID int64
	Model            string
	Status           string
	LatencyMs        *int
	PingLatencyMs    *int
	Message          string
	CheckedAt        time.Time
}

// AccountMonitorCreateParams 创建参数。Provider/Model/Interval 为空时用默认值。
type AccountMonitorCreateParams struct {
	AccountID       int64
	Provider        string // 空 = 按账号 platform 推断，回退 openai
	Model           string // 空 = accountMonitorDefaultModel
	Enabled         bool
	IntervalSeconds int // 0 = accountMonitorDefaultInterval
	JitterSeconds   int
	CreatedBy       int64
}

// AccountMonitorUpdateParams 更新参数（指针字段表示"未提供则不更新"）。
type AccountMonitorUpdateParams struct {
	Provider        *string
	Model           *string
	Enabled         *bool
	IntervalSeconds *int
	JitterSeconds   *int
}

// AccountMonitorStatus 账号监控的聚合状态（admin 视图，按 account_id 索引返回前端）。
type AccountMonitorStatus struct {
	MonitorID int64
	AccountID int64
	Model     string
	Enabled   bool
	// IntervalSeconds is copied from the monitor configuration so consumers
	// can decide whether LastCheckedAt is still fresh. A probe result must not
	// remain health evidence indefinitely after the worker is stopped.
	IntervalSeconds int
	LatestStatus    string   // 最近一次探测状态；空 = 尚无探测
	LatestLatency   *int     // 最近一次探测延迟（ms）
	PingLatencyMs   *int     // 最近一次 endpoint ping 延迟（ms）
	Availability1h  float64  // 近 1 小时可用率（百分比）
	AvgLatency1h    *float64 // 近 1 小时平均探测响应耗时（ms）；nil = 该时段无探测样本
	LastCheckedAt   *time.Time
	// Timeline 最近 N 条探测（newest-first），供前端渲染彩虹状态条。
	Timeline []*AccountMonitorCheck
}
