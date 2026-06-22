package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAISnapshotCacheStub struct {
	SchedulerCache
	snapshotAccounts []*Account
	accountsByID     map[int64]*Account
}

type schedulerTestOpenAIAccountRepo struct {
	AccountRepository
	accounts []Account
}

type schedulerTestGroupAwareOpenAIAccountRepo struct {
	schedulerTestOpenAIAccountRepo
}

func (r schedulerTestGroupAwareOpenAIAccountRepo) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if openAIAccountBelongsToGroup(acc, groupID) {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestGroupAwareOpenAIAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform && openAIAccountBelongsToGroup(acc, groupID) {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestOpenAIAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			return &r.accounts[i], nil
		}
	}
	return nil, errors.New("account not found")
}

func (r schedulerTestOpenAIAccountRepo) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	return append([]Account(nil), r.accounts...), nil
}

func (r schedulerTestOpenAIAccountRepo) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerTestOpenAIAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

func (r schedulerTestOpenAIAccountRepo) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	return nil
}

type schedulerGroupAwareOpenAIAccountRepo struct {
	schedulerTestOpenAIAccountRepo
}

func (r schedulerGroupAwareOpenAIAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform && openAIStickyAccountMatchesGroup(&acc, &groupID) {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r schedulerGroupAwareOpenAIAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform && openAIStickyAccountMatchesGroup(&acc, nil) {
			result = append(result, acc)
		}
	}
	return result, nil
}

type schedulerTestConcurrencyCache struct {
	ConcurrencyCache
	loadBatchErr    error
	loadMap         map[int64]*AccountLoadInfo
	acquireResults  map[int64]bool
	waitCounts      map[int64]int
	skipDefaultLoad bool
}

func (c schedulerTestConcurrencyCache) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	if c.acquireResults != nil {
		if result, ok := c.acquireResults[accountID]; ok {
			return result, nil
		}
	}
	return true, nil
}

func (c schedulerTestConcurrencyCache) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	return nil
}

func (c schedulerTestConcurrencyCache) GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	if c.loadBatchErr != nil {
		return nil, c.loadBatchErr
	}
	out := make(map[int64]*AccountLoadInfo, len(accounts))
	if c.skipDefaultLoad && c.loadMap != nil {
		for _, acc := range accounts {
			if load, ok := c.loadMap[acc.ID]; ok {
				out[acc.ID] = load
			}
		}
		return out, nil
	}
	for _, acc := range accounts {
		if c.loadMap != nil {
			if load, ok := c.loadMap[acc.ID]; ok {
				out[acc.ID] = load
				continue
			}
		}
		out[acc.ID] = &AccountLoadInfo{AccountID: acc.ID, LoadRate: 0}
	}
	return out, nil
}

func (c schedulerTestConcurrencyCache) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	if c.waitCounts != nil {
		if count, ok := c.waitCounts[accountID]; ok {
			return count, nil
		}
	}
	return 0, nil
}

type schedulerTestGatewayCache struct {
	sessionBindings map[string]int64
	deletedSessions map[string]int
}

func (c *schedulerTestGatewayCache) GetSessionAccountID(ctx context.Context, groupID int64, sessionHash string) (int64, error) {
	if id, ok := c.sessionBindings[sessionHash]; ok {
		return id, nil
	}
	return 0, errors.New("not found")
}

func (c *schedulerTestGatewayCache) SetSessionAccountID(ctx context.Context, groupID int64, sessionHash string, accountID int64, ttl time.Duration) error {
	if c.sessionBindings == nil {
		c.sessionBindings = make(map[string]int64)
	}
	c.sessionBindings[sessionHash] = accountID
	return nil
}

func (c *schedulerTestGatewayCache) RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error {
	return nil
}

func (c *schedulerTestGatewayCache) DeleteSessionAccountID(ctx context.Context, groupID int64, sessionHash string) error {
	if c.sessionBindings == nil {
		return nil
	}
	if c.deletedSessions == nil {
		c.deletedSessions = make(map[string]int)
	}
	c.deletedSessions[sessionHash]++
	delete(c.sessionBindings, sessionHash)
	return nil
}

func newSchedulerTestOpenAIWSV2Config() *config.Config {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	return cfg
}

type openAIAdvancedSchedulerSettingRepoStub struct {
	values map[string]string
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected call to Set")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected call to GetMultiple")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected call to SetMultiple")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected call to GetAll")
}

func (s *openAIAdvancedSchedulerSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected call to Delete")
}

func newOpenAIAdvancedSchedulerRateLimitService(enabled string) *RateLimitService {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	repo := &openAIAdvancedSchedulerSettingRepoStub{
		values: map[string]string{},
	}
	if enabled != "" {
		repo.values[openAIAdvancedSchedulerSettingKey] = enabled
	}
	return &RateLimitService{
		settingService: NewSettingService(repo, &config.Config{}),
	}
}

func (s *openAISnapshotCacheStub) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if len(s.snapshotAccounts) == 0 {
		return nil, false, nil
	}
	out := make([]*Account, 0, len(s.snapshotAccounts))
	for _, account := range s.snapshotAccounts {
		if account == nil {
			continue
		}
		cloned := *account
		out = append(out, &cloned)
	}
	return out, true, nil
}

