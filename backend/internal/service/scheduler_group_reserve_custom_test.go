package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type groupReserveAccountRepoStub struct {
	AccountRepository
	candidates map[int64][]Account
}

func (r groupReserveAccountRepoStub) ListModelAvailabilityCandidates(_ context.Context, groupID *int64, platforms []string, _ bool) ([]Account, error) {
	if groupID == nil {
		return nil, nil
	}
	allowedPlatforms := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowedPlatforms[platform] = struct{}{}
	}
	accounts := r.candidates[*groupID]
	result := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if _, allowed := allowedPlatforms[account.Platform]; allowed {
			result = append(result, account)
		}
	}
	return result, nil
}

func cooledOpenAIAccount(id int64, until time.Time, groupIDs ...int64) Account {
	return Account{
		ID:                      id,
		Platform:                PlatformOpenAI,
		Type:                    AccountTypeAPIKey,
		Status:                  StatusActive,
		Schedulable:             true,
		GroupIDs:                groupIDs,
		TempUnschedulableUntil:  &until,
		TempUnschedulableReason: groupReserveReasonUpstream5xx + ": temporary upstream failure",
	}
}

func TestGroupReserveCandidatesStayWithinCurrentGroup(t *testing.T) {
	until := time.Now().Add(5 * time.Minute)
	groupA := int64(101)
	groupB := int64(202)
	accountA := cooledOpenAIAccount(1, until, groupA)
	accountB := cooledOpenAIAccount(2, until, groupB)
	repo := groupReserveAccountRepoStub{candidates: map[int64][]Account{
		groupA: {accountA},
		groupB: {accountB},
	}}

	accounts, err := listGroupReserveCandidates(context.Background(), repo, &groupA, []string{PlatformOpenAI}, nil)

	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, accountA.ID, accounts[0].ID)

	svc := &OpenAIGatewayService{accountRepo: repo}
	selected, err := svc.selectGroupReserveOpenAIAccount(context.Background(), &groupA, PlatformOpenAI, "gpt-5.5", nil, OpenAIUpstreamTransportAny, "", "", false)
	require.NoError(t, err)
	require.Equal(t, accountA.ID, selected.ID)

	emptyGroup := int64(303)
	selected, err = svc.selectGroupReserveOpenAIAccount(context.Background(), &emptyGroup, PlatformOpenAI, "gpt-5.5", nil, OpenAIUpstreamTransportAny, "", "", false)
	require.ErrorIs(t, err, ErrNoAvailableAccounts)
	require.Nil(t, selected)
}

func TestGroupReserveNeverBypassesRuntimeBlocksOrExclusions(t *testing.T) {
	until := time.Now().Add(5 * time.Minute)
	groupID := int64(101)
	allowed := cooledOpenAIAccount(1, until, groupID)
	rateLimited := cooledOpenAIAccount(2, until, groupID)
	rateLimited.RateLimitResetAt = &until
	overloaded := cooledOpenAIAccount(3, until, groupID)
	overloaded.OverloadUntil = &until
	manual := cooledOpenAIAccount(4, until, groupID)
	manual.Schedulable = false
	notTransient := cooledOpenAIAccount(5, until, groupID)
	notTransient.TempUnschedulableReason = "account_monitor_auth_failed"
	repo := groupReserveAccountRepoStub{candidates: map[int64][]Account{
		groupID: {rateLimited, overloaded, manual, notTransient, allowed},
	}}
	svc := &OpenAIGatewayService{accountRepo: repo}

	selected, err := svc.selectGroupReserveOpenAIAccount(context.Background(), &groupID, PlatformOpenAI, "gpt-5.5", nil, OpenAIUpstreamTransportAny, "", "", false)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, allowed.ID, selected.ID)

	selected, err = svc.selectGroupReserveOpenAIAccount(context.Background(), &groupID, PlatformOpenAI, "gpt-5.5", map[int64]struct{}{allowed.ID: {}}, OpenAIUpstreamTransportAny, "", "", false)
	require.ErrorIs(t, err, ErrNoAvailableAccounts)
	require.Nil(t, selected)
}

func TestGroupReserveBypassesMatchingTransientRuntimeMirror(t *testing.T) {
	until := time.Now().Add(5 * time.Minute)
	groupID := int64(101)
	account := cooledOpenAIAccount(1, until, groupID)
	repo := groupReserveAccountRepoStub{candidates: map[int64][]Account{
		groupID: {account},
	}}
	svc := &OpenAIGatewayService{accountRepo: repo}
	svc.BlockAccountScheduling(&account, time.Now().Add(time.Minute), groupReserveReasonUpstream5xx)

	selected, err := svc.selectGroupReserveOpenAIAccount(
		context.Background(),
		&groupID,
		PlatformOpenAI,
		"gpt-5.5",
		nil,
		OpenAIUpstreamTransportAny,
		"",
		"",
		false,
	)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, account.ID, selected.ID)
}

func TestGroupReserveNeverBypassesOpenAIModelRuntimeCircuit(t *testing.T) {
	until := time.Now().Add(5 * time.Minute)
	groupID := int64(101)
	account := cooledOpenAIAccount(1, until, groupID)
	repo := groupReserveAccountRepoStub{candidates: map[int64][]Account{
		groupID: {account},
	}}
	svc := &OpenAIGatewayService{accountRepo: repo}
	now := time.Now()
	svc.recordOpenAIAccountModelTransientFailure(&account, "gpt-5.5", now)
	svc.recordOpenAIAccountModelTransientFailure(&account, "gpt-5.5", now.Add(time.Millisecond))

	selected, err := svc.selectGroupReserveOpenAIAccount(
		context.Background(),
		&groupID,
		PlatformOpenAI,
		"gpt-5.5",
		nil,
		OpenAIUpstreamTransportAny,
		"",
		"",
		false,
	)

	require.ErrorIs(t, err, ErrNoAvailableAccounts)
	require.Nil(t, selected)
}

func TestGroupReserveReasonEligibility(t *testing.T) {
	now := time.Now()
	until := now.Add(time.Minute)
	for _, test := range []struct {
		name   string
		reason string
		want   bool
	}{
		{name: "regular upstream 5xx", reason: groupReserveReasonUpstream5xx, want: true},
		{name: "pool upstream 5xx", reason: groupReserveReasonPool5xx + ": timeout", want: true},
		{name: "network status zero", reason: groupReserveReasonOpenAIIO, want: true},
		{name: "monitor consecutive failures", reason: groupReserveReasonMonitor + ": HTTP 503", want: true},
		{name: "authentication failure", reason: "account_monitor_auth_failed", want: false},
		{name: "insufficient balance", reason: "account_monitor_insufficient_balance", want: false},
		{name: "empty response", reason: "empty stream response", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			account := cooledOpenAIAccount(1, until, 101)
			account.TempUnschedulableReason = test.reason
			require.Equal(t, test.want, account.IsGroupReserveEligibleAt(now))
		})
	}
}
