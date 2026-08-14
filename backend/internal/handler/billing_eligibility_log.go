package handler

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func billingEligibilityFailureFields(
	ctx context.Context,
	billingCacheService *service.BillingCacheService,
	err error,
	apiKey *service.APIKey,
	group *service.Group,
	subscription *service.UserSubscription,
	platform string,
	body []byte,
	model string,
	endpoint string,
) []zap.Field {
	appErr := infraerrors.FromError(err)
	fields := []zap.Field{
		zap.Error(err),
		zap.Int32("billing_http_status", appErr.Code),
		zap.String("billing_error_code", appErr.Reason),
		zap.String("billing_mode", billingModeForLog(group, subscription)),
		zap.String("precheck_scope", "user_key_group"),
		zap.String("platform", platform),
		zap.String("model", model),
		zap.String("endpoint", endpoint),
		zap.Int("body_bytes", len(body)),
		zap.Int("estimated_input_tokens", estimateRequestInputTokens(body)),
		zap.Float64("reserved_inflight_cost", 0),
		zap.Bool("estimated_cost_available", false),
		zap.String("estimated_cost_unavailable_reason", "eligibility_precheck_does_not_estimate_request_cost"),
		zap.Bool("precheck_uses_upstream_account", false),
	}
	if maxOutputTokens := requestMaxOutputTokens(body); maxOutputTokens != nil {
		fields = append(fields, zap.Int("max_output_tokens", *maxOutputTokens))
	}
	if apiKey != nil {
		fields = append(fields,
			zap.Int64("api_key_id", apiKey.ID),
			zap.Float64("api_key_quota", apiKey.Quota),
			zap.Float64("api_key_quota_used", apiKey.QuotaUsed),
		)
		if apiKey.GroupID != nil {
			fields = append(fields, zap.Int64("group_id", *apiKey.GroupID))
		}
		if apiKey.User != nil {
			fields = append(fields,
				zap.Int64("user_id", apiKey.User.ID),
				zap.Float64("user_snapshot_balance", apiKey.User.Balance),
			)
			if billingCacheService != nil {
				if balance, balanceErr := billingCacheService.GetUserBalance(ctx, apiKey.User.ID); balanceErr == nil {
					fields = append(fields,
						zap.Float64("balance", balance),
						zap.Float64("available_balance", balance),
						zap.String("balance_source", "billing_cache_service"),
					)
				} else {
					fields = append(fields, zap.NamedError("balance_lookup_error", balanceErr))
				}
			}
		}
	}
	if group != nil {
		fields = append(fields,
			zap.Int64("resolved_group_id", group.ID),
			zap.Float64("rate_multiplier", group.RateMultiplier),
			zap.Bool("rate_multiplier_applied_in_precheck", false),
		)
	}
	return fields
}

func billingModeForLog(group *service.Group, subscription *service.UserSubscription) string {
	if group != nil && group.IsSubscriptionType() && subscription != nil {
		return "subscription"
	}
	return "balance"
}

func estimateRequestInputTokens(body []byte) int {
	if len(body) == 0 {
		return 0
	}
	return (len(body) + 3) / 4
}

func requestMaxOutputTokens(body []byte) *int {
	for _, field := range []string{"max_output_tokens", "max_tokens"} {
		value := gjson.GetBytes(body, field)
		if value.Type != gjson.Number {
			continue
		}
		n := int(value.Int())
		return &n
	}
	return nil
}
