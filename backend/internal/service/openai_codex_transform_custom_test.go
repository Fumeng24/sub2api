package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestStripCodexSparkImageGenerationToolFromRawPayload_TransformFixture(t *testing.T) {
	payload := []byte(`{"model":"gpt-5.3-codex-spark","tools":[{"type":"image_generation"},{"type":"web_search"}],"input":"hello"}`)

	stripped, changed, err := stripCodexSparkImageGenerationToolFromRawPayload(payload, "gpt-5.3-codex-spark")
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(stripped, `tools.#(type=="image_generation")`).Exists())
	require.True(t, gjson.GetBytes(stripped, `tools.#(type=="web_search")`).Exists())
}

func TestEnsureOpenAIResponsesImageGenerationToolChoiceAuto(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.4",
		"tools": []any{
			map[string]any{"type": "image_generation"},
		},
	}

	require.True(t, ensureOpenAIResponsesImageGenerationToolChoiceAuto(reqBody))
	require.Equal(t, "auto", reqBody["tool_choice"])
	require.False(t, ensureOpenAIResponsesImageGenerationToolChoiceAuto(reqBody))
}

func TestEnsureOpenAIResponsesImageGenerationToolChoiceAuto_SkipsSpark(t *testing.T) {
	reqBody := map[string]any{
		"model": "gpt-5.3-codex-spark",
		"tools": []any{
			map[string]any{"type": "image_generation"},
		},
	}

	require.False(t, ensureOpenAIResponsesImageGenerationToolChoiceAuto(reqBody))
	_, exists := reqBody["tool_choice"]
	require.False(t, exists)
}
