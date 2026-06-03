//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type snapshotHydrationCache struct {
	snapshot     []*Account
	accounts     map[int64]*Account
	setSnapshots [][]Account
}

func (c *snapshotHydrationCache) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	if c.snapshot == nil {
		return nil, false, nil
	}
	return c.snapshot, true, nil
}

func (c *snapshotHydrationCache) SetSnapshot(ctx context.Context, bucket SchedulerBucket, accounts []Account) error {
	copied := append([]Account(nil), accounts...)
	c.setSnapshots = append(c.setSnapshots, copied)
	c.snapshot = make([]*Account, 0, len(copied))
	for i := range copied {
		c.snapshot = append(c.snapshot, &copied[i])
	}
	return nil
}

func (c *snapshotHydrationCache) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if c.accounts == nil {
		return nil, nil
	}
	return c.accounts[accountID], nil
}

func (c *snapshotHydrationCache) SetAccount(ctx context.Context, account *Account) error {
	return nil
}

func (c *snapshotHydrationCache) DeleteAccount(ctx context.Context, accountID int64) error {
	return nil
}

func (c *snapshotHydrationCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return nil
}

func (c *snapshotHydrationCache) TryLockBucket(ctx context.Context, bucket SchedulerBucket, ttl time.Duration) (bool, error) {
	return true, nil
}

func (c *snapshotHydrationCache) UnlockBucket(ctx context.Context, bucket SchedulerBucket) error {
	return nil
}

func (c *snapshotHydrationCache) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *snapshotHydrationCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	return 0, nil
}

func (c *snapshotHydrationCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return nil
}

func (c *snapshotHydrationCache) SetBucketMembers(ctx context.Context, bucket SchedulerBucket, accountIDs []int64) error {
	return nil
}

func (c *snapshotHydrationCache) RemoveAccountFromBuckets(ctx context.Context, accountID int64) error {
	return nil
}

func TestSchedulerSnapshot_ListSchedulableAccounts_KeepsRuntimeBlockedAccountForAutoReturn(t *testing.T) {
	blockedUntil := time.Now().Add(30 * time.Minute)
	groupID := int64(7)
	cache := &snapshotHydrationCache{}
	repo := stubOpenAIAccountRepo{accounts: []Account{
		{
			ID:                      1,
			Platform:                PlatformOpenAI,
			Type:                    AccountTypeAPIKey,
			Status:                  StatusActive,
			Schedulable:             true,
			Concurrency:             1,
			TempUnschedulableUntil:  &blockedUntil,
			TempUnschedulableReason: "upstream_502",
			AccountGroups:           []AccountGroup{{GroupID: groupID, SortOrder: 10}},
			GroupIDs:                []int64{groupID},
		},
		{
			ID:            2,
			Platform:      PlatformOpenAI,
			Type:          AccountTypeAPIKey,
			Status:        StatusActive,
			Schedulable:   true,
			Concurrency:   1,
			AccountGroups: []AccountGroup{{GroupID: groupID, SortOrder: 20}},
			GroupIDs:      []int64{groupID},
		},
	}}
	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, repo, nil, nil, nil)

	accounts, _, err := schedulerSnapshot.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	if err != nil {
		t.Fatalf("ListSchedulableAccounts error: %v", err)
	}
	if len(accounts) != 1 || accounts[0].ID != 2 {
		t.Fatalf("expected only available account 2, got %#v", accounts)
	}
	if len(cache.setSnapshots) != 1 || len(cache.setSnapshots[0]) != 2 {
		t.Fatalf("expected blocked account to remain in snapshot bucket, got %#v", cache.setSnapshots)
	}

	expired := time.Now().Add(-time.Minute)
	cache.snapshot[0].TempUnschedulableUntil = &expired
	accounts, _, err = schedulerSnapshot.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	if err != nil {
		t.Fatalf("ListSchedulableAccounts cache hit error: %v", err)
	}
	if len(accounts) != 2 || accounts[0].ID != 1 || accounts[1].ID != 2 {
		t.Fatalf("expected expired account to return from same snapshot, got %#v", accounts)
	}
}

func TestOpenAISelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot(t *testing.T) {
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-4": "gpt-4",
					},
				},
			},
		},
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"api_key":       "sk-live",
					"model_mapping": map[string]any{"gpt-4": "gpt-4"},
				},
			},
		},
	}

	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, nil, nil, nil, nil)
	groupID := int64(2)
	svc := &OpenAIGatewayService{
		schedulerSnapshot: schedulerSnapshot,
		cache:             &stubGatewayCache{},
	}

	selection, err := svc.SelectAccountWithLoadAwareness(context.Background(), &groupID, "", "gpt-4", nil)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if selection == nil || selection.Account == nil {
		t.Fatalf("expected selected account")
	}
	if got := selection.Account.GetOpenAIApiKey(); got != "sk-live" {
		t.Fatalf("expected hydrated api key, got %q", got)
	}
}

func TestOpenAINewAcquiredSelectionResult_ReleasesSlotWhenHydrationFails(t *testing.T) {
	cache := &snapshotHydrationCache{
		accounts: map[int64]*Account{},
	}
	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, stubOpenAIAccountRepo{}, nil, nil, nil)
	svc := &OpenAIGatewayService{
		schedulerSnapshot: schedulerSnapshot,
	}
	releaseCalls := 0

	selection, err := svc.newAcquiredSelectionResult(context.Background(), &Account{ID: 1001}, func() {
		releaseCalls++
	})

	if err == nil {
		t.Fatalf("expected hydration error")
	}
	if selection != nil {
		t.Fatalf("expected nil selection on hydration error")
	}
	if releaseCalls != 1 {
		t.Fatalf("expected release to be called once, got %d", releaseCalls)
	}
}

func TestGatewaySelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot(t *testing.T) {
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          9,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
			},
		},
		accounts: map[int64]*Account{
			9: {
				ID:          9,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"api_key": "anthropic-live-key",
				},
			},
		},
	}

	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, nil, nil, nil, nil)
	svc := &GatewayService{
		schedulerSnapshot: schedulerSnapshot,
		cache:             &mockGatewayCacheForPlatform{},
		cfg:               testConfig(),
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "claude-3-5-sonnet-20241022", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account")
	}
	if got := result.Account.GetCredential("api_key"); got != "anthropic-live-key" {
		t.Fatalf("expected hydrated api key, got %q", got)
	}
}

func TestGatewaySelectAccountWithLoadAwareness_SkipsAntigravityGeminiFamilyRateLimitedSnapshot(t *testing.T) {
	resetAt := time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          1,
				Platform:    PlatformAntigravity,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				AccountGroups: []AccountGroup{
					{AccountID: 1, GroupID: 22},
				},
				GroupIDs: []int64{22},
				Extra: map[string]any{
					"mixed_scheduling": true,
					modelRateLimitsKey: map[string]any{
						antigravityGeminiModelRateLimitKey: map[string]any{
							"rate_limit_reset_at": resetAt,
						},
					},
				},
			},
			{
				ID:          2,
				Platform:    PlatformAntigravity,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    2,
				AccountGroups: []AccountGroup{
					{AccountID: 2, GroupID: 22},
				},
				GroupIDs: []int64{22},
				Extra: map[string]any{
					"mixed_scheduling": true,
				},
			},
		},
		accounts: map[int64]*Account{
			1: {ID: 1, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
			2: {ID: 2, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
		},
	}
	groupID := int64(22)
	svc := &GatewayService{
		schedulerSnapshot: NewSchedulerSnapshotService(cache, nil, nil, nil, nil, nil),
		groupRepo: &mockGroupRepoForGateway{
			groups: map[int64]*Group{
				groupID: {
					ID:       groupID,
					Platform: PlatformGemini,
					Status:   StatusActive,
					Hydrated: true,
				},
			},
		},
		concurrencyService: NewConcurrencyService(&mockConcurrencyCache{}),
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				Scheduling: config.GatewaySchedulingConfig{
					LoadBatchEnabled:         true,
					StickySessionMaxWaiting:  3,
					StickySessionWaitTimeout: time.Second,
					FallbackWaitTimeout:      time.Second,
					FallbackMaxWaiting:       10,
				},
			},
		},
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), &groupID, "", "gemini-3-flash-preview", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account")
	}
	if result.Account.ID != 2 {
		t.Fatalf("expected scheduler to skip Gemini-family limited antigravity account 1, got %d", result.Account.ID)
	}
}
