package service

import (
	"context"
	"fmt"
	"time"
)

type AccountRecentFirstTokenStats struct {
	AvgFirstTokenMs float64
	SampleCount     int64
}

type GroupRecentUserUsage struct {
	UserID       int64     `json:"user_id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	RequestCount int64     `json:"request_count"`
	ActualCost   float64   `json:"actual_cost"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

type recentAccountFirstTokenStatsReader interface {
	GetRecentAccountFirstTokenStats(ctx context.Context, accountIDs []int64, groupID int64, since time.Time) (map[int64]AccountRecentFirstTokenStats, error)
}

type recentGroupUsersReader interface {
	ListRecentGroupUsers(ctx context.Context, groupID int64, since time.Time, limit int) ([]GroupRecentUserUsage, error)
}

func getRecentAccountFirstTokenStats(repo UsageLogRepository, ctx context.Context, accountIDs []int64, groupID int64, since time.Time) (map[int64]AccountRecentFirstTokenStats, error) {
	reader, ok := repo.(recentAccountFirstTokenStatsReader)
	if !ok {
		return nil, errUsageLogExtensionUnavailable("recent account first-token stats")
	}
	return reader.GetRecentAccountFirstTokenStats(ctx, accountIDs, groupID, since)
}

func listRecentGroupUsers(repo UsageLogRepository, ctx context.Context, groupID int64, since time.Time, limit int) ([]GroupRecentUserUsage, error) {
	reader, ok := repo.(recentGroupUsersReader)
	if !ok {
		return nil, errUsageLogExtensionUnavailable("recent group users")
	}
	return reader.ListRecentGroupUsers(ctx, groupID, since, limit)
}

func errUsageLogExtensionUnavailable(name string) error {
	return fmt.Errorf("usage log repository does not support %s", name)
}
