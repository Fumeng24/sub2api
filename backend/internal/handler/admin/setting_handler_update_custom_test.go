package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestApplyBusinessSettingsUpdatePreservesOmittedValues(t *testing.T) {
	previous := &service.SystemSettings{}
	previous.GroupRateDiscountSettings = service.GroupRateDiscountSettings{
		Enabled: true,
		Name:    "Night rate",
	}
	previous.TicketSystemConfig = service.TicketSystemSettings{
		Templates: []service.TicketTemplate{{Key: "billing", Name: "Billing"}},
	}

	settings := &service.SystemSettings{}
	applyBusinessSettingsUpdate(settings, previous, UpdateSettingsRequest{})

	require.Equal(t, previous.GroupRateDiscountSettings, settings.GroupRateDiscountSettings)
	require.Equal(t, previous.TicketSystemConfig, settings.TicketSystemConfig)
}

func TestApplyBusinessSettingsUpdateAppliesProvidedValues(t *testing.T) {
	discount := dto.GroupRateDiscountSettings{
		Enabled:            true,
		Name:               "Weekend",
		DiscountMultiplier: 0.8,
		ScheduleMode:       "weekly",
		Weekdays:           []int{6, 7},
		DailyStartTime:     "00:00",
		DailyEndTime:       "23:59",
		GroupIDs:           []int64{2},
	}
	tickets := service.TicketSystemSettings{
		Templates: []service.TicketTemplate{{Key: "technical", Name: "Technical"}},
	}

	settings := &service.SystemSettings{}
	applyBusinessSettingsUpdate(settings, &service.SystemSettings{}, UpdateSettingsRequest{
		GroupRateDiscountSettings: &discount,
		TicketSystemConfig:        &tickets,
	})

	require.Equal(t, service.GroupRateDiscountSettings{
		Enabled:            true,
		Name:               "Weekend",
		DiscountMultiplier: 0.8,
		ScheduleMode:       "weekly",
		Weekdays:           []int{6, 7},
		DailyStartTime:     "00:00",
		DailyEndTime:       "23:59",
		GroupIDs:           []int64{2},
	}, settings.GroupRateDiscountSettings)
	require.Equal(t, tickets, settings.TicketSystemConfig)
}