func (s *openAISnapshotCacheStub) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s.accountsByID == nil {
		return nil, nil
	}
	account := s.accountsByID[accountID]
	if account == nil {
		return nil, nil
	}
	cloned := *account
	return &cloned, nil
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabledUsesLegacyLoadAwareness(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10106)
	accounts := []Account{
		{
			ID:          36001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
		},
		{
			ID:          36002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	cache := &schedulerTestGatewayCache{}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_disabled_001", 36001, time.Hour))
	require.False(t, svc.isOpenAIAdvancedSchedulerEnabled(ctx))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_disabled_001",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(36002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabled_RequiredWSV2_SkipsHTTPOnlyAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10108)
	accounts := []Account{
		{
			ID:          36011,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          36012,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(36012), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabled_RequiredWSV2_NoAvailableAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10109)
	accounts := []Account{
		{
			ID:          36021,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.ErrorContains(t, err, "no available OpenAI accounts")
	require.Nil(t, selection)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.True(t, decision.Diagnostics.Collected)
	require.Equal(t, groupID, decision.Diagnostics.GroupID)
	require.Equal(t, "gpt-5.1", decision.Diagnostics.Model)
	require.Equal(t, 0, decision.Diagnostics.GroupBindingAccountCount)
	require.Empty(t, decision.Diagnostics.CandidateAccountIDs)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabled_EmbeddingsSkipsChatOnlyAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10110)
	accounts := []Account{
		{
			ID:          36031,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			Credentials: map[string]any{
				"openai_capabilities": []any{"chat_completions"},
			},
		},
		{
			ID:          36032,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Credentials: map[string]any{
				"openai_capabilities": []any{"chat_completions", "embeddings"},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		ctx,
		&groupID,
		"",
		"",
		"text-embedding-3-small",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(36032), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_EnabledUsesAdvancedPreviousResponseRouting(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10107)
	accounts := []Account{
		{
			ID:          37001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
		{
			ID:          37002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_enabled_001", 37001, time.Hour))
	require.True(t, svc.isOpenAIAdvancedSchedulerEnabled(ctx))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_enabled_001",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37001), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
	require.True(t, decision.StickyPreviousHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_Enabled_EmbeddingsSkipsChatOnlyAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10111)
	accounts := []Account{
		{
			ID:          37011,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			Credentials: map[string]any{
				"openai_capabilities": []any{"chat_completions"},
			},
		},
		{
			ID:          37012,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Credentials: map[string]any{
				"openai_capabilities": []any{"chat_completions", "embeddings"},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		ctx,
		&groupID,
		"",
		"",
		"text-embedding-3-small",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37012), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 1, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_ImageGenerationBridgeRequiredSkipsTextAccount(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10113)
	accounts := []Account{
		{
			ID:          37031,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    10,
			Extra:       map[string]any{},
		},
		{
			ID:          37032,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    1,
			Extra:       map[string]any{featureKeyCodexImageGenerationBridge: true},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithSchedulerForCapabilityAndOptions(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.5",
		nil,
		OpenAIUpstreamTransportAny,
		OpenAIEndpointCapabilityChatCompletions,
		false,
		OpenAIAccountScheduleOptions{RequireCodexImageGenerationBridge: true},
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37032), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 1, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_ImageGenerationBridgeCircuitDoesNotBlockTextResponses(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10114)
	account := Account{
		ID:          37041,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{groupID},
		Extra:       map[string]any{featureKeyCodexImageGenerationBridge: true},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}

	imageEndpoint := OpenAIResponsesSchedulerEndpointForIntent("/v1/responses", true)
	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.4", imageEndpoint, &UpstreamFailoverError{
		StatusCode:   http.StatusServiceUnavailable,
		ResponseBody: []byte(`{"error":{"message":"Service temporarily unavailable","type":"api_error"}}`),
	})

	textSelection, textDecision, err := svc.SelectAccountWithSchedulerForCapabilityAndOptions(
		WithSchedulerEndpoint(ctx, "/v1/responses"),
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		OpenAIEndpointCapabilityChatCompletions,
		false,
		OpenAIAccountScheduleOptions{},
	)
	require.NoError(t, err)
	require.NotNil(t, textSelection)
	require.NotNil(t, textSelection.Account)
	require.Equal(t, account.ID, textSelection.Account.ID)
	require.False(t, textSelection.WeakFallback)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, textDecision.Layer)
	if textSelection.ReleaseFunc != nil {
		textSelection.ReleaseFunc()
	}

	imageSelection, imageDecision, err := svc.SelectAccountWithSchedulerForCapabilityAndOptions(
		WithSchedulerEndpoint(ctx, imageEndpoint),
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		OpenAIEndpointCapabilityChatCompletions,
		false,
		OpenAIAccountScheduleOptions{RequireCodexImageGenerationBridge: true},
	)
	require.Error(t, err)
	require.Nil(t, imageSelection)
	require.True(t, imageDecision.Diagnostics.Collected)
	require.Equal(t, 1, imageDecision.Diagnostics.FilterReasonCounts["scheduler_circuit_open"])
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_ImageGenerationBridgeUsesSingleHalfOpenProbe(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10115)
	account := Account{
		ID:          37042,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{groupID},
		Extra:       map[string]any{featureKeyCodexImageGenerationBridge: true},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}

	imageEndpoint := OpenAIResponsesSchedulerEndpointForIntent("/v1/responses", true)
	svc.schedulerHealth.reportFailure(account.ID, "gpt-5.4", imageEndpoint, "transient", time.Nanosecond)
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.4", imageEndpoint, false)
		return snap.CircuitState == schedulerCircuitHalfOpen
	}, time.Second, time.Millisecond)

	imageCtx := WithSchedulerEndpoint(ctx, imageEndpoint)
	firstProbe, _, err := svc.SelectAccountWithSchedulerForCapabilityAndOptions(
		imageCtx,
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		OpenAIEndpointCapabilityChatCompletions,
		false,
		OpenAIAccountScheduleOptions{RequireCodexImageGenerationBridge: true},
	)
	require.NoError(t, err)
	require.NotNil(t, firstProbe)
	require.NotNil(t, firstProbe.Account)
	require.Equal(t, account.ID, firstProbe.Account.ID)
	require.False(t, firstProbe.WeakFallback)
	if firstProbe.ReleaseFunc != nil {
		firstProbe.ReleaseFunc()
	}

	secondProbe, secondDecision, err := svc.SelectAccountWithSchedulerForCapabilityAndOptions(
		imageCtx,
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		OpenAIEndpointCapabilityChatCompletions,
		false,
		OpenAIAccountScheduleOptions{RequireCodexImageGenerationBridge: true},
	)
	require.Error(t, err)
	require.Nil(t, secondProbe)
	require.True(t, secondDecision.Diagnostics.Collected)
	require.Equal(t, 1, secondDecision.Diagnostics.FilterReasonCounts["scheduler_probe_pending"])

	svc.ReportOpenAIAccountScheduleResultForRequest(account.ID, "gpt-5.4", imageEndpoint, true, nil)
	recovered, _, err := svc.SelectAccountWithSchedulerForCapabilityAndOptions(
		imageCtx,
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		OpenAIEndpointCapabilityChatCompletions,
		false,
		OpenAIAccountScheduleOptions{RequireCodexImageGenerationBridge: true},
	)
	require.NoError(t, err)
	require.NotNil(t, recovered)
	require.NotNil(t, recovered.Account)
	require.Equal(t, account.ID, recovered.Account.ID)
	require.False(t, recovered.WeakFallback)
	if recovered.ReleaseFunc != nil {
		recovered.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_Enabled_EmbeddingsSkipsChatOnlyStickyBindings(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10112)
	accounts := []Account{
		{
			ID:          37021,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			Credentials: map[string]any{
				"openai_capabilities": []any{"chat_completions"},
			},
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
		{
			ID:          37022,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			Credentials: map[string]any{
				"openai_capabilities": []any{"chat_completions", "embeddings"},
			},
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_embeddings": 37021,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_embeddings_chat_only", 37021, time.Hour))

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		ctx,
		&groupID,
		"resp_embeddings_chat_only",
		"session_hash_embeddings",
		"text-embedding-3-small",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37022), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, int64(37022), cache.sessionBindings["openai:session_hash_embeddings"])
}

func TestOpenAIGatewayService_OpenAIAccountSchedulerMetrics_DisabledNoOp(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	svc := &OpenAIGatewayService{}
	ttft := 120
	svc.ReportOpenAIAccountScheduleResult(10, true, &ttft)
	svc.RecordOpenAIAccountSwitch()

	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.Equal(t, OpenAIAccountSchedulerMetricsSnapshot{}, snapshot)
}

func TestOpenAIGatewayService_MaxOpenAIAccountSwitchesRespectsConfiguredLimit(t *testing.T) {
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)

	svc := &OpenAIGatewayService{
		rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("true"),
	}

	require.Equal(t, 7, svc.MaxOpenAIAccountSwitches(context.Background(), 7, nil))
	require.Equal(t, 3, svc.MaxOpenAIAccountSwitches(context.Background(), 0, nil))
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRateLimitedAccountFallsBackToFreshCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10101)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	staleSticky := &Account{ID: 31001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleBackup := &Account{ID: 31002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	freshSticky := &Account{ID: 31001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	freshBackup := &Account{ID: 31002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_rate_limited": 31001}}
	snapshotCache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{staleSticky, staleBackup}, accountsByID: map[int64]*Account{31001: freshSticky, 31002: freshBackup}}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{*freshSticky, *freshBackup}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot:  snapshotService,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_rate_limited", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(31002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_AllNormalCandidatesCoolingReturnsUnavailable(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	accounts := []Account{
		{
			ID:               31101,
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      1,
			Priority:         0,
			RateLimitResetAt: &rateLimitedUntil,
		},
		{
			ID:               31102,
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      1,
			Priority:         5,
			RateLimitResetAt: &rateLimitedUntil,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, nil, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Contains(t, err.Error(), "no available OpenAI accounts")
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_CoolingPrimaryFallsBackToHealthyCandidate(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	accounts := []Account{
		{
			ID:               31103,
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      1,
			Priority:         0,
			RateLimitResetAt: &rateLimitedUntil,
		},
		{
			ID:          31104,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, nil, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(31104), selection.Account.ID)
	require.False(t, selection.WeakFallback)
	require.False(t, selection.BypassOpenAIHeaderTO)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SlowStreamingTTFTOpensCircuitAndBlocksUserFallback(t *testing.T) {
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	schedulerHealthyTTFTThreshold = 15 * time.Millisecond
	defer func() {
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	ctx := context.Background()
	account := Account{
		ID:          31105,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}

	slowTTFT := 16
	svc.ReportOpenAIAccountScheduleResultForRequest(account.ID, "gpt-5.5", "/v1/responses", true, &slowTTFT)

	snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.5", "/v1/responses", false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
	require.Equal(t, schedulerSlowTTFTCategory, snap.LastFailureReason)

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, nil, "", "", "gpt-5.5", nil, OpenAIUpstreamTransportAny, false)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Contains(t, err.Error(), "no available OpenAI accounts")
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_WeakFallbackSkipsExcludedAccounts(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	accounts := []Account{
		{
			ID:               31111,
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      1,
			Priority:         0,
			RateLimitResetAt: &rateLimitedUntil,
		},
		{
			ID:               31112,
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      1,
			Priority:         5,
			RateLimitResetAt: &rateLimitedUntil,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, _, err := svc.SelectAccountWithScheduler(
		ctx,
		nil,
		"",
		"",
		"gpt-5.1",
		map[int64]struct{}{31111: {}, 31112: {}},
		OpenAIUpstreamTransportAny,
		false,
	)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Contains(t, err.Error(), "no available OpenAI accounts")
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_WeakFallbackDoesNotBorrowUnboundAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(12)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	boundCooling := Account{
		ID:               31201,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeAPIKey,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		GroupIDs:         []int64{groupID},
		RateLimitResetAt: &rateLimitedUntil,
		AccountGroups: []AccountGroup{{
			AccountID:            31201,
			GroupID:              groupID,
			SortOrder:            10,
			Priority:             1,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SchedulingConfigured: true,
		}},
	}
	unboundHealthy := Account{
		ID:          31202,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: []Account{boundCooling, unboundHealthy}}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Contains(t, err.Error(), "no available OpenAI accounts")
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_AutoPauseBy5hThreshold(t *testing.T) {
	ctx := context.Background()
	primary := Account{
		ID:          35001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   95.0,
			"auto_pause_5h_threshold": 0.95,
		},
	}
	secondary := Account{ID: 35002, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35002), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_AllowsBelow5hThreshold(t *testing.T) {
	ctx := context.Background()
	primary := Account{
		ID:          35101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   80.0,
			"auto_pause_5h_threshold": 0.95,
		},
	}
	secondary := Account{ID: 35102, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35101), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_AutoPauseBy7dThreshold(t *testing.T) {
	ctx := context.Background()
	primary := Account{
		ID:          35201,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_7d_used_percent":   95.0,
			"auto_pause_7d_threshold": 0.95,
		},
	}
	secondary := Account{ID: 35202, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35202), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_UnconfiguredThresholdKeepsLegacyBehavior(t *testing.T) {
	ctx := context.Background()
	primary := Account{ID: 35301, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"codex_5h_used_percent": 99.0, "codex_7d_used_percent": 99.0}}
	secondary := Account{ID: 35302, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35301), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_UsesGlobalDefaultThreshold(t *testing.T) {
	ctx := withOpenAIQuotaAutoPauseSettings(context.Background(), OpsOpenAIAccountQuotaAutoPauseSettings{DefaultThreshold5h: 0.95})
	primary := Account{
		ID:          35401,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent": 95.0,
		},
	}
	secondary := Account{ID: 35402, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35402), account.ID)
}

// Regression: a per-account explicit-disable flag exempts the account from auto-pause
// even when a global default threshold is set. Without this, "leave threshold blank"
// silently falls back to global default and admins have no way to whitelist a single
// account.
func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_PerAccountDisableOverridesGlobalDefault(t *testing.T) {
	ctx := withOpenAIQuotaAutoPauseSettings(context.Background(), OpsOpenAIAccountQuotaAutoPauseSettings{DefaultThreshold5h: 0.95})
	// Account has high usage AND no per-account threshold (would normally fall back to
	// the global default and get paused), but the explicit disable flag is set.
	primary := Account{
		ID:          35701,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":  99.0,
			"auto_pause_5h_disabled": true,
		},
	}
	secondary := Account{ID: 35702, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35701), account.ID)
}

// Disable is per-window: disabling only 5h must still allow 7d auto-pause to fire.
func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_PerWindowDisableScoped(t *testing.T) {
	ctx := context.Background()
	primary := Account{
		ID:          35801,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   99.0,
			"codex_7d_used_percent":   99.0,
			"auto_pause_5h_disabled":  true,
			"auto_pause_7d_threshold": 0.95,
		},
	}
	secondary := Account{ID: 35802, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35802), account.ID, "7d auto-pause must still fire even though 5h is disabled")
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_StaleUsageWindowResetSkipsPause(t *testing.T) {
	ctx := context.Background()
	// Usage is over threshold but the window's reset time has already passed, so the
	// cached percentage is stale (the real window rolled over) and the account must NOT
	// stay paused — otherwise it could be skipped forever with no traffic to refresh it.
	primary := Account{
		ID:          35501,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   99.0,
			"auto_pause_5h_threshold": 0.95,
			"codex_5h_reset_at":       time.Now().Add(-time.Minute).Format(time.RFC3339),
		},
	}
	secondary := Account{ID: 35502, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35501), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_FreshUsageWindowStillPauses(t *testing.T) {
	ctx := context.Background()
	// Same as above but the window has not reset yet, so the account stays paused.
	primary := Account{
		ID:          35601,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   99.0,
			"auto_pause_5h_threshold": 0.95,
			"codex_5h_reset_at":       time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}
	secondary := Account{ID: 35602, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35602), account.ID)
}

// Issue #2994: an account poisoned with an inflated used% (e.g. from the reverted #2918
// inversion) gets excluded from scheduling, and a paused account never receives traffic to
// refresh its snapshot. When the snapshot is stale (codex_usage_updated_at older than the
// staleness bound) the account must be allowed a request so it can self-heal from the real
// response headers — independent of the window's reset time.
func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_StaleUsageSnapshotSkipsPause_Issue2994(t *testing.T) {
	ctx := context.Background()
	primary := Account{
		ID:          35701,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   99.0,
			"auto_pause_5h_threshold": 0.95,
			// Window has NOT reset yet, so the reset guard stays inactive.
			"codex_5h_reset_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			// Snapshot is stale: older than openAICodexAutoPauseStaleAfter (2h).
			"codex_usage_updated_at": time.Now().Add(-3 * time.Hour).Format(time.RFC3339),
		},
	}
	secondary := Account{ID: 35702, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35701), account.ID)
}

// Issue #2994 guardrail: a genuinely-exhausted account whose snapshot was refreshed recently
// (codex_usage_updated_at fresh) must STILL be auto-paused. The stale self-heal must not let a
// real 99%-used account escape pause.
func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_FreshExhaustedSnapshotStillPauses_Issue2994(t *testing.T) {
	ctx := context.Background()
	primary := Account{
		ID:          35801,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			"codex_5h_used_percent":   99.0,
			"auto_pause_5h_threshold": 0.95,
			"codex_5h_reset_at":       time.Now().Add(time.Hour).Format(time.RFC3339),
			// Snapshot refreshed 1 minute ago: not stale, so the account stays paused.
			"codex_usage_updated_at": time.Now().Add(-time.Minute).Format(time.RFC3339),
		},
	}
	secondary := Account{ID: 35802, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	svc := &OpenAIGatewayService{accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}}, cfg: &config.Config{}}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(35802), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_SkipsFreshlyRateLimitedSnapshotCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10102)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	stalePrimary := &Account{ID: 32001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleSecondary := &Account{ID: 32002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	freshPrimary := &Account{ID: 32001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	freshSecondary := &Account{ID: 32002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	snapshotCache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{stalePrimary, staleSecondary}, accountsByID: map[int64]*Account{32001: freshPrimary, 32002: freshSecondary}}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:       schedulerTestOpenAIAccountRepo{accounts: []Account{*freshPrimary, *freshSecondary}},
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot: snapshotService,
	}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(32002), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_ModelRateLimitOnlySkipsThatModel(t *testing.T) {
	ctx := context.Background()
	resetAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)
	primary := Account{
		ID:          32101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Extra: map[string]any{
			modelRateLimitsKey: map[string]any{
				"gpt-5.4": map[string]any{
					"rate_limit_reset_at": resetAt,
				},
			},
		},
	}
	secondary := Account{
		ID:          32102,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    5,
	}
	svc := &OpenAIGatewayService{
		accountRepo: schedulerTestOpenAIAccountRepo{accounts: []Account{primary, secondary}},
		cfg:         &config.Config{},
	}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.4", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(32102), account.ID)

	account, err = svc.SelectAccountForModelWithExclusions(ctx, nil, "", "gpt-5.3", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(32101), account.ID)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10103)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	staleSticky := &Account{ID: 33001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleBackup := &Account{ID: 33002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	dbSticky := Account{ID: 33001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	dbBackup := Account{ID: 33002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_db_runtime_recheck": 33001}}
	snapshotCache := &openAISnapshotCacheStub{
		snapshotAccounts: []*Account{staleSticky, staleBackup},
		accountsByID:     map[int64]*Account{33001: staleSticky, 33002: staleBackup},
	}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{dbSticky, dbBackup}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot:  snapshotService,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_db_runtime_recheck", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(33002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_DBRuntimeRecheckSkipsStaleCachedCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10104)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	stalePrimary := &Account{ID: 34001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleSecondary := &Account{ID: 34002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	dbPrimary := Account{ID: 34001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	dbSecondary := Account{ID: 34002, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	snapshotCache := &openAISnapshotCacheStub{
		snapshotAccounts: []*Account{stalePrimary, staleSecondary},
		accountsByID:     map[int64]*Account{34001: stalePrimary, 34002: staleSecondary},
	}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:       schedulerTestOpenAIAccountRepo{accounts: []Account{dbPrimary, dbSecondary}},
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		schedulerSnapshot: snapshotService,
	}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(34002), account.ID)
}

func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_RetriesFreshDBWhenSnapshotPoolEmpty(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10110)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	staleCooling := &Account{ID: 37001, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	dbCooling := Account{ID: 37001, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, RateLimitResetAt: &rateLimitedUntil}
	dbAvailable := Account{ID: 37002, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5}
	snapshotCache := &openAISnapshotCacheStub{
		snapshotAccounts: []*Account{staleCooling},
		accountsByID:     map[int64]*Account{37001: staleCooling},
	}
	snapshotService := &SchedulerSnapshotService{cache: snapshotCache}
	svc := &OpenAIGatewayService{
		accountRepo:       schedulerTestOpenAIAccountRepo{accounts: []Account{dbCooling, dbAvailable}},
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService(""),
		schedulerSnapshot: snapshotService,
	}

	account, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gpt-5.1", nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(37002), account.ID)
}

func TestOpenAIGatewayService_SelectAccountWithLoadAwareness_CoolingAccountReturnsUnavailable(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	account := Account{
		ID:               37011,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeAPIKey,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		RateLimitResetAt: &rateLimitedUntil,
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, err := svc.SelectAccountWithLoadAwareness(ctx, nil, "", "gpt-5.1", nil)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Contains(t, err.Error(), "no available OpenAI accounts")
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9)
	account := Account{
		ID:          1001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 2,
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
		},
	}
	cache := &schedulerTestGatewayCache{}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 1800
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_prev_001", account.ID, time.Hour))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_prev_001",
		"session_hash_001",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
	require.True(t, decision.StickyPreviousHit)
	require.Equal(t, account.ID, cache.sessionBindings["openai:session_hash_001"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10)
	account := Account{
		ID:          2001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		GroupIDs:    []int64{groupID},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_abc": account.ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_abc",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_CacheAffinitySticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10110)
	account := Account{
		ID:          36101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		GroupIDs:    []int64{groupID},
	}
	body := []byte(`{
		"model": "gpt-5.4",
		"messages": [
			{"role": "system", "content": "You are a coding assistant."},
			{"role": "user", "content": "fix the bug"}
		]
	}`)
	affinityHash := BuildOpenAICacheAffinityHash(9001, 7001, &groupID, "gpt-5.4", "/v1/responses", body)
	require.NotEmpty(t, affinityHash)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:" + affinityHash: account.ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: []Account{account}}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	reqCtx := WithOpenAICacheAffinityHash(ctx, affinityHash)
	selection, decision, err := svc.SelectAccountWithScheduler(
		reqCtx,
		&groupID,
		"",
		"new_turn_session_hash",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerCacheAffinity, decision.Layer)
	require.True(t, decision.CacheAffinityHit)
	require.Equal(t, account.ID, cache.sessionBindings["openai:new_turn_session_hash"])
	require.Equal(t, account.ID, cache.sessionBindings["openai:"+affinityHash])

	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.GreaterOrEqual(t, snapshot.CacheAffinityHitTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.StickyHitRatio, 1.0)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyBeatsCacheAffinity(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10114)
	accounts := []Account{
		{
			ID:          36141,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          36142,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9005,
		7005,
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","input":[{"role":"user","content":"keep cache hot"}]}`),
	)
	require.NotEmpty(t, affinityHash)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:normal_session_hash": accounts[0].ID,
			"openai:" + affinityHash:     accounts[1].ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"normal_session_hash",
		"gpt-5.5",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[0].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.False(t, decision.CacheAffinityHit)
	require.True(t, decision.StickySessionHit)
	require.Equal(t, accounts[0].ID, cache.sessionBindings["openai:normal_session_hash"])
	require.Equal(t, accounts[0].ID, cache.sessionBindings["openai:"+affinityHash])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyIgnoresCacheAffinityGroup(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10117)
	accounts := []Account{
		{
			ID:          36171,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra:       map[string]any{"cache_affinity_group": "prompt-a"},
		},
		{
			ID:          36172,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra:       map[string]any{"cache_affinity_group": "prompt-b"},
		},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9008,
		7008,
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","input":[{"role":"user","content":"same working tree"}]}`),
	)
	require.NotEmpty(t, affinityHash)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:normal_session_hash": accounts[0].ID,
			"openai:" + affinityHash:     accounts[1].ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"normal_session_hash",
		"gpt-5.5",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[0].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	require.False(t, decision.CacheAffinityHit)
	require.Equal(t, accounts[0].ID, cache.sessionBindings["openai:normal_session_hash"])
	require.Equal(t, accounts[0].ID, cache.sessionBindings["openai:"+affinityHash])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_CacheAffinityExcludedRebinds(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10111)
	accounts := []Account{
		{
			ID:          36111,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          36112,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9002,
		7002,
		&groupID,
		"gpt-5.4",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"same project"}]}`),
	)
	require.NotEmpty(t, affinityHash)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:" + affinityHash: accounts[0].ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"fresh_session_after_failover",
		"gpt-5.4",
		map[int64]struct{}{accounts[0].ID: {}},
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[1].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.CacheAffinityHit)
	require.Equal(t, accounts[1].ID, cache.sessionBindings["openai:"+affinityHash])
	require.Equal(t, accounts[1].ID, cache.sessionBindings["openai:fresh_session_after_failover"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_CacheAffinityPrefersCompatibleGroup(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10115)
	accounts := []Account{
		{
			ID:          36151,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra:       map[string]any{"cache_affinity_group": "prompt-a"},
		},
		{
			ID:          36152,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Priority:    99,
			Extra:       map[string]any{"cache_affinity_group": "prompt-a"},
		},
		{
			ID:          36153,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Priority:    -10,
			Extra:       map[string]any{"cache_affinity_group": "prompt-b"},
		},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9006,
		7006,
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","input":[{"role":"user","content":"same repo"}]}`),
	)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:" + affinityHash: accounts[0].ID,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"new_session",
		"gpt-5.5",
		map[int64]struct{}{accounts[0].ID: {}},
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[1].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, accounts[1].ID, cache.sessionBindings["openai:"+affinityHash])
	require.Equal(t, accounts[1].ID, cache.sessionBindings["openai:new_session"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_CacheAffinityFallsBackAcrossGroup(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10116)
	accounts := []Account{
		{
			ID:          36161,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra:       map[string]any{"cache_affinity_group": "prompt-a"},
		},
		{
			ID:          36162,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra:       map[string]any{"cache_affinity_group": "prompt-a"},
		},
		{
			ID:          36163,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra:       map[string]any{"cache_affinity_group": "prompt-b"},
		},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9007,
		7007,
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.5","input":[{"role":"user","content":"same repo"}]}`),
	)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:" + affinityHash: accounts[0].ID,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:      schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:            cache,
		cfg:              &config.Config{},
		rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
			acquireResults: map[int64]bool{
				accounts[1].ID: false,
			},
		}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"new_session",
		"gpt-5.5",
		map[int64]struct{}{accounts[0].ID: {}},
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[2].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, accounts[2].ID, cache.sessionBindings["openai:"+affinityHash])
	require.Equal(t, accounts[2].ID, cache.sessionBindings["openai:new_session"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_LoadBalanceBindsCacheAffinity(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10112)
	account := Account{
		ID:          36121,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		GroupIDs:    []int64{groupID},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9003,
		7003,
		&groupID,
		"gpt-5.4",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"bind after first selection"}]}`),
	)
	require.NotEmpty(t, affinityHash)
	cache := &schedulerTestGatewayCache{}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: []Account{account}}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"first_session_for_affinity",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, account.ID, cache.sessionBindings["openai:"+affinityHash])
	require.Equal(t, account.ID, cache.sessionBindings["openai:first_session_for_affinity"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabled_CacheAffinitySticky(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	ctx := context.Background()
	groupID := int64(10113)
	accounts := []Account{
		{
			ID:          36131,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Priority:    99,
		},
		{
			ID:          36132,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Priority:    0,
		},
	}
	affinityHash := BuildOpenAICacheAffinityHash(
		9004,
		7004,
		&groupID,
		"gpt-5.4",
		"/v1/responses",
		[]byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"legacy affinity"}]}`),
	)
	require.NotEmpty(t, affinityHash)
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:" + affinityHash: accounts[0].ID,
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		WithOpenAICacheAffinityHash(ctx, affinityHash),
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, accounts[0].ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyBusyKeepsSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10100)
	accounts := []Account{
		{
			ID:          21001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          21002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    9,
			GroupIDs:    []int64{groupID},
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_sticky_busy": 21001,
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.StickySessionMaxWaiting = 2
	cfg.Gateway.Scheduling.StickySessionWaitTimeout = 45 * time.Second
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = false
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 15000
	cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.5
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true

	concurrencyCache := schedulerTestConcurrencyCache{
		acquireResults: map[int64]bool{
			21001: false, // sticky 账号已满
			21002: true,  // 若回退调度队列会命中该账号（本测试要求不能切换）
		},
		waitCounts: map[int64]int{
			21001: 999,
		},
		loadMap: map[int64]*AccountLoadInfo{
			21001: {AccountID: 21001, LoadRate: 90, WaitingCount: 9},
			21002: {AccountID: 21002, LoadRate: 1, WaitingCount: 0},
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_sticky_busy",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21001), selection.Account.ID, "busy sticky account should remain selected")
	require.False(t, selection.Acquired)
	require.NotNil(t, selection.WaitPlan)
	require.Equal(t, int64(21001), selection.WaitPlan.AccountID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyEscapeByTTFT(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10101)
	accounts := []Account{
		{
			ID:          21101,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          21102,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    1,
			GroupIDs:    []int64{groupID},
		},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_sticky_ttft": 21101}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 15000
	cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.5
	concurrencyCache := schedulerTestConcurrencyCache{acquireResults: map[int64]bool{21102: true}}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
		openaiAccountStats: newOpenAIAccountRuntimeStats(),
	}
	fastTTFT := 14999
	svc.openaiAccountStats.report(21101, true, &fastTTFT)
	stableTTFT := 14999
	svc.openaiAccountStats.report(21101, true, &stableTTFT)

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_ttft", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21101), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}

	slowTTFT := 20000
	for i := 0; i < 3; i++ {
		svc.openaiAccountStats.report(21101, true, &slowTTFT)
	}

	selection, decision, err = svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_ttft", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21102), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, int64(21101), cache.sessionBindings["openai:session_hash_sticky_ttft"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyEscapeByErrorRate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10102)
	accounts := []Account{
		{ID: 21201, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 21202, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{groupID}},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_sticky_error_rate": 21201}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 15000
	cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.5
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{acquireResults: map[int64]bool{21202: true}}),
		openaiAccountStats: newOpenAIAccountRuntimeStats(),
	}
	for i := 0; i < 3; i++ {
		svc.openaiAccountStats.report(21201, false, nil)
	}
	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_error_rate", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21201), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
	for i := 0; i < 2; i++ {
		svc.openaiAccountStats.report(21201, false, nil)
	}

	selection, decision, err = svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_error_rate", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21202), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, int64(21201), cache.sessionBindings["openai:session_hash_sticky_error_rate"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyBusyEscapes(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10103)
	accounts := []Account{
		{ID: 21301, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 21302, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{groupID}},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_sticky_busy_escape": 21301}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 15000
	cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.5
	cfg.Gateway.Scheduling.StickySessionMaxWaiting = 2
	cfg.Gateway.Scheduling.StickySessionWaitTimeout = 45 * time.Second
	concurrencyCache := schedulerTestConcurrencyCache{
		acquireResults: map[int64]bool{21301: false, 21302: true},
		waitCounts:     map[int64]int{21301: 999},
		loadMap: map[int64]*AccountLoadInfo{
			21301: {AccountID: 21301, LoadRate: 95, WaitingCount: 9},
			21302: {AccountID: 21302, LoadRate: 1, WaitingCount: 0},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_busy_escape", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21302), selection.Account.ID)
	require.Nil(t, selection.WaitPlan)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyEscapeDisabledKeepsLegacyBehavior(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10104)
	accounts := []Account{
		{ID: 21401, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 21402, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{groupID}},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_sticky_disabled": 21401}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = false
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 15000
	cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.5
	cfg.Gateway.Scheduling.StickySessionMaxWaiting = 2
	cfg.Gateway.Scheduling.StickySessionWaitTimeout = 45 * time.Second
	concurrencyCache := schedulerTestConcurrencyCache{
		acquireResults: map[int64]bool{21401: false, 21402: true},
		waitCounts:     map[int64]int{21401: 999},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
		openaiAccountStats: newOpenAIAccountRuntimeStats(),
	}
	slowTTFT := 20000
	svc.openaiAccountStats.report(21401, true, &slowTTFT)
	for i := 0; i < 5; i++ {
		svc.openaiAccountStats.report(21401, false, nil)
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_disabled", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21401), selection.Account.ID)
	require.NotNil(t, selection.WaitPlan)
	require.Equal(t, int64(21401), selection.WaitPlan.AccountID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
}

func TestDefaultOpenAIAccountScheduler_ShouldEscapeStickyAccount_ThresholdBoundary(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	accountID := int64(21501)
	ttft := 15000
	stats.report(accountID, true, &ttft)
	stats.report(accountID, false, nil)
	stats.report(accountID, true, nil)
	scheduler := &defaultOpenAIAccountScheduler{stats: stats}

	reason, errorRate, observedTTFT, shouldEscape := scheduler.shouldEscapeStickyAccount(accountID, openAIStickyEscapeConfig{
		enabled:   true,
		ttftMs:    15000,
		errorRate: 0.5,
	})
	require.False(t, shouldEscape)
	require.Empty(t, reason)
	require.InDelta(t, 0.16, errorRate, 1e-9)
	require.InDelta(t, 15000, observedTTFT, 1e-9)

	for i := 0; i < 4; i++ {
		stats.report(accountID, false, nil)
	}
	reason, errorRate, _, shouldEscape = scheduler.shouldEscapeStickyAccount(accountID, openAIStickyEscapeConfig{
		enabled:   true,
		ttftMs:    15000,
		errorRate: 1,
	})
	require.False(t, shouldEscape)
	require.Empty(t, reason)
	reason, errorRate, observedTTFT, shouldEscape = scheduler.shouldEscapeStickyAccount(accountID, openAIStickyEscapeConfig{
		enabled:   true,
		ttftMs:    15000,
		errorRate: errorRate,
	})
	require.False(t, shouldEscape)
	require.Empty(t, reason)
	require.InDelta(t, 0.655936, errorRate, 1e-9)
	require.InDelta(t, 15000, observedTTFT, 1e-9)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_SessionSticky_ForceHTTP(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1010)
	account := Account{
		ID:          2101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		GroupIDs:    []int64{groupID},
		Extra: map[string]any{
			"openai_ws_force_http": true,
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_force_http": account.ID,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_force_http",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_RequiredWSV2_SkipsStickyHTTPAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1011)
	accounts := []Account{
		{
			ID:          2201,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          2202,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			GroupIDs:    []int64{groupID},
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_ws_only": 2201,
		},
	}
	cfg := newSchedulerTestOpenAIWSV2Config()

	// 构造“HTTP-only 账号负载更低”的场景，验证 required transport 会强制过滤。
	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			2201: {AccountID: 2201, LoadRate: 0, WaitingCount: 0},
			2202: {AccountID: 2202, LoadRate: 90, WaitingCount: 5},
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_ws_only",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(2202), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, 1, decision.CandidateCount)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_ClearsStickyAccountOutsideGroup(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1013)
	accounts := []Account{
		{
			ID:          2401,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          2402,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
			AccountGroups: []AccountGroup{
				{AccountID: 2402, GroupID: groupID},
			},
		},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_removed_group": 2401,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_removed_group",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(2402), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, 1, cache.deletedSessions["openai:session_hash_removed_group"])
	require.Equal(t, int64(2402), cache.sessionBindings["openai:session_hash_removed_group"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_RequiredWSV2_NoAvailableAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(1012)
	accounts := []Account{
		{
			ID:          2301,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                newSchedulerTestOpenAIWSV2Config(),
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportResponsesWebsocketV2,
		false,
	)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 0, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_OrderedFallbackUsesFullQueue(t *testing.T) {
	ctx := context.Background()
	groupID := int64(11)
	accounts := []Account{
		{
			ID:          3001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          3002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          3003,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}

	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 0.4
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 1.0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 1.0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 0.2
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 0.1

	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			3001: {AccountID: 3001, LoadRate: 95, WaitingCount: 8},
			3002: {AccountID: 3002, LoadRate: 20, WaitingCount: 1},
			3003: {AccountID: 3003, LoadRate: 10, WaitingCount: 0},
		},
		acquireResults: map[int64]bool{
			3003: false, // 最优候选抢不到槽时，继续尝试完整队列里的下一候选。
			3002: true,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(3002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 3, decision.CandidateCount)
	require.Equal(t, 3, decision.TopK)
	require.Greater(t, decision.LoadSkew, 0.0)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

// Regression: TopK initial filter must drop quota-auto-paused accounts. Otherwise
// the candidate pool is filled with paused accounts, healthy accounts fall outside
// TopK, and the scheduler returns "no available accounts" even though healthy ones
// exist.
func TestOpenAIGatewayService_SelectAccountWithScheduler_LoadBalanceTopKExcludesQuotaPaused(t *testing.T) {
	ctx := context.Background()
	groupID := int64(110)
	accounts := []Account{
		{
			ID:          37001,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
			Extra: map[string]any{
				"codex_5h_used_percent":   96.0,
				"auto_pause_5h_threshold": 0.95,
			},
		},
		{
			ID:          37002,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    5,
		},
	}

	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.LBTopK = 1 // TopK=1 makes the bug fatal: paused account would crowd out the healthy one entirely
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 0.4
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 1.0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 1.0

	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			37001: {AccountID: 37001, LoadRate: 5, WaitingCount: 0},
			37002: {AccountID: 37002, LoadRate: 5, WaitingCount: 0},
		},
		acquireResults: map[int64]bool{
			37002: true,
		},
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(37002), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	// Only the healthy account should ever enter the candidate pool; the paused one
	// must be filtered out at the initial-filter stage.
	require.Equal(t, 1, decision.CandidateCount)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_OpenAIAccountSchedulerMetrics(t *testing.T) {
	ctx := context.Background()
	groupID := int64(12)
	account := Account{
		ID:          4001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		GroupIDs:    []int64{groupID},
	}
	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_metrics": account.ID,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, _, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_metrics", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	svc.ReportOpenAIAccountScheduleResult(account.ID, true, intPtrForTest(120))
	svc.RecordOpenAIAccountSwitch()

	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.GreaterOrEqual(t, snapshot.SelectTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.StickySessionHitTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.AccountSwitchTotal, int64(1))
	require.GreaterOrEqual(t, snapshot.SchedulerLatencyMsAvg, float64(0))
	require.GreaterOrEqual(t, snapshot.StickyHitRatio, 0.0)
	require.GreaterOrEqual(t, snapshot.RuntimeStatsAccountCount, 1)
}

func TestOpenAISelectionDiagnostics_RecordsStageAccountIDs(t *testing.T) {
	ctx := context.Background()
	groupID := int64(12)
	tempUntil := time.Now().Add(time.Minute)
	accounts := []Account{
		{
			ID:          38795,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          38796,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          38797,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-4": "gpt-4"},
			},
		},
		{
			ID:          38798,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Credentials: map[string]any{
				openAIEndpointCapabilitiesCredentialKey: []any{string(OpenAIEndpointCapabilityEmbeddings)},
			},
		},
		{
			ID:                      38801,
			Platform:                PlatformOpenAI,
			Type:                    AccountTypeAPIKey,
			Status:                  StatusActive,
			Schedulable:             true,
			Concurrency:             1,
			GroupIDs:                []int64{groupID},
			TempUnschedulableUntil:  &tempUntil,
			TempUnschedulableReason: "upstream_502",
		},
		{
			ID:          38802,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts},
		cfg:         &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
			loadMap: map[int64]*AccountLoadInfo{
				38802: {AccountID: 38802, LoadRate: 100},
			},
		}),
	}
	svc.openaiAccountRuntimeBlockUntil.Store(int64(38795), time.Now().Add(time.Minute))
	schedulerAny := newDefaultOpenAIAccountScheduler(svc, nil)
	scheduler, ok := schedulerAny.(*defaultOpenAIAccountScheduler)
	require.True(t, ok)

	diag := scheduler.buildOpenAISelectionDiagnostics(ctx, OpenAIAccountScheduleRequest{
		GroupID:            &groupID,
		RequestedModel:     "gpt-5.5",
		RequiredCapability: OpenAIEndpointCapabilityChatCompletions,
		SchedulerEndpoint:  "/v1/responses",
		ExcludedIDs:        map[int64]struct{}{38796: {}},
	}, nil)

	require.True(t, diag.Collected)
	require.Equal(t, []int64{38795, 38796, 38797, 38798, 38801, 38802}, diag.GroupBindingAccountIDs)
	require.Equal(t, 6, diag.ActiveSchedulableCount)
	require.Equal(t, []int64{38795, 38796, 38797, 38798, 38801, 38802}, diag.ActiveSchedulableAccountIDs)
	require.Equal(t, []int64{38795, 38797, 38798, 38801, 38802}, diag.AfterExcludedAccountIDs)
	require.Equal(t, []int64{38795, 38798, 38801, 38802}, diag.ModelSupportedAccountIDs)
	require.Equal(t, []int64{38795, 38801, 38802}, diag.EndpointSupportedAccountIDs)
	require.Equal(t, []int64{38795, 38801, 38802}, diag.CompactSupportedAccountIDs)
	require.Equal(t, []int64{38795, 38802}, diag.StateAllowedAccountIDs)
	require.Equal(t, []int64{38802}, diag.CircuitAllowedAccountIDs)
	require.Empty(t, diag.CandidateAccountIDs)
	require.Equal(t, []int64{38797}, diag.ModelUnsupportedAccountIDs)
	require.Equal(t, []int64{38798}, diag.EndpointUnsupportedAccountIDs)
	require.Equal(t, []int64{38801}, diag.StateFilteredAccountIDs)
	require.Equal(t, []int64{38795}, diag.CircuitFilteredAccountIDs)
	require.Equal(t, []int64{38802}, diag.ConcurrencySlotFilteredAccountIDs)
	require.Equal(t, 0, diag.FinalCandidateCount)
	require.Equal(t, 1, diag.TempUnschedulableFilteredCount)
	require.Equal(t, 1, diag.ConcurrencySlotFilteredCount)
	require.Equal(t, 0, diag.StatusFilteredCount)
	require.Equal(t, 0, diag.OverloadFilteredCount)
	require.Equal(t, 0, diag.RateLimitFilteredCount)
	require.Equal(t, 0, diag.ModelRateLimitFilteredCount)
	require.Equal(t, 0, diag.ChannelRestrictionFilteredCount)
	require.Empty(t, diag.OrderedCandidateAccountIDs)
	require.Equal(t, 1, diag.FilterReasonCounts["excluded"])
	require.Equal(t, 1, diag.FilterReasonCounts["model_unsupported"])
	require.Equal(t, 1, diag.FilterReasonCounts["endpoint_unsupported"])
	require.Equal(t, 1, diag.FilterReasonCounts["temp_unschedulable"])
	require.Equal(t, 1, diag.FilterReasonCounts["runtime_circuit_open"])
	require.Equal(t, 1, diag.FilterReasonCounts["concurrency_full"])
}

func TestOpenAISelectionDiagnostics_AllCooldownIncludesEarliestRetry(t *testing.T) {
	ctx := context.Background()
	groupID := int64(12012)
	model := "gpt-5.9"
	endpoint := "/v1/responses"
	now := time.Now().UTC().Truncate(time.Second)
	rateLimitReset := now.Add(30 * time.Second)
	overloadUntil := now.Add(2 * time.Minute)
	tempUntil := now.Add(90 * time.Second)
	modelReset := now.Add(75 * time.Second)
	accounts := []Account{
		{
			ID:               40101,
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      1,
			GroupIDs:         []int64{groupID},
			RateLimitResetAt: &rateLimitReset,
		},
		{
			ID:            40102,
			Platform:      PlatformOpenAI,
			Type:          AccountTypeAPIKey,
			Status:        StatusActive,
			Schedulable:   true,
			Concurrency:   1,
			GroupIDs:      []int64{groupID},
			OverloadUntil: &overloadUntil,
		},
		{
			ID:                     40103,
			Platform:               PlatformOpenAI,
			Type:                   AccountTypeAPIKey,
			Status:                 StatusActive,
			Schedulable:            true,
			Concurrency:            1,
			GroupIDs:               []int64{groupID},
			TempUnschedulableUntil: &tempUntil,
		},
		{
			ID:          40104,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
			Extra: map[string]any{
				modelRateLimitsKey: map[string]any{
					model: map[string]any{"rate_limit_reset_at": modelReset.Format(time.RFC3339)},
				},
			},
		},
		{
			ID:          40105,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			GroupIDs:    []int64{groupID},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}
	svc.schedulerHealth.reportFailure(40105, model, endpoint, "transient", 45*time.Second)
	scheduler := &defaultOpenAIAccountScheduler{service: svc}
	req := OpenAIAccountScheduleRequest{
		GroupID:            &groupID,
		RequestedModel:     model,
		SchedulerEndpoint:  endpoint,
		RequiredTransport:  OpenAIUpstreamTransportAny,
		RequiredCapability: OpenAIEndpointCapabilityChatCompletions,
	}

	diag := scheduler.buildOpenAISelectionDiagnostics(ctx, req, nil)

	require.True(t, diag.Collected)
	require.Equal(t, 5, diag.GroupBindingAccountCount)
	require.Equal(t, 0, diag.FinalCandidateCount)
	require.Equal(t, 4, diag.StateFilteredCount)
	require.Equal(t, 1, diag.CircuitFilteredCount)
	require.Equal(t, 1, diag.FilterReasonCounts["rate_limited"])
	require.Equal(t, 1, diag.FilterReasonCounts["overloaded"])
	require.Equal(t, 1, diag.FilterReasonCounts["temp_unschedulable"])
	require.Equal(t, 1, diag.FilterReasonCounts["model_rate_limited"])
	require.Equal(t, 1, diag.FilterReasonCounts["scheduler_circuit_open"])
	require.Equal(t, int64(40101), diag.EarliestRetryAccountID)
	require.Equal(t, "rate_limited", diag.EarliestRetryReason)
	require.WithinDuration(t, rateLimitReset, diag.EarliestRetryAt, time.Second)
	require.Greater(t, diag.RetryAfterSeconds, 0)
	require.LessOrEqual(t, diag.RetryAfterSeconds, 31)

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "", model, nil, OpenAIUpstreamTransportAny, false)
	require.Error(t, err)
	require.Nil(t, selection)
	require.True(t, decision.Diagnostics.Collected)
	var noAvailable *OpenAINoAvailableAccountsError
	require.ErrorAs(t, err, &noAvailable)
	require.ErrorIs(t, err, ErrNoAvailableAccounts)
	require.Equal(t, int64(40101), noAvailable.Diagnostics.EarliestRetryAccountID)
	require.Equal(t, "rate_limited", noAvailable.Diagnostics.EarliestRetryReason)
	require.Greater(t, noAvailable.RetryAfterSeconds, 0)
	require.Contains(t, err.Error(), "filter_reasons=")
	require.Contains(t, err.Error(), "scheduler_circuit_open")
	require.Contains(t, err.Error(), "rate_limited")
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_CooldownPrimaryUsesHealthyBackup(t *testing.T) {
	ctx := context.Background()
	groupID := int64(12013)
	rateLimitReset := time.Now().Add(10 * time.Minute)
	cooling := Account{
		ID:               40201,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeAPIKey,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		GroupIDs:         []int64{groupID},
		RateLimitResetAt: &rateLimitReset,
	}
	healthy := Account{
		ID:          40202,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
		GroupIDs:    []int64{groupID},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{cooling, healthy}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "", "gpt-5.9", nil, OpenAIUpstreamTransportAny, false)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(40202), selection.Account.ID)
	require.True(t, selection.Acquired)
	require.False(t, selection.WeakFallback)
	require.Nil(t, selection.WaitPlan)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func intPtrForTest(v int) *int {
	return &v
}

func TestOpenAIAccountRuntimeStats_ReportAndSnapshot(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	stats.report(1001, true, nil)
	firstTTFT := 100
	stats.report(1001, false, &firstTTFT)
	secondTTFT := 200
	stats.report(1001, false, &secondTTFT)

	errorRate, ttft, hasTTFT := stats.snapshot(1001)
	require.True(t, hasTTFT)
	require.InDelta(t, 0.36, errorRate, 1e-9)
	require.InDelta(t, 120.0, ttft, 1e-9)
	require.Equal(t, 1, stats.size())
}

func TestOpenAIAccountRuntimeStats_ReportConcurrent(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()

	const (
		accountCount = 4
		workers      = 16
		iterations   = 800
	)
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				accountID := int64(i%accountCount + 1)
				success := (i+worker)%3 != 0
				ttft := 80 + (i+worker)%40
				stats.report(accountID, success, &ttft)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, accountCount, stats.size())
	for accountID := int64(1); accountID <= accountCount; accountID++ {
		errorRate, ttft, hasTTFT := stats.snapshot(accountID)
		require.GreaterOrEqual(t, errorRate, 0.0)
		require.LessOrEqual(t, errorRate, 1.0)
		require.True(t, hasTTFT)
		require.Greater(t, ttft, 0.0)
	}
}

func TestBuildOpenAIOrderedSelectionOrder_RanksFullQueue(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 11, Priority: 2},
			loadInfo: &AccountLoadInfo{LoadRate: 10, WaitingCount: 1},
			score:    10.0,
		},
		{
			account:  &Account{ID: 12, Priority: 1},
			loadInfo: &AccountLoadInfo{LoadRate: 20, WaitingCount: 1},
			score:    9.5,
		},
		{
			account:  &Account{ID: 13, Priority: 1},
			loadInfo: &AccountLoadInfo{LoadRate: 30, WaitingCount: 0},
			score:    10.0,
		},
		{
			account:  &Account{ID: 14, Priority: 0},
			loadInfo: &AccountLoadInfo{LoadRate: 40, WaitingCount: 0},
			score:    8.0,
		},
	}

	order := buildOpenAIOrderedSelectionOrder(candidates)
	require.Len(t, order, len(candidates))
	require.Equal(t, int64(14), order[0].account.ID)
	require.Equal(t, int64(13), order[1].account.ID)
	require.Equal(t, int64(12), order[2].account.ID)
	require.Equal(t, int64(11), order[3].account.ID)
}

func TestBuildOpenAIOrderedSelectionOrder_GroupOrderOverridesPriorityScoreAndLoad(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:    &Account{ID: 21, Priority: 99},
			loadInfo:   &AccountLoadInfo{LoadRate: 90, WaitingCount: 5},
			score:      1.0,
			sortOrder:  10,
			groupOrder: true,
			groupPrio:  50,
		},
		{
			account:    &Account{ID: 22, Priority: 1},
			loadInfo:   &AccountLoadInfo{LoadRate: 0, WaitingCount: 0},
			score:      1000.0,
			sortOrder:  20,
			groupOrder: true,
			groupPrio:  1,
		},
	}

	order := buildOpenAIOrderedSelectionOrder(candidates)
	require.Len(t, order, len(candidates))
	require.Equal(t, int64(21), order[0].account.ID)
	require.Equal(t, int64(22), order[1].account.ID)
}

func TestBuildOpenAIOrderedSelectionOrder_GroupOrderUsesScoreWithinSameBucket(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:    &Account{ID: 31, Priority: 1},
			loadInfo:   &AccountLoadInfo{LoadRate: 95, WaitingCount: 4},
			score:      0.2,
			sortOrder:  10,
			groupOrder: true,
			groupPrio:  10,
		},
		{
			account:    &Account{ID: 32, Priority: 99},
			loadInfo:   &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:      0.9,
			sortOrder:  10,
			groupOrder: true,
			groupPrio:  10,
		},
	}

	order := buildOpenAIOrderedSelectionOrder(candidates)
	require.Len(t, order, len(candidates))
	require.Equal(t, int64(32), order[0].account.ID)
	require.Equal(t, int64(31), order[1].account.ID)
}

func TestBuildOpenAIOrderedSelectionOrder_Deterministic(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 101},
			loadInfo: &AccountLoadInfo{LoadRate: 10, WaitingCount: 0},
			score:    4.2,
		},
		{
			account:  &Account{ID: 102},
			loadInfo: &AccountLoadInfo{LoadRate: 30, WaitingCount: 1},
			score:    3.5,
		},
		{
			account:  &Account{ID: 103},
			loadInfo: &AccountLoadInfo{LoadRate: 50, WaitingCount: 2},
			score:    2.1,
		},
	}

	first := buildOpenAIOrderedSelectionOrder(candidates)
	second := buildOpenAIOrderedSelectionOrder(candidates)
	require.Len(t, first, len(candidates))
	require.Len(t, second, len(candidates))
	for i := range first {
		require.Equal(t, first[i].account.ID, second[i].account.ID)
	}
}

func TestBuildOpenAIOrderedSelectionOrderWithSeed_OverridesLRUOnlyWithinSameRank(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Hour)
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 5101, Priority: 0, LastUsedAt: &now},
			loadInfo: &AccountLoadInfo{AccountID: 5101, LoadRate: 20, WaitingCount: 1},
			score:    1,
		},
		{
			account:  &Account{ID: 5102, Priority: 0, LastUsedAt: &old},
			loadInfo: &AccountLoadInfo{AccountID: 5102, LoadRate: 20, WaitingCount: 1},
			score:    1,
		},
		{
			account:  &Account{ID: 5103, Priority: 0},
			loadInfo: &AccountLoadInfo{AccountID: 5103, LoadRate: 20, WaitingCount: 1},
			score:    1,
		},
	}

	legacy := buildOpenAIOrderedSelectionOrder(candidates)
	require.Equal(t, []int64{5103, 5102, 5101}, openAIStableLowTTFTCandidateIDs(legacy))

	seed := newSchedulerSeededOrder("group", "model", "endpoint", "session-a")
	seeded := buildOpenAIOrderedSelectionOrderWithSeed(candidates, seed)
	seededAgain := buildOpenAIOrderedSelectionOrderWithSeed(candidates, seed)

	require.Equal(t, openAIStableLowTTFTCandidateIDs(seeded), openAIStableLowTTFTCandidateIDs(seededAgain))
	require.NotEqual(t, openAIStableLowTTFTCandidateIDs(legacy), openAIStableLowTTFTCandidateIDs(seeded))
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_OrderedSelectionUsesSessionSeedAcrossSessions(t *testing.T) {
	ctx := context.Background()
	groupID := int64(15)
	accounts := []Account{
		{
			ID:          5101,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 3,
			Priority:    0,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          5102,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 3,
			Priority:    0,
			GroupIDs:    []int64{groupID},
		},
		{
			ID:          5103,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 3,
			Priority:    0,
			GroupIDs:    []int64{groupID},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 1

	concurrencyCache := schedulerTestConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			5101: {AccountID: 5101, LoadRate: 20, WaitingCount: 1},
			5102: {AccountID: 5102, LoadRate: 20, WaitingCount: 1},
			5103: {AccountID: 5103, LoadRate: 20, WaitingCount: 1},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cache:              &schedulerTestGatewayCache{sessionBindings: map[string]int64{}},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	selected := make(map[int64]int, len(accounts))
	for i := 0; i < 60; i++ {
		sessionHash := fmt.Sprintf("session-seed-%02d", i)
		selection, decision, err := svc.SelectAccountWithScheduler(
			ctx,
			&groupID,
			"",
			sessionHash,
			"gpt-5.1",
			nil,
			OpenAIUpstreamTransportAny,
			false,
		)
		require.NoError(t, err)
		require.NotNil(t, selection)
		require.NotNil(t, selection.Account)
		require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
		selected[selection.Account.ID]++
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	}

	require.GreaterOrEqual(t, len(selected), 2, "different sessions should not all pin to the first account")

	repeatedSession := "session-seed-stable"
	first, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		repeatedSession,
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, first.Account)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	if first.ReleaseFunc != nil {
		first.ReleaseFunc()
	}

	second, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		repeatedSession,
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, second)
	require.NotNil(t, second.Account)
	require.Equal(t, first.Account.ID, second.Account.ID, "same session should keep its sticky account")
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	if second.ReleaseFunc != nil {
		second.ReleaseFunc()
	}
}

func TestBuildOpenAIOrderedSelectionOrder_HandlesInvalidScores(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{
			account:  &Account{ID: 901},
			loadInfo: &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:    math.NaN(),
		},
		{
			account:  &Account{ID: 902},
			loadInfo: &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:    math.Inf(1),
		},
		{
			account:  &Account{ID: 903},
			loadInfo: &AccountLoadInfo{LoadRate: 5, WaitingCount: 0},
			score:    -1,
		},
	}

	order := buildOpenAIOrderedSelectionOrder(candidates)
	require.Len(t, order, len(candidates))
	seen := map[int64]struct{}{}
	for _, item := range order {
		seen[item.account.ID] = struct{}{}
	}
	require.Len(t, seen, len(candidates))
}

func TestOpenAICodexAccountCircuit_GlobalAcrossGroupsAndHalfOpenRecovery(t *testing.T) {
	ctx := context.Background()
	groups := []int64{2, 10, 12}
	primary := Account{
		ID:          38797,
		Name:        "codex-primary",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 3,
		Priority:    0,
		GroupIDs:    groups,
	}
	backup := Account{
		ID:          38801,
		Name:        "codex-backup",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 3,
		Priority:    0,
		GroupIDs:    groups,
	}
	for _, groupID := range groups {
		primary.AccountGroups = append(primary.AccountGroups, AccountGroup{
			AccountID:            primary.ID,
			GroupID:              groupID,
			SortOrder:            10,
			Priority:             1,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SchedulingConfigured: true,
		})
		backup.AccountGroups = append(backup.AccountGroups, AccountGroup{
			AccountID:            backup.ID,
			GroupID:              groupID,
			SortOrder:            20,
			Priority:             1,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SchedulingConfigured: true,
		})
	}

	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{primary, backup}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}
	selectID := func(groupID int64) int64 {
		t.Helper()
		selection, decision, err := svc.SelectAccountWithScheduler(
			ctx,
			&groupID,
			"",
			"",
			"gpt-5.1",
			nil,
			OpenAIUpstreamTransportAny,
			false,
		)
		require.NoError(t, err)
		require.NotNil(t, selection)
		require.NotNil(t, selection.Account)
		require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		return selection.Account.ID
	}

	for _, groupID := range groups {
		require.Equal(t, int64(38797), selectID(groupID), "healthy primary should stay first for group %d", groupID)
	}

	svc.schedulerHealth.reportFailure(primary.ID, "gpt-5.1", "/v1/responses", "transient", time.Minute)
	for _, groupID := range groups {
		require.Equal(t, int64(38801), selectID(groupID), "open global circuit should skip primary for group %d", groupID)
	}
	selection, _, err := svc.SelectAccountWithScheduler(
		ctx,
		&groups[0],
		"",
		"",
		"gpt-5.1",
		map[int64]struct{}{backup.ID: struct{}{}},
		OpenAIUpstreamTransportAny,
		false,
	)
	require.Error(t, err)
	require.Nil(t, selection)
	require.Contains(t, err.Error(), "no available OpenAI accounts")

	svc.schedulerHealth.reportSuccess(primary.ID, "gpt-5.1", "/v1/responses", nil)
	svc.schedulerHealth.reportFailure(primary.ID, "gpt-5.1", "/v1/responses", "transient", time.Nanosecond)
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(primary.ID, "gpt-5.1", "/v1/responses", false)
		return snap.CircuitState == schedulerCircuitHalfOpen
	}, time.Second, time.Millisecond)
	for _, groupID := range groups {
		require.Equal(t, int64(38801), selectID(groupID), "expired circuit should wait for background probe, not user traffic, for group %d", groupID)
	}

	svc.ReportOpenAIAccountScheduleResultForRequest(primary.ID, "gpt-5.1", "/v1/responses", true, nil)
	require.Equal(t, int64(38797), selectID(12), "successful half-open probe should restore primary order")

	svc.schedulerHealth.reportFailure(primary.ID, "gpt-5.1", "/v1/responses", "transient", time.Nanosecond)
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(primary.ID, "gpt-5.1", "/v1/responses", false)
		return snap.CircuitState == schedulerCircuitHalfOpen
	}, time.Second, time.Millisecond)
	require.Equal(t, int64(38801), selectID(2), "later half-open state should still wait for background probe")
	svc.ReportOpenAIAccountScheduleResultForRequest(primary.ID, "gpt-5.1", "/v1/responses", false, nil)
	require.Equal(t, int64(38801), selectID(10), "failed half-open result should not recover the primary")
}

