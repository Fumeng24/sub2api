package service

import "net/http"

func IsUpstreamBillingExhaustionError(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if statusCode == http.StatusPaymentRequired {
		return true
	}
	text := normalizeOpenAIUpstreamErrorText(upstreamMsg, upstreamBody)
	return containsAnyOpenAIErrorText(text,
		"precharge",
		"pre charge",
		"预扣费",
		"insufficient_quota",
		"insufficient quota",
		"insufficient balance",
		"insufficient account balance",
		"insufficient credit balance",
		"credit balance",
		"payment required",
		"billing issue",
		"quota exhausted",
		"usage quota exhausted",
		"account balance is too low",
		"balance too low",
		"out of credits",
		"not enough credits",
		"deactivated_workspace",
		"workspace has been deactivated",
		"余额不足",
		"账户余额不足",
		"账号余额不足",
		"额度不足",
		"欠费",
	)
}

func isUpstreamBillingExhaustionError(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	return IsUpstreamBillingExhaustionError(statusCode, upstreamMsg, upstreamBody)
}
