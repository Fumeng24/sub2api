package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	openAIAccountStateUpdateTimeout           = 5 * time.Second
	openAIOAuth429FallbackCooldown            = 5 * time.Second
	openAIRequestErrorCooldown                = 30 * time.Second
	openAITransientCooldownPersistMinInterval = 15 * time.Second
	openAIStopSchedulingBridgeCooldown        = 2 * time.Minute
	openAIOAuth429StormWindow                 = 10 * time.Second
	openAIOAuth429StormThreshold              = 20
	openAIOAuth429StormMaxAccountSwitches     = 1
)

var defaultOpenAITransientCooldownPersistThrottle = newAccountWriteThrottle(openAITransientCooldownPersistMinInterval)

func openAIAccountStateContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	return context.WithTimeout(base, openAIAccountStateUpdateTimeout)
}

func isOpenAIOAuthAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformOpenAI && account.Type == AccountTypeOAuth
}

func isOpenAIAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformOpenAI
}

func (s *OpenAIGatewayService) usesOpenAIAdvancedSchedulerPolicy(ctx context.Context, account *Account) bool {
	if s == nil ||
		account == nil ||
		account.Platform != PlatformOpenAI ||
		account.Type != AccountTypeAPIKey ||
		s.openAIAdvancedSchedulerSettingRepo() == nil {
		return false
	}
	return s.isOpenAIAdvancedSchedulerEnabled(ctx)
}

func (s *OpenAIGatewayService) shouldFailoverOpenAIUpstreamResponseForAccount(ctx context.Context, account *Account, statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if s.shouldFailoverOpenAIUpstreamResponse(statusCode, upstreamMsg, upstreamBody) {
		return true
	}
	return s.usesOpenAIAdvancedSchedulerPolicy(ctx, account) && isUpstreamModelNotFoundError(statusCode, upstreamBody)
}

func (s *OpenAIGatewayService) retryableOnSameOpenAIAccount(ctx context.Context, account *Account, statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if s.usesOpenAIAdvancedSchedulerPolicy(ctx, account) {
		return false
	}
	return account.IsPoolMode() &&
		(account.IsPoolModeRetryableStatus(statusCode) || isOpenAITransientProcessingError(statusCode, upstreamMsg, upstreamBody))
}

func (s *OpenAIGatewayService) retryableOnSameOpenAIAccountStatus(ctx context.Context, account *Account, statusCode int) bool {
	if s.usesOpenAIAdvancedSchedulerPolicy(ctx, account) {
		return false
	}
	return account.IsPoolMode() && account.IsPoolModeRetryableStatus(statusCode)
}

func (s *OpenAIGatewayService) handleOpenAIAccountUpstreamError(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, requestedModel ...string) bool {
	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()

	transientBlocked := s.markOpenAITransient5xxCoolingDown(stateCtx, account, statusCode, headers, responseBody)
	if statusCode == http.StatusTooManyRequests {
		s.markOpenAIOAuth429RateLimited(stateCtx, account, headers, responseBody)
	}
	if s == nil || account == nil || s.rateLimitService == nil {
		return transientBlocked
	}
	if s.usesOpenAIAdvancedSchedulerPolicy(ctx, account) {
		return transientBlocked || s.shouldFailoverOpenAIUpstreamResponseForAccount(ctx, account, statusCode, extractUpstreamErrorMessage(responseBody), responseBody)
	}
	if len(requestedModel) > 0 && s.rateLimitService.HandleUpstreamModelNotFound(stateCtx, account, requestedModel[0], statusCode, responseBody) {
		return true
	}
	if transientBlocked && shouldSkipLegacyOpenAITransientErrorPolicy(account, statusCode) {
		return true
	}
	shouldDisable := s.rateLimitService.HandleUpstreamError(stateCtx, account, statusCode, headers, responseBody)
	if shouldDisable {
		s.BlockAccountScheduling(account, time.Time{}, "upstream_disable")
	}
	return transientBlocked || shouldDisable
}

