package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryListRecentGroupUsers(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	since := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	lastUsed := since.Add(5 * time.Minute)
	mock.ExpectQuery("FROM usage_logs ul").
		WithArgs(int64(42), since, service.StatusActive, 100).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "username", "request_count", "actual_cost", "last_used_at"}).
			AddRow(int64(7), "user@example.com", "alice", int64(12), 1.25, lastUsed))

	users, err := repo.ListRecentGroupUsers(context.Background(), 42, since, 100)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, int64(7), users[0].UserID)
	require.Equal(t, "user@example.com", users[0].Email)
	require.Equal(t, "alice", users[0].Username)
	require.Equal(t, int64(12), users[0].RequestCount)
	require.Equal(t, 1.25, users[0].ActualCost)
	require.Equal(t, lastUsed, users[0].LastUsedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}
