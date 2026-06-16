package service

import (
	"net/http"
	"strings"
)

const clientFacingTemporaryUnavailableMessage = "Service temporarily unavailable, please retry later"

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
	case strings.Contains(lower, "image generation concurrency limit exceeded"):
		return "Image generation concurrency limit exceeded, please retry later", true
	case strings.Contains(lower, "upstream stream disconnected"):
		return message, true
	case strings.Contains(lower, "responses image_generation"):
		return "This API key does not support Responses image generation", true
	case strings.Contains(lower, "/responses/compact"):
		return "This API key does not support /responses/compact", true
	case strings.Contains(lower, "requested endpoint") && strings.Contains(lower, "account"):
		return "This API key does not support the requested endpoint", true
	case strings.Contains(lower, "no available") && (strings.Contains(lower, "requested model") || strings.Contains(lower, "supporting model")):
		return "Service temporarily unavailable for the requested model, please retry later", true
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
