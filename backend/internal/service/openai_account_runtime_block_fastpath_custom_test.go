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
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
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

func TestOpenAI429FastPath_UsesShortCooldownInsteadOfGlobalDisable(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{rateLimitService: &RateLimitService{accountRepo: repo}}
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	apiKeyAccount := &Account{ID: 43, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusTooManyRequests, http.Header{}, nil)
	apiKeyShouldDisable := svc.handleOpenAIAccountUpstreamError(context.Background(), apiKeyAccount, http.StatusTooManyRequests, http.Header{}, nil)

	require.False(t, shouldDisable)
	require.False(t, apiKeyShouldDisable)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(apiKeyAccount))
	require.Equal(t, 2, repo.setRateLimitedCalls)
	require.Zero(t, repo.setErrorCalls)
}

func TestOpenAICompatibleAccountRequestEligibilityReasons(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(time.Minute)

	t.Run("platform mismatch is hard reason", func(t *testing.T) {
		account := &Account{ID: 201, Platform: PlatformGrok, Status: StatusActive, Schedulable: true}

		eligibility := openAICompatibleAccountRequestEligibility(ctx, account, PlatformOpenAI, "", false, "")

		require.False(t, eligibility.Allowed())
		require.Equal(t, "platform_mismatch", eligibility.HardReason)
		require.Equal(t, "platform_mismatch", eligibility.Reason())
	})

	t.Run("runtime account block is soft reason", func(t *testing.T) {
		account := &Account{
			ID:               202,
			Platform:         PlatformOpenAI,
			Status:           StatusActive,
			Schedulable:      true,
			RateLimitResetAt: &future,
		}

		eligibility := openAICompatibleAccountRequestEligibility(ctx, account, PlatformOpenAI, "", false, "")

		require.False(t, eligibility.Allowed())
		require.Empty(t, eligibility.HardReason)
		require.Equal(t, AccountSchedulingBlockRateLimited.String(), eligibility.SoftReason)
		require.Equal(t, AccountSchedulingBlockRateLimited.String(), eligibility.Reason())
	})

	t.Run("model rate limit only applies to requested model", func(t *testing.T) {
		resetAt := time.Now().Add(time.Minute).Format(time.RFC3339)
		account := &Account{
			ID:          204,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				modelRateLimitsKey: map[string]any{
					"gpt-5.4": map[string]any{
						"rate_limit_reset_at": resetAt,
					},
				},
			},
		}

		limited := openAICompatibleAccountRequestEligibility(ctx, account, PlatformOpenAI, "gpt-5.4", false, "")
		other := openAICompatibleAccountRequestEligibility(ctx, account, PlatformOpenAI, "gpt-5.3", false, "")

		require.False(t, limited.Allowed())
		require.Equal(t, "model_rate_limited", limited.SoftReason)
		require.True(t, other.Allowed())
	})

	t.Run("image rate limit only applies to image intent", func(t *testing.T) {
		resetAt := time.Now().Add(time.Minute).Format(time.RFC3339)
		account := &Account{
			ID:          205,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				modelRateLimitsKey: map[string]any{
					openAIImageGenerationRateLimitKey: map[string]any{
						"rate_limit_reset_at": resetAt,
					},
				},
			},
		}

		image := openAICompatibleAccountRequestEligibility(WithOpenAIImageGenerationIntent(ctx), account, PlatformOpenAI, "gpt-5.4", false, "")
		text := openAICompatibleAccountRequestEligibility(ctx, account, PlatformOpenAI, "gpt-5.4", false, "")

		require.False(t, image.Allowed())
		require.Equal(t, "model_rate_limited", image.SoftReason)
		require.True(t, text.Allowed())
	})

	t.Run("grok quota pause is shared eligibility", func(t *testing.T) {
		zero := int64(0)
		limit := int64(10)
		resetFuture := time.Now().Add(time.Minute).Unix()
		account := &Account{
			ID:          203,
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				grokQuotaSnapshotExtraKey: xai.QuotaSnapshot{
					Requests:  &xai.QuotaWindow{Limit: &limit, Remaining: &zero, ResetUnix: &resetFuture},
					UpdatedAt: time.Now().UTC().Format(time.RFC3339),
				},
			},
		}

		eligibility := openAICompatibleAccountRequestEligibility(ctx, account, PlatformGrok, "", false, "")

		require.False(t, eligibility.Allowed())
		require.True(t, eligibility.PausedByQuota)
		require.Equal(t, "quota_auto_paused", eligibility.Reason())
	})
}

