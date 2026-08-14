package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type adminUserCustomFields struct {
	GroupDiscounts map[int64]float64 `json:"group_discounts,omitempty"`
}

type apiKeyCustomFields struct {
	Category string `json:"category"`
}

type groupCustomFields struct {
	ForceOpenAIPriority bool                         `json:"force_openai_priority"`
	OpenAIStableLowTTFT bool                         `json:"openai_stable_low_ttft"`
	ModelsListConfig    domain.GroupModelsListConfig `json:"models_list_config"`
}

type adminGroupCustomFields struct {
	AutoSortConfig domain.GroupAutoSortConfig `json:"auto_sort_config"`
}

type accountCustomFields struct {
	CurrentConcurrency          int    `json:"current_concurrency,omitempty"`
	TempUnschedulableStatusCode *int   `json:"temp_unschedulable_status_code,omitempty"`
	UpstreamID                  *int64 `json:"upstream_id,omitempty"`
}

type accountGroupCustomFields struct {
	Role                 string `json:"role"`
	Weight               int    `json:"weight"`
	SortOrder            int    `json:"sort_order"`
	SchedulingConfigured bool   `json:"scheduling_configured"`
}

type redeemCodeCustomFields struct {
	BusinessCategory string `json:"business_category"`
}

type batchUpdateRedeemCodeCustomFields struct {
	BusinessCategory *string `json:"business_category,omitempty"`
}

type userSubscriptionCustomFields struct {
	AutoResetDaily bool `json:"auto_reset_daily"`
}

type AccountSchedulingEntry struct {
	AccountID                       int64      `json:"account_id"`
	GroupID                         int64      `json:"group_id"`
	Role                            string     `json:"role"`
	Weight                          int        `json:"weight"`
	SortOrder                       int        `json:"sort_order"`
	SchedulingConfigured            bool       `json:"scheduling_configured"`
	Account                         *Account   `json:"account,omitempty"`
	State                           string     `json:"state,omitempty"`
	BlockReason                     string     `json:"block_reason,omitempty"`
	GroupReserve                    bool       `json:"group_reserve,omitempty"`
	GroupReserveUntil               *time.Time `json:"group_reserve_until,omitempty"`
	GroupReserveReason              string     `json:"group_reserve_reason,omitempty"`
	RecentUserAvgFirstTokenMs       *float64   `json:"recent_user_avg_first_token_ms,omitempty"`
	RecentUserFirstTokenSampleCount int64      `json:"recent_user_first_token_sample_count,omitempty"`
}
