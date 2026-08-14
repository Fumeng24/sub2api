package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type openAISingleCandidateRetryRepo struct {
	AccountRepository
	account   *Account
	tempCalls int
}

func (r *openAISingleCandidateRetryRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		return r.account, nil
	}
	return nil, ErrAccountNotFound
}

func (r *openAISingleCandidateRetryRepo) SetTempUnschedulable(_ context.Context, _ int64, _ time.Time, _ string) error {
	r.tempCalls++
	return nil
}

func TestOpenAITransient5xxSingleCandidateStillBlocksScheduling(t *testing.T) {
	account := &Account{
		ID:          110,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	repo := &openAISingleCandidateRetryRepo{account: account}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc := &OpenAIGatewayService{
		accountRepo:      repo,
		rateLimitService: rateLimitService,
	}
	ctx := WithSingleAccountRetry(context.Background(), true, false)

	shouldDisable := svc.handleOpenAIAccountUpstreamError(
		ctx,
		account,
		http.StatusBadGateway,
		http.Header{},
		[]byte(`{"error":{"message":"bad gateway"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.tempCalls)
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}
