package service

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *GeminiMessagesCompatService) writeGeminiMappedErrorOverride(c *gin.Context, upstreamStatus int, upstreamMsg string, body []byte) (error, bool) {
	if isUpstreamBillingExhaustionError(upstreamStatus, upstreamMsg, body) {
		writeClientClaudeError(c, http.StatusBadGateway, "upstream_error", "Upstream service temporarily unavailable")
		return geminiUpstreamStatusError(upstreamStatus, upstreamMsg), true
	}

	if status, errType, errMsg, matched := applyErrorPassthroughRule(
		c,
		PlatformGemini,
		upstreamStatus,
		body,
		http.StatusBadGateway,
		"upstream_error",
		"Upstream request failed",
	); matched {
		writeClientClaudeError(c, status, errType, errMsg)
		if upstreamMsg == "" {
			upstreamMsg = errMsg
		}
		if upstreamMsg == "" {
			return fmt.Errorf("upstream error: %d (passthrough rule matched)", upstreamStatus), true
		}
		return fmt.Errorf("upstream error: %d (passthrough rule matched) message=%s", upstreamStatus, upstreamMsg), true
	}

	statusCode, errType, errMsg, clientRequestMapped := ClientRequestErrorFromUpstream(upstreamStatus, upstreamMsg, body)
	needsOverride := clientRequestMapped
	switch upstreamStatus {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, 529,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		needsOverride = true
	}
	if !needsOverride {
		return nil, false
	}

	if mapped := mapGeminiErrorBodyToClaudeError(body); mapped != nil {
		errType = mapped.Type
		if mapped.Message != "" {
			errMsg = mapped.Message
		}
		if mapped.StatusCode > 0 {
			statusCode = mapped.StatusCode
		}
	}
	switch upstreamStatus {
	case http.StatusBadRequest:
		if statusCode == 0 {
			statusCode = http.StatusBadRequest
		}
		if errType == "" {
			errType = "invalid_request_error"
		}
		if errMsg == "" {
			errMsg = "Invalid request"
		}
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, 529:
		normalized := NormalizeUpstreamClientError(upstreamStatus, errType, errMsg)
		statusCode, errType, errMsg = normalized.Status, normalized.Type, normalized.Message
	case http.StatusNotFound:
		if statusCode == 0 {
			statusCode = http.StatusNotFound
		}
		if errType == "" {
			errType = "not_found_error"
		}
		if errMsg == "" {
			errMsg = "Resource not found"
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		if !clientRequestMapped {
			normalized := NormalizeUpstreamClientError(upstreamStatus, errType, errMsg)
			statusCode, errType, errMsg = normalized.Status, normalized.Type, normalized.Message
		}
	default:
		if statusCode == 0 {
			statusCode = http.StatusBadGateway
		}
		if errType == "" {
			errType = "upstream_error"
		}
		if errMsg == "" {
			errMsg = "Upstream request failed"
		}
	}

	writeClientClaudeError(c, statusCode, errType, errMsg)
	return geminiUpstreamStatusError(upstreamStatus, upstreamMsg), true
}

func geminiUpstreamStatusError(status int, message string) error {
	if message == "" {
		return fmt.Errorf("upstream error: %d", status)
	}
	return fmt.Errorf("upstream error: %d message=%s", status, message)
}
