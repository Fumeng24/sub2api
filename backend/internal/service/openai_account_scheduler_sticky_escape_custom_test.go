package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newOpenAIAdvancedSchedulerRateLimitServiceCustomTest(enabled string, values ...string) *RateLimitService {
	service := newOpenAIAdvancedSchedulerRateLimitService(enabled, values...)
	service.accountRepo = &rateLimitAccountRepoStub{}
	return service
}

type schedulerOutboxRecordingOpenAIAccountRepo struct {
	schedulerTestOpenAIAccountRepo
	events []schedulerOutboxAppendCall
}

func (r *schedulerOutboxRecordingOpenAIAccountRepo) AppendSchedulerOutboxEvent(_ context.Context, eventType string, accountID *int64, groupID *int64, payload map[string]any) error {
	call := schedulerOutboxAppendCall{
		eventType: eventType,
		payload:   shallowCopyMap(payload),
	}
	if accountID != nil {
		v := *accountID
		call.accountID = &v
	}
	if groupID != nil {
		v := *groupID
		call.groupID = &v
	}
	r.events = append(r.events, call)
	return nil
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_HealthSignalsKeepStickyWithoutSchedulerEvent(t *testing.T) {
	ctx := context.Background()
	groupID := int64(10101)
	accounts := []Account{
		{ID: 21101, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, GroupIDs: []int64{groupID}},
		{ID: 21102, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, GroupIDs: []int64{groupID}},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_sticky_ttft_outbox": 21101}}
	repo := &schedulerOutboxRecordingOpenAIAccountRepo{schedulerTestOpenAIAccountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
	cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs = 15000
	cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.5
	svc := &OpenAIGatewayService{
		accountRepo:        repo,
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitServiceCustomTest("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{acquireResults: map[int64]bool{21102: true}}),
		openaiAccountStats: newOpenAIAccountRuntimeStats(),
	}

	fastTTFT := 14999
	svc.openaiAccountStats.report(21101, true, &fastTTFT)
	svc.openaiAccountStats.report(21101, true, &fastTTFT)
	slowTTFT := 20000
	for i := 0; i < 3; i++ {
		svc.openaiAccountStats.report(21101, true, &slowTTFT)
	}

	selection, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, "", "session_hash_sticky_ttft_outbox", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, int64(21101), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	require.Equal(t, int64(21101), cache.sessionBindings["openai:session_hash_sticky_ttft_outbox"])
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(&accounts[0]))
	require.Empty(t, repo.events)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}