func TestOpenAIStreamInterruptionCircuit_DoesNotReplayAfterClientOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          38797,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	svc := &OpenAIGatewayService{
		accountRepo:                     schedulerTestOpenAIAccountRepo{accounts: []Account{*account}},
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	_, _ = c.Writer.Write([]byte("data: prelude\n\n"))

	failoverErr := svc.handleOpenAIUpstreamStreamError(
		context.Background(),
		c,
		account,
		errors.New("stream data interval timeout"),
		"",
		false,
	)

	require.Nil(t, failoverErr, "stream already committed, handler must not replay on another account")
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account), "stream transport errors must not globally remove a shared account from every group")
}

func TestClamp01_AllBranches(t *testing.T) {
	require.Equal(t, 0.0, clamp01(-0.2))
	require.Equal(t, 1.0, clamp01(1.3))
	require.Equal(t, 0.5, clamp01(0.5))
}

func TestCalcLoadSkewByMoments_Branches(t *testing.T) {
	require.Equal(t, 0.0, calcLoadSkewByMoments(1, 1, 1))
	// variance < 0 分支：sumSquares/count - mean^2 为负值时应钳制为 0。
	require.Equal(t, 0.0, calcLoadSkewByMoments(1, 0, 2))
	require.GreaterOrEqual(t, calcLoadSkewByMoments(6, 20, 3), 0.0)
}

