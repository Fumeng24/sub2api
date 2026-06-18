//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserAvailableChannel_Unauthenticated401(t *testing.T) {
	// 没有 AuthSubject 注入时，handler 应返回 401 且不触达 service 依赖。
	gin.SetMode(gin.TestMode)
	h := &AvailableChannelHandler{} // nil services — 401 路径不会调用它们
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/channels/available", nil)

	h.List(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFilterUserVisibleGroups_IntersectionOnly(t *testing.T) {
	// 渠道挂在 {g1, g2, g3}，用户只允许 {g1, g3} —— 响应必须仅含 g1/g3。
	groups := []service.AvailableGroupRef{
		{ID: 1, Name: "g1", Platform: "anthropic"},
		{ID: 2, Name: "g2", Platform: "anthropic"},
		{ID: 3, Name: "g3", Platform: "openai"},
	}
	allowed := map[int64]service.GroupModelsListConfig{1: {}, 3: {}}

	visible := filterUserVisibleGroups(groups, allowed, nil)
	require.Len(t, visible, 2)
	ids := []int64{visible[0].ID, visible[1].ID}
	require.ElementsMatch(t, []int64{1, 3}, ids)
}

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

	visible := filterUserVisibleGroups(groups, allowed, discount)
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

	visible := filterUserVisibleGroups(groups, allowed, nil)
	require.Len(t, visible, 1)
	require.Equal(t, []string{"filtered-model"}, visible[0].modelsListConfig.Models)
}

func TestToUserSupportedModels_FiltersByAllowedPlatforms(t *testing.T) {
	// 用户可访问分组只覆盖 anthropic；anthropic 平台的模型保留，openai 模型被剔除。
	src := []service.SupportedModel{
		{Name: "claude-sonnet-4-6", Platform: "anthropic", Pricing: nil},
		{Name: "gpt-4o", Platform: "openai", Pricing: nil},
	}
	allowed := map[string]struct{}{"anthropic": {}}
	out := toUserSupportedModels(src, allowed)
	require.Len(t, out, 1)
	require.Equal(t, "claude-sonnet-4-6", out[0].Name)
}

func TestToUserSupportedModels_NilAllowedPlatformsKeepsAll(t *testing.T) {
	// 显式传 nil allowedPlatforms 表示不做过滤。
	src := []service.SupportedModel{
		{Name: "a", Platform: "anthropic"},
		{Name: "b", Platform: "openai"},
	}
	require.Len(t, toUserSupportedModels(src, nil), 2)
}

func TestUserAvailableChannel_FieldWhitelist(t *testing.T) {
	// 通过序列化 userAvailableChannel 结构体验证响应形状：
	// 只有 name / description / platforms；不含管理端字段。
	row := userAvailableChannel{
		Name:        "ch",
		Description: "d",
		Platforms: []userChannelPlatformSection{
			{
				Platform:        "anthropic",
				Groups:          []userAvailableGroup{{ID: 1, Name: "g1", Platform: "anthropic"}},
				SupportedModels: []userSupportedModel{},
			},
		},
	}
	raw, err := json.Marshal(row)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{"id", "status", "billing_model_source", "restrict_models"} {
		_, exists := decoded[key]
		require.Falsef(t, exists, "user DTO must not expose %q", key)
	}
	for _, key := range []string{"name", "description", "platforms"} {
		_, exists := decoded[key]
		require.Truef(t, exists, "user DTO must expose %q", key)
	}

	// 验证 section 的字段（platform / groups / supported_models）。
	rawSection, err := json.Marshal(row.Platforms[0])
	require.NoError(t, err)
	var sectionDecoded map[string]any
	require.NoError(t, json.Unmarshal(rawSection, &sectionDecoded))
	for _, key := range []string{"platform", "groups", "supported_models"} {
		_, exists := sectionDecoded[key]
		require.Truef(t, exists, "platform section must expose %q", key)
	}

	// Group DTO 暴露区分专属/公开、订阅类型、默认倍率所需的字段，
	// 前端据此渲染 GroupBadge 并与 API 密钥页保持一致的视觉。
	rawGroup, err := json.Marshal(row.Platforms[0].Groups[0])
	require.NoError(t, err)
	var groupDecoded map[string]any
	require.NoError(t, json.Unmarshal(rawGroup, &groupDecoded))
	for _, key := range []string{"id", "name", "platform", "subscription_type", "rate_multiplier", "is_exclusive", "supported_models"} {
		_, exists := groupDecoded[key]
		require.Truef(t, exists, "group DTO must expose %q", key)
	}

	// pricing interval 白名单：不应暴露 id / sort_order。
	pricing := toUserPricing(&service.ChannelModelPricing{
		BillingMode: service.BillingModeToken,
		Intervals: []service.PricingInterval{
			{ID: 7, MinTokens: 0, MaxTokens: nil, SortOrder: 3},
		},
	})
	require.NotNil(t, pricing)
	require.Len(t, pricing.Intervals, 1)
	rawIv, err := json.Marshal(pricing.Intervals[0])
	require.NoError(t, err)
	var ivDecoded map[string]any
	require.NoError(t, json.Unmarshal(rawIv, &ivDecoded))
	for _, key := range []string{"id", "pricing_id", "sort_order"} {
		_, exists := ivDecoded[key]
		require.Falsef(t, exists, "user pricing interval must not expose %q", key)
	}
}

func TestBuildPlatformSections_GroupsByPlatform(t *testing.T) {
	// 一个渠道横跨 anthropic / openai / 空平台：应该生成 2 个 section，
	// 按 platform 字母序排序；未配置 /v1/models 清单的分组不展示模型。
	ch := service.AvailableChannel{
		Name: "ch",
		SupportedModels: []service.SupportedModel{
			{Name: "claude-sonnet-4-6", Platform: "anthropic"},
			{Name: "gpt-4o", Platform: "openai"},
		},
	}
	visible := []userAvailableGroup{
		{ID: 1, Name: "g-openai", Platform: "openai"},
		{ID: 2, Name: "g-ant", Platform: "anthropic"},
		{ID: 3, Name: "g-empty", Platform: ""},
	}
	sections := buildPlatformSections(ch, visible)
	require.Len(t, sections, 2)
	require.Equal(t, "anthropic", sections[0].Platform)
	require.Equal(t, "openai", sections[1].Platform)
	require.Len(t, sections[0].Groups, 1)
	require.Equal(t, int64(2), sections[0].Groups[0].ID)
	require.Empty(t, sections[0].SupportedModels)
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
	visible := []userAvailableGroup{
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

	sections := buildPlatformSections(ch, visible)
	require.Len(t, sections, 1)
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
	visible := []userAvailableGroup{{ID: 1, Name: "OpenAI", Platform: "openai"}}

	sections := buildPlatformSections(ch, visible)
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
	visible := []userAvailableGroup{{
		ID:       1,
		Name:     "OpenAI",
		Platform: "openai",
		modelsListConfig: service.GroupModelsListConfig{
			Enabled: true,
			Models:  nil,
		},
	}}

	sections := buildPlatformSections(ch, visible)
	require.Len(t, sections, 1)
	require.Empty(t, sections[0].Groups[0].SupportedModels)
}

func groupModelNames(group userAvailableGroup) []string {
	names := make([]string, 0, len(group.SupportedModels))
	for _, model := range group.SupportedModels {
		names = append(names, model.Name)
	}
	return names
}

func sectionModelNames(section userChannelPlatformSection) []string {
	names := make([]string, 0, len(section.SupportedModels))
	for _, model := range section.SupportedModels {
		names = append(names, model.Name)
	}
	return names
}
