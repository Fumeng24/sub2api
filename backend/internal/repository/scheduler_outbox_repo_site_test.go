package repository

import (
	"context"
	"errors"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestSchedulerOutboxRepositoryListByGroupUsesSchedulerHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	now := time.Now()
	mock.ExpectQuery("FROM scheduler_history").
		WithArgs(int64(10), 400).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "account_id", "group_id", "payload", "created_at"}).
			AddRow(int64(7), "scheduling_blocked", int64(42), int64(10), []byte(`{"reason":"test"}`), now))

	events, err := repo.ListByGroup(context.Background(), 10, 20)

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.EqualValues(t, 7, events[0].ID)
	require.Equal(t, "scheduling_blocked", events[0].EventType)
	require.EqualValues(t, 42, *events[0].AccountID)
	require.EqualValues(t, 10, *events[0].GroupID)
	require.Equal(t, "test", events[0].Payload["reason"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryListByGroupFallsBackToOutboxWhenHistoryMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	now := time.Now()
	mock.ExpectQuery("FROM scheduler_history").
		WithArgs(int64(10), 400).
		WillReturnError(&pq.Error{Code: "42P01", Message: `relation "scheduler_history" does not exist`})
	mock.ExpectQuery("FROM scheduler_outbox").
		WithArgs(int64(10), 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "account_id", "group_id", "payload", "created_at"}).
			AddRow(int64(8), "scheduling_block_skipped", int64(43), int64(10), []byte(`{"reason":"last_group_account"}`), now))

	events, err := repo.ListByGroup(context.Background(), 10, 20)

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.EqualValues(t, 8, events[0].ID)
	require.Equal(t, "scheduling_block_skipped", events[0].EventType)
	require.Equal(t, "last_group_account", events[0].Payload["reason"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCompactSchedulerHistoryEventsMergesRepeatedEvents(t *testing.T) {
	accountID := int64(42)
	groupID := int64(10)
	newest := time.Date(2026, 7, 2, 2, 10, 0, 0, time.UTC)
	events := []service.SchedulerOutboxEvent{
		{
			ID:        3,
			EventType: "scheduling_blocked",
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":             "account_monitor_consecutive_failures",
				"source":             "account_monitor",
				"failure_category":   "account_monitor_consecutive_failures",
				"cooldown_minutes":   5,
				"block_granularity":  "",
				"latest_message":     "upstream HTTP 503 latest",
				"failure_threshold":  3,
				"irrelevant_version": "new",
			},
			CreatedAt: newest,
		},
		{
			ID:        2,
			EventType: "scheduling_blocked",
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":             "account_monitor_consecutive_failures",
				"source":             "account_monitor",
				"failure_category":   "account_monitor_consecutive_failures",
				"cooldown_minutes":   5,
				"block_granularity":  "",
				"latest_message":     "upstream HTTP 503 older",
				"failure_threshold":  3,
				"irrelevant_version": "old",
			},
			CreatedAt: newest.Add(-time.Minute),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 1)
	require.EqualValues(t, 3, compact[0].ID)
	require.Equal(t, "upstream HTTP 503 latest", compact[0].Payload["latest_message"])
	require.Equal(t, 2, compact[0].Payload["history_count"])
	require.Equal(t, newest.Format(time.RFC3339), compact[0].Payload["history_last_at"])
	require.Equal(t, newest.Add(-time.Minute).Format(time.RFC3339), compact[0].Payload["history_first_at"])
}

func TestCompactSchedulerHistoryEventsKeepsDistinctModelCooldowns(t *testing.T) {
	accountID := int64(42)
	groupID := int64(10)
	now := time.Now()
	events := []service.SchedulerOutboxEvent{
		{
			ID:        2,
			EventType: "scheduling_blocked",
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":            "account_monitor_model_unsupported",
				"failure_category":  "account_monitor_model_unsupported",
				"block_granularity": "model",
				"model_rate_limit":  "gpt-5.5",
			},
			CreatedAt: now,
		},
		{
			ID:        1,
			EventType: "scheduling_blocked",
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":            "account_monitor_model_unsupported",
				"failure_category":  "account_monitor_model_unsupported",
				"block_granularity": "model",
				"model_rate_limit":  "gpt-5.4",
			},
			CreatedAt: now.Add(-time.Minute),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 2)
	require.Equal(t, "gpt-5.5", compact[0].Payload["model_rate_limit"])
	require.Equal(t, "gpt-5.4", compact[1].Payload["model_rate_limit"])
}

func TestCompactSchedulerHistoryEventsSuppressesAccountChangeNearSchedulingBlock(t *testing.T) {
	accountID := int64(42)
	groupID := int64(10)
	now := time.Now()
	events := []service.SchedulerOutboxEvent{
		{
			ID:        3,
			EventType: service.SchedulerOutboxEventSchedulingBlocked,
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":           "account_monitor_consecutive_failures",
				"failure_category": "account_monitor_consecutive_failures",
			},
			CreatedAt: now,
		},
		{
			ID:        2,
			EventType: service.SchedulerOutboxEventAccountChanged,
			AccountID: &accountID,
			CreatedAt: now.Add(-time.Second),
		},
		{
			ID:        1,
			EventType: service.SchedulerOutboxEventAccountChanged,
			AccountID: &accountID,
			CreatedAt: now.Add(-time.Minute),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 2)
	require.EqualValues(t, 3, compact[0].ID)
	require.EqualValues(t, 1, compact[1].ID)
}

