package service

import "github.com/Wei-Shaw/sub2api/internal/domain"

const (
	RoleSupport = domain.RoleSupport

	AffiliateBindBonusAmountDefault = 0.0

	BalanceBusinessCategoryUnclassified     = domain.BalanceBusinessCategoryUnclassified
	BalanceBusinessCategoryRecharge         = domain.BalanceBusinessCategoryRecharge
	BalanceBusinessCategoryManualCollection = domain.BalanceBusinessCategoryManualCollection
	BalanceBusinessCategoryManualRefund     = domain.BalanceBusinessCategoryManualRefund
	BalanceBusinessCategoryGiftCompensation = domain.BalanceBusinessCategoryGiftCompensation
	BalanceBusinessCategoryGiftReversal     = domain.BalanceBusinessCategoryGiftReversal
	BalanceBusinessCategorySystemServiceFee = domain.BalanceBusinessCategorySystemServiceFee
	BalanceBusinessCategoryAffiliateReward  = domain.BalanceBusinessCategoryAffiliateReward

	SettingKeyAffiliateBindBonusAmount  = "affiliate_bind_bonus_amount"
	SettingKeyTicketSystemConfig        = "ticket_system_config"
	SettingKeyGroupRateDiscountSettings = "group_rate_discount_settings"
)