func TestOpenAIContextWindowError_DoesNotUpdateAccountState(t *testing.T) {
	repo := &modelNotFoundAccountRepoStub{}
	svc := &OpenAIGatewayService{
		rateLimitService: &RateLimitService{accountRepo: repo},
	}
	account := &Account{ID: 46, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		http.StatusBadRequest,
		http.Header{},
		[]byte(`{"error":{"type":"invalid_request_error","code":"context_length_exceeded","message":"Your input exceeds the context window."}}`),
		"gpt-5.5",
	)

	require.False(t, shouldDisable)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Zero(t, repo.tempCalls)
	require.Empty(t, repo.modelRateLimitCalls)
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
		svc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitServiceCustomTest("true")}

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
		svc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitServiceCustomTest("false")}

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
		svc := &OpenAIGatewayService{rateLimitService: newOpenAIAdvancedSchedulerRateLimitServiceCustomTest("false")}

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
		accountRepo:                      repo,
		rateLimitService:                 rateLimitService,
		openAIGatewayServiceCustomFields: openAIGatewayServiceCustomFields{openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour)},
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
		"gpt-5.5",
	)

	require.False(t, shouldDisable)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.False(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-5.5"))

	shouldDisable = svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		524,
		http.Header{},
		[]byte(`{"error":{"message":"Cloudflare timeout"}}`),
		"gpt-5.5",
	)

	require.False(t, shouldDisable)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.True(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-5.5"))
	require.Zero(t, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
}

func TestOpenAITransient5xxMarksTempUnschedulable(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc := &OpenAIGatewayService{
		accountRepo:      repo,
		rateLimitService: rateLimitService,
	}
	account := &Account{
		ID:          108,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"bad gateway"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.tempCalls)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAITransient504MarksTempUnschedulable(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc := &OpenAIGatewayService{
		accountRepo:      repo,
		rateLimitService: rateLimitService,
	}
	account := &Account{
		ID:          109,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
	}

	shouldDisable := svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		http.StatusGatewayTimeout,
		http.Header{},
		[]byte(`{"error":{"message":"gateway timeout"}}`),
		"gpt-5.5",
	)

	require.False(t, shouldDisable)
	require.False(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-5.5"))

	shouldDisable = svc.handleOpenAIAccountUpstreamError(
		context.Background(),
		account,
		http.StatusGatewayTimeout,
		http.Header{},
		[]byte(`{"error":{"message":"gateway timeout"}}`),
		"gpt-5.5",
	)

	require.False(t, shouldDisable)
	require.Equal(t, 0, repo.tempCalls)
	require.True(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-5.5"))
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIRuntimeBlockDoesNotPersistTempUnschedulable(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo: repo,
	}
	account := &Account{
		ID:          106,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
	}

	svc.BlockAccountScheduling(account, time.Now().Add(time.Minute), "openai_transient_5xx")

	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 0, repo.tempCalls)
}

func TestOpenAIRequestErrorFailoversAndCoolsAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                      repo,
		openAIGatewayServiceCustomFields: openAIGatewayServiceCustomFields{openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour)},
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

func TestOpenAIRequestErrorFailoversOnProxyAuthenticationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                      repo,
		openAIGatewayServiceCustomFields: openAIGatewayServiceCustomFields{openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour)},
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
	require.Equal(t, 1, repo.tempCalls)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIRequestErrorDoesNotFailoverWhenRequestCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                      repo,
		openAIGatewayServiceCustomFields: openAIGatewayServiceCustomFields{openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour)},
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
		accountRepo:                      repo,
		openAIGatewayServiceCustomFields: openAIGatewayServiceCustomFields{openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour)},
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

