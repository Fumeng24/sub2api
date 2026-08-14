package service

import (
	"context"
	"fmt"
	"time"
)

type rateLimitAccountRepoStubCustom struct {
	AccountRepository
	lastTempUntil          time.Time
	setRateLimitedCalls    int
	lastRateLimitedID      int64
	lastRateLimitedResetAt time.Time
	modelRateLimitCalls    int
	lastModelRateLimitID   int64
	lastModelRateLimitKey  string
	lastModelRateLimitAt   time.Time
	lastModelRateLimitWhy  string
	schedulableByGroup     map[int64][]Account
	schedulableByGroupPlat map[string][]Account
	outboxEvents           []schedulerOutboxAppendCall
}

type schedulerOutboxAppendCall struct {
	eventType string
	accountID *int64
	groupID   *int64
	payload   map[string]any
}

func (r *rateLimitAccountRepoStub) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	r.setRateLimitedCalls++
	r.lastRateLimitedID = id
	r.lastRateLimitedResetAt = resetAt
	return nil
}

func (r *rateLimitAccountRepoStub) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time, reason ...string) error {
	r.modelRateLimitCalls++
	r.lastModelRateLimitID = id
	r.lastModelRateLimitKey = scope
	r.lastModelRateLimitAt = resetAt
	if len(reason) > 0 {
		r.lastModelRateLimitWhy = reason[0]
	} else {
		r.lastModelRateLimitWhy = ""
	}
	return nil
}

func (r *rateLimitAccountRepoStub) ListSchedulableByGroupID(_ context.Context, groupID int64) ([]Account, error) {
	if r.schedulableByGroup == nil {
		return nil, nil
	}
	return r.schedulableByGroup[groupID], nil
}

func (r *rateLimitAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]Account, error) {
	if r.schedulableByGroupPlat != nil {
		return r.schedulableByGroupPlat[groupPlatformKey(groupID, platform)], nil
	}
	if r.schedulableByGroup == nil {
		return nil, nil
	}
	accounts := r.schedulableByGroup[groupID]
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == platform {
			filtered = append(filtered, account)
		}
	}
	return filtered, nil
}

func (r *rateLimitAccountRepoStub) AppendSchedulerOutboxEvent(_ context.Context, eventType string, accountID *int64, groupID *int64, payload map[string]any) error {
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
	r.outboxEvents = append(r.outboxEvents, call)
	return nil
}

func groupPlatformKey(groupID int64, platform string) string {
	return fmt.Sprintf("%d:%s", groupID, platform)
}

type openAI403CounterCacheStubCustom struct {
	lastCount int64
}

type transientErrorCounterCacheStub struct {
	counts     []int64
	lastCount  int64
	resetCalls []int64
	err        error
	onceSeen   map[string]struct{}
	onceCalls  []string
}

func (s *transientErrorCounterCacheStub) IncrementTransientErrorCount(_ context.Context, _ int64, _ int) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	if len(s.counts) == 0 {
		s.lastCount = 1
		return 1, nil
	}
	count := s.counts[0]
	s.counts = s.counts[1:]
	s.lastCount = count
	return count, nil
}

func (s *transientErrorCounterCacheStub) ResetTransientErrorCount(_ context.Context, accountID int64) error {
	s.resetCalls = append(s.resetCalls, accountID)
	return nil
}

func (s *transientErrorCounterCacheStub) IncrementTransientErrorCountOnce(_ context.Context, accountID int64, requestID string, _ int) (int64, bool, error) {
	if s.err != nil {
		return 0, false, s.err
	}
	key := fmt.Sprintf("%d:%s", accountID, requestID)
	if s.onceSeen == nil {
		s.onceSeen = make(map[string]struct{})
	}
	s.onceCalls = append(s.onceCalls, key)
	if _, seen := s.onceSeen[key]; seen {
		return s.lastCount, false, nil
	}
	s.onceSeen[key] = struct{}{}
	s.lastCount++
	return s.lastCount, true, nil
}
