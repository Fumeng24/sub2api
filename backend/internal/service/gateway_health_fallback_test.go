package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

type gatewayHealthFallbackConcurrencyCache struct {
	acquireCalls   map[int64]int
	acquireResults map[int64]bool
	loadMap        map[int64]*AccountLoadInfo
}

func (c *gatewayHealthFallbackConcurrencyCache) AcquireAccountSlot(_ context.Context, accountID int64, _ int, _ string) (bool, error) {
	if c.acquireCalls == nil {
		c.acquireCalls = make(map[int64]int)
	}
	c.acquireCalls[accountID]++
	if c.acquireResults != nil {
		if acquired, ok := c.acquireResults[accountID]; ok {
			return acquired, nil
		}
	}
	return true, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	return nil
}

func (c *gatewayHealthFallbackConcurrencyCache) GetAccountConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) GetAccountConcurrencyBatch(_ context.Context, accountIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = 0
	}
	return result, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) DecrementAccountWaitCount(context.Context, int64) error {
	return nil
}

func (c *gatewayHealthFallbackConcurrencyCache) GetAccountWaitingCount(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) AcquireUserSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) ReleaseUserSlot(context.Context, int64, string) error {
	return nil
}

func (c *gatewayHealthFallbackConcurrencyCache) GetUserConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) IncrementWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) DecrementWaitCount(context.Context, int64) error {
	return nil
}

func (c *gatewayHealthFallbackConcurrencyCache) GetAccountsLoadBatch(_ context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	result := make(map[int64]*AccountLoadInfo, len(accounts))
	for _, account := range accounts {
		if c.loadMap != nil {
			if load, ok := c.loadMap[account.ID]; ok {
				result[account.ID] = load
				continue
			}
		}
		result[account.ID] = &AccountLoadInfo{AccountID: account.ID}
	}
	return result, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) GetUsersLoadBatch(context.Context, []UserWithConcurrency) (map[int64]*UserLoadInfo, error) {
	return map[int64]*UserLoadInfo{}, nil
}

func (c *gatewayHealthFallbackConcurrencyCache) CleanupExpiredAccountSlots(context.Context, int64) error {
	return nil
}

func (c *gatewayHealthFallbackConcurrencyCache) CleanupStaleProcessSlots(context.Context, string) error {
	return nil
}

func TestGatewayServiceSchedulerCircuitOpenSkipsBadUpstreamAndSelectsHealthy(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)
	ctx = WithSchedulerEndpoint(ctx, "/v1/messages")
	model := "claude-3-5-sonnet-20241022"

	badID := int64(51001)
	healthyID := int64(51002)
	repo := stubOpenAIAccountRepo{accounts: []Account{
		{
			ID:          badID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          healthyID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    10,
		},
	}}
	concurrencyCache := &gatewayHealthFallbackConcurrencyCache{}
	svc := &GatewayService{
		accountRepo:        repo,
		cache:              &stubGatewayCache{},
		cfg:                &config.Config{RunMode: config.RunModeStandard},
		concurrencyService: NewConcurrencyService(concurrencyCache),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}
	svc.schedulerHealth.reportFailure(badID, model, "/v1/messages", "transient_transport", 0)

	result, err := svc.SelectAccountWithLoadAwareness(ctx, nil, "", model, nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness() error = %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account, got %#v", result)
	}
	if result.Account.ID != healthyID {
		t.Fatalf("selected account=%d want=%d", result.Account.ID, healthyID)
	}
	if result.WeakFallback {
		t.Fatalf("healthy alternative must not be marked weak fallback")
	}
	if got := concurrencyCache.acquireCalls[badID]; got != 0 {
		t.Fatalf("circuit-open account acquire calls=%d want=0", got)
	}
	if got := concurrencyCache.acquireCalls[healthyID]; got != 1 {
		t.Fatalf("healthy account acquire calls=%d want=1", got)
	}
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

