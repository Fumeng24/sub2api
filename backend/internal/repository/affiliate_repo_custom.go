package repository

import (
	"context"
	"database/sql"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func hydrateAffiliateSummaryCustom(ctx context.Context, client affiliateQueryExecer, summary *service.AffiliateSummary) error {
	if summary == nil {
		return nil
	}
	rows, err := client.QueryContext(ctx, `
	SELECT ua.bind_bonus_claimed_at, u.created_at
	FROM user_affiliates ua
	JOIN users u ON u.id = ua.user_id
	WHERE ua.user_id = $1`, summary.UserID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrAffiliateProfileNotFound
	}

	var bindBonusClaimedAt sql.NullTime
	if err := rows.Scan(&bindBonusClaimedAt, &summary.UserCreatedAt); err != nil {
		return err
	}
	if bindBonusClaimedAt.Valid {
		v := bindBonusClaimedAt.Time
		summary.BindBonusClaimedAt = &v
	}
	return rows.Err()
}

func (r *affiliateRepository) ClaimBindBonus(ctx context.Context, userID int64, amount float64) (bool, float64, error) {
	if userID <= 0 {
		return false, 0, service.ErrUserNotFound
	}
	if amount <= 0 {
		return false, 0, service.ErrAffiliateBindBonusUnavailable
	}

	var claimed bool
	var newBalance float64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET bind_bonus_claimed_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1
  AND inviter_id IS NOT NULL
  AND bind_bonus_claimed_at IS NULL`, userID)
		if err != nil {
			return fmt.Errorf("mark affiliate bind bonus claimed: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return nil
		}
		if err := txClient.User.UpdateOneID(userID).AddBalance(amount).Exec(txCtx); err != nil {
			return fmt.Errorf("apply affiliate bind bonus: %w", err)
		}
		balance, err := queryUserBalance(txCtx, txClient, userID)
		if err != nil {
			return err
		}
		newBalance = balance
		claimed = true
		return nil
	})
	if err != nil {
		return false, 0, err
	}
	return claimed, newBalance, nil
}
