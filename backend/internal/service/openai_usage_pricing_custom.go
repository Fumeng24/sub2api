package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func (s *OpenAIGatewayService) applyOpenAIUsageRatePolicyCustom(
	ctx context.Context,
	apiKey *APIKey,
	baseMultiplier float64,
	now time.Time,
) (float64, float64, float64) {
	discountMultiplier := 1.0
	if groupID, ok := resolveAPIKeyBillingGroupID(apiKey); ok {
		discountMultiplier = activeGroupRateDiscountMultiplierAt(ctx, s.settingService, groupID, now)
	}
	baseMultiplier *= discountMultiplier
	tokenMultiplier, imageMultiplier := computePeakAwareMultipliers(apiKey, baseMultiplier, now)
	videoMultiplier := resolveVideoRateMultiplier(apiKey, baseMultiplier, discountMultiplier)
	return tokenMultiplier, imageMultiplier, videoMultiplier
}

func openAIUsageServiceTierCustom(apiKey *APIKey, serviceTier string) string {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform == PlatformOpenAI && apiKey.Group.ForceOpenAIPriority {
		return ""
	}
	return serviceTier
}

func handleOpenAIUsageCostErrorCustom(
	input *OpenAIRecordUsageInput,
	result *OpenAIForwardResult,
	apiKey *APIKey,
	account *Account,
	billingModels []string,
	err error,
) (error, bool) {
	logger.L().With(
		zap.String("component", "service.openai_gateway"),
		zap.Strings("billing_models", billingModels),
		zap.String("requested_model", input.OriginalModel),
		zap.String("mapped_model", input.ChannelMappedModel),
		zap.String("upstream_model", result.UpstreamModel),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Int64("account_id", account.ID),
	).Warn("openai_usage.cost_calculation_failed", zap.Error(err))
	return err, true
}
