package service

import (
	"context"
	"net/http"
)

func (s *GeminiMessagesCompatService) handleGeminiUpstreamErrorCustom(ctx context.Context, account *Account, statusCode int, headers http.Header, body []byte) bool {
	if !isUpstreamBillingExhaustionError(statusCode, extractUpstreamErrorMessage(body), body) {
		return false
	}
	if s.rateLimitService != nil {
		s.rateLimitService.HandleUpstreamError(ctx, account, statusCode, headers, body)
	}
	return true
}
