package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GatewayNoAvailableAccountsError carries structured diagnostics while keeping
// errors.Is(err, ErrNoAvailableAccounts) compatible with upstream callers.
type GatewayNoAvailableAccountsError struct {
	Cause       error
	Diagnostics GatewaySelectionDiagnostics
}

func (s *GatewayService) diagnoseSelectionFailureCustom(
	ctx context.Context,
	acc *Account,
	requestedModel, platform string,
	excludedIDs map[int64]struct{},
	allowMixedScheduling bool,
) (selectionFailureDiagnosis, bool) {
	if acc == nil {
		return selectionFailureDiagnosis{Category: "unschedulable", Detail: "account_nil"}, true
	}
	if _, excluded := excludedIDs[acc.ID]; excluded {
		return selectionFailureDiagnosis{Category: "excluded"}, true
	}
	if reason := acc.HardSchedulingBlockReasonAt(time.Now()).String(); reason != "" {
		return selectionFailureDiagnosis{Category: "unschedulable", Detail: "generic_unschedulable"}, true
	}
	if reason := s.gatewaySelectionReason(ctx, acc, nil, requestedModel, platform, allowMixedScheduling, nil, false, false); reason != "" {
		switch reason {
		case "endpoint_unsupported":
			return selectionFailureDiagnosis{Category: "platform_filtered", Detail: fmt.Sprintf("account_platform=%s requested_platform=%s", acc.Platform, strings.TrimSpace(platform))}, true
		case "model_unsupported":
			return selectionFailureDiagnosis{Category: "model_unsupported", Detail: fmt.Sprintf("model=%s", requestedModel)}, true
		case "model_rate_limited":
			remaining := acc.GetRateLimitRemainingTimeWithContext(ctx, requestedModel).Truncate(time.Second)
			return selectionFailureDiagnosis{Category: "model_rate_limited", Detail: fmt.Sprintf("remaining=%s", remaining)}, true
		default:
			return selectionFailureDiagnosis{Category: "unschedulable", Detail: reason}, true
		}
	}
	return selectionFailureDiagnosis{Category: "eligible"}, true
}

func (e *GatewayNoAvailableAccountsError) Error() string {
	if e == nil {
		return ErrNoAvailableAccounts.Error()
	}
	cause := e.Cause
	if cause == nil {
		cause = ErrNoAvailableAccounts
	}
	if e.Diagnostics.Collected {
		return fmt.Sprintf("%s (%s)", cause.Error(), e.Diagnostics.Summary())
	}
	return cause.Error()
}

func (e *GatewayNoAvailableAccountsError) Unwrap() error {
	if e == nil || e.Cause == nil {
		return ErrNoAvailableAccounts
	}
	return e.Cause
}

// GatewaySelectionSkippedAccount is the bounded per-account detail attached to
// a no-available-account error for scheduler observability.
type GatewaySelectionSkippedAccount struct {
	AccountID                int64      `json:"account_id"`
	Reason                   string     `json:"reason"`
	CircuitState             string     `json:"circuit_state,omitempty"`
	CircuitReason            string     `json:"circuit_reason,omitempty"`
	CircuitModel             string     `json:"circuit_model,omitempty"`
	CircuitEndpoint          string     `json:"circuit_endpoint,omitempty"`
	CircuitRetryAt           *time.Time `json:"circuit_retry_at,omitempty"`
	CircuitRetryRemainingSec *int64     `json:"circuit_retry_remaining_sec,omitempty"`
	LoadRate                 int        `json:"load_rate,omitempty"`
	CurrentConcurrency       int        `json:"current_concurrency,omitempty"`
	MaxConcurrency           int        `json:"max_concurrency,omitempty"`
}

