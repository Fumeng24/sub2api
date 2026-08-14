//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayDiagnoseModelAvailability_UnschedulableConfiguredAccountStillSupportsModel(t *testing.T) {
	repo := stubOpenAIAccountRepo{accounts: []Account{
		{
			ID:          1,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: false,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.5": "gpt-5.5"},
			},
		},
	}}
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{}}

	diag := svc.DiagnoseModelAvailabilityForPlatform(context.Background(), ptrInt64(2), "gpt-5.5", PlatformOpenAI)

	require.True(t, diag.HasAccountsInPool)
	require.True(t, diag.HasModelSupport)
}
