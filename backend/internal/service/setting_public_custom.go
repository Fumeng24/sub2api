package service

import "time"

func withPublicSettingsCustom(result *PublicSettings, settings map[string]string) *PublicSettings {
	if result == nil {
		return nil
	}
	result.PaymentBalanceRechargeMultiplier = normalizeBalanceRechargeMultiplier(
		pcParseFloat(settings[SettingBalanceRechargeMult], defaultBalanceRechargeMultiplier),
	)
	result.GroupRateDiscount, result.UpcomingGroupRateDiscount = publicGroupRateDiscountState(
		settings[SettingKeyGroupRateDiscountSettings],
		time.Now(),
	)
	return result
}
