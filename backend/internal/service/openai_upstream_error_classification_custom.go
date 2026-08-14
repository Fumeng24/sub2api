package service

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

type openAIUpstreamErrorClass string

const (
	openAIUpstreamErrorUnknown          openAIUpstreamErrorClass = "unknown"
	openAIUpstreamErrorAuth             openAIUpstreamErrorClass = "auth"
	openAIUpstreamErrorBilling          openAIUpstreamErrorClass = "billing"
	openAIUpstreamErrorForbidden        openAIUpstreamErrorClass = "forbidden"
	openAIUpstreamErrorRateLimit        openAIUpstreamErrorClass = "rate_limit"
	openAIUpstreamErrorTransient        openAIUpstreamErrorClass = "transient"
	openAIUpstreamErrorModelUnsupported openAIUpstreamErrorClass = "model_unsupported"
	openAIUpstreamErrorBusiness         openAIUpstreamErrorClass = "business"
)

func classifyOpenAIUpstreamError(statusCode int, upstreamMsg string, upstreamBody []byte) openAIUpstreamErrorClass {
	// A provider returning this response has permanently moved/disabled the
	// account's configured endpoint. It is account-specific (not a transient
	// provider outage) and must fail over immediately on the first request.
	if isOpenAIEndpointMigrationError(statusCode, upstreamMsg, upstreamBody) {
		return openAIUpstreamErrorAuth
	}
	if statusCode >= 400 && isOpenAIThinkingSignatureInvalidError(upstreamBody, upstreamMsg) {
		return openAIUpstreamErrorBusiness
	}
	if isUpstreamModelNotFoundError(statusCode, upstreamBody) {
		return openAIUpstreamErrorModelUnsupported
	}

	code := strings.ToLower(strings.TrimSpace(extractUpstreamErrorCode(upstreamBody)))
	errType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "error.type").String()))
	detailCode := strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "detail.code").String()))
	text := normalizeOpenAIUpstreamErrorText(upstreamMsg, upstreamBody)

	if isOpenAIGroupDisabledUpstreamError(statusCode, upstreamMsg, upstreamBody) {
		return openAIUpstreamErrorAuth
	}

	if containsAnyOpenAIErrorText(code+" "+errType+" "+detailCode+" "+text,
		"user_inactive",
		"token_invalidated",
		"token_revoked",
		"invalidated token",
		"revoked token",
		"invalid token",
		"invalid api key",
		"invalid_api_key",
		"validate api key failed",
		"authentication failed",
		"unauthorized",
	) {
		return openAIUpstreamErrorAuth
	}

	if containsAnyOpenAIErrorText(code+" "+errType+" "+text,
		"usage_limit_reached",
		"rate_limit_exceeded",
		"rate limit",
		"too many requests",
		"usage limit",
		"no available image quota",
	) {
		return openAIUpstreamErrorRateLimit
	}

	if containsAnyOpenAIErrorText(text,
		"precharge",
		"pre-charge",
		"预扣费",
		"insufficient_quota",
		"insufficient quota",
		"insufficient balance",
		"insufficient account balance",
		"payment required",
		"billing issue",
		"quota exhausted",
		"usage quota exhausted",
		"deactivated_workspace",
	) {
		return openAIUpstreamErrorBilling
	}

	if containsAnyOpenAIErrorText(text,
		"channel affinity disabled",
		"channel is disabled",
		"channel disabled",
		"渠道被禁用",
		"model/group not supported",
		"model group not supported",
		"model not support",
		"group not support",
		"group not enabled",
		"not enabled for this group",
		"image generation is not enabled for this group",
		"endpoint not supported",
		"endpoint unsupported",
	) {
		return openAIUpstreamErrorBusiness
	}

	if containsAnyOpenAIErrorText(text,
		"upstream access forbidden",
		"access forbidden",
		"permission denied",
	) {
		return openAIUpstreamErrorForbidden
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return openAIUpstreamErrorAuth
	case http.StatusPaymentRequired:
		return openAIUpstreamErrorBilling
	case http.StatusForbidden:
		return openAIUpstreamErrorForbidden
	case http.StatusTooManyRequests:
		return openAIUpstreamErrorRateLimit
	case 529:
		return openAIUpstreamErrorTransient
	}
	if statusCode >= 500 {
		return openAIUpstreamErrorTransient
	}
	if isOpenAITransientProcessingError(statusCode, upstreamMsg, upstreamBody) {
		return openAIUpstreamErrorTransient
	}
	if containsAnyOpenAIErrorText(text,
		"connection reset by peer",
		"http2:",
		"client connection force closed",
		"clientconn.close",
		"stream data interval timeout",
		"dial tcp",
		"i/o timeout",
		"context deadline exceeded",
		"upstream connection error",
	) {
		return openAIUpstreamErrorTransient
	}
	return openAIUpstreamErrorUnknown
}

// isOpenAIEndpointMigrationError identifies the deterministic 410/endpoint
// migration response emitted by panel-backed OpenAI-compatible providers.
// Matching the provider's explicit wording avoids treating arbitrary 410s or
// user request validation errors as account failures.
func isOpenAIEndpointMigrationError(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if statusCode < http.StatusBadRequest {
		return false
	}
	text := normalizeOpenAIUpstreamErrorText(upstreamMsg, upstreamBody)
	if text == "" {
		return false
	}
	return containsAnyOpenAIErrorText(text,
		"the api endpoint is not served from the panel domain",
		"api endpoint is not served from the panel domain",
		"please use the published api endpoint",
		"use the published api endpoint",
		"endpoint is not served from the panel domain",
		"api endpoint has moved",
		"endpoint has moved",
		"endpoint migration required",
	)
}

