package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func openAIStreamClientOutputStartedCustom(c *gin.Context, localStarted bool) (bool, bool) {
	if localStarted {
		return true, true
	}
	return OpenAICompactKeepaliveAdjustedWrittenSize(c) > 0, true
}

type openAIResponseOutputTracker struct {
	hasOutput bool
}

type openAIStreamOutputObservation struct {
	tracker openAIResponseOutputTracker
}

func (o *openAIStreamOutputObservation) Observe(data []byte) {
	if o == nil {
		return
	}
	o.tracker.ObserveSSEData(data)
}

func (o *openAIStreamOutputObservation) ObserveCorrected(data []byte) {
	if o == nil {
		return
	}
	o.tracker.ObserveSSEData(data)
}

func (o *openAIStreamOutputObservation) HasEffectiveOutput(imageCounter *openAIImageOutputCounter) bool {
	return o != nil && (o.tracker.HasOutput() || (imageCounter != nil && imageCounter.Count() > 0))
}

func (o *openAIStreamOutputObservation) Validate(
	service *OpenAIGatewayService,
	c *gin.Context,
	account *Account,
	resp *http.Response,
	payload []byte,
	imageCounter *openAIImageOutputCounter,
	clientOutputStarted bool,
	passthrough bool,
	upstreamRequestID string,
) error {
	return service.openAIEmptyOutputStreamValidationError(
		c,
		account,
		resp,
		payload,
		o.HasEffectiveOutput(imageCounter),
		clientOutputStarted,
		passthrough,
		upstreamRequestID,
	)
}

func (t *openAIResponseOutputTracker) ObserveSSEData(data []byte) {
	if t == nil || len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("[DONE]")) || !gjson.ValidBytes(data) {
		return
	}
	if openAIResponsePayloadHasEffectiveOutput(gjson.ParseBytes(data)) {
		t.hasOutput = true
	}
}

func (t *openAIResponseOutputTracker) HasOutput() bool {
	return t != nil && t.hasOutput
}

func openAICompletedPayloadIsEmptyEffectiveOutput(data []byte, hasEffectiveOutput bool) bool {
	if hasEffectiveOutput || len(bytes.TrimSpace(data)) == 0 || !gjson.ValidBytes(data) {
		return false
	}
	eventType := strings.TrimSpace(gjson.GetBytes(data, "type").String())
	if eventType != "response.completed" && eventType != "response.done" {
		return false
	}
	if !openAICompletedPayloadHasExplicitEmptyOutput(data) {
		return false
	}
	return openAIEmptyOutputReason(data) != ""
}

func openAICompletedPayloadHasExplicitEmptyOutput(data []byte) bool {
	root := gjson.ParseBytes(data)
	for _, path := range []string{"output", "response.output"} {
		output := root.Get(path)
		if output.Exists() && output.IsArray() && len(output.Array()) == 0 {
			return true
		}
	}
	return false
}

func sanitizeOpenAITransientFailedEventForClient(payload []byte, message string) []byte {
	if len(bytes.TrimSpace(payload)) == 0 || !gjson.ValidBytes(payload) {
		return payload
	}
	if !isOpenAITransientProcessingError(http.StatusBadRequest, message, payload) {
		return payload
	}
	out := append([]byte(nil), payload...)
	updated := false
	for _, path := range []string{"response.error.message", "error.message", "message"} {
		if !gjson.GetBytes(out, path).Exists() {
			continue
		}
		if next, err := sjson.SetBytes(out, path, clientFacingTemporaryUnavailableMessage); err == nil {
			out = next
			updated = true
		}
	}
	if updated {
		return out
	}
	if next, err := sjson.SetBytes(out, "error.message", clientFacingTemporaryUnavailableMessage); err == nil {
		return next
	}
	return payload
}

func newOpenAIStreamTerminalError(payload []byte, message string) error {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "Upstream response failed"
	}
	class := classifyOpenAIUpstreamError(http.StatusBadRequest, message, payload)
	statusCode, _, _ := openAIErrorResponseForClass(http.StatusBadRequest, class, message, false)
	return newUpstreamTerminalError(statusCode, message)
}

