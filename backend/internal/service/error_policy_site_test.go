//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func siteBillingExhaustedGeminiAccount(id int64) *Account {
	return &Account{
		ID:       id,
		Type:     AccountTypeAPIKey,
		Platform: PlatformGemini,
		Credentials: map[string]any{
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(http.StatusTooManyRequests)},
		},
	}
}

func TestCheckErrorPolicy_SiteBillingExhaustionBypassesCustomCodeMiss(t *testing.T) {
	repo := &errorPolicyRepoStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)

	result := svc.CheckErrorPolicy(
		context.Background(),
		siteBillingExhaustedGeminiAccount(16),
		http.StatusPaymentRequired,
		[]byte(`{"error":{"message":"insufficient balance"}}`),
	)

	require.Equal(t, ErrorPolicyMatched, result)
}

func TestHandleUpstreamError_SiteBillingExhaustionBypassesCustomCodes(t *testing.T) {
	repo := &errorPolicyRepoStub{}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)

	shouldDisable := svc.HandleUpstreamError(
		context.Background(),
		siteBillingExhaustedGeminiAccount(32),
		http.StatusPaymentRequired,
		http.Header{},
		[]byte(`{"error":{"message":"insufficient balance"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrCalls)
	require.Contains(t, repo.lastErrorMsg, "Upstream billing exhausted")
	require.Zero(t, repo.tempCalls)
}