func TestGatewayServiceSchedulerCircuitOpenSingleCandidateTransientFallback(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)
	ctx = WithSchedulerEndpoint(ctx, "/v1/messages")
	model := "claude-opus-4-8"

	accountID := int64(51501)
	repo := stubOpenAIAccountRepo{accounts: []Account{
		{
			ID:          accountID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
	}}
	concurrencyCache := &gatewayHealthFallbackConcurrencyCache{}
	svc := &GatewayService{
		accountRepo: repo,
		cache:       &stubGatewayCache{},
		cfg: &config.Config{
			RunMode: config.RunModeStandard,
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
		concurrencyService: NewConcurrencyService(concurrencyCache),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}
	svc.schedulerHealth.reportFailure(accountID, model, "/v1/messages", "transient_timeout", time.Minute)

	result, err := svc.SelectAccountWithLoadAwareness(ctx, nil, "", model, nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness() error = %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account, got %#v", result)
	}
	if result.Account.ID != accountID {
		t.Fatalf("selected account=%d want=%d", result.Account.ID, accountID)
	}
	if !result.Acquired {
		t.Fatalf("expected acquired single-candidate fallback")
	}
	if !result.WeakFallback || result.WeakFallbackReason != gatewaySingleCandidateCircuitFallbackReason {
		t.Fatalf("fallback flags = weak:%t reason:%q", result.WeakFallback, result.WeakFallbackReason)
	}
	if got := concurrencyCache.acquireCalls[accountID]; got != 1 {
		t.Fatalf("single circuit-open account acquire calls=%d want=1", got)
	}
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

func TestGatewayServiceSchedulerCircuitOpenAllCandidatesNoWeakFallback(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)
	ctx = WithSchedulerEndpoint(ctx, "/v1/messages")
	model := "claude-3-5-sonnet-20241022"

	firstID := int64(52001)
	secondID := int64(52002)
	repo := stubOpenAIAccountRepo{accounts: []Account{
		{
			ID:          firstID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    0,
		},
		{
			ID:          secondID,
			Platform:    PlatformAnthropic,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Priority:    10,
		},
	}}
	concurrencyCache := &gatewayHealthFallbackConcurrencyCache{}
	svc := &GatewayService{
		accountRepo:        repo,
		cache:              &stubGatewayCache{},
		cfg:                &config.Config{RunMode: config.RunModeStandard},
		concurrencyService: NewConcurrencyService(concurrencyCache),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}
	svc.schedulerHealth.reportFailure(firstID, model, "/v1/messages", "transient_transport", 0)
	svc.schedulerHealth.reportFailure(secondID, model, "/v1/messages", "transient_timeout", 0)

	result, err := svc.SelectAccountWithLoadAwareness(ctx, nil, "", model, nil, "", 0)
	if result != nil {
		t.Fatalf("expected no selection, got account=%v weak_fallback=%v", result.Account, result.WeakFallback)
	}
	if !errors.Is(err, ErrNoAvailableAccounts) {
		t.Fatalf("error=%v want ErrNoAvailableAccounts", err)
	}
	if got := concurrencyCache.acquireCalls[firstID]; got != 0 {
		t.Fatalf("first circuit-open account acquire calls=%d want=0", got)
	}
	if got := concurrencyCache.acquireCalls[secondID]; got != 0 {
		t.Fatalf("second circuit-open account acquire calls=%d want=0", got)
	}
}

func TestGatewayServiceWeakFallbackRequiresExplicitEnable(t *testing.T) {
	ctx := context.Background()
	model := "claude-3-5-sonnet-20241022"
	endpoint := "/v1/messages"

	accountID := int64(53001)
	account := &Account{
		ID:          accountID,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}
	concurrencyCache := &gatewayHealthFallbackConcurrencyCache{}
	svc := &GatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{*account}},
		cache:              &stubGatewayCache{},
		cfg:                &config.Config{RunMode: config.RunModeStandard},
		concurrencyService: NewConcurrencyService(concurrencyCache),
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}

	result, attempted, err := svc.selectGatewayCooldownFallbackResult(ctx, []*Account{account}, nil, "", model, endpoint, PlatformAnthropic, false, nil, false, false, nil, false)
	if err != nil {
		t.Fatalf("selectGatewayCooldownFallbackResult() error = %v", err)
	}
	if result != nil || attempted {
		t.Fatalf("default weak fallback must be disabled, got result=%#v attempted=%t", result, attempted)
	}
	if got := concurrencyCache.acquireCalls[accountID]; got != 0 {
		t.Fatalf("disabled weak fallback acquire calls=%d want=0", got)
	}

	svc.cfg.Gateway.Scheduling.WeakFallbackEnabled = true
	result, attempted, err = svc.selectGatewayCooldownFallbackResult(ctx, []*Account{account}, nil, "", model, endpoint, PlatformAnthropic, false, nil, false, false, nil, false)
	if err != nil {
		t.Fatalf("selectGatewayCooldownFallbackResult() enabled error = %v", err)
	}
	if result == nil || result.Account == nil || result.Account.ID != accountID {
		t.Fatalf("expected weak fallback account %d, got %#v", accountID, result)
	}
	if !attempted {
		t.Fatalf("expected weak fallback attempt")
	}
	if !result.WeakFallback {
		t.Fatalf("expected explicit weak fallback selection")
	}
	if got := concurrencyCache.acquireCalls[accountID]; got != 1 {
		t.Fatalf("weak fallback account acquire calls=%d want=1", got)
	}
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}
