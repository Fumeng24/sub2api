package service

import (
	"context"
	"time"
)

type rateLimitAccountRepoStub struct {
	AccountRepository
	setErrorCalls          int
	tempCalls              int
	updateCredentialsCalls int
	lastCredentials        map[string]any
	lastErrorMsg           string
	lastTempReason         string
	lastTempUntil          time.Time
}

func (r *rateLimitAccountRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorMsg = errorMsg
	return nil
}

func (r *rateLimitAccountRepoStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	r.tempCalls++
	r.lastTempUntil = until
	r.lastTempReason = reason
	return nil
}

func (r *rateLimitAccountRepoStub) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	r.updateCredentialsCalls++
	r.lastCredentials = cloneCredentials(credentials)
	return nil
}

type openAI403CounterCacheStub struct {
	counts     []int64
	lastCount  int64
	resetCalls []int64
	err        error
}

func (s *openAI403CounterCacheStub) IncrementOpenAI403Count(_ context.Context, _ int64, _ int) (int64, error) {
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

func (s *openAI403CounterCacheStub) ResetOpenAI403Count(_ context.Context, accountID int64) error {
	s.resetCalls = append(s.resetCalls, accountID)
	return nil
}
