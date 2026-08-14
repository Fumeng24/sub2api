package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newOpenAISlowReserveSchedulerTestService(accounts []Account) *OpenAIGatewayService {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.LBTopK = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 1
	cfg.Gateway.OpenAIScheduler.SlowReserve = config.GatewayOpenAISlowReserveConfig{
		Enabled: true, TTFTMs: 15000, TTLSeconds: 120, MaxEntries: 64,
	}
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 25000
	return &OpenAIGatewayService{
		accountRepo:        schedulerGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
}

func TestOpenAISlowReservePrefersNormalCandidate(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()

	groupID := int64(81001)
	accounts := []Account{
		{ID: 81001, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 81002, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 10, GroupIDs: []int64{groupID}},
	}
	svc := newOpenAISlowReserveSchedulerTestService(accounts)
	slowTTFT := 15001
	svc.ReportOpenAIAccountScheduleResult(accounts[0].ID, "gpt-5.5", true, &slowTTFT)
	svc.ReportOpenAIAccountScheduleResult(accounts[0].ID, "gpt-5.5", true, &slowTTFT)

	selection, decision, err := svc.SelectAccountWithScheduler(context.Background(), &groupID, "", "", "gpt-5.5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[1].ID, selection.Account.ID)
	require.False(t, decision.SlowReserveSelected)
	require.True(t, svc.isOpenAIAccountSlowReserve(accounts[0].ID, "gpt-5.5"))
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAISlowReserveEscapesExistingStickySession(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()

	groupID := int64(81007)
	accounts := []Account{
		{ID: 81007, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 81008, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{groupID}},
	}
	svc := newOpenAISlowReserveSchedulerTestService(accounts)
	cache := svc.cache.(*schedulerTestGatewayCache)
	cache.sessionBindings = map[string]int64{"openai:slow-reserve-sticky": accounts[0].ID}
	slowTTFT := 25001
	svc.ReportOpenAIAccountScheduleResult(accounts[0].ID, "gpt-5.5", true, &slowTTFT)
	svc.ReportOpenAIAccountScheduleResult(accounts[0].ID, "gpt-5.5", true, &slowTTFT)

	selection, decision, err := svc.SelectAccountWithScheduler(context.Background(), &groupID, "", "slow-reserve-sticky", "gpt-5.5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[1].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, accounts[1].ID, cache.sessionBindings["openai:slow-reserve-sticky"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAISlowReserveStickyEscapeKeepsOnlyCandidateUsable(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()

	groupID := int64(81009)
	account := Account{ID: 81009, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}}
	svc := newOpenAISlowReserveSchedulerTestService([]Account{account})
	cache := svc.cache.(*schedulerTestGatewayCache)
	cache.sessionBindings = map[string]int64{"openai:slow-reserve-only-sticky": account.ID}
	slowTTFT := 25001
	svc.ReportOpenAIAccountScheduleResult(account.ID, "gpt-5.5", true, &slowTTFT)
	svc.ReportOpenAIAccountScheduleResult(account.ID, "gpt-5.5", true, &slowTTFT)

	selection, decision, err := svc.SelectAccountWithScheduler(context.Background(), &groupID, "", "slow-reserve-only-sticky", "gpt-5.5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.True(t, decision.SlowReserveSelected)
	require.Equal(t, account.ID, cache.sessionBindings["openai:slow-reserve-only-sticky"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAISlowReserveBelowStickyEscapeThresholdKeepsBinding(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()

	groupID := int64(81010)
	accounts := []Account{
		{ID: 81010, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 81011, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{groupID}},
	}
	svc := newOpenAISlowReserveSchedulerTestService(accounts)
	cache := svc.cache.(*schedulerTestGatewayCache)
	cache.sessionBindings = map[string]int64{"openai:slow-reserve-below-sticky-threshold": accounts[0].ID}
	slowTTFT := 10001
	svc.ReportOpenAIAccountScheduleResult(accounts[0].ID, "gpt-5.5", true, &slowTTFT)

	selection, decision, err := svc.SelectAccountWithScheduler(context.Background(), &groupID, "", "slow-reserve-below-sticky-threshold", "gpt-5.5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[0].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	require.Equal(t, accounts[0].ID, cache.sessionBindings["openai:slow-reserve-below-sticky-threshold"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAISlowReserveKeepsOnlySlowCandidateAvailable(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	defer resetOpenAIAdvancedSchedulerSettingCacheForTest()

	groupID := int64(81002)
	account := Account{ID: 81003, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}}
	svc := newOpenAISlowReserveSchedulerTestService([]Account{account})
	slowTTFT := 15001
	svc.ReportOpenAIAccountScheduleResult(account.ID, "gpt-5.5", true, &slowTTFT)
	svc.ReportOpenAIAccountScheduleResult(account.ID, "gpt-5.5", true, &slowTTFT)

	selection, decision, err := svc.SelectAccountWithScheduler(context.Background(), &groupID, "", "", "gpt-5.5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.True(t, decision.SlowReserveSelected)
	require.GreaterOrEqual(t, svc.SnapshotOpenAIAccountSchedulerMetrics().SlowReserveSelectedTotal, int64(1))
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAISlowReserveFastRecoveryDoesNotClearMappedModel(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	slowTTFT := 15001
	fastTTFT := 9999

	svc.ReportOpenAIAccountScheduleResult(81004, "gpt-5.5", true, &slowTTFT)
	require.False(t, svc.isOpenAIAccountSlowReserve(81004, "gpt-5.5"), "one slow sample should remain pending")
	svc.ReportOpenAIAccountScheduleResult(81004, "gpt-5.5", true, &slowTTFT)
	require.True(t, svc.isOpenAIAccountSlowReserve(81004, "gpt-5.5"))
	require.False(t, svc.isOpenAIAccountSlowReserve(81004, "gpt-5.4"))

	svc.ReportOpenAIAccountScheduleResult(81004, "gpt-5.5", true, &fastTTFT)
	require.True(t, svc.isOpenAIAccountSlowReserve(81004, "gpt-5.5"))
}

func TestOpenAISlowReserveExpiryAndTimeoutClassificationDoNotMutateAccount(t *testing.T) {
	state := newOpenAIAccountSlowReserveState()
	now := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
	cfg := openAISlowReserveConfig{enabled: true, ttl: 2 * time.Minute, maxEntries: 2}
	_, marked := state.mark(81005, "gpt-5.5", "ttft", 12000, now, cfg)
	require.True(t, marked)
	require.True(t, state.isReserved(81005, "gpt-5.5", now.Add(time.Minute)))
	require.False(t, state.isReserved(81005, "gpt-5.5", now.Add(2*time.Minute)))
	require.Zero(t, state.size(now.Add(2*time.Minute)))

	account := &Account{ID: 81006, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	before := *account
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.5", &UpstreamFailoverError{StatusCode: http.StatusGatewayTimeout})
	require.False(t, svc.isOpenAIAccountSlowReserve(account.ID, "gpt-5.5"), "one recovered timeout should remain pending")
	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.5", &UpstreamFailoverError{StatusCode: http.StatusGatewayTimeout})
	require.True(t, svc.isOpenAIAccountSlowReserve(account.ID, "gpt-5.5"), "a repeated timeout should promote to slow reserve")
	require.Equal(t, before, *account)

	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.4", &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable})
	require.False(t, svc.isOpenAIAccountSlowReserve(account.ID, "gpt-5.4"))
}
