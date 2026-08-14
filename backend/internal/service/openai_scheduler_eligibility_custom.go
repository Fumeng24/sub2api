package service

import (
	"context"
	"time"
)

const openAIStickyEscapeCooldown = time.Minute

type openAIAccountRequestEligibility struct {
	HardReason    string
	SoftReason    string
	PausedByQuota bool
}

func (e openAIAccountRequestEligibility) Allowed() bool {
	return e.HardReason == "" && e.SoftReason == "" && !e.PausedByQuota
}

func (e openAIAccountRequestEligibility) Reason() string {
	if e.HardReason != "" {
		return e.HardReason
	}
	if e.SoftReason != "" {
		return e.SoftReason
	}
	if e.PausedByQuota {
		return "quota_auto_paused"
	}
	return ""
}

func openAIStickySessionClearReason(ctx context.Context, account *Account, platform, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability) string {
	if reason := stickySessionClearReason(account, requestedModel); reason != "" {
		return reason
	}
	if account == nil {
		return AccountSchedulingBlockInactive.String()
	}
	if account.Platform != normalizeOpenAICompatiblePlatform(platform) {
		return "platform_mismatch"
	}
	if !account.IsOpenAICompatible() {
		return "endpoint_unsupported"
	}
	return openAICompatibleAccountRequestEligibility(ctx, account, platform, requestedModel, requireCompact, requiredCapability).Reason()
}

func openAICompatibleAccountRequestEligibility(ctx context.Context, account *Account, platform, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability) openAIAccountRequestEligibility {
	platform = normalizeOpenAICompatiblePlatform(platform)
	if account == nil {
		return openAIAccountRequestEligibility{HardReason: AccountSchedulingBlockInactive.String()}
	}
	if account.Platform != platform {
		return openAIAccountRequestEligibility{HardReason: "platform_mismatch"}
	}
	if !account.IsOpenAICompatible() {
		return openAIAccountRequestEligibility{HardReason: "endpoint_unsupported"}
	}
	eligibility := openAIAccountRequestEligibilityForRequest(ctx, account, requestedModel, requireCompact, requiredCapability)
	if eligibility.Allowed() && account.IsGrok() {
		if paused, _ := shouldAutoPauseGrokAccountByQuota(account); paused {
			return openAIAccountRequestEligibility{PausedByQuota: true}
		}
	}
	return eligibility
}

func openAIAccountRequestEligibilityForRequest(ctx context.Context, account *Account, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability) openAIAccountRequestEligibility {
	if account == nil {
		return openAIAccountRequestEligibility{HardReason: AccountSchedulingBlockInactive.String()}
	}
	if !account.IsOpenAICompatible() {
		return openAIAccountRequestEligibility{HardReason: "endpoint_unsupported"}
	}
	if reason := account.HardSchedulingBlockReasonAt(time.Now()).String(); reason != "" {
		return openAIAccountRequestEligibility{HardReason: reason}
	}
	now := time.Now()
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		return openAIAccountRequestEligibility{SoftReason: AccountSchedulingBlockOverloaded.String()}
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		return openAIAccountRequestEligibility{SoftReason: AccountSchedulingBlockRateLimited.String()}
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		return openAIAccountRequestEligibility{SoftReason: AccountSchedulingBlockTempUnschedulable.String()}
	}
	if account.isModelRateLimitedWithContext(ctx, requestedModel) {
		return openAIAccountRequestEligibility{SoftReason: "model_rate_limited"}
	}
	if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
		return openAIAccountRequestEligibility{PausedByQuota: true}
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return openAIAccountRequestEligibility{HardReason: "model_unsupported"}
	}
	if !account.SupportsOpenAIEndpointCapability(requiredCapability) {
		return openAIAccountRequestEligibility{HardReason: "endpoint_unsupported"}
	}
	if requireCompact && openAICompactSupportTier(account) == 0 {
		return openAIAccountRequestEligibility{HardReason: "compact_unsupported"}
	}
	return openAIAccountRequestEligibility{}
}

func isOpenAIAccountHardEligibleForRequest(ctx context.Context, account *Account, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability) bool {
	eligibility := openAIAccountRequestEligibilityForRequest(ctx, account, requestedModel, requireCompact, requiredCapability)
	return eligibility.HardReason == "" && !eligibility.PausedByQuota
}

func (s *OpenAIGatewayService) IsSingleSchedulableAccountForRequest(ctx context.Context, groupID *int64, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability) bool {
	if s == nil || s.accountRepo == nil {
		return false
	}
	ctx = s.withOpenAIQuotaAutoPauseContext(ctx)
	if s.checkChannelPricingRestriction(ctx, groupID, requestedModel) {
		return false
	}
	accounts, err := s.listSchedulableAccounts(ctx, groupID, PlatformOpenAI)
	if err != nil || len(accounts) == 0 {
		return false
	}
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, groupID)
	count := 0
	for i := range accounts {
		account := &accounts[i]
		if !isOpenAIAccountHardEligibleForRequest(ctx, account, requestedModel, requireCompact, requiredCapability) {
			continue
		}
		if needsUpstreamCheck && groupID != nil && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact) {
			continue
		}
		count++
		if count > 1 {
			return false
		}
	}
	return count == 1
}

func (s *OpenAIGatewayService) RecordOpenAISchedulingBlockSkipped(ctx context.Context, account *Account, statusCode int, reason, source string) {
	if s == nil || s.accountRepo == nil || account == nil || account.ID <= 0 {
		return
	}
	groupID := int64(0)
	if groupIDs := schedulingProtectionGroupIDs(ctx, account); len(groupIDs) > 0 {
		groupID = groupIDs[0]
	}
	recordSchedulerBlockSkipped(ctx, s.accountRepo, account.ID, groupID, statusCode, reason, source)
}