type GatewaySelectionDiagnostics struct {
	Collected                     bool
	GroupID                       int64
	Model                         string
	Endpoint                      string
	Platform                      string
	GroupBindingAccountCount      int
	ActiveSchedulableCount        int
	ExcludedAccountCount          int
	AfterExcludedCount            int
	ModelSupportedCount           int
	EndpointSupportedCount        int
	StateAllowedCount             int
	CircuitAllowedCount           int
	ConcurrencySlotAllowedCount   int
	FinalCandidateCount           int
	ExcludedAccountIDs            []int64
	ActiveSchedulableAccountIDs   []int64
	AfterExcludedAccountIDs       []int64
	ModelSupportedAccountIDs      []int64
	EndpointSupportedAccountIDs   []int64
	StateAllowedAccountIDs        []int64
	CircuitAllowedAccountIDs      []int64
	CandidateAccountIDs           []int64
	ModelUnsupportedAccountIDs    []int64
	EndpointUnsupportedAccountIDs []int64
	StateFilteredAccountIDs       []int64
	CircuitFilteredAccountIDs     []int64
	ChannelRestrictionAccountIDs  []int64
	ConcurrencyFullAccountIDs     []int64
	SkippedAccounts               []GatewaySelectionSkippedAccount
	FilterReasonCounts            map[string]int
}

func (d GatewaySelectionDiagnostics) Summary() string {
	return fmt.Sprintf(
		"group_id=%d platform=%s model=%s endpoint=%s group_binding_count=%d active_schedulable_count=%d excluded_count=%d after_excluded_count=%d model_supported_count=%d endpoint_supported_count=%d state_allowed_count=%d circuit_allowed_count=%d concurrency_slot_allowed_count=%d final_candidate_count=%d skip_reason=%v",
		d.GroupID,
		d.Platform,
		d.Model,
		d.Endpoint,
		d.GroupBindingAccountCount,
		d.ActiveSchedulableCount,
		d.ExcludedAccountCount,
		d.AfterExcludedCount,
		d.ModelSupportedCount,
		d.EndpointSupportedCount,
		d.StateAllowedCount,
		d.CircuitAllowedCount,
		d.ConcurrencySlotAllowedCount,
		d.FinalCandidateCount,
		d.FilterReasonCounts,
	)
}

func (d *GatewaySelectionDiagnostics) addReason(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	if d.FilterReasonCounts == nil {
		d.FilterReasonCounts = make(map[string]int)
	}
	d.FilterReasonCounts[reason]++
}

func (d *GatewaySelectionDiagnostics) addSkipped(account *Account, reason string, loadInfo *AccountLoadInfo) {
	if account == nil {
		d.addReason(reason)
		return
	}
	d.addReason(reason)
	skipped := GatewaySelectionSkippedAccount{
		AccountID:      account.ID,
		Reason:         strings.TrimSpace(reason),
		MaxConcurrency: account.Concurrency,
	}
	if loadInfo != nil {
		skipped.LoadRate = loadInfo.LoadRate
		skipped.CurrentConcurrency = loadInfo.CurrentConcurrency
	}
	d.SkippedAccounts = appendGatewaySelectionSkippedAccount(d.SkippedAccounts, skipped)
}

func appendGatewaySelectionID(ids []int64, account *Account) []int64 {
	if account == nil {
		return ids
	}
	return append(ids, account.ID)
}

func appendGatewaySelectionSkippedAccount(items []GatewaySelectionSkippedAccount, item GatewaySelectionSkippedAccount) []GatewaySelectionSkippedAccount {
	const limit = 20
	if len(items) >= limit {
		return items
	}
	return append(items, item)
}

func (s *GatewayService) newGatewayNoAvailableError(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	platform string,
	endpoint string,
	accounts []*Account,
	excludedIDs map[int64]struct{},
	useMixed bool,
	schedGroup *Group,
	needsUpstreamCheck bool,
	loadMap map[int64]*AccountLoadInfo,
) error {
	cause := error(ErrNoAvailableAccounts)
	if strings.TrimSpace(requestedModel) != "" {
		cause = fmt.Errorf("%w supporting model: %s", ErrNoAvailableAccounts, strings.TrimSpace(requestedModel))
	}
	return &GatewayNoAvailableAccountsError{
		Cause: cause,
		Diagnostics: s.buildGatewaySelectionDiagnostics(
			ctx, groupID, requestedModel, platform, endpoint, accounts, excludedIDs,
			useMixed, schedGroup, needsUpstreamCheck, loadMap,
		),
	}
}

