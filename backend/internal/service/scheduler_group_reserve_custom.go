package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

// groupReserveDisabledKey prevents an inner legacy selector from consuming a
// reserve before its outer load-aware selector has exhausted normal candidates.
// A reserve is intentionally a final fallback, never part of the normal pool.
type groupReserveDisabledKey struct{}

func withGroupReserveDisabled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, groupReserveDisabledKey{}, true)
}

func groupReserveEnabled(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	disabled, _ := ctx.Value(groupReserveDisabledKey{}).(bool)
	return !disabled
}

func isReserveSelectionExhausted(err error) bool {
	return errors.Is(err, ErrNoAvailableAccounts) || errors.Is(err, ErrNoAvailableCompactAccounts)
}

const (
	groupReserveReasonUpstream5xx = "upstream_transient_5xx"
	groupReserveReasonPool5xx     = "pool_transient_5xx"
	groupReserveReasonOpenAIIO    = "openai_status_zero"
	groupReserveReasonMonitor     = "account_monitor_consecutive_failures"
)

// IsGroupReserveEligibleAt reports whether an actively cooled account may be
// used as the final compatible candidate in one of its own groups. Persistent
// stops and request-specific restrictions remain absolute: a reserve never
// bypasses authentication, balance/quota, rate-limit, overload, model-limit,
// expiry, or manual scheduling controls.
func (a *Account) IsGroupReserveEligibleAt(now time.Time) bool {
	if a == nil || a.TempUnschedulableUntil == nil || !now.Before(*a.TempUnschedulableUntil) {
		return false
	}
	if a.HardSchedulingBlockReasonAt(now) != AccountSchedulingBlockNone {
		return false
	}
	if a.OverloadUntil != nil && now.Before(*a.OverloadUntil) {
		return false
	}
	if a.RateLimitResetAt != nil && now.Before(*a.RateLimitResetAt) {
		return false
	}
	if a.IsAPIKeyOrBedrock() && a.IsQuotaExceeded() {
		return false
	}
	return isGroupReserveCooldownReason(a.TempUnschedulableReason)
}

func isGroupReserveCooldownReason(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	for _, prefix := range []string{
		groupReserveReasonUpstream5xx,
		groupReserveReasonPool5xx,
		groupReserveReasonOpenAIIO,
		groupReserveReasonMonitor,
	} {
		if reason == prefix || strings.HasPrefix(reason, prefix+":") {
			return true
		}
	}
	return false
}

func accountCanServeModelAsGroupReserve(ctx context.Context, account *Account, requestedModel string) bool {
	if account == nil || !account.IsGroupReserveEligibleAt(time.Now()) {
		return false
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return false
	}
	if !account.isModelRateLimitedWithContext(ctx, requestedModel) {
		return true
	}
	// Antigravity overages are the one existing exception to a model cooldown.
	return account.Platform == PlatformAntigravity && account.IsOveragesEnabled() && !account.isCreditsExhausted()
}

// isOpenAIGroupReserveRuntimeBlocked keeps request-specific runtime circuits
// absolute, but does not let the in-memory mirror of the same account-level
// transient cooldown defeat the final-candidate policy. Persistent eligibility
// above already rejects manual/auth/balance/expiry/429/overload states.
func (s *OpenAIGatewayService) isOpenAIGroupReserveRuntimeBlocked(account *Account, requestedModel string) bool {
	if s == nil || account == nil {
		return false
	}
	if s.isOpenAIAccountModelRuntimeBlocked(account, requestedModel) {
		return true
	}
	if !s.isOpenAIAccountRuntimeBlocked(account) {
		return false
	}
	return !account.IsGroupReserveEligibleAt(time.Now())
}

// listGroupReserveCandidates reads the persistent group membership directly,
// rather than the schedulable snapshot. This is necessary because a candidate
// is deliberately in account-level cooldown and therefore absent from the
// normal snapshot. The repository query still enforces active + schedulable.
func listGroupReserveCandidates(
	ctx context.Context,
	repo AccountRepository,
	groupID *int64,
	platforms []string,
	excludedIDs map[int64]struct{},
) ([]Account, error) {
	if repo == nil || groupID == nil || *groupID <= 0 || len(platforms) == 0 {
		return nil, nil
	}
	accounts, err := repo.ListModelAvailabilityCandidates(ctx, groupID, platforms, false)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	filtered := make([]Account, 0, len(accounts))
	for i := range accounts {
		account := accounts[i]
		if _, excluded := excludedIDs[account.ID]; excluded {
			continue
		}
		if !account.IsGroupReserveEligibleAt(now) {
			continue
		}
		filtered = append(filtered, account)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left, right := filtered[i], filtered[j]
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		switch {
		case left.LastUsedAt == nil && right.LastUsedAt != nil:
			return true
		case left.LastUsedAt != nil && right.LastUsedAt == nil:
			return false
		case left.LastUsedAt != nil && right.LastUsedAt != nil && !left.LastUsedAt.Equal(*right.LastUsedAt):
			return left.LastUsedAt.Before(*right.LastUsedAt)
		default:
			return left.ID < right.ID
		}
	})
	return filtered, nil
}

