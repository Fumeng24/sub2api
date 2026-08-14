package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *GatewayService) applyGatewayUsageRatePolicyCustom(ctx context.Context, apiKey *APIKey, baseMultiplier float64, now time.Time) (float64, float64) {
	discountMultiplier := 1.0
	if groupID, ok := resolveAPIKeyBillingGroupID(apiKey); ok {
		discountMultiplier = activeGroupRateDiscountMultiplierAt(ctx, s.settingService, groupID, now)
	}
	baseMultiplier *= discountMultiplier
	return computePeakAwareMultipliers(apiKey, baseMultiplier, now)
}

func (s *GatewayService) calculateRecordUsageCostChecked(
	ctx context.Context,
	result *ForwardResult,
	apiKey *APIKey,
	billingModel string,
	multiplier float64,
	imageMultiplier float64,
	opts *recordUsageOpts,
) (*CostBreakdown, error) {
	if result.ImageCount > 0 {
		if resolved := s.resolveChannelPricing(ctx, billingModel, apiKey); resolved != nil && resolved.Mode == BillingModeToken {
			return s.calculateRecordUsageTokenCostChecked(ctx, result, apiKey, billingModel, multiplier, opts, resolved)
		}
		return s.calculateImageCost(ctx, result, apiKey, billingModel, imageMultiplier), nil
	}
	return s.calculateRecordUsageTokenCostChecked(ctx, result, apiKey, billingModel, multiplier, opts, s.resolveChannelPricing(ctx, billingModel, apiKey))
}

func (s *GatewayService) calculateRecordUsageCostCustom(
	ctx context.Context,
	result *ForwardResult,
	apiKey *APIKey,
	billingModel string,
	multiplier float64,
	imageMultiplier float64,
	opts *recordUsageOpts,
) (*CostBreakdown, error) {
	cost, err := s.calculateRecordUsageCostChecked(ctx, result, apiKey, billingModel, multiplier, imageMultiplier, opts)
	if err != nil {
		return nil, err
	}
	return cost, nil
}

func (s *GatewayService) calculateRecordUsageTokenCostChecked(
	ctx context.Context,
	result *ForwardResult,
	apiKey *APIKey,
	billingModel string,
	multiplier float64,
	opts *recordUsageOpts,
	resolved *ResolvedPricing,
) (*CostBreakdown, error) {
	tokens := UsageTokens{
		InputTokens:           result.Usage.InputTokens,
		OutputTokens:          result.Usage.OutputTokens,
		CacheCreationTokens:   result.Usage.CacheCreationInputTokens,
		CacheReadTokens:       result.Usage.CacheReadInputTokens,
		CacheCreation5mTokens: result.Usage.CacheCreation5mTokens,
		CacheCreation1hTokens: result.Usage.CacheCreation1hTokens,
		ImageOutputTokens:     result.Usage.ImageOutputTokens,
	}

	var cost *CostBreakdown
	var err error
	if resolved != nil && apiKey != nil && apiKey.Group != nil {
		gid := apiKey.Group.ID
		cost, err = s.billingService.CalculateCostUnified(CostInput{
			Ctx:            ctx,
			Model:          billingModel,
			GroupID:        &gid,
			Tokens:         tokens,
			RequestCount:   1,
			RateMultiplier: multiplier,
			Resolver:       s.resolver,
			Resolved:       resolved,
		})
	} else if opts != nil && opts.LongContextThreshold > 0 {
		cost, err = s.billingService.CalculateCostWithLongContext(billingModel, tokens, multiplier, opts.LongContextThreshold, opts.LongContextMultiplier)
	} else {
		cost, err = s.billingService.CalculateCost(billingModel, tokens, multiplier)
	}
	if err != nil {
		return nil, fmt.Errorf("calculate usage cost failed for model %s: %w", billingModel, err)
	}
	return cost, nil
}

func (s *GatewayService) ValidateUsagePricingAvailable(ctx context.Context, apiKey *APIKey, requestedModel string, mapping ChannelMappingResult) error {
	return validateUsagePricingAvailable(ctx, s.billingService, s.resolver, apiKey, usagePreflightBillingModelCandidates(requestedModel, mapping), "")
}

func (s *OpenAIGatewayService) ValidateUsagePricingAvailable(ctx context.Context, apiKey *APIKey, requestedModel string, mapping ChannelMappingResult) error {
	return validateUsagePricingAvailable(ctx, s.billingService, s.resolver, apiKey, usagePreflightBillingModelCandidates(requestedModel, mapping), "")
}

func validateUsagePricingAvailable(ctx context.Context, billingService *BillingService, resolver *ModelPricingResolver, apiKey *APIKey, candidates []string, serviceTier string) error {
	if billingService == nil {
		return nil
	}
	if len(candidates) == 0 {
		return fmt.Errorf("%w for model: empty", ErrModelPricingUnavailable)
	}

	var lastErr error
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		var err error
		if resolver != nil && apiKey != nil && apiKey.Group != nil {
			gid := apiKey.Group.ID
			_, err = billingService.CalculateCostUnified(CostInput{
				Ctx:            ctx,
				Model:          candidate,
				GroupID:        &gid,
				Tokens:         UsageTokens{},
				RequestCount:   1,
				RateMultiplier: 1,
				Resolver:       resolver,
			})
		} else {
			_, err = billingService.CalculateCostWithServiceTier(candidate, UsageTokens{}, 1, serviceTier)
		}
		if err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = ErrModelPricingUnavailable
	}
	return fmt.Errorf("usage pricing unavailable for models %s: %w", strings.Join(candidates, ","), lastErr)
}

func usagePreflightBillingModelCandidates(requestedModel string, mapping ChannelMappingResult) []string {
	requestedModel = strings.TrimSpace(requestedModel)
	mappedModel := strings.TrimSpace(mapping.MappedModel)
	candidates := make([]string, 0, 2)
	switch mapping.BillingModelSource {
	case BillingModelSourceRequested:
		candidates = append(candidates, requestedModel)
	case BillingModelSourceUpstream:
		if mappedModel != "" && mappedModel != requestedModel {
			candidates = append(candidates, mappedModel)
		}
		candidates = append(candidates, requestedModel)
	default:
		if mappedModel != "" {
			candidates = append(candidates, mappedModel)
		}
		candidates = append(candidates, requestedModel)
	}
	return uniqueNonEmptyStrings(candidates)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
