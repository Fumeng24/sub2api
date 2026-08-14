package service

import "context"

// OpenAIAlternativeAccountRequest describes the same scheduling constraints as
// the current attempt. It is used after an upstream failure to decide whether
// pool mode should retry the current account or fail over immediately.
type OpenAIAlternativeAccountRequest struct {
	GroupID                 *int64
	Platform                string
	RequestedModel          string
	RequiredTransport       OpenAIUpstreamTransport
	RequiredCapability      OpenAIEndpointCapability
	RequiredImageCapability OpenAIImagesCapability
	RequireCompact          bool
	CurrentAccountID        int64
	ExcludedIDs             map[int64]struct{}
}

// HasEligibleOpenAIAccountAlternative checks for an untried account in the
// same scheduling group without acquiring a concurrency slot or rebinding a
// sticky session. A false result permits the existing pool-mode same-account
// retry; a true result lets the caller fail over immediately.
func (s *OpenAIGatewayService) HasEligibleOpenAIAccountAlternative(ctx context.Context, req OpenAIAlternativeAccountRequest) (bool, error) {
	if s == nil || (s.accountRepo == nil && s.schedulerSnapshot == nil) {
		return false, nil
	}
	ctx = s.withOpenAIQuotaAutoPauseContext(ctx)
	if s.checkChannelPricingRestriction(ctx, req.GroupID, req.RequestedModel) {
		return false, nil
	}
	platform := normalizeOpenAICompatiblePlatform(req.Platform)
	accounts, err := s.listSchedulableAccounts(ctx, req.GroupID, platform)
	if err != nil {
		return false, err
	}

	var schedGroup *Group
	if req.GroupID != nil && s.schedulerSnapshot != nil {
		schedGroup, _ = s.schedulerSnapshot.GetGroupByID(ctx, *req.GroupID)
	}
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, req.GroupID)
	for i := range accounts {
		candidate := &accounts[i]
		if candidate.ID <= 0 || candidate.ID == req.CurrentAccountID {
			continue
		}
		if req.ExcludedIDs != nil {
			if _, excluded := req.ExcludedIDs[candidate.ID]; excluded {
				continue
			}
		}
		if schedGroup != nil && schedGroup.RequirePrivacySet && !candidate.IsPrivacySet() {
			continue
		}

		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, candidate, platform, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil || !s.isOpenAIAccountTransportCompatible(fresh, req.RequiredTransport) ||
			!accountSupportsOpenAICapabilities(fresh, req.RequiredCapability, req.RequiredImageCapability) {
			continue
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.GroupID, platform, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil || !s.openAIAccountMatchesSchedulingGroup(fresh, req.GroupID) ||
			!s.isOpenAIAccountTransportCompatible(fresh, req.RequiredTransport) ||
			!accountSupportsOpenAICapabilities(fresh, req.RequiredCapability, req.RequiredImageCapability) {
			continue
		}
		if needsUpstreamCheck && req.GroupID != nil &&
			s.isUpstreamModelRestrictedByChannel(ctx, *req.GroupID, fresh, req.RequestedModel, req.RequireCompact) {
			continue
		}
		return true, nil
	}
	return false, nil
}