func TestCompactSchedulerHistoryEventsSuppressesBulkChangeNearSchedulingBlock(t *testing.T) {
	accountID := int64(42)
	groupID := int64(10)
	now := time.Now()
	events := []service.SchedulerOutboxEvent{
		{
			ID:        2,
			EventType: service.SchedulerOutboxEventSchedulingBlocked,
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":           "account_monitor_consecutive_failures",
				"failure_category": "account_monitor_consecutive_failures",
			},
			CreatedAt: now,
		},
		{
			ID:        1,
			EventType: service.SchedulerOutboxEventAccountBulkChanged,
			Payload:   map[string]any{"account_ids": []any{float64(42)}},
			CreatedAt: now.Add(-time.Second),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 1)
	require.EqualValues(t, 2, compact[0].ID)
}

func TestCompactSchedulerHistoryEventsSuppressesLowSignalEvents(t *testing.T) {
	accountID := int64(42)
	groupID := int64(10)
	now := time.Now()
	events := []service.SchedulerOutboxEvent{
		{
			ID:        3,
			EventType: service.SchedulerOutboxEventSchedulingBlocked,
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":           "account_monitor_consecutive_failures",
				"failure_category": "account_monitor_consecutive_failures",
			},
			CreatedAt: now,
		},
		{
			ID:        2,
			EventType: service.SchedulerOutboxEventAccountLastUsed,
			Payload:   map[string]any{"last_used": map[string]any{"42": float64(1782958506)}},
			CreatedAt: now.Add(-time.Second),
		},
		{
			ID:        1,
			EventType: service.SchedulerOutboxEventAccountBulkChanged,
			Payload:   map[string]any{"account_ids": []any{float64(42)}},
			CreatedAt: now.Add(-2 * time.Second),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 1)
	require.EqualValues(t, 3, compact[0].ID)
}

func TestCompactSchedulerHistoryEventsKeepsInformativeBulkChange(t *testing.T) {
	accountID := int64(42)
	groupID := int64(10)
	now := time.Now()
	events := []service.SchedulerOutboxEvent{
		{
			ID:        2,
			EventType: service.SchedulerOutboxEventSchedulingBlocked,
			AccountID: &accountID,
			GroupID:   &groupID,
			Payload: map[string]any{
				"reason":           "account_monitor_consecutive_failures",
				"failure_category": "account_monitor_consecutive_failures",
			},
			CreatedAt: now,
		},
		{
			ID:        1,
			EventType: service.SchedulerOutboxEventAccountBulkChanged,
			Payload: map[string]any{
				"account_ids": []any{float64(42)},
				"source":      "manual",
			},
			CreatedAt: now.Add(-time.Second),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 2)
	require.EqualValues(t, 2, compact[0].ID)
	require.EqualValues(t, 1, compact[1].ID)
}

func TestCompactSchedulerHistoryEventsDoesNotMergeOrdinaryAccountChanges(t *testing.T) {
	accountID := int64(42)
	now := time.Now()
	events := []service.SchedulerOutboxEvent{
		{
			ID:        2,
			EventType: service.SchedulerOutboxEventAccountChanged,
			AccountID: &accountID,
			Payload:   map[string]any{"field": "schedulable", "value": true},
			CreatedAt: now,
		},
		{
			ID:        1,
			EventType: service.SchedulerOutboxEventAccountChanged,
			AccountID: &accountID,
			Payload:   map[string]any{"field": "priority", "value": 10},
			CreatedAt: now.Add(-time.Minute),
		},
	}

	compact := compactSchedulerHistoryEvents(events, 10)

	require.Len(t, compact, 2)
	require.EqualValues(t, 2, compact[0].ID)
	require.EqualValues(t, 1, compact[1].ID)
}

func TestEnqueueSchedulerOutboxAlsoWritesSchedulerHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	accountID := int64(42)
	groupID := int64(10)
	payload := map[string]any{"reason": "account_monitor_consecutive_failures"}
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload)
		VALUES ($1, $2, $3, $4)
	`)).
		WithArgs("scheduling_blocked", &accountID, &groupID, []byte(`{"reason":"account_monitor_consecutive_failures"}`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO scheduler_history").
		WithArgs("scheduling_blocked", &accountID, &groupID, []byte(`{"reason":"account_monitor_consecutive_failures"}`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = enqueueSchedulerOutbox(context.Background(), db, "scheduling_blocked", &accountID, &groupID, payload)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnqueueSchedulerOutboxIgnoresSchedulerHistoryInsertFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	accountID := int64(42)
	groupID := int64(10)
	payload := map[string]any{"reason": "account_monitor_consecutive_failures"}
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload)
		VALUES ($1, $2, $3, $4)
	`)).
		WithArgs("scheduling_blocked", &accountID, &groupID, []byte(`{"reason":"account_monitor_consecutive_failures"}`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO scheduler_history").
		WithArgs("scheduling_blocked", &accountID, &groupID, []byte(`{"reason":"account_monitor_consecutive_failures"}`)).
		WillReturnError(errors.New("history insert failed"))

	err = enqueueSchedulerOutbox(context.Background(), db, "scheduling_blocked", &accountID, &groupID, payload)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnqueueSchedulerOutboxSkipsHistoryWhenDedupConflictDoesNothing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	accountID := int64(42)
	mock.ExpectExec("INSERT INTO scheduler_outbox").
		WithArgs("account_changed", &accountID, nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = enqueueSchedulerOutbox(context.Background(), db, "account_changed", &accountID, nil, nil)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
