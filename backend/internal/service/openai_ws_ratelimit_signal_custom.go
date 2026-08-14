package service

import (
	"context"
	"net/http"
)

func (s *OpenAIGatewayService) persistOpenAIWSRateLimitSignalCustom(
	ctx context.Context,
	account *Account,
	headers http.Header,
	responseBody []byte,
	codeRaw string,
	errTypeRaw string,
	msgRaw string,
) bool {
	if s == nil || account == nil || account.Platform != PlatformGrok || !isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw) {
		return false
	}
	s.handleGrokAccountUpstreamError(ctx, account, http.StatusTooManyRequests, headers, responseBody)
	return true
}
