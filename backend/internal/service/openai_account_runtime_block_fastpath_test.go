//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type closeIdleHTTPUpstreamStub struct {
	closed []int64
}

func (s *closeIdleHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return nil, errors.New("unexpected Do")
}

func (s *closeIdleHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return nil, errors.New("unexpected DoWithTLS")
}

func (s *closeIdleHTTPUpstreamStub) CloseIdleConnectionsForAccount(accountID int64) {
	s.closed = append(s.closed, accountID)
}

func TestOpenAI429FastPath_DoesNotGloballyBlockSharedAccount(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	apiKeyAccount := &Account{ID: 43, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusTooManyRequests, http.Header{}, nil)
	apiKeyShouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), apiKeyAccount, http.StatusTooManyRequests, http.Header{}, nil)

	require.False(t, shouldDisable)
	require.False(t, apiKeyShouldDisable)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
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
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 0, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
}

func TestOpenAITransientCooldownDoesNotPersistGlobalRuntimeBlock(t *testing.T) {
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
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 0, repo.tempCalls)
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
	require.Equal(t, 0, repo.tempCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "failover", events[0].Kind)
	require.False(t, events[0].CooldownApplied)
	require.Empty(t, events[0].CooldownReason)
}

func TestOpenAIRequestErrorFailoversOnProxyAuthenticationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{ID: 117, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	safeErr, failoverErr := svc.handleOpenAIUpstreamRequestError(
		context.Background(),
		c,
		account,
		errors.New(`Post "https://chatgpt.com/backend-api/codex/responses": socks connect tcp 1.2.3.4:1080->chatgpt.com:443: username/password authentication failed`),
		"https://api.openai.com/v1/responses",
		false,
	)

	require.Contains(t, safeErr, "authentication failed")
	require.NotNil(t, failoverErr)
	require.Equal(t, 0, w.Body.Len(), "failover path must not write the response")
	require.Equal(t, 0, repo.tempCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
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
	require.Equal(t, 0, repo.tempCalls)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "failover", events[0].Kind)
	require.False(t, events[0].CooldownApplied)
	require.Empty(t, events[0].CooldownReason)
}

func TestOpenAIRequestErrorClosesIdleConnectionsOnConnectionReset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	upstream := &closeIdleHTTPUpstreamStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                     repo,
		httpUpstream:                    upstream,
		openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour),
	}
	account := &Account{ID: 110, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	_, failoverErr := svc.handleOpenAIUpstreamRequestError(
		context.Background(),
		c,
		account,
		errors.New("read tcp 10.0.0.2:443: connection reset by peer"),
		"",
		false,
	)

	require.NotNil(t, failoverErr)
	require.Equal(t, []int64{account.ID}, upstream.closed)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIStreamDiagnosticErrorsFailoverWithoutCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	upstream := &closeIdleHTTPUpstreamStub{}
	svc := &OpenAIGatewayService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}
	account := &Account{ID: 111, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "usage_incomplete", err: errors.New("stream usage incomplete: missing terminal event")},
		{name: "missing_terminal", err: errors.New("missing terminal event")},
		{name: "upstream_ended_without_terminal", err: errors.New("upstream stream ended without terminal event")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

			failoverErr := svc.handleOpenAIUpstreamStreamError(context.Background(), c, account, tc.err, "", false)

			require.NotNil(t, failoverErr)
			require.Equal(t, 0, failoverErr.StatusCode)
			require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
			require.Zero(t, repo.tempCalls)
			require.Empty(t, upstream.closed)
		})
	}
}

func TestOpenAIStreamClientCancelDoesNotFailoverOrCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 111, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	failoverErr := svc.handleOpenAIUpstreamStreamError(context.Background(), c, account, context.Canceled, "", false)

	require.Nil(t, failoverErr)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Zero(t, repo.tempCalls)
}

func TestOpenAIStreamFailoverDiagnosticMessageDoesNotCooldownAccount(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 112, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}

	failoverErr := svc.newOpenAIStreamFailoverError(nil, account, false, "", nil, "OpenAI stream ended before a terminal event")

	require.NotNil(t, failoverErr)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Zero(t, repo.tempCalls)
}

func TestOpenAIRuntimeBlock_ReopenDoesNotPersistStatusZero(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 113, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}

	svc.reopenOpenAIAccountCircuit(account.ID, "openai_stream_error", time.Minute)

	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Zero(t, repo.tempCalls)
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

func TestOpenAIAdvancedSchedulerPolicy_RetriesTransientFailuresOnSameAccount(t *testing.T) {
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)

	account := &Account{
		ID:       103,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode":                    true,
			"pool_mode_retry_status_codes": []any{float64(http.StatusPaymentRequired), float64(http.StatusForbidden), float64(http.StatusTooManyRequests), float64(http.StatusBadGateway)},
		},
	}

	legacySvc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("false")}
	require.False(t, legacySvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusPaymentRequired))
	require.False(t, legacySvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusForbidden))
	require.False(t, legacySvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusTooManyRequests))
	require.True(t, legacySvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusBadGateway))

	advancedSvc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("true")}
	require.False(t, advancedSvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusPaymentRequired))
	require.False(t, advancedSvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusForbidden))
	require.False(t, advancedSvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusTooManyRequests))
	require.True(t, advancedSvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, http.StatusBadGateway))
	require.True(t, advancedSvc.retryableOnSameOpenAIAccountStatus(context.Background(), account, 0))
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

