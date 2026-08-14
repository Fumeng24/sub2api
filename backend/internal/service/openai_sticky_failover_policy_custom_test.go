package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenAIStickyFailoverTransientFailureMigratesImmediately(t *testing.T) {
	svc := &OpenAIGatewayService{openaiModelTransient: newOpenAIAccountModelTransientState(128)}
	account := &Account{ID: 7101, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	failure := &UpstreamFailoverError{StatusCode: http.StatusBadGateway}

	ctx := withOpenAIStickyOriginalAccountID(context.Background(), account.ID)
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, account, "gpt-5.6-sol", failure))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, account, "gpt-5.6-sol", failure))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, account, "gpt-5.6-sol", failure))
	require.Equal(t, 3, svc.openaiModelTransient.activeFailureStreak(account.ID, "gpt-5.6-sol", time.Now()))
}

func TestOpenAIStickyFailoverHardFailuresMigrateImmediately(t *testing.T) {
	tests := []struct {
		name    string
		failure *UpstreamFailoverError
	}{
		{name: "invalid auth", failure: &UpstreamFailoverError{StatusCode: http.StatusUnauthorized, ResponseBody: []byte(`{"error":{"code":"invalid_api_key","message":"Invalid API key"}}`)}},
		{name: "insufficient balance", failure: &UpstreamFailoverError{StatusCode: http.StatusPaymentRequired, ResponseBody: []byte(`{"error":{"message":"insufficient balance"}}`)}},
		{name: "unsupported model", failure: &UpstreamFailoverError{StatusCode: http.StatusNotFound, ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"model not found"}}`)}},
		{name: "persistent transport", failure: &UpstreamFailoverError{StatusCode: http.StatusBadGateway, SchedulerCategory: "error"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &OpenAIGatewayService{openaiModelTransient: newOpenAIAccountModelTransientState(128)}
			account := &Account{ID: 7201, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

			require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(context.Background(), account, "gpt-5.6-sol", tt.failure))
			require.Zero(t, svc.openaiModelTransient.size())
		})
	}
}

func TestOpenAIStickyProviderScopedCredentialFailureKeepsOriginalWithoutPenalizingAccount(t *testing.T) {
	svc := &OpenAIGatewayService{openaiModelTransient: newOpenAIAccountModelTransientState(128)}
	original := &Account{ID: 7251, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	fallback := &Account{ID: 7252, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	ctx := withOpenAIStickyOriginalAccountID(context.Background(), original.ID)
	failure := &UpstreamFailoverError{
		StatusCode: http.StatusUnauthorized,
		Stage:      GatewayFailureStageAccountAuth,
		Scope:      GatewayFailureScopeProvider,
	}

	require.True(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, original, "gpt-5.6-sol", failure))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, fallback, "gpt-5.6-sol", failure))
	require.Zero(t, svc.openaiModelTransient.size())
}

func TestOpenAIStickyFailoverStatusZeroRequiresTransientCategory(t *testing.T) {
	svc := &OpenAIGatewayService{openaiModelTransient: newOpenAIAccountModelTransientState(128)}
	account := &Account{ID: 7301, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	ctx := withOpenAIStickyOriginalAccountID(context.Background(), account.ID)
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, account, "gpt-5.6-sol", &UpstreamFailoverError{
		StatusCode: 0, SchedulerCategory: "transient_timeout",
	}))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, account, "gpt-5.6-terra", &UpstreamFailoverError{
		StatusCode: 0, SchedulerCategory: "error",
	}))
}

func TestOpenAIStickyRequestScopedTransientDoesNotPenalizeAccount(t *testing.T) {
	svc := &OpenAIGatewayService{openaiModelTransient: newOpenAIAccountModelTransientState(128)}
	account := &Account{ID: 7351, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(
		context.Background(), account, "gpt-5.6-sol",
		&UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, RequestScopedTransient: true},
	))
	require.Zero(t, svc.openaiModelTransient.size())
}

func TestOpenAIStickyFailoverTrackingDefersModelCooldownToHandler(t *testing.T) {
	for _, accountType := range []string{AccountTypeAPIKey, AccountTypeOAuth} {
		t.Run(accountType, func(t *testing.T) {
			svc := &OpenAIGatewayService{openaiModelTransient: newOpenAIAccountModelTransientState(128)}
			svc.rateLimitService = NewRateLimitService(transientCooldownAccountRepo{}, nil, &config.Config{}, nil, nil)
			account := &Account{ID: 7401, Platform: PlatformOpenAI, Type: accountType}
			ctx := WithOpenAIStickyFailoverTracking(context.Background())

			require.False(t, svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusServiceUnavailable, http.Header{}, []byte(`{"error":{"message":"temporarily unavailable"}}`), "gpt-5.6-sol"))
			require.Zero(t, svc.openaiModelTransient.size(), "forward path must not count a pool retry as a new logical failure")
			preserveCtx := withOpenAIStickyOriginalAccountID(ctx, account.ID)
			require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(preserveCtx, account, "gpt-5.6-sol", &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable}))
			require.True(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-5.6-sol"))
			require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
		})
	}
}

func TestOpenAIPreserveStickyBindingMakesAllBindingMutationsNoOp(t *testing.T) {
	groupID := int64(10)
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session-a": 7501}}
	svc := &OpenAIGatewayService{cache: cache}
	ctx := WithOpenAIPreserveStickyBinding(context.Background())

	require.NoError(t, svc.setStickySessionAccountID(ctx, &groupID, "session-a", 7502, time.Hour))
	require.NoError(t, svc.deleteStickySessionAccountID(ctx, &groupID, "session-a"))
	require.NoError(t, svc.bindOpenAIStickySessionDuringSelection(ctx, &groupID, "session-a", 7502))
	require.NoError(t, svc.BindStickySessionAfterProfitAdmission(ctx, &groupID, "session-a", 7502))
	require.Equal(t, int64(7501), cache.sessionBindings["openai:session-a"])
	require.Empty(t, cache.deletedSessions)
}

func TestOpenAIStickyActiveCooldownNeverPreservesStaleBinding(t *testing.T) {
	groupID := int64(10)
	account := Account{
		ID: 7601, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, GroupIDs: []int64{groupID},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session-a": account.ID}}
	svc := &OpenAIGatewayService{
		accountRepo:          schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:                cache,
		openaiModelTransient: newOpenAIAccountModelTransientState(128),
	}
	failure := &UpstreamFailoverError{StatusCode: http.StatusBadGateway}

	ctx := svc.PrepareOpenAIStickyFailoverContext(context.Background(), &groupID, "session-a", "gpt-5.6-sol")
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, &account, "gpt-5.6-sol", failure))
	require.False(t, openAIPreserveStickyBinding(ctx), "a transient failure must not keep the old sticky binding")
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingForActiveCooldown(context.Background(), &groupID, "session-a", "gpt-5.6-sol"))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, &account, "gpt-5.6-sol", failure))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(ctx, &account, "gpt-5.6-sol", failure))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingForActiveCooldown(context.Background(), &groupID, "session-a", "gpt-5.6-sol"))
}

func TestOpenAIStickyTransientFailoverMigratesOnFirstFailure(t *testing.T) {
	groupID := int64(10)
	model := "gpt-5.6-sol"
	original := Account{ID: 7701, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}}
	fallback := Account{ID: 7702, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session-a": original.ID}}
	svc := newOpenAIStickyFailoverSchedulerTestService([]Account{original, fallback}, cache)
	failure := &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable}

	requestCtx := svc.PrepareOpenAIStickyFailoverContext(context.Background(), &groupID, "session-a", model)
	require.Equal(t, original.ID, openAIStickyOriginalAccountID(requestCtx))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(requestCtx, &original, model, failure))

	selection, _, err := svc.SelectAccountWithScheduler(requestCtx, &groupID, "", "session-a", model, map[int64]struct{}{original.ID: {}}, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, fallback.ID, selection.Account.ID)
	require.Equal(t, fallback.ID, cache.sessionBindings["openai:session-a"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}

	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(requestCtx, &original, model, failure))
	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(requestCtx, &original, model, failure))
	migrationCtx := svc.PrepareOpenAIStickyFailoverContext(context.Background(), &groupID, "session-a", model)
	require.False(t, openAIPreserveStickyBinding(migrationCtx))

	selection, _, err = svc.SelectAccountWithScheduler(migrationCtx, &groupID, "", "session-a", model, map[int64]struct{}{original.ID: {}}, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, fallback.ID, selection.Account.ID)
	require.Equal(t, fallback.ID, cache.sessionBindings["openai:session-a"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIStickyTransientFallbackDoesNotCaptureBindingAfterHardOriginalFailure(t *testing.T) {
	groupID := int64(10)
	model := "gpt-5.6-sol"
	original := Account{ID: 7801, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}}
	firstFallback := Account{ID: 7802, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}}
	secondFallback := Account{ID: 7803, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session-a": original.ID}}
	svc := newOpenAIStickyFailoverSchedulerTestService([]Account{original, firstFallback, secondFallback}, cache)
	requestCtx := svc.PrepareOpenAIStickyFailoverContext(context.Background(), &groupID, "session-a", model)

	selection, _, err := svc.SelectAccountWithScheduler(requestCtx, &groupID, "", "session-a", model, map[int64]struct{}{original.ID: {}, secondFallback.ID: {}}, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, firstFallback.ID, selection.Account.ID)
	require.Equal(t, firstFallback.ID, cache.sessionBindings["openai:session-a"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}

	require.False(t, svc.ShouldPreserveOpenAIStickyBindingAfterFailure(requestCtx, &firstFallback, model, &UpstreamFailoverError{StatusCode: http.StatusBadGateway}))
	selection, _, err = svc.SelectAccountWithScheduler(requestCtx, &groupID, "", "session-a", model, map[int64]struct{}{original.ID: {}, firstFallback.ID: {}}, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, secondFallback.ID, selection.Account.ID)
	require.Equal(t, secondFallback.ID, cache.sessionBindings["openai:session-a"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIStickyModelCooldownSkipsPreviousResponseWithoutDeletingChain(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10)
	model := "gpt-5.6-sol"
	account := Account{
		ID: 7901, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 1,
		Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true},
	}
	cache := &stubGatewayCache{}
	store := NewOpenAIWSStateStore(cache)
	svc := &OpenAIGatewayService{
		accountRepo:          stubOpenAIAccountRepo{accounts: []Account{account}},
		cache:                cache,
		cfg:                  newOpenAIWSV2TestConfig(),
		concurrencyService:   NewConcurrencyService(stubConcurrencyCache{}),
		openaiWSStateStore:   store,
		openaiModelTransient: newOpenAIAccountModelTransientState(128),
	}
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp-short-cooldown", account.ID, time.Hour))
	svc.recordOpenAIAccountModelTransientFailure(&account, model, time.Now())

	selection, err := svc.SelectAccountByPreviousResponseID(ctx, &groupID, "resp-short-cooldown", model, nil, false)
	require.NoError(t, err)
	require.Nil(t, selection)
	boundAccountID, err := store.GetResponseAccount(ctx, groupID, "resp-short-cooldown")
	require.NoError(t, err)
	require.Equal(t, account.ID, boundAccountID)

	svc.clearOpenAIAccountModelTransientState(account.ID, model)
	selection, err = svc.SelectAccountByPreviousResponseID(ctx, &groupID, "resp-short-cooldown", model, nil, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.Equal(t, account.ID, selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func newOpenAIStickyFailoverSchedulerTestService(accounts []Account, cache *schedulerTestGatewayCache) *OpenAIGatewayService {
	return &OpenAIGatewayService{
		accountRepo:          schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:                cache,
		cfg:                  &config.Config{},
		rateLimitService:     newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService:   NewConcurrencyService(schedulerTestConcurrencyCache{}),
		openaiModelTransient: newOpenAIAccountModelTransientState(128),
	}
}
