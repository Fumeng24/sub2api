package domain

const RoleSupport = "support"

// API key user-facing category constants.
const (
	APIKeyCategoryOpenAI    = "openai"
	APIKeyCategoryAnthropic = "anthropic"
	APIKeyCategoryOther     = "other"
)

// Balance business category constants.
const (
	BalanceBusinessCategoryUnclassified     = ""
	BalanceBusinessCategoryRecharge         = "recharge"
	BalanceBusinessCategoryManualCollection = "manual_collection"
	BalanceBusinessCategoryManualRefund     = "manual_refund"
	BalanceBusinessCategoryGiftCompensation = "gift_compensation"
	BalanceBusinessCategoryGiftReversal     = "gift_reversal"
	BalanceBusinessCategorySystemServiceFee = "system_service_fee"
	BalanceBusinessCategoryAffiliateReward  = "affiliate_reward"
)
