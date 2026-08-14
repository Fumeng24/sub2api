package service

import "time"

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
