package service

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *GeminiMessagesCompatService) handleGeminiMessagesSkippedErrorCustom(c *gin.Context, account *Account, resp *http.Response, requestID string, body []byte) (error, bool) {
	return s.writeGeminiMappedError(c, account, resp.StatusCode, requestID, body), true
}

func (s *GeminiMessagesCompatService) handleGeminiNativeSkippedErrorCustom(c *gin.Context, account *Account, resp *http.Response, requestID string, isOAuth bool, body []byte) (error, bool) {
	body = unwrapIfNeeded(isOAuth, body)
	upstreamMsg := sanitizeUpstreamErrorMessage(extractUpstreamErrorMessage(body))
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  requestID,
		Kind:               "http_error",
		Message:            upstreamMsg,
		Detail:             upstreamDetail,
	})
	if status, _, message, ok := ClientRequestErrorFromUpstream(resp.StatusCode, upstreamMsg, body); ok {
		return s.writeGoogleError(c, status, message), true
	}
	normalized := NormalizeUpstreamClientError(resp.StatusCode, "upstream_error", upstreamMsg)
	return s.writeGoogleError(c, normalized.Status, normalized.Message), true
}

func (s *GeminiMessagesCompatService) handleGeminiNativeResponseErrorCustom(c *gin.Context, resp *http.Response, upstreamMsg string, body []byte) (error, bool) {
	if status, _, message, ok := ClientRequestErrorFromUpstream(resp.StatusCode, upstreamMsg, body); ok {
		return s.writeGoogleError(c, status, message), true
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	body = ClientFacingErrorBody(resp.StatusCode, "upstream_error", body)
	MarkResponseCommitted(c)
	c.Data(resp.StatusCode, contentType, body)
	if upstreamMsg == "" {
		return fmt.Errorf("gemini upstream error: %d", resp.StatusCode), true
	}
	return fmt.Errorf("gemini upstream error: %d message=%s", resp.StatusCode, upstreamMsg), true
}
