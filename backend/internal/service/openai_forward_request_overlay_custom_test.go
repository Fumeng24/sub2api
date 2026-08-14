package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func decodeOpenAIOverlayTestBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	return decoded
}

func TestApplyOpenAIDisabledImageToolsOverlay(t *testing.T) {
	t.Run("removes implicit image tool and keeps regular tools", func(t *testing.T) {
		body := []byte(`{"model":"gpt-5.4","tools":[{"type":"function","name":"shell"},{"type":"image_generation"}]}`)
		decoded := decodeOpenAIOverlayTestBody(t, body)
		modified := false

		err := applyOpenAIDisabledImageToolsOverlay(body, false, func() (map[string]any, error) {
			return decoded, nil
		}, func() {
			modified = true
		})

		require.NoError(t, err)
		require.True(t, modified)
		tools, ok := decoded["tools"].([]any)
		require.True(t, ok)
		require.Len(t, tools, 1)
		require.Equal(t, "function", tools[0].(map[string]any)["type"])
	})

	t.Run("does not decode explicit image tool", func(t *testing.T) {
		body := []byte(`{"model":"gpt-5.4","tools":[{"type":"image_generation","model":"gpt-image-2"}]}`)
		decoded := false

		err := applyOpenAIDisabledImageToolsOverlay(body, false, func() (map[string]any, error) {
			decoded = true
			return nil, nil
		}, func() {})

		require.NoError(t, err)
		require.False(t, decoded)
	})
}

func TestApplyOpenAIForwardCompatibilityOverlay(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.5",
		"previous_response_id":" ",
		"input":[{
			"type":"message",
			"status":"completed",
			"content":[{"type":"input_text","text":"hello","status":"nested-kept"}]
		}]
	}`)
	decoded := decodeOpenAIOverlayTestBody(t, body)
	modified := false

	err := applyOpenAIForwardCompatibilityOverlay(
		body,
		func() (map[string]any, error) { return decoded, nil },
		func(path string) { delete(decoded, path) },
		func() { modified = true },
	)

	require.NoError(t, err)
	require.True(t, modified)
	require.NotContains(t, decoded, "previous_response_id")
	item := decoded["input"].([]any)[0].(map[string]any)
	require.NotContains(t, item, "status")
	content := item["content"].([]any)[0].(map[string]any)
	require.Equal(t, "nested-kept", content["status"])
}

func TestApplyOpenAIForwardCompatibilityOverlayPreservesValidContinuation(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","previous_response_id":"resp_keep","input":"hello"}`)
	deleteCalled := false
	decodeCalled := false

	err := applyOpenAIForwardCompatibilityOverlay(
		body,
		func() (map[string]any, error) {
			decodeCalled = true
			return nil, nil
		},
		func(string) { deleteCalled = true },
		func() {},
	)

	require.NoError(t, err)
	require.False(t, deleteCalled)
	require.False(t, decodeCalled)
}