func (s *OpenAIGatewayService) markOpenAITransient5xxCoolingDown(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte) bool {
	if s == nil || !isOpenAIAccount(account) || !isOpenAITransient5xxStatus(statusCode) {
		return false
	}

	category := schedulerFailureCategory(statusCode, responseBody)
	cooldown := schedulerCooldownForCategory(category, headers)
	if cooldown <= 0 {
		cooldown = schedulerCooldownForCategory("transient", nil)
	}
	return s.markOpenAIAccountTemporarilyUnschedulable(ctx, account, statusCode, "openai_transient_5xx", cooldown, responseBody)
}

func isOpenAITransient5xxStatus(statusCode int) bool {
	return statusCode >= 500 && statusCode <= 599
}

func shouldSkipLegacyOpenAITransientErrorPolicy(account *Account, statusCode int) bool {
	if account == nil {
		return false
	}
	if account.IsPoolMode() && !account.IsCustomErrorCodesEnabled() {
		return true
	}
	return account.IsCustomErrorCodesEnabled() && !account.ShouldHandleErrorCode(statusCode)
}

func (s *OpenAIGatewayService) markOpenAIAccountTemporarilyUnschedulable(ctx context.Context, account *Account, statusCode int, reason string, cooldown time.Duration, responseBody []byte) bool {
	if s == nil || !isOpenAIAccount(account) || cooldown <= 0 {
		return false
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "openai_temp_unschedulable"
	}

	now := time.Now()
	until := now.Add(cooldown)

	s.BlockAccountScheduling(account, until, reason)

	state := &TempUnschedState{
		UntilUnix:       until.Unix(),
		TriggeredAtUnix: now.Unix(),
		StatusCode:      statusCode,
		MatchedKeyword:  reason,
		RuleIndex:       -1,
		ErrorMessage:    truncateTempUnschedMessage(responseBody, tempUnschedMessageMaxBytes),
	}
	if state.ErrorMessage == "" {
		state.ErrorMessage = defaultOpenAIAccountCooldownMessage(reason)
	}

	persistReason := state.ErrorMessage
	if raw, err := json.Marshal(state); err == nil {
		persistReason = string(raw)
	}

	persisted := s.getOpenAITransientCooldownPersistThrottle().Allow(account.ID, now)
	if persisted {
		if s.accountRepo != nil {
			if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, until, persistReason); err != nil {
				slog.Warn("openai_temp_unsched_failed", "account_id", account.ID, "status_code", statusCode, "reason", reason, "error", err)
			}
		}
		if s.rateLimitService != nil && s.rateLimitService.tempUnschedCache != nil {
			if err := s.rateLimitService.tempUnschedCache.SetTempUnsched(ctx, account.ID, state); err != nil {
				slog.Warn("openai_temp_unsched_cache_failed", "account_id", account.ID, "status_code", statusCode, "reason", reason, "error", err)
			}
		}
	}

	slog.Info("openai_temp_unschedulable", "account_id", account.ID, "status_code", statusCode, "reason", reason, "until", until, "persisted", persisted)
	return true
}

func defaultOpenAIAccountCooldownMessage(reason string) string {
	switch reason {
	case "openai_transient_5xx":
		return "OpenAI upstream transient 5xx"
	case "openai_request_error":
		return "OpenAI upstream request error"
	default:
		return "OpenAI account temporarily unschedulable"
	}
}

func (s *OpenAIGatewayService) markOpenAIOAuth429RateLimited(ctx context.Context, account *Account, headers http.Header, responseBody []byte) {
	if s == nil || !isOpenAIOAuthAccount(account) {
		return
	}
	s.recordOpenAIOAuth429()

	cooldownUntil := time.Now().Add(openAIOAuth429FallbackCooldown)
	if s.rateLimitService != nil {
		if resetAt := s.rateLimitService.calculateOpenAI429ResetTime(headers); resetAt != nil && resetAt.After(time.Now()) {
			cooldownUntil = *resetAt
		} else if resetUnix := parseOpenAIRateLimitResetTime(responseBody); resetUnix != nil {
			if resetAt := time.Unix(*resetUnix, 0); resetAt.After(time.Now()) {
				cooldownUntil = resetAt
			}
		} else if cooldown, ok := s.rateLimitService.get429FallbackCooldown(ctx, account); ok && cooldown > 0 {
			cooldownUntil = time.Now().Add(cooldown)
		}
	}
	s.BlockAccountScheduling(account, cooldownUntil, "429")
}

