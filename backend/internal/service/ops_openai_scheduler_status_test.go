package service

import (
	"context"
	"testing"
	"time"
)

func TestOpsOpenAISchedulerStatusManualUnschedulableDoesNotShowCircuit(t *testing.T) {
	account := &Account{
		ID:          38549,
		Name:        "manual-off",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: false,
	}
	gateway := &OpenAIGatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	gateway.schedulerHealth.reportFailure(account.ID, "gpt-5.4", "/v1/responses", "transient_transport", time.Minute)
	ops := &OpsService{openAIGatewayService: gateway}

	item := ops.openAISchedulerAccountStatus(
		context.Background(),
		account,
		&Group{ID: 10, Name: "Codex", Platform: PlatformOpenAI},
		openAISchedulerStatusRequest("gpt-5.4", "/v1/responses"),
		nil,
		time.Now().UTC(),
	)

	if item.StateAllowed {
		t.Fatal("expected manual unschedulable account to be state filtered")
	}
	if item.StateReason != "manual_unschedulable" || item.BlockReason != "manual_unschedulable" {
		t.Fatalf("expected manual unschedulable display, got state_reason=%q block_reason=%q", item.StateReason, item.BlockReason)
	}
	if item.CircuitState != schedulerCircuitClosed || item.CircuitReason != "" || item.CircuitScope != "" {
		t.Fatalf("manual unschedulable account must not be overlaid by scoped circuit, got state=%q reason=%q scope=%q", item.CircuitState, item.CircuitReason, item.CircuitScope)
	}
}

func TestOpsOpenAISchedulerStatusShowsScopedCircuitDetails(t *testing.T) {
	account := &Account{
		ID:          38805,
		Name:        "shared",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
	}
	gateway := &OpenAIGatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	gateway.schedulerHealth.reportFailure(account.ID, "gpt-5.4", "/v1/responses", "transient_transport", time.Minute)
	ops := &OpsService{openAIGatewayService: gateway}

	item := ops.openAISchedulerAccountStatus(
		context.Background(),
		account,
		&Group{ID: 10, Name: "Codex", Platform: PlatformOpenAI},
		openAISchedulerStatusRequest("gpt-5.4", "/v1/responses"),
		nil,
		time.Now().UTC(),
	)

	if item.CircuitState != schedulerCircuitOpen {
		t.Fatalf("expected scoped circuit open, got %q", item.CircuitState)
	}
	if item.CircuitScope != "account_model_endpoint" || item.CircuitModel != "gpt-5.4" || item.CircuitEndpoint != "/v1/responses" {
		t.Fatalf("expected scoped circuit details, got scope=%q model=%q endpoint=%q", item.CircuitScope, item.CircuitModel, item.CircuitEndpoint)
	}
	if item.SchedulerLastFailureReason != "transient_transport" {
		t.Fatalf("expected transient transport detail, got %q", item.SchedulerLastFailureReason)
	}
	if item.BlockReason != "scheduler_circuit_open" {
		t.Fatalf("expected scheduler circuit block reason, got %q", item.BlockReason)
	}
}
