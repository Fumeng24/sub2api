//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

func TestFilterUserVisibleGroups_EmbedsActiveDiscount(t *testing.T) {
	groups := []service.AvailableGroupRef{
		{ID: 1, Name: "g1", Platform: "anthropic", RateMultiplier: 2},
		{ID: 2, Name: "g2", Platform: "openai", RateMultiplier: 3},
	}
	allowed := map[int64]service.GroupModelsListConfig{1: {}, 2: {}}
	discount := &service.ActiveGroupRateDiscount{
		Name:               "Promo",
		DiscountMultiplier: 0.5,
		ScheduleMode:       "weekly",
		StartAt:            "2026-01-01T00:00:00Z",
		EndAt:              "2026-01-02T00:00:00Z",
		Weekdays:           []int{1, 2, 3},
		DailyStartTime:     "09:00",
		DailyEndTime:       "18:00",
		Timezone:           "Asia/Shanghai",
		GroupIDs:           []int64{2},
	}

	visible := filterUserVisibleGroupsCustom(groups, allowed, nil, discount)
	require.Len(t, visible, 2)
	require.Nil(t, visible[0].GroupRateDiscountMultiplier)
	require.NotNil(t, visible[1].GroupRateDiscountMultiplier)
	require.Equal(t, 0.5, *visible[1].GroupRateDiscountMultiplier)
	require.Equal(t, 1.5, *visible[1].DiscountedRateMultiplier)
	require.Equal(t, "Promo", *visible[1].GroupRateDiscountName)
	require.Equal(t, "weekly", *visible[1].GroupRateDiscountScheduleMode)
	require.Equal(t, []int{1, 2, 3}, visible[1].GroupRateDiscountWeekdays)
	require.Equal(t, "09:00", *visible[1].GroupRateDiscountDailyStartTime)
	require.Equal(t, "18:00", *visible[1].GroupRateDiscountDailyEndTime)
	require.Equal(t, "Asia/Shanghai", *visible[1].GroupRateDiscountTimezone)
}

func TestFilterUserVisibleGroups_UsesAllowedGroupModelsConfig(t *testing.T) {
	// 用户可见分组来自 APIKeyService，那里可能已经按权限/账号可用性过滤过模型。
	// available-channels 必须使用这份过滤后的配置，而不是 ChannelService.ListAvailable
	// 里活跃分组的原始配置。
	groups := []service.AvailableGroupRef{
		{
			ID:       1,
			Name:     "image",
			Platform: "openai",
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"too-wide-model"},
			},
		},
	}
	allowed := map[int64]service.GroupModelsListConfig{
		1: {Enabled: true, Models: []string{"filtered-model"}},
	}

	visible := filterUserVisibleGroupsCustom(groups, allowed, nil, nil)
	require.Len(t, visible, 1)
	require.Equal(t, []string{"filtered-model"}, visible[0].modelsListConfig.Models)
}

func TestFilterUserVisibleGroups_UsesEffectiveUserRateInsteadOfGroupBaseRate(t *testing.T) {
	groups := []service.AvailableGroupRef{{
		ID:             1,
		Name:           "custom-rate",
		Platform:       "openai",
		RateMultiplier: 0.6,
	}}
	allowed := map[int64]service.GroupModelsListConfig{1: {}}

	visible := filterUserVisibleGroupsCustom(groups, allowed, map[int64]float64{1: 0.25}, nil)
	require.Len(t, visible, 1)
	require.Equal(t, 0.25, visible[0].RateMultiplier)
}

func TestFilterPublicVisibleGroups_OnlyPublicStandardWithModelList(t *testing.T) {
	groups := []service.AvailableGroupRef{
		{
			ID:               1,
			Name:             "public",
			Platform:         "openai",
			SubscriptionType: service.SubscriptionTypeStandard,
			ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.4"}},
		},
		{
			ID:               2,
			Name:             "exclusive",
			Platform:         "openai",
			SubscriptionType: service.SubscriptionTypeStandard,
			IsExclusive:      true,
			ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.4"}},
		},
		{
			ID:               3,
			Name:             "subscription",
			Platform:         "anthropic",
			SubscriptionType: service.SubscriptionTypeSubscription,
			ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"claude-sonnet-4-6"}},
		},
		{
			ID:               4,
			Name:             "no-model-list",
			Platform:         "gemini",
			SubscriptionType: service.SubscriptionTypeStandard,
			ModelsListConfig: service.GroupModelsListConfig{Enabled: false, Models: []string{"gemini-pro"}},
		},
	}

	visible := filterPublicVisibleGroups(groups, nil)

	require.Len(t, visible, 1)
	require.Equal(t, int64(1), visible[0].ID)
	require.Equal(t, []string{"gpt-5.4"}, visible[0].modelsListConfig.Models)
}

