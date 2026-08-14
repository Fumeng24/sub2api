package service

import (
	"context"
	"time"
)

const (
	AccountGroupRolePrimary = "primary"
	AccountGroupRoleBackup  = "backup"
)

type AccountSchedulingConfig struct {
	AccountID            int64
	Priority             int
	Role                 string
	Weight               int
	SortOrder            int
	SchedulingConfigured bool
}

type AccountSchedulingEntry struct {
	AccountSchedulingConfig
	GroupID                       int64
	Account                       *Account
	State                         string
	BlockReason                   AccountSchedulingBlockReason
	GroupReserve                  bool
	GroupReserveUntil             *time.Time
	GroupReserveReason            string
	RecentUserAvgFirstTokenMs     *float64
	RecentUserFirstTokenSampleCnt int64
}

type AccountSchedulingConfigRepository interface {
	ListAccountSchedulingConfigs(ctx context.Context, groupID int64) ([]AccountSchedulingEntry, error)
	UpdateAccountSchedulingConfigs(ctx context.Context, groupID int64, configs []AccountSchedulingConfig) error
}

func (ag AccountGroup) NormalizedRole() string {
	if ag.Role == AccountGroupRoleBackup {
		return AccountGroupRoleBackup
	}
	return AccountGroupRolePrimary
}

func (ag AccountGroup) EffectiveWeight() int {
	if ag.Weight <= 0 {
		return 1
	}
	return ag.Weight
}

func (ag AccountGroup) EffectiveSortOrder() int {
	if ag.SortOrder != 0 {
		return ag.SortOrder
	}
	if ag.Priority != 0 {
		return ag.Priority
	}
	return 50
}

// SchedulingPriorityForGroup returns the account order that belongs to the
// requested group. Account.Priority is global legacy state and must not be
// used to rank an account shared by groups with different scheduling policies.
func (a *Account) SchedulingPriorityForGroup(groupID int64) int {
	if a == nil || groupID <= 0 {
		if a == nil {
			return 0
		}
		return a.Priority
	}
	for i := range a.AccountGroups {
		if a.AccountGroups[i].GroupID == groupID {
			return a.AccountGroups[i].EffectiveSortOrder()
		}
	}
	return a.Priority
}
