package handler

import (
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type customAvailableGroup struct {
	ID                              int64                `json:"id"`
	Name                            string               `json:"name"`
	Platform                        string               `json:"platform"`
	SubscriptionType                string               `json:"subscription_type"`
	RateMultiplier                  float64              `json:"rate_multiplier"`
	PeakRateEnabled                 bool                 `json:"peak_rate_enabled"`
	PeakStart                       string               `json:"peak_start"`
	PeakEnd                         string               `json:"peak_end"`
	PeakRateMultiplier              float64              `json:"peak_rate_multiplier"`
	GroupRateDiscountMultiplier     *float64             `json:"group_rate_discount_multiplier,omitempty"`
	DiscountedRateMultiplier        *float64             `json:"discounted_rate_multiplier,omitempty"`
	GroupRateDiscountName           *string              `json:"group_rate_discount_name,omitempty"`
	GroupRateDiscountScheduleMode   *string              `json:"group_rate_discount_schedule_mode,omitempty"`
	GroupRateDiscountStartAt        *string              `json:"group_rate_discount_start_at,omitempty"`
	GroupRateDiscountEndAt          *string              `json:"group_rate_discount_end_at,omitempty"`
	GroupRateDiscountWeekdays       []int                `json:"group_rate_discount_weekdays,omitempty"`
	GroupRateDiscountDailyStartTime *string              `json:"group_rate_discount_daily_start_time,omitempty"`
	GroupRateDiscountDailyEndTime   *string              `json:"group_rate_discount_daily_end_time,omitempty"`
	GroupRateDiscountTimezone       *string              `json:"group_rate_discount_timezone,omitempty"`
	IsExclusive                     bool                 `json:"is_exclusive"`
	SupportedModels                 []userSupportedModel `json:"supported_models"`
	modelsListConfig                service.GroupModelsListConfig
}

type customPlatformSection struct {
	Platform               string                 `json:"platform"`
	Endpoints              []string               `json:"endpoints"`
	SupportedEndpointTypes []string               `json:"supported_endpoint_types"`
	Groups                 []customAvailableGroup `json:"groups"`
	SupportedModels        []userSupportedModel   `json:"supported_models"`
}

type customAvailableChannel struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Platforms   []customPlatformSection `json:"platforms"`
}

// buildCustomPlatformSections owns the site-specific channel presentation.
func buildCustomPlatformSections(
	ch service.AvailableChannel,
	visibleGroups []customAvailableGroup,
) []customPlatformSection {
	groupsByPlatform := make(map[string][]customAvailableGroup, 4)
	for _, g := range visibleGroups {
		if g.Platform == "" {
			continue
		}
		groupsByPlatform[g.Platform] = append(groupsByPlatform[g.Platform], g)
	}
	if len(groupsByPlatform) == 0 {
		return nil
	}

	platforms := make([]string, 0, len(groupsByPlatform))
	for platform := range groupsByPlatform {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)

	sections := make([]customPlatformSection, 0, len(platforms))
	for _, platform := range platforms {
		groups := groupsByPlatform[platform]
		for i := range groups {
			groups[i].SupportedModels = supportedModelsForGroup(ch.SupportedModels, groups[i])
		}
		sections = append(sections, customPlatformSection{
			Platform:               platform,
			Endpoints:              endpointLabelsForPlatform(platform),
			SupportedEndpointTypes: supportedEndpointTypesForPlatform(platform),
			Groups:                 groups,
			SupportedModels:        supportedModelsForGroups(groups),
		})
	}
	return sections
}

