package service

import (
	"context"
	"time"
)

func (userSubRepoNoop) UpdateAutoResetDaily(context.Context, int64, bool) error {
	panic("unexpected UpdateAutoResetDaily call")
}

func (userSubRepoNoop) ShortenExpiryAndResetDaily(context.Context, int64, time.Time, time.Time, time.Time) (bool, error) {
	panic("unexpected ShortenExpiryAndResetDaily call")
}
