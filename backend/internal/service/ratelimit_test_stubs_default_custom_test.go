//go:build !unit

package service

import (
	"context"
	"time"
)

type rateLimitAccountRepoStub struct {
	rateLimitAccountRepoStubCustom
	setErrorCalls          int
	tempCalls              int
	updateCredentialsCalls int
	updateExtraCalls       int
	lastCredentials        map[string]any
	lastExtraUpdates       map[string]any
	lastErrorMsg           string
	lastTempReason         string
	lastErrorID            int64
	lastTempID             int64
	accountsByID           map[int64]*Account
}

func (r *rateLimitAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	account, ok := r.accountsByID[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (r *rateLimitAccountRepoStub) SetError(_ context.Context, id int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorID = id
	r.lastErrorMsg = errorMsg
	return nil
}

func (r *rateLimitAccountRepoStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.tempCalls++
	r.lastTempID = id
	r.lastTempUntil = until
	r.lastTempReason = reason
	return nil
}

func (r *rateLimitAccountRepoStub) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	r.updateCredentialsCalls++
	r.lastCredentials = shallowCopyMap(credentials)
	return nil
}

func (r *rateLimitAccountRepoStub) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updateExtraCalls++
	r.lastExtraUpdates = shallowCopyMap(updates)
	return nil
}

type openAI403CounterCacheStub struct {
	openAI403CounterCacheStubCustom
	counts     []int64
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
