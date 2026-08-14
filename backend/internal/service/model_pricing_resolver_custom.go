package service

func firstChannelPricingModel(chPricing *ChannelModelPricing) string {
	if chPricing == nil || len(chPricing.Models) == 0 {
		return ""
	}
	return chPricing.Models[0]
}

func intervalToModelPricingCustom(
	iv *PricingInterval,
	supportsCacheBreakdown bool,
	chPricing *ChannelModelPricing,
	basePricing *ModelPricing,
) (*ModelPricing, bool) {
	if iv == nil || iv.CacheWritePrice == nil {
		return nil, false
	}
	pricing := intervalToModelPricing(iv, supportsCacheBreakdown, chPricing)
	if basePricing != nil {
		pricing.CacheCreation5mPrice = basePricing.CacheCreation5mPrice
		pricing.CacheCreation1hPrice = basePricing.CacheCreation1hPrice
	}
	platform := ""
	model := ""
	if chPricing != nil {
		platform = chPricing.Platform
		model = firstChannelPricingModel(chPricing)
	}
	applyChannelCacheWritePrice(pricing, *iv.CacheWritePrice, platform, model)
	return pricing, true
}
