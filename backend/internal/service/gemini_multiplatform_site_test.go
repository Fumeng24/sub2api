//go:build unit

package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGeminiMessagesCompatService_SelectAccountForModelWithExclusions_GroupOrderOverridesGlobalPreference(t *testing.T) {
	ctx := context.Background()
	groupID := int64(7)
	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now()

	repo := &mockAccountRepoForGemini{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Type:        AccountTypeAPIKey,
				Priority:    99,
				Status:      StatusActive,
				Schedulable: true,
				LastUsedAt:  &newTime,
				AccountGroups: []AccountGroup{{
					AccountID:            1,
					GroupID:              groupID,
					Priority:             50,
					SortOrder:            10,
					Weight:               1,
					SchedulingConfigured: true,
				}},
			},
			{
				ID:          2,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				LastUsedAt:  &oldTime,
				AccountGroups: []AccountGroup{{
					AccountID:            2,
					GroupID:              groupID,
					Priority:             1,
					SortOrder:            20,
					Weight:               1000,
					SchedulingConfigured: true,
				}},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	cache := &mockGatewayCacheForGemini{}
	groupRepo := &mockGroupRepoForGemini{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Platform: PlatformGemini},
		},
	}

	svc := &GeminiMessagesCompatService{
		accountRepo: repo,
		groupRepo:   groupRepo,
		cache:       cache,
	}

	acc, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gemini-2.5-flash", nil)
	require.NoError(t, err)
	require.NotNil(t, acc)
	require.Equal(t, int64(1), acc.ID)
}

func TestGeminiMessagesCompatService_SelectAccountForModelWithExclusions_ForcePlatformFallbackUsesGlobalRanking(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9)
	ctx = context.WithValue(ctx, ctxkey.ForcePlatform, PlatformAntigravity)

	repo := &mockAccountRepoForGemini{
		listByGroupFunc: func(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
			return nil, nil
		},
		listByPlatformFunc: func(ctx context.Context, platforms []string) ([]Account, error) {
			return []Account{
				{ID: 1, Platform: PlatformAntigravity, Priority: 9, Status: StatusActive, Schedulable: true},
				{ID: 2, Platform: PlatformAntigravity, Priority: 1, Status: StatusActive, Schedulable: true},
			}, nil
		},
		accountsByID: map[int64]*Account{
			1: {ID: 1, Platform: PlatformAntigravity, Priority: 9, Status: StatusActive, Schedulable: true},
			2: {ID: 2, Platform: PlatformAntigravity, Priority: 1, Status: StatusActive, Schedulable: true},
		},
	}

	cache := &mockGatewayCacheForGemini{}
	groupRepo := &mockGroupRepoForGemini{groups: map[int64]*Group{}}

	svc := &GeminiMessagesCompatService{
		accountRepo: repo,
		groupRepo:   groupRepo,
		cache:       cache,
	}

	acc, err := svc.SelectAccountForModelWithExclusions(ctx, &groupID, "", "gemini-2.5-flash", nil)
	require.NoError(t, err)
	require.NotNil(t, acc)
	require.Equal(t, int64(2), acc.ID)
}
