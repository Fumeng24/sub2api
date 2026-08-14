package service

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	poolTransient5xxThreshold     = 3
	poolTransient5xxWindowMinutes = 5

	Transient5xxCooldownReasonPrefix     = "upstream_transient_5xx"
	PoolTransient5xxCooldownReasonPrefix = "pool_transient_5xx"

	transientCooldownReasonPrefix     = Transient5xxCooldownReasonPrefix
	poolTransientCooldownReasonPrefix = PoolTransient5xxCooldownReasonPrefix
)

// handleTransient5xx records a real account-level health signal. A group may
// still use the account as its last compatible reserve, but the account itself
// is never made healthy in one group and unhealthy in another.
func (s *RateLimitService) handleTransient5xx(ctx context.Context, account *Account, statusCode int, upstreamMsg string, requestedModel ...string) bool {
	if s == nil || account == nil {
		return false
	}
	if account.IsPoolMode() {
		return s.handlePoolTransient5xx(ctx, account, statusCode, upstreamMsg, requestedModel...)
	}
	until := time.Now().Add(transient5xxCooldown(statusCode))
	return s.setTransientAccountCooldown(
		ctx,
		account,
		until,
		transientCooldownReason(transientCooldownReasonPrefix, upstreamMsg),
		statusCode,
		"transient_5xx",
		requestedModel...,
	)
}

func (s *RateLimitService) handlePoolTransient5xx(ctx context.Context, account *Account, statusCode int, upstreamMsg string, requestedModel ...string) bool {
	if s == nil || account == nil || s.accountRepo == nil {
		return false
	}

	count, incremented := s.incrementPoolTransient5xx(ctx, account.ID)
	if !incremented {
		slog.Debug("pool_transient_5xx_duplicate_request",
			"account_id", account.ID,
			"status_code", statusCode,
			"count", count,
		)
		return false
	}
	slog.Info("pool_transient_5xx_count",
		"account_id", account.ID,
		"status_code", statusCode,
		"count", count,
		"threshold", poolTransient5xxThreshold,
	)
	if count <= poolTransient5xxThreshold {
		return false
	}

	until := time.Now().Add(transient5xxCooldown(statusCode))
	applied := s.setPoolTransientTempUnschedulable(
		ctx,
		account,
		until,
		transientCooldownReason(poolTransientCooldownReasonPrefix, upstreamMsg),
		statusCode,
		requestedModel...,
	)
	if applied && s.transientErrorCounter != nil {
		if err := s.transientErrorCounter.ResetTransientErrorCount(ctx, account.ID); err != nil {
			slog.Warn("pool_transient_5xx_reset_failed", "account_id", account.ID, "error", err)
		}
	}
	return applied
}

func (s *RateLimitService) incrementPoolTransient5xx(ctx context.Context, accountID int64) (int64, bool) {
	if s == nil || s.transientErrorCounter == nil {
		return 1, true
	}
	if deduplicated, ok := s.transientErrorCounter.(TransientErrorRequestCounter); ok && deduplicated != nil {
		if requestID := transientLogicalRequestID(ctx); requestID != "" {
			count, incremented, err := deduplicated.IncrementTransientErrorCountOnce(ctx, accountID, requestID, poolTransient5xxWindowMinutes)
			if err != nil {
				slog.Warn("pool_transient_5xx_count_once_failed", "account_id", accountID, "error", err)
				return 1, true
			}
			return count, incremented
		}
	}
	count, err := s.transientErrorCounter.IncrementTransientErrorCount(ctx, accountID, poolTransient5xxWindowMinutes)
	if err != nil {
		slog.Warn("pool_transient_5xx_count_failed", "account_id", accountID, "error", err)
		return 1, true
	}
	return count, true
}

func transientLogicalRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	for _, key := range []ctxkey.Key{ctxkey.RequestID, ctxkey.ClientRequestID} {
		if requestID, ok := ctx.Value(key).(string); ok && strings.TrimSpace(requestID) != "" {
			return strings.TrimSpace(requestID)
		}
	}
	return ""
}

func transient5xxCooldown(statusCode int) time.Duration {
	if statusCode == http.StatusServiceUnavailable || statusCode == http.StatusGatewayTimeout || statusCode == 524 {
		return 10 * time.Minute
	}
	return time.Minute
}

func transientCooldownReason(prefix, upstreamMsg string) string {
	message := strings.TrimSpace(upstreamMsg)
	if message == "" {
		return prefix
	}
	return prefix + ": " + message
}

// setPoolTransientTempUnschedulable intentionally keeps its historical name
// because callers and tests use it, but it now writes a single account-level
// state. Group isolation is handled by reserve selection, not by duplicating
// cooldown state per group.
func (s *RateLimitService) setPoolTransientTempUnschedulable(ctx context.Context, account *Account, until time.Time, reason string, statusCode int, requestedModel ...string) bool {
	return s.setTransientAccountCooldown(ctx, account, until, reason, statusCode, "pool_transient_5xx", requestedModel...)
}

func (s *RateLimitService) setTransientAccountCooldown(
	ctx context.Context,
	account *Account,
	until time.Time,
	reason string,
	statusCode int,
	source string,
	requestedModel ...string,
) bool {
	if s == nil || s.accountRepo == nil || account == nil || account.ID <= 0 {
		return false
	}

	current := account
	if fresh, err := s.accountRepo.GetByID(ctx, account.ID); err == nil && fresh != nil {
		current = fresh
	}
	if current.TempUnschedulableUntil != nil && time.Now().Before(*current.TempUnschedulableUntil) {
		// Do not extend an active episode or add duplicate history. Replaying the
		// runtime block is harmless and repairs a process-local cache after restart.
		s.notifyAccountSchedulingBlocked(current, *current.TempUnschedulableUntil, source)
		s.scheduleTransientRecoveryProbe(current.ID, firstRequestedModel(requestedModel))
		return true
	}

	if err := s.accountRepo.SetTempUnschedulable(ctx, current.ID, until, reason); err != nil {
		slog.Warn("account_transient_5xx_temp_unschedulable_failed",
			"account_id", current.ID,
			"status_code", statusCode,
			"source", source,
			"error", err,
		)
		return false
	}
	s.notifyAccountSchedulingBlocked(current, until, source)
	recordSchedulerBlocked(ctx, s.accountRepo, current.ID, firstAccountGroupID(ctx, current), statusCode, reason, source, until, map[string]any{
		"block_granularity": "account",
		"cooldown_minutes":  int(time.Until(until).Round(time.Minute) / time.Minute),
	})
	s.scheduleTransientRecoveryProbe(current.ID, firstRequestedModel(requestedModel))
	return true
}
