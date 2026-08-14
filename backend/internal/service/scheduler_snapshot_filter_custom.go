package service

import "time"

func filterSchedulableAccountsForGroup(accounts []Account, groupID *int64) []Account {
	if len(accounts) == 0 {
		return accounts
	}

	now := time.Now()
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if groupID != nil && *groupID > 0 {
			if account.IsSchedulableAt(now) {
				account.Priority = account.SchedulingPriorityForGroup(*groupID)
				filtered = append(filtered, account)
			}
			continue
		}
		if account.IsSchedulableAt(now) {
			filtered = append(filtered, account)
		}
	}
	return filtered
}
