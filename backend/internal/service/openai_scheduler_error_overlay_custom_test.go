package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func (r stubOpenAIAccountRepo) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

func (r stubOpenAIAccountRepo) ListByGroup(_ context.Context, _ int64) ([]Account, error) {
	return append([]Account(nil), r.accounts...), nil
}

func TestOpenAISelectAccountWithLoadAwareness_LegacyAliasDoesNotMatchDifferentModel(t *testing.T) {
	groupID := int64(1)
	available := Account{
		ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
		Schedulable: true, Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.5": "gpt-5.5"}},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{available}},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}

	selection, err := svc.SelectAccountWithLoadAwareness(context.Background(), &groupID, "", "gpt-5.2", nil)
	require.Error(t, err)
	require.Nil(t, selection)
}

func TestOpenAIErrorResponseForClass_ModelUnsupportedReturnsModelGroupUnavailable(t *testing.T) {
	status, errType, msg := openAIErrorResponseForClass(http.StatusNotFound, openAIUpstreamErrorModelUnsupported, "model not found", false)
	require.Equal(t, http.StatusServiceUnavailable, status)
	require.Equal(t, "api_error", errType)
	require.Equal(t, clientFacingModelGroupUnavailableMessage, msg)
}
