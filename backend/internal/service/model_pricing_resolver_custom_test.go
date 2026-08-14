//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetIntervalPricing_AnthropicCacheWriteDerives1hPrice(t *testing.T) {
	bs := newTestBillingServiceForResolver()
	r := NewModelPricingResolver(&ChannelService{}, bs)

	resolved := &ResolvedPricing{
		Mode:        BillingModeToken,
		BasePricing: &ModelPricing{},
		channelPricing: &ChannelModelPricing{
			Platform: "anthropic",
		},
		Intervals: []PricingInterval{
			{MinTokens: 0, CacheWritePrice: testPtrFloat64(10e-6)},
		},
	}

	result := r.GetIntervalPricing(resolved, 50000)
	require.NotNil(t, result)
	require.True(t, result.SupportsCacheBreakdown)
	require.InDelta(t, 10e-6, result.CacheCreationPricePerToken, 1e-12)
	require.InDelta(t, 10e-6, result.CacheCreation5mPrice, 1e-12)
	require.InDelta(t, 16e-6, result.CacheCreation1hPrice, 1e-12)
}
