package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
)

func prepareOpenAIPassthroughImageGateCustom(apiKey *APIKey, body []byte) ([]byte, error) {
	if GroupAllowsImageGeneration(apiKeyGroup(apiKey)) {
		return body, nil
	}
	updated, stripped, err := stripDisabledGroupImplicitOpenAIImageTools(body)
	if err != nil {
		return nil, err
	}
	if stripped {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI passthrough] Removed implicit /responses image_generation tool for disabled group")
	}
	return updated, nil
}

func (s *OpenAIGatewayService) prepareOpenAIPassthroughFailoverCustom(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	body := s.readUpstreamErrorBody(resp)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return shouldFailoverOpenAIPassthroughResponseCustom(resp.StatusCode, body)
}

func shouldFailoverOpenAIPassthroughResponseCustom(statusCode int, responseBody []byte) bool {
	// Context-window errors are deterministic request failures. They must remain
	// on the current account so the client can compact or shorten the request.
	if isOpenAIContextWindowError("", responseBody) {
		return false
	}
	if isOpenAIGroupDisabledUpstreamError(statusCode, "", responseBody) {
		return true
	}
	switch statusCode {
	case http.StatusPaymentRequired, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func openAIStreamDataStartsClientOutputCustom(data, eventType string) (bool, bool) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" || strings.TrimSpace(eventType) == "response.failed" || openAIStreamEventIsPreamble(eventType) {
		return false, true
	}
	if trimmed == "[DONE]" {
		return true, true
	}
	if strings.TrimSpace(eventType) == "error" {
		payload := []byte(trimmed)
		return !openAIStreamFailedEventShouldFailover(payload, extractOpenAISSEErrorMessage(payload)), true
	}
	if gjson.Valid(trimmed) {
		return openAIResponsePayloadHasEffectiveOutput(gjson.Parse(trimmed)), true
	}
	return true, true
}

func openAIStreamFailedEventShouldFailoverCustom(payload []byte, message string) (bool, bool) {
	if isOpenAIUpstreamCapacityShedEvent(payload) {
		return true, true
	}
	if hit, _, _ := detectOpenAICyberPolicy(payload); hit {
		return false, true
	}
	if isOpenAIThinkingSignatureInvalidError(payload, message) {
		return false, true
	}
	class := classifyOpenAIUpstreamError(http.StatusBadRequest, message, payload)
	if !openAIUpstreamErrorClassShouldFailover(class) && class != openAIUpstreamErrorUnknown {
		return false, true
	}
	return false, false
}
