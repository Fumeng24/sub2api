package web

import (
	"net/http"
	"strings"
)

// Frontend routing only owns browser navigation. Methods other than GET/HEAD
// must continue to Gin so API aliases never receive the SPA HTML fallback.
func shouldBypassEmbeddedFrontendRequest(method, path string) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return true
	}
	return shouldBypassEmbeddedFrontend(path)
}

func shouldBypassEmbeddedFrontend(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/api/") ||
		strings.HasPrefix(trimmed, "/v1/") ||
		strings.HasPrefix(trimmed, "/v1beta/") ||
		strings.HasPrefix(trimmed, "/backend-api/") ||
		strings.HasPrefix(trimmed, "/antigravity/") ||
		strings.HasPrefix(trimmed, "/setup/") ||
		trimmed == "/health" ||
		trimmed == "/models" ||
		trimmed == "/responses" ||
		strings.HasPrefix(trimmed, "/responses/") ||
		trimmed == "/alpha/search" ||
		trimmed == "/chat/completions" ||
		trimmed == "/embeddings" ||
		trimmed == "/messages/count_tokens" ||
		trimmed == "/live" ||
		strings.HasPrefix(trimmed, "/live/") ||
		trimmed == "/sub2api/billing" ||
		strings.HasPrefix(trimmed, "/images/") ||
		strings.HasPrefix(trimmed, "/videos/")
}
