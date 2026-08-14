package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCustomDTOEmbeddedFields_JSONContract(t *testing.T) {
	statusCode := 503
	category := "subscription"
	values := []struct {
		value any
		keys  []string
	}{
		{AdminUser{adminUserCustomFields: adminUserCustomFields{GroupDiscounts: map[int64]float64{1: 0.8}}}, []string{"group_discounts"}},
		{APIKey{apiKeyCustomFields: apiKeyCustomFields{Category: "openai"}}, []string{"category"}},
		{Group{groupCustomFields: groupCustomFields{ForceOpenAIPriority: true, OpenAIStableLowTTFT: true, ModelsListConfig: domain.GroupModelsListConfig{Enabled: true}}}, []string{"force_openai_priority", "openai_stable_low_ttft", "models_list_config"}},
		{Account{accountCustomFields: accountCustomFields{CurrentConcurrency: 2, TempUnschedulableStatusCode: &statusCode}}, []string{"current_concurrency", "temp_unschedulable_status_code"}},
		{AccountGroup{accountGroupCustomFields: accountGroupCustomFields{Role: "primary", Weight: 2, SortOrder: 3, SchedulingConfigured: true}}, []string{"role", "weight", "sort_order", "scheduling_configured"}},
		{RedeemCode{redeemCodeCustomFields: redeemCodeCustomFields{BusinessCategory: category}}, []string{"business_category"}},
		{UserSubscription{userSubscriptionCustomFields: userSubscriptionCustomFields{AutoResetDaily: true}}, []string{"auto_reset_daily"}},
	}

	for _, tc := range values {
		raw, err := json.Marshal(tc.value)
		require.NoError(t, err)
		var fields map[string]any
		require.NoError(t, json.Unmarshal(raw, &fields))
		for _, key := range tc.keys {
			require.Contains(t, fields, key, "serialized %T", tc.value)
		}
	}

	var update BatchUpdateRedeemCodeFields
	require.NoError(t, json.Unmarshal([]byte(`{"business_category":"subscription"}`), &update))
	require.NotNil(t, update.BusinessCategory)
	require.Equal(t, category, *update.BusinessCategory)
}
