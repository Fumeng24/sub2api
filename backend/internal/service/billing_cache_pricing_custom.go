package service

import "strings"

const anthropicCacheCreation1hTo5mRatio = 1.6

func isAnthropicPricingPlatform(platform string) bool {
	return strings.EqualFold(strings.TrimSpace(platform), PlatformAnthropic)
}

func isAnthropicPricingModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "claude-")
}

func deriveChannelCacheCreation1hPrice(cacheWritePrice, base5mPrice, base1hPrice float64, platform, model string) float64 {
	if cacheWritePrice <= 0 {
		return cacheWritePrice
	}
	if base5mPrice > 0 && base1hPrice > base5mPrice {
		return cacheWritePrice * (base1hPrice / base5mPrice)
	}
	if isAnthropicPricingPlatform(platform) || isAnthropicPricingModel(model) {
		return cacheWritePrice * anthropicCacheCreation1hTo5mRatio
	}
	return cacheWritePrice
}

func applyChannelCacheWritePrice(pricing *ModelPricing, cacheWritePrice float64, platform, model string) {
	if pricing == nil {
		return
	}
	base5mPrice := pricing.CacheCreation5mPrice
	base1hPrice := pricing.CacheCreation1hPrice
	pricing.CacheCreationPricePerToken = cacheWritePrice
	pricing.CacheCreationPricePerTokenPriority = cacheWritePrice
	pricing.CacheCreationPriceExplicit = true
	pricing.CacheCreation5mPrice = cacheWritePrice
	pricing.CacheCreation1hPrice = deriveChannelCacheCreation1hPrice(cacheWritePrice, base5mPrice, base1hPrice, platform, model)
	if pricing.SupportsCacheBreakdown || isAnthropicPricingPlatform(platform) || isAnthropicPricingModel(model) {
		pricing.SupportsCacheBreakdown = true
	}
}
