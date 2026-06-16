package service

import (
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
	if !gjson.ValidBytes(body) {
		original := strings.TrimSpace(string(body))
		if original == "" {
			return body
		}
		safe := ClientFacingErrorMessage(statusCode, errType, original)
		if safe == original {
			return body
		}
		return []byte(safe)
	}

	out := body
	patchStringMessage := func(path, typePath string) {
		value := gjson.GetBytes(out, path)
		if !value.Exists() || value.Type != gjson.String {
			return
		}
		localType := errType
		if typePath != "" {
			if t := strings.TrimSpace(gjson.GetBytes(out, typePath).String()); t != "" {
				localType = t
			}
		}
		original := value.String()
		safe := ClientFacingErrorMessage(statusCode, localType, original)
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
