package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func prepareOpenAIStreamingAwareErrorCustom(c *gin.Context, status int, errType, message string, streamStarted bool) string {
	return service.ClientFacingErrorMessage(status, errType, message)
}

func (h *OpenAIGatewayHandler) validateResponsesPricingCustom(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, model string, mapping service.ChannelMappingResult, imageIntent bool) bool {
	if imageIntent {
		return true
	}
	if err := h.gatewayService.ValidateUsagePricingAvailable(c.Request.Context(), apiKey, model, mapping); err != nil {
		reqLog.Warn("openai.responses.pricing_unavailable", zap.Error(err))
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", usagePricingUnavailableMessage(model))
		return false
	}
	return true
}

func (h *OpenAIGatewayHandler) validateMessagesPricingCustom(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, model string, mapping service.ChannelMappingResult) bool {
	if err := h.gatewayService.ValidateUsagePricingAvailable(c.Request.Context(), apiKey, model, mapping); err != nil {
		reqLog.Warn("openai.messages.pricing_unavailable", zap.Error(err))
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", usagePricingUnavailableMessage(model))
		return false
	}
	return true
}

func (h *OpenAIGatewayHandler) validateWebSocketPricingCustom(ctx context.Context, conn *coderws.Conn, reqLog *zap.Logger, apiKey *service.APIKey, model string, mapping service.ChannelMappingResult, imageIntent bool) bool {
	if imageIntent {
		return true
	}
	if err := h.gatewayService.ValidateUsagePricingAvailable(ctx, apiKey, model, mapping); err != nil {
		reqLog.Warn("openai.websocket.pricing_unavailable", zap.Error(err))
		message := usagePricingUnavailableMessage(model)
		writeOpenAIWebSocketError(ctx, conn, "invalid_request_error", message)
		closeOpenAIClientWS(conn, coderws.StatusPolicyViolation, message)
		return false
	}
	return true
}

func (h *OpenAIGatewayHandler) configureSingleAccountRetryCustom(c *gin.Context, ctx context.Context, groupID *int64, model string, requireCompact bool) (context.Context, bool) {
	enabled := h.gatewayService.IsSingleSchedulableAccountForRequest(ctx, groupID, model, requireCompact, service.OpenAIEndpointCapabilityChatCompletions)
	if !enabled {
		return ctx, false
	}
	ctx = service.WithSingleAccountRetry(ctx, true, false)
	c.Request = c.Request.WithContext(ctx)
	return ctx, true
}

func (h *OpenAIGatewayHandler) configureOpenAIStickyFailoverCustom(c *gin.Context, groupID *int64, sessionHash, requestedModel string) {
	if c == nil || c.Request == nil || h == nil || h.gatewayService == nil {
		return
	}
	ctx := h.gatewayService.PrepareOpenAIStickyFailoverContext(c.Request.Context(), groupID, sessionHash, requestedModel)
	c.Request = c.Request.WithContext(ctx)
}

func (h *OpenAIGatewayHandler) applyOpenAIStickyFailoverFailureCustom(
	c *gin.Context,
	reqLog *zap.Logger,
	account *service.Account,
	canonicalModel string,
	failoverErr *service.UpstreamFailoverError,
) {
	if c == nil || c.Request == nil || h == nil || h.gatewayService == nil {
		return
	}
	ctx := h.applyOpenAIStickyFailoverFailureContextCustom(c.Request.Context(), reqLog, account, canonicalModel, failoverErr)
	c.Request = c.Request.WithContext(ctx)
}

func (h *OpenAIGatewayHandler) applyOpenAIStickyFailoverFailureContextCustom(
	ctx context.Context,
	reqLog *zap.Logger,
	account *service.Account,
	canonicalModel string,
	failoverErr *service.UpstreamFailoverError,
) context.Context {
	if h == nil || h.gatewayService == nil ||
		!h.gatewayService.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, account, canonicalModel, failoverErr) {
		return ctx
	}
	ctx = service.WithOpenAIPreserveStickyBinding(ctx)
	if reqLog == nil {
		return ctx
	}
	reqLog.Info("openai.sticky_binding_preserved_after_transient_failover",
		zap.Int64("account_id", account.ID),
		zap.String("model", canonicalModel),
		zap.Int("upstream_status", failoverErr.StatusCode),
	)
	return ctx
}

