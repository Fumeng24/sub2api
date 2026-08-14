package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) handleOpenAIImagesRequestErrorCustom(ctx context.Context, c *gin.Context, account *Account, err error, upstreamURL string) (error, bool) {
	safeErr, failoverErr := s.handleOpenAIUpstreamRequestError(ctx, c, account, err, upstreamURL, false)
	if failoverErr != nil {
		return failoverErr, true
	}
	return fmt.Errorf("upstream request failed: %s", safeErr), true
}

func (s *OpenAIGatewayService) handleOpenAIEmbeddingsRequestErrorCustom(ctx context.Context, c *gin.Context, account *Account, err error) (error, bool) {
	safeErr, failoverErr := s.handleOpenAIUpstreamRequestError(ctx, c, account, err, "", false)
	if failoverErr != nil {
		return failoverErr, true
	}
	writeOpenAIEmbeddingsError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
	return fmt.Errorf("upstream request failed: %s", safeErr), true
}

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
	cooldownApplied := s.cooldownOpenAIStatusZeroFailure(ctx, account, safeErr, "openai_request_error")

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
		CooldownReason:     cooldownReason(cooldownApplied, "openai_request_error"),
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
		"authentication failed",
		"proxy authentication required",
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
	cooldownApplied := false
	if !responseCommitted {
		cooldownApplied = s.cooldownOpenAIStatusZeroFailure(ctx, account, safeErr, "openai_stream_error")
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
			CooldownApplied:    cooldownApplied,
			CooldownReason:     cooldownReason(cooldownApplied, "openai_stream_error"),
		})
	}
	if responseCommitted {
		return nil
	}
	return newNetworkUpstreamFailoverError(safeErr)
}

func (s *OpenAIGatewayService) cooldownOpenAIStatusZeroFailure(ctx context.Context, account *Account, message string, reason string) bool {
	if s == nil || !isOpenAIAccount(account) {
		return false
	}
	cooldown := s.openAISchedulerCooldownForCategory(schedulerStatusZeroFailureCategory([]byte(message)), http.Header{})
	if cooldown <= 0 {
		cooldown = openAIRequestErrorCooldown
	}
	until := time.Now().Add(cooldown)

	if s.accountRepo == nil {
		s.BlockAccountScheduling(account, until, reason)
		return true
	}

	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()
	current := account
	if loaded, err := safeOpenAIStatusZeroGetAccountByID(stateCtx, s.accountRepo, account.ID); err == nil && loaded != nil {
		current = loaded
	}
	if current.TempUnschedulableUntil != nil && time.Now().Before(*current.TempUnschedulableUntil) {
		// Do not extend a live cooldown or add duplicate scheduler history. The
		// runtime block still needs replaying after a process restart.
		s.BlockAccountScheduling(current, *current.TempUnschedulableUntil, reason)
		return true
	}
	if s.openaiTransientCooldownThrottle != nil && !s.openaiTransientCooldownThrottle.Allow(account.ID, time.Now()) {
		s.BlockAccountScheduling(current, until, reason)
		return true
	}

	safeReason := groupReserveReasonOpenAIIO + ": " + sanitizeUpstreamErrorMessage(message)
	s.BlockAccountScheduling(current, until, reason)
	if err := safeOpenAIStatusZeroSetTempUnschedulable(stateCtx, s.accountRepo, current.ID, until, safeReason); err != nil {
		return true
	}
	recordSchedulerBlocked(stateCtx, s.accountRepo, current.ID, firstAccountGroupID(stateCtx, current), 0, safeReason, "upstream_error", until, map[string]any{
		"block_granularity": "account",
		"cooldown_seconds":  int(cooldown / time.Second),
	})
	return true
}

func safeOpenAIStatusZeroGetAccountByID(ctx context.Context, repo AccountRepository, accountID int64) (account *Account, err error) {
	if repo == nil || accountID <= 0 {
		return nil, ErrAccountNotFound
	}
	defer func() {
		if recover() != nil {
			account = nil
			err = ErrAccountNotFound
		}
	}()
	return repo.GetByID(ctx, accountID)
}

func safeOpenAIStatusZeroSetTempUnschedulable(ctx context.Context, repo AccountRepository, accountID int64, until time.Time, reason string) (err error) {
	if repo == nil || accountID <= 0 {
		return nil
	}
	defer func() {
		if recover() != nil {
			err = ErrAccountNotFound
		}
	}()
	return repo.SetTempUnschedulable(ctx, accountID, until, reason)
}

func cooldownReason(applied bool, reason string) string {
	if !applied {
		return ""
	}
	return reason
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
