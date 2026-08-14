//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func siteModelMappingAccount(platform string, mapping map[string]any) *Account {
	return &Account{
		Platform: platform,
		Credentials: map[string]any{
			"model_mapping": mapping,
		},
	}
}

func TestAccountIsModelSupported_SiteStrictAliases(t *testing.T) {
	tests := []struct {
		name      string
		platform  string
		mapping   map[string]any
		requested string
		want      bool
	}{
		{
			name:      "anthropic short haiku alias matches dated mapping",
			platform:  PlatformAnthropic,
			mapping:   map[string]any{"claude-haiku-4-5-20251001": "claude-haiku-4-5-20251001"},
			requested: "claude-haiku-4-5",
			want:      true,
		},
		{
			name:      "anthropic dated haiku alias matches short mapping",
			platform:  PlatformAnthropic,
			mapping:   map[string]any{"claude-haiku-4-5": "claude-haiku-4-5-20251001"},
			requested: "claude-haiku-4-5-20251001",
			want:      true,
		},
		{
			name:      "openai legacy alias does not match configured model",
			platform:  PlatformOpenAI,
			mapping:   map[string]any{"gpt-5.5": "gpt-5.5"},
			requested: "gpt-5.2",
			want:      false,
		},
		{
			name:      "openai compact suffix does not match base model",
			platform:  PlatformOpenAI,
			mapping:   map[string]any{"gpt-5.5": "gpt-5.5"},
			requested: "gpt-5.5-openai-compact",
			want:      false,
		},
		{
			name:      "openai codex does not fall back to spark",
			platform:  PlatformOpenAI,
			mapping:   map[string]any{"gpt-5.3-codex-spark": "gpt-5.3-codex-spark"},
			requested: "gpt-5.3-codex",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, siteModelMappingAccount(tt.platform, tt.mapping).IsModelSupported(tt.requested))
		})
	}
}

func TestAccountGetMappedModel_SiteStrictAliases(t *testing.T) {
	tests := []struct {
		name      string
		platform  string
		mapping   map[string]any
		requested string
		want      string
	}{
		{
			name:      "anthropic short haiku alias resolves through dated mapping",
			platform:  PlatformAnthropic,
			mapping:   map[string]any{"claude-haiku-4-5-20251001": "claude-haiku-4-5-20251001"},
			requested: "claude-haiku-4-5",
			want:      "claude-haiku-4-5-20251001",
		},
		{
			name:      "anthropic dated haiku alias resolves through short mapping",
			platform:  PlatformAnthropic,
			mapping:   map[string]any{"claude-haiku-4-5": "claude-haiku-4-5-20251001"},
			requested: "claude-haiku-4-5-20251001",
			want:      "claude-haiku-4-5-20251001",
		},
		{
			name:      "openai legacy alias preserves unmatched request",
			platform:  PlatformOpenAI,
			mapping:   map[string]any{"gpt-5.5": "gpt-5.5"},
			requested: "gpt-5.2",
			want:      "gpt-5.2",
		},
		{
			name:      "openai namespaced compact suffix preserves unmatched request",
			platform:  PlatformOpenAI,
			mapping:   map[string]any{"gpt-5.5": "gpt-5.5"},
			requested: "openai/gpt-5.5-openai-compact",
			want:      "openai/gpt-5.5-openai-compact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, siteModelMappingAccount(tt.platform, tt.mapping).GetMappedModel(tt.requested))
		})
	}
}

func TestAccountResolveMappedModel_SiteStrictOpenAIModels(t *testing.T) {
	for _, requested := range []string{"gpt-5.2", "gpt-5.5-openai-compact"} {
		t.Run(requested, func(t *testing.T) {
			model, matched := siteModelMappingAccount(
				PlatformOpenAI,
				map[string]any{"gpt-5.5": "gpt-5.5"},
			).ResolveMappedModel(requested)
			require.Equal(t, requested, model)
			require.False(t, matched)
		})
	}
}
