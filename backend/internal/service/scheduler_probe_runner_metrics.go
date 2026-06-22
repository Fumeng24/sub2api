package service

import "sync/atomic"

type SchedulerProbeRunnerMetricsSnapshot struct {
	Attempts               uint64
	Successes              uint64
	Failures               uint64
	Timeouts               uint64
	ConcurrencyWaitTimeout uint64
}

type schedulerProbeRunnerMetrics struct {
	attempts               atomic.Uint64
	successes              atomic.Uint64
	failures               atomic.Uint64
	timeouts               atomic.Uint64
	concurrencyWaitTimeout atomic.Uint64
}

var defaultSchedulerProbeRunnerMetrics schedulerProbeRunnerMetrics

func GetSchedulerProbeRunnerMetricsSnapshot() SchedulerProbeRunnerMetricsSnapshot {
	return SchedulerProbeRunnerMetricsSnapshot{
		Attempts:               defaultSchedulerProbeRunnerMetrics.attempts.Load(),
		Successes:              defaultSchedulerProbeRunnerMetrics.successes.Load(),
		Failures:               defaultSchedulerProbeRunnerMetrics.failures.Load(),
		Timeouts:               defaultSchedulerProbeRunnerMetrics.timeouts.Load(),
		ConcurrencyWaitTimeout: defaultSchedulerProbeRunnerMetrics.concurrencyWaitTimeout.Load(),
	}
}