func (s *GatewayService) buildGatewaySelectionDiagnostics(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	platform string,
	endpoint string,
	accounts []*Account,
	excludedIDs map[int64]struct{},
	useMixed bool,
	schedGroup *Group,
	needsUpstreamCheck bool,
	loadMap map[int64]*AccountLoadInfo,
) GatewaySelectionDiagnostics {
	diag := GatewaySelectionDiagnostics{
		Collected:                true,
		GroupID:                  derefGroupID(groupID),
		Model:                    strings.TrimSpace(requestedModel),
		Endpoint:                 strings.TrimSpace(endpoint),
		Platform:                 strings.TrimSpace(platform),
		GroupBindingAccountCount: len(accounts),
		ExcludedAccountCount:     len(excludedIDs),
		ExcludedAccountIDs:       gatewaySelectionIDsFromMap(excludedIDs),
		FilterReasonCounts:       make(map[string]int),
	}

	circuitAllowed := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			diag.addReason("nil_account")
			continue
		}
		if account.Status == StatusActive && account.Schedulable {
			diag.ActiveSchedulableCount++
			diag.ActiveSchedulableAccountIDs = appendGatewaySelectionID(diag.ActiveSchedulableAccountIDs, account)
		}
		if _, excluded := excludedIDs[account.ID]; excluded {
			diag.addSkipped(account, "excluded", gatewaySelectionLoadInfo(loadMap, account))
			continue
		}
		diag.AfterExcludedCount++
		diag.AfterExcludedAccountIDs = appendGatewaySelectionID(diag.AfterExcludedAccountIDs, account)

		modelSupported := requestedModel == "" || s.isModelSupportedByAccountWithContext(ctx, account, requestedModel)
		if modelSupported {
			diag.ModelSupportedCount++
			diag.ModelSupportedAccountIDs = appendGatewaySelectionID(diag.ModelSupportedAccountIDs, account)
		} else {
			diag.ModelUnsupportedAccountIDs = appendGatewaySelectionID(diag.ModelUnsupportedAccountIDs, account)
		}

		endpointSupported := s.isAccountAllowedForPlatform(account, platform, useMixed)
		if endpointSupported {
			diag.EndpointSupportedCount++
			diag.EndpointSupportedAccountIDs = appendGatewaySelectionID(diag.EndpointSupportedAccountIDs, account)
		} else {
			diag.EndpointUnsupportedAccountIDs = appendGatewaySelectionID(diag.EndpointUnsupportedAccountIDs, account)
		}

		if reason := s.gatewaySelectionReason(ctx, account, groupID, requestedModel, platform, useMixed, schedGroup, needsUpstreamCheck, false); reason != "" {
			diag.StateFilteredAccountIDs = appendGatewaySelectionID(diag.StateFilteredAccountIDs, account)
			if reason == "channel_pricing_restricted" {
				diag.ChannelRestrictionAccountIDs = appendGatewaySelectionID(diag.ChannelRestrictionAccountIDs, account)
			}
			diag.addSkipped(account, reason, gatewaySelectionLoadInfo(loadMap, account))
			continue
		}
		diag.StateAllowedCount++
		diag.StateAllowedAccountIDs = appendGatewaySelectionID(diag.StateAllowedAccountIDs, account)
		diag.CircuitAllowedCount++
		diag.CircuitAllowedAccountIDs = appendGatewaySelectionID(diag.CircuitAllowedAccountIDs, account)
		circuitAllowed = append(circuitAllowed, account)
	}

	for _, account := range circuitAllowed {
		loadInfo := gatewaySelectionLoadInfo(loadMap, account)
		if loadInfo != nil && loadInfo.LoadRate >= 100 {
			diag.ConcurrencyFullAccountIDs = appendGatewaySelectionID(diag.ConcurrencyFullAccountIDs, account)
			diag.addSkipped(account, "concurrency_full", loadInfo)
			continue
		}
		diag.ConcurrencySlotAllowedCount++
		diag.CandidateAccountIDs = appendGatewaySelectionID(diag.CandidateAccountIDs, account)
	}
	diag.FinalCandidateCount = len(diag.CandidateAccountIDs)
	return diag
}

