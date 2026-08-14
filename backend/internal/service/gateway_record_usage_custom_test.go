//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type openAIRecordUsageSettingRepoStub struct {
	SettingRepository
	values map[string]string
}

func (s *openAIRecordUsageSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func TestGatewayServiceRecordUsage_AppliesWeeklySelectedGroupDiscount(t *testing.T) {
	groupID := int64(11)
	groupRate := 1.4
	discount := 0.5
	usage := ClaudeUsage{InputTokens: 1000, OutputTokens: 500}

	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	subRepo := &openAIRecordUsageSubRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, subRepo)
	now := time.Now().In(time.Local)
	start := now.Add(-5 * time.Minute).Format("15:04")
	end := now.Add(5 * time.Minute).Format("15:04")
	svc.settingService = NewSettingService(&openAIRecordUsageSettingRepoStub{
		values: map[string]string{
			SettingKeyGroupRateDiscountSettings: `{"enabled":true,"name":"Weekly Promo","discount_multiplier":0.5,"schedule_mode":"weekly","weekdays":[1,2,3,4,5,6,7],"daily_start_time":"` + start + `","daily_end_time":"` + end + `","group_ids":[11]}`,
		},
	}, &config.Config{})

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway_weekly_group_rate_discount",
			Usage:     usage,
			Model:     "claude-sonnet-4",
			Duration:  time.Second,
		},
		APIKey: &APIKey{
			ID:      501,
			GroupID: i64p(groupID),
			Group: &Group{
				ID:             groupID,
				RateMultiplier: groupRate,
			},
		},
		User:    &User{ID: 601},
		Account: &Account{ID: 701},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.InDelta(t, groupRate*discount, usageRepo.lastLog.RateMultiplier, 1e-12)
	expected, err := svc.billingService.CalculateCost("claude-sonnet-4", UsageTokens{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
	}, groupRate*discount)
	require.NoError(t, err)
	require.InDelta(t, expected.ActualCost, usageRepo.lastLog.ActualCost, 1e-12)
	require.InDelta(t, expected.ActualCost, userRepo.lastAmount, 1e-12)
}

func TestGatewayServiceRecordUsage_DiscountsGroupIDWithoutHydratedGroup(t *testing.T) {
	groupID := int64(11)
	discount := 0.5
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})
	now := time.Now().In(time.Local)
	start := now.Add(-5 * time.Minute).Format("15:04")
	end := now.Add(5 * time.Minute).Format("15:04")
	svc.settingService = NewSettingService(&openAIRecordUsageSettingRepoStub{
		values: map[string]string{
			SettingKeyGroupRateDiscountSettings: `{"enabled":true,"name":"Weekly Promo","discount_multiplier":0.5,"schedule_mode":"weekly","weekdays":[1,2,3,4,5,6,7],"daily_start_time":"` + start + `","daily_end_time":"` + end + `","group_ids":[11]}`,
		},
	}, &config.Config{})

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway_discount_without_hydrated_group",
			Usage:     ClaudeUsage{InputTokens: 1000, OutputTokens: 500},
			Model:     "claude-sonnet-4",
			Duration:  time.Second,
		},
		APIKey: &APIKey{ID: 501, GroupID: i64p(groupID)},
		User:   &User{ID: 601},
		Account: &Account{
			ID: 701,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.InDelta(t, svc.cfg.Default.RateMultiplier*discount, usageRepo.lastLog.RateMultiplier, 1e-12)
}