func isCustomOpenAITransientProcessingError(upstreamStatusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if upstreamStatusCode != http.StatusBadRequest && upstreamStatusCode != http.StatusServiceUnavailable {
		return false
	}
	text := normalizeOpenAIUpstreamErrorText(upstreamMsg, upstreamBody)
	matchesTransientText := strings.Contains(text, "our servers are currently overloaded") ||
		strings.Contains(text, "temporarily unavailable") && strings.Contains(text, "try again") ||
		strings.Contains(text, "please try again later") && strings.Contains(text, "overloaded")
	if !matchesTransientText || upstreamStatusCode == http.StatusServiceUnavailable {
		return matchesTransientText
	}

	// A bare HTTP 400 message is ambiguous and may be a business validation
	// error. Require an upstream server classification before treating it as a
	// global transient error. Account-specific temporary rules still run in the
	// real gateway request path and can deliberately match message-only 400s.
	code := strings.ToLower(strings.TrimSpace(extractUpstreamErrorCode(upstreamBody)))
	errType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "error.type").String()))
	if errType == "" {
		errType = strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "response.error.type").String()))
	}
	return code == "server_is_overloaded" || code == "slow_down" ||
		errType == "server_error" || errType == "api_error"
}

func isCustomOpenAIContextWindowError(upstreamMsg string, upstreamBody []byte) bool {
	parts := []string{upstreamMsg}
	if len(upstreamBody) > 0 {
		for _, path := range []string{
			"response.error.code",
			"error.code",
			"response.error.type",
			"error.type",
			"response.error.message",
			"error.message",
		} {
			if value := strings.TrimSpace(gjson.GetBytes(upstreamBody, path).String()); value != "" {
				parts = append(parts, value)
			}
		}
		raw := string(upstreamBody)
		if len(raw) > 4096 {
			raw = raw[:4096]
		}
		parts = append(parts, raw)
	}
	text := strings.ToLower(strings.TrimSpace(strings.Join(parts, " ")))
	if text == "" {
		return false
	}
	text = strings.NewReplacer("_", " ", "-", " ", "\n", " ", "\r", " ", "\t", " ").Replace(text)
	text = strings.Join(strings.Fields(text), " ")
	return containsAnyOpenAIErrorText(text,
		"context window",
		"context length",
		"maximum context",
		"max context",
		"context window exceeded",
		"context limit",
		"input exceeds",
		"exceeds the context",
		"request exceeds",
		"too many input tokens",
		"too many tokens",
		"input is too long",
		"input too large",
		"request too large",
		"token limit exceeded",
		"tokens exceed",
		"maximum number of tokens",
		"prompt is too long",
		"prompt too long",
	)
}

func isOpenAIGroupDisabledUpstreamError(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if statusCode < 400 {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(extractUpstreamErrorCode(upstreamBody)))
	text := normalizeOpenAIUpstreamErrorText(upstreamMsg, upstreamBody)
	return containsAnyOpenAIErrorText(code+" "+text,
		"group_disabled",
		"group disabled",
		"api key 所属分组已停用",
		"所属分组已停用",
		"分组已停用",
	)
}

func normalizeOpenAIUpstreamErrorText(upstreamMsg string, upstreamBody []byte) string {
	parts := []string{upstreamMsg}
	if len(upstreamBody) > 0 {
		if msg := extractUpstreamErrorMessage(upstreamBody); strings.TrimSpace(msg) != "" {
			parts = append(parts, msg)
		}
		parts = append(parts, string(upstreamBody))
	}
	joined := strings.ToLower(strings.Join(parts, " "))
	joined = strings.NewReplacer("_", " ", "-", " ", "\n", " ", "\r", " ", "\t", " ").Replace(joined)
	return strings.Join(strings.Fields(joined), " ")
}

func containsAnyOpenAIErrorText(text string, needles ...string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}

func openAIUpstreamErrorClassShouldFailover(class openAIUpstreamErrorClass) bool {
	switch class {
	case openAIUpstreamErrorAuth, openAIUpstreamErrorBilling, openAIUpstreamErrorForbidden, openAIUpstreamErrorRateLimit, openAIUpstreamErrorTransient, openAIUpstreamErrorModelUnsupported:
		return true
	default:
		return false
	}
}

func openAIUpstreamErrorClassImmediateFailover(class openAIUpstreamErrorClass) bool {
	switch class {
	case openAIUpstreamErrorBilling, openAIUpstreamErrorForbidden:
		return true
	default:
		return false
	}
}

func openAIUpstreamErrorClassSchedulerCategory(class openAIUpstreamErrorClass) string {
	switch class {
	case openAIUpstreamErrorAuth:
		return "auth"
	case openAIUpstreamErrorBilling:
		return "balance"
	case openAIUpstreamErrorForbidden:
		return "forbidden"
	case openAIUpstreamErrorRateLimit:
		return "rate_limit"
	case openAIUpstreamErrorTransient:
		return "transient"
	case openAIUpstreamErrorModelUnsupported:
		return "model_unsupported"
	case openAIUpstreamErrorBusiness:
		return "business"
	default:
		return "unknown"
	}
}
