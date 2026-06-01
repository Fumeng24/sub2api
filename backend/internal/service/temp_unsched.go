package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const tempUnschedNetworkOrStreamInterruption = "network_or_stream_interruption"

// TempUnschedState 临时不可调度状态
type TempUnschedState struct {
	UntilUnix       int64  `json:"until_unix"`        // 解除时间（Unix 时间戳）
	TriggeredAtUnix int64  `json:"triggered_at_unix"` // 触发时间（Unix 时间戳）
	StatusCode      int    `json:"status_code"`       // 触发的错误码
	MatchedKeyword  string `json:"matched_keyword"`   // 匹配的关键词
	RuleIndex       int    `json:"rule_index"`        // 触发的规则索引
	ErrorMessage    string `json:"error_message"`     // 错误消息
}

// TempUnschedulableReasonDetails is the display-safe view of a stored temp unschedulable reason.
type TempUnschedulableReasonDetails struct {
	DisplayReason string
	StatusCode    *int
}

// TempUnschedulableReasonDetailsFromRaw parses current and legacy temp unschedulable reason formats.
func TempUnschedulableReasonDetailsFromRaw(raw string) TempUnschedulableReasonDetails {
	raw = strings.TrimSpace(raw)
	if isUninformativeTempUnschedReason(raw) {
		return TempUnschedulableReasonDetails{}
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || len(payload) == 0 {
		return TempUnschedulableReasonDetails{DisplayReason: truncateString(raw, 512)}
	}

	statusCode, hasStatusCode := tempUnschedStatusCode(payload["status_code"])
	details := TempUnschedulableReasonDetails{}
	if hasStatusCode {
		details.StatusCode = &statusCode
	}

	for _, key := range []string{"matched_keyword", "reason"} {
		if value := tempUnschedReasonString(payload[key]); value != "" {
			details.DisplayReason = truncateString(value, 512)
			return details
		}
	}

	if hasStatusCode && statusCode == 0 {
		details.DisplayReason = tempUnschedNetworkOrStreamInterruption
		return details
	}

	if value := tempUnschedReasonString(payload["error_message"]); value != "" {
		details.DisplayReason = truncateString(value, 512)
		return details
	}
	if hasStatusCode {
		details.DisplayReason = fmt.Sprintf("HTTP %d", statusCode)
	}
	return details
}

// TempUnschedulableDisplayReasonFromRaw returns a display-safe reason string.
func TempUnschedulableDisplayReasonFromRaw(raw string) string {
	return TempUnschedulableReasonDetailsFromRaw(raw).DisplayReason
}

func tempUnschedReasonString(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	s = strings.TrimSpace(s)
	if isUninformativeTempUnschedReason(s) {
		return ""
	}
	return s
}

func isUninformativeTempUnschedReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "", "unknown", "<nil>", "null":
		return true
	default:
		return false
	}
}

func tempUnschedStatusCode(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case json.Number:
		n, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		s := strings.TrimSpace(typed)
		if s == "" {
			return 0, false
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

// TempUnschedCache 临时不可调度缓存接口
type TempUnschedCache interface {
	SetTempUnsched(ctx context.Context, accountID int64, state *TempUnschedState) error
	GetTempUnsched(ctx context.Context, accountID int64) (*TempUnschedState, error)
	DeleteTempUnsched(ctx context.Context, accountID int64) error
}

// TimeoutCounterCache 超时计数器缓存接口
type TimeoutCounterCache interface {
	// IncrementTimeoutCount 增加账户的超时计数，返回当前计数值
	// windowMinutes 是计数窗口时间（分钟），超过此时间计数器会自动重置
	IncrementTimeoutCount(ctx context.Context, accountID int64, windowMinutes int) (int64, error)
	// GetTimeoutCount 获取账户当前的超时计数
	GetTimeoutCount(ctx context.Context, accountID int64) (int64, error)
	// ResetTimeoutCount 重置账户的超时计数
	ResetTimeoutCount(ctx context.Context, accountID int64) error
	// GetTimeoutCountTTL 获取计数器剩余过期时间
	GetTimeoutCountTTL(ctx context.Context, accountID int64) (time.Duration, error)
}
