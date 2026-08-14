//go:build unit

package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSyncGroupRelativeRateMultipliersReplacesOnlyRelativeCoefficients(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE user_group_rate_multipliers\s+SET discount_multiplier = NULL`).
		WithArgs(int64(9), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`(?s)DELETE FROM user_group_rate_multipliers\s+WHERE group_id = \$1\s+AND rate_multiplier IS NULL\s+AND discount_multiplier IS NULL\s+AND rpm_override IS NULL`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`(?s)INSERT INTO user_group_rate_multipliers \(user_id, group_id, discount_multiplier, created_at, updated_at\)`).
		WithArgs(int64(9), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err = (&userGroupRateRepository{sql: db}).SyncGroupRelativeRateMultipliers(context.Background(), 9, []service.GroupRelativeRateMultiplierInput{
		{UserID: 11, Multiplier: 0.8},
		{UserID: 12, Multiplier: 1.2},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncGroupRelativeRateMultipliersClearPreservesFixedRateAndRPMRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE user_group_rate_multipliers\s+SET discount_multiplier = NULL`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`(?s)DELETE FROM user_group_rate_multipliers\s+WHERE group_id = \$1\s+AND rate_multiplier IS NULL\s+AND discount_multiplier IS NULL\s+AND rpm_override IS NULL`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = (&userGroupRateRepository{sql: db}).SyncGroupRelativeRateMultipliers(context.Background(), 9, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