// shouldRetryOpenAIPoolModeSameAccount preserves a final account's configured
// pool retry while avoiding repeated response-header waits when another
// compatible, untried account can serve the same request.
func (h *OpenAIGatewayHandler) shouldRetryOpenAIPoolModeSameAccount(
	ctx context.Context,
	reqLog *zap.Logger,
	account *service.Account,
	failoverErr *service.UpstreamFailoverError,
	req service.OpenAIAlternativeAccountRequest,
) bool {
	if failoverErr == nil || !failoverErr.RetryableOnSameAccount || account == nil {
		return false
	}
	if h == nil || h.gatewayService == nil {
		return true
	}
	req.CurrentAccountID = account.ID
	hasAlternative, err := h.gatewayService.HasEligibleOpenAIAccountAlternative(ctx, req)
	if err != nil {
		if reqLog != nil {
			reqLog.Warn("openai.pool_mode_same_account_retry_alternative_check_failed",
				zap.Int64("account_id", account.ID),
				zap.Error(err),
			)
		}
		// Preserve the old retry behavior if availability cannot be verified.
		return true
	}
	if !hasAlternative {
		return true
	}
	if reqLog != nil {
		reqLog.Info("openai.pool_mode_same_account_retry_skipped",
			zap.Int64("account_id", account.ID),
			zap.String("model", req.RequestedModel),
			zap.String("platform", req.Platform),
			zap.Int("upstream_status", failoverErr.StatusCode),
		)
	}
	return false
}

// handleSingleAccountRetryCustom owns the local retry policy while callers keep
// protocol-specific terminal responses and post-retry slot handling.
func (h *OpenAIGatewayHandler) handleSingleAccountRetryCustom(
	ctx context.Context,
	reqLog *zap.Logger,
	account *service.Account,
	failoverErr *service.UpstreamFailoverError,
	enabled bool,
	switchCount *int,
	maxSwitches int,
	endpoint string,
	logEvent string,
	recordAccountSwitch bool,
	onExhausted func(),
) (handled bool, retry bool) {
	if !enabled || failoverErr == nil || !shouldSingleCandidateRetryStatus(failoverErr.StatusCode) {
		return false, false
	}
	h.gatewayService.RecordOpenAISchedulingBlockSkipped(ctx, account, failoverErr.StatusCode, "single_candidate_retry", endpoint)
	if *switchCount >= maxSwitches {
		onExhausted()
		return true, false
	}
	*switchCount++
	if recordAccountSwitch {
		h.gatewayService.RecordOpenAIAccountSwitch()
	}
	reqLog.Warn(logEvent,
		zap.Int64("account_id", account.ID),
		zap.Int("upstream_status", failoverErr.StatusCode),
		zap.Int("retry_count", *switchCount),
		zap.Int("max_retries", maxSwitches),
	)
	if !sleepWithContext(ctx, singleAccountBackoffDelay) {
		return true, false
	}
	return true, true
}

type singleAccountRetryAction uint8

const (
	singleAccountRetryActionUnhandled singleAccountRetryAction = iota
	singleAccountRetryActionRetry
	singleAccountRetryActionStop
)

func singleAccountRetryActionFromResult(handled, retry bool) singleAccountRetryAction {
	if !handled {
		return singleAccountRetryActionUnhandled
	}
	if retry {
		return singleAccountRetryActionRetry
	}
	return singleAccountRetryActionStop
}

func (a singleAccountRetryAction) handled() bool {
	return a != singleAccountRetryActionUnhandled
}

func (a singleAccountRetryAction) shouldRetry() bool {
	return a == singleAccountRetryActionRetry
}

