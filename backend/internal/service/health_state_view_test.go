package service

import (
	"testing"
	"time"
)

func TestHealthStateViewRuntimeOpen(t *testing.T) {
	now := time.Now().UTC()
	until := now.Add(time.Minute)
	gateway := &OpenAIGatewayService{}
	gateway.openaiAccountRuntimeBlockUntil.Store(int64(101), until)

	view := openAIAccountRuntimeHealthStateView(gateway, 101, now)
	if view.Allowed {
		t.Fatal("expected runtime-open view to be blocked")
	}
	if view.State != healthStateRuntimeOpen || view.Reason != healthReasonRuntimeCircuitOpen {
		t.Fatalf("state=%q reason=%q", view.State, view.Reason)
	}
	if view.Scope != healthScopeAccount || view.LegacyCircuitState != "runtime_open" {
		t.Fatalf("scope=%q legacy=%q", view.Scope, view.LegacyCircuitState)
	}
	if !view.RetryAt.Equal(until) {
		t.Fatalf("retry_at=%v want %v", view.RetryAt, until)
	}
}

func TestHealthStateViewSchedulerOpen(t *testing.T) {
	now := time.Now().UTC()
	retryAt := now.Add(time.Minute)
	snap := schedulerHealthSnapshot{
		Key:               makeAccountSchedulerHealthKey(102, "gpt-5.5", "/v1/responses"),
		CircuitState:      schedulerCircuitOpen,
		CooldownUntil:     retryAt,
		LastFailureReason: "transient_transport",
	}

	view := schedulerHealthStateViewFromSnapshot(snap, now, false, "")
	if view.Allowed {
		t.Fatal("expected scheduler-open view to be blocked")
	}
	if view.State != healthStateSchedulerOpen || view.Reason != healthReasonSchedulerCircuitOpen {
		t.Fatalf("state=%q reason=%q", view.State, view.Reason)
	}
	if view.Scope != healthScopeAccountModelEndpoint || view.LegacyCircuitState != schedulerCircuitOpen {
		t.Fatalf("scope=%q legacy=%q", view.Scope, view.LegacyCircuitState)
	}
	if view.Model != "gpt-5.5" || view.Endpoint != "/v1/responses" || view.LastFailureReason != "transient_transport" {
		t.Fatalf("model=%q endpoint=%q last_failure=%q", view.Model, view.Endpoint, view.LastFailureReason)
	}
	if !view.RetryAt.Equal(retryAt) {
		t.Fatalf("retry_at=%v want %v", view.RetryAt, retryAt)
	}
}

func TestHealthStateViewHalfOpenInFlight(t *testing.T) {
	now := time.Now().UTC()
	gateway := &OpenAIGatewayService{}
	gateway.openaiAccountCircuitHalfOpen.Store(int64(103), now)

	view := openAIAccountRuntimeHealthStateView(gateway, 103, now)
	if view.Allowed {
		t.Fatal("expected half-open in-flight view to be blocked")
	}
	if view.State != healthStateHalfOpenInFlight || view.Reason != healthReasonRuntimeHalfOpenInFlight {
		t.Fatalf("state=%q reason=%q", view.State, view.Reason)
	}
	if view.Scope != healthScopeAccount || view.LegacyCircuitState != "runtime_half_open" {
		t.Fatalf("scope=%q legacy=%q", view.Scope, view.LegacyCircuitState)
	}
	if !view.RetryAt.IsZero() {
		t.Fatalf("expected no retry_at for in-flight half-open, got %v", view.RetryAt)
	}
}

func TestHealthStateViewProbePending(t *testing.T) {
	now := time.Now().UTC()
	retryAt := now.Add(-time.Second)
	snap := schedulerHealthSnapshot{
		Key:           makeAccountSchedulerHealthKey(104, "gpt-5.5", "/v1/responses"),
		CircuitState:  schedulerCircuitHalfOpen,
		CooldownUntil: retryAt,
		HalfOpenProbe: false,
	}

	view := schedulerHealthStateViewFromSnapshot(snap, now, true, healthReasonSchedulerProbePending)
	if view.Allowed {
		t.Fatal("expected probe-pending view to be blocked")
	}
	if view.State != healthStateProbePending || view.Reason != healthReasonSchedulerProbePending {
		t.Fatalf("state=%q reason=%q", view.State, view.Reason)
	}
	if view.Scope != healthScopeAccountModelEndpoint || view.LegacyCircuitState != schedulerCircuitHalfOpen {
		t.Fatalf("scope=%q legacy=%q", view.Scope, view.LegacyCircuitState)
	}
	if !view.RetryAt.Equal(retryAt) {
		t.Fatalf("retry_at=%v want %v", view.RetryAt, retryAt)
	}
}
