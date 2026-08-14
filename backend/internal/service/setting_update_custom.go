package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *SettingService) prepareSystemSettingsUpdatesCustom(ctx context.Context, settings *SystemSettings) error {
	if err := s.validateGroupRateDiscountSettings(ctx, &settings.GroupRateDiscountSettings); err != nil {
		return err
	}
	settings.TicketSystemConfig = NormalizeTicketSystemSettings(settings.TicketSystemConfig)
	return nil
}

func appendSystemSettingsUpdatesCustom(updates map[string]string, settings *SystemSettings) error {
	groupRateDiscountJSON, err := json.Marshal(settings.GroupRateDiscountSettings)
	if err != nil {
		return fmt.Errorf("marshal group rate discount settings: %w", err)
	}
	updates[SettingKeyGroupRateDiscountSettings] = string(groupRateDiscountJSON)
	ticketSystemConfigJSON, err := json.Marshal(settings.TicketSystemConfig)
	if err != nil {
		return fmt.Errorf("marshal ticket system config: %w", err)
	}
	updates[SettingKeyTicketSystemConfig] = string(ticketSystemConfigJSON)
	return nil
}

func (s *SettingService) refreshCachedSettingsCustom(settings *SystemSettings) {
	s.groupRateDiscountCache.Store(&cachedGroupRateDiscountSettings{
		settings:  settings.GroupRateDiscountSettings,
		expiresAt: time.Now().Add(groupRateDiscountSettingsCacheTTL).UnixNano(),
	})
}
