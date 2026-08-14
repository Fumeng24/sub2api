package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUsageBillingRepositoryApply_SkipsArchiveDedupWhenArchiveTableMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewUsageBillingRepository(nil, db)
	cmd := &service.UsageBillingCommand{
		RequestID:   "req-missing-archive",
		APIKeyID:    11,
		UserID:      22,
		BalanceCost: 1.25,
	}
	cmd.Normalize()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`)).
		WithArgs(cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`)).
		WithArgs(cmd.RequestID, cmd.APIKeyID).
		WillReturnError(&pq.Error{Code: "42P01", Message: "relation \"usage_billing_dedup_archive\" does not exist"})
	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance
	`)).
		WithArgs(cmd.BalanceCost, cmd.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(98.75))
	mock.ExpectCommit()

	result, err := repo.Apply(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Applied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardAggregationRepositoryCleanupUsageBillingDedup_FallbackWhenArchiveTableMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newDashboardAggregationRepositoryWithSQL(db)
	mock.ExpectExec("INSERT INTO usage_billing_dedup_archive").
		WillReturnError(&pq.Error{Code: "42P01", Message: "relation \"usage_billing_dedup_archive\" does not exist"})
	mock.ExpectExec("DELETE FROM usage_billing_dedup").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.CleanupUsageBillingDedup(context.Background(), time.Now().UTC().AddDate(0, 0, -365)))
	require.NoError(t, mock.ExpectationsWereMet())
}
