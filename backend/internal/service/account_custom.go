package service

import (
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

// AccountSchedulingBlockReason is the local scheduler diagnostic overlay.
// The upstream Account model remains the source of truth for persisted fields;
// these helpers only derive runtime eligibility from those fields.
type AccountSchedulingBlockReason string

const (
	AccountSchedulingBlockNone              AccountSchedulingBlockReason = ""
	AccountSchedulingBlockInactive          AccountSchedulingBlockReason = "inactive"
	AccountSchedulingBlockManual            AccountSchedulingBlockReason = "manual_unschedulable"
	AccountSchedulingBlockExpired           AccountSchedulingBlockReason = "expired"
	AccountSchedulingBlockOverloaded        AccountSchedulingBlockReason = "overloaded"
	AccountSchedulingBlockRateLimited       AccountSchedulingBlockReason = "rate_limited"
	AccountSchedulingBlockTempUnschedulable AccountSchedulingBlockReason = "temp_unschedulable"
	AccountSchedulingBlockQuotaExceeded     AccountSchedulingBlockReason = "quota_exceeded"
)

func (r AccountSchedulingBlockReason) String() string { return string(r) }

func (r AccountSchedulingBlockReason) SchedulerState() string {
	switch r {
	case AccountSchedulingBlockInactive:
		return "error"
	case AccountSchedulingBlockManual:
		return "stopped"
	case AccountSchedulingBlockExpired:
		return "expired"
	case AccountSchedulingBlockTempUnschedulable:
		return "temp_unschedulable"
	case AccountSchedulingBlockRateLimited:
		return "rate_limited"
	case AccountSchedulingBlockOverloaded:
		return "overloaded"
	case AccountSchedulingBlockQuotaExceeded:
		return "quota_exceeded"
	default:
		return "active"
	}
}

func (a *Account) SchedulingBlockReasonAt(now time.Time) AccountSchedulingBlockReason {
	if reason := a.HardSchedulingBlockReasonAt(now); reason != AccountSchedulingBlockNone {
		return reason
	}
	if a.OverloadUntil != nil && now.Before(*a.OverloadUntil) {
		return AccountSchedulingBlockOverloaded
	}
	if a.RateLimitResetAt != nil && now.Before(*a.RateLimitResetAt) {
		return AccountSchedulingBlockRateLimited
	}
	if a.TempUnschedulableUntil != nil && now.Before(*a.TempUnschedulableUntil) {
		return AccountSchedulingBlockTempUnschedulable
	}
	if a.IsAPIKeyOrBedrock() && a.IsQuotaExceeded() {
		return AccountSchedulingBlockQuotaExceeded
	}
	return AccountSchedulingBlockNone
}

func (a *Account) HardSchedulingBlockReasonAt(now time.Time) AccountSchedulingBlockReason {
	if a == nil || !a.IsActive() {
		return AccountSchedulingBlockInactive
	}
	if !a.Schedulable {
		return AccountSchedulingBlockManual
	}
	if a.AutoPauseOnExpired && a.ExpiresAt != nil && !now.Before(*a.ExpiresAt) {
		return AccountSchedulingBlockExpired
	}
	if a.IsAPIKeyOrBedrock() && a.IsQuotaExceeded() {
		return AccountSchedulingBlockQuotaExceeded
	}
	return AccountSchedulingBlockNone
}

func (a *Account) IsSchedulableAt(now time.Time) bool {
	return a.SchedulingBlockReasonAt(now) == AccountSchedulingBlockNone
}

func (a *Account) IsSchedulableForGroupAt(groupID int64, now time.Time) bool {
	// Runtime cooldown is account-scoped. The group argument is retained for
	// callers compiled against the older API, but legacy group cooldown data
	// must not make one shared account healthy in one group and unhealthy in
	// another.
	return a.SchedulingBlockReasonAt(now) == AccountSchedulingBlockNone
}

func (a *Account) SchedulingBlockReasonForGroupAt(groupID int64, now time.Time) AccountSchedulingBlockReason {
	return a.SchedulingBlockReasonAt(now)
}

func (a *Account) SchedulerStateAt(now time.Time) string {
	if a == nil {
		return "unknown"
	}
	return a.SchedulingBlockReasonAt(now).SchedulerState()
}

type AccountSchedulabilityClass struct {
	Reason             AccountSchedulingBlockReason
	Schedulable        bool
	StatusError        bool
	TemporarilyLimited bool
	RateLimited        bool
	Overloaded         bool
	TempUnschedulable  bool
}

func (a *Account) SchedulabilityClassAt(now time.Time) AccountSchedulabilityClass {
	reason := a.SchedulingBlockReasonAt(now)
	return AccountSchedulabilityClass{
		Reason:      reason,
		Schedulable: reason == AccountSchedulingBlockNone,
		StatusError: reason == AccountSchedulingBlockInactive && a != nil && a.Status == StatusError,
		TemporarilyLimited: reason == AccountSchedulingBlockRateLimited ||
			reason == AccountSchedulingBlockOverloaded ||
			reason == AccountSchedulingBlockTempUnschedulable,
		RateLimited:       reason == AccountSchedulingBlockRateLimited,
		Overloaded:        reason == AccountSchedulingBlockOverloaded,
		TempUnschedulable: reason == AccountSchedulingBlockTempUnschedulable,
	}
}

func (a *Account) isSchedulableCustom(now time.Time) bool { return a.IsSchedulableAt(now) }

// IsSchedulerBucketMember excludes only durable membership blockers. Runtime
// windows are evaluated when selecting an account for an individual request.
func (a *Account) IsSchedulerBucketMember() bool {
	if a == nil || !a.IsActive() || !a.Schedulable {
		return false
	}
	now := time.Now()
	return !(a.AutoPauseOnExpired && a.ExpiresAt != nil && !now.Before(*a.ExpiresAt))
}

func requestedModelLookupCandidates(platform, requestedModel string) []string {
	trimmed := strings.TrimSpace(requestedModel)
	if trimmed == "" {
		return nil
	}
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
	}
	add(trimmed)
	add(normalizeRequestedModelForLookup(platform, trimmed))
	if platform == PlatformAnthropic {
		add(claude.NormalizeModelID(trimmed))
		add(claude.DenormalizeModelID(trimmed))
	}
	return candidates
}

