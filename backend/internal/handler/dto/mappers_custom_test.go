package dto

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestCustomMapperFields(t *testing.T) {
	t.Run("admin user group discounts", func(t *testing.T) {
		discounts := map[int64]float64{7: 0.8}
		got := UserFromServiceAdmin(&service.User{GroupDiscounts: discounts})

		require.NotNil(t, got)
		require.Equal(t, discounts, got.GroupDiscounts)
	})

	t.Run("group routing and auto sort", func(t *testing.T) {
		models := service.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.6"}}
		autoSort := service.GroupAutoSortConfig{Enabled: true, Basis: "latency"}
		group := &service.Group{
			ForceOpenAIPriority: true,
			OpenAIStableLowTTFT: true,
			ModelsListConfig:    models,
			AutoSortConfig:      autoSort,
		}

		userDTO := GroupFromServiceShallow(group)
		adminDTO := GroupFromServiceAdmin(group)

		require.NotNil(t, userDTO)
		require.True(t, userDTO.ForceOpenAIPriority)
		require.True(t, userDTO.OpenAIStableLowTTFT)
		require.Equal(t, models, userDTO.ModelsListConfig)
		require.NotNil(t, adminDTO)
		require.Equal(t, autoSort, adminDTO.AutoSortConfig)
	})

	t.Run("account group scheduling defaults", func(t *testing.T) {
		accountGroup := &service.AccountGroup{
			Role:      service.AccountGroupRoleBackup,
			Weight:    3,
			SortOrder: 9,
		}

		got := AccountGroupFromService(accountGroup)

		require.NotNil(t, got)
		require.Equal(t, service.AccountGroupRoleBackup, got.Role)
		require.Equal(t, 3, got.Weight)
		require.Equal(t, 9, got.SortOrder)
		require.True(t, got.SchedulingConfigured)
	})

	t.Run("scheduler diagnostics", func(t *testing.T) {
		avgFirstTokenMs := 432.5
		entry := &service.AccountSchedulingEntry{
			AccountSchedulingConfig: service.AccountSchedulingConfig{
				AccountID:            11,
				Role:                 service.AccountGroupRolePrimary,
				Weight:               2,
				SortOrder:            4,
				SchedulingConfigured: true,
			},
			GroupID:                       22,
			State:                         "cooldown",
			BlockReason:                   service.AccountSchedulingBlockRateLimited,
			RecentUserAvgFirstTokenMs:     &avgFirstTokenMs,
			RecentUserFirstTokenSampleCnt: 8,
		}

		got := AccountSchedulingEntryFromService(entry)

		require.NotNil(t, got)
		require.Equal(t, int64(11), got.AccountID)
		require.Equal(t, int64(22), got.GroupID)
		require.Equal(t, "rate_limited", got.BlockReason)
		require.Equal(t, &avgFirstTokenMs, got.RecentUserAvgFirstTokenMs)
		require.Equal(t, int64(8), got.RecentUserFirstTokenSampleCount)
	})

	t.Run("redeem business category", func(t *testing.T) {
		got := RedeemCodeFromService(&service.RedeemCode{RedeemCodeCustom: service.RedeemCodeCustom{BusinessCategory: "subscription"}})

		require.NotNil(t, got)
		require.Equal(t, "subscription", got.BusinessCategory)
	})

	t.Run("subscription daily reset", func(t *testing.T) {
		got := UserSubscriptionFromService(&service.UserSubscription{AutoResetDaily: true})

		require.NotNil(t, got)
		require.True(t, got.AutoResetDaily)
	})
}
