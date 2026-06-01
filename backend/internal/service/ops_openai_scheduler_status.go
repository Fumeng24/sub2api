package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

type OpsOpenAISchedulerStatus struct {
	Enabled                  bool                                  `json:"enabled"`
	AdvancedSchedulerEnabled bool                                  `json:"advanced_scheduler_enabled"`
	Model                    string                                `json:"model"`
	Endpoint                 string                                `json:"endpoint"`
	Timestamp                time.Time                             `json:"timestamp"`
	Metrics                  OpenAIAccountSchedulerMetricsSnapshot `json:"metrics"`
	Summary                  OpsOpenAISchedulerSummary             `json:"summary"`
	Groups                   []*OpsOpenAISchedulerGroupStatus      `json:"groups"`
}

type OpsOpenAISchedulerSummary struct {
	GroupCount           int `json:"group_count"`
	AccountCount         int `json:"account_count"`
	AvailableCount       int `json:"available_count"`
	BlockedCount         int `json:"blocked_count"`
	CircuitOpenCount     int `json:"circuit_open_count"`
	HalfOpenCount        int `json:"half_open_count"`
	ConcurrencyFullCount int `json:"concurrency_full_count"`
}

type OpsOpenAISchedulerGroupStatus struct {
	GroupID              int64                              `json:"group_id"`
	GroupName            string                             `json:"group_name"`
	Platform             string                             `json:"platform"`
	TotalAccounts        int                                `json:"total_accounts"`
	AvailableCount       int                                `json:"available_count"`
	BlockedCount         int                                `json:"blocked_count"`
	CircuitOpenCount     int                                `json:"circuit_open_count"`
	HalfOpenCount        int                                `json:"half_open_count"`
	ConcurrencyFullCount int                                `json:"concurrency_full_count"`
	Accounts             []*OpsOpenAISchedulerAccountStatus `json:"accounts"`
}

type OpsOpenAISchedulerAccountStatus struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	AccountType string `json:"account_type"`
	Status      string `json:"status"`

	Role                 string `json:"role"`
	Priority             int    `json:"priority"`
	Weight               int    `json:"weight"`
	SortOrder            int    `json:"sort_order"`
	SchedulingConfigured bool   `json:"scheduling_configured"`

	ModelSupported    bool   `json:"model_supported"`
	EndpointSupported bool   `json:"endpoint_supported"`
	CompactSupported  bool   `json:"compact_supported"`
	StateAllowed      bool   `json:"state_allowed"`
	StateReason       string `json:"state_reason,omitempty"`
	CircuitState      string `json:"circuit_state"`
	CircuitReason     string `json:"circuit_reason,omitempty"`
	IsAvailable       bool   `json:"is_available"`
	BlockReason       string `json:"block_reason,omitempty"`

	CurrentConcurrency int `json:"current_concurrency"`
	MaxConcurrency     int `json:"max_concurrency"`
	WaitingCount       int `json:"waiting_count"`
	LoadRate           int `json:"load_rate"`

	RuntimeCircuitUntil         *time.Time `json:"runtime_circuit_until,omitempty"`
	RuntimeCircuitRemainingSec  *int64     `json:"runtime_circuit_remaining_sec,omitempty"`
	TempUnschedulableUntil      *time.Time `json:"temp_unschedulable_until,omitempty"`
	TempUnschedulableReason     string     `json:"temp_unschedulable_reason,omitempty"`
	TempUnschedulableStatusCode *int       `json:"temp_unschedulable_status_code,omitempty"`

	SchedulerHealthScore       float64 `json:"scheduler_health_score"`
	SchedulerErrorRate         float64 `json:"scheduler_error_rate"`
	SchedulerTTFTMs            float64 `json:"scheduler_ttft_ms,omitempty"`
	SchedulerConsecutiveFailed int     `json:"scheduler_consecutive_failed"`
	SchedulerLastFailureReason string  `json:"scheduler_last_failure_reason,omitempty"`
}

