package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func withAdminUserCustomFields(out *AdminUser, user *service.User) *AdminUser {
	if out != nil && user != nil {
		out.GroupDiscounts = user.GroupDiscounts
	}
	return out
}

func applyAPIKeyCustomFields(out *APIKey, key *service.APIKey) {
	if out != nil && key != nil {
		out.Category = key.Category
	}
}

func applyAdminGroupCustomFields(out *AdminGroup, group *service.Group) {
	if out != nil && group != nil {
		out.AutoSortConfig = group.AutoSortConfig
	}
}

func withGroupCustomFields(out Group, group *service.Group) Group {
	if group != nil {
		out.ForceOpenAIPriority = group.ForceOpenAIPriority
		out.OpenAIStableLowTTFT = group.OpenAIStableLowTTFT
		out.ModelsListConfig = group.ModelsListConfig
	}
	return out
}

func applyAccountCustomFields(out *Account, account *service.Account) {
	if out == nil || account == nil {
		return
	}

	out.Extra = redactAccountManagedExtra(account.Extra)
	out.UpstreamID = account.UpstreamID
	out.TempUnschedulableUntil = nil
	out.TempUnschedulableReason = ""
	out.TempUnschedulableStatusCode = nil
	if account.TempUnschedulableUntil == nil || !time.Now().Before(*account.TempUnschedulableUntil) {
		return
	}

	details := service.TempUnschedulableReasonDetailsFromRaw(account.TempUnschedulableReason)
	out.TempUnschedulableUntil = account.TempUnschedulableUntil
	out.TempUnschedulableReason = details.DisplayReason
	out.TempUnschedulableStatusCode = details.StatusCode
}

func withAccountGroupCustomFields(out *AccountGroup, accountGroup *service.AccountGroup) *AccountGroup {
	if out != nil && accountGroup != nil {
		out.Role = accountGroup.NormalizedRole()
		out.Weight = accountGroup.EffectiveWeight()
		out.SortOrder = accountGroup.EffectiveSortOrder()
		out.SchedulingConfigured = true
	}
	return out
}

func AccountSchedulingEntryFromService(entry *service.AccountSchedulingEntry) *AccountSchedulingEntry {
	if entry == nil {
		return nil
	}
	return &AccountSchedulingEntry{
		AccountID:                       entry.AccountID,
		GroupID:                         entry.GroupID,
		Role:                            entry.Role,
		Weight:                          entry.Weight,
		SortOrder:                       entry.SortOrder,
		SchedulingConfigured:            entry.SchedulingConfigured,
		Account:                         AccountFromServiceShallow(entry.Account),
		State:                           entry.State,
		BlockReason:                     entry.BlockReason.String(),
		GroupReserve:                    entry.GroupReserve,
		GroupReserveUntil:               entry.GroupReserveUntil,
		GroupReserveReason:              entry.GroupReserveReason,
		RecentUserAvgFirstTokenMs:       entry.RecentUserAvgFirstTokenMs,
		RecentUserFirstTokenSampleCount: entry.RecentUserFirstTokenSampleCnt,
	}
}

func applyRedeemCodeCustomFields(out *RedeemCode, redeemCode *service.RedeemCode) {
	if out != nil && redeemCode != nil {
		out.BusinessCategory = redeemCode.BusinessCategory
	}
}

func withUserSubscriptionCustomFields(out UserSubscription, subscription *service.UserSubscription) UserSubscription {
	if subscription != nil {
		out.AutoResetDaily = subscription.AutoResetDaily
	}
	return out
}
