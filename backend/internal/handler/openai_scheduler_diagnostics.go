package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
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
		zap.Bool("selection_diag_require_codex_image_generation_bridge", diag.RequireCodexImageGenerationBridge),
		zap.Int("selection_diag_group_binding_count", diag.GroupBindingAccountCount),
		zap.Int("selection_diag_active_schedulable_count", diag.ActiveSchedulableCount),
		zap.Int("selection_diag_excluded_count", diag.ExcludedAccountCount),
		zap.Int("selection_diag_after_excluded_count", diag.AfterExcludedCount),
		zap.Int("selection_diag_model_supported_count", diag.ModelSupportedCount),
		zap.Int("selection_diag_endpoint_supported_count", diag.EndpointSupportedCount),
		zap.Int("selection_diag_image_generation_bridge_supported_count", diag.ImageGenerationBridgeSupportedCount),
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
		zap.Int("selection_diag_image_generation_bridge_unsupported_count", diag.ImageGenerationBridgeUnsupportedCount),
		zap.Int("selection_diag_status_filtered_count", diag.StatusFilteredCount),
		zap.Int("selection_diag_temp_unschedulable_filtered_count", diag.TempUnschedulableFilteredCount),
		zap.Int("selection_diag_overload_filtered_count", diag.OverloadFilteredCount),
		zap.Int("selection_diag_rate_limit_filtered_count", diag.RateLimitFilteredCount),
		zap.Int("selection_diag_model_rate_limit_filtered_count", diag.ModelRateLimitFilteredCount),
		zap.Int("selection_diag_channel_pricing_restriction_filtered_count", diag.ChannelRestrictionFilteredCount),
		zap.Int("selection_diag_group_scope_filtered_count", diag.GroupScopeFilteredCount),
	}
	if diag.RetryAfterSeconds > 0 {
		fields = append(fields, zap.Int("selection_diag_retry_after_seconds", diag.RetryAfterSeconds))
	}
	if !diag.EarliestRetryAt.IsZero() {
		fields = append(fields,
			zap.Time("selection_diag_earliest_retry_at", diag.EarliestRetryAt),
			zap.String("selection_diag_earliest_retry_reason", diag.EarliestRetryReason),
			zap.Int64("selection_diag_earliest_retry_account_id", diag.EarliestRetryAccountID),
		)
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
	if len(diag.ImageGenerationBridgeSupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_image_generation_bridge_supported_account_ids", diag.ImageGenerationBridgeSupportedAccountIDs))
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
	if len(diag.ImageGenerationBridgeUnsupportedAccountIDs) > 0 {
		fields = append(fields, zap.Int64s("selection_diag_image_generation_bridge_unsupported_account_ids", diag.ImageGenerationBridgeUnsupportedAccountIDs))
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
	if diag := decision.Diagnostics; diag.Collected {
		fields = append(fields,
			zap.Int64("group_id", diag.GroupID),
			zap.String("model", diag.Model),
			zap.String("endpoint", diag.Endpoint),
			zap.Int("group_binding_count", diag.GroupBindingAccountCount),
			zap.Int("active_schedulable_count", diag.ActiveSchedulableCount),
			zap.Int("model_supported_count", diag.ModelSupportedCount),
			zap.Int("endpoint_supported_count", diag.EndpointSupportedCount),
			zap.Int("state_allowed_count", diag.StateAllowedCount),
			zap.Int("circuit_allowed_count", diag.CircuitAllowedCount),
			zap.Int("concurrency_slot_allowed_count", diag.ConcurrencySlotAllowedCount),
			zap.Int("final_candidate_count", diag.FinalCandidateCount),
		)
		if diag.RetryAfterSeconds > 0 {
			fields = append(fields, zap.Int("retry_after_seconds", diag.RetryAfterSeconds))
		}
		if !diag.EarliestRetryAt.IsZero() {
			fields = append(fields,
				zap.Time("earliest_retry_at", diag.EarliestRetryAt),
				zap.String("earliest_retry_reason", diag.EarliestRetryReason),
				zap.Int64("earliest_retry_account_id", diag.EarliestRetryAccountID),
			)
		}
		if len(diag.ExcludedAccountIDs) > 0 {
			fields = append(fields, zap.Int64s("tried_accounts", diag.ExcludedAccountIDs))
		}
		if len(diag.GroupBindingAccountIDs) > 0 {
			fields = append(fields, zap.Int64s("group_binding_accounts", diag.GroupBindingAccountIDs))
		}
		if len(diag.CandidateAccountIDs) > 0 {
			fields = append(fields, zap.Int64s("candidate_accounts", diag.CandidateAccountIDs))
		}
		if len(diag.OrderedCandidateAccountIDs) > 0 {
			fields = append(fields, zap.Int64s("ordered_candidate_accounts", diag.OrderedCandidateAccountIDs))
		}
		if skipped := openAISelectionSkippedAccounts(diag); len(skipped) > 0 {
			fields = append(fields, zap.Any("skipped_accounts", skipped))
		}
		if len(diag.FilterReasonCounts) > 0 {
			fields = append(fields, zap.Any("skip_reason", diag.FilterReasonCounts))
		}
	}
	fields = append(fields, openAISelectionDiagnosticZapFields(decision)...)
	return fields
}

func setOpenAISelectionRetryAfterHeader(c *gin.Context, err error) {
	if retryAfter := service.OpenAIRetryAfterSecondsFromError(err); retryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(retryAfter))
	}
}

