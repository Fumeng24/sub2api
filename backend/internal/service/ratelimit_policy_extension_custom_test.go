package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type handledRateLimitPolicyStub struct{}

func (handledRateLimitPolicyStub) HandleBeforeDefault(context.Context, *Account, int, http.Header, []byte, ...string) (bool, bool) {
	return true, true
}

func TestCustomRateLimitPolicyDelegatesByDefault(t *testing.T) {
	disable, handled := newCustomRateLimitPolicy(nil).HandleBeforeDefault(context.Background(), &Account{}, http.StatusBadGateway, nil, nil)
	require.False(t, disable)
	require.False(t, handled)
}

func TestCustomRateLimitPolicyHandlesLocalSkipRules(t *testing.T) {
	policy := newCustomRateLimitPolicy(nil)

	disable, handled := policy.HandleBeforeDefault(context.Background(), &Account{Type: AccountTypeAPIKey, Credentials: map[string]any{"pool_mode": true}}, http.StatusTeapot, nil, nil)
	require.False(t, disable)
	require.True(t, handled)

	disable, handled = policy.HandleBeforeDefault(context.Background(), &Account{Type: AccountTypeAPIKey, Credentials: map[string]any{"custom_error_codes_enabled": true, "custom_error_codes": []any{float64(http.StatusBadGateway)}}}, http.StatusTeapot, nil, nil)
	require.False(t, disable)
	require.True(t, handled)
}

func TestRateLimitPolicyExtensionCanShortCircuitDefaultFlow(t *testing.T) {
	svc := &RateLimitService{policyExtension: handledRateLimitPolicyStub{}}
	disable := svc.HandleUpstreamError(context.Background(), &Account{}, http.StatusTeapot, nil, nil)
	require.True(t, disable)
}

func TestCustomRateLimitPolicyOpenAIKnownModelTransientDelegatesToGatewayCircuit(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{ID: 101, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	disable := svc.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"temporary upstream failure"}}`),
		"gpt-5.5",
	)

	require.False(t, disable)
	require.Zero(t, repo.tempCalls)
}

func TestCustomRateLimitPolicyOpenAIUnknownModelTransientUsesAccountFallback(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{ID: 102, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	disable := svc.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"temporary upstream failure"}}`),
	)

	require.True(t, disable)
	require.Equal(t, 1, repo.tempCalls)
}

func TestCustomRateLimitPolicyOpenAIPoolTransientKeepsThreshold(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &transientErrorCounterCacheStub{counts: []int64{1, 2, 3, 4}}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetTransientErrorCounterCache(counter)
	account := &Account{
		ID:       103,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode": true,
		},
	}
	body := []byte(`{"error":{"message":"temporary upstream failure"}}`)

	for range 3 {
		require.False(t, svc.HandleUpstreamError(context.Background(), account, http.StatusBadGateway, http.Header{}, body, "gpt-5.5"))
	}
	require.True(t, svc.HandleUpstreamError(context.Background(), account, http.StatusBadGateway, http.Header{}, body, "gpt-5.5"))
	require.Equal(t, 1, repo.tempCalls)
}

func TestCustomRateLimitPolicyModelUnavailablePreemptsTransientFallback(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       104,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode": true,
		},
	}

	disable := svc.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusServiceUnavailable,
		http.Header{},
		[]byte(`{"error":{"message":"No available channel for model gpt-5.5"}}`),
		"gpt-5.5",
	)

	require.True(t, disable)
	require.Equal(t, 1, repo.modelRateLimitCalls)
	require.Equal(t, "gpt-5.5", repo.lastModelRateLimitKey)
	require.Zero(t, repo.tempCalls)
}
