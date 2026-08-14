//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func newAvailableChannelServiceWithPricing(channels []Channel, groupRepo GroupRepository, pricing *PricingService) *ChannelService {
	repo := &mockChannelRepository{
		listAllFn: func(ctx context.Context) ([]Channel, error) { return channels, nil },
	}
	return NewChannelService(repo, groupRepo, nil, pricing)
}

func TestListAvailable_CarriesGroupModelsListConfig(t *testing.T) {
	channels := []Channel{{
		ID:       1,
		Name:     "chA",
		Status:   StatusActive,
		GroupIDs: []int64{1},
	}}
	groupRepo := &stubGroupRepoForAvailable{
		activeGroups: []Group{{
			ID:       1,
			Name:     "Claude Pro",
			Platform: PlatformAnthropic,
			ModelsListConfig: GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"claude-sonnet-4-6", "claude-opus-4-6"},
			},
		}},
	}
	svc := newAvailableChannelService(channels, groupRepo)

	out, err := svc.ListAvailable(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].Groups, 1)
	require.True(t, out[0].Groups[0].ModelsListConfig.Enabled)
	require.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6"}, out[0].Groups[0].ModelsListConfig.Models)
}

func TestListAvailable_GroupWhitelistModelsUseGlobalPricingFallback(t *testing.T) {
	channels := []Channel{{
		ID:       1,
		Name:     "OpenAI",
		Status:   StatusActive,
		GroupIDs: []int64{1},
	}}
	groupRepo := &stubGroupRepoForAvailable{
		activeGroups: []Group{{
			ID:       1,
			Name:     "GPT",
			Platform: PlatformOpenAI,
			ModelsListConfig: GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"},
			},
		}},
	}
	pricingSvc := newStubPricingServiceFromMap(map[string]*LiteLLMModelPricing{
		"gpt-5.6-sol": {
			Mode:                    "chat",
			InputCostPerToken:       5e-6,
			OutputCostPerToken:      3e-5,
			CacheReadInputTokenCost: 5e-7,
		},
		"gpt-5.6-terra": {
			Mode:                    "chat",
			InputCostPerToken:       5e-6,
			OutputCostPerToken:      3e-5,
			CacheReadInputTokenCost: 5e-7,
		},
		"gpt-5.6-luna": {
			Mode:                    "chat",
			InputCostPerToken:       5e-6,
			OutputCostPerToken:      3e-5,
			CacheReadInputTokenCost: 5e-7,
		},
	})
	svc := newAvailableChannelServiceWithPricing(channels, groupRepo, pricingSvc)

	out, err := svc.ListAvailable(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)

	byName := make(map[string]SupportedModel, len(out[0].SupportedModels))
	for _, model := range out[0].SupportedModels {
		byName[model.Name] = model
	}
	for _, name := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		model, ok := byName[name]
		require.Truef(t, ok, "%s should be included from group whitelist", name)
		require.Equal(t, PlatformOpenAI, model.Platform)
		require.NotNil(t, model.Pricing)
		require.NotNil(t, model.Pricing.InputPrice)
		require.InDelta(t, 5e-6, *model.Pricing.InputPrice, 1e-12)
		require.NotNil(t, model.Pricing.OutputPrice)
		require.InDelta(t, 3e-5, *model.Pricing.OutputPrice, 1e-12)
	}
}

func TestResolveAvailableGroupModelsListConfig_UsesAccountModelsWhenGroupListDisabled(t *testing.T) {
	got := resolveAvailableGroupModelsListConfig(
		Group{Platform: PlatformAnthropic},
		[]Account{
			{
				Platform: PlatformAnthropic,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"claude-opus-4-6":   "claude-opus-4-6",
						"claude-sonnet-4-6": "claude-sonnet-4-6",
					},
				},
			},
			{
				Platform: PlatformOpenAI,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"},
				},
			},
		},
	)

	require.True(t, got.Enabled)
	require.Equal(t, []string{"claude-opus-4-6", "claude-sonnet-4-6"}, got.Models)
}

func TestResolveAvailableGroupModelsListConfig_EnabledListRemainsStableWhenAccountsOmitModel(t *testing.T) {
	got := resolveAvailableGroupModelsListConfig(
		Group{
			Platform: PlatformAnthropic,
			ModelsListConfig: GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"claude-sonnet-4-6", "claude-opus-4-6", "claude-haiku-4-5"},
			},
		},
		[]Account{
			{
				Platform: PlatformAnthropic,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"claude-sonnet-4-6": "claude-sonnet-4-6",
						"claude-haiku-4-5":  "claude-haiku-4-5",
					},
				},
			},
		},
	)

	require.True(t, got.Enabled)
	require.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6", "claude-haiku-4-5"}, got.Models)
}

func TestResolveAvailableGroupModelsListConfig_EnabledListRemainsVisibleWithoutSchedulableAccounts(t *testing.T) {
	got := resolveAvailableGroupModelsListConfig(
		Group{
			Platform:         PlatformAnthropic,
			ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"claude-sonnet-4-6"}},
		},
		nil,
	)

	require.True(t, got.Enabled)
	require.Equal(t, []string{"claude-sonnet-4-6"}, got.Models)
}

func TestResolveAvailableGroupModelsListConfig_UnenumerableAccountsKeepManualList(t *testing.T) {
	got := resolveAvailableGroupModelsListConfig(
		Group{
			Platform:         PlatformOpenAI,
			ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.4"}},
		},
		[]Account{{Platform: PlatformOpenAI}},
	)

	require.True(t, got.Enabled)
	require.Equal(t, []string{"gpt-5.4"}, got.Models)
}

func TestResolveAvailableGroupModelsListConfig_WildcardAccountModelsAreNotDisplayed(t *testing.T) {
	got := resolveAvailableGroupModelsListConfig(
		Group{Platform: PlatformAnthropic},
		[]Account{{
			Platform: PlatformAnthropic,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"claude-*": "claude-sonnet-4-6"},
			},
		}},
	)

	require.False(t, got.Enabled)
	require.Empty(t, got.Models)
}

func TestResolveAvailableGroupModelsListConfig_ImageGroupOnlyShowsImageAccountModels(t *testing.T) {
	got := resolveAvailableGroupModelsListConfig(
		Group{
			Name:                 "Gemini生图",
			Platform:             PlatformGemini,
			AllowImageGeneration: true,
		},
		[]Account{{
			Platform: PlatformGemini,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3.1-flash-image": "gemini-3.1-flash-image",
					"gemini-3.1-pro-preview": "gemini-3.1-pro-preview",
				},
			},
		}},
	)

	require.True(t, got.Enabled)
	require.Equal(t, []string{"gemini-3.1-flash-image"}, got.Models)
}
