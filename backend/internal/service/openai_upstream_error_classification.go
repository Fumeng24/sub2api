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
		"upstream access forbidden",
		"access forbidden",
		"permission denied",
	) {
		return openAIUpstreamErrorForbidden
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
	case openAIUpstreamErrorAuth, openAIUpstreamErrorBilling, openAIUpstreamErrorRateLimit, openAIUpstreamErrorTransient, openAIUpstreamErrorModelUnsupported:
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
