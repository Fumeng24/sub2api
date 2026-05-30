//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAI429FastPath_MarksOAuthAccountCoolingDown(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	apiKeyAccount := &Account{ID: 43, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusTooManyRequests, http.Header{}, nil)
	apiKeyShouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), apiKeyAccount, http.StatusTooManyRequests, http.Header{}, nil)

	require.False(t, shouldDisable)
	require.False(t, apiKeyShouldDisable)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(apiKeyAccount))
}

func TestOpenAIRuntimeBlock_AppliesToOpenAIAPIKeyWhenRateLimitServiceStopsScheduling(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 44, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	svc.BlockAccountScheduling(account, time.Time{}, "custom_error_code")

	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIRuntimeBlock_DoesNotApplyToOtherPlatforms(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 45, Platform: PlatformGemini, Type: AccountTypeOAuth}

	svc.BlockAccountScheduling(account, time.Time{}, "custom_error_code")

	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIRuntimeBlocker_IgnoresNonOpenAIFromRateLimitService(t *testing.T) {
	gateway := &OpenAIGatewayService{}
	repo := &rateLimitAccountRepoStub{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetAccountRuntimeBlocker(gateway)
	account := &Account{ID: 45, Platform: PlatformGemini, Type: AccountTypeOAuth}

	shouldDisable := rateLimitService.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte("forbidden"))

	require.True(t, shouldDisable)
	require.False(t, gateway.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIModelNotFound_DoesNotRuntimeBlockWholeAccount(t *testing.T) {
	repo := &modelNotFoundAccountRepoStub{}
	svc := &OpenAIGatewayService{
		rateLimitService: &RateLimitService{accountRepo: repo},
	}
	account := openAIModelNotFoundTempAccount()

	shouldDisable := svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		http.StatusNotFound,
		http.Header{},
		[]byte(`{"error":{"code":"model_not_found","message":"model not found"}}`),
		"gpt-5.4",
	)

	require.True(t, shouldDisable)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Zero(t, repo.tempCalls)
	require.Len(t, repo.modelRateLimitCalls, 1)
}

func TestOpenAIAdvancedSchedulerPolicy_CustomErrorCodesAndTransient5xx(t *testing.T) {
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)

	account := &Account{
		ID:       102,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(599)},
		},
	}

	t.Run("advanced_scheduler_handles_error_even_when_custom_code_misses", func(t *testing.T) {
		svc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("true")}

		shouldDisable := svc.handleOpenAIAccountUpstreamError(
			context.Background(),
			account,
			http.StatusBadGateway,
			http.Header{},
			[]byte(`{"error":{"message":"upstream failed"}}`),
		)

		require.True(t, shouldDisable)
	})

	t.Run("legacy_scheduler_short_cools_transient_5xx", func(t *testing.T) {
		svc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("false")}

		shouldDisable := svc.handleOpenAIAccountUpstreamError(
			context.Background(),
			account,
			http.StatusBadGateway,
			http.Header{},
			[]byte(`{"error":{"message":"upstream failed"}}`),
		)

		require.True(t, shouldDisable)
	})

	t.Run("legacy_scheduler_keeps_custom_code_skip_semantics_for_non_5xx", func(t *testing.T) {
		svc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("false")}

		shouldDisable := svc.handleOpenAIAccountUpstreamError(
			context.Background(),
			account,
			http.StatusTeapot,
			http.Header{},
			[]byte(`{"error":{"message":"teapot"}}`),
		)

		require.False(t, shouldDisable)
	})
}

func TestOpenAITransient5xxShortCooldownIgnoresCustomCodeSkip(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		rateLimitService:                rateLimitService,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{
		ID:          105,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(599)},
		},
	}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		524,
		http.Header{},
		[]byte(`{"error":{"message":"Cloudflare timeout"}}`),
	)

	require.True(t, shouldDisable)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 1, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
	require.True(t, repo.lastTempUntil.After(time.Now()))

	var state TempUnschedState
	require.NoError(t, json.Unmarshal([]byte(repo.lastTempReason), &state))
	require.Equal(t, 524, state.StatusCode)
	require.Equal(t, "openai_transient_5xx", state.MatchedKeyword)
	require.True(t, state.UntilUnix > time.Now().Unix())
}

func TestOpenAITransientCooldownPersistThrottleKeepsRuntimeBlock(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{
		ID:          106,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
	}

	first := svc.markOpenAITransient5xxCoolingDown(context.Background(), account, 524, http.Header{}, []byte(`{"error":{"message":"timeout"}}`))
	second := svc.markOpenAITransient5xxCoolingDown(context.Background(), account, 524, http.Header{}, []byte(`{"error":{"message":"timeout again"}}`))

	require.True(t, first)
	require.True(t, second)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 1, repo.tempCalls)
}