func (s *GatewayService) selectGroupReserveGatewayAccount(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	platform string,
	useMixed bool,
	excludedIDs map[int64]struct{},
) (*Account, error) {
	platforms := []string{platform}
	if useMixed {
		platforms = append(platforms, PlatformAntigravity)
	}
	candidates, err := listGroupReserveCandidates(ctx, s.accountRepo, groupID, platforms, excludedIDs)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, ErrNoAvailableAccounts
	}

	ctx = s.withWindowCostPrefetch(ctx, candidates)
	ctx = s.withRPMPrefetch(ctx, candidates)
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, groupID)
	routingIDs := s.routingAccountIDsForRequest(ctx, groupID, requestedModel, platform)
	routingSet := make(map[int64]struct{}, len(routingIDs))
	for _, id := range routingIDs {
		if id > 0 {
			routingSet[id] = struct{}{}
		}
	}

	var group *Group
	if groupID != nil && s.groupRepo != nil {
		group, _ = s.groupRepo.GetByIDLite(ctx, *groupID)
	}
	for i := range candidates {
		account := &candidates[i]
		if !s.isAccountAllowedForPlatform(account, platform, useMixed) {
			continue
		}
		if len(routingSet) > 0 {
			if _, routed := routingSet[account.ID]; !routed {
				continue
			}
		}
		if group != nil && group.RequirePrivacySet && !account.IsPrivacySet() {
			continue
		}
		if !accountCanServeModelAsGroupReserve(ctx, account, requestedModel) {
			continue
		}
		if !s.isAccountSchedulableForQuota(account) ||
			!s.isAccountSchedulableForWindowCost(ctx, account, false) ||
			!s.isAccountSchedulableForRPM(ctx, account, false) {
			continue
		}
		if needsUpstreamCheck && groupID != nil && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel) {
			continue
		}
		return s.hydrateSelectedAccount(ctx, account)
	}
	return nil, ErrNoAvailableAccounts
}

func (s *GatewayService) selectGroupReserveGatewayWithLoadAwareness(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	requestedModel string,
	platform string,
	useMixed bool,
	excludedIDs map[int64]struct{},
) (*AccountSelectionResult, error) {
	account, err := s.selectGroupReserveGatewayAccount(ctx, groupID, requestedModel, platform, useMixed, excludedIDs)
	if err != nil {
		return nil, err
	}
	if !s.checkAndRegisterSession(ctx, account, sessionHash) {
		return nil, ErrNoAvailableAccounts
	}
	if result, acquireErr := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency); acquireErr == nil && result != nil && result.Acquired {
		selection, selectErr := s.newSelectionResult(ctx, account, true, result.ReleaseFunc, nil)
		if selectErr != nil {
			if result.ReleaseFunc != nil {
				result.ReleaseFunc()
			}
			return nil, selectErr
		}
		selection.GroupReserve = true
		selection.GroupReserveReason = account.TempUnschedulableReason
		if sessionHash != "" && s.cache != nil {
			_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, account.ID, stickySessionTTL)
		}
		return selection, nil
	}

	cfg := s.schedulingConfig()
	selection, err := s.newSelectionResult(ctx, account, false, nil, &AccountWaitPlan{
		AccountID:      account.ID,
		MaxConcurrency: account.Concurrency,
		Timeout:        cfg.FallbackWaitTimeout,
		MaxWaiting:     cfg.FallbackMaxWaiting,
	})
	if err != nil {
		return nil, err
	}
	selection.GroupReserve = true
	selection.GroupReserveReason = account.TempUnschedulableReason
	if sessionHash != "" && s.cache != nil {
		_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, account.ID, stickySessionTTL)
	}
	return selection, nil
}

