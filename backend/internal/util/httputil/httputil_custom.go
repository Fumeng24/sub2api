package httputil

import (
	"net/http"
	"strings"
)

var customCloudflareChallengeMarkers = []string{
	"error code: 1010",
	"error code 1010",
	"error code: 1020",
	"error code 1020",
	"browser's signature",
	"access denied",
}

func isCloudflareChallengeResponseCustom(headers http.Header, body []byte) bool {
	preview := strings.ToLower(TruncateBody(body, 4096))
	for _, marker := range customCloudflareChallengeMarkers {
		if strings.Contains(preview, marker) {
			return true
		}
	}

	if headers == nil || strings.TrimSpace(headers.Get("cf-ray")) == "" {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(headers.Get("content-type")))
	return strings.Contains(contentType, "text/html") &&
		(strings.Contains(preview, "cloudflare") || strings.Contains(preview, "error code"))
}
