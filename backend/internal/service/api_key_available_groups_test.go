package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterImageModelsSupportedByAccounts_GeminiMappingKeepsOnlySupportedImages(t *testing.T) {
	models := []string{
		"gemini-2.5-pro",
		"gemini-2.5-flash-image",
		"gemini-3.1-flash-image",
		"gemini-3.1-flash-image",
	}
	accounts := []Account{
		{
			Platform: PlatformGemini,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3.1-flash-image": "gemini-3.1-flash-image",
				},
			},
		},
	}

	got := filterImageModelsSupportedByAccounts(models, PlatformGemini, accounts)

	require.Equal(t, []string{"gemini-3.1-flash-image"}, got)
}

func TestFilterImageModelsSupportedByAccounts_OpenAIMappingKeepsOnlySupportedImages(t *testing.T) {
	models := []string{
		"gpt-5",
		"gpt-image-1",
		"gpt-image-2",
		"codex-gpt-image-2",
	}
	accounts := []Account{
		{
			Platform: PlatformOpenAI,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-image-2": "gpt-image-2",
				},
			},
		},
	}

	got := filterImageModelsSupportedByAccounts(models, PlatformOpenAI, accounts)

	require.Equal(t, []string{"gpt-image-2"}, got)
}

func TestFilterImageModelsSupportedByAccounts_EmptyMappingPreservesConfiguredImages(t *testing.T) {
	models := []string{"gpt-5", "gpt-image-1", "gpt-image-2"}
	accounts := []Account{{Platform: PlatformOpenAI}}

	got := filterImageModelsSupportedByAccounts(models, PlatformOpenAI, accounts)

	require.Equal(t, []string{"gpt-image-1", "gpt-image-2"}, got)
}

func TestIsUserVisibleImageGenerationGroup(t *testing.T) {
	require.True(t, isUserVisibleImageGenerationGroup(&Group{
		Name:                 "GPT生图",
		Platform:             PlatformOpenAI,
		AllowImageGeneration: true,
	}))
	require.True(t, isUserVisibleImageGenerationGroup(&Group{
		Name:                 "Gemini生图",
		Platform:             PlatformGemini,
		AllowImageGeneration: true,
	}))
	require.True(t, isUserVisibleImageGenerationGroup(&Group{
		Name:                 "GPT普通",
		Platform:             PlatformOpenAI,
		AllowImageGeneration: true,
	}))
	require.False(t, isUserVisibleImageGenerationGroup(&Group{
		Name:                 "Claude生图",
		Platform:             PlatformAnthropic,
		AllowImageGeneration: true,
	}))
}
