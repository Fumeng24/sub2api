package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

type openAIAccountScheduleDecisionCustom struct {
	SlowReserveSelected bool
}

type openAIAccountSchedulerMetricsSnapshotCustom struct {
	SlowReserveSelectedTotal int64
	SlowReserveActiveCount   int
}

type openAIAccountSchedulerMetricsCustom struct {
	slowReserveSelectedTotal atomic.Int64
}

func (s *defaultOpenAIAccountScheduler) preferOAuthOverStickyCustom(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	current *AccountSelectionResult,
) *AccountSelectionResult {
	if current == nil || current.Account == nil || current.Account.Type == AccountTypeOAuth {
		return nil
	}
	if req.ExcludedIDs != nil {
		if _, excluded := req.ExcludedIDs[current.Account.ID]; excluded {
			return nil
		}
	}

	oauthReq := req
	if oauthReq.ExcludedIDs == nil {
		oauthReq.ExcludedIDs = map[int64]struct{}{}
	} else {
		oauthReq.ExcludedIDs = cloneInt64SetCustom(oauthReq.ExcludedIDs)
	}
	oauthReq.ExcludedIDs[current.Account.ID] = struct{}{}

	accounts, err := s.service.listSchedulableAccounts(ctx, oauthReq.GroupID, oauthReq.Platform)
	if err != nil || len(accounts) == 0 {
		return nil
	}

	var schedGroup *Group
	if oauthReq.GroupID != nil && s.service.schedulerSnapshot != nil {
		schedGroup, _ = s.service.schedulerSnapshot.GetGroupByID(ctx, *oauthReq.GroupID)
	}
	oauthReq.stableLowTTFT = isOpenAIStableLowTTFTGroup(schedGroup)

	oauthCandidates := make([]*Account, 0, len(accounts))
	loadReq := make([]AccountWithConcurrency, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if account.Type != AccountTypeOAuth {
			continue
		}
		if _, excluded := oauthReq.ExcludedIDs[account.ID]; excluded {
			continue
		}
		if !openAICompatibleAccountRequestEligibility(ctx, account, req.Platform, req.RequestedModel, req.RequireCompact, req.RequiredCapability).Allowed() {
			continue
		}
		if s.service.isOpenAIAccountRuntimeBlocked(account) {
			continue
		}
		if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
			s.service.BlockAccountScheduling(account, time.Time{}, "privacy_not_set")
			_ = s.service.accountRepo.SetError(ctx, account.ID,
				fmt.Sprintf("Privacy not set, required by group [%s]", schedGroup.Name))
			continue
		}
		if !s.isAccountRequestCompatible(ctx, account, oauthReq) || !s.isAccountTransportCompatible(account, oauthReq.RequiredTransport) {
			continue
		}
		oauthCandidates = append(oauthCandidates, account)
		loadReq = append(loadReq, AccountWithConcurrency{ID: account.ID, MaxConcurrency: account.EffectiveLoadFactor()})
	}
	if len(oauthCandidates) == 0 {
		return nil
	}

	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil {
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		}
	}
	plan := s.buildOpenAIAccountLoadPlan(ctx, oauthReq, oauthCandidates, loadMap)
	if len(plan.selectionOrder) == 0 {
		return nil
	}
	result, _, err := s.tryAcquireOpenAISelectionOrder(ctx, oauthReq, plan.selectionOrder)
	if err != nil || result == nil || result.Account == nil {
		return nil
	}
	slog.Info("sticky_oauth_preferred",
		"sticky_account_id", current.Account.ID,
		"oauth_account_id", result.Account.ID,
		"group_id", derefGroupID(req.GroupID),
		"model", req.RequestedModel,
	)
	return result
}

func cloneInt64SetCustom(in map[int64]struct{}) map[int64]struct{} {
	out := make(map[int64]struct{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func openAIStickyAccountSchedulableForGroupAt(account *Account, groupID *int64, now time.Time) bool {
	if account == nil {
		return false
	}
	if groupID == nil || *groupID <= 0 {
		return true
	}
	return account.IsSchedulableForGroupAt(*groupID, now)
}

func openAIStickySelectionGroupClearReasonCustom(account *Account, groupID *int64) string {
	if openAIStickyAccountSchedulableForGroupAt(account, groupID, time.Now()) {
		return ""
	}
	return "group_unschedulable"
}

func openAIStickySelectionGroupBlockedCustom(account *Account, groupID *int64) bool {
	return openAIStickySelectionGroupClearReasonCustom(account, groupID) != ""
}

func openAIAccountRequestIneligibleCustom(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) (bool, bool) {
	eligibility := openAICompatibleAccountRequestEligibility(ctx, account, req.Platform, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
	return !eligibility.Allowed(), true
}

func (s *defaultOpenAIAccountScheduler) customizeOpenAISelectionOrder(
	req OpenAIAccountScheduleRequest,
	base func([]openAIAccountCandidateScore) []openAIAccountCandidateScore,
) func([]openAIAccountCandidateScore) []openAIAccountCandidateScore {
	if !req.SubscriptionPriority {
		return base
	}
	return func(pool []openAIAccountCandidateScore) []openAIAccountCandidateScore {
		if len(pool) == 0 {
			return nil
		}
		oauth := make([]openAIAccountCandidateScore, 0, len(pool))
		others := make([]openAIAccountCandidateScore, 0, len(pool))
		for _, candidate := range pool {
			if candidate.account != nil && candidate.account.Type == AccountTypeOAuth {
				oauth = append(oauth, candidate)
			} else {
				others = append(others, candidate)
			}
		}
		selectionOrder := make([]openAIAccountCandidateScore, 0, len(pool))
		selectionOrder = append(selectionOrder, base(oauth)...)
		selectionOrder = append(selectionOrder, base(others)...)
		return selectionOrder
	}
}