func TestOpenAIForbiddenBusinessErrorDoesNotFailover(t *testing.T) {
	account := &Account{ID: 114, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"message":"Image generation is not enabled for this group"}}`)
	svc := &OpenAIGatewayService{}

	require.Equal(t, openAIUpstreamErrorBusiness, classifyOpenAIUpstreamError(http.StatusForbidden, "", body))
	require.False(t, svc.shouldFailoverOpenAIUpstreamResponseForAccount(context.Background(), account, http.StatusForbidden, "", body))
}

func TestOpenAIGroupDisabledForbiddenFailsOver(t *testing.T) {
	account := &Account{ID: 116, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"code":"GROUP_DISABLED","message":"API Key 所属分组已停用"}`)
	svc := &OpenAIGatewayService{}

	require.Equal(t, "GROUP_DISABLED", extractUpstreamErrorCode(body))
	require.Equal(t, openAIUpstreamErrorAuth, classifyOpenAIUpstreamError(http.StatusForbidden, "", body))
	require.True(t, svc.shouldFailoverOpenAIUpstreamResponseForAccount(context.Background(), account, http.StatusForbidden, "", body))
}

func TestOpenAITransientScheduleFailureUsesScopedSchedulerCooldown(t *testing.T) {
	svc := &OpenAIGatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	accountID := int64(115)
	model := "gpt-5.5"
	endpoint := "/v1/responses"
	failoverErr := &UpstreamFailoverError{
		StatusCode:   http.StatusBadGateway,
		ResponseBody: []byte(`{"error":{"message":"bad gateway"}}`),
	}

	svc.ReportOpenAIAccountScheduleFailure(accountID, model, endpoint, failoverErr)

	require.False(t, svc.isOpenAIAccountRuntimeBlocked(&Account{ID: accountID, Platform: PlatformOpenAI}))
	sameScope := svc.schedulerHealth.snapshot(accountID, model, endpoint, false)
	require.Equal(t, schedulerCircuitOpen, sameScope.CircuitState)
	otherModel := svc.schedulerHealth.snapshot(accountID, "gpt-5.4", endpoint, false)
	require.Equal(t, schedulerCircuitClosed, otherModel.CircuitState)
}

func TestOpenAITransientTransportScheduleFailureUsesConfiguredCooldown(t *testing.T) {
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIScheduler: config.GatewayOpenAISchedulerConfig{
					RuntimeCooldowns: config.GatewayOpenAIRuntimeCooldownsConfig{
						RequestErrorCooldownSeconds: 2,
					},
				},
			},
		},
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	accountID := int64(119)
	model := "gpt-5.5"
	endpoint := "/v1/responses"
	failoverErr := &UpstreamFailoverError{SchedulerCategory: "transient_transport"}

	before := time.Now()
	svc.ReportOpenAIAccountScheduleFailure(accountID, model, endpoint, failoverErr)

	snap := svc.schedulerHealth.snapshot(accountID, model, endpoint, false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
	require.Equal(t, "transient_transport", snap.LastFailureReason)
	remaining := snap.CooldownUntil.Sub(before)
	require.GreaterOrEqual(t, remaining, 1500*time.Millisecond)
	require.LessOrEqual(t, remaining, 2500*time.Millisecond)
}

func TestOpenAI403ScheduleFailureOverrideUsesTransientProbeCircuit(t *testing.T) {
	svc := &OpenAIGatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	accountID := int64(117)
	model := "gpt-5.5"
	endpoint := "/v1/responses"
	failoverErr := &UpstreamFailoverError{
		StatusCode:        http.StatusForbidden,
		ResponseBody:      []byte(`{"error":{"message":"403错误，请稍后再试"}}`),
		SchedulerCategory: "transient",
	}

	svc.ReportOpenAIAccountScheduleFailure(accountID, model, endpoint, failoverErr)

	snap := svc.schedulerHealth.snapshot(accountID, model, endpoint, false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
	require.Equal(t, "transient", snap.LastFailureReason)
	require.Less(t, time.Until(snap.CooldownUntil), 2*time.Minute)
	require.Greater(t, time.Until(snap.CooldownUntil), 10*time.Second)
}

func TestOpenAI403ScheduleFailureOverrideIgnoresBusinessForbidden(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 118, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"code":"GROUP_DISABLED","message":"API Key 所属分组已停用"}`)

	category := svc.schedulerCategoryOverrideForOpenAIUpstreamError(context.Background(), account, http.StatusForbidden, body)

	require.Empty(t, category)
}

func TestOpenAITerminalScheduleResultDoesNotRecoverHalfOpenCircuit(t *testing.T) {
	svc := &OpenAIGatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	accountID := int64(116)
	model := "gpt-5.5"
	endpoint := "/v1/responses"

	svc.schedulerHealth.reportFailure(accountID, model, endpoint, "transient", time.Minute)
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	entry, ok := svc.schedulerHealth.get(key)
	require.True(t, ok)
	entry.mu.Lock()
	entry.cooldownUntil = time.Now().Add(-time.Second)
	entry.mu.Unlock()

	require.True(t, svc.schedulerHealth.tryBeginHalfOpenProbe(accountID, model, endpoint))
	svc.ReportOpenAIAccountScheduleTerminal(accountID, model, endpoint)

	snap := svc.schedulerHealth.snapshot(accountID, model, endpoint, true)
	require.Equal(t, schedulerCircuitHalfOpen, snap.CircuitState)
	require.True(t, snap.HalfOpenProbe)
}

func TestOpenAIThinkingSignatureInvalid_DoesNotFailoverOrCooldown(t *testing.T) {
	body := []byte(`{"error":{"code":"thinking_signature_invalid","message":"thinking signature invalid"}}`)
	account := &Account{ID: 105, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	svc := &OpenAIGatewayService{}

	require.True(t, isOpenAIThinkingSignatureInvalidError(body, ""))
	require.False(t, svc.shouldFailoverOpenAIUpstreamResponseForAccount(context.Background(), account, http.StatusInternalServerError, "thinking signature invalid", body))
	require.False(t, svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusInternalServerError, http.Header{}, body))
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
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
