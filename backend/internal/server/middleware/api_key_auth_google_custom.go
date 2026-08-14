package middleware

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func tryResetGoogleAutoDailySubscriptionCustom(
	ctx context.Context,
	subscriptionService *service.SubscriptionService,
	subscription *service.UserSubscription,
	validationErr error,
) (*service.UserSubscription, bool) {
	if subscriptionService == nil || subscription == nil ||
		!subscription.AutoResetDaily || !errors.Is(validationErr, service.ErrDailyLimitExceeded) {
		return nil, false
	}
	resetSubscription, err := subscriptionService.ResetSubscriptionWithCost(ctx, subscription.ID, 0)
	if err != nil || resetSubscription == nil {
		return nil, false
	}
	return resetSubscription, true
}
