//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
}

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

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","*.EDU.CN","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_ExposesTablePreferences(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyTableDefaultPageSize: "50",
			SettingKeyTablePageSizeOptions: "[20,50,100]",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 50, settings.TableDefaultPageSize)
	require.Equal(t, []int{20, 50, 100}, settings.TablePageSizeOptions)
}

func TestSettingService_GetPublicSettings_ExposesForceEmailOnThirdPartySignup(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyForceEmailOnThirdPartySignup: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.ForceEmailOnThirdPartySignup)
}

func TestSettingService_GetPublicSettings_ExposesAllowUserViewErrorRequests(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyAllowUserViewErrorRequests: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.AllowUserViewErrorRequests)
}

func TestSettingService_GetPublicSettings_ExposesWeChatOAuthModeCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectAppID:               "wx-mp-app",
			SettingKeyWeChatConnectAppSecret:           "wx-mp-secret",
			SettingKeyWeChatConnectMode:                "mp",
			SettingKeyWeChatConnectScopes:              "snsapi_base",
			SettingKeyWeChatConnectOpenEnabled:         "true",
			SettingKeyWeChatConnectMPEnabled:           "true",
			SettingKeyWeChatConnectRedirectURL:         "https://api.example.com/api/v1/auth/oauth/wechat/callback",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.True(t, settings.WeChatOAuthMPEnabled)
}

func TestSettingService_GetPublicSettings_DoesNotExposeMobileOnlyWeChatAsWebOAuthAvailable(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectMobileEnabled:       "true",
			SettingKeyWeChatConnectMode:                "mobile",
			SettingKeyWeChatConnectMobileAppID:         "wx-mobile-app",
			SettingKeyWeChatConnectMobileAppSecret:     "wx-mobile-secret",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.WeChatOAuthEnabled)
	require.False(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.True(t, settings.WeChatOAuthMobileEnabled)
}

func TestSettingService_GetPublicSettings_FallsBackToConfigForWeChatOAuthCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			OpenEnabled:         true,
			OpenAppID:           "wx-open-config",
			OpenAppSecret:       "wx-open-secret",
			FrontendRedirectURL: "/auth/wechat/config-callback",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.False(t, settings.WeChatOAuthMobileEnabled)
}
