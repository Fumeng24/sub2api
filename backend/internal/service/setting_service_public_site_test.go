//go:build unit

package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSettingService_GetPublicSettings_ExposesActiveGroupRateDiscount(t *testing.T) {
	start := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	end := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyGroupRateDiscountSettings: `{"enabled":true,"name":"Night Sale","discount_multiplier":0.8,"start_at":"` + start + `","end_at":"` + end + `","group_ids":[2,1,1]}`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.NotNil(t, settings.GroupRateDiscount)
	require.Equal(t, "Night Sale", settings.GroupRateDiscount.Name)
	require.Equal(t, 0.8, settings.GroupRateDiscount.DiscountMultiplier)
	require.Equal(t, groupRateDiscountScheduleOnce, settings.GroupRateDiscount.ScheduleMode)
	require.NotEmpty(t, settings.GroupRateDiscount.Timezone)
	require.Equal(t, []int64{1, 2}, settings.GroupRateDiscount.GroupIDs)
}

func TestSettingService_GetPublicSettings_ExposesWeeklyGroupRateDiscount(t *testing.T) {
	now := time.Now().In(time.Local)
	start := now.Add(-30 * time.Minute).Format("15:04")
	end := now.Add(30 * time.Minute).Format("15:04")
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyGroupRateDiscountSettings: `{"enabled":true,"name":"Lunch Sale","discount_multiplier":0.7,"schedule_mode":"weekly","weekdays":[1,2,3,4,5,6,7],"daily_start_time":"` + start + `","daily_end_time":"` + end + `","group_ids":[3]}`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.NotNil(t, settings.GroupRateDiscount)
	require.Equal(t, "Lunch Sale", settings.GroupRateDiscount.Name)
	require.Equal(t, 0.7, settings.GroupRateDiscount.DiscountMultiplier)
	require.Equal(t, groupRateDiscountScheduleWeekly, settings.GroupRateDiscount.ScheduleMode)
	require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7}, settings.GroupRateDiscount.Weekdays)
	require.Equal(t, start, settings.GroupRateDiscount.DailyStartTime)
	require.Equal(t, end, settings.GroupRateDiscount.DailyEndTime)
	require.NotEmpty(t, settings.GroupRateDiscount.Timezone)
	require.Equal(t, []int64{3}, settings.GroupRateDiscount.GroupIDs)
}

func TestSettingService_GetPublicSettings_ExposesUpcomingWeeklyGroupRateDiscount(t *testing.T) {
	now := time.Now().In(time.Local)
	start := now.Add(time.Hour).Format("15:04")
	end := now.Add(2 * time.Hour).Format("15:04")
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyGroupRateDiscountSettings: `{"enabled":true,"name":"Night Sale","discount_multiplier":0.7,"schedule_mode":"weekly","weekdays":[1,2,3,4,5,6,7],"daily_start_time":"` + start + `","daily_end_time":"` + end + `","group_ids":[5]}`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Nil(t, settings.GroupRateDiscount)
	require.NotNil(t, settings.UpcomingGroupRateDiscount)
	require.Equal(t, "Night Sale", settings.UpcomingGroupRateDiscount.Name)
	require.Equal(t, 0.7, settings.UpcomingGroupRateDiscount.DiscountMultiplier)
	require.Equal(t, groupRateDiscountScheduleWeekly, settings.UpcomingGroupRateDiscount.ScheduleMode)
	require.Equal(t, []int64{5}, settings.UpcomingGroupRateDiscount.GroupIDs)
}

func TestSettingService_GetPublicSettings_ExposesUpcomingOneOffGroupRateDiscount(t *testing.T) {
	start := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	end := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyGroupRateDiscountSettings: `{"enabled":true,"name":"Future Sale","discount_multiplier":0.8,"start_at":"` + start + `","end_at":"` + end + `","group_ids":[7]}`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Nil(t, settings.GroupRateDiscount)
	require.NotNil(t, settings.UpcomingGroupRateDiscount)
	require.Equal(t, "Future Sale", settings.UpcomingGroupRateDiscount.Name)
	require.Equal(t, groupRateDiscountScheduleOnce, settings.UpcomingGroupRateDiscount.ScheduleMode)
	require.Equal(t, []int64{7}, settings.UpcomingGroupRateDiscount.GroupIDs)
}
