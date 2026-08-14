package service

import (
	"context"
	"math"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrAffiliateBindExpired          = infraerrors.BadRequest("AFFILIATE_BIND_WINDOW_EXPIRED", "affiliate inviter can only be bound within 24 hours after registration")
	ErrAffiliateBindBonusUnavailable = infraerrors.BadRequest("AFFILIATE_BIND_BONUS_UNAVAILABLE", "affiliate bind bonus is not available")
	ErrAffiliateBindBonusClaimed     = infraerrors.Conflict("AFFILIATE_BIND_BONUS_ALREADY_CLAIMED", "affiliate bind bonus already claimed")
)

const affiliateBindWindow = 24 * time.Hour

type AffiliateSummaryCustom struct {
	BindBonusClaimedAt *time.Time `json:"bind_bonus_claimed_at,omitempty"`
	UserCreatedAt      time.Time  `json:"user_created_at"`
}

type affiliateDetailCustom struct {
	BindBonusAmount    float64    `json:"bind_bonus_amount"`
	CanBindInviter     bool       `json:"can_bind_inviter"`
	CanClaimBindBonus  bool       `json:"can_claim_bind_bonus"`
	BindBonusClaimedAt *time.Time `json:"bind_bonus_claimed_at,omitempty"`
	RebateDurationDays int        `json:"rebate_duration_days"`
}

type affiliateRepositoryCustom interface {
	ClaimBindBonus(ctx context.Context, userID int64, amount float64) (bool, float64, error)
}

func withAffiliateDetailCustom(detail *AffiliateDetail, service *AffiliateService, ctx context.Context, summary *AffiliateSummary) *AffiliateDetail {
	detail.BindBonusAmount = service.resolveBindBonusAmountForSummary(ctx, summary)
	detail.CanBindInviter = affiliateCanBindInviter(summary)
	detail.CanClaimBindBonus = service.canClaimBindBonus(ctx, summary)
	detail.BindBonusClaimedAt = summary.BindBonusClaimedAt
	detail.RebateDurationDays = service.resolveRebateDurationDays(ctx)
	return detail
}

func validateAffiliateBindCustom(summary *AffiliateSummary) error {
	if !affiliateCanBindInviter(summary) {
		return ErrAffiliateBindExpired
	}
	return nil
}

func (s *AffiliateService) ClaimBindBonus(ctx context.Context, userID int64) (float64, error) {
	if userID <= 0 {
		return 0, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if s == nil || s.repo == nil {
		return 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if !s.IsEnabled(ctx) {
		return 0, ErrAffiliateBindBonusUnavailable
	}
	summary, err := s.repo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return 0, err
	}
	if summary.InviterID == nil || *summary.InviterID <= 0 {
		return 0, ErrAffiliateBindBonusUnavailable
	}
	if summary.BindBonusClaimedAt != nil {
		return 0, ErrAffiliateBindBonusClaimed
	}
	if !affiliateBindWindowOpen(time.Now(), summary.UserCreatedAt) {
		return 0, ErrAffiliateBindExpired
	}
	amount := s.resolveBindBonusAmount(ctx)
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, ErrAffiliateBindBonusUnavailable
	}
	claimed, balance, err := s.repo.ClaimBindBonus(ctx, userID, amount)
	if err != nil {
		return 0, err
	}
	if !claimed {
		return 0, ErrAffiliateBindBonusClaimed
	}
	s.invalidateAffiliateCaches(ctx, userID)
	return balance, nil
}

func (s *AffiliateService) resolveBindBonusAmount(ctx context.Context) float64 {
	if s == nil || s.settingService == nil {
		return AffiliateBindBonusAmountDefault
	}
	return s.settingService.GetAffiliateBindBonusAmount(ctx)
}

func (s *AffiliateService) resolveBindBonusAmountForSummary(ctx context.Context, summary *AffiliateSummary) float64 {
	if !affiliateCanBindInviter(summary) && !s.canClaimBindBonus(ctx, summary) {
		return 0
	}
	return s.resolveBindBonusAmount(ctx)
}

func (s *AffiliateService) canClaimBindBonus(ctx context.Context, summary *AffiliateSummary) bool {
	if summary == nil || summary.InviterID == nil || *summary.InviterID <= 0 || summary.BindBonusClaimedAt != nil {
		return false
	}
	amount := s.resolveBindBonusAmount(ctx)
	return amount > 0 && !math.IsNaN(amount) && !math.IsInf(amount, 0) && affiliateBindWindowOpen(time.Now(), summary.UserCreatedAt)
}

func (s *AffiliateService) resolveRebateDurationDays(ctx context.Context) int {
	if s == nil || s.settingService == nil {
		return AffiliateRebateDurationDaysDefault
	}
	return s.settingService.GetAffiliateRebateDurationDays(ctx)
}

func affiliateCanBindInviter(summary *AffiliateSummary) bool {
	return summary != nil && summary.InviterID == nil && affiliateBindWindowOpen(time.Now(), summary.UserCreatedAt)
}

func affiliateBindWindowOpen(now, userCreatedAt time.Time) bool {
	return !userCreatedAt.IsZero() && !now.After(userCreatedAt.Add(affiliateBindWindow))
}
