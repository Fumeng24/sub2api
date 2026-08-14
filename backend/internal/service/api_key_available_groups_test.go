package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveAvailableGroupModelsListConfig_EnabledImageListKeepsConfiguredModels(t *testing.T) {
	configured := []string{"gpt-image-1", "gpt-image-2"}
	got := resolveAvailableGroupModelsListConfig(
		Group{
			Platform:             PlatformOpenAI,
			AllowImageGeneration: true,
			ModelsListConfig: GroupModelsListConfig{
				Enabled: true,
				Models:  configured,
			},
		},
		[]Account{{
			Platform: PlatformOpenAI,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-image-2": "gpt-image-2"},
			},
		}},
	)

	require.True(t, got.Enabled)
	require.Equal(t, configured, got.Models)
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
		Name:                 "Grok生图",
		Platform:             PlatformGrok,
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