func TestDefaultOpenAIAccountScheduler_ReportSwitchAndSnapshot(t *testing.T) {
	schedulerAny := newDefaultOpenAIAccountScheduler(&OpenAIGatewayService{}, nil)
	scheduler, ok := schedulerAny.(*defaultOpenAIAccountScheduler)
	require.True(t, ok)

	ttft := 100
	scheduler.ReportResult(1001, true, &ttft)
	scheduler.ReportSwitch()
	scheduler.metrics.recordSelect(OpenAIAccountScheduleDecision{
		Layer:             openAIAccountScheduleLayerLoadBalance,
		LatencyMs:         8,
		LoadSkew:          0.5,
		StickyPreviousHit: true,
	})
	scheduler.metrics.recordSelect(OpenAIAccountScheduleDecision{
		Layer:            openAIAccountScheduleLayerSessionSticky,
		LatencyMs:        6,
		LoadSkew:         0.2,
		StickySessionHit: true,
	})

	snapshot := scheduler.SnapshotMetrics()
	require.Equal(t, int64(2), snapshot.SelectTotal)
	require.Equal(t, int64(1), snapshot.StickyPreviousHitTotal)
	require.Equal(t, int64(1), snapshot.StickySessionHitTotal)
	require.Equal(t, int64(1), snapshot.LoadBalanceSelectTotal)
	require.Equal(t, int64(1), snapshot.AccountSwitchTotal)
	require.Greater(t, snapshot.SchedulerLatencyMsAvg, 0.0)
	require.Greater(t, snapshot.StickyHitRatio, 0.0)
	require.Greater(t, snapshot.LoadSkewAvg, 0.0)
}

