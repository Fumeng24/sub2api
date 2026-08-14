//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupModelsListCandidatesFromAccounts_UsesBoundAccountMappings(t *testing.T) {
	accounts := []Account{
		{
			Platform: PlatformGemini,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3.1-flash-image": "gemini-3.1-flash-image",
					"gemini-3.1-pro-preview": "gemini-3.1-pro-preview",
				},
			},
		},
		{
			Platform: PlatformGemini,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3.1-flash-image": "gemini-3.1-flash-image",
					"gemini-*":               "gemini-3.1-flash-image",
				},
			},
		},
		{
			Platform: PlatformOpenAI,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.4": "gpt-5.4"},
			},
		},
	}
	service := &adminServiceImpl{
		groupRepo: &groupRepoStubForAdmin{getByID: &Group{ID: 42, Platform: PlatformGemini}},
		accountRepo: &modelsListAccountRepoStub{
			byGroup: map[int64][]Account{42: accounts},
		},
	}

	got, err := service.GetGroupModelsListCandidates(context.Background(), 42, PlatformGemini)
	require.NoError(t, err)

	require.Contains(t, got, "gemini-3.1-flash-image")
	require.Contains(t, got, "gemini-3.1-pro-preview")
	require.NotContains(t, got, "gpt-5.4")
}
