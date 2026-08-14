package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func (s *openAIUserGroupRateRepoStub) GetRateConfigByUserAndGroup(_ context.Context, _, _ int64) (*UserGroupRateConfig, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.rate == nil {
		return nil, nil
	}
	return &UserGroupRateConfig{RateMultiplier: s.rate}, nil
}

func TestGatewayServiceRecordUsage_MissingPricingFailsClosed(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	userRepo := &openAIRecordUsageUserRepoStub{}
	subRepo := &openAIRecordUsageSubRepoStub{}
	quotaSvc := &openAIRecordUsageAPIKeyQuotaStub{}
	svc := &GatewayService{
		cfg:                 cfg,
		usageLogRepo:        usageRepo,
		userRepo:            userRepo,
		userSubRepo:         subRepo,
		billingService:      NewBillingService(cfg, nil),
		billingCacheService: &BillingCacheService{},
		usageBillingRepo:    billingRepo,
		deferredService:     &DeferredService{},
	}

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway_missing_pricing",
			Usage: ClaudeUsage{
				InputTokens:  1200,
				OutputTokens: 300,
			},
			Model:    "pricing-missing-gateway-model",
			Duration: time.Second,
		},
		APIKey:        &APIKey{ID: 1003, Quota: 100, Group: &Group{RateMultiplier: 1}},
		User:          &User{ID: 2003},
		Account:       &Account{ID: 3003},
		APIKeyService: quotaSvc,
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrModelPricingUnavailable)
	require.Equal(t, 0, billingRepo.calls)
	require.Equal(t, 0, usageRepo.calls)
	require.Equal(t, 0, userRepo.deductCalls)
	require.Equal(t, 0, subRepo.incrementCalls)
	require.Equal(t, 0, quotaSvc.quotaCalls)
	require.Equal(t, 0, quotaSvc.rateLimitCalls)
	require.Nil(t, usageRepo.lastLog)
	require.Nil(t, billingRepo.lastCmd)
}

func TestGatewayServiceValidateUsagePricingAvailable(t *testing.T) {
	cfg := &config.Config{}
	svc := &GatewayService{billingService: NewBillingService(cfg, nil)}

	err := svc.ValidateUsagePricingAvailable(context.Background(), &APIKey{}, "claude-sonnet-4", ChannelMappingResult{})
	require.NoError(t, err)

	err = svc.ValidateUsagePricingAvailable(context.Background(), &APIKey{}, "pricing-missing-gateway-model", ChannelMappingResult{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrModelPricingUnavailable)
}

func TestOpenAIGatewayServiceValidateUsagePricingAvailableUsesMappedModel(t *testing.T) {
	cfg := &config.Config{}
	svc := &OpenAIGatewayService{billingService: NewBillingService(cfg, nil)}

	err := svc.ValidateUsagePricingAvailable(context.Background(), &APIKey{}, "missing-alias", ChannelMappingResult{
		Mapped:             true,
		MappedModel:        "gpt-5.4",
		BillingModelSource: BillingModelSourceChannelMapped,
	})
	require.NoError(t, err)
}

func TestOpenAIGatewayServiceRecordUsage_ForceOpenAIPriorityBillsAsNormalTier(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	subRepo := &openAIRecordUsageSubRepoStub{}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, subRepo, nil)
	serviceTier := "priority"
	groupID := int64(24)
	usage := OpenAIUsage{InputTokens: 100, OutputTokens: 50}

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID:   "resp_force_priority_normal_billing",
			ServiceTier: &serviceTier,
			Usage:       usage,
			Model:       "gpt-5.4",
			Duration:    time.Second,
		},
		APIKey: &APIKey{
			ID:      1017,
			GroupID: &groupID,
			Group: &Group{
				ID:                  groupID,
				Platform:            PlatformOpenAI,
				RateMultiplier:      1,
				ForceOpenAIPriority: true,
			},
		},
		User:    &User{ID: 2017},
		Account: &Account{ID: 3017},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.ServiceTier)
	require.Equal(t, serviceTier, *usageRepo.lastLog.ServiceTier)

	baseCost, calcErr := svc.billingService.CalculateCost("gpt-5.4", UsageTokens{InputTokens: 100, OutputTokens: 50}, 1.0)
	require.NoError(t, calcErr)
	require.InDelta(t, baseCost.TotalCost, usageRepo.lastLog.TotalCost, 1e-10)
}

func TestOpenAIGatewayServiceRecordUsage_DoesNotMutateForwardResult(t *testing.T) {
	imagePrice4K := 0.44
	groupID := int64(1203)
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	result := &OpenAIForwardResult{
		RequestID:        "resp_image_output_size_immutable",
		Model:            "gpt-image-2",
		ImageCount:       1,
		ImageInputSize:   "1024x1024",
		ImageOutputSizes: []string{"3840x2160"},
		Duration:         time.Second,
	}

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: result,
		APIKey: &APIKey{
			ID:      11203,
			GroupID: i64p(groupID),
			Group: &Group{
				ID:             groupID,
				RateMultiplier: 1.0,
				ImagePrice4K:   &imagePrice4K,
			},
		},
		User:    &User{ID: 21203},
		Account: &Account{ID: 31203},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, ImageBillingSize4K, *usageRepo.lastLog.ImageSize)
	require.Empty(t, result.ImageSize)
	require.Empty(t, result.ImageOutputSize)
	require.Empty(t, result.ImageSizeSource)
	require.Nil(t, result.ImageSizeBreakdown)
}
