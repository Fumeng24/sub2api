package service

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

// 账号监控默认值与错误常量。运行时/调度相关的超时、并发、状态字符串常量
// 直接复用 channel_monitor_const.go 中的同包定义（monitorWorkerConcurrency、
// monitorRequestTimeout、MonitorStatus*、monitorAvailability7Days 等）。

const (
	// accountMonitorDefaultModel 默认探测模型（admin 可在 UI 改）。
	accountMonitorDefaultModel = "gpt-5.4-mini"
	// accountMonitorDefaultInterval 默认探测间隔（秒）。
	accountMonitorDefaultInterval = 60
	// accountMonitorDefaultBaseURL 账号未配 base_url 时的回退 endpoint。
	accountMonitorDefaultBaseURL = "https://api.openai.com"
)

var (
	// ErrAccountMonitorNotFound 监控不存在。
	ErrAccountMonitorNotFound = infraerrors.NotFound(
		"ACCOUNT_MONITOR_NOT_FOUND", "account monitor not found",
	)
	// ErrAccountMonitorInvalidAccountID account_id 非法。
	ErrAccountMonitorInvalidAccountID = infraerrors.BadRequest(
		"ACCOUNT_MONITOR_INVALID_ACCOUNT_ID", "account_id must be a positive integer",
	)
	// ErrAccountMonitorExists 该账号已存在监控（account_id 唯一）。
	ErrAccountMonitorExists = infraerrors.BadRequest(
		"ACCOUNT_MONITOR_EXISTS", "this account already has a monitor",
	)
	// ErrAccountMonitorNotEligible 绑定的账号不存在 / 非 api_key 类 / 非 active / 缺 api_key 凭证。
	ErrAccountMonitorNotEligible = infraerrors.BadRequest(
		"ACCOUNT_MONITOR_NOT_ELIGIBLE", "account must be an active api_key-type account with an api_key credential",
	)
)
