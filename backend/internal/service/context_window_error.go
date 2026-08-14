package service

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const clientFacingContextWindowExceededMessage = "Your input exceeds the context window of this model (model context window exceeded). Please reduce the conversation length, start a new chat, or compact the context before retrying."
const clientFacingGPT55ContextWindowExceededMessage = "gpt-5.5 context window is 272k tokens. Your input exceeds this limit; please use gpt-5.4 to compress the context first, then retry with gpt-5.5."

func IsContextWindowExceededError(message string, payload []byte) bool {
	return isOpenAIContextWindowError(message, payload)
}

func ContextWindowExceededClientMessage(model string) string {
	if strings.EqualFold(strings.TrimSpace(model), "gpt-5.5") {
		return clientFacingGPT55ContextWindowExceededMessage
	}
	return clientFacingContextWindowExceededMessage
}

func ContextWindowExceededClientError(model, message string, payload []byte) (int, string, string, bool) {
	if !IsContextWindowExceededError(message, payload) {
		return 0, "", "", false
	}
	return http.StatusBadRequest, "invalid_request_error", ContextWindowExceededClientMessage(model), true
}

func writeContextWindowExceededClientError(c *gin.Context, model, message string, payload []byte, writeError func(*gin.Context, int, string, string)) (int, string, string, bool) {
	status, errType, msg, ok := ContextWindowExceededClientError(model, message, payload)
	if !ok {
		return 0, "", "", false
	}
	MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonContextWindowExceeded)
	MarkResponseCommitted(c)
	writeError(c, status, errType, msg)
	return status, errType, msg, true
}

func requestModelFromVariadic(models []string) string {
	if len(models) == 0 {
		return ""
	}
	return strings.TrimSpace(models[0])
}

func IsClaudeCodeVersionError(message string, payload []byte) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		text = strings.ToLower(strings.TrimSpace(ExtractUpstreamErrorMessage(payload)))
	}
	raw := strings.ToLower(string(payload))
	return (strings.Contains(text, "claude code version") && strings.Contains(text, "minimum required")) ||
		(strings.Contains(raw, "claude code version") && strings.Contains(raw, "minimum required"))
}

func ClaudeCodeVersionClientError(message string, payload []byte) (int, string, string, bool) {
	if !IsClaudeCodeVersionError(message, payload) {
		return 0, "", "", false
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = strings.TrimSpace(ExtractUpstreamErrorMessage(payload))
	}
	if msg == "" {
		msg = "Claude Code version is below the minimum required version. Please update Claude Code."
	}
	return http.StatusBadRequest, "invalid_request_error", ClientFacingErrorMessage(http.StatusBadRequest, "invalid_request_error", msg), true
}
