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