func (s *OpsService) GetOpenAISchedulerStatus(ctx context.Context, model string, endpoint string, groupIDFilter *int64) (*OpsOpenAISchedulerStatus, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	model = strings.TrimSpace(model)
	endpoint = normalizeOpenAISchedulerStatusEndpoint(endpoint)
	status := &OpsOpenAISchedulerStatus{
		Enabled:   s != nil && s.openAIGatewayService != nil,
		Model:     model,
		Endpoint:  endpoint,
		Timestamp: now,
	}
	if s == nil || s.openAIGatewayService == nil {
		return status, nil
	}
	status.AdvancedSchedulerEnabled = s.openAIGatewayService.isOpenAIAdvancedSchedulerEnabled(ctx)
	status.Metrics = s.openAIGatewayService.SnapshotOpenAIAccountSchedulerMetrics()

	accounts, err := s.listAllAccountsForOps(ctx, PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	if groupIDFilter != nil && *groupIDFilter > 0 {
		filtered := make([]Account, 0, len(accounts))
		for _, acc := range accounts {
			if accountHasGroupForOpsStatus(acc, *groupIDFilter) {
				filtered = append(filtered, acc)
			}
		}
		accounts = filtered
	}

	loadMap := s.getAccountsLoadMapBestEffort(ctx, accounts)
	req := openAISchedulerStatusRequest(model, endpoint)
	groups := map[int64]*OpsOpenAISchedulerGroupStatus{}
	seenAccounts := map[int64]struct{}{}

	for i := range accounts {
		acc := accounts[i]
		if acc.ID <= 0 || !acc.IsOpenAI() {
			continue
		}
		seenAccounts[acc.ID] = struct{}{}
		groupRefs := openAISchedulerStatusGroupsForAccount(acc, groupIDFilter)
		for _, groupRef := range groupRefs {
			if groupRef == nil {
				continue
			}
			groupID := groupRef.ID
			if _, ok := groups[groupID]; !ok {
				groups[groupID] = &OpsOpenAISchedulerGroupStatus{
					GroupID:   groupID,
					GroupName: groupRef.Name,
					Platform:  groupRef.Platform,
					Accounts:  []*OpsOpenAISchedulerAccountStatus{},
				}
			}
			item := s.openAISchedulerAccountStatus(ctx, &acc, groupRef, req, loadMap[acc.ID], now)
			g := groups[groupID]
			g.Accounts = append(g.Accounts, item)
			g.TotalAccounts++
			if item.IsAvailable {
				g.AvailableCount++
			} else {
				g.BlockedCount++
			}
			switch item.CircuitState {
			case schedulerCircuitOpen, "runtime_open":
				g.CircuitOpenCount++
			case schedulerCircuitHalfOpen, "runtime_half_open":
				g.HalfOpenCount++
			}
			if item.LoadRate >= 100 {
				g.ConcurrencyFullCount++
			}
		}
	}

	status.Groups = make([]*OpsOpenAISchedulerGroupStatus, 0, len(groups))
	for _, group := range groups {
		sort.SliceStable(group.Accounts, func(i, j int) bool {
			a, b := group.Accounts[i], group.Accounts[j]
			if a.SortOrder != b.SortOrder {
				return a.SortOrder < b.SortOrder
			}
			if a.Priority != b.Priority {
				return a.Priority < b.Priority
			}
			return a.AccountID < b.AccountID
		})
		status.Groups = append(status.Groups, group)
		status.Summary.AvailableCount += group.AvailableCount
		status.Summary.BlockedCount += group.BlockedCount
		status.Summary.CircuitOpenCount += group.CircuitOpenCount
		status.Summary.HalfOpenCount += group.HalfOpenCount
		status.Summary.ConcurrencyFullCount += group.ConcurrencyFullCount
	}
	sort.SliceStable(status.Groups, func(i, j int) bool {
		if status.Groups[i].GroupID == 0 || status.Groups[j].GroupID == 0 {
			return status.Groups[j].GroupID != 0
		}
		return status.Groups[i].GroupID < status.Groups[j].GroupID
	})
	status.Summary.GroupCount = len(status.Groups)
	status.Summary.AccountCount = len(seenAccounts)
	return status, nil
}

func (s *OpsService) openAISchedulerAccountStatus(ctx context.Context, account *Account, group *Group, baseReq OpenAIAccountScheduleRequest, load *AccountLoadInfo, now time.Time) *OpsOpenAISchedulerAccountStatus {
	groupID := int64(0)
	if group != nil {
		groupID = group.ID
	}
	var groupIDPtr *int64
	if groupID > 0 {
		groupIDPtr = &groupID
	}
	req := baseReq
	req.GroupID = groupIDPtr

	cfg := accountGroupConfigFor(account, groupIDPtr)
	item := &OpsOpenAISchedulerAccountStatus{
		AccountID:            account.ID,
		AccountName:          account.Name,
		AccountType:          account.Type,
		Status:               account.Status,
		Role:                 cfg.NormalizedRole(),
		Priority:             cfg.Priority,
		Weight:               cfg.EffectiveWeight(),
		SortOrder:            cfg.EffectiveSortOrder(),
		SchedulingConfigured: cfg.SchedulingConfigured,
		ModelSupported:       req.RequestedModel == "" || account.IsModelSupported(req.RequestedModel),
		EndpointSupported:    accountSupportsOpenAICapabilities(account, req.RequiredCapability, req.RequiredImageCapability),
		CompactSupported:     openAICompactAccountAllowedForRequest(req, account),
		StateAllowed:         true,
		CircuitState:         schedulerCircuitClosed,
		SchedulerHealthScore: 1,
	}
	if load != nil {
		item.CurrentConcurrency = load.CurrentConcurrency
		item.WaitingCount = load.WaitingCount
		item.LoadRate = load.LoadRate
	}
	item.MaxConcurrency = account.EffectiveLoadFactor()

	if reason := openAIAccountStatusFilterReason(ctx, account, req, group); reason != "" {
		item.StateAllowed = false
		item.StateReason = reason
	}
	item.applyOpenAISchedulerCircuitStatus(s.openAIGatewayService, account.ID, req, now)
	if account.TempUnschedulableUntil != nil && now.Before(account.TempUnschedulableUntil.UTC()) {
		t := account.TempUnschedulableUntil.UTC()
		item.TempUnschedulableUntil = &t
		details := TempUnschedulableReasonDetailsFromRaw(account.TempUnschedulableReason)
		item.TempUnschedulableReason = details.DisplayReason
		item.TempUnschedulableStatusCode = details.StatusCode
	}
	item.IsAvailable = item.ModelSupported &&
		item.EndpointSupported &&
		item.CompactSupported &&
		item.StateAllowed &&
		item.CircuitState == schedulerCircuitClosed &&
		item.LoadRate < 100
	item.BlockReason = openAISchedulerStatusBlockReason(item)
	return item
}

func (item *OpsOpenAISchedulerAccountStatus) applyOpenAISchedulerCircuitStatus(gateway *OpenAIGatewayService, accountID int64, req OpenAIAccountScheduleRequest, now time.Time) {
	if item == nil || gateway == nil || accountID <= 0 {
		return
	}
	if until, ok := gateway.openAIAccountRuntimeBlockUntil(accountID); ok && until.After(now) {
		item.CircuitState = "runtime_open"
		item.CircuitReason = "runtime_circuit_open"
		t := until.UTC()
		item.RuntimeCircuitUntil = &t
		remaining := int64(until.Sub(now).Seconds())
		if remaining > 0 {
			item.RuntimeCircuitRemainingSec = &remaining
		}
	}
	if gateway.isOpenAIAccountCircuitHalfOpenInFlight(accountID, now) {
		item.CircuitState = "runtime_half_open"
		item.CircuitReason = "runtime_half_open_in_flight"
	}
	if gateway.schedulerHealth == nil {
		return
	}
	snap := gateway.schedulerHealth.snapshot(accountID, req.RequestedModel, schedulerEndpointFromOpenAIRequest(req), false)
	item.SchedulerHealthScore = snap.HealthScore
	item.SchedulerErrorRate = snap.ErrorRate
	if snap.HasTTFT {
		item.SchedulerTTFTMs = snap.TTFTEWMA
	}
	item.SchedulerConsecutiveFailed = snap.ConsecutiveFailed
	item.SchedulerLastFailureReason = snap.LastFailureReason
	if item.CircuitState != schedulerCircuitClosed {
		return
	}
	switch snap.CircuitState {
	case schedulerCircuitOpen:
		item.CircuitState = schedulerCircuitOpen
		item.CircuitReason = "scheduler_circuit_open"
		if !snap.CooldownUntil.IsZero() {
			t := snap.CooldownUntil.UTC()
			item.RuntimeCircuitUntil = &t
			remaining := int64(snap.CooldownUntil.Sub(now).Seconds())
			if remaining > 0 {
				item.RuntimeCircuitRemainingSec = &remaining
			}
		}
	case schedulerCircuitHalfOpen:
		item.CircuitState = schedulerCircuitHalfOpen
		item.CircuitReason = "scheduler_half_open"
	}
}

func openAISchedulerStatusBlockReason(item *OpsOpenAISchedulerAccountStatus) string {
	if item == nil || item.IsAvailable {
		return ""
	}
	switch {
	case !item.ModelSupported:
		return "model_unsupported"
	case !item.EndpointSupported:
		return "endpoint_unsupported"
	case !item.CompactSupported:
		return "compact_unsupported"
	case !item.StateAllowed:
		return item.StateReason
	case item.CircuitState != schedulerCircuitClosed:
		return item.CircuitReason
	case item.LoadRate >= 100:
		return "concurrency_full"
	default:
		return "unavailable"
	}
}

func normalizeOpenAISchedulerStatusEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "/v1/responses"
	}
	return endpoint
}

