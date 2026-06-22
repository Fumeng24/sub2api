package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var errSchedulerProbeUnschedulable = errors.New("probe account not active or schedulable")

const (
	defaultSchedulerProbeMaxConcurrency = 10000
	maxSchedulerProbeMaxConcurrency     = 10000
)

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
	limiter    *schedulerProbeLimiter
}

func (r schedulerProbeRunner) run(ctx context.Context, key schedulerProbeKey, initialCategory string) {
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		attempt++
		defaultSchedulerProbeRunnerMetrics.attempts.Add(1)

		probeCtx := ctx
		cancel := func() {}
		if r.timeout > 0 {
			probeCtx, cancel = context.WithTimeout(ctx, r.timeout)
		}
		if r.limiter != nil {
			if !r.limiter.acquire(probeCtx) {
				cancel()
				defaultSchedulerProbeRunnerMetrics.concurrencyWaitTimeout.Add(1)
				slog.Warn("account_circuit_probe_concurrency_wait_timeout",
					append(r.adapter.LogAttrs(key),
						"attempt", attempt,
						"timeout", r.timeout.String(),
						"retry_delay", r.retryDelay.String(),
						"max_concurrency", r.limiter.limit,
					)...,
				)
				if !sleepSchedulerProbeRetry(ctx, r.retryDelay) {
					return
				}
				continue
			}
		}
		statusCode, body, ttftMs, err := r.adapter.Probe(probeCtx, key)
		timedOut := errors.Is(probeCtx.Err(), context.DeadlineExceeded)
		if timedOut {
			defaultSchedulerProbeRunnerMetrics.timeouts.Add(1)
		}
		if r.limiter != nil {
			r.limiter.release()
		}
		cancel()

		if errors.Is(err, errSchedulerProbeUnschedulable) {
			if r.health != nil {
				r.health.clear(key.AccountID, key.Model, key.Endpoint)
			}
			r.adapter.OnUnschedulable(key)
			slog.Info("account_circuit_probe_stopped",
				append(r.adapter.LogAttrs(key),
					"category", "manual_unschedulable",
					"attempt", attempt,
					"timeout", r.timeout.String(),
					"retry_delay", r.retryDelay.String(),
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
				defaultSchedulerProbeRunnerMetrics.successes.Add(1)
				r.adapter.OnRecovered(key)
				slog.Info("account_circuit_probe_recovered",
					append(r.adapter.LogAttrs(key),
						"status_code", statusCode,
						"ttft_ms", ttftMs,
						"attempt", attempt,
						"timeout", r.timeout.String(),
					)...,
				)
				return
			}
			if ttftMs > 0 {
				if r.health != nil {
					r.health.reportFailure(key.AccountID, key.Model, key.Endpoint, schedulerSlowTTFTCategory, schedulerCooldownForCategory(schedulerSlowTTFTCategory, nil))
				}
				defaultSchedulerProbeRunnerMetrics.failures.Add(1)
				slog.Warn("account_circuit_probe_slow_ttft",
					append(r.adapter.LogAttrs(key),
						"status_code", statusCode,
						"ttft_ms", ttftMs,
						"healthy_ttft_ms", schedulerHealthyTTFTThreshold.Milliseconds(),
						"attempt", attempt,
						"timeout", r.timeout.String(),
					)...,
				)
				if !r.adapter.ShouldContinue(key, schedulerSlowTTFTCategory) {
					slog.Info("account_circuit_probe_stopped",
						append(r.adapter.LogAttrs(key),
							"category", schedulerSlowTTFTCategory,
							"attempt", attempt,
							"retry_delay", r.retryDelay.String(),
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
		defaultSchedulerProbeRunnerMetrics.failures.Add(1)

		slog.Warn("account_circuit_probe_failed",
			append(r.adapter.LogAttrs(key),
				"status_code", statusCode,
				"category", category,
				"ttft_ms", ttftMs,
				"attempt", attempt,
				"timed_out", timedOut,
				"timeout", r.timeout.String(),
				"retry_delay", r.retryDelay.String(),
				"error", err,
			)...,
		)

		if !r.adapter.ShouldContinue(key, category) {
			slog.Info("account_circuit_probe_stopped",
				append(r.adapter.LogAttrs(key),
					"category", category,
					"attempt", attempt,
					"retry_delay", r.retryDelay.String(),
				)...,
			)
			return
		}

		if !sleepSchedulerProbeRetry(ctx, r.retryDelay) {
			return
		}
	}
}

type schedulerProbeRunnerConfig struct {
	timeout        time.Duration
	retryDelay     time.Duration
	maxConcurrency int
}

func schedulerProbeRunnerConfigFromConfig(cfg *config.Config, fallbackTimeout, fallbackRetryDelay time.Duration) schedulerProbeRunnerConfig {
	runnerCfg := schedulerProbeRunnerConfig{
		timeout:        fallbackTimeout,
		retryDelay:     fallbackRetryDelay,
		maxConcurrency: defaultSchedulerProbeMaxConcurrency,
	}
	if cfg == nil {
		return runnerCfg
	}
	runtime := cfg.Gateway.OpenAIScheduler.RuntimeCooldowns
	runnerCfg.timeout = schedulerProbeDurationSeconds(runtime.ProbeTimeoutSeconds, fallbackTimeout)
	runnerCfg.retryDelay = schedulerProbeDurationSeconds(runtime.ProbeRetryDelaySeconds, fallbackRetryDelay)
	if runtime.ProbeMaxConcurrency > 0 && runtime.ProbeMaxConcurrency <= maxSchedulerProbeMaxConcurrency {
		runnerCfg.maxConcurrency = runtime.ProbeMaxConcurrency
	}
	return runnerCfg
}

func schedulerProbeDurationSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	maxSeconds := int64(time.Duration(1<<63-1) / time.Second)
	if int64(seconds) > maxSeconds {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

type schedulerProbeLimiter struct {
	limit int
	sem   chan struct{}
}

func newSchedulerProbeLimiter(limit int) *schedulerProbeLimiter {
	if limit <= 0 || limit >= defaultSchedulerProbeMaxConcurrency {
		return nil
	}
	return &schedulerProbeLimiter{limit: limit, sem: make(chan struct{}, limit)}
}

func (l *schedulerProbeLimiter) acquire(ctx context.Context) bool {
	if l == nil {
		return true
	}
	select {
	case l.sem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (l *schedulerProbeLimiter) release() {
	if l == nil {
		return
	}
	select {
	case <-l.sem:
	default:
	}
}

type schedulerProbeLimiterHolder struct {
	mu      sync.Mutex
	limit   int
	limiter *schedulerProbeLimiter
}

func (h *schedulerProbeLimiterHolder) get(limit int) *schedulerProbeLimiter {
	if limit <= 0 || limit >= defaultSchedulerProbeMaxConcurrency {
		return nil
	}
	if h == nil {
		return newSchedulerProbeLimiter(limit)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.limiter == nil || h.limit != limit {
		h.limit = limit
		h.limiter = newSchedulerProbeLimiter(limit)
	}
	return h.limiter
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
