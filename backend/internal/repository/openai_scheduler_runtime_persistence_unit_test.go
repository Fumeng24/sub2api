//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpenAISchedulerRuntimeStatsLoad(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(`(?s)SELECT account_id, canonical_model, error_rate_ewma, ttft_ewma,.*FROM openai_account_scheduler_runtime_stats`).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id", "canonical_model", "error_rate_ewma", "ttft_ewma",
			"sample_count", "ttft_samples", "updated_at",
			"transient_failure_streak", "transient_last_failure_at", "transient_block_until",
			"slow_reserve_marked_at", "slow_reserve_last_touched_at", "slow_reserve_expires_at",
			"slow_reserve_reason", "slow_reserve_ttft_ms",
		}).AddRow(
			int64(42), "gpt-5.6-sol", 0.25, 1200.0,
			int64(20), int64(10), now,
			int64(3), now.Add(-time.Minute), now.Add(time.Minute),
			now.Add(-2*time.Minute), now, now.Add(3*time.Minute),
			"ttft", 15000,
		))

	records, err := (&accountRepository{sql: db}).LoadOpenAIAccountRuntimeStats(context.Background())
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, int64(42), records[0].AccountID)
	require.Equal(t, 3, records[0].TransientFailureStreak)
	require.NotNil(t, records[0].SlowReserveExpiresAt)
	require.Equal(t, 15000, records[0].SlowReserveTTFTMs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpenAISchedulerRuntimeStatsSaveUsesSeparateSafetyUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	ttft := 900.0
	mock.ExpectExec(`(?s)INSERT INTO openai_account_scheduler_runtime_stats.*ON CONFLICT`).
		WithArgs(
			int64(42), "gpt-5.6-sol", 0.1, ttft, int64(12), int64(8), now,
			0, nil, nil, nil, nil, nil, "", 0,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	repo := &accountRepository{sql: db}
	require.NoError(t, repo.SaveOpenAIAccountRuntimeStats(context.Background(), []service.OpenAIAccountRuntimeStatRecord{{
		AccountID: 42, CanonicalModel: "gpt-5.6-sol", ErrorRateEWMA: 0.1,
		TTFTEWMA: &ttft, SampleCount: 12, TTFTSamples: 8, UpdatedAt: now,
	}}))

	blockUntil := now.Add(45 * time.Second)
	mock.ExpectExec(`(?s)INSERT INTO openai_account_scheduler_runtime_stats.*transient_failure_streak.*ON CONFLICT`).
		WithArgs(int64(42), "gpt-5.6-sol", 3, now, blockUntil, nil, nil, nil, "", 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.SaveOpenAISchedulerSafetyState(context.Background(), service.OpenAISchedulerSafetyStateRecord{
		AccountID: 42, CanonicalModel: "gpt-5.6-sol",
		TransientFailureStreak: 3, TransientLastFailureAt: &now, TransientBlockUntil: &blockUntil,
	}))
	require.NoError(t, mock.ExpectationsWereMet())
}
