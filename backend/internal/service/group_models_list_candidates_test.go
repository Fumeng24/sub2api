//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupModelsListCandidatesFromAccounts_UsesBoundAccountMappings(t *testing.T) {
	got := groupModelsListCandidatesFromAccounts(PlatformGemini, []Account{
		{
			Platform: PlatformGemini,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3.1-flash-image": "gemini-3.1-flash-image",
					"gemini-3.1-pro-preview": "gemini-3.1-pro-preview",
				},
			},
		},
		{
			Platform: PlatformGemini,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3.1-flash-image": "gemini-3.1-flash-image",
					"gemini-*":               "gemini-3.1-flash-image",
				},
			},
		},
		{
			Platform: PlatformOpenAI,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"},
			},
		},
	})

	require.Equal(t, []string{"gemini-3.1-flash-image", "gemini-3.1-pro-preview"}, got)
}
