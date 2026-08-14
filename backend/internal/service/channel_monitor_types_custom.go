package service

import "time"

// ChannelMonitorSortOrderUpdate updates one monitor's public display order.
type ChannelMonitorSortOrderUpdate struct {
	ID        int64
	SortOrder int
}

// UserMonitorOverview carries the public monitor list and its freshness facts.
type UserMonitorOverview struct {
	Items         []*UserMonitorView
	LastUpdatedAt *time.Time
	TrendPeriod   string
}
