package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

func isOpenAIThinkingSignatureInvalidError(payload []byte, message string) bool {
	parts := []string{message}
	if len(payload) > 0 {
		for _, path := range []string{
			"response.error.code",
			"error.code",
			"response.error.type",
			"error.type",
			"response.error.message",
			"error.message",
		} {
			if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
				parts = append(parts, value)
			}
		}
		raw := string(payload)
		if len(raw) > 4096 {
			raw = raw[:4096]
		}
		parts = append(parts, raw)
	}
	combined := strings.ToLower(strings.TrimSpace(strings.Join(parts, " ")))
	if strings.Contains(combined, "thinking_signature_invalid") {
		return true
	}
	return strings.Contains(combined, "signature") &&
		(strings.Contains(combined, "thinking") || strings.Contains(combined, "thought_signature"))
}
