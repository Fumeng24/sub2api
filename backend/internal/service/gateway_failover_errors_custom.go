package service

import (
	"fmt"
	"strings"
)

func newNetworkUpstreamFailoverError(message string) *UpstreamFailoverError {
	return &UpstreamFailoverError{
		StatusCode:             0,
		ResponseBody:           []byte(message),
		RetryableOnSameAccount: true,
		SchedulerCategory:      schedulerStatusZeroFailureCategory([]byte(message)),
	}
}

// schedulerStatusFailureCategory 对状态码为 0（网络/传输层）的上游错误做分类，
// 用于填充 UpstreamFailoverError.SchedulerCategory 供错误透传/计费等业务逻辑使用。
// 注意：原实现位于已删除的调度增强文件 scheduler_health.go 中，但 SchedulerCategory
// 字段被 gemini/openai 等业务路径广泛使用，故保留这一纯字符串分类工具函数。
func schedulerStatusZeroFailureCategory(body []byte) string {
	text := strings.ToLower(strings.TrimSpace(string(body)))
	if text == "" {
		return "error"
	}
	normalized := strings.NewReplacer("_", " ", "-", " ", "\n", " ", "\r", " ", "\t", " ").Replace(text)
	normalized = strings.Join(strings.Fields(normalized), " ")
	combined := text + " " + normalized
	if containsAnySchedulerText(combined,
		"timeout",
		"deadline exceeded",
		"awaiting response headers",
		"stream data interval timeout",
	) {
		return "transient_timeout"
	}
	if containsAnySchedulerText(combined,
		"openai_request_error",
		"openai request error",
		"transport_closed",
		"transport closed",
		"account_circuit_transport_closed",
		"account circuit transport closed",
		"context canceled",
		"context cancelled",
		"connection reset by peer",
		"connection refused",
		"use of closed network connection",
		"client connection force closed",
		"clientconn.close",
		"http2:",
		"goaway",
		"dial tcp",
		"dial udp",
		"network is unreachable",
		"no such host",
		"broken pipe",
		"unexpected eof",
		" eof",
		"stream error",
		"request failed",
		"upstream connection error",
	) {
		return "transient_transport"
	}
	if strings.Contains(combined, "overload") {
		return "transient"
	}
	return "error"
}

func containsAnySchedulerText(text string, markers ...string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	for _, marker := range markers {
		marker = strings.ToLower(strings.TrimSpace(marker))
		if marker != "" && strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// UpstreamTerminalError means the service already wrote a final client error
// and the failure should not be fed back into account health or failover.
type UpstreamTerminalError struct {
	StatusCode int
	Message    string
}

func (e *UpstreamTerminalError) Error() string {
	if e == nil {
		return "upstream terminal error"
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		return fmt.Sprintf("upstream error: %d", e.StatusCode)
	}
	return fmt.Sprintf("upstream error: %d message=%s", e.StatusCode, message)
}

func newUpstreamTerminalError(statusCode int, message string) error {
	return &UpstreamTerminalError{StatusCode: statusCode, Message: sanitizeUpstreamErrorMessage(message)}
}