func (s *GatewayService) gatewaySelectionReason(
	ctx context.Context,
	account *Account,
	groupID *int64,
	requestedModel string,
	platform string,
	useMixed bool,
	schedGroup *Group,
	needsUpstreamCheck bool,
	isSticky bool,
) string {
	if account == nil {
		return "nil_account"
	}
	if reason := account.SchedulingBlockReasonAt(time.Now()).String(); reason != "" {
		return reason
	}
	if !s.isAccountAllowedForPlatform(account, platform, useMixed) {
		return "endpoint_unsupported"
	}
	if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
		return "privacy_not_set"
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
		return "model_unsupported"
	}
	if needsUpstreamCheck && groupID != nil && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel) {
		return "channel_pricing_restricted"
	}
	if !s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) {
		return "model_rate_limited"
	}
	if !s.isAccountSchedulableForQuota(account) {
		return "quota_exceeded"
	}
	if !s.isAccountSchedulableForWindowCost(ctx, account, isSticky) {
		return "window_cost_limited"
	}
	if !s.isAccountSchedulableForRPM(ctx, account, isSticky) {
		return "rpm_limited"
	}
	return ""
}

func (s *GatewayService) gatewayHardSelectionReason(
	ctx context.Context,
	account *Account,
	groupID *int64,
	requestedModel string,
	platform string,
	useMixed bool,
	schedGroup *Group,
	needsUpstreamCheck bool,
) string {
	if account == nil {
		return "nil_account"
	}
	if reason := account.HardSchedulingBlockReasonAt(time.Now()).String(); reason != "" {
		return reason
	}
	if !s.isAccountAllowedForPlatform(account, platform, useMixed) {
		return "endpoint_unsupported"
	}
	if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
		return "privacy_not_set"
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
		return "model_unsupported"
	}
	if needsUpstreamCheck && groupID != nil && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel) {
		return "channel_pricing_restricted"
	}
	if !s.isAccountSchedulableForQuota(account) {
		return "quota_exceeded"
	}
	return ""
}

func (s *GatewayService) IsSingleSchedulableAccountForRequest(ctx context.Context, groupID *int64, requestedModel string) bool {
	group, resolvedGroupID, err := s.checkClaudeCodeRestriction(ctx, groupID)
	if err != nil {
		return false
	}
	ctx = s.withGroupContext(ctx, group)
	platform, hasForcePlatform, err := s.resolvePlatform(ctx, resolvedGroupID, group, requestedModel)
	if err != nil {
		return false
	}
	accounts, useMixed, err := s.listSchedulableAccounts(ctx, resolvedGroupID, platform, hasForcePlatform)
	if err != nil || len(accounts) == 0 {
		return false
	}
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, resolvedGroupID)
	count := 0
	for i := range accounts {
		if s.gatewayHardSelectionReason(ctx, &accounts[i], resolvedGroupID, requestedModel, platform, useMixed, group, needsUpstreamCheck) == "" {
			count++
			if count > 1 {
				return false
			}
		}
	}
	return count == 1
}

func gatewaySelectionLoadInfo(loadMap map[int64]*AccountLoadInfo, account *Account) *AccountLoadInfo {
	if account == nil || loadMap == nil {
		return nil
	}
	return loadMap[account.ID]
}

func gatewaySelectionIDsFromMap(values map[int64]struct{}) []int64 {
	if len(values) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