func TestOpenAIGatewayService_SchedulerWrappersAndDefaults(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	svc := &OpenAIGatewayService{}
	ttft := 120
	svc.ReportOpenAIAccountScheduleResult(10, true, &ttft)
	svc.RecordOpenAIAccountSwitch()
	snapshot := svc.SnapshotOpenAIAccountSchedulerMetrics()
	require.Equal(t, OpenAIAccountSchedulerMetricsSnapshot{}, snapshot)
	require.Equal(t, openaiStickySessionTTL, svc.openAIWSSessionStickyTTL())

	defaultWeights := svc.openAIWSSchedulerWeights()
	require.Equal(t, 1.0, defaultWeights.Priority)
	require.Equal(t, 1.0, defaultWeights.Load)
	require.Equal(t, 0.7, defaultWeights.Queue)
	require.Equal(t, 0.8, defaultWeights.ErrorRate)
	require.Equal(t, 0.5, defaultWeights.TTFT)

	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 180
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 0.2
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 0.3
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 0.4
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 0.5
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 0.6
	svcWithCfg := &OpenAIGatewayService{cfg: cfg}

	require.Equal(t, 180*time.Second, svcWithCfg.openAIWSSessionStickyTTL())
	customWeights := svcWithCfg.openAIWSSchedulerWeights()
	require.Equal(t, 0.2, customWeights.Priority)
	require.Equal(t, 0.3, customWeights.Load)
	require.Equal(t, 0.4, customWeights.Queue)
	require.Equal(t, 0.5, customWeights.ErrorRate)
	require.Equal(t, 0.6, customWeights.TTFT)
}

