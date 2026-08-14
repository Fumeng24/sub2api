package service

import "encoding/json"

const defaultOpenAIFallbackModel = "gpt-5.4-mini"

func applySettingDefaultsCustom(defaults map[string]string) {
	defaults[SettingKeyGroupRateDiscountSettings] = `{"enabled":false,"name":"限时折扣","discount_multiplier":1,"schedule_mode":"weekly","start_at":"","end_at":"","weekdays":[1,2,3,4,5,6,7],"daily_start_time":"00:00","daily_end_time":"23:59","group_ids":[]}`
	defaults[SettingKeyTicketSystemConfig] = defaultTicketSystemSettingsJSON()
}

func applyParsedSystemSettingsCustom(result *SystemSettings, settings map[string]string) {
	result.GroupRateDiscountSettings = parseGroupRateDiscountSettings(settings[SettingKeyGroupRateDiscountSettings])
	result.TicketSystemConfig = DefaultTicketSystemSettings()
	if raw := settings[SettingKeyTicketSystemConfig]; raw != "" {
		var ticketSettings TicketSystemSettings
		if err := json.Unmarshal([]byte(raw), &ticketSettings); err == nil {
			result.TicketSystemConfig = NormalizeTicketSystemSettings(ticketSettings)
		}
	}
}
