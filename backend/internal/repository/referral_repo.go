package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type referralRepository struct {
	db *sql.DB
}

func NewReferralRepository(db *sql.DB) service.ReferralRepository {
	return &referralRepository{db: db}
}

// ===================== Profile =====================

func (r *referralRepository) CreateProfile(ctx context.Context, userID int64, referralCode string) (*service.UserReferralProfile, error) {
	q := `INSERT INTO user_referral_profiles (user_id, referral_code, created_at) VALUES ($1, $2, NOW()) RETURNING id, user_id, referral_code, created_at`
	p := &service.UserReferralProfile{}
	if err := r.db.QueryRowContext(ctx, q, userID, referralCode).Scan(&p.ID, &p.UserID, &p.ReferralCode, &p.CreatedAt); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *referralRepository) GetProfileByUserID(ctx context.Context, userID int64) (*service.UserReferralProfile, error) {
	q := `SELECT id, user_id, referral_code, created_at FROM user_referral_profiles WHERE user_id = $1`
	p := &service.UserReferralProfile{}
	if err := r.db.QueryRowContext(ctx, q, userID).Scan(&p.ID, &p.UserID, &p.ReferralCode, &p.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *referralRepository) GetProfileByCode(ctx context.Context, code string) (*service.UserReferralProfile, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	q := `SELECT id, user_id, referral_code, created_at FROM user_referral_profiles WHERE referral_code = $1`
	p := &service.UserReferralProfile{}
	if err := r.db.QueryRowContext(ctx, q, code).Scan(&p.ID, &p.UserID, &p.ReferralCode, &p.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrReferralCodeNotFound
		}
		return nil, err
	}
	return p, nil
}

// ===================== Relation =====================

func (r *referralRepository) CreateRelation(ctx context.Context, relation *service.ReferralRelation) error {
	q := `INSERT INTO referral_relations (inviter_id, invitee_id, inviter_reward, invitee_reward, reward_granted, created_at)
		VALUES ($1, $2, $3, $4, FALSE, NOW()) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q, relation.InviterID, relation.InviteeID, relation.InviterReward, relation.InviteeReward).Scan(&relation.ID, &relation.CreatedAt)
}

func (r *referralRepository) GetRelationByInviteeID(ctx context.Context, inviteeID int64) (*service.ReferralRelation, error) {
	q := `SELECT id, inviter_id, invitee_id, inviter_reward, invitee_reward, reward_granted, created_at FROM referral_relations WHERE invitee_id = $1`
	rel := &service.ReferralRelation{}
	if err := r.db.QueryRowContext(ctx, q, inviteeID).Scan(&rel.ID, &rel.InviterID, &rel.InviteeID, &rel.InviterReward, &rel.InviteeReward, &rel.RewardGranted, &rel.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rel, nil
}

func (r *referralRepository) MarkRewardGranted(ctx context.Context, id int64) error {
	q := `UPDATE referral_relations SET reward_granted = TRUE WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ===================== Query =====================

func (r *referralRepository) ListByInviterID(ctx context.Context, inviterID int64, params pagination.PaginationParams) ([]service.ReferralRelation, *pagination.PaginationResult, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM referral_relations WHERE inviter_id = $1`, inviterID).Scan(&total); err != nil {
		return nil, nil, err
	}

	q := `SELECT r.id, r.inviter_id, r.invitee_id, r.inviter_reward, r.invitee_reward, r.reward_granted, r.created_at, COALESCE(u.email, '')
		FROM referral_relations r LEFT JOIN users u ON u.id = r.invitee_id
		WHERE r.inviter_id = $1 ORDER BY r.id DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, q, inviterID, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var out []service.ReferralRelation
	for rows.Next() {
		var rel service.ReferralRelation
		if err := rows.Scan(&rel.ID, &rel.InviterID, &rel.InviteeID, &rel.InviterReward, &rel.InviteeReward, &rel.RewardGranted, &rel.CreatedAt, &rel.InviteeEmail); err != nil {
			return nil, nil, err
		}
		out = append(out, rel)
	}

	return out, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.Limit(),
	}, nil
}

func (r *referralRepository) CountByInviterID(ctx context.Context, inviterID int64) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM referral_relations WHERE inviter_id = $1`, inviterID).Scan(&n)
	return n, err
}

func (r *referralRepository) SumRewardsByInviterID(ctx context.Context, inviterID int64) (float64, error) {
	var sum float64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(inviter_reward), 0) FROM referral_relations WHERE inviter_id = $1`, inviterID).Scan(&sum)
	return sum, err
}

// ===================== Admin =====================

func (r *referralRepository) GetPlatformStats(ctx context.Context) (*service.ReferralStats, error) {
	stats := &service.ReferralStats{}
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(inviter_reward), 0), COALESCE(SUM(invitee_reward), 0) FROM referral_relations`).
		Scan(&stats.TotalRelations, &stats.TotalInviterRewardGiven, &stats.TotalInviteeRewardGiven)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