func TestDefaultOpenAIAccountScheduler_IsAccountTransportCompatible_Branches(t *testing.T) {
	scheduler := &defaultOpenAIAccountScheduler{}
	require.True(t, scheduler.isAccountTransportCompatible(nil, OpenAIUpstreamTransportAny))
	require.True(t, scheduler.isAccountTransportCompatible(nil, OpenAIUpstreamTransportHTTPSSE))
	require.False(t, scheduler.isAccountTransportCompatible(nil, OpenAIUpstreamTransportResponsesWebsocketV2))

	cfg := newSchedulerTestOpenAIWSV2Config()
	scheduler.service = &OpenAIGatewayService{cfg: cfg}
	account := &Account{
		ID:          8801,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
		},
	}
	require.True(t, scheduler.isAccountTransportCompatible(account, OpenAIUpstreamTransportResponsesWebsocketV2))
}

func TestOpenAIStableLowTTFTGroupDetection(t *testing.T) {
	require.True(t, isOpenAIStableLowTTFTGroup(&Group{Name: "Codex(特价)", Platform: PlatformOpenAI, OpenAIStableLowTTFT: true}))
	require.False(t, isOpenAIStableLowTTFTGroup(&Group{Name: "Codex(稳定)", Platform: PlatformOpenAI}))
	require.False(t, isOpenAIStableLowTTFTGroup(&Group{Name: "Claude(稳定)", Platform: PlatformAnthropic, OpenAIStableLowTTFT: true}))
	require.False(t, isOpenAIStableLowTTFTGroup(nil))
}