func (h *OpenAIGatewayHandler) handleResponsesSingleAccountRetryCustom(c *gin.Context, reqLog *zap.Logger, account *service.Account, failoverErr *service.UpstreamFailoverError, enabled bool, switchCount *int, maxSwitches int, streamStarted bool) singleAccountRetryAction {
	handled, retry := h.handleSingleAccountRetryCustom(c.Request.Context(), reqLog, account, failoverErr, enabled, switchCount, maxSwitches, "openai_responses", "openai.single_candidate_retry", false, func() {
		h.handleFailoverExhausted(c, failoverErr, streamStarted)
	})
	return singleAccountRetryActionFromResult(handled, retry)
}

func (h *OpenAIGatewayHandler) handleMessagesSingleAccountRetryCustom(c *gin.Context, reqLog *zap.Logger, account *service.Account, failoverErr *service.UpstreamFailoverError, enabled bool, switchCount *int, maxSwitches int, streamStarted bool) singleAccountRetryAction {
	handled, retry := h.handleSingleAccountRetryCustom(c.Request.Context(), reqLog, account, failoverErr, enabled, switchCount, maxSwitches, "openai_messages", "openai_messages.single_candidate_retry", false, func() {
		h.handleAnthropicFailoverExhausted(c, failoverErr, streamStarted)
	})
	return singleAccountRetryActionFromResult(handled, retry)
}

func (h *OpenAIGatewayHandler) handleChatCompletionsSingleAccountRetryCustom(c *gin.Context, reqLog *zap.Logger, account *service.Account, failoverErr *service.UpstreamFailoverError, enabled bool, switchCount *int, maxSwitches int, streamStarted bool) singleAccountRetryAction {
	handled, retry := h.handleSingleAccountRetryCustom(c.Request.Context(), reqLog, account, failoverErr, enabled, switchCount, maxSwitches, "openai_chat_completions", "openai_chat_completions.single_candidate_retry", false, func() {
		h.handleFailoverExhausted(c, failoverErr, streamStarted)
	})
	return singleAccountRetryActionFromResult(handled, retry)
}

func (h *OpenAIGatewayHandler) handleWebSocketSingleAccountRetryCustom(ctx context.Context, conn *coderws.Conn, reqLog *zap.Logger, account *service.Account, failoverErr *service.UpstreamFailoverError, enabled bool, switchCount *int, maxSwitches int, ensureUserSlotHeld func() bool) singleAccountRetryAction {
	handled, retry := h.handleSingleAccountRetryCustom(ctx, reqLog, account, failoverErr, enabled, switchCount, maxSwitches, "openai_websocket", "openai.websocket_single_candidate_retry", true, func() {
		closeOpenAIWSFailoverExhausted(conn, failoverErr)
	})
	if retry && !ensureUserSlotHeld() {
		retry = false
	}
	return singleAccountRetryActionFromResult(handled, retry)
}

func (h *OpenAIGatewayHandler) submitUsageRecordTaskCustom(task service.UsageRecordTask) bool {
	if h.usageRecordWorkerPool == nil {
		return false
	}
	if mode := h.usageRecordWorkerPool.Submit(task); mode != service.UsageRecordSubmitModeDropped {
		return true
	}
	logger.L().With(
		zap.String("component", "handler.openai_gateway.responses"),
	).Warn("openai.usage_record_task_dropped_sync_fallback")
	runUsageRecordTaskSync("handler.openai_gateway.responses", "openai.usage_record_task_panic_recovered", task)
	return true
}

func sanitizeOpenAIWSCloseReason(status coderws.StatusCode, reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "request failed"
	}
	httpStatus := http.StatusBadGateway
	errType := "upstream_error"
	switch status {
	case coderws.StatusPolicyViolation:
		httpStatus = http.StatusBadRequest
		errType = "invalid_request_error"
	case coderws.StatusTryAgainLater:
		httpStatus = http.StatusServiceUnavailable
	}
	return service.ClientFacingErrorMessage(httpStatus, errType, reason)
}