func (h *AvailableChannelHandler) listAvailableChannelsCustom(c *gin.Context, userID int64) bool {
	if !h.featureEnabled(c) {
		response.Success(c, []customAvailableChannel{})
		return true
	}

	userGroups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return true
	}
	allowedGroupModels := make(map[int64]service.GroupModelsListConfig, len(userGroups))
	effectiveRates := make(map[int64]float64, len(userGroups))
	for i := range userGroups {
		allowedGroupModels[userGroups[i].ID] = userGroups[i].ModelsListConfig
		effectiveRates[userGroups[i].ID] = userGroups[i].RateMultiplier
	}

	channels, err := h.channelService.ListAvailable(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return true
	}

	out := make([]customAvailableChannel, 0, len(channels))
	discount := activeAvailableChannelDiscountCustom(h, c)
	for _, ch := range channels {
		if ch.Status != service.StatusActive {
			continue
		}
		visibleGroups := filterUserVisibleGroupsCustom(ch.Groups, allowedGroupModels, effectiveRates, discount)
		if len(visibleGroups) == 0 {
			continue
		}
		sections := buildCustomPlatformSections(ch, visibleGroups)
		if len(sections) == 0 {
			continue
		}
		out = append(out, customAvailableChannel{Name: ch.Name, Description: ch.Description, Platforms: sections})
	}

	response.Success(c, out)
	return true
}

func activeAvailableChannelDiscountCustom(h *AvailableChannelHandler, c *gin.Context) *service.ActiveGroupRateDiscount {
	if h == nil || h.settingService == nil {
		return nil
	}
	return h.settingService.ActiveGroupRateDiscount(c.Request.Context(), time.Now())
}

