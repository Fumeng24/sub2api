package handler

import (
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

func gatewaySelectionDiagnosticZapFields(err error) []zap.Field {
	var noAvailable *service.GatewayNoAvailableAccountsError
	if !errors.As(err, &noAvailable) || noAvailable == nil || !noAvailable.Diagnostics.Collected {
		return nil
	}
	diag := noAvailable.Diagnostics
	fields := []zap.Field{
		zap.Int64("selection_diag_group_id", diag.GroupID),
		zap.String("selection_diag_model", diag.Model),
		zap.String("selection_diag_endpoint", diag.Endpoint),
		zap.String("selection_diag_platform", diag.Platform),
		zap.Int("group_binding_count", diag.GroupBindingAccountCount),
		zap.Int("active_schedulable_count", diag.ActiveSchedulableCount),
		zap.Int("excluded_count", diag.ExcludedAccountCount),
		zap.Int("after_excluded_count", diag.AfterExcludedCount),
		zap.Int("model_supported_count", diag.ModelSupportedCount),
		zap.Int("endpoint_supported_count", diag.EndpointSupportedCount),
		zap.Int("state_allowed_count", diag.StateAllowedCount),
		zap.Int("circuit_allowed_count", diag.CircuitAllowedCount),
		zap.Int("concurrency_slot_allowed_count", diag.ConcurrencySlotAllowedCount),
		zap.Int("final_candidate_count", diag.FinalCandidateCount),
	}
	if len(diag.ExcludedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_excluded_account_ids", diag.ExcludedAccountIDs))
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
	if len(diag.StateAllowedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_state_allowed_account_ids", diag.StateAllowedAccountIDs))
	}
	if len(diag.CircuitAllowedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_circuit_allowed_account_ids", diag.CircuitAllowedAccountIDs))
	}
	if len(diag.CandidateAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_candidate_account_ids", diag.CandidateAccountIDs))
	}
	if len(diag.ModelUnsupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_model_unsupported_account_ids", diag.ModelUnsupportedAccountIDs))
	}
	if len(diag.EndpointUnsupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_endpoint_unsupported_account_ids", diag.EndpointUnsupportedAccountIDs))
	}
	if len(diag.StateFilteredAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_state_filtered_account_ids", diag.StateFilteredAccountIDs))
	}
	if len(diag.CircuitFilteredAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_circuit_filtered_account_ids", diag.CircuitFilteredAccountIDs))
	}
	if len(diag.ChannelRestrictionAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_channel_pricing_restriction_account_ids", diag.ChannelRestrictionAccountIDs))
	}
	if len(diag.ConcurrencyFullAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_concurrency_full_account_ids", diag.ConcurrencyFullAccountIDs))
	}
	if len(diag.SkippedAccounts) > 0 {
		fields = append(fields, zap.Any("skipped_accounts", diag.SkippedAccounts))
	}
	if len(diag.FilterReasonCounts) > 0 {
		fields = append(fields, zap.Any("skip_reason", diag.FilterReasonCounts))
	}
	return fields
}