func TestOpenAIStableLowTTFTSelectionOrderColdStartRotatesUnknownAccounts(t *testing.T) {
	pool := []openAIAccountCandidateScore{
		stableLowTTFTCandidateForTest(1, 10, 0, false),
		stableLowTTFTCandidateForTest(2, 20, 0, false),
		stableLowTTFTCandidateForTest(3, 30, 0, false),
	}

	first := buildOpenAIStableLowTTFTSelectionOrder(pool, 0)
	second := buildOpenAIStableLowTTFTSelectionOrder(pool, 1)
	third := buildOpenAIStableLowTTFTSelectionOrder(pool, 2)

	require.Equal(t, []int64{1, 2, 3}, openAIStableLowTTFTCandidateIDs(first))
	require.Equal(t, []int64{2, 3, 1}, openAIStableLowTTFTCandidateIDs(second))
	require.Equal(t, []int64{3, 1, 2}, openAIStableLowTTFTCandidateIDs(third))
}

func TestOpenAIStableLowTTFTSelectionOrderPrefersLowestKnownTTFT(t *testing.T) {
	pool := []openAIAccountCandidateScore{
		stableLowTTFTCandidateForTest(1, 10, 1800, true),
		stableLowTTFTCandidateForTest(2, 20, 350, true),
		stableLowTTFTCandidateForTest(3, 30, 0, false),
	}

	normal := buildOpenAIStableLowTTFTSelectionOrder(pool, 1)
	later := buildOpenAIStableLowTTFTSelectionOrder(pool, 8)

	require.Equal(t, []int64{2, 1, 3}, openAIStableLowTTFTCandidateIDs(normal))
	require.Equal(t, []int64{2, 1, 3}, openAIStableLowTTFTCandidateIDs(later))
}

