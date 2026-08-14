//go:build unit

package middleware

import (
	"context"
	"errors"
	"time"
)

func (r *stubUserSubscriptionRepo) UpdateAutoResetDaily(ctx context.Context, subscriptionID int64, enabled bool) error {
	return errors.New("not implemented")
}

func (r *stubUserSubscriptionRepo) ShortenExpiryAndResetDaily(_ context.Context, _ int64, _ time.Time, _ time.Time, _ time.Time) (bool, error) {
	return false, nil
}
