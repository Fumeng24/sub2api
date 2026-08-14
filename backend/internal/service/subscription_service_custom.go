package service

import (
	"context"
	"fmt"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrSubscriptionNotOwned         = infraerrors.Forbidden("SUBSCRIPTION_NOT_OWNED", "subscription does not belong to user")
	ErrSubscriptionInactive         = infraerrors.Conflict("SUBSCRIPTION_INACTIVE", "subscription is not active")
	ErrSubscriptionTimeInsufficient = infraerrors.BadRequest("SUBSCRIPTION_TIME_INSUFFICIENT", "subscription remaining time must be greater than 1 day")
)

func (s *SubscriptionService) resetExpiredWindowsOnExtensionCustom(ctx context.Context, sub *UserSubscription, startsAt time.Time) error {
	if sub.DailyWindowStart == nil || sub.NeedsDailyReset() {
		if err := s.userSubRepo.ResetDailyUsage(ctx, sub.ID, sub.DailyWindowStart, startsAt); err != nil {
			return fmt.Errorf("reset daily window on extend: %w", err)
		}
	}
	if sub.WeeklyWindowStart == nil || sub.NeedsWeeklyReset() {
		if err := s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, sub.WeeklyWindowStart, startsAt); err != nil {
			return fmt.Errorf("reset weekly window on extend: %w", err)
		}
	}
	if sub.MonthlyWindowStart == nil || sub.NeedsMonthlyReset() {
		if err := s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, sub.MonthlyWindowStart, startsAt); err != nil {
			return fmt.Errorf("reset monthly window on extend: %w", err)
		}
	}
	return nil
}

func activateSubscriptionWindowsOnAssignmentCustom(sub *UserSubscription, now time.Time) {
	if sub == nil {
		return
	}
	sub.DailyWindowStart = &now
	sub.WeeklyWindowStart = &now
	sub.MonthlyWindowStart = &now
}

func subscriptionRollingWindowStartCustom(now time.Time) time.Time {
	return now
}

// ResetSubscriptionWithCost resets daily usage early and deducts the skipped
// part of the current rolling window from the subscription term.
func (s *SubscriptionService) ResetSubscriptionWithCost(ctx context.Context, subscriptionID, ownerUserID int64) (*UserSubscription, error) {
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if ownerUserID != 0 && sub.UserID != ownerUserID {
		return nil, ErrSubscriptionNotOwned
	}
	if sub.Status != SubscriptionStatusActive {
		return nil, ErrSubscriptionInactive
	}

	now := time.Now()
	if sub.DailyWindowStart == nil {
		return nil, ErrSubscriptionTimeInsufficient
	}
	windowEnd := sub.DailyWindowStart.Add(24 * time.Hour)
	if !windowEnd.After(now) || !sub.ExpiresAt.After(now.Add(24*time.Hour)) {
		return nil, ErrSubscriptionTimeInsufficient
	}

	originalWindowStart := *sub.DailyWindowStart
	newExpiresAt := sub.ExpiresAt.Add(-windowEnd.Sub(now))
	updated, err := shortenSubscriptionExpiryAndResetDaily(s.userSubRepo, ctx, subscriptionID, originalWindowStart, newExpiresAt, now)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrSubscriptionTimeInsufficient
	}

	s.InvalidateSubCache(sub.UserID, sub.GroupID)
	if s.subCacheL1 != nil {
		s.subCacheL1.Wait()
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
	}
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

// SetAutoResetDaily controls automatic early reset after the daily quota is exhausted.
func (s *SubscriptionService) SetAutoResetDaily(ctx context.Context, subscriptionID, ownerUserID int64, enabled bool) (*UserSubscription, error) {
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if ownerUserID != 0 && sub.UserID != ownerUserID {
		return nil, ErrSubscriptionNotOwned
	}
	if err := updateSubscriptionAutoResetDaily(s.userSubRepo, ctx, subscriptionID, enabled); err != nil {
		return nil, err
	}

	s.InvalidateSubCache(sub.UserID, sub.GroupID)
	if s.subCacheL1 != nil {
		s.subCacheL1.Wait()
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
	}
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}
