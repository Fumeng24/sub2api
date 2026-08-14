//go:build unit

package service

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type corePolicyAnthropicRepo struct {
	anthropicWindowLimitRepo
	setErrorCalls int
}

func (r *corePolicyAnthropicRepo) SetError(_ context.Context, _ int64, _ string) error {
	r.setErrorCalls++
	return nil
}

func TestHandleUpstreamError_AnthropicWindowLimitPreemptsBillingClassification(t *testing.T) {
	resetAt := time.Now().Add(3 * time.Hour).Truncate(time.Second)
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "1.02")
	headers.Set("anthropic-ratelimit-unified-5h-reset", strconv.FormatInt(resetAt.Unix(), 10))
	repo := &corePolicyAnthropicRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 43, Type: AccountTypeOAuth, Platform: PlatformAnthropic}

	disabled := svc.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		headers,
		[]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"insufficient balance"}}`),
	)

	require.False(t, disabled)
	require.Equal(t, 1, repo.rateLimitCalls)
	require.Zero(t, repo.setErrorCalls)
}
