//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGatewayService_UpstreamTransient5xxCoolsFinalGroupCandidate(t *testing.T) {
	account := &Account{
		ID:          38873,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		GroupIDs:    []int64{11},
	}
	repo := &rateLimitAccountRepoStub{}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	gatewaySvc := &GatewayService{accountRepo: repo, rateLimitService: rateLimitSvc}

	disabled := gatewaySvc.handleUpstreamErrorForScheduling(
		context.Background(),
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"cloudflare bad gateway"}}`),
		"claude-opus-4-8",
	)

	require.True(t, disabled)
	require.Equal(t, 1, repo.tempCalls)
	require.Len(t, repo.outboxEvents, 1)
	require.Equal(t, "account", repo.outboxEvents[0].payload["block_granularity"])
}

func TestGatewayService_PoolTransient5xxCoolsSharedAccountOnceThresholdReached(t *testing.T) {
	account := &Account{
		ID:          38839,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		GroupIDs:    []int64{10, 31},
		Credentials: map[string]any{"pool_mode": true},
	}
	repo := &rateLimitAccountRepoStub{}
	counter := &transientErrorCounterCacheStub{counts: []int64{4}}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitSvc.SetTransientErrorCounterCache(counter)
	gatewaySvc := &GatewayService{accountRepo: repo, rateLimitService: rateLimitSvc}

	disabled := gatewaySvc.handleUpstreamErrorForScheduling(
		context.Background(),
		account,
		http.StatusServiceUnavailable,
		http.Header{},
		[]byte(`{"error":{"message":"Service temporarily unavailable"}}`),
		"gpt-5.5",
	)

	require.True(t, disabled)
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, []int64{account.ID}, counter.resetCalls)
	require.Len(t, repo.outboxEvents, 1)
	require.Equal(t, "account", repo.outboxEvents[0].payload["block_granularity"])
}

func TestGatewayService_TempUnscheduleRetryableErrorCoolsFinalCandidate(t *testing.T) {
	accountID := int64(38839)
	repo := &rateLimitAccountRepoStub{}
	gatewaySvc := &GatewayService{accountRepo: repo}

	gatewaySvc.TempUnscheduleRetryableError(context.Background(), accountID, &UpstreamFailoverError{
		StatusCode:             http.StatusBadGateway,
		RetryableOnSameAccount: true,
	})

	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, accountID, repo.lastTempID)
}
