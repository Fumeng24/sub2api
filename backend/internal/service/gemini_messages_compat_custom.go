package service

import "context"

// accountsToPointers converts the fallback repository result to the snapshot
// service representation used by Gemini compatibility routing.
func accountsToPointers(accounts []Account) []*Account {
	result := make([]*Account, len(accounts))
	for i := range accounts {
		result[i] = &accounts[i]
	}
	return result
}

func (s *GeminiMessagesCompatService) selectBestGeminiAccountWithPolicy(
	ctx context.Context,
	groupID *int64,
	accounts []Account,
	requestedModel string,
	excludedIDs map[int64]struct{},
	platform string,
	useMixedScheduling bool,
) *Account {
	accountPointers := accountsToPointers(accounts)
	if !accountsContainCurrentGroupBinding(accountPointers, groupID) {
		return s.selectBestGeminiAccount(ctx, accounts, requestedModel, excludedIDs, platform, useMixedScheduling)
	}

	var selected *Account
	precheckResult := s.buildPreCheckUsageResultMap(ctx, accounts, requestedModel)
	for _, account := range accountPointers {
		if _, excluded := excludedIDs[account.ID]; excluded {
			continue
		}
		if !s.isAccountUsableForRequestWithPrecheck(ctx, account, requestedModel, platform, useMixedScheduling, precheckResult) {
			continue
		}
		if selected == nil || isAccountBetterByCurrentGroupOrder(account, selected, groupID) {
			selected = account
		}
	}
	return selected
}
