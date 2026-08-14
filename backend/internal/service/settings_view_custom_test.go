//go:build unit

package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettingsCustomFieldsRemainFlattenedInJSON(t *testing.T) {
	systemBody, err := json.Marshal(SystemSettings{
		systemSettingsCustom: systemSettingsCustom{
			GroupRateDiscountSettings: GroupRateDiscountSettings{Name: "Promo"},
			TicketSystemConfig:        TicketSystemSettings{},
		},
	})
	require.NoError(t, err)
	var systemJSON map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(systemBody, &systemJSON))
	require.Contains(t, systemJSON, "group_rate_discount_settings")
	require.Contains(t, systemJSON, "ticket_system_config")
	require.NotContains(t, systemJSON, "systemSettingsCustom")
	var groupRate GroupRateDiscountSettings
	require.NoError(t, json.Unmarshal(systemJSON["group_rate_discount_settings"], &groupRate))
	require.Equal(t, "Promo", groupRate.Name)

	discount := &ActiveGroupRateDiscount{Name: "Promo", DiscountMultiplier: 0.8}
	publicBody, err := json.Marshal(PublicSettings{
		publicSettingsCustom: publicSettingsCustom{
			PaymentBalanceRechargeMultiplier: 6.8,
			GroupRateDiscount:                discount,
		},
	})
	require.NoError(t, err)
	var publicJSON map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(publicBody, &publicJSON))
	require.JSONEq(t, `6.8`, string(publicJSON["PaymentBalanceRechargeMultiplier"]))
	require.Contains(t, publicJSON, "group_rate_discount")
	require.NotContains(t, publicJSON, "publicSettingsCustom")
}
