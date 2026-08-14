package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBillingEligibilityFailureFields_PrecheckScopeExcludesUpstreamAccount(t *testing.T) {
	groupID := int64(12)
	group := &service.Group{ID: groupID, RateMultiplier: 0.0061}
	apiKey := &service.APIKey{
		ID:      327,
		GroupID: &groupID,
		Group:   group,
		User:    &service.User{ID: 283, Balance: 4.96912842},
	}

	fields := billingEligibilityFailureFields(
		context.Background(),
		nil,
		service.ErrInsufficientBalance,
		apiKey,
		group,
		nil,
		service.PlatformOpenAI,
		[]byte(`{"model":"gpt-5.5","max_output_tokens":4096,"input":"hello"}`),
		"gpt-5.5",
		"/v1/responses",
	)
	keys := zapFieldKeys(fields)

	require.Contains(t, keys, "precheck_scope")
	require.Contains(t, keys, "precheck_uses_upstream_account")
	require.Contains(t, keys, "user_snapshot_balance")
	require.Contains(t, keys, "rate_multiplier")
	require.Contains(t, keys, "rate_multiplier_applied_in_precheck")
	require.Contains(t, keys, "estimated_input_tokens")
	require.Contains(t, keys, "max_output_tokens")
	require.NotContains(t, keys, "account_rate_multiplier")
}

func zapFieldKeys(fields []zap.Field) map[string]struct{} {
	keys := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		keys[field.Key] = struct{}{}
	}
	return keys
}
