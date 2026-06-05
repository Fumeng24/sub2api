package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCollectSelectionFailureStats(t *testing.T) {
	svc := &GatewayService{}
	model := "gpt-5.4"
	resetAt := time.Now().Add(2 * time.Minute).Format(time.RFC3339)

	accounts := []*Account{
		// excluded
		{
			ID:          1,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
		// unschedulable
		{
			ID:          2,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: false,
		},
		// platform filtered
		{
			ID:          3,
			Platform:    PlatformAntigravity,
			Status:      StatusActive,
			Schedulable: true,
		},
		// model unsupported
		{
			ID:          4,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-image": "gpt-image",
				},
			},
		},
		// model rate limited
		{
			ID:          5,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"model_rate_limits": map[string]any{
					model: map[string]any{
						"rate_limit_reset_at": resetAt,
					},
				},
			},
		},
		// eligible
		{
			ID:          6,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
	}

	excluded := map[int64]struct{}{1: {}}
	stats := svc.collectSelectionFailureStats(context.Background(), accounts, model, PlatformOpenAI, excluded, false)

	if stats.Total != 6 {
		t.Fatalf("total=%d want=6", stats.Total)
	}
	if stats.Excluded != 1 {
		t.Fatalf("excluded=%d want=1", stats.Excluded)
	}
	if stats.Unschedulable != 1 {
		t.Fatalf("unschedulable=%d want=1", stats.Unschedulable)
	}
	if stats.PlatformFiltered != 1 {
		t.Fatalf("platform_filtered=%d want=1", stats.PlatformFiltered)
	}
	if stats.ModelUnsupported != 1 {
		t.Fatalf("model_unsupported=%d want=1", stats.ModelUnsupported)
	}
	if stats.ModelRateLimited != 1 {
		t.Fatalf("model_rate_limited=%d want=1", stats.ModelRateLimited)
	}
	if stats.Eligible != 1 {
		t.Fatalf("eligible=%d want=1", stats.Eligible)
	}
}

func TestDiagnoseSelectionFailure_UnschedulableDetail(t *testing.T) {
	svc := &GatewayService{}
	acc := &Account{
		ID:          7,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: false,
	}

	diagnosis := svc.diagnoseSelectionFailure(context.Background(), acc, "gpt-5.4", PlatformOpenAI, map[int64]struct{}{}, false)
	if diagnosis.Category != "unschedulable" {
		t.Fatalf("category=%s want=unschedulable", diagnosis.Category)
	}
	if diagnosis.Detail != "generic_unschedulable" {
		t.Fatalf("detail=%s want=generic_unschedulable", diagnosis.Detail)
	}
}

func TestDiagnoseSelectionFailure_ModelRateLimitedDetail(t *testing.T) {
	svc := &GatewayService{}
	model := "gpt-5.4"
	resetAt := time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
	acc := &Account{
		ID:          8,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				model: map[string]any{
					"rate_limit_reset_at": resetAt,
				},
			},
		},
	}

	diagnosis := svc.diagnoseSelectionFailure(context.Background(), acc, model, PlatformOpenAI, map[int64]struct{}{}, false)
	if diagnosis.Category != "model_rate_limited" {
		t.Fatalf("category=%s want=model_rate_limited", diagnosis.Category)
	}
	if !strings.Contains(diagnosis.Detail, "remaining=") {
		t.Fatalf("detail=%s want contains remaining=", diagnosis.Detail)
	}
}

