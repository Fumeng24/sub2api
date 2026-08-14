package service

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *GeminiMessagesCompatService) shouldFailoverGeminiBillingExhaustion(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	statusCode int,
	headers http.Header,
	upstreamRequestID string,
	rawBody []byte,
	eventBody []byte,
) bool {
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(eventBody))
	if upstreamMsg == "" && !bytes.Equal(eventBody, rawBody) {
		upstreamMsg = strings.TrimSpace(extractUpstreamErrorMessage(rawBody))
	}
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	if !isUpstreamBillingExhaustionError(statusCode, upstreamMsg, eventBody) &&
		!isUpstreamBillingExhaustionError(statusCode, upstreamMsg, rawBody) {
		return false
	}

	s.handleGeminiUpstreamError(ctx, account, statusCode, headers, rawBody)

	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(eventBody), maxBytes)
	}
	setOpsUpstreamError(c, statusCode, upstreamMsg, upstreamDetail)
	if account != nil {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: statusCode,
			UpstreamRequestID:  upstreamRequestID,
			Kind:               "failover",
			Message:            upstreamMsg,
			Detail:             upstreamDetail,
		})
	}
	return true
}

func (s *GeminiMessagesCompatService) geminiBillingExhaustionFailoverErrorCustom(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	resp *http.Response,
	requestIDHeader string,
	upstreamRequestID string,
	isOAuth bool,
	rawBody []byte,
) error {
	if resp == nil {
		return nil
	}
	if upstreamRequestID == "" {
		if requestIDHeader != "" {
			upstreamRequestID = resp.Header.Get(requestIDHeader)
		}
		if upstreamRequestID == "" {
			upstreamRequestID = resp.Header.Get("x-goog-request-id")
		}
	}
	eventBody := unwrapIfNeeded(isOAuth, rawBody)
	if !s.shouldFailoverGeminiBillingExhaustion(ctx, c, account, resp.StatusCode, resp.Header, upstreamRequestID, rawBody, eventBody) {
		return nil
	}
	return &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: eventBody}
}
