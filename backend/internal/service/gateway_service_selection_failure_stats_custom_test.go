package service

import (
	"context"
	"testing"
	"time"
)

func TestGatewayHardSelectionReason_TreatsRuntimeWindowsAsEligible(t *testing.T) {
	svc := &GatewayService{}
	future := time.Now().Add(2 * time.Minute)
	acc := &Account{ID: 9, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, RateLimitResetAt: &future, OverloadUntil: &future, TempUnschedulableUntil: &future}
	if reason := svc.gatewayHardSelectionReason(context.Background(), acc, nil, "gpt-5.4", PlatformOpenAI, false, nil, false); reason != "" {
		t.Fatalf("reason=%s want empty", reason)
	}
}

func TestBuildGatewaySelectionDiagnostics_UsesUnifiedSelectionReason(t *testing.T) {
	svc := &GatewayService{}
	acc := &Account{ID: 10, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Extra: map[string]any{"quota_limit": 10.0, "quota_used": 10.0}}
	diag := svc.buildGatewaySelectionDiagnostics(context.Background(), nil, "", PlatformOpenAI, "", []*Account{acc}, nil, false, nil, false, nil)
	if diag.StateAllowedCount != 0 || len(diag.StateFilteredAccountIDs) != 1 || diag.StateFilteredAccountIDs[0] != acc.ID || diag.FilterReasonCounts[AccountSchedulingBlockQuotaExceeded.String()] != 1 {
		t.Fatalf("unexpected diagnostics: %+v", diag)
	}
}

func TestBuildGatewaySelectionDiagnostics_StateReasonWinsOverDimensionCounters(t *testing.T) {
	svc := &GatewayService{}
	acc := &Account{ID: 11, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: false, Credentials: map[string]any{"model_mapping": map[string]any{"other-model": "other-model"}}}
	diag := svc.buildGatewaySelectionDiagnostics(context.Background(), nil, "gpt-5.4", PlatformAntigravity, "", []*Account{acc}, nil, false, nil, false, nil)
	if diag.FilterReasonCounts[AccountSchedulingBlockManual.String()] != 1 || diag.FilterReasonCounts["model_unsupported"] != 0 {
		t.Fatalf("unexpected reason counts: %+v", diag.FilterReasonCounts)
	}
	if len(diag.ModelUnsupportedAccountIDs) != 1 || diag.ModelUnsupportedAccountIDs[0] != acc.ID || len(diag.EndpointUnsupportedAccountIDs) != 1 || diag.EndpointUnsupportedAccountIDs[0] != acc.ID {
		t.Fatalf("unexpected dimension diagnostics: %+v", diag)
	}
}
