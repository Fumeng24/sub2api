//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type openAIImageQuotaAccountRepoStub struct {
	modelNotFoundAccountRepoStub
	setErrorCalls  int
	rateLimitCalls int
}

func (r *openAIImageQuotaAccountRepoStub) SetError(_ context.Context, _ int64, _ string) error {
	r.setErrorCalls++
	return nil
}

func (r *openAIImageQuotaAccountRepoStub) SetRateLimited(_ context.Context, _ int64, _ time.Time) error {
	r.rateLimitCalls++
	return nil
}

func TestIsOpenAIImageRateLimitError_SiteImageQuotaError(t *testing.T) {
	imageQuotaBody := []byte(`{"error":{"code":"insufficient_quota","message":"no available image quota","type":"insufficient_quota"}}`)
	require.True(t, isOpenAIImageRateLimitError(http.StatusTooManyRequests, imageQuotaBody))
}

func TestRateLimitService_HandleUpstreamError_OpenAIImageQuota429UsesImageRateLimit(t *testing.T) {
	repo := &openAIImageQuotaAccountRepoStub{}
	svc := &RateLimitService{accountRepo: repo}
	account := &Account{ID: 205, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"code":"insufficient_quota","message":"no available image quota","type":"insufficient_quota"}}`)

	disabled := svc.HandleUpstreamError(context.Background(), account, http.StatusTooManyRequests, http.Header{}, body, "gpt-image-2")

	require.False(t, disabled)
	require.Zero(t, repo.setErrorCalls)
	require.Zero(t, repo.rateLimitCalls)
	require.Len(t, repo.modelRateLimitCalls, 1)
	require.Equal(t, openAIImageGenerationRateLimitKey, repo.modelRateLimitCalls[0].scope)
	require.Equal(t, openAIImageRateLimitReason, repo.modelRateLimitCalls[0].reason)
}

func TestOpenAIGatewayService_HandleOpenAIAccountUpstreamError_ImageQuotaDoesNotDisableAccount(t *testing.T) {
	repo := &openAIImageQuotaAccountRepoStub{}
	svc := &OpenAIGatewayService{rateLimitService: &RateLimitService{accountRepo: repo}}
	account := &Account{ID: 206, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"code":"insufficient_quota","message":"no available image quota","type":"insufficient_quota"}}`)

	disabled := svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusTooManyRequests, http.Header{}, body, "gpt-image-2")

	require.False(t, disabled)
	require.Zero(t, repo.setErrorCalls)
	require.Zero(t, repo.rateLimitCalls)
	require.Len(t, repo.modelRateLimitCalls, 1)
	require.Equal(t, openAIImageGenerationRateLimitKey, repo.modelRateLimitCalls[0].scope)
	_, wholeAccountBlocked := svc.openaiAccountRuntimeBlockUntil.Load(account.ID)
	require.False(t, wholeAccountBlocked)
}