func (s *OpenAIGatewayService) BlockAccountScheduling(account *Account, until time.Time, reason string) {
	if s == nil || !isOpenAIAccount(account) {
		return
	}
	now := time.Now()
	blockUntil := until
	if blockUntil.IsZero() || !blockUntil.After(now) {
		blockUntil = now.Add(openAIStopSchedulingBridgeCooldown)
	}

	for {
		current, loaded := s.openaiAccountRuntimeBlockUntil.Load(account.ID)
		if !loaded {
			actual, stored := s.openaiAccountRuntimeBlockUntil.LoadOrStore(account.ID, blockUntil)
			if !stored {
				return
			}
			current = actual
		}

		currentUntil, ok := current.(time.Time)
		if !ok || currentUntil.IsZero() {
			if s.openaiAccountRuntimeBlockUntil.CompareAndSwap(account.ID, current, blockUntil) {
				return
			}
			continue
		}
		if currentUntil.After(blockUntil) {
			return
		}
		if s.openaiAccountRuntimeBlockUntil.CompareAndSwap(account.ID, current, blockUntil) {
			return
		}
	}
}

func (s *OpenAIGatewayService) ClearAccountSchedulingBlock(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiAccountRuntimeBlockUntil.Delete(accountID)
}

func (s *OpenAIGatewayService) isOpenAIAccountRuntimeBlocked(account *Account) bool {
	if s == nil || !isOpenAIAccount(account) {
		return false
	}
	cooldownUntil, ok := s.openAIAccountRuntimeBlockUntil(account.ID)
	if !ok {
		return false
	}
	return time.Now().Before(cooldownUntil)
}

func (s *OpenAIGatewayService) openAIAccountRuntimeBlockUntil(accountID int64) (time.Time, bool) {
	if s == nil || accountID <= 0 {
		return time.Time{}, false
	}
	value, ok := s.openaiAccountRuntimeBlockUntil.Load(accountID)
	if !ok {
		return time.Time{}, false
	}
	cooldownUntil, ok := value.(time.Time)
	if !ok || cooldownUntil.IsZero() {
		s.openaiAccountRuntimeBlockUntil.Delete(accountID)
		return time.Time{}, false
	}
	if time.Now().Before(cooldownUntil) {
		return cooldownUntil, true
	}
	s.openaiAccountRuntimeBlockUntil.Delete(accountID)
	return time.Time{}, false
}

func (s *OpenAIGatewayService) recordOpenAIOAuth429() {
	if s == nil {
		return
	}
	now := time.Now()
	windowStart := s.openaiOAuth429WindowStartUnixNano.Load()
	if windowStart == 0 || now.Sub(time.Unix(0, windowStart)) >= openAIOAuth429StormWindow {
		if s.openaiOAuth429WindowStartUnixNano.CompareAndSwap(windowStart, now.UnixNano()) {
			s.openaiOAuth429WindowCount.Store(1)
			return
		}
	}
	s.openaiOAuth429WindowCount.Add(1)
}

func (s *OpenAIGatewayService) isOpenAIOAuth429Storm() bool {
	if s == nil {
		return false
	}
	windowStart := s.openaiOAuth429WindowStartUnixNano.Load()
	if windowStart == 0 || time.Since(time.Unix(0, windowStart)) >= openAIOAuth429StormWindow {
		return false
	}
	return s.openaiOAuth429WindowCount.Load() >= openAIOAuth429StormThreshold
}

func (s *OpenAIGatewayService) ShouldStopOpenAIOAuth429Failover(account *Account, statusCode int, failedSwitches int) bool {
	if statusCode != http.StatusTooManyRequests || failedSwitches < openAIOAuth429StormMaxAccountSwitches {
		return false
	}
	if !isOpenAIOAuthAccount(account) {
		return false
	}
	return s.isOpenAIOAuth429Storm()
}
