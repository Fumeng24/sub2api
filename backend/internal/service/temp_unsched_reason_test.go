package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempUnschedulableReasonDetailsFromRaw(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		wantReason     string
		wantStatusCode *int
	}{
		{
			name:           "matched keyword wins",
			raw:            `{"status_code":502,"matched_keyword":"openai_transient_5xx","reason":"fallback","error_message":"body"}`,
			wantReason:     "openai_transient_5xx",
			wantStatusCode: intPtrForTempUnschedReasonTest(502),
		},
		{
			name:           "reason fallback",
			raw:            `{"status_code":503,"reason":"openai_request_error","error_message":"body"}`,
			wantReason:     "openai_request_error",
			wantStatusCode: intPtrForTempUnschedReasonTest(503),
		},
		{
			name:           "network status code fallback",
			raw:            `{"status_code":0,"error_message":"unknown"}`,
			wantReason:     tempUnschedNetworkOrStreamInterruption,
			wantStatusCode: intPtrForTempUnschedReasonTest(0),
		},
		{
			name:           "error message fallback",
			raw:            `{"status_code":401,"error_message":"missing refresh token"}`,
			wantReason:     "missing refresh token",
			wantStatusCode: intPtrForTempUnschedReasonTest(401),
		},
		{
			name:       "legacy plain text",
			raw:        "token refresh failed: missing refresh token",
			wantReason: "token refresh failed: missing refresh token",
		},
		{
			name:       "legacy unknown is suppressed",
			raw:        "unknown",
			wantReason: "",
		},
		{
			name:       "long plain text is truncated",
			raw:        strings.Repeat("a", 600),
			wantReason: strings.Repeat("a", 512),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TempUnschedulableReasonDetailsFromRaw(tt.raw)
			require.Equal(t, tt.wantReason, got.DisplayReason)
			if tt.wantStatusCode == nil {
				require.Nil(t, got.StatusCode)
				return
			}
			require.NotNil(t, got.StatusCode)
			require.Equal(t, *tt.wantStatusCode, *got.StatusCode)
		})
	}
}

func intPtrForTempUnschedReasonTest(v int) *int {
	return &v
}

func TestEnrichTempUnschedStateFromRaw(t *testing.T) {
	state := &TempUnschedState{}
	enrichTempUnschedStateFromRaw(state, `{"status_code":503,"reason":"upstream_unavailable"}`)
	require.Equal(t, 503, state.StatusCode)
	require.Equal(t, "upstream_unavailable", state.MatchedKeyword)

	state = &TempUnschedState{MatchedKeyword: "existing"}
	enrichTempUnschedStateFromRaw(state, `{"status_code":502,"reason":"replacement"}`)
	require.Equal(t, 502, state.StatusCode)
	require.Equal(t, "existing", state.MatchedKeyword)
}
