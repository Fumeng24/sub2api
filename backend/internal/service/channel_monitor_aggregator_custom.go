package service

import (
	"context"
	"fmt"
	"time"
)

func (s *ChannelMonitorService) ListUserOverview(ctx context.Context) (*UserMonitorOverview, error) {
	views, err := s.ListUserView(ctx)
	if err != nil {
		return nil, err
	}
	return &UserMonitorOverview{
		Items:         views,
		LastUpdatedAt: latestUserMonitorCheckedAt(views),
		TrendPeriod:   fmt.Sprintf("%dd", monitorAvailability7Days),
	}, nil
}

func latestUserMonitorCheckedAt(views []*UserMonitorView) *time.Time {
	var latest *time.Time
	for _, view := range views {
		if view == nil {
			continue
		}
		for _, point := range view.Timeline {
			if point.CheckedAt.IsZero() {
				continue
			}
			checkedAt := point.CheckedAt
			if latest == nil || checkedAt.After(*latest) {
				value := checkedAt
				latest = &value
			}
		}
	}
	return latest
}
