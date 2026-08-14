package service

import "strings"

// appendGroupModelPricingFallback adds group-whitelisted models to the public
// channel view and synthesizes display-only global pricing when needed.
func (s *ChannelService) appendGroupModelPricingFallback(models []SupportedModel, groups []AvailableGroupRef) []SupportedModel {
	type modelKey struct {
		platform string
		name     string
	}
	index := make(map[modelKey]int, len(models))
	for i := range models {
		index[modelKey{platform: models[i].Platform, name: strings.ToLower(models[i].Name)}] = i
	}
	for _, group := range groups {
		if group.Platform == "" || !group.ModelsListConfig.Enabled {
			continue
		}
		for _, rawName := range group.ModelsListConfig.Models {
			name := strings.TrimSpace(rawName)
			if name == "" {
				continue
			}
			key := modelKey{platform: group.Platform, name: strings.ToLower(name)}
			if idx, ok := index[key]; ok {
				if pricingNeedsFallback(models[idx].Pricing) {
					models[idx].Pricing = s.synthesizeGlobalPricing(name, models[idx].Pricing)
				}
				continue
			}
			models = append(models, SupportedModel{Name: name, Platform: group.Platform, Pricing: s.synthesizeGlobalPricing(name, nil)})
			index[key] = len(models) - 1
		}
	}
	return models
}

func (s *ChannelService) synthesizeGlobalPricing(model string, existing *ChannelModelPricing) *ChannelModelPricing {
	if s.pricingService == nil {
		return existing
	}
	pricing := s.pricingService.GetModelPricing(model)
	if pricing == nil {
		return existing
	}
	return synthesizePricingFromLiteLLM(pricing, existing)
}
