package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func newAnthropicPassthroughRequestFailoverCustom(safeErr string) (*UpstreamFailoverError, bool) {
	return &UpstreamFailoverError{
		StatusCode:             0,
		ResponseBody:           []byte(safeErr),
		RetryableOnSameAccount: !isAnthropicResponseHeaderTimeoutCustom(safeErr),
		SchedulerCategory:      schedulerStatusZeroFailureCategory([]byte(safeErr)),
	}, true
}

func isAnthropicResponseHeaderTimeoutCustom(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	return strings.Contains(message, "timeout awaiting response headers") ||
		strings.Contains(message, "timeout awaiting headers") ||
		strings.Contains(message, "response header timeout")
}

func (s *GatewayService) handleAnthropicPassthrough400FailoverCustom(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	resp *http.Response,
	requestModel string,
) (*ForwardResult, error, bool) {
	if resp == nil || resp.StatusCode != http.StatusBadRequest || !s.shouldFailoverOn400FromBodyCustom(resp) {
		return nil, nil, false
	}

	respBody, _ := s.readUpstreamErrorBody(resp)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	logger.LegacyPrintf("service.gateway", "[Anthropic Passthrough] Upstream error (failover): Account=%d(%s) Status=%d RequestID=%s Body=%s",
		account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))

	s.handleFailoverSideEffects(ctx, resp, account, requestModel)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  resp.Header.Get("x-request-id"),
		Passthrough:        true,
		Kind:               "failover_on_400",
		Message:            extractUpstreamErrorMessage(respBody),
		Detail: func() string {
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
			}
			return ""
		}(),
	})
	return nil, &UpstreamFailoverError{
		StatusCode:             resp.StatusCode,
		ResponseBody:           respBody,
		RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
	}, true
}

func (s *GatewayService) shouldFailoverOn400FromBodyCustom(resp *http.Response) bool {
	if resp == nil || resp.Body == nil {
		return false
	}
	body, _ := s.readUpstreamErrorBody(resp)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return s.shouldFailoverOn400(body)
}

func handleAnthropicPassthroughIncompleteStreamCustom(account *Account, usage *ClaudeUsage, firstTokenMs *int, clientDisconnected bool) (*streamingResult, bool) {
	if !claudeUsageHasTokens(usage) {
		return nil, false
	}
	logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Stream ended without terminal event but usage was collected: account=%d input=%d output=%d cache_creation=%d cache_read=%d", account.ID, usage.InputTokens, usage.OutputTokens, usage.CacheCreationInputTokens, usage.CacheReadInputTokens)
	return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: clientDisconnected}, true
}
