package service

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) handleOpenAIUpstreamRequestError(ctx context.Context, c *gin.Context, account *Account, err error, upstreamURL string, passthrough bool) (string, *UpstreamFailoverError) {
	safeErr := ""
	if err != nil {
		safeErr = sanitizeUpstreamErrorMessage(err.Error())
	}
	if safeErr == "" {
		safeErr = "upstream request failed"
	}

	platform := ""
	accountID := int64(0)
	accountName := ""
	if account != nil {
		platform = account.Platform
		accountID = account.ID
		accountName = account.Name
	}

	setOpsUpstreamError(c, 0, safeErr, "")
	if !shouldFailoverOpenAIUpstreamRequestError(ctx, err, safeErr) {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           platform,
			AccountID:          accountID,
			AccountName:        accountName,
			UpstreamStatusCode: 0,
			UpstreamURL:        upstreamURL,
			Passthrough:        passthrough,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return safeErr, nil
	}

	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()

	cooldownApplied := s.markOpenAIAccountTemporarilyUnschedulable(
		stateCtx,
		account,
		0,
		"openai_request_error",
		openAIRequestErrorCooldown,
		[]byte(safeErr),
	)
	cooldownReason := ""
	if cooldownApplied {
		cooldownReason = "openai_request_error"
	}

	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           platform,
		AccountID:          accountID,
		AccountName:        accountName,
		UpstreamStatusCode: 0,
		UpstreamURL:        upstreamURL,
		Passthrough:        passthrough,
		Kind:               "failover",
		Message:            safeErr,
		CooldownApplied:    cooldownApplied,
		CooldownReason:     cooldownReason,
	})
	return safeErr, newNetworkUpstreamFailoverError(safeErr)
}

func shouldFailoverOpenAIUpstreamRequestError(ctx context.Context, err error, safeErr string) bool {
	if ctx == nil || ctx.Err() == nil {
		return true
	}
	return isOpenAIUpstreamNetworkFailoverError(err, safeErr)
}

func isOpenAIUpstreamNetworkFailoverError(err error, safeErr string) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(strings.TrimSpace(safeErr))
	if msg == "" && err != nil {
		msg = strings.ToLower(strings.TrimSpace(err.Error()))
	}
	if msg == "" {
		return false
	}

	networkSignals := []string{
		"context deadline exceeded",
		"i/o timeout",
		"dial tcp",
		"dial udp",
		"connect timeout",
		"connection timeout",
		"client.timeout exceeded",
		"timeout awaiting response headers",
		"tls handshake timeout",
		"connection refused",
		"connection reset by peer",
		"use of closed network connection",
		"http2: client connection force closed",
		"client connection force closed",
		"clientconn.close",
		"stream error",
		"goaway",
		"network is unreachable",
		"no such host",
	}
	for _, signal := range networkSignals {
		if strings.Contains(msg, signal) {
			return true
		}
	}
	return false
}

func (s *OpenAIGatewayService) handleOpenAIUpstreamStreamError(ctx context.Context, c *gin.Context, account *Account, err error, upstreamURL string, passthrough bool) *UpstreamFailoverError {
	if err == nil || !isOpenAIUpstreamStreamCircuitError(err) {
		return nil
	}
	safeErr := sanitizeUpstreamErrorMessage(err.Error())
	if safeErr == "" {
		safeErr = "upstream stream failed"
	}
	responseCommitted := c != nil && c.Writer.Written()

	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()
	cooldownApplied := s.markOpenAIAccountTemporarilyUnschedulable(
		stateCtx,
		account,
		0,
		"openai_stream_error",
		openAIRequestErrorCooldown,
		[]byte(safeErr),
	)

	kind := "failover"
	if responseCommitted {
		kind = "stream_error"
	}
	if c != nil {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           accountPlatform(account),
			AccountID:          accountID(account),
			AccountName:        accountName(account),
			UpstreamStatusCode: 0,
			UpstreamURL:        upstreamURL,
			Passthrough:        passthrough,
			Kind:               kind,
			Message:            safeErr,
			CooldownApplied:    cooldownApplied,
			CooldownReason:     "openai_stream_error",
		})
	}
	if responseCommitted {
		return nil
	}
	return newNetworkUpstreamFailoverError(safeErr)
}

func isOpenAIUpstreamStreamCircuitError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		msg := strings.ToLower(err.Error())
		return strings.Contains(msg, "connection reset") ||
			strings.Contains(msg, "client connection force closed") ||
			strings.Contains(msg, "clientconn.close")
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	markers := []string{
		"connection reset by peer",
		"http2: client connection force closed",
		"client connection force closed",
		"clientconn.close",
		"stream data interval timeout",
		"missing terminal event",
		"upstream stream ended without terminal event",
		"stream usage incomplete",
		"stream read error",
	}
	for _, marker := range markers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func accountPlatform(account *Account) string {
	if account == nil {
		return PlatformOpenAI
	}
	return account.Platform
}

func accountID(account *Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}

func accountName(account *Account) string {
	if account == nil {
		return ""
	}
	return account.Name
}
