package service

import (
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"testing"
)

func TestSanitizeOpenAIWSV2ClientPayload_ErrorEvent(t *testing.T) {
	payload := []byte(`{"type":"error","error":{"type":"server_error","message":"Upstream request failed for account 38850 via https://upstream.example"}}`)

	sanitized := sanitizeOpenAIWSV2ClientPayload(coderws.MessageText, payload)

	require.Equal(t, "error", gjson.GetBytes(sanitized, "type").String())
	require.Equal(t, ClientFacingTemporaryUnavailableMessage(), gjson.GetBytes(sanitized, "error.message").String())
	require.NotContains(t, string(sanitized), "38850")
	require.NotContains(t, string(sanitized), "https://upstream.example")
	require.NotContains(t, string(sanitized), "Upstream")
}

func TestSanitizeOpenAIWSV2ClientPayload_ResponseFailed(t *testing.T) {
	payload := []byte(`{"type":"response.failed","response":{"id":"resp_1","instructions":"secret","usage":{"input_tokens":10},"output":[{"type":"message","content":[{"type":"output_text","text":"hidden"}]}],"error":{"code":"server_error","message":"Upstream request failed for account 38850 via https://upstream.example"}}}`)

	sanitized := sanitizeOpenAIWSV2ClientPayload(coderws.MessageText, payload)

	require.Equal(t, "response.failed", gjson.GetBytes(sanitized, "type").String())
	require.Equal(t, ClientFacingTemporaryUnavailableMessage(), gjson.GetBytes(sanitized, "response.error.message").String())
	require.False(t, gjson.GetBytes(sanitized, "response.instructions").Exists())
	require.False(t, gjson.GetBytes(sanitized, "response.usage").Exists())
	require.False(t, gjson.GetBytes(sanitized, "response.output").Exists())
	require.NotContains(t, string(sanitized), "38850")
	require.NotContains(t, string(sanitized), "https://upstream.example")
	require.NotContains(t, string(sanitized), "secret")
}