func sortedModelMappingKeys(mapping map[string]string) []string {
	if len(mapping) == 0 {
		return nil
	}
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	// Mapping diagnostics are intentionally deterministic for API responses and logs.
	sort.Strings(keys)
	return keys
}

func mappingMatchKeyForDiagnostics(mapping map[string]string, requestedModel string) (string, bool) {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" || len(mapping) == 0 {
		return "", false
	}
	if _, exists := mapping[requestedModel]; exists {
		return requestedModel, true
	}
	for _, pattern := range sortedModelMappingKeys(mapping) {
		if matchWildcard(pattern, requestedModel) {
			return pattern, true
		}
	}
	return "", false
}

func (a *Account) modelMappingKeysForDiagnostics() []string {
	if a == nil {
		return nil
	}
	return sortedModelMappingKeys(a.GetModelMapping())
}

func (a *Account) modelMappingMatchForDiagnostics(requestedModel string) (bool, string, string) {
	if a == nil {
		return false, "", ""
	}
	mapping := a.GetModelMapping()
	if len(mapping) == 0 {
		return false, "", ""
	}
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if key, ok := mappingMatchKeyForDiagnostics(mapping, candidate); ok {
			return true, candidate, key
		}
	}
	return false, "", ""
}

func (a *Account) isModelSupportedCustom(requestedModel string) bool {
	mapping := a.GetModelMapping()
	if len(mapping) == 0 {
		if a.IsOpenAIOAuth() && !a.IsOpenAIPassthroughEnabled() {
			return isOpenAIOAuthServableModel(requestedModel)
		}
		return true
	}
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if mappingSupportsRequestedModel(mapping, candidate) {
			return true
		}
	}
	return false
}

func (a *Account) isAdditionalModelSupportedCustom(requestedModel string) bool {
	mapping := a.GetModelMapping()
	if len(mapping) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(requestedModel)
	normalized := normalizeRequestedModelForLookup(a.Platform, trimmed)
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if candidate == trimmed || candidate == normalized {
			continue
		}
		if mappingSupportsRequestedModel(mapping, candidate) {
			return true
		}
	}
	return false
}

func (a *Account) resolveMappedModelCustom(requestedModel string) (string, bool) {
	mapping := a.GetModelMapping()
	if len(mapping) == 0 {
		return requestedModel, false
	}
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if mappedModel, matched := resolveRequestedModelInMapping(mapping, candidate); matched {
			return mappedModel, true
		}
	}
	return requestedModel, false
}

func (a *Account) resolveAdditionalMappedModelCustom(requestedModel string) (string, bool) {
	mapping := a.GetModelMapping()
	trimmed := strings.TrimSpace(requestedModel)
	normalized := normalizeRequestedModelForLookup(a.Platform, trimmed)
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if candidate == trimmed || candidate == normalized {
			continue
		}
		if mappedModel, matched := resolveRequestedModelInMapping(mapping, candidate); matched {
			return mappedModel, true
		}
	}
	return requestedModel, false
}

// GetCacheAffinityGroup returns an operator-defined prompt compatibility group.
func (a *Account) GetCacheAffinityGroup() string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.GetExtraString("cache_affinity_group"))
}