// ListPublic exposes the site's public model and CNY pricing catalogue without
// mixing that local endpoint into the upstream authenticated handler.
func (h *AvailableChannelHandler) ListPublic(c *gin.Context) {
	if !h.featureEnabled(c) {
		response.Success(c, []customAvailableChannel{})
		return
	}

	channels, err := h.channelService.ListAvailable(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	discount := activeAvailableChannelDiscountCustom(h, c)
	out := make([]customAvailableChannel, 0, len(channels))
	for _, ch := range channels {
		if ch.Status != service.StatusActive {
			continue
		}
		visibleGroups := filterPublicVisibleGroups(ch.Groups, discount)
		if len(visibleGroups) == 0 {
			continue
		}
		sections := buildCustomPlatformSections(ch, visibleGroups)
		if len(sections) == 0 {
			continue
		}
		out = append(out, customAvailableChannel{
			Name:        ch.Name,
			Description: ch.Description,
			Platforms:   sections,
		})
	}
	response.Success(c, out)
}

func endpointLabelsForPlatform(platform string) []string {
	switch platform {
	case "openai":
		return []string{"OpenAI Chat", "Responses", "Images"}
	case "anthropic":
		return []string{"Anthropic Messages"}
	case "gemini", "antigravity":
		return []string{"Gemini API"}
	default:
		return []string{"OpenAI Compatible"}
	}
}

func supportedEndpointTypesForPlatform(platform string) []string {
	switch platform {
	case "openai":
		return []string{"openai", "responses", "images"}
	case "anthropic":
		return []string{"anthropic"}
	case "gemini", "antigravity":
		return []string{"gemini"}
	default:
		return []string{"openai"}
	}
}

func supportedModelsForGroup(channelModels []service.SupportedModel, group customAvailableGroup) []userSupportedModel {
	cfg := group.modelsListConfig
	if !cfg.Enabled || len(cfg.Models) == 0 {
		return nil
	}

	channelUserModels := toUserSupportedModels(channelModels, map[string]struct{}{group.Platform: {}})
	pricingByName := make(map[string]userSupportedModel, len(channelUserModels))
	for _, model := range channelUserModels {
		pricingByName[strings.ToLower(model.Name)] = model
	}

	out := make([]userSupportedModel, 0, len(cfg.Models))
	seen := make(map[string]struct{}, len(cfg.Models))
	for _, name := range cfg.Models {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if model, ok := pricingByName[key]; ok {
			out = append(out, model)
		} else {
			out = append(out, userSupportedModel{Name: name, Platform: group.Platform})
		}
	}
	return out
}

func supportedModelsForGroups(groups []customAvailableGroup) []userSupportedModel {
	seen := make(map[string]struct{})
	out := make([]userSupportedModel, 0)
	for _, group := range groups {
		for _, model := range group.SupportedModels {
			key := strings.ToLower(model.Platform + ":" + model.Name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, model)
		}
	}
	return out
}

func filterUserVisibleGroupsCustom(
	groups []service.AvailableGroupRef,
	allowedGroupModels map[int64]service.GroupModelsListConfig,
	effectiveRates map[int64]float64,
	discount *service.ActiveGroupRateDiscount,
) []customAvailableGroup {
	allowed := make(map[int64]struct{}, len(allowedGroupModels))
	for id := range allowedGroupModels {
		allowed[id] = struct{}{}
	}

	visible := make([]customAvailableGroup, 0, len(groups))
	for _, group := range groups {
		if _, ok := allowed[group.ID]; !ok {
			continue
		}
		rate := group.RateMultiplier
		if effectiveRate, ok := effectiveRates[group.ID]; ok {
			rate = effectiveRate
		}
		visible = append(visible, customAvailableGroup{
			ID: group.ID, Name: group.Name, Platform: group.Platform,
			SubscriptionType: group.SubscriptionType, RateMultiplier: rate,
			PeakRateEnabled: group.PeakRateEnabled, PeakStart: group.PeakStart,
			PeakEnd: group.PeakEnd, PeakRateMultiplier: group.PeakRateMultiplier,
			IsExclusive: group.IsExclusive,
		})
	}
	for i := range visible {
		visible[i].modelsListConfig = allowedGroupModels[visible[i].ID]
		applyActiveGroupDiscount(&visible[i], discount)
	}
	return visible
}

func filterPublicVisibleGroups(groups []service.AvailableGroupRef, discount *service.ActiveGroupRateDiscount) []customAvailableGroup {
	visible := make([]customAvailableGroup, 0, len(groups))
	for _, g := range groups {
		if g.IsExclusive || g.SubscriptionType == service.SubscriptionTypeSubscription ||
			!g.ModelsListConfig.Enabled || len(g.ModelsListConfig.Models) == 0 {
			continue
		}
		group := customAvailableGroup{
			ID:                 g.ID,
			Name:               g.Name,
			Platform:           g.Platform,
			SubscriptionType:   g.SubscriptionType,
			RateMultiplier:     g.RateMultiplier,
			PeakRateEnabled:    g.PeakRateEnabled,
			PeakStart:          g.PeakStart,
			PeakEnd:            g.PeakEnd,
			PeakRateMultiplier: g.PeakRateMultiplier,
			IsExclusive:        g.IsExclusive,
			modelsListConfig:   g.ModelsListConfig,
		}
		applyActiveGroupDiscount(&group, discount)
		visible = append(visible, group)
	}
	return visible
}

func applyActiveGroupDiscount(group *customAvailableGroup, discount *service.ActiveGroupRateDiscount) {
	if group == nil || discount == nil || !discount.AppliesToGroup(group.ID) {
		return
	}
	multiplier := discount.DiscountMultiplier
	discounted := group.RateMultiplier * multiplier
	group.GroupRateDiscountMultiplier = &multiplier
	group.DiscountedRateMultiplier = &discounted
	group.GroupRateDiscountName = &discount.Name
	group.GroupRateDiscountScheduleMode = &discount.ScheduleMode
	group.GroupRateDiscountStartAt = &discount.StartAt
	group.GroupRateDiscountEndAt = &discount.EndAt
	group.GroupRateDiscountWeekdays = append([]int(nil), discount.Weekdays...)
	group.GroupRateDiscountDailyStartTime = &discount.DailyStartTime
	group.GroupRateDiscountDailyEndTime = &discount.DailyEndTime
	group.GroupRateDiscountTimezone = &discount.Timezone
}
