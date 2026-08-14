//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type siteRateLimitClearRepoStub struct {
	rateLimitClearRepoStub
	setSchedulableCalls []bool
}

func (r *siteRateLimitClearRepoStub) SetSchedulable(_ context.Context, _ int64, schedulable bool) error {
	r.setSchedulableCalls = append(r.setSchedulableCalls, schedulable)
	return nil
}

func TestRateLimitService_RecoverAccountAfterSuccessfulTest_SiteRestoresAutoDisabledScheduling(t *testing.T) {
	now := time.Now()
	repo := &siteRateLimitClearRepoStub{rateLimitClearRepoStub: rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:                     42,
			Status:                 StatusError,
			Schedulable:            false,
			RateLimitedAt:          &now,
			TempUnschedulableUntil: &now,
			Extra: map[string]any{
				"model_rate_limits":        map[string]any{"gpt-5.5": map[string]any{"rate_limit_reset_at": now.Format(time.RFC3339)}},
				"antigravity_quota_scopes": map[string]any{"gemini": true},
			},
		},
	}}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, &tempUnschedCacheRecorder{})

	result, err := svc.RecoverAccountAfterSuccessfulTest(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, result.ClearedError)
	require.True(t, result.ClearedRateLimit)
	require.Equal(t, []bool{true}, repo.setSchedulableCalls)
}

func TestRateLimitService_RecoverAccountAfterSuccessfulTest_SiteKeepsManualDisableWithRuntimeState(t *testing.T) {
	now := time.Now()
	repo := &siteRateLimitClearRepoStub{rateLimitClearRepoStub: rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:                     8,
			Status:                 StatusActive,
			Schedulable:            false,
			TempUnschedulableUntil: &now,
		},
	}}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, &tempUnschedCacheRecorder{})

	result, err := svc.RecoverAccountAfterSuccessfulTest(context.Background(), 8)
	require.NoError(t, err)
	require.True(t, result.ClearedRateLimit)
	require.Empty(t, repo.setSchedulableCalls)
}

func TestRateLimitService_RecoverAccountAfterSuccessfulTest_SiteKeepsManualDisableWithoutRuntimeState(t *testing.T) {
	repo := &siteRateLimitClearRepoStub{rateLimitClearRepoStub: rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:          10,
			Status:      StatusActive,
			Schedulable: false,
			Extra:       map[string]any{},
		},
	}}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, &tempUnschedCacheRecorder{})

	result, err := svc.RecoverAccountAfterSuccessfulTest(context.Background(), 10)
	require.NoError(t, err)
	require.False(t, result.ClearedRateLimit)
	require.Empty(t, repo.setSchedulableCalls)
}
