package service

import (
	"net/http"
	"strings"

	coderws "github.com/coder/websocket"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func sanitizeOpenAIWSV2ClientPayload(msgType coderws.MessageType, payload []byte) []byte {
	if msgType != coderws.MessageText || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return payload
	}
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	switch eventType {
	case "error":
		errType := strings.TrimSpace(gjson.GetBytes(payload, "error.type").String())
		if errType == "" {
			errType = "upstream_error"
		}
		out := append([]byte(nil), payload...)
		if next, err := sjson.SetBytes(out, "error.message", ClientFacingErrorMessage(http.StatusBadGateway, errType, gjson.GetBytes(out, "error.message").String())); err == nil {
			return next
		}
		return out
	case "response.failed":
		out := payload
		if sanitized, ok := sanitizeOpenAIResponseFailedEventForClient(payload, eventType, true); ok {
			out = sanitized
		}
		if next, err := sjson.SetBytes(out, "response.error.message", ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", gjson.GetBytes(out, "response.error.message").String())); err == nil {
			out = next
		}
		if next, err := sjson.SetBytes(out, "error.message", ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", gjson.GetBytes(out, "error.message").String())); err == nil {
			out = next
		}
		if next, err := sjson.SetBytes(out, "message", ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", gjson.GetBytes(out, "message").String())); err == nil {
			out = next
		}
		return out
	default:
		return payload
	}
}
