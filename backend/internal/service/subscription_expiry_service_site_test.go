package service

import (
	"context"
	"time"
)

func (r *subscriptionExpiryRepoStub) UpdateAutoResetDaily(context.Context, int64, bool) error {
	return nil
}

func (r *subscriptionExpiryRepoStub) ShortenExpiryAndResetDaily(context.Context, int64, time.Time, time.Time, time.Time) (bool, error) {
	return false, nil
}
