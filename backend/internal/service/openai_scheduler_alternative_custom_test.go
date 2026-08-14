package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHasEligibleOpenAIAccountAlternativeHonorsExclusionsAndConstraints(t *testing.T) {
	groupID := int64(82001)
	accounts := []Account{
		{ID: 82001, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}},
		{ID: 82002, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}, Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.5": "gpt-5.5"}}},
	}
	svc := &OpenAIGatewayService{
		accountRepo: schedulerGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}},
		cfg:         &config.Config{},
	}
	req := OpenAIAlternativeAccountRequest{
		GroupID:            &groupID,
		Platform:           PlatformOpenAI,
		RequestedModel:     "gpt-5.5",
		RequiredTransport:  OpenAIUpstreamTransportAny,
		RequiredCapability: OpenAIEndpointCapabilityChatCompletions,
		CurrentAccountID:   accounts[0].ID,
	}

	hasAlternative, err := svc.HasEligibleOpenAIAccountAlternative(context.Background(), req)
	require.NoError(t, err)
	require.True(t, hasAlternative)

	req.ExcludedIDs = map[int64]struct{}{accounts[1].ID: {}}
	hasAlternative, err = svc.HasEligibleOpenAIAccountAlternative(context.Background(), req)
	require.NoError(t, err)
	require.False(t, hasAlternative)

	req.ExcludedIDs = nil
	accounts[1].Schedulable = false
	svc.accountRepo = schedulerGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}}
	hasAlternative, err = svc.HasEligibleOpenAIAccountAlternative(context.Background(), req)
	require.NoError(t, err)
	require.False(t, hasAlternative)

	accounts[1].Schedulable = true
	accounts[1].Credentials = map[string]any{"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"}}
	svc.accountRepo = schedulerGroupAwareOpenAIAccountRepo{schedulerTestOpenAIAccountRepo{accounts: accounts}}
	hasAlternative, err = svc.HasEligibleOpenAIAccountAlternative(context.Background(), req)
	require.NoError(t, err)
	require.False(t, hasAlternative)
}
