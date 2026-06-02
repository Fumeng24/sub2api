package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

func openAISelectionDiagnosticZapFields(decision service.OpenAIAccountScheduleDecision) []zap.Field {
	diag := decision.Diagnostics
	if !diag.Collected {
		return nil
	}
	fields := []zap.Field{
		zap.Int64("selection_diag_group_id", diag.GroupID),
		zap.String("selection_diag_model", diag.Model),
		zap.String("selection_diag_endpoint", diag.Endpoint),
		zap.Bool("selection_diag_require_compact", diag.RequireCompact),
		zap.Bool("selection_diag_compact_strict_supported_only", diag.CompactStrictSupportedOnly),
		zap.String("selection_diag_required_transport", diag.RequiredTransport),
		zap.String("selection_diag_required_capability", diag.RequiredCapability),
		zap.String("selection_diag_required_image_capability", diag.RequiredImageCapability),
		zap.Int("selection_diag_group_binding_count", diag.GroupBindingAccountCount),
		zap.Int("selection_diag_active_schedulable_count", diag.ActiveSchedulableCount),
		zap.Int("selection_diag_excluded_count", diag.ExcludedAccountCount),
		zap.Int("selection_diag_after_excluded_count", diag.AfterExcludedCount),
		zap.Int("selection_diag_model_supported_count", diag.ModelSupportedCount),
		zap.Int("selection_diag_endpoint_supported_count", diag.EndpointSupportedCount),
		zap.Int("selection_diag_compact_supported_count", diag.CompactSupportedCount),
		zap.Int("selection_diag_state_allowed_count", diag.StateAllowedCount),
		zap.Int("selection_diag_circuit_allowed_count", diag.CircuitAllowedCount),
		zap.Int("selection_diag_concurrency_slot_allowed_count", diag.ConcurrencySlotAllowedCount),
		zap.Int("selection_diag_final_candidate_count", diag.FinalCandidateCount),
		zap.Int("selection_diag_state_filtered_count", diag.StateFilteredCount),
		zap.Int("selection_diag_circuit_filtered_count", diag.CircuitFilteredCount),
		zap.Int("selection_diag_concurrency_slot_filtered_count", diag.ConcurrencySlotFilteredCount),
		zap.Int("selection_diag_half_open_filtered_count", diag.HalfOpenFilteredCount),
		zap.Int("selection_diag_compact_unsupported_count", diag.CompactUnsupportedCount),
		zap.Int("selection_diag_status_filtered_count", diag.StatusFilteredCount),
		zap.Int("selection_diag_temp_unschedulable_filtered_count", diag.TempUnschedulableFilteredCount),
		zap.Int("selection_diag_overload_filtered_count", diag.OverloadFilteredCount),
		zap.Int("selection_diag_rate_limit_filtered_count", diag.RateLimitFilteredCount),
		zap.Int("selection_diag_model_rate_limit_filtered_count", diag.ModelRateLimitFilteredCount),
		zap.Int("selection_diag_channel_pricing_restriction_filtered_count", diag.ChannelRestrictionFilteredCount),
		zap.Int("selection_diag_group_scope_filtered_count", diag.GroupScopeFilteredCount),
	}
	if len(diag.ExcludedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_excluded_account_ids", diag.ExcludedAccountIDs))
	}
	if len(diag.GroupBindingAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_group_binding_account_ids", diag.GroupBindingAccountIDs))
	}
	if len(diag.ActiveSchedulableAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_active_schedulable_account_ids", diag.ActiveSchedulableAccountIDs))
	}
	if len(diag.AfterExcludedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_after_excluded_account_ids", diag.AfterExcludedAccountIDs))
	}
	if len(diag.ModelSupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_model_supported_account_ids", diag.ModelSupportedAccountIDs))
	}
	if len(diag.EndpointSupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_endpoint_supported_account_ids", diag.EndpointSupportedAccountIDs))
	}
	if len(diag.CompactSupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_compact_supported_account_ids", diag.CompactSupportedAccountIDs))
	}
	if len(diag.StateAllowedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_state_allowed_account_ids", diag.StateAllowedAccountIDs))
	}
	if len(diag.CircuitAllowedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_circuit_allowed_account_ids", diag.CircuitAllowedAccountIDs))
	}
	if len(diag.CandidateAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_candidate_account_ids", diag.CandidateAccountIDs))
	}
	if len(diag.OrderedCandidateAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_ordered_candidate_account_ids", diag.OrderedCandidateAccountIDs))
	}
	if len(diag.ModelUnsupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_model_unsupported_account_ids", diag.ModelUnsupportedAccountIDs))
	}
	if len(diag.EndpointUnsupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_endpoint_unsupported_account_ids", diag.EndpointUnsupportedAccountIDs))
	}
	if len(diag.ChannelRestrictionAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_channel_pricing_restriction_account_ids", diag.ChannelRestrictionAccountIDs))
	}
	if len(diag.CompactUnsupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_compact_unsupported_account_ids", diag.CompactUnsupportedAccountIDs))
	}
	if len(diag.StateFilteredAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_state_filtered_account_ids", diag.StateFilteredAccountIDs))
	}
	if len(diag.CircuitFilteredAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_circuit_filtered_account_ids", diag.CircuitFilteredAccountIDs))
	}
	if len(diag.ConcurrencySlotFilteredAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_concurrency_slot_filtered_account_ids", diag.ConcurrencySlotFilteredAccountIDs))
	}
	if len(diag.HalfOpenFilteredAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_half_open_filtered_account_ids", diag.HalfOpenFilteredAccountIDs))
	}
	if len(diag.FilterReasonCounts) > 0 {
		fields = append(fields, zap.Any("selection_diag_filter_reason_counts", diag.FilterReasonCounts))
	}
	return fields
}

func openAIAccountSelectFailedFields(err error, excludedAccountCount int, decision service.OpenAIAccountScheduleDecision) []zap.Field {
	fields := []zap.Field{
		zap.Error(err),
		zap.Int("excluded_account_count", excludedAccountCount),
	}
	fields = append(fields, openAISelectionDiagnosticZapFields(decision)...)
	return fields
}
