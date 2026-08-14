package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ShortenExpiryAndResetDaily atomically charges a reset by shortening expiry.
func (r *userSubscriptionRepository) ShortenExpiryAndResetDaily(
	ctx context.Context,
	subID int64,
	originalWindowStart time.Time,
	newExpiresAt time.Time,
	now time.Time,
) (bool, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.UserSubscription.Update().
		Where(
			usersubscription.IDEQ(subID),
			usersubscription.StatusEQ(service.SubscriptionStatusActive),
			usersubscription.DailyWindowStartEQ(originalWindowStart),
			usersubscription.ExpiresAtGT(now.Add(24*time.Hour)),
		).
		SetExpiresAt(newExpiresAt).
		SetDailyWindowStart(now).
		SetDailyUsageUsd(0).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