func openAISelectionSkippedAccounts(diag service.OpenAIAccountSelectionDiagnostics) map[string][]int64 {
	skipped := make(map[string][]int64)
	add := func(reason string, ids []int64) {
		if len(ids) == 0 {
			return
		}
		skipped[reason] = ids
	}
	add("excluded", diag.ExcludedAccountIDs)
	add("model_unsupported", diag.ModelUnsupportedAccountIDs)
	add("endpoint_unsupported", diag.EndpointUnsupportedAccountIDs)
	add("image_generation_bridge_unsupported", diag.ImageGenerationBridgeUnsupportedAccountIDs)
	add("channel_pricing_restricted", diag.ChannelRestrictionAccountIDs)
	add("compact_unsupported", diag.CompactUnsupportedAccountIDs)
	add("state_filtered", diag.StateFilteredAccountIDs)
	add("circuit_filtered", diag.CircuitFilteredAccountIDs)
	add("concurrency_full", diag.ConcurrencySlotFilteredAccountIDs)
	add("half_open_filtered", diag.HalfOpenFilteredAccountIDs)
	return skipped
}

func openAISelectionEmptyErrorResponse(decision service.OpenAIAccountScheduleDecision) (int, string, string) {
	diag := decision.Diagnostics
	if !diag.Collected {
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	}
	if diag.GroupBindingAccountCount > 0 && diag.ActiveSchedulableCount > 0 && diag.ModelSupportedCount == 0 {
		model := strings.TrimSpace(diag.Model)
		if model == "" {
			return http.StatusBadRequest, "model_not_supported", "Requested model is not available in this group"
		}
		return http.StatusBadRequest, "model_not_supported", "Requested model is not available in this group: " + model
	}
	if diag.RequireCodexImageGenerationBridge && diag.ImageGenerationBridgeSupportedCount == 0 && diag.ImageGenerationBridgeUnsupportedCount > 0 {
		return http.StatusBadRequest, "image_generation_bridge_not_available", "No accounts in this group support Responses image_generation"
	}
	if diag.RequireCompact && diag.CompactSupportedCount == 0 && diag.CompactUnsupportedCount > 0 {
		return http.StatusServiceUnavailable, "compact_not_supported", "No available OpenAI accounts support /responses/compact"
	}
	if diag.EndpointSupportedCount == 0 && diag.ModelSupportedCount > 0 {
		return http.StatusServiceUnavailable, "endpoint_not_supported", "No accounts in this group support the requested endpoint"
	}
	return http.StatusServiceUnavailable, "api_error", "No available accounts for the requested model and capability"
}
