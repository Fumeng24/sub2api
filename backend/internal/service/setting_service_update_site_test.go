//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSettingService_UpdateSettings_GroupRateDiscount_ValidatesAndStores(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	groupReader := &defaultSubGroupReaderStub{
		byID: map[int64]*Group{
			11: {ID: 11, SubscriptionType: SubscriptionTypeStandard},
			12: {ID: 12, SubscriptionType: SubscriptionTypeSubscription},
		},
	}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetDefaultSubscriptionGroupReader(groupReader)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		systemSettingsCustom: systemSettingsCustom{GroupRateDiscountSettings: GroupRateDiscountSettings{
			Enabled:            true,
			Name:               "Promo",
			DiscountMultiplier: 0.75,
			ScheduleMode:       groupRateDiscountScheduleWeekly,
			Weekdays:           []int{3, 1, 1, 8},
			DailyStartTime:     "08:05",
			DailyEndTime:       "23:30",
			GroupIDs:           []int64{12, 11, 11},
		}},
	})
	require.NoError(t, err)

	raw := repo.updates[SettingKeyGroupRateDiscountSettings]
	var got GroupRateDiscountSettings
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	require.True(t, got.Enabled)
	require.Equal(t, "Promo", got.Name)
	require.Equal(t, 0.75, got.DiscountMultiplier)
	require.Equal(t, groupRateDiscountScheduleWeekly, got.ScheduleMode)
	require.Equal(t, []int{1, 3}, got.Weekdays)
	require.Equal(t, "08:05", got.DailyStartTime)
	require.Equal(t, "23:30", got.DailyEndTime)
	require.Equal(t, []int64{11, 12}, got.GroupIDs)
}

func TestSettingService_UpdateSettings_GroupRateDiscount_RejectsInvalidWindow(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	now := time.Now().UTC().Format(time.RFC3339)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		systemSettingsCustom: systemSettingsCustom{GroupRateDiscountSettings: GroupRateDiscountSettings{
			Enabled:            true,
			Name:               "Promo",
			DiscountMultiplier: 0.8,
			ScheduleMode:       groupRateDiscountScheduleOnce,
			StartAt:            now,
			EndAt:              now,
			GroupIDs:           []int64{11},
		}},
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_GROUP_RATE_DISCOUNT_WINDOW", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_GroupRateDiscount_RejectsEmptyWeekdays(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		systemSettingsCustom: systemSettingsCustom{GroupRateDiscountSettings: GroupRateDiscountSettings{
			Enabled:            true,
			Name:               "Promo",
			DiscountMultiplier: 0.8,
			ScheduleMode:       groupRateDiscountScheduleWeekly,
			Weekdays:           []int{},
			DailyStartTime:     "09:00",
			DailyEndTime:       "18:00",
			GroupIDs:           []int64{11},
		}},
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_GROUP_RATE_DISCOUNT_WEEKDAYS", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestGroupRateDiscountSettings_ActiveAtWeeklySchedule(t *testing.T) {
	oldLocal := time.Local
	time.Local = time.FixedZone("test-local", 8*60*60)
	t.Cleanup(func() {
		time.Local = oldLocal
	})

	settings := GroupRateDiscountSettings{
		Enabled:            true,
		Name:               "Promo",
		DiscountMultiplier: 0.8,
		ScheduleMode:       groupRateDiscountScheduleWeekly,
		Weekdays:           []int{1, 3},
		DailyStartTime:     "09:00",
		DailyEndTime:       "18:00",
		GroupIDs:           []int64{11},
	}

	require.NotNil(t, settings.ActiveAt(time.Date(2026, 5, 4, 10, 0, 0, 0, time.Local)))
	require.Nil(t, settings.ActiveAt(time.Date(2026, 5, 5, 10, 0, 0, 0, time.Local)))
	require.Nil(t, settings.ActiveAt(time.Date(2026, 5, 4, 18, 0, 0, 0, time.Local)))
}

func TestGroupRateDiscountSettings_ActiveAtWeeklyEarlyMorningWindow(t *testing.T) {
	oldLocal := time.Local
	time.Local = time.FixedZone("test-local", 8*60*60)
	t.Cleanup(func() {
		time.Local = oldLocal
	})

	settings := GroupRateDiscountSettings{
		Enabled:            true,
		Name:               "Night Promo",
		DiscountMultiplier: 0.7,
		ScheduleMode:       groupRateDiscountScheduleWeekly,
		Weekdays:           []int{1, 2, 3, 4, 5, 6, 7},
		DailyStartTime:     "02:00",
		DailyEndTime:       "08:00",
		GroupIDs:           []int64{11},
	}

	require.Nil(t, settings.ActiveAt(time.Date(2026, 5, 11, 1, 59, 0, 0, time.Local)))
	require.NotNil(t, settings.ActiveAt(time.Date(2026, 5, 11, 2, 0, 0, 0, time.Local)))
	require.NotNil(t, settings.ActiveAt(time.Date(2026, 5, 11, 7, 59, 0, 0, time.Local)))
	require.Nil(t, settings.ActiveAt(time.Date(2026, 5, 11, 8, 0, 0, 0, time.Local)))
}

func TestGroupRateDiscountSettings_ActiveAtWeeklyCrossMidnight(t *testing.T) {
	oldLocal := time.Local
	time.Local = time.FixedZone("test-local", 8*60*60)
	t.Cleanup(func() {
		time.Local = oldLocal
	})

	settings := GroupRateDiscountSettings{
		Enabled:            true,
		Name:               "Promo",
		DiscountMultiplier: 0.8,
		ScheduleMode:       groupRateDiscountScheduleWeekly,
		Weekdays:           []int{1},
		DailyStartTime:     "22:00",
		DailyEndTime:       "08:00",
		GroupIDs:           []int64{11},
	}

	require.NotNil(t, settings.ActiveAt(time.Date(2026, 5, 4, 23, 0, 0, 0, time.Local)))
	require.NotNil(t, settings.ActiveAt(time.Date(2026, 5, 5, 7, 59, 0, 0, time.Local)))
	require.Nil(t, settings.ActiveAt(time.Date(2026, 5, 5, 8, 0, 0, 0, time.Local)))
	require.Nil(t, settings.ActiveAt(time.Date(2026, 5, 6, 7, 0, 0, 0, time.Local)))
}
