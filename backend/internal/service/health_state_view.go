package service

import (
	"strings"
	"time"
)

const (
	healthStateAvailable        = "available"
	healthStateRuntimeOpen      = "runtime_open"
	healthStateSchedulerOpen    = "scheduler_open"
	healthStateHalfOpenInFlight = "half_open_in_flight"
	healthStateProbePending     = "probe_pending"

	healthScopeAccount              = "account"
	healthScopeAccountModelEndpoint = "account_model_endpoint"

	healthReasonRuntimeCircuitOpen        = "runtime_circuit_open"
	healthReasonRuntimeHalfOpenInFlight   = "runtime_half_open_in_flight"
	healthReasonSchedulerCircuitOpen      = "scheduler_circuit_open"
	healthReasonSchedulerHalfOpen         = "scheduler_half_open"
	healthReasonSchedulerHalfOpenInFlight = "scheduler_half_open_in_flight"
	healthReasonSchedulerProbePending     = "scheduler_probe_pending"
)

type healthStateView struct {
	State              string
	Allowed            bool
	Reason             string
	Scope              string
	Model              string
	Endpoint           string
	RetryAt            time.Time
	LegacyCircuitState string
	LastFailureReason  string
	Snapshot           schedulerHealthSnapshot
}

func closedHealthStateView() healthStateView {
	return healthStateView{
		State:              healthStateAvailable,
		Allowed:            true,
		LegacyCircuitState: schedulerCircuitClosed,
	}
}

func runtimeOpenHealthStateView(retryAt time.Time) healthStateView {
	return healthStateView{
		State:              healthStateRuntimeOpen,
		Allowed:            false,
		Reason:             healthReasonRuntimeCircuitOpen,
		Scope:              healthScopeAccount,
		RetryAt:            retryAt,
		LegacyCircuitState: healthStateRuntimeOpen,
	}
}

func runtimeHalfOpenInFlightHealthStateView() healthStateView {
	return healthStateView{
		State:              healthStateHalfOpenInFlight,
		Allowed:            false,
		Reason:             healthReasonRuntimeHalfOpenInFlight,
		Scope:              healthScopeAccount,
		LegacyCircuitState: "runtime_half_open",
	}
}

func openAIAccountRuntimeHealthStateView(gateway *OpenAIGatewayService, accountID int64, now time.Time) healthStateView {
	if now.IsZero() {
		now = time.Now()
	}
	if gateway == nil || accountID <= 0 {
		return closedHealthStateView()
	}
	if until, ok := gateway.openAIAccountRuntimeBlockUntil(accountID); ok && until.After(now) {
		return runtimeOpenHealthStateView(until)
	}
	if gateway.isOpenAIAccountCircuitHalfOpenInFlight(accountID, now) {
		return runtimeHalfOpenInFlightHealthStateView()
	}
	return closedHealthStateView()
}

func schedulerHealthStateViewFromSnapshot(snap schedulerHealthSnapshot, now time.Time, allowHalfOpen bool, halfOpenBlockedReason string) healthStateView {
	if now.IsZero() {
		now = time.Now()
	}
	view := closedHealthStateView()
	view.Snapshot = snap
	view.Model = snap.Key.Model
	view.Endpoint = snap.Key.Endpoint
	view.LastFailureReason = strings.TrimSpace(snap.LastFailureReason)
	view.LegacyCircuitState = schedulerCircuitClosed

	switch snap.CircuitState {
	case "", schedulerCircuitClosed:
		return view
	case schedulerCircuitOpen:
		if snap.CooldownUntil.IsZero() || snap.CooldownUntil.After(now) {
			view.State = healthStateSchedulerOpen
			view.Allowed = false
			view.Reason = healthReasonSchedulerCircuitOpen
			view.Scope = healthScopeAccountModelEndpoint
			view.RetryAt = snap.CooldownUntil
			view.LegacyCircuitState = schedulerCircuitOpen
			return view
		}
	case schedulerCircuitHalfOpen:
		view.State = schedulerHalfOpenHealthState(halfOpenBlockedReason)
		view.Scope = healthScopeAccountModelEndpoint
		view.LegacyCircuitState = schedulerCircuitHalfOpen
		view.RetryAt = snap.CooldownUntil
		if allowHalfOpen && snap.HalfOpenProbe {
			view.Allowed = true
			view.Reason = ""
			return view
		}
		view.Allowed = false
		if halfOpenBlockedReason == "" {
			halfOpenBlockedReason = healthReasonSchedulerProbePending
		}
		view.Reason = halfOpenBlockedReason
		return view
	default:
		view.State = healthStateSchedulerOpen
		view.Allowed = false
		view.Reason = healthReasonSchedulerCircuitOpen
		view.Scope = healthScopeAccountModelEndpoint
		view.RetryAt = snap.CooldownUntil
		view.LegacyCircuitState = snap.CircuitState
		return view
	}
	return view
}

func schedulerHalfOpenHealthState(reason string) string {
	switch strings.TrimSpace(reason) {
	case healthReasonSchedulerHalfOpenInFlight:
		return healthStateHalfOpenInFlight
	default:
		return healthStateProbePending
	}
}

func (v healthStateView) retryAtPtr(now time.Time) (*time.Time, *int64) {
	if now.IsZero() {
		now = time.Now()
	}
	if v.RetryAt.IsZero() || !v.RetryAt.After(now) {
		return nil, nil
	}
	retryAt := v.RetryAt.UTC()
	var remainingPtr *int64
	if remaining := int64(v.RetryAt.Sub(now).Seconds()); remaining > 0 {
		remainingPtr = &remaining
	}
	return &retryAt, remainingPtr
}
