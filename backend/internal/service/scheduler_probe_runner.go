package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

var errSchedulerProbeUnschedulable = errors.New("probe account not active or schedulable")

type schedulerProbeKey = accountSchedulerHealthKey

type schedulerProbeAdapter interface {
	Probe(ctx context.Context, key schedulerProbeKey) (statusCode int, body []byte, ttftMs int, err error)
	OnRecovered(key schedulerProbeKey)
	OnUnschedulable(key schedulerProbeKey)
	ShouldContinue(key schedulerProbeKey, category string) bool
	LogAttrs(key schedulerProbeKey) []any
}

type schedulerProbeRunner struct {
	health     *accountSchedulerHealthStats
	classifier schedulerErrorClassifier
	adapter    schedulerProbeAdapter
	timeout    time.Duration
	retryDelay time.Duration
}

func (r schedulerProbeRunner) run(ctx context.Context, key schedulerProbeKey, initialCategory string) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		probeCtx := ctx
		cancel := func() {}
		if r.timeout > 0 {
			probeCtx, cancel = context.WithTimeout(ctx, r.timeout)
		}
		statusCode, body, ttftMs, err := r.adapter.Probe(probeCtx, key)
		cancel()

		if errors.Is(err, errSchedulerProbeUnschedulable) {
			if r.health != nil {
				r.health.clear(key.AccountID, key.Model, key.Endpoint)
			}
			r.adapter.OnUnschedulable(key)
			slog.Info("account_circuit_probe_stopped",
				append(r.adapter.LogAttrs(key),
					"category", "manual_unschedulable",
					"error", err,
				)...,
			)
			return
		}

		if err == nil && statusCode >= 200 && statusCode < 400 {
			if schedulerTTFTWithinHealthyThreshold(ttftMs) {
				if r.health != nil {
					r.health.clear(key.AccountID, key.Model, key.Endpoint)
				}
				r.adapter.OnRecovered(key)
				slog.Info("account_circuit_probe_recovered",
					append(r.adapter.LogAttrs(key),
						"status_code", statusCode,
						"ttft_ms", ttftMs,
					)...,
				)
				return
			}
			if ttftMs > 0 {
				if r.health != nil {
					r.health.reportFailure(key.AccountID, key.Model, key.Endpoint, schedulerSlowTTFTCategory, schedulerCooldownForCategory(schedulerSlowTTFTCategory, nil))
				}
				slog.Warn("account_circuit_probe_slow_ttft",
					append(r.adapter.LogAttrs(key),
						"status_code", statusCode,
						"ttft_ms", ttftMs,
						"healthy_ttft_ms", schedulerHealthyTTFTThreshold.Milliseconds(),
					)...,
				)
				if !r.adapter.ShouldContinue(key, schedulerSlowTTFTCategory) {
					slog.Info("account_circuit_probe_stopped",
						append(r.adapter.LogAttrs(key),
							"category", schedulerSlowTTFTCategory,
						)...,
					)
					return
				}
				if !sleepSchedulerProbeRetry(ctx, r.retryDelay) {
					return
				}
				continue
			}
			err = fmt.Errorf("probe response did not produce first token")
		}

		category := r.failureCategory(statusCode, body, err, ttftMs)
		if category == "" || category == "unknown" {
			category = strings.TrimSpace(initialCategory)
			if category == "" || category == "unknown" {
				category = "error"
			}
		}
		if r.health != nil {
			r.health.reportFailure(key.AccountID, key.Model, key.Endpoint, category, schedulerCooldownForCategory(category, nil))
		}

		slog.Warn("account_circuit_probe_failed",
			append(r.adapter.LogAttrs(key),
				"status_code", statusCode,
				"category", category,
				"ttft_ms", ttftMs,
				"error", err,
			)...,
		)

		if !r.adapter.ShouldContinue(key, category) {
			slog.Info("account_circuit_probe_stopped",
				append(r.adapter.LogAttrs(key),
					"category", category,
				)...,
			)
			return
		}

		if !sleepSchedulerProbeRetry(ctx, r.retryDelay) {
			return
		}
	}
}

func (r schedulerProbeRunner) failureCategory(statusCode int, body []byte, err error, ttftMs int) string {
	if err != nil && statusCode >= 200 && statusCode < 400 && ttftMs <= 0 {
		return schedulerSlowTTFTCategory
	}
	if err != nil && statusCode == 0 {
		return schedulerFailureCategory(0, []byte(err.Error()))
	}
	classifier := r.classifier
	if classifier == nil {
		classifier = defaultSchedulerErrorClassifier{}
	}
	return classifier.Classify(statusCode, body, nil)
}

func sleepSchedulerProbeRetry(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