func (s *OpenAIGatewayService) selectGroupReserveOpenAIAccount(
	ctx context.Context,
	groupID *int64,
	platform string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requiredCapability OpenAIEndpointCapability,
	requiredImageCapability OpenAIImagesCapability,
	requireCompact bool,
) (*Account, error) {
	platform = normalizeOpenAICompatiblePlatform(platform)
	if s == nil || s.accountRepo == nil || groupID == nil || *groupID <= 0 {
		return nil, ErrNoAvailableAccounts
	}
	if s.checkChannelPricingRestriction(ctx, groupID, requestedModel) {
		return nil, ErrNoAvailableAccounts
	}
	candidates, err := listGroupReserveCandidates(ctx, s.accountRepo, groupID, []string{platform}, excludedIDs)
	if err != nil {
		return nil, err
	}
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, groupID)
	var group *Group
	if s.schedulerSnapshot != nil {
		group, _ = s.schedulerSnapshot.GetGroupByID(ctx, *groupID)
	}
	for i := range candidates {
		account := &candidates[i]
		if account.Platform != platform || !account.IsOpenAICompatible() ||
			!accountCanServeModelAsGroupReserve(ctx, account, requestedModel) {
			continue
		}
		if account.IsOpenAI() {
			if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
				continue
			}
		}
		if account.IsGrok() {
			if paused, _ := shouldAutoPauseGrokAccountByQuota(account); paused {
				continue
			}
		}
		if group != nil && group.RequirePrivacySet && !account.IsPrivacySet() {
			continue
		}
		if !accountSupportsOpenAICapabilities(account, requiredCapability, requiredImageCapability) ||
			(requireCompact && openAICompactSupportTier(account) == 0) ||
			!s.isOpenAIAccountTransportCompatible(account, requiredTransport) ||
			s.isOpenAIGroupReserveRuntimeBlocked(account, requestedModel) ||
			s.isOpenAIProxyStreamQuarantined(ctx, account) {
			continue
		}
		if !parentHealthyForShadow(account, s.parentAccountLookup(ctx)) {
			continue
		}
		if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact) {
			continue
		}
		return s.hydrateSelectedAccount(ctx, account)
	}
	return nil, ErrNoAvailableAccounts
}

func (s *OpenAIGatewayService) selectGroupReserveOpenAIWithLoadAwareness(
	ctx context.Context,
	groupID *int64,
	platform string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requiredCapability OpenAIEndpointCapability,
	requiredImageCapability OpenAIImagesCapability,
	requireCompact bool,
) (*AccountSelectionResult, error) {
	account, err := s.selectGroupReserveOpenAIAccount(ctx, groupID, platform, requestedModel, excludedIDs, requiredTransport, requiredCapability, requiredImageCapability, requireCompact)
	if err != nil {
		return nil, err
	}
	if result, acquireErr := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency); acquireErr == nil && result != nil && result.Acquired {
		selection, selectErr := s.newAcquiredSelectionResult(ctx, account, result.ReleaseFunc)
		if selectErr != nil {
			return nil, selectErr
		}
		selection.GroupReserve = true
		selection.GroupReserveReason = account.TempUnschedulableReason
		if sessionHash != "" {
			_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, account.ID, openaiStickySessionTTL)
		}
		return selection, nil
	}

	cfg := s.schedulingConfig()
	selection, err := s.newSelectionResult(ctx, account, false, nil, &AccountWaitPlan{
		AccountID:      account.ID,
		MaxConcurrency: account.Concurrency,
		Timeout:        cfg.FallbackWaitTimeout,
		MaxWaiting:     cfg.FallbackMaxWaiting,
	})
	if err != nil {
		return nil, err
	}
	selection.GroupReserve = true
	selection.GroupReserveReason = account.TempUnschedulableReason
	if sessionHash != "" {
		_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, account.ID, openaiStickySessionTTL)
	}
	return selection, nil
}

func (s *GeminiMessagesCompatService) selectGroupReserveGeminiAccount(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	excludedIDs map[int64]struct{},
	platform string,
	useMixedScheduling bool,
) *Account {
	if s == nil || s.accountRepo == nil || groupID == nil || *groupID <= 0 {
		return nil
	}
	platforms := []string{platform}
	if useMixedScheduling {
		platforms = append(platforms, PlatformAntigravity)
	}
	candidates, err := listGroupReserveCandidates(ctx, s.accountRepo, groupID, platforms, excludedIDs)
	if err != nil {
		return nil
	}
	for i := range candidates {
		account := &candidates[i]
		if !s.isAccountValidForPlatform(account, platform, useMixedScheduling) ||
			!accountCanServeModelAsGroupReserve(ctx, account, requestedModel) {
			continue
		}
		if requestedModel != "" && !s.isModelSupportedByAccount(account, requestedModel) {
			continue
		}
		if !s.passesRateLimitPreCheckWithCache(ctx, account, requestedModel, nil) {
			continue
		}
		hydrated, hydrateErr := s.hydrateSelectedAccount(ctx, account)
		if hydrateErr != nil {
			continue
		}
		return hydrated
	}
	return nil
}
