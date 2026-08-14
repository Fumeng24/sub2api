package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// GetDiscountsByUserID returns non-NULL per-group discount multipliers.
func (r *userGroupRateRepository) GetDiscountsByUserID(ctx context.Context, userID int64) (map[int64]float64, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT group_id, discount_multiplier FROM user_group_rate_multipliers WHERE user_id = $1 AND discount_multiplier IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]float64)
	for rows.Next() {
		var groupID int64
		var discount float64
		if err := rows.Scan(&groupID, &discount); err != nil {
			return nil, err
		}
		result[groupID] = discount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetEffectiveByUserID resolves fixed rates before relative discounts.
func (r *userGroupRateRepository) GetEffectiveByUserID(ctx context.Context, userID int64) (map[int64]float64, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT ugr.group_id,
		       COALESCE(ugr.rate_multiplier, g.rate_multiplier * ugr.discount_multiplier) AS effective_rate
		FROM user_group_rate_multipliers ugr
		JOIN groups g ON g.id = ugr.group_id AND g.deleted_at IS NULL
		WHERE ugr.user_id = $1
		  AND (ugr.rate_multiplier IS NOT NULL OR ugr.discount_multiplier IS NOT NULL)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]float64)
	for rows.Next() {
		var groupID int64
		var rate float64
		if err := rows.Scan(&groupID, &rate); err != nil {
			return nil, err
		}
		result[groupID] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetDiscountsByUserIDs returns non-NULL discount multipliers for valid unique users.
func (r *userGroupRateRepository) GetDiscountsByUserIDs(ctx context.Context, userIDs []int64) (map[int64]map[int64]float64, error) {
	result := make(map[int64]map[int64]float64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	uniqueIDs := make([]int64, 0, len(userIDs))
	seen := make(map[int64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		uniqueIDs = append(uniqueIDs, userID)
		result[userID] = make(map[int64]float64)
	}
	if len(uniqueIDs) == 0 {
		return result, nil
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT user_id, group_id, discount_multiplier
		FROM user_group_rate_multipliers
		WHERE user_id = ANY($1) AND discount_multiplier IS NOT NULL
	`, pq.Array(uniqueIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var userID int64
		var groupID int64
		var discount float64
		if err := rows.Scan(&userID, &groupID, &discount); err != nil {
			return nil, err
		}
		if _, ok := result[userID]; !ok {
			result[userID] = make(map[int64]float64)
		}
		result[userID][groupID] = discount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRateConfigByUserAndGroup returns both fixed and relative rate configuration.
func (r *userGroupRateRepository) GetRateConfigByUserAndGroup(ctx context.Context, userID, groupID int64) (*service.UserGroupRateConfig, error) {
	query := `SELECT rate_multiplier, discount_multiplier FROM user_group_rate_multipliers WHERE user_id = $1 AND group_id = $2`
	var rate sql.NullFloat64
	var discount sql.NullFloat64
	err := scanSingleRow(ctx, r.sql, query, []any{userID, groupID}, &rate, &discount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !rate.Valid && !discount.Valid {
		return nil, nil
	}
	cfg := &service.UserGroupRateConfig{}
	if rate.Valid {
		v := rate.Float64
		cfg.RateMultiplier = &v
	}
	if discount.Valid {
		v := discount.Float64
		cfg.DiscountMultiplier = &v
	}
	return cfg, nil
}

// SyncUserGroupDiscounts updates discounts without changing fixed rates or RPM overrides.
func (r *userGroupRateRepository) SyncUserGroupDiscounts(ctx context.Context, userID int64, discounts map[int64]*float64) error {
	if len(discounts) == 0 {
		if _, err := r.sql.ExecContext(ctx, `
			UPDATE user_group_rate_multipliers
			SET discount_multiplier = NULL, updated_at = NOW()
			WHERE user_id = $1
		`, userID); err != nil {
			return err
		}
		_, err := r.sql.ExecContext(ctx,
			`DELETE FROM user_group_rate_multipliers WHERE user_id = $1 AND rate_multiplier IS NULL AND discount_multiplier IS NULL AND rpm_override IS NULL`,
			userID)
		return err
	}

	var clearGroupIDs []int64
	upsertGroupIDs := make([]int64, 0, len(discounts))
	upsertDiscounts := make([]float64, 0, len(discounts))
	for groupID, discount := range discounts {
		if discount == nil {
			clearGroupIDs = append(clearGroupIDs, groupID)
		} else {
			upsertGroupIDs = append(upsertGroupIDs, groupID)
			upsertDiscounts = append(upsertDiscounts, *discount)
		}
	}

	if len(clearGroupIDs) > 0 {
		if _, err := r.sql.ExecContext(ctx, `
			UPDATE user_group_rate_multipliers
			SET discount_multiplier = NULL, updated_at = NOW()
			WHERE user_id = $1 AND group_id = ANY($2)
		`, userID, pq.Array(clearGroupIDs)); err != nil {
			return err
		}
		if _, err := r.sql.ExecContext(ctx,
			`DELETE FROM user_group_rate_multipliers WHERE user_id = $1 AND group_id = ANY($2) AND rate_multiplier IS NULL AND discount_multiplier IS NULL AND rpm_override IS NULL`,
			userID, pq.Array(clearGroupIDs)); err != nil {
			return err
		}
	}

	if len(upsertGroupIDs) > 0 {
		now := time.Now()
		_, err := r.sql.ExecContext(ctx, `
			INSERT INTO user_group_rate_multipliers (user_id, group_id, discount_multiplier, created_at, updated_at)
			SELECT
				$1::bigint,
				data.group_id,
				data.discount_multiplier,
				$2::timestamptz,
				$2::timestamptz
			FROM unnest($3::bigint[], $4::double precision[]) AS data(group_id, discount_multiplier)
			ON CONFLICT (user_id, group_id)
			DO UPDATE SET
				discount_multiplier = EXCLUDED.discount_multiplier,
				updated_at = EXCLUDED.updated_at
		`, userID, now, pq.Array(upsertGroupIDs), pq.Array(upsertDiscounts))
		if err != nil {
			return err
		}
	}

	return nil
}
