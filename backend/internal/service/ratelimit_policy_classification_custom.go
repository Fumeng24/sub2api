package service

import "net/http"

func shouldBypassCustomErrorCodeSkip(account *Account, statusCode int, responseBody []byte) bool {
	if account == nil {
		return false
	}
	if account.Platform == PlatformOpenAI && isOpenAIImageQuotaRateLimitError(statusCode, extractUpstreamErrorMessage(responseBody), responseBody) {
		return false
	}
	if isUpstreamBillingExhaustionError(statusCode, extractUpstreamErrorMessage(responseBody), responseBody) {
		return true
	}
	if account.Platform != PlatformOpenAI {
		return false
	}
	switch statusCode {
	case http.StatusPaymentRequired, http.StatusForbidden:
		return true
	default:
		return isUpstreamModelNotFoundError(statusCode, responseBody)
	}
}

func isTransient5xxStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 524:
		return true
	default:
		return false
	}
}

func isOpenAI403ProbeCircuitError(account *Account, statusCode int, responseBody []byte) bool {
	return statusCode == http.StatusForbidden && isOpenAI403ProbeCircuitBody(account, responseBody)
}

func isOpenAI403ProbeCircuitBody(account *Account, responseBody []byte) bool {
	if account == nil || account.Platform != PlatformOpenAI {
		return false
	}
	return classifyOpenAIUpstreamError(http.StatusForbidden, extractUpstreamErrorMessage(responseBody), responseBody) == openAIUpstreamErrorForbidden
}

func isOpenAIImageQuotaRateLimitError(statusCode int, upstreamMsg string, body []byte) bool {
	if statusCode != http.StatusTooManyRequests {
		return false
	}
	return containsAnyOpenAIErrorText(normalizeOpenAIUpstreamErrorText(upstreamMsg, body), "no available image quota")
}