func closeOpenAIWSFailoverExhaustedCustom(conn *coderws.Conn, failoverErr *service.UpstreamFailoverError) bool {
	if failoverErr == nil || !service.IsUpstreamModelUnsupportedError(failoverErr.StatusCode, failoverErr.ResponseBody) {
		return false
	}
	closeOpenAIClientWS(conn, coderws.StatusTryAgainLater, service.ClientFacingGroupUnavailableMessage())
	return true
}

func (h *OpenAIGatewayHandler) handleFailoverExhaustedCustom(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) bool {
	if failoverErr == nil {
		h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, streamStarted)
		return true
	}
	if !service.IsUpstreamModelUnsupportedError(failoverErr.StatusCode, failoverErr.ResponseBody) {
		return false
	}
	upstreamMsg := service.ExtractUpstreamErrorMessage(failoverErr.ResponseBody)
	service.SetOpsUpstreamError(c, failoverErr.StatusCode, upstreamMsg, "")
	h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", service.ClientFacingGroupUnavailableMessage(), streamStarted)
	return true
}

func (h *OpenAIGatewayHandler) handleAnthropicFailoverExhaustedCustom(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) bool {
	if failoverErr == nil {
		h.anthropicStreamingAwareError(c, http.StatusBadGateway, "api_error", service.ClientFacingTemporaryUnavailableMessage(), streamStarted)
		return true
	}
	if !service.IsUpstreamModelUnsupportedError(failoverErr.StatusCode, failoverErr.ResponseBody) {
		return false
	}
	h.anthropicStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", service.ClientFacingGroupUnavailableMessage(), streamStarted)
	return true
}

func openAIWSFailoverCloseStatusAndReason(failoverErr *service.UpstreamFailoverError) (coderws.StatusCode, string) {
	if failoverErr == nil {
		return coderws.StatusInternalError, service.ClientFacingTemporaryUnavailableMessage()
	}
	if service.IsUpstreamModelUnsupportedError(failoverErr.StatusCode, failoverErr.ResponseBody) {
		return coderws.StatusTryAgainLater, service.ClientFacingGroupUnavailableMessage()
	}
	switch failoverErr.StatusCode {
	case http.StatusTooManyRequests, 529, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return coderws.StatusTryAgainLater, service.ClientFacingTemporaryUnavailableMessage()
	case http.StatusUnauthorized, http.StatusForbidden:
		return coderws.StatusPolicyViolation, service.ClientFacingTemporaryUnavailableMessage()
	default:
		return coderws.StatusInternalError, service.ClientFacingTemporaryUnavailableMessage()
	}
}

func writeOpenAIWebSocketError(ctx context.Context, conn *coderws.Conn, errorType string, message string) {
	if conn == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payload := buildOpenAIWebSocketErrorPayload(errorType, message)
	writeCtx, cancel := context.WithTimeout(ctx, openAIWebSocketErrorWriteTimeout)
	defer cancel()
	_ = conn.Write(writeCtx, coderws.MessageText, payload)
}

const openAIWebSocketErrorWriteTimeout = 2 * time.Second

func buildOpenAIWebSocketErrorPayload(errorType string, message string) []byte {
	errorType = strings.TrimSpace(errorType)
	if errorType == "" {
		errorType = "invalid_request_error"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "request failed"
	}
	message = service.ClientFacingErrorMessage(http.StatusBadRequest, errorType, message)
	payload, err := json.Marshal(gin.H{
		"event_id": "evt_request_rejected",
		"type":     "error",
		"error": gin.H{
			"type": errorType, "code": "pricing_unavailable", "message": message,
		},
	})
	if err != nil {
		return []byte(`{"event_id":"evt_request_rejected","type":"error","error":{"type":"invalid_request_error","code":"pricing_unavailable","message":"request failed"}}`)
	}
	return payload
}