func TestGatewaySelectionDiagnostics_CircuitOpen(t *testing.T) {
	groupID := int64(11)
	model := "claude-opus-4-7"
	endpoint := "/v1/messages"
	account := &Account{
		ID:            38800,
		Platform:      PlatformAnthropic,
		Status:        StatusActive,
		Schedulable:   true,
		Concurrency:   5,
		AccountGroups: []AccountGroup{{GroupID: groupID}},
		Credentials: map[string]any{
			"model_mapping": map[string]any{model: model},
		},
	}
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	svc.schedulerHealth.reportFailure(account.ID, model, endpoint, "transient_transport", time.Minute)

	err := svc.newGatewayNoAvailableError(
		context.Background(),
		&groupID,
		model,
		PlatformAnthropic,
		endpoint,
		[]*Account{account},
		nil,
		false,
		&Group{ID: groupID, Platform: PlatformAnthropic, Status: StatusActive},
		false,
		map[int64]*AccountLoadInfo{account.ID: {AccountID: account.ID, LoadRate: 0}},
	)

	var noAvailable *GatewayNoAvailableAccountsError
	if !errors.As(err, &noAvailable) {
		t.Fatalf("expected GatewayNoAvailableAccountsError, got %T", err)
	}
	diag := noAvailable.Diagnostics
	if !diag.Collected {
		t.Fatal("expected diagnostics to be collected")
	}
	if diag.ModelSupportedCount != 1 || diag.EndpointSupportedCount != 1 || diag.StateAllowedCount != 1 {
		t.Fatalf("unexpected pre-circuit counts: model=%d endpoint=%d state=%d", diag.ModelSupportedCount, diag.EndpointSupportedCount, diag.StateAllowedCount)
	}
	if diag.CircuitAllowedCount != 0 || diag.FinalCandidateCount != 0 {
		t.Fatalf("expected circuit to remove final candidate, circuit_allowed=%d final=%d", diag.CircuitAllowedCount, diag.FinalCandidateCount)
	}
	if len(diag.CircuitFilteredAccountIDs) != 1 || diag.CircuitFilteredAccountIDs[0] != account.ID {
		t.Fatalf("expected account %d circuit-filtered, got %v", account.ID, diag.CircuitFilteredAccountIDs)
	}
	if diag.FilterReasonCounts["scheduler_circuit_open"] != 1 {
		t.Fatalf("expected scheduler_circuit_open skip reason, got %v", diag.FilterReasonCounts)
	}
	if len(diag.SkippedAccounts) != 1 || diag.SkippedAccounts[0].CircuitState != schedulerCircuitOpen || diag.SkippedAccounts[0].CircuitEndpoint != endpoint {
		t.Fatalf("expected scoped circuit details in skipped account, got %+v", diag.SkippedAccounts)
	}
}

func TestGatewayServiceReportAccountScheduleSuccessForRequestClearsContextEndpoint(t *testing.T) {
	accountID := int64(38800)
	model := "claude-opus-4-7"
	explicitEndpoint := "anthropic"
	contextEndpoint := "/v1/messages"
	ctx := WithSchedulerEndpoint(context.Background(), contextEndpoint)
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}

	failoverErr := &UpstreamFailoverError{
		StatusCode:   0,
		ResponseBody: []byte("openai_request_error: context canceled"),
	}
	svc.ReportAccountScheduleFailure(ctx, accountID, model, explicitEndpoint, failoverErr)

	if snap := svc.schedulerHealth.snapshot(accountID, model, explicitEndpoint, true); snap.CircuitState != schedulerCircuitOpen {
		t.Fatalf("explicit endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitOpen)
	}
	if snap := svc.schedulerHealth.snapshot(accountID, model, contextEndpoint, true); snap.CircuitState != schedulerCircuitOpen {
		t.Fatalf("context endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitOpen)
	}

	firstTokenMs := 42
	svc.ReportAccountScheduleSuccessForRequest(ctx, accountID, model, explicitEndpoint, &firstTokenMs)

	if snap := svc.schedulerHealth.snapshot(accountID, model, explicitEndpoint, true); snap.CircuitState != schedulerCircuitClosed {
		t.Fatalf("explicit endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitClosed)
	}
	if snap := svc.schedulerHealth.snapshot(accountID, model, contextEndpoint, true); snap.CircuitState != schedulerCircuitClosed {
		t.Fatalf("context endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitClosed)
	}
}