func TestBuildPlatformSections_GroupSupportedModelsUseCustomList(t *testing.T) {
	// 同一个渠道、同一个平台下两个分组模型清单不同：用户侧必须按分组展示，
	// 不能把平台聚合模型无脑塞给每个分组。
	ch := service.AvailableChannel{
		Name: "ch",
		SupportedModels: []service.SupportedModel{
			{
				Name:     "claude-sonnet-4-6",
				Platform: "anthropic",
				Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
			},
			{
				Name:     "claude-opus-4-6",
				Platform: "anthropic",
				Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
			},
			{Name: "gpt-5.4", Platform: "openai"},
		},
	}
	visible := []customAvailableGroup{
		{
			ID:       1,
			Name:     "Claude Pro",
			Platform: "anthropic",
			modelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"claude-sonnet-4-6", "claude-opus-4-6"},
			},
		},
		{
			ID:       2,
			Name:     "Claude Lite",
			Platform: "anthropic",
			modelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"claude-haiku-4-5"},
			},
		},
	}

	sections := buildCustomPlatformSections(ch, visible)
	require.Len(t, sections, 1)
	require.Equal(t, []string{"Anthropic Messages"}, sections[0].Endpoints)
	require.Equal(t, []string{"anthropic"}, sections[0].SupportedEndpointTypes)
	require.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6", "claude-haiku-4-5"}, sectionModelNames(sections[0]))
	require.Len(t, sections[0].Groups, 2)
	require.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6"}, groupModelNames(sections[0].Groups[0]))
	require.Equal(t, []string{"claude-haiku-4-5"}, groupModelNames(sections[0].Groups[1]))
	require.NotNil(t, sections[0].Groups[0].SupportedModels[0].Pricing)
	require.Nil(t, sections[0].Groups[1].SupportedModels[0].Pricing)
}

func TestBuildPlatformSections_GroupWithoutCustomListShowsNoModels(t *testing.T) {
	ch := service.AvailableChannel{
		Name: "ch",
		SupportedModels: []service.SupportedModel{
			{Name: "gpt-5.4", Platform: "openai"},
			{Name: "claude-sonnet-4-6", Platform: "anthropic"},
		},
	}
	visible := []customAvailableGroup{{ID: 1, Name: "OpenAI", Platform: "openai"}}

	sections := buildCustomPlatformSections(ch, visible)
	require.Len(t, sections, 1)
	require.Empty(t, groupModelNames(sections[0].Groups[0]))
}

func TestBuildPlatformSections_EnabledEmptyCustomListDoesNotFallback(t *testing.T) {
	ch := service.AvailableChannel{
		Name: "ch",
		SupportedModels: []service.SupportedModel{
			{Name: "gpt-5.4", Platform: "openai"},
		},
	}
	visible := []customAvailableGroup{{
		ID:       1,
		Name:     "OpenAI",
		Platform: "openai",
		modelsListConfig: service.GroupModelsListConfig{
			Enabled: true,
			Models:  nil,
		},
	}}

	sections := buildCustomPlatformSections(ch, visible)
	require.Len(t, sections, 1)
	require.Empty(t, sections[0].Groups[0].SupportedModels)
}

func groupModelNames(group customAvailableGroup) []string {
	names := make([]string, 0, len(group.SupportedModels))
	for _, model := range group.SupportedModels {
		names = append(names, model.Name)
	}
	return names
}

func sectionModelNames(section customPlatformSection) []string {
	names := make([]string, 0, len(section.SupportedModels))
	for _, model := range section.SupportedModels {
		names = append(names, model.Name)
	}
	return names
}

func TestCustomAvailableChannelJSONContract(t *testing.T) {
	discount := 0.8
	payload := customAvailableChannel{
		Name: "channel",
		Platforms: []customPlatformSection{{
			Platform:               "openai",
			Endpoints:              []string{"Responses"},
			SupportedEndpointTypes: []string{"responses"},
			Groups: []customAvailableGroup{{
				ID: 1, Name: "Codex", Platform: "openai",
				GroupRateDiscountMultiplier: &discount,
				SupportedModels:             []userSupportedModel{{Name: "gpt-5.6", Platform: "openai"}},
			}},
		}},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"name":"channel",
		"description":"",
		"platforms":[{
			"platform":"openai",
			"endpoints":["Responses"],
			"supported_endpoint_types":["responses"],
			"groups":[{
				"id":1,"name":"Codex","platform":"openai","subscription_type":"",
				"rate_multiplier":0,"peak_rate_enabled":false,"peak_start":"","peak_end":"",
				"peak_rate_multiplier":0,"group_rate_discount_multiplier":0.8,
				"is_exclusive":false,
				"supported_models":[{"name":"gpt-5.6","platform":"openai","pricing":null}]
			}],
			"supported_models":null
		}]
	}`, string(body))
}