func TestOpenAIStableLowTTFTSelectionUsesDefaultProbeTTFTFallback(t *testing.T) {
	health := newAccountSchedulerHealthStats()
	health.reportSuccess(1, defaultSchedulerModel, defaultSchedulerEndpoint, intPtrForTest(1800))
	health.reportSuccess(2, defaultSchedulerModel, defaultSchedulerEndpoint, intPtrForTest(350))

	groupID := int64(10)
	accounts := []*Account{
		{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1},
		{ID: 2, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 2},
	}
	scores := buildSchedulerAccountScores(accounts, &groupID, "gpt-5.5", "/v1/responses", nil, health, false)
	pool := openAIAccountCandidatesFromSchedulerScores(scores)
	order := buildOpenAIStableLowTTFTSelectionOrder(pool, 1)

	require.Equal(t, []int64{2, 1}, openAIStableLowTTFTCandidateIDs(order))
	require.True(t, order[0].hasTTFT)
	require.Equal(t, float64(350), order[0].ttft)
}

func TestResolveOpenAIStableTTFTProbeModelOnlyUsesConfiguredModels(t *testing.T) {
	allModels := &Account{
		Platform:    PlatformOpenAI,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.4-mini": "gpt-5.4-mini", "gpt-5.4": "gpt-5.4"}},
	}
	require.Equal(t, "gpt-5.4-mini", resolveOpenAIStableTTFTProbeModel(allModels))

	withoutMini := &Account{
		Platform:    PlatformOpenAI,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.5": "gpt-5.5", "gpt-5.4": "gpt-5.4"}},
	}
	require.Equal(t, "gpt-5.5", resolveOpenAIStableTTFTProbeModel(withoutMini))

	unsupported := &Account{
		Platform:    PlatformOpenAI,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.1": "gpt-5.1", "gpt-5": "gpt-5"}},
	}
	require.Empty(t, resolveOpenAIStableTTFTProbeModel(unsupported))
}

func stableLowTTFTCandidateForTest(id int64, sortOrder int, ttft float64, hasTTFT bool) openAIAccountCandidateScore {
	return openAIAccountCandidateScore{
		account:   &Account{ID: id, Priority: sortOrder},
		score:     1,
		ttft:      ttft,
		hasTTFT:   hasTTFT,
		sortOrder: sortOrder,
		groupPrio: sortOrder,
		loadInfo:  &AccountLoadInfo{AccountID: id},
	}
}

func openAIStableLowTTFTCandidateIDs(candidates []openAIAccountCandidateScore) []int64 {
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, openAIAccountCandidateID(candidate))
	}
	return ids
}
