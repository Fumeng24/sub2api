//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestTrySetGroupTempUnschedulableUnlessLastProtectsLastAccount(t *testing.T) {
	repo, groupID, accounts := newSchedulerProtectionTestPool(t, 1)

	applied, err := repo.TrySetGroupTempUnschedulableUnlessLast(
		context.Background(),
		accounts[0].ID,
		groupID,
		service.PlatformOpenAI,
		time.Now().Add(10*time.Minute),
		"upstream_503",
	)
	require.NoError(t, err)
	require.False(t, applied)

	got, err := repo.GetByID(context.Background(), accounts[0].ID)
	require.NoError(t, err)
	require.False(t, got.IsGroupTempUnschedulableAt(groupID, time.Now()))
}

func TestTrySetGroupTempUnschedulableUnlessLastAppliesWithAnotherCandidate(t *testing.T) {
	repo, groupID, accounts := newSchedulerProtectionTestPool(t, 2)
	cache := &schedulerCacheRecorder{}
	repo.schedulerCache = cache
	until := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)

	applied, err := repo.TrySetGroupTempUnschedulableUnlessLast(
		context.Background(),
		accounts[0].ID,
		groupID,
		service.PlatformOpenAI,
		until,
		"upstream_503",
	)
	require.NoError(t, err)
	require.True(t, applied)

	got, err := repo.GetByID(context.Background(), accounts[0].ID)
	require.NoError(t, err)
	require.True(t, got.IsGroupTempUnschedulableAt(groupID, time.Now()))
	require.Len(t, cache.setAccounts, 1)
	require.Equal(t, accounts[0].ID, cache.setAccounts[0].ID)

	candidates, err := repo.ListSchedulableByGroupIDAndPlatform(
		context.Background(),
		groupID,
		service.PlatformOpenAI,
	)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, accounts[1].ID, candidates[0].ID)
}

func TestTrySetGroupTempUnschedulableUnlessLastDoesNotShortenCooldown(t *testing.T) {
	repo, groupID, accounts := newSchedulerProtectionTestPool(t, 2)
	longUntil := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	shortUntil := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)

	applied, err := repo.TrySetGroupTempUnschedulableUnlessLast(
		context.Background(),
		accounts[0].ID,
		groupID,
		service.PlatformOpenAI,
		longUntil,
		"long-cooldown",
	)
	require.NoError(t, err)
	require.True(t, applied)

	applied, err = repo.TrySetGroupTempUnschedulableUnlessLast(
		context.Background(),
		accounts[0].ID,
		groupID,
		service.PlatformOpenAI,
		shortUntil,
		"short-cooldown",
	)
	require.NoError(t, err)
	require.False(t, applied)

	got, err := repo.GetByID(context.Background(), accounts[0].ID)
	require.NoError(t, err)
	require.True(t, got.IsGroupTempUnschedulableAt(groupID, shortUntil))
	require.False(t, got.IsGroupTempUnschedulableAt(groupID, longUntil.Add(time.Second)))

	block, ok := got.Extra["group_temp_unschedulable"].(map[string]any)
	require.True(t, ok)
	entry, ok := block[fmt.Sprintf("%d", groupID)].(map[string]any)
	require.True(t, ok)
	require.Equal(t, longUntil.Format(time.RFC3339), entry["until"])
	require.Equal(t, "long-cooldown", entry["reason"])
}

func TestTrySetGroupTempUnschedulableUnlessLastSerializesConcurrentBlocks(t *testing.T) {
	repo, groupID, accounts := newSchedulerProtectionTestPool(t, 2)
	start := make(chan struct{})
	results := make(chan bool, len(accounts))
	errs := make(chan error, len(accounts))
	var wg sync.WaitGroup

	for _, account := range accounts {
		accountID := account.ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			applied, err := repo.TrySetGroupTempUnschedulableUnlessLast(
				context.Background(),
				accountID,
				groupID,
				service.PlatformOpenAI,
				time.Now().Add(10*time.Minute),
				"concurrent-upstream-failure",
			)
			results <- applied
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	appliedCount := 0
	for applied := range results {
		if applied {
			appliedCount++
		}
	}
	require.Equal(t, 1, appliedCount)

	candidates, err := repo.ListSchedulableByGroupIDAndPlatform(
		context.Background(),
		groupID,
		service.PlatformOpenAI,
	)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
}

func TestTrySetGroupTempUnschedulableUnlessLastIgnoresQuotaExhaustedPeer(t *testing.T) {
	repo, groupID, accounts := newSchedulerProtectionTestPool(t, 2)
	require.NoError(t, repo.UpdateExtra(context.Background(), accounts[1].ID, map[string]any{
		"quota_limit": 10.0,
		"quota_used":  10.0,
	}))

	applied, err := repo.TrySetGroupTempUnschedulableUnlessLast(
		context.Background(),
		accounts[0].ID,
		groupID,
		service.PlatformOpenAI,
		time.Now().Add(10*time.Minute),
		"upstream_503",
	)

	require.NoError(t, err)
	require.False(t, applied)
	got, err := repo.GetByID(context.Background(), accounts[0].ID)
	require.NoError(t, err)
	require.False(t, got.IsGroupTempUnschedulableAt(groupID, time.Now()))
}

func newSchedulerProtectionTestPool(t *testing.T, count int) (*accountRepository, int64, []*service.Account) {
	t.Helper()
	ctx := context.Background()
	seed := time.Now().UnixNano()
	group := mustCreateGroup(t, integrationEntClient, &service.Group{
		Name: fmt.Sprintf("scheduler-protection-%d", seed),
	})
	accounts := make([]*service.Account, 0, count)
	for i := 0; i < count; i++ {
		account := mustCreateAccount(t, integrationEntClient, &service.Account{
			Name:        fmt.Sprintf("scheduler-protection-%d-%d", seed, i),
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
		})
		mustBindAccountToGroup(t, integrationEntClient, account.ID, group.ID, i+1)
		accounts = append(accounts, account)
	}

	t.Cleanup(func() {
		for _, account := range accounts {
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM scheduler_outbox WHERE account_id = $1`, account.ID)
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM account_groups WHERE account_id = $1`, account.ID)
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, account.ID)
		}
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM groups WHERE id = $1`, group.ID)
	})

	return newAccountRepositoryWithSQL(integrationEntClient, integrationDB, nil), group.ID, accounts
}