func openAISchedulerStatusRequest(model string, endpoint string) OpenAIAccountScheduleRequest {
	req := OpenAIAccountScheduleRequest{
		RequestedModel:     strings.TrimSpace(model),
		SchedulerEndpoint:  normalizeOpenAISchedulerStatusEndpoint(endpoint),
		RequiredTransport:  OpenAIUpstreamTransportAny,
		RequiredCapability: OpenAIEndpointCapabilityChatCompletions,
	}
	lower := strings.ToLower(req.SchedulerEndpoint)
	if strings.Contains(lower, "compact") {
		req.RequireCompact = true
	}
	if strings.Contains(lower, "embeddings") {
		req.RequiredCapability = OpenAIEndpointCapabilityEmbeddings
	}
	return req
}

func accountHasGroupForOpsStatus(account Account, groupID int64) bool {
	if groupID <= 0 {
		return true
	}
	for _, group := range account.Groups {
		if group != nil && group.ID == groupID {
			return true
		}
	}
	for _, ag := range account.AccountGroups {
		if ag.GroupID == groupID {
			return true
		}
	}
	return false
}

func openAISchedulerStatusGroupsForAccount(account Account, groupIDFilter *int64) []*Group {
	if groupIDFilter != nil && *groupIDFilter > 0 {
		group := openAISchedulerStatusGroupByID(account, *groupIDFilter)
		if group != nil {
			return []*Group{group}
		}
		return nil
	}
	if len(account.Groups) > 0 {
		out := make([]*Group, 0, len(account.Groups))
		seen := map[int64]struct{}{}
		for _, group := range account.Groups {
			if group == nil {
				continue
			}
			if _, ok := seen[group.ID]; ok {
				continue
			}
			seen[group.ID] = struct{}{}
			out = append(out, group)
		}
		if len(out) > 0 {
			return out
		}
	}
	if len(account.AccountGroups) > 0 {
		out := make([]*Group, 0, len(account.AccountGroups))
		seen := map[int64]struct{}{}
		for _, ag := range account.AccountGroups {
			if ag.GroupID <= 0 {
				continue
			}
			if _, ok := seen[ag.GroupID]; ok {
				continue
			}
			seen[ag.GroupID] = struct{}{}
			name := "Group " + strconvFormatInt64(ag.GroupID)
			out = append(out, &Group{ID: ag.GroupID, Name: name, Platform: PlatformOpenAI})
		}
		if len(out) > 0 {
			return out
		}
	}
	return []*Group{{ID: 0, Name: "Ungrouped", Platform: PlatformOpenAI}}
}

func openAISchedulerStatusGroupByID(account Account, groupID int64) *Group {
	for _, group := range account.Groups {
		if group != nil && group.ID == groupID {
			return group
		}
	}
	for _, ag := range account.AccountGroups {
		if ag.GroupID == groupID {
			return &Group{ID: groupID, Name: "Group " + strconvFormatInt64(groupID), Platform: PlatformOpenAI}
		}
	}
	return nil
}

func strconvFormatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}
