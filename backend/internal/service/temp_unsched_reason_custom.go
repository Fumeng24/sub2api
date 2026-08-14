package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const tempUnschedNetworkOrStreamInterruption = "network_or_stream_interruption"

// TempUnschedulableReasonDetails is the display-safe view of a stored temp unschedulable reason.
type TempUnschedulableReasonDetails struct {
	DisplayReason string
	StatusCode    *int
}

// TempUnschedulableReasonDetailsFromRaw parses current and legacy temp unschedulable reason formats.
func TempUnschedulableReasonDetailsFromRaw(raw string) TempUnschedulableReasonDetails {
	raw = strings.TrimSpace(raw)
	if isUninformativeTempUnschedReason(raw) {
		return TempUnschedulableReasonDetails{}
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || len(payload) == 0 {
		return TempUnschedulableReasonDetails{DisplayReason: truncateString(raw, 512)}
	}

	statusCode, hasStatusCode := tempUnschedStatusCode(payload["status_code"])
	details := TempUnschedulableReasonDetails{}
	if hasStatusCode {
		details.StatusCode = &statusCode
	}

	for _, key := range []string{"matched_keyword", "reason"} {
		if value := tempUnschedReasonString(payload[key]); value != "" {
			details.DisplayReason = truncateString(value, 512)
			return details
		}
	}

	if hasStatusCode && statusCode == 0 {
		details.DisplayReason = tempUnschedNetworkOrStreamInterruption
		return details
	}

	if value := tempUnschedReasonString(payload["error_message"]); value != "" {
		details.DisplayReason = truncateString(value, 512)
		return details
	}
	if hasStatusCode {
		details.DisplayReason = fmt.Sprintf("HTTP %d", statusCode)
	}
	return details
}

// TempUnschedulableDisplayReasonFromRaw returns a display-safe reason string.
func TempUnschedulableDisplayReasonFromRaw(raw string) string {
	return TempUnschedulableReasonDetailsFromRaw(raw).DisplayReason
}

func enrichTempUnschedStateFromRaw(state *TempUnschedState, raw string) {
	if state == nil {
		return
	}
	details := TempUnschedulableReasonDetailsFromRaw(raw)
	if state.MatchedKeyword == "" {
		state.MatchedKeyword = details.DisplayReason
	}
	if details.StatusCode != nil {
		state.StatusCode = *details.StatusCode
	}
}

func tempUnschedReasonString(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	s = strings.TrimSpace(s)
	if isUninformativeTempUnschedReason(s) {
		return ""
	}
	return s
}

func isUninformativeTempUnschedReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "", "unknown", "<nil>", "null":
		return true
	default:
		return false
	}
}

func tempUnschedStatusCode(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case json.Number:
		n, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		s := strings.TrimSpace(typed)
		if s == "" {
			return 0, false
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
