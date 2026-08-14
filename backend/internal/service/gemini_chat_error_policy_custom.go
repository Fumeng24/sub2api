package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func geminiChatCompletionsRequestFailoverCustom(safeErr string) (error, bool) {
	return newNetworkUpstreamFailoverError(safeErr), true
}

func (s *GeminiMessagesCompatService) writeGeminiChatCompletionsErrorOverride(c *gin.Context, upstreamStatus int, upstreamMsg string, body []byte) (error, bool) {
	if isUpstreamBillingExhaustionError(upstreamStatus, upstreamMsg, body) {
		return s.writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Upstream service temporarily unavailable"), true
	}

	statusCode, errType, errMsg, clientRequestMapped := ClientRequestErrorFromUpstream(upstreamStatus, upstreamMsg, body)
	needsOverride := clientRequestMapped
	switch upstreamStatus {
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden,
		http.StatusTooManyRequests, 529, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		needsOverride = true
	}
	if !needsOverride {
		return nil, false
	}
	if mapped := mapGeminiErrorBodyToClaudeError(body); mapped != nil {
		if mapped.Type != "" {
			errType = mapped.Type
		}
		if mapped.Message != "" {
			errMsg = mapped.Message
		}
		if mapped.StatusCode > 0 {
			statusCode = mapped.StatusCode
		}
	}
	switch upstreamStatus {
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden,
		http.StatusTooManyRequests, 529:
		normalized := NormalizeUpstreamClientError(upstreamStatus, errType, errMsg)
		statusCode, errType, errMsg = normalized.Status, normalized.Type, normalized.Message
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		if !clientRequestMapped {
			normalized := NormalizeUpstreamClientError(upstreamStatus, errType, errMsg)
			statusCode, errType, errMsg = normalized.Status, normalized.Type, normalized.Message
		}
	}
	return s.writeChatCompletionsError(c, statusCode, errType, errMsg), true
}
