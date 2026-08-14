package service

import (
	"net/http"
	"strings"
)

const clientFacingTemporaryUnavailableMessage = "Service temporarily unavailable, please retry later"
const clientFacingGroupUnavailableMessage = "Current group is unavailable, please switch groups"

// This is distinct from clientFacingGroupUnavailableMessage: the group may be
// healthy for other models, while the requested model has no currently usable
// account. Keep it retryable (503) but make the scope actionable to users.
const clientFacingModelGroupUnavailableMessage = "The requested model is temporarily unavailable in the current group. Please retry later or switch groups."
const clientFacingModelUnsupportedMessage = "The requested model is not supported by the current group. Please choose a supported model or switch groups."

func ClientFacingTemporaryUnavailableMessage() string {
	return clientFacingTemporaryUnavailableMessage
}

func ClientFacingGroupUnavailableMessage() string {
	return clientFacingGroupUnavailableMessage
}

func ClientFacingModelGroupUnavailableMessage() string {
	return clientFacingModelGroupUnavailableMessage
}

func ClientFacingModelUnsupportedMessage() string {
	return clientFacingModelUnsupportedMessage
}

// ClientFacingErrorMessage removes internal routing/upstream details from
// messages before they are returned to API clients. Ops logs keep the original
// sanitized upstream message via setOpsUpstreamError/appendOpsUpstreamError.
func ClientFacingErrorMessage(statusCode int, errType, message string) string {
	msg := strings.TrimSpace(sanitizeUpstreamErrorMessage(message))
	if msg == "" {
		return clientErrorFallback(statusCode, errType)
	}
	if replacement, ok := clientErrorSpecificReplacement(msg); ok {
		return replacement
	}
	if clientErrorMessageLeaksInternalState(msg) {
		return clientLeakFallback(statusCode, errType)
	}
	return msg
}

func clientErrorSpecificReplacement(message string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(lower, "claude code version") && strings.Contains(lower, "please update"):
		return message, true
	case strings.Contains(lower, "gpt-5.5 context window") && strings.Contains(lower, "272k"):
		return message, true
	case strings.Contains(lower, "project_id"):
		return message, true
	case IsContextWindowExceededError(message, nil):
		return ContextWindowExceededClientMessage(""), true
	case strings.Contains(lower, "image generation concurrency limit exceeded"):
		return "Image generation concurrency limit exceeded, please retry later", true
	case strings.Contains(lower, "token counting is not supported for this openai account type"):
		return message, true
	case strings.Contains(lower, "upstream stream disconnected"):
		return message, true
	case strings.Contains(lower, "responses image_generation"):
		return "This API key does not support Responses image generation", true
	case strings.Contains(lower, "/responses/compact"):
		return "This API key does not support /responses/compact", true
	case strings.Contains(lower, "requested endpoint") && strings.Contains(lower, "account"):
		return "This API key does not support the requested endpoint", true
	case lower == "no available openai accounts":
		return clientFacingGroupUnavailableMessage, true
	case lower == "no available accounts":
		return clientFacingGroupUnavailableMessage, true
	case strings.HasPrefix(lower, "no available gemini accounts"):
		return clientFacingGroupUnavailableMessage, true
	case strings.Contains(lower, "no available") && (strings.Contains(lower, "requested model") || strings.Contains(lower, "supporting model")):
		return clientFacingModelGroupUnavailableMessage, true
	case strings.Contains(lower, "requested model is not supported by upstream"):
		return clientFacingModelGroupUnavailableMessage, true
	default:
		return "", false
	}
}

func clientErrorMessageLeaksInternalState(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		return true
	}
	if strings.Contains(lower, "cf-ray") ||
		strings.Contains(lower, "cloudflare") ||
		strings.Contains(lower, "origin web server") ||
		strings.Contains(lower, "invalid or incomplete response") ||
		strings.Contains(lower, "traceid") ||
		strings.Contains(lower, "trace id") ||
		strings.Contains(lower, "request id") ||
		strings.Contains(lower, "x-request-id") ||
		strings.Contains(lower, "url:") {
		return true
	}
	if strings.Contains(lower, "upstream") || strings.Contains(lower, "上游") {
		return true
	}
	if strings.Contains(lower, "account") ||
		strings.Contains(lower, "账号") ||
		strings.Contains(lower, "账户") {
		return true
	}
	if strings.Contains(lower, "candidate") ||
		strings.Contains(lower, "excluded") ||
		strings.Contains(lower, "schedul") ||
		strings.Contains(lower, "selection_diag") ||
		strings.Contains(lower, "sticky") ||
		strings.Contains(lower, "concurrency") ||
		strings.Contains(lower, "channel pricing") ||
		strings.Contains(lower, "temp_unschedulable") ||
		strings.Contains(lower, "routing") {
		return true
	}
	return false
}

func clientLeakFallback(statusCode int, errType string) string {
	if statusCode == http.StatusTooManyRequests || strings.EqualFold(errType, "rate_limit_error") {
		return "Service rate limit reached, please retry later"
	}
	return clientFacingTemporaryUnavailableMessage
}

func clientErrorFallback(statusCode int, errType string) string {
	switch {
	case statusCode == http.StatusTooManyRequests || strings.EqualFold(errType, "rate_limit_error"):
		return "Rate limit exceeded, please retry later"
	case statusCode == http.StatusUnauthorized || strings.EqualFold(errType, "authentication_error"):
		return "Authentication failed"
	case statusCode == http.StatusForbidden ||
		strings.EqualFold(errType, "permission_error") ||
		strings.EqualFold(errType, "forbidden_error"):
		return "Request is not allowed"
	case statusCode == http.StatusNotFound || strings.EqualFold(errType, "not_found_error"):
		return "Requested resource was not found"
	case statusCode == http.StatusBadRequest || strings.EqualFold(errType, "invalid_request_error"):
		return "Invalid request"
	default:
		return clientFacingTemporaryUnavailableMessage
	}
}
