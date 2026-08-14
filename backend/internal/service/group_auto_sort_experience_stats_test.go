package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGroupAutoSortExperienceUsesRealTrafficAndTokenWeightedCacheRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	since := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)

	successPattern := `(?s)FROM usage_logs ul.*FROM channel_monitors cm.*COALESCE\(ul\.user_agent, ''\) LIKE 'Go-http-client/%'.*jsonb_each_text.*LOWER\(monitor_header\.header_name\) = 'user-agent'.*monitor_header\.header_value = COALESCE\(ul\.user_agent, ''\).*SUM\(cache_read_tokens\).*SUM\(input_tokens \+ cache_read_tokens\)`
	mock.ExpectQuery(successPattern).
		WithArgs(int64(7), sqlmock.AnyArg(), since).
		WillReturnRows(sqlmock.NewRows([]string{
			"account_id", "model_name", "success_count", "first_token_samples", "duration_samples",
			"p95_first_token_ms", "p95_duration_ms", "cache_read_tokens", "cache_eligible_tokens",
		}).AddRow(int64(11), nil, int64(2), int64(2), int64(2), 500.0, 1000.0, int64(90), int64(100)))

	failurePattern := `(?s)FROM ops_error_logs e.*FROM channel_monitors cm.*COALESCE\(e\.user_agent, ''\) LIKE 'Go-http-client/%'.*jsonb_each_text.*monitor_header\.header_value = COALESCE\(e\.user_agent, ''\)`
	mock.ExpectQuery(failurePattern).
		WithArgs(int64(7), sqlmock.AnyArg(), since).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "model_name", "failure_count", "failover_count"}))

	provider := newSQLGroupAutoSortExperienceProvider(db)
	stats, err := provider.StatsByAccountID(context.Background(), 7, []int64{11}, since)
	require.NoError(t, err)
	require.InDelta(t, 0.9, stats[11].cacheHitRate(), 1e-12)
	require.Equal(t, int64(90), stats[11].CacheReadTokens)
	require.Equal(t, int64(100), stats[11].CacheEligibleTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}
