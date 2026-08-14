package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryGetEndpointStatsWithUsageFiltersAppliesSharedFilters(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	requestType := int16(3)
	billingType := int8(2)
	filters := usagestats.UsageLogFilters{
		UserID:            42,
		APIKeyID:          7,
		GroupID:           9,
		Model:             "gpt-5.6",
		ModelFilterSource: usagestats.ModelSourceRequested,
		RequestType:       &requestType,
		BillingType:       &billingType,
		BillingMode:       "image",
	}

	mock.ExpectQuery("COALESCE\\(NULLIF\\(TRIM\\(requested_model\\), ''\\), model\\) = \\$6.*billing_mode = \\$9").
		WithArgs(start, end, int64(42), int64(7), int64(9), "gpt-5.6", requestType, int16(billingType), "image").
		WillReturnRows(sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"}).
			AddRow("/v1/responses", int64(3), int64(120), 0.30, 0.24))

	stats, err := repo.GetEndpointStatsWithUsageFilters(context.Background(), start, end, filters)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, "/v1/responses", stats[0].Endpoint)
	require.Equal(t, int64(3), stats[0].Requests)
	require.NoError(t, mock.ExpectationsWereMet())
}
