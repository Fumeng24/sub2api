package service

import (
	"net/http"
	"strings"
)

var upstreamModelUnsupportedKeywordsCustom = []string{
	"model not supported",
	"model unsupported",
	"model is not supported",
	"requested model is not supported",
	"no available channel for model",
}

// IsUpstreamModelUnsupportedError identifies account/model capacity errors so
// callers can retry another account without declaring the group unavailable.
func IsUpstreamModelUnsupportedError(statusCode int, body []byte) bool {
	return isUpstreamModelNotFoundError(statusCode, body)
}

func isUpstreamModelNotFoundErrorCustom(statusCode int, body []byte) (bool, bool) {
	if statusCode == http.StatusNotFound {
		normalized := normalizeModelNotFoundBody(body)
		matched := normalized != "" && strings.Contains(normalized, "model") && containsModelUnsupportedKeywordCustom(normalized)
		return matched, matched
	}
	switch statusCode {
	case http.StatusBadRequest, http.StatusForbidden, http.StatusServiceUnavailable:
	default:
		return false, true
	}
	normalized := normalizeModelNotFoundBody(body)
	if normalized == "" || !strings.Contains(normalized, "model") {
		return false, true
	}
	return containsModelNotFoundKeyword(normalized) || containsModelUnsupportedKeywordCustom(normalized), true
}

func containsModelUnsupportedKeywordCustom(normalizedBody string) bool {
	for _, keyword := range upstreamModelUnsupportedKeywordsCustom {
		if strings.Contains(normalizedBody, keyword) {
			return true
		}
	}
	return false
}
