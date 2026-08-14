package service

import (
	"context"
	"fmt"
	"time"
)

type userSubscriptionAutoResetUpdater interface {
	UpdateAutoResetDaily(ctx context.Context, subscriptionID int64, enabled bool) error
}

type userSubscriptionExpiryResetter interface {
	ShortenExpiryAndResetDaily(ctx context.Context, subscriptionID int64, originalWindowStart time.Time, newExpiresAt time.Time, now time.Time) (bool, error)
}

func updateSubscriptionAutoResetDaily(repo UserSubscriptionRepository, ctx context.Context, subscriptionID int64, enabled bool) error {
	updater, ok := repo.(userSubscriptionAutoResetUpdater)
	if !ok {
		return fmt.Errorf("subscription repository does not support auto-reset updates")
	}
	return updater.UpdateAutoResetDaily(ctx, subscriptionID, enabled)
}

func shortenSubscriptionExpiryAndResetDaily(repo UserSubscriptionRepository, ctx context.Context, subscriptionID int64, originalWindowStart, newExpiresAt, now time.Time) (bool, error) {
	resetter, ok := repo.(userSubscriptionExpiryResetter)
	if !ok {
		return false, fmt.Errorf("subscription repository does not support atomic daily reset")
	}
	return resetter.ShortenExpiryAndResetDaily(ctx, subscriptionID, originalWindowStart, newExpiresAt, now)
}
