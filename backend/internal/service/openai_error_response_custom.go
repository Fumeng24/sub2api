package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) writeCustomOpenAIClassifiedError(c *gin.Context, statusCode int, upstreamMsg string, body []byte) bool {
	class := classifyOpenAIUpstreamError(statusCode, upstreamMsg, body)
	if class == openAIUpstreamErrorUnknown && statusCode == http.StatusNotFound {
		return false
	}
	status, errType, message := openAIErrorResponseForClass(statusCode, class, upstreamMsg, false)
	c.JSON(status, gin.H{"error": gin.H{"type": errType, "message": message}})
	return true
}

func sanitizeOpenAIResponsesInputStatusFields(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	items, ok := reqBody["input"].([]any)
	if !ok {
		return false
	}
	changed := false
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := itemMap["status"]; exists {
			delete(itemMap, "status")
			changed = true
		}
	}
	return changed
}

func openAIErrorResponseForClass(statusCode int, class openAIUpstreamErrorClass, upstreamMsg string, compat bool) (int, string, string) {
	msg := strings.TrimSpace(upstreamMsg)
	switch class {
	case openAIUpstreamErrorBilling, openAIUpstreamErrorForbidden, openAIUpstreamErrorAuth:
		return http.StatusBadGateway, "upstream_error", clientFacingTemporaryUnavailableMessage
	case openAIUpstreamErrorBusiness:
		if msg == "" {
			msg = "Upstream request is not enabled or not supported"
		}
		return clientFacingOpenAI4xxStatus(statusCode, http.StatusBadRequest), "invalid_request_error", msg
	case openAIUpstreamErrorRateLimit:
		return http.StatusServiceUnavailable, "upstream_error", clientFacingTemporaryUnavailableMessage
	case openAIUpstreamErrorTransient:
		return http.StatusBadGateway, "api_error", clientFacingTemporaryUnavailableMessage
	case openAIUpstreamErrorModelUnsupported:
		return http.StatusServiceUnavailable, "api_error", clientFacingGroupUnavailableMessage
	}
	switch statusCode {
	case http.StatusBadRequest:
		if msg == "" || !compat {
			msg = "Upstream request failed"
		}
		return http.StatusBadRequest, "invalid_request_error", msg
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden:
		return http.StatusBadGateway, "upstream_error", clientFacingTemporaryUnavailableMessage
	case http.StatusNotFound:
		if msg == "" || !compat {
			msg = "Upstream resource not found"
		}
		return http.StatusNotFound, "not_found_error", msg
	case http.StatusTooManyRequests:
		return http.StatusServiceUnavailable, "upstream_error", clientFacingTemporaryUnavailableMessage
	case 529:
		return http.StatusServiceUnavailable, "api_error", clientFacingTemporaryUnavailableMessage
	}
	if statusCode >= 500 {
		return http.StatusBadGateway, "api_error", clientFacingTemporaryUnavailableMessage
	}
	if compat && msg != "" && statusCode >= 400 && statusCode < 500 {
		return statusCode, "api_error", msg
	}
	return http.StatusBadGateway, "upstream_error", clientFacingTemporaryUnavailableMessage
}

func clientFacingOpenAI4xxStatus(statusCode, fallback int) int {
	if statusCode >= 400 && statusCode < 500 {
		return statusCode
	}
	return fallback
}

