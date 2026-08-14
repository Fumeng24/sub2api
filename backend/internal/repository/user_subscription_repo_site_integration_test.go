//go:build integration

package repository

import (
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"time"
)

func (s *UserSubscriptionRepoSuite) TestShortenExpiryAndResetDaily_Success() {
	user := s.mustCreateUser("reset-success@test.com", service.RoleUser)
	group := s.mustCreateGroup("g-reset-success")
	originalExpiry := time.Now().Add(30 * 24 * time.Hour).UTC()
	originalWindowStart := time.Now().Add(-10 * time.Hour).UTC()
	sub := s.mustCreateSubscription(user.ID, group.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetExpiresAt(originalExpiry).SetDailyUsageUsd(25.5).SetDailyWindowStart(originalWindowStart)
	})

	newExpiry := originalExpiry.Add(-14 * time.Hour)
	now := time.Now().UTC()
	updated, err := s.repo.ShortenExpiryAndResetDaily(s.ctx, sub.ID, originalWindowStart, newExpiry, now)

	s.Require().NoError(err)
	s.Require().True(updated)

	fresh, err := s.repo.GetByID(s.ctx, sub.ID)
	s.Require().NoError(err)
	s.Require().WithinDuration(newExpiry, fresh.ExpiresAt, time.Microsecond)
	s.Require().Equal(float64(0), fresh.DailyUsageUSD)
	s.Require().NotNil(fresh.DailyWindowStart)
	s.Require().WithinDuration(now, *fresh.DailyWindowStart, time.Second)
}

func (s *UserSubscriptionRepoSuite) TestShortenExpiryAndResetDaily_InsufficientTimeBoundary() {
	user := s.mustCreateUser("reset-boundary@test.com", service.RoleUser)
	group := s.mustCreateGroup("g-reset-boundary")

	now := time.Now().UTC()
	expiry := now.Add(24 * time.Hour)
	originalWindowStart := now.Add(-10 * time.Hour)
	sub := s.mustCreateSubscription(user.ID, group.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetExpiresAt(expiry).SetDailyWindowStart(originalWindowStart)
	})

	updated, err := s.repo.ShortenExpiryAndResetDaily(s.ctx, sub.ID, originalWindowStart, expiry.Add(-14*time.Hour), now)

	s.Require().NoError(err)
	s.Require().False(updated, "剩余 = 24h 时应不满足 expires_at > now + 24h 条件")
}

func (s *UserSubscriptionRepoSuite) TestShortenExpiryAndResetDaily_NonActiveSkipped() {
	user := s.mustCreateUser("reset-nonactive@test.com", service.RoleUser)
	group := s.mustCreateGroup("g-reset-nonactive")
	expiry := time.Now().Add(30 * 24 * time.Hour).UTC()
	originalWindowStart := time.Now().Add(-10 * time.Hour).UTC()
	sub := s.mustCreateSubscription(user.ID, group.ID, func(c *dbent.UserSubscriptionCreate) {
		c.SetExpiresAt(expiry).SetStatus(service.SubscriptionStatusSuspended).SetDailyWindowStart(originalWindowStart)
	})

	now := time.Now().UTC()
	updated, err := s.repo.ShortenExpiryAndResetDaily(s.ctx, sub.ID, originalWindowStart, expiry.Add(-14*time.Hour), now)

	s.Require().NoError(err)
	s.Require().False(updated, "非 active 订阅应不被更新")

	fresh, err := s.repo.GetByID(s.ctx, sub.ID)
	s.Require().NoError(err)
	s.Require().WithinDuration(expiry, fresh.ExpiresAt, time.Microsecond, "expires_at 应未被修改")
}