func (s *OpenAIGatewayService) recordOpenAIStreamUpstreamErrorWithCooldown(
	c *gin.Context,
	account *Account,
	passthrough bool,
	upstreamRequestID string,
	kind string,
	payload []byte,
	message string,
	cooldownApplied bool,
	cooldownReasonValue string,
) string {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "OpenAI upstream response failed"
	}
	statusCode := openAIStreamFailureStatus(payload, message)
	if s != nil && account != nil && shouldCooldownOpenAIStreamFailover(message, payload) {
		s.closeOpenAIAccountIdleConnectionsForCircuit(account.ID, 0, "openai_stream_error", []byte(message))
	}
	detail := ""
	if len(payload) > 0 && s != nil && s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		detail = truncateString(string(payload), maxBytes)
	}
	if c != nil {
		setOpsUpstreamError(c, statusCode, message, detail)
		event := OpsUpstreamErrorEvent{
			Platform:           PlatformOpenAI,
			UpstreamStatusCode: statusCode,
			UpstreamRequestID:  strings.TrimSpace(upstreamRequestID),
			Passthrough:        passthrough,
			Kind:               kind,
			CooldownApplied:    cooldownApplied,
			CooldownReason:     cooldownReasonValue,
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
	return message
}

func (s *OpenAIGatewayService) recordOpenAIStreamUpstreamErrorCustom(
	c *gin.Context,
	account *Account,
	passthrough bool,
	upstreamRequestID string,
	kind string,
	payload []byte,
	message string,
) (string, bool) {
	return s.recordOpenAIStreamUpstreamErrorWithCooldown(c, account, passthrough, upstreamRequestID, kind, payload, message, false, ""), true
}

func (s *OpenAIGatewayService) buildCustomOpenAIStreamFailoverError(
	c *gin.Context,
	account *Account,
	passthrough bool,
	upstreamRequestID string,
	payload []byte,
	message string,
	responseHeaders ...http.Header,
) *UpstreamFailoverError {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "OpenAI stream disconnected before completion"
	}
	statusCode := openAIStreamFailureStatus(payload, message)
	var headers http.Header
	if len(responseHeaders) > 0 && responseHeaders[0] != nil {
		headers = responseHeaders[0].Clone()
	}
	stateCtx := context.Background()
	if c != nil && c.Request != nil {
		stateCtx = c.Request.Context()
	}
	cooldownApplied := false
	if len(bytes.TrimSpace(payload)) == 0 && !isOpenAIContextWindowError(message, payload) {
		cooldownApplied = s.cooldownOpenAIStatusZeroFailure(stateCtx, account, message, "openai_stream_error")
	}
	message = s.recordOpenAIStreamUpstreamErrorWithCooldown(
		c,
		account,
		passthrough,
		upstreamRequestID,
		"failover",
		payload,
		message,
		cooldownApplied,
		cooldownReason(cooldownApplied, "openai_stream_error"),
	)
	errType := "upstream_error"
	if statusCode == http.StatusTooManyRequests {
		errType = "rate_limit_error"
	}
	clientMessage := ClientFacingErrorMessage(statusCode, errType, message)
	if isOpenAIContextWindowError(message, payload) {
		clientMessage = message
	}
	body, _ := json.Marshal(gin.H{
		"error": gin.H{
			"type":    errType,
			"message": clientMessage,
		},
	})
	return &UpstreamFailoverError{
		StatusCode:             statusCode,
		ResponseBody:           body,
		ResponseHeaders:        headers,
		RetryableOnSameAccount: openAIStreamFailedEventRetryableOnSameAccount(account, payload, message),
		RequestScopedTransient: isOpenAIUpstreamCapacityShedEvent(payload),
	}
}

func (s *OpenAIGatewayService) newOpenAIStreamFailoverErrorCustom(
	c *gin.Context,
	account *Account,
	passthrough bool,
	upstreamRequestID string,
	payload []byte,
	message string,
	responseHeaders ...http.Header,
) (*UpstreamFailoverError, bool) {
	return s.buildCustomOpenAIStreamFailoverError(c, account, passthrough, upstreamRequestID, payload, message, responseHeaders...), true
}

func shouldCooldownOpenAIStreamFailover(message string, payload []byte) bool {
	combined := strings.ToLower(strings.TrimSpace(message + " " + string(payload)))
	if combined == "" {
		return false
	}
	for _, marker := range []string{
		"stream usage incomplete",
		"missing terminal event",
		"upstream stream ended without terminal event",
		"stream ended before a terminal event",
		"client disconnected",
		"context canceled",
	} {
		if strings.Contains(combined, marker) {
			return false
		}
	}
	return true
}
