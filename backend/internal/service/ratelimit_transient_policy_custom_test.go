//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestRateLimitService_PoolTransient5xxRequiresMoreThanThreeFailures(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &transientErrorCounterCacheStub{counts: []int64{1, 2, 3, 4}}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetTransientErrorCounterCache(counter)
	account := &Account{
		ID:          38805,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"pool_mode": true},
	}
	body := []byte(`{"error":{"message":"timeout awaiting response headers"}}`)

	for i := 0; i < 3; i++ {
		disabled := service.HandleUpstreamError(context.Background(), account, http.StatusBadGateway, http.Header{}, body)
		require.False(t, disabled)
	}
	require.Zero(t, repo.tempCalls)

	disabled := service.HandleUpstreamError(context.Background(), account, http.StatusBadGateway, http.Header{}, body)

	require.True(t, disabled)
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, []int64{account.ID}, counter.resetCalls)
	require.True(t, repo.lastTempUntil.After(time.Now()))
}

func TestRateLimitService_PoolTransient5xxCoolsLastGroupAccount(t *testing.T) {
	accountID := int64(38873)
	groupID := int64(11)
	repo := &rateLimitAccountRepoStub{
		rateLimitAccountRepoStubCustom: rateLimitAccountRepoStubCustom{
			schedulableByGroup: map[int64][]Account{groupID: {{ID: accountID}}},
		},
	}
	counter := &transientErrorCounterCacheStub{counts: []int64{4}}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetTransientErrorCounterCache(counter)
	account := &Account{
		ID:          accountID,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		GroupIDs:    []int64{groupID},
		Credentials: map[string]any{"pool_mode": true},
	}

	disabled := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"cloudflare bad gateway"}}`),
	)

	require.True(t, disabled)
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, []int64{account.ID}, counter.resetCalls)
}

func TestRateLimitService_PoolTransient5xxCountsLogicalRequestOnce(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &transientErrorCounterCacheStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetTransientErrorCounterCache(counter)
	account := &Account{
		ID:          38866,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"pool_mode": true},
	}
	body := []byte(`{"error":{"message":"temporary upstream failure"}}`)

	first := context.WithValue(context.Background(), ctxkey.RequestID, "logical-request-1")
	for i := 0; i < 3; i++ {
		require.False(t, service.HandleUpstreamError(first, account, http.StatusBadGateway, http.Header{}, body))
	}
	require.Equal(t, int64(1), counter.lastCount)
	require.Zero(t, repo.tempCalls)

	for _, requestID := range []string{"logical-request-2", "logical-request-3"} {
		ctx := context.WithValue(context.Background(), ctxkey.RequestID, requestID)
		require.False(t, service.HandleUpstreamError(ctx, account, http.StatusBadGateway, http.Header{}, body))
	}
	last := context.WithValue(context.Background(), ctxkey.RequestID, "logical-request-4")
	require.True(t, service.HandleUpstreamError(last, account, http.StatusBadGateway, http.Header{}, body))
	require.Equal(t, int64(4), counter.lastCount)
	require.Equal(t, 1, repo.tempCalls)
	require.Len(t, counter.onceCalls, 7)
}

func TestRateLimitService_HandleUpstreamError_Transient5xxTempUnschedules(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{ID: 203, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"bad gateway","type":"server_error"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
}

func TestRateLimitService_HandleUpstreamError_Transient504TempUnschedules(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{ID: 204, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusGatewayTimeout,
		http.Header{},
		[]byte(`{"error":{"message":"gateway timeout","type":"server_error"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
}

func TestRateLimitService_CheckErrorPolicy_Transient502TempUnschedulesEvenWithCustomErrorCodes(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	account := &Account{
		ID:       205,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(599)},
			"temp_unschedulable_enabled": true,
			"temp_unschedulable_rules": []any{
				map[string]any{
					"error_code":       float64(http.StatusBadGateway),
					"keywords":         []any{"bad gateway"},
					"duration_minutes": float64(10),
				},
			},
		},
	}

	result := service.CheckErrorPolicy(
		context.Background(),
		account,
		http.StatusBadGateway,
		[]byte(`{"error":{"message":"bad gateway","type":"server_error"}}`),
	)

	require.Equal(t, ErrorPolicyTempUnscheduled, result)
	require.Equal(t, 1, repo.tempCalls)
}
