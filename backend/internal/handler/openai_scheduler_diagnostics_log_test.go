package handler

import (
	"errors"
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