func (s *OpenAIGatewayService) handleCustomOpenAIErrorPreflight(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestBody, body []byte,
	upstreamMsg, upstreamDetail string,
	requestedModel ...string,
) (*OpenAIForwardResult, error, bool) {
	if isOpenAILogicalCompactRequest(c) && isOpenAIContextWindowError(upstreamMsg, body) {
		return nil, s.newOpenAICompactContextWindowFailoverError(c, account, resp, body, false, upstreamMsg, upstreamDetail), true
	}

	reqModel := requestModelFromVariadic(requestedModel)
	if reqModel == "" {
		reqModel, _, _ = extractOpenAIRequestMetaFromBody(requestBody)
	}
	if status, _, _, ok := writeContextWindowExceededClientError(c, reqModel, upstreamMsg, body, writeClientOpenAIError); ok {
		return nil, newUpstreamTerminalError(status, upstreamMsg), true
	}

	upstreamClass := classifyOpenAIUpstreamError(resp.StatusCode, upstreamMsg, body)
	if s.autoDisableCodexImageBridgeForUnsupportedUpstream(ctx, account, upstreamMsg, body) {
		s.appendCustomOpenAIErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "failover")
		return nil, s.newCustomOpenAIFailoverError(ctx, account, resp, body, upstreamMsg), true
	}
	if openAIUpstreamErrorClassImmediateFailover(upstreamClass) {
		_ = s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, reqModel)
		s.appendCustomOpenAIErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "failover")
		return nil, s.newCustomOpenAIFailoverError(ctx, account, resp, body, upstreamMsg), true
	}
	return nil, nil, false
}

func (s *OpenAIGatewayService) handleCustomOpenAIPassthroughErrorPreflight(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestBody, body []byte,
	upstreamMsg, upstreamDetail string,
	cyberHit bool,
) (error, bool) {
	if isOpenAILogicalCompactRequest(c) && isOpenAIContextWindowError(upstreamMsg, body) {
		return s.newOpenAICompactContextWindowFailoverError(c, account, resp, body, true, upstreamMsg, upstreamDetail), true
	}

	// Passthrough keeps the upstream HTTP status and its sanitized actionable
	// message. The non-passthrough bridge owns the local 400 conversion.
	reqModel, _, _ := extractOpenAIRequestMetaFromBody(requestBody)
	if cyberHit {
		return nil, false
	}

	upstreamClass := classifyOpenAIUpstreamError(resp.StatusCode, upstreamMsg, body)
	if s.autoDisableCodexImageBridgeForUnsupportedUpstream(ctx, account, upstreamMsg, body) {
		s.appendCustomOpenAIPassthroughErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "failover")
		return s.newCustomOpenAIFailoverError(ctx, account, resp, body, upstreamMsg), true
	}
	shouldDisable := s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, reqModel)
	forbiddenGroupDisabled := upstreamClass == openAIUpstreamErrorForbidden && isOpenAIGroupDisabledUpstreamError(resp.StatusCode, upstreamMsg, body)
	if upstreamClass == openAIUpstreamErrorBilling || forbiddenGroupDisabled ||
		shouldDisable && openAIUpstreamErrorClassShouldFailover(upstreamClass) {
		s.appendCustomOpenAIPassthroughErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "failover")
		return s.newCustomOpenAIFailoverError(ctx, account, resp, body, upstreamMsg), true
	}
	return nil, false
}

func (s *OpenAIGatewayService) handleCustomOpenAICompatErrorPreflight(
	resp *http.Response,
	c *gin.Context,
	account *Account,
	writeError compatErrorWriter,
	body []byte,
	upstreamMsg, upstreamDetail string,
	requestedModel ...string,
) (*OpenAIForwardResult, error, bool) {
	ctx := openAIContextFromGin(context.Background(), c)
	if status, _, _, ok := writeContextWindowExceededClientError(c, requestModelFromVariadic(requestedModel), upstreamMsg, body, writeError); ok {
		return nil, newUpstreamTerminalError(status, upstreamMsg), true
	}
	class := classifyOpenAIUpstreamError(resp.StatusCode, upstreamMsg, body)
	if !openAIUpstreamErrorClassImmediateFailover(class) {
		return nil, nil, false
	}
	model := requestModelFromVariadic(requestedModel)
	_ = s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, model)
	s.appendCustomOpenAIErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "failover")
	return nil, s.newCustomOpenAIFailoverError(ctx, account, resp, body, upstreamMsg), true
}

