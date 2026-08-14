package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// TransientRecoveryProbeScheduler is intentionally narrow so RateLimitService
// can request recovery without owning worker lifecycle or probe transport.
type TransientRecoveryProbeScheduler interface {
	ScheduleTransientRecoveryProbe(accountID int64, requestedModel string)
}

// transient5xxCooldownRepository is an optional repository capability. Keeping
// it outside AccountRepository avoids making every gateway test double expose a
// deployment-specific conditional mutation.
type transient5xxCooldownRepository interface {
	ClearTransient5xxCooldown(ctx context.Context, id int64) (bool, error)
	RenewTransient5xxCooldown(ctx context.Context, id int64, until time.Time) (bool, error)
}

func (s *RateLimitService) scheduleTransientRecoveryProbe(accountID int64, requestedModel string) {
	if s == nil || s.transientRecovery == nil || accountID <= 0 {
		return
	}
	s.transientRecovery.ScheduleTransientRecoveryProbe(accountID, requestedModel)
}

// ClearTransient5xxCooldown clears only a currently-owned transient 5xx
// cooldown. It deliberately does not call ClearRateLimit: a successful health
// probe must not erase a concurrent 429, overload, model restriction, or
// administrator action.
func (s *RateLimitService) ClearTransient5xxCooldown(ctx context.Context, accountID int64) (bool, error) {
	if s == nil || s.accountRepo == nil || accountID <= 0 {
		return false, nil
	}
	repo, ok := s.accountRepo.(transient5xxCooldownRepository)
	if !ok {
		return false, nil
	}

	cleared, err := repo.ClearTransient5xxCooldown(ctx, accountID)
	if err != nil || !cleared {
		return cleared, err
	}
	if s.tempUnschedCache != nil {
		if err := s.tempUnschedCache.DeleteTempUnsched(ctx, accountID); err != nil {
			slog.Warn("transient_5xx_recovery_cache_delete_failed", "account_id", accountID, "error", err)
		}
	}
	s.reconcileRuntimeSchedulingBlock(ctx, accountID)
	return true, nil
}

// RenewTransient5xxCooldown holds an account out of the scheduler while an
// active recovery probe keeps failing. The database condition protects manual
// changes and a different later runtime state from being overwritten.
func (s *RateLimitService) RenewTransient5xxCooldown(ctx context.Context, accountID int64, until time.Time) (bool, error) {
	if s == nil || s.accountRepo == nil || accountID <= 0 || until.IsZero() {
		return false, nil
	}
	repo, ok := s.accountRepo.(transient5xxCooldownRepository)
	if !ok {
		return false, nil
	}

	updated, err := repo.RenewTransient5xxCooldown(ctx, accountID, until)
	if err != nil {
		return false, err
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return false, err
	}
	if !isCurrentTransient5xxCooldown(account) {
		return false, nil
	}

	if updated && s.tempUnschedCache != nil {
		effectiveUntil := until
		if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(effectiveUntil) {
			effectiveUntil = *account.TempUnschedulableUntil
		}
		state := &TempUnschedState{
			UntilUnix:    effectiveUntil.Unix(),
			ErrorMessage: TempUnschedulableDisplayReasonFromRaw(account.TempUnschedulableReason),
		}
		if err := s.tempUnschedCache.SetTempUnsched(ctx, accountID, state); err != nil {
			slog.Warn("transient_5xx_recovery_cache_renew_failed", "account_id", accountID, "error", err)
		}
	}
	if account.TempUnschedulableUntil != nil {
		s.notifyAccountSchedulingBlocked(account, *account.TempUnschedulableUntil, "transient_recovery_probe")
	}
	return true, nil
}

func isTransient5xxCooldownReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	return strings.HasPrefix(reason, transientCooldownReasonPrefix) ||
		strings.HasPrefix(reason, poolTransientCooldownReasonPrefix)
}

func isCurrentTransient5xxCooldown(account *Account) bool {
	return account != nil &&
		account.Status == StatusActive &&
		account.Schedulable &&
		account.TempUnschedulableUntil != nil &&
		isTransient5xxCooldownReason(account.TempUnschedulableReason)
}

// reconcileRuntimeSchedulingBlock makes the OpenAI fast-path reflect the
// persistent state after a conditional clear. A new 429/overload written while
// the probe was in flight remains blocked instead of being accidentally opened.
func (s *RateLimitService) reconcileRuntimeSchedulingBlock(ctx context.Context, accountID int64) {
	if s == nil || s.accountRepo == nil || accountID <= 0 {
		return
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return
	}
	if until, reason, ok := persistentAccountSchedulingBlock(account, time.Now()); ok {
		s.notifyAccountSchedulingBlocked(account, until, reason)
		return
	}
	s.notifyAccountSchedulingBlockCleared(accountID)
}

func persistentAccountSchedulingBlock(account *Account, now time.Time) (time.Time, string, bool) {
	if account == nil {
		return time.Time{}, "", false
	}
	if account.Status != StatusActive || !account.Schedulable {
		return time.Time{}, "account_not_schedulable", true
	}

	var latest time.Time
	reason := ""
	consider := func(until *time.Time, value string) {
		if until != nil && until.After(now) && until.After(latest) {
			latest = *until
			reason = value
		}
	}
	consider(account.TempUnschedulableUntil, "temp_unschedulable")
	consider(account.RateLimitResetAt, "rate_limited")
	consider(account.OverloadUntil, "overloaded")
	if latest.IsZero() {
		return time.Time{}, "", false
	}
	return latest, reason, true
}
