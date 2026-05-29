package service

import (
	"context"
	"time"
)

const (
	AccountGroupRolePrimary = "primary"
	AccountGroupRoleBackup  = "backup"
)

type AccountGroup struct {
	AccountID            int64
	GroupID              int64
	Priority             int
	Role                 string
	Weight               int
	SortOrder            int
	SchedulingConfigured bool
	CreatedAt            time.Time

	Account *Account
	Group   *Group
}

type AccountSchedulingConfig struct {
	AccountID int64
	Role      string
	Weight    int
	SortOrder int
}

type AccountSchedulingEntry struct {
	AccountSchedulingConfig
	GroupID int64
	Account *Account
}

type AccountSchedulingConfigRepository interface {
	ListAccountSchedulingConfigs(ctx context.Context, groupID int64) ([]AccountSchedulingEntry, error)
	UpdateAccountSchedulingConfigs(ctx context.Context, groupID int64, configs []AccountSchedulingConfig) error
}

func (ag AccountGroup) NormalizedRole() string {
	switch ag.Role {
	case AccountGroupRoleBackup:
		return AccountGroupRoleBackup
	default:
		return AccountGroupRolePrimary
	}
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
