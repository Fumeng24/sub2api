//go:build unit

package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUserGroupRateRepositoryGetEffectiveByUserIDPrefersFixedRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`(?s)SELECT ugr\.group_id,\s+COALESCE\(ugr\.rate_multiplier, g\.rate_multiplier \* ugr\.discount_multiplier\) AS effective_rate`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "effective_rate"}).
			AddRow(int64(1), 0.3).
			AddRow(int64(2), 0.6))

	rates, err := (&userGroupRateRepository{sql: db}).GetEffectiveByUserID(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{1: 0.3, 2: 0.6}, rates)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserGroupRateRepositoryGetRateConfigReturnsFixedAndDiscount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT rate_multiplier, discount_multiplier FROM user_group_rate_multipliers`).
		WithArgs(int64(42), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier", "discount_multiplier"}).AddRow(0.25, 0.8))

	cfg, err := (&userGroupRateRepository{sql: db}).GetRateConfigByUserAndGroup(context.Background(), 42, 7)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.RateMultiplier)
	require.NotNil(t, cfg.DiscountMultiplier)
	require.InDelta(t, 0.25, *cfg.RateMultiplier, 1e-12)
	require.InDelta(t, 0.8, *cfg.DiscountMultiplier, 1e-12)
	require.NoError(t, mock.ExpectationsWereMet())
}
