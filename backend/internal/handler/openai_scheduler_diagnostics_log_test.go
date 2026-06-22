package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpenAIAccountSelectFailedFields_IncludesPlainDiagnostics(t *testing.T) {
	decision := service.OpenAIAccountScheduleDecision{
		Diagnostics: service.OpenAIAccountSelectionDiagnostics{
			Collected:                    true,
			GroupID:                      10,
			Model:                        "gpt-5.5",
			Endpoint:                     "/v1/responses",
			GroupBindingAccountCount:     1,
			GroupBindingAccountIDs:       []int64{38804},
			ExcludedAccountIDs:           []int64{38804},
			ModelSupportedCount:          1,
			EndpointSupportedCount:       1,
			StateAllowedCount:            1,
			CircuitAllowedCount:          0,
			ConcurrencySlotAllowedCount:  0,
			FinalCandidateCount:          0,
			CircuitFilteredAccountIDs:    []int64{38804},
			FilterReasonCounts:           map[string]int{"excluded": 1, "runtime_circuit_open": 1},
			OrderedCandidateAccountIDs:   nil,
			ActiveSchedulableCount:       1,
			ActiveSchedulableAccountIDs:  []int64{38804},
			AfterExcludedAccountIDs:      nil,
			ModelSupportedAccountIDs:     []int64{38804},
			EndpointSupportedAccountIDs:  []int64{38804},
			CompactSupportedAccountIDs:   []int64{38804},
			StateAllowedAccountIDs:       []int64{38804},
			CircuitAllowedAccountIDs:     nil,
			ConcurrencySlotFilteredCount: 0,
		},
	}

	fields := openAIAccountSelectFailedFields(errors.New("no available accounts"), 1, decision)
	keys := zapFieldKeys(fields)

	require.Contains(t, keys, "group_binding_count")
	require.Contains(t, keys, "final_candidate_count")
	require.Contains(t, keys, "tried_accounts")
	require.Contains(t, keys, "group_binding_accounts")
	require.Contains(t, keys, "skipped_accounts")
	require.Contains(t, keys, "skip_reason")
}

func TestOpenAISelectionEmptyErrorResponse_ModelUnsupported(t *testing.T) {
	status, errType, message := openAISelectionEmptyErrorResponse(service.OpenAIAccountScheduleDecision{
		Diagnostics: service.OpenAIAccountSelectionDiagnostics{
			Collected:                true,
			Model:                    "gpt-5.2",
			GroupBindingAccountCount: 8,
			ActiveSchedulableCount:   8,
			AfterExcludedCount:       8,
			ModelSupportedCount:      0,
			FilterReasonCounts:       map[string]int{"model_unsupported": 8},
		},
	})

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "model_not_supported", errType)
	require.Contains(t, message, "gpt-5.2")
}

func TestOpenAISelectionEmptyErrorMetadata_ModelMismatchRedactsAccountSummary(t *testing.T) {
	metadata := openAISelectionEmptyErrorMetadata(service.OpenAIAccountScheduleDecision{
		Diagnostics: service.OpenAIAccountSelectionDiagnostics{
			Collected:                           true,
			GroupID:                             12,
			Model:                               "gpt-5.3-codex",
			Endpoint:                            "/v1/responses",
			GroupBindingAccountCount:            2,
			ModelSupportedCount:                 0,
			FilterReasonCounts:                  map[string]int{"model_unsupported": 2},
			GroupVisibleModelsKnown:             true,
			GroupVisibleModelsEnabled:           true,
			GroupVisibleModels:                  []string{"gpt-5.3-codex-spark"},
			GroupVisibleModelsCount:             1,
			GroupAvailableModelsKnown:           true,
			GroupAvailableModels:                []string{"gpt-5.3-codex-spark"},
			GroupAvailableModelsCount:           1,
			AccountModelSupportSummaryCount:     2,
			AccountModelSupportSummaryTruncated: false,
			AccountModelSupportSummary:          []service.OpenAIAccountModelSupportDiagnostic{{AccountID: 38865, ModelMappingModels: []string{"gpt-5.3-codex-spark"}}},
		},
	})

	require.NotNil(t, metadata)
	diag, ok := metadata["openai_selection_diagnostics"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-5.3-codex", diag["requested_model"])
	require.Equal(t, []string{"gpt-5.3-codex-spark"}, diag["group_visible_models"])
	require.Equal(t, []string{"gpt-5.3-codex-spark"}, diag["group_available_models"])
	require.Equal(t, 2, diag["account_model_support_summary_count"])
	require.NotContains(t, diag, "account_model_support_summary")
}

func TestOpenAISelectionEmptyErrorResponse_ImageGenerationBridgeUnsupported(t *testing.T) {
	status, errType, message := openAISelectionEmptyErrorResponse(service.OpenAIAccountScheduleDecision{
		Diagnostics: service.OpenAIAccountSelectionDiagnostics{
			Collected:                             true,
			Model:                                 "gpt-5.5",
			RequireCodexImageGenerationBridge:     true,
			GroupBindingAccountCount:              8,
			ActiveSchedulableCount:                8,
			ModelSupportedCount:                   8,
			EndpointSupportedCount:                8,
			ImageGenerationBridgeSupportedCount:   0,
			ImageGenerationBridgeUnsupportedCount: 8,
			FilterReasonCounts:                    map[string]int{"image_generation_bridge_unsupported": 8},
		},
	})

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "image_generation_bridge_not_available", errType)
	require.Contains(t, message, "image_generation")
}