func (s *OpenAIGatewayService) handleUnhandledOpenAIStatusCustom(
	c *gin.Context,
	account *Account,
	resp *http.Response,
	upstreamMsg, upstreamDetail string,
) (*OpenAIForwardResult, error, bool) {
	if account.ShouldHandleErrorCode(resp.StatusCode) {
		return nil, nil, false
	}
	s.appendCustomOpenAIErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "http_error")
	MarkResponseCommitted(c)
	writeClientOpenAIError(c, http.StatusBadGateway, "upstream_error", clientFacingTemporaryUnavailableMessage)
	return nil, unhandledOpenAIStatusError(resp.StatusCode, upstreamMsg), true
}

func (s *OpenAIGatewayService) handleUnhandledOpenAICompatStatusCustom(
	c *gin.Context,
	account *Account,
	resp *http.Response,
	writeError compatErrorWriter,
	upstreamMsg, upstreamDetail string,
) (*OpenAIForwardResult, error, bool) {
	if account.ShouldHandleErrorCode(resp.StatusCode) {
		return nil, nil, false
	}
	s.appendCustomOpenAIErrorEvent(c, account, resp, upstreamMsg, upstreamDetail, "http_error")
	MarkResponseCommitted(c)
	writeError(c, http.StatusBadGateway, "upstream_error", clientFacingTemporaryUnavailableMessage)
	return nil, unhandledOpenAIStatusError(resp.StatusCode, upstreamMsg), true
}

func unhandledOpenAIStatusError(statusCode int, upstreamMsg string) error {
	if upstreamMsg == "" {
		return fmt.Errorf("upstream error: %d (not in custom error codes)", statusCode)
	}
	return fmt.Errorf("upstream error: %d (not in custom error codes) message=%s", statusCode, upstreamMsg)
}

func (s *OpenAIGatewayService) appendCustomOpenAIErrorEvent(c *gin.Context, account *Account, resp *http.Response, message, detail, kind string) {
	event := OpsUpstreamErrorEvent{
		Platform:           PlatformOpenAI,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  resp.Header.Get("x-request-id"),
		Kind:               kind,
		Message:            message,
		Detail:             detail,
	}
	if account != nil {
		event.Platform = account.Platform
		event.AccountID = account.ID
		event.AccountName = account.Name
	}
	appendOpsUpstreamError(c, event)
}

func (s *OpenAIGatewayService) appendCustomOpenAIPassthroughErrorEvent(c *gin.Context, account *Account, resp *http.Response, message, detail, kind string) {
	event := OpsUpstreamErrorEvent{
		Platform:             PlatformOpenAI,
		UpstreamStatusCode:   resp.StatusCode,
		UpstreamRequestID:    resp.Header.Get("x-request-id"),
		Passthrough:          true,
		Kind:                 kind,
		Message:              message,
		Detail:               detail,
		UpstreamResponseBody: detail,
	}
	if account != nil {
		event.Platform = account.Platform
		event.AccountID = account.ID
		event.AccountName = account.Name
	}
	appendOpsUpstreamError(c, event)
}

func (s *OpenAIGatewayService) newCustomOpenAIFailoverError(ctx context.Context, account *Account, resp *http.Response, body []byte, message string) *UpstreamFailoverError {
	err := &UpstreamFailoverError{
		StatusCode:      resp.StatusCode,
		ResponseBody:    body,
		ResponseHeaders: resp.Header.Clone(),
	}
	if account != nil {
		err.SchedulerCategory = s.schedulerCategoryOverrideForOpenAIUpstreamError(ctx, account, resp.StatusCode, body)
		err.RetryableOnSameAccount = s.retryableOnSameOpenAIAccount(ctx, account, resp.StatusCode, message, body)
	}
	return err
}

func openAIContextFromGin(fallback context.Context, c *gin.Context) context.Context {
	if c != nil && c.Request != nil && c.Request.Context() != nil {
		requestCtx := c.Request.Context()
		if openAICodexImageBridgeApplied(fallback) {
			requestCtx = withOpenAICodexImageBridgeApplied(requestCtx)
		}
		return requestCtx
	}
	if fallback != nil {
		return fallback
	}
	return context.Background()
}
