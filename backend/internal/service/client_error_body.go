package service

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ClientFacingErrorBody sanitizes common JSON error message fields while
// preserving the upstream response shape. This covers passthrough paths that
// return an upstream error body directly instead of using writeClient*Error.
func ClientFacingErrorBody(statusCode int, errType string, body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	normalized := NormalizeUpstreamClientError(statusCode, errType, ExtractUpstreamErrorMessage(body))
	statusCode = normalized.Status
	errType = normalized.Type
	message := normalized.Message
	normalizedSensitiveUpstream := message == clientFacingTemporaryUnavailableMessage
	if !gjson.ValidBytes(body) {
		original := strings.TrimSpace(string(body))
		if original == "" {
			return body
		}
		safe := message
		if !normalizedSensitiveUpstream {
			safe = ClientFacingErrorMessage(statusCode, errType, original)
		}
		if safe == original {
			return body
		}
		return []byte(safe)
	}

	out := body
	if normalizedSensitiveUpstream {
		patchString := func(path, value string) {
			if !gjson.GetBytes(out, path).Exists() {
				return
			}
			if next, err := sjson.SetBytes(out, path, value); err == nil {
				out = next
			}
		}
		patchNumber := func(path string, value int) {
			if !gjson.GetBytes(out, path).Exists() {
				return
			}
			if next, err := sjson.SetBytes(out, path, value); err == nil {
				out = next
			}
		}
		statusText := http.StatusText(statusCode)
		patchString("error.type", errType)
		patchString("response.error.type", errType)
		patchString("type", errType)
		patchString("error.status", statusText)
		patchString("response.error.status", statusText)
		patchString("status", statusText)
		patchNumber("error.code", statusCode)
		patchNumber("response.error.code", statusCode)
		patchNumber("code", statusCode)
	}
	patchStringMessage := func(path, typePath string) {
		value := gjson.GetBytes(out, path)
		if !value.Exists() || value.Type != gjson.String {
			return
		}
		localType := errType
		if typePath != "" && !normalizedSensitiveUpstream {
			if t := strings.TrimSpace(gjson.GetBytes(out, typePath).String()); t != "" {
				localType = t
			}
		}
		original := value.String()
		safe := message
		if !normalizedSensitiveUpstream {
			safe = ClientFacingErrorMessage(statusCode, localType, original)
		}
		if safe == original {
			return
		}
		if next, err := sjson.SetBytes(out, path, safe); err == nil {
			out = next
		}
	}

	patchStringMessage("error.message", "error.type")
	patchStringMessage("response.error.message", "response.error.type")
	patchStringMessage("message", "type")
	patchStringMessage("error", "")

	return out
}
