package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *OpenAIGatewayService) emitOpenAISelectionEmptyAlert(ctx context.Context, req OpenAIAccountScheduleRequest, decision OpenAIAccountScheduleDecision, err error) {
	if s == nil || s.opsRuntimeAlerts == nil || !decision.Diagnostics.Collected || decision.Diagnostics.FinalCandidateCount != 0 {
		return
	}
	diag := decision.Diagnostics
	endpoint := strings.TrimSpace(diag.Endpoint)
	if endpoint == "" {
		endpoint = schedulerEndpointFromOpenAIRequest(req)
	}
	model := strings.TrimSpace(diag.Model)
	if model == "" {
		model = strings.TrimSpace(req.RequestedModel)
	}
	groupID := diag.GroupID
	if groupID == 0 {
		groupID = derefGroupID(req.GroupID)
	}

	dimensions := map[string]any{
		"platform":                              PlatformOpenAI,
		"group_id":                              groupID,
		"model":                                 model,
		"endpoint":                              endpoint,
		"require_compact":                       diag.RequireCompact,
		"compact_strict_supported_only":         diag.CompactStrictSupportedOnly,
		"required_transport":                    diag.RequiredTransport,
		"required_capability":                   diag.RequiredCapability,
		"required_image_capability":             diag.RequiredImageCapability,
		"group_binding_count":                   diag.GroupBindingAccountCount,
		"excluded_count":                        diag.ExcludedAccountCount,
		"after_excluded_count":                  diag.AfterExcludedCount,
		"model_supported_count":                 diag.ModelSupportedCount,
		"endpoint_supported_count":              diag.EndpointSupportedCount,
		"compact_supported_count":               diag.CompactSupportedCount,
		"state_allowed_count":                   diag.StateAllowedCount,
		"circuit_allowed_count":                 diag.CircuitAllowedCount,
		"concurrency_slot_allowed_count":        diag.ConcurrencySlotAllowedCount,
		"final_candidate_count":                 diag.FinalCandidateCount,
		"group_binding_account_ids":             diag.GroupBindingAccountIDs,
		"after_excluded_account_ids":            diag.AfterExcludedAccountIDs,
		"model_supported_account_ids":           diag.ModelSupportedAccountIDs,
		"endpoint_supported_account_ids":        diag.EndpointSupportedAccountIDs,
		"compact_supported_account_ids":         diag.CompactSupportedAccountIDs,
		"state_allowed_account_ids":             diag.StateAllowedAccountIDs,
		"circuit_allowed_account_ids":           diag.CircuitAllowedAccountIDs,
		"candidate_account_ids":                 diag.CandidateAccountIDs,
		"excluded_account_ids":                  diag.ExcludedAccountIDs,
		"model_unsupported_account_ids":         diag.ModelUnsupportedAccountIDs,
		"endpoint_unsupported_account_ids":      diag.EndpointUnsupportedAccountIDs,
		"compact_unsupported_account_ids":       diag.CompactUnsupportedAccountIDs,
		"state_filtered_account_ids":            diag.StateFilteredAccountIDs,
		"circuit_filtered_account_ids":          diag.CircuitFilteredAccountIDs,
		"concurrency_slot_filtered_account_ids": diag.ConcurrencySlotFilteredAccountIDs,
		"half_open_filtered_account_ids":        diag.HalfOpenFilteredAccountIDs,
		"filter_reason_counts":                  diag.FilterReasonCounts,
	}
	description := opsRuntimeAlertOpenAISelectionEmptyDescription
	if err != nil {
		description = fmt.Sprintf("%s: %s", description, err.Error())
	}
	s.opsRuntimeAlerts.Emit(ctx, OpsRuntimeAlertInput{
		Type:        OpsRuntimeAlertTypeOpenAISelectionEmpty,
		Severity:    "P1",
		Title:       "OpenAI scheduler empty candidates",
		Description: description,
		Dimensions:  dimensions,
		DedupKey:    fmt.Sprintf("%s:%d:%s:%s:%t", OpsRuntimeAlertTypeOpenAISelectionEmpty, groupID, model, endpoint, diag.RequireCompact),
		DedupWindow: 2 * time.Minute,
	})
}
