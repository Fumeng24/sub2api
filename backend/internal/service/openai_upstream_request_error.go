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

	if account != nil {
		s.closeOpenAIAccountIdleConnectionsForCircuit(account.ID, 0, "openai_request_error", []byte(safeErr))
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
		CooldownApplied:    false,
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
		"openai_request_error",
		"openai request error",
		"transport_closed",
		"transport closed",
		"account_circuit_transport_closed",
		"account circuit transport closed",
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
		"broken pipe",
		"unexpected eof",
	}
	for _, signal := range networkSignals {
		if strings.Contains(msg, signal) {
			return true
		}
	}
	return false
}

func (s *OpenAIGatewayService) handleOpenAIUpstreamStreamError(ctx context.Context, c *gin.Context, account *Account, err error, upstreamURL string, passthrough bool) *UpstreamFailoverError {
	if err == nil || !isOpenAIUpstreamStreamFailoverError(err) {
		return nil
	}
	safeErr := sanitizeUpstreamErrorMessage(err.Error())
	if safeErr == "" {
		safeErr = "upstream stream failed"
	}
	responseCommitted := c != nil && c.Writer.Written()

	if isOpenAIUpstreamStreamCircuitError(err) {
		if account != nil {
			s.closeOpenAIAccountIdleConnectionsForCircuit(account.ID, 0, "openai_stream_error", []byte(safeErr))
		}
	}

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
			CooldownApplied:    false,
		})
	}
	if responseCommitted {
		return nil
	}
	return newNetworkUpstreamFailoverError(safeErr)
}

func isOpenAIUpstreamStreamFailoverError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	nonReplayableMarkers := []string{
		"client disconnected",
		"context canceled",
	}
	for _, marker := range nonReplayableMarkers {
		if strings.Contains(msg, marker) {
			return false
		}
	}
	if isOpenAIUpstreamStreamCircuitError(err) {
		return true
	}
	diagnosticFailoverMarkers := []string{
		"stream usage incomplete",
		"missing terminal event",
		"upstream stream ended without terminal event",
		"empty_effective_output",
	}
	for _, marker := range diagnosticFailoverMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func isOpenAIUpstreamStreamCircuitError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	diagnosticOnlyMarkers := []string{
		"stream usage incomplete",
		"missing terminal event",
		"upstream stream ended without terminal event",
		"empty_effective_output",
		"client disconnected",
	}
	for _, marker := range diagnosticOnlyMarkers {
		if strings.Contains(msg, marker) {
			return false
		}
	}
	markers := []string{
		"connection reset by peer",
		"http2: client connection force closed",
		"client connection force closed",
		"clientconn.close",
		"stream data interval timeout",
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
