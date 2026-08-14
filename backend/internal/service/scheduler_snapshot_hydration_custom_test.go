//go:build unit

package service

import (
	"context"
	"testing"
	"time"
)

type customSnapshotHydrationCache struct {
	*snapshotHydrationCache
	setSnapshots [][]Account
}

func newCustomSnapshotHydrationCache() *customSnapshotHydrationCache {
	return &customSnapshotHydrationCache{snapshotHydrationCache: &snapshotHydrationCache{}}
}

func (c *customSnapshotHydrationCache) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if c.snapshot == nil {
		return nil, false, nil
	}
	return c.snapshot, true, nil
}

func (c *customSnapshotHydrationCache) SetSnapshot(ctx context.Context, bucket SchedulerBucket, _ SchedulerBucketWriteToken, accounts []Account) error {
	copied := append([]Account(nil), accounts...)
	c.setSnapshots = append(c.setSnapshots, copied)
	c.snapshot = make([]*Account, 0, len(copied))
	for i := range copied {
		c.snapshot = append(c.snapshot, &copied[i])
	}
	return nil
}

func TestSchedulerSnapshot_ListSchedulableAccounts_KeepsRuntimeBlockedAccountForAutoReturn(t *testing.T) {
	blockedUntil := time.Now().Add(30 * time.Minute)
	groupID := int64(7)
	cache := newCustomSnapshotHydrationCache()
	repo := stubOpenAIAccountRepo{accounts: []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, TempUnschedulableUntil: &blockedUntil, TempUnschedulableReason: "upstream_502", AccountGroups: []AccountGroup{{GroupID: groupID, SortOrder: 10}}, GroupIDs: []int64{groupID}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []AccountGroup{{GroupID: groupID, SortOrder: 20}}, GroupIDs: []int64{groupID}},
	}}
	snapshot := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)
	accounts, _, err := snapshot.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	if err != nil || len(accounts) != 1 || accounts[0].ID != 2 || len(cache.setSnapshots) != 1 || len(cache.setSnapshots[0]) != 2 {
		t.Fatalf("unexpected initial snapshot result: accounts=%#v snapshots=%#v err=%v", accounts, cache.setSnapshots, err)
	}
	expired := time.Now().Add(-time.Minute)
	cache.snapshot[0].TempUnschedulableUntil = &expired
	accounts, _, err = snapshot.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	if err != nil || len(accounts) != 2 || accounts[0].ID != 1 || accounts[1].ID != 2 {
		t.Fatalf("expected expired account to return from snapshot: accounts=%#v err=%v", accounts, err)
	}
}

func TestSchedulerSnapshot_ListSchedulableAccounts_FiltersGroupScopedTempBlock(t *testing.T) {
	blockedUntil := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
	groupID, otherGroupID := int64(7), int64(8)
	cache := newCustomSnapshotHydrationCache()
	repo := groupAwareStubOpenAIAccountRepo{stubOpenAIAccountRepo{accounts: []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []AccountGroup{{GroupID: groupID, SortOrder: 10}, {GroupID: otherGroupID, SortOrder: 10}}, GroupIDs: []int64{groupID, otherGroupID}, Extra: map[string]any{groupTempUnschedulableKey: map[string]any{"7": map[string]any{"until": blockedUntil, "reason": "upstream_503"}}}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, AccountGroups: []AccountGroup{{GroupID: groupID, SortOrder: 20}}, GroupIDs: []int64{groupID}},
	}}}
	snapshot := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)
	accounts, _, err := snapshot.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	if err != nil || len(accounts) != 1 || accounts[0].ID != 2 {
		t.Fatalf("unexpected blocked group result: accounts=%#v err=%v", accounts, err)
	}
	cache.snapshot = nil
	accounts, _, err = snapshot.ListSchedulableAccounts(context.Background(), &otherGroupID, PlatformOpenAI, false)
	if err != nil || len(accounts) != 1 || accounts[0].ID != 1 {
		t.Fatalf("unexpected other group result: accounts=%#v err=%v", accounts, err)
	}
}
