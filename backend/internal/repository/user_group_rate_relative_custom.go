package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// SyncGroupRelativeRateMultipliers replaces only the relative multiplier part
// for a group. Fixed final rates and RPM overrides remain untouched.
func (r *userGroupRateRepository) SyncGroupRelativeRateMultipliers(
	ctx context.Context,
	groupID int64,
	entries []service.GroupRelativeRateMultiplierInput,
) error {
	if db, ok := r.sql.(*sql.DB); ok {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()

		if err := syncGroupRelativeRateMultipliers(ctx, tx, groupID, entries); err != nil {
			return err
		}
		return tx.Commit()
	}

	return syncGroupRelativeRateMultipliers(ctx, r.sql, groupID, entries)
}

func syncGroupRelativeRateMultipliers(
	ctx context.Context,
	executor sqlExecutor,
	groupID int64,
	entries []service.GroupRelativeRateMultiplierInput,
) error {
	keepUserIDs := make([]int64, 0, len(entries))
	for _, entry := range entries {
		keepUserIDs = append(keepUserIDs, entry.UserID)
	}

	if len(keepUserIDs) == 0 {
		if _, err := executor.ExecContext(ctx, `
			UPDATE user_group_rate_multipliers
			SET discount_multiplier = NULL, updated_at = NOW()
			WHERE group_id = $1
		`, groupID); err != nil {
			return err
		}
	} else {
		if _, err := executor.ExecContext(ctx, `
			UPDATE user_group_rate_multipliers
			SET discount_multiplier = NULL, updated_at = NOW()
			WHERE group_id = $1 AND user_id <> ALL($2)
		`, groupID, pq.Array(keepUserIDs)); err != nil {
			return err
		}
	}

	if _, err := executor.ExecContext(ctx, `
		DELETE FROM user_group_rate_multipliers
		WHERE group_id = $1
		  AND rate_multiplier IS NULL
		  AND discount_multiplier IS NULL
		  AND rpm_override IS NULL
	`, groupID); err != nil {
		return err
	}

	if len(entries) == 0 {
		return nil
	}

	userIDs := make([]int64, len(entries))
	multipliers := make([]float64, len(entries))
	for i, entry := range entries {
		userIDs[i] = entry.UserID
		multipliers[i] = entry.Multiplier
	}

	now := time.Now()
	_, err := executor.ExecContext(ctx, `
		INSERT INTO user_group_rate_multipliers (user_id, group_id, discount_multiplier, created_at, updated_at)
		SELECT data.user_id, $1::bigint, data.discount_multiplier, $2::timestamptz, $2::timestamptz
		FROM unnest($3::bigint[], $4::double precision[]) AS data(user_id, discount_multiplier)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET discount_multiplier = EXCLUDED.discount_multiplier, updated_at = EXCLUDED.updated_at
	`, groupID, now, pq.Array(userIDs), pq.Array(multipliers))
	return err
}