func TestOpenAIRequestErrorFailoversAndCoolsAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{ID: 107, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	safeErr, failoverErr := svc.handleOpenAIUpstreamRequestError(
		context.Background(),
		c,
		account,
		errors.New("dial tcp: connection refused"),
		"https://api.openai.com/v1/responses",
		false,
	)

	require.NotEmpty(t, safeErr)
	require.NotNil(t, failoverErr)
	require.Equal(t, 0, failoverErr.StatusCode)
	require.Equal(t, 0, w.Body.Len(), "failover path must not write the response")
	require.Equal(t, 1, repo.tempCalls)
	require.True(t, repo.lastTempUntil.After(time.Now()))
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))

	var state TempUnschedState
	require.NoError(t, json.Unmarshal([]byte(repo.lastTempReason), &state))
	require.Equal(t, 0, state.StatusCode)
	require.Equal(t, "openai_request_error", state.MatchedKeyword)

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "failover", events[0].Kind)
	require.True(t, events[0].CooldownApplied)
	require.Equal(t, "openai_request_error", events[0].CooldownReason)
}

func TestOpenAIRequestErrorDoesNotFailoverWhenRequestCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{ID: 108, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	safeErr, failoverErr := svc.handleOpenAIUpstreamRequestError(
		ctx,
		c,
		account,
		errors.New("context canceled"),
		"",
		false,
	)

	require.NotEmpty(t, safeErr)
	require.Nil(t, failoverErr)
	require.Equal(t, 0, repo.tempCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "request_error", events[0].Kind)
	require.False(t, events[0].CooldownApplied)
	require.Empty(t, events[0].CooldownReason)
}

func TestOpenAIRequestErrorFailoversWhenCanceledContextHasNetworkTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{ID: 109, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	safeErr, failoverErr := svc.handleOpenAIUpstreamRequestError(
		ctx,
		c,
		account,
		errors.New("dial tcp 10.0.0.1:443: i/o timeout"),
		"",
		false,
	)

	require.NotEmpty(t, safeErr)
	require.NotNil(t, failoverErr)
	require.Equal(t, 0, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "i/o timeout")
	require.Equal(t, 0, w.Body.Len(), "failover path must not write the response")
	require.Equal(t, 1, repo.tempCalls)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "failover", events[0].Kind)
	require.True(t, events[0].CooldownApplied)
	require.Equal(t, "openai_request_error", events[0].CooldownReason)
}

func TestOpsUpstreamErrorAnnotatesOpenAITransientFailoverCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           PlatformOpenAI,
		AccountID:          109,
		UpstreamStatusCode: http.StatusGatewayTimeout,
		Kind:               "failover",
		Message:            "gateway timeout",
	})

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.True(t, events[0].CooldownApplied)
	require.Equal(t, "openai_transient_5xx", events[0].CooldownReason)
}

func TestOpenAIAdvancedSchedulerPolicy_DisablesLegacySameAccountRetry(t *testing.T) {
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)

	account := &Account{
		ID:       103,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode":                    true,
			"pool_mode_retry_status_codes": []any{float64(http.StatusTooManyRequests)},
		},
	}

	legacySvc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("false")}
	require.True(t, legacySvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusTooManyRequests))

	advancedSvc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("true")}
	require.False(t, advancedSvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusTooManyRequests))
}

func TestOpenAIAdvancedSchedulerPolicy_FailoversModelNotFound(t *testing.T) {
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)

	account := &Account{ID: 104, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"code":"model_not_found","message":"model not found"}}`)

	legacySvc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("false")}
	require.False(t, legacySvc.shouldFailoverOpenAIUpstreamResponseForAccount(context.Background(), account, http.StatusNotFound, "model not found", body))

	advancedSvc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("true")}
	require.True(t, advancedSvc.shouldFailoverOpenAIUpstreamResponseForAccount(context.Background(), account, http.StatusNotFound, "model not found", body))
}

func TestOpenAIRuntimeBlock_DoesNotShortenExistingBlock(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 46, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	longUntil := time.Now().Add(10 * time.Minute)

	svc.BlockAccountScheduling(account, longUntil, "oauth_401")
	svc.BlockAccountScheduling(account, time.Time{}, "upstream_disable")

	value, ok := svc.openaiAccountRuntimeBlockUntil.Load(account.ID)
	require.True(t, ok)
	actualUntil, ok := value.(time.Time)
	require.True(t, ok)
	require.WithinDuration(t, longUntil, actualUntil, time.Second)
}

func TestOpenAIRuntimeBlock_ClearAccountSchedulingBlock(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 47, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	svc.BlockAccountScheduling(account, time.Now().Add(time.Minute), "429")
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))

	svc.ClearAccountSchedulingBlock(account.ID)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestShouldStopOpenAIOAuth429Failover_OnlyDuringStorm(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	apiKeyAccount := &Account{ID: 43, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	require.False(t, svc.ShouldStopOpenAIOAuth429Failover(account, http.StatusTooManyRequests, 1))

	for i := 0; i < openAIOAuth429StormThreshold; i++ {
		svc.recordOpenAIOAuth429()
	}

	require.True(t, svc.ShouldStopOpenAIOAuth429Failover(account, http.StatusTooManyRequests, 1))
	require.False(t, svc.ShouldStopOpenAIOAuth429Failover(apiKeyAccount, http.StatusTooManyRequests, 1))
	require.False(t, svc.ShouldStopOpenAIOAuth429Failover(account, http.StatusInternalServerError, 1))
	require.False(t, svc.ShouldStopOpenAIOAuth429Failover(account, http.StatusTooManyRequests, 0))
}