func TestOpenAIRequestErrorClosesIdleConnectionsAndShortCooldowns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	upstream := &closeIdleHTTPUpstreamStub{}
	svc := &OpenAIGatewayService{
		accountRepo:                      repo,
		httpUpstream:                     upstream,
		openAIGatewayServiceCustomFields: openAIGatewayServiceCustomFields{openaiTransientCooldownThrottle: newAccountWriteThrottle(time.Hour)},
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
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, account.ID, repo.lastTempID)
}

func TestOpenAIStreamDiagnosticErrorsFailoverWithShortCooldown(t *testing.T) {
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
			beforeTempCalls := repo.tempCalls
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

			failoverErr := svc.handleOpenAIUpstreamStreamError(context.Background(), c, account, tc.err, "", false)

			require.NotNil(t, failoverErr)
			require.Equal(t, 0, failoverErr.StatusCode)
			require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
			require.Equal(t, beforeTempCalls+1, repo.tempCalls)
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

func TestOpenAIStreamFailoverDiagnosticMessageShortCooldownsAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 112, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	failoverErr := svc.newOpenAIStreamFailoverError(c, account, false, "", nil, "OpenAI stream ended before a terminal event")

	require.NotNil(t, failoverErr)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, account.ID, repo.lastTempID)

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.True(t, events[0].CooldownApplied)
	require.Equal(t, "openai_stream_error", events[0].CooldownReason)
}

func TestOpenAIStreamRateLimitPreservesSemanticStatusWithoutAccountCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 114, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	headers := http.Header{"Retry-After": []string{"12"}}
	payload := []byte(`{"type":"response.failed","response":{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"limited"}}}`)

	failoverErr := svc.newOpenAIStreamFailoverError(c, account, false, "", payload, "limited", headers)

	require.NotNil(t, failoverErr)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.Equal(t, "12", failoverErr.ResponseHeaders.Get("Retry-After"))
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.Zero(t, repo.tempCalls)

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, http.StatusTooManyRequests, events[0].UpstreamStatusCode)
	require.False(t, events[0].CooldownApplied)
	require.Empty(t, events[0].CooldownReason)
}

func TestOpenAIRuntimeBlock_DoesNotPersistStatusZero(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 113, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}

	svc.BlockAccountScheduling(account, time.Now().Add(time.Minute), "openai_stream_error")

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
	require.True(t, legacySvc.shouldFailoverOpenAIUpstreamResponseForAccount(context.Background(), account, http.StatusNotFound, "model not found", body))

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

func TestOpenAIRuntimeCooldownsFromConfigRejectsOverflow(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			OpenAIScheduler: config.GatewayOpenAISchedulerConfig{
				GatewayOpenAISchedulerConfigCustom: config.GatewayOpenAISchedulerConfigCustom{
					RuntimeCooldowns: config.GatewayOpenAIRuntimeCooldownsConfig{
						RequestErrorCooldownSeconds: int(time.Duration(1<<63-1)/time.Second) + 1,
					},
				},
			},
		},
	}

	cooldowns := openAIRuntimeCooldownsFromConfig(cfg)

	require.Equal(t, openAIRequestErrorCooldown, cooldowns.requestError)
}

func TestOpenAI403ScheduleFailureOverrideIgnoresBusinessForbidden(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 118, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"code":"GROUP_DISABLED","message":"API Key 所属分组已停用"}`)

	category := svc.schedulerCategoryOverrideForOpenAIUpstreamError(context.Background(), account, http.StatusForbidden, body)

	require.Empty(t, category)
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
