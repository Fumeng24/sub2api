package service

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
)

// AccountMonitorRunner 账号监控调度器。
//
// 设计与 ChannelMonitorRunner 一致（per-monitor goroutine + timer + jitter + pond 池 +
// in-flight 去重），但完全独立：自己的池、任务表、生命周期。复用同包的并发/超时常量。
// 功能开关复用 channel-monitor 的运行时开关（账号监控随渠道监控总开关一起启停）。
type accountMonitorRunnerSvc interface {
	ListEnabledMonitors(ctx context.Context) ([]*AccountMonitor, error)
	RunCheck(ctx context.Context, id int64) (*AccountMonitorCheck, error)
}

type AccountMonitorRunner struct {
	svc            accountMonitorRunnerSvc
	settingService *SettingService

	pool         pond.Pool
	parentCtx    context.Context
	parentCancel context.CancelFunc

	mu      sync.Mutex
	tasks   map[int64]*scheduledAccountMonitor
	wg      sync.WaitGroup
	started bool
	stopped bool

	inFlight   map[int64]struct{}
	inFlightMu sync.Mutex
}

// scheduledAccountMonitor 单个账号监控的运行时上下文。
type scheduledAccountMonitor struct {
	id       int64
	interval time.Duration
	jitter   time.Duration
	cancel   context.CancelFunc
}

func (t *scheduledAccountMonitor) nextDelay() time.Duration {
	if t.jitter <= 0 {
		return t.interval
	}
	offset := time.Duration(rand.Int64N(int64(2*t.jitter) + 1))
	d := t.interval - t.jitter + offset
	if floor := monitorMinIntervalSeconds * time.Second; d < floor {
		d = floor
	}
	return d
}

// NewAccountMonitorRunner 构造调度器。Start 在 wire 中调用一次。
func NewAccountMonitorRunner(svc *AccountMonitorService, settingService *SettingService) *AccountMonitorRunner {
	return newAccountMonitorRunner(svc, settingService)
}

func newAccountMonitorRunner(svc accountMonitorRunnerSvc, settingService *SettingService) *AccountMonitorRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &AccountMonitorRunner{
		svc:            svc,
		settingService: settingService,
		pool:           pond.NewPool(monitorWorkerConcurrency),
		parentCtx:      ctx,
		parentCancel:   cancel,
		tasks:          make(map[int64]*scheduledAccountMonitor),
		inFlight:       make(map[int64]struct{}),
	}
}

// Start 加载所有 enabled monitor 并为每个建立独立定时任务（只调一次）。
func (r *AccountMonitorRunner) Start() {
	if r == nil || r.svc == nil {
		return
	}
	r.mu.Lock()
	if r.started || r.stopped {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), monitorStartupLoadTimeout)
	defer cancel()
	enabled, err := r.svc.ListEnabledMonitors(ctx)
	if err != nil {
		slog.Error("account_monitor: load enabled monitors failed at startup", "error", err)
		return
	}
	for _, m := range enabled {
		r.Schedule(m)
	}
	slog.Info("account_monitor: runner started", "scheduled_tasks", len(enabled))
}

// Schedule 为指定监控创建（或重置）独立定时任务。m.Enabled=false 等同 Unschedule。
func (r *AccountMonitorRunner) Schedule(m *AccountMonitor) {
	if r == nil || m == nil {
		return
	}
	if !m.Enabled {
		r.Unschedule(m.ID)
		return
	}
	interval := time.Duration(m.IntervalSeconds) * time.Second
	if interval <= 0 {
		slog.Error("account_monitor: skip schedule for invalid interval",
			"monitor_id", m.ID, "interval_seconds", m.IntervalSeconds)
		return
	}
	jitter := time.Duration(m.JitterSeconds) * time.Second
	if jitter < 0 {
		jitter = 0
	}

	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	if !r.started {
		r.mu.Unlock()
		slog.Warn("account_monitor: schedule before runner started, skip", "monitor_id", m.ID)
		return
	}
	if existing, ok := r.tasks[m.ID]; ok {
		existing.cancel()
	}
	ctx, cancel := context.WithCancel(r.parentCtx)
	task := &scheduledAccountMonitor{
		id:       m.ID,
		interval: interval,
		jitter:   jitter,
		cancel:   cancel,
	}
	r.tasks[m.ID] = task
	r.wg.Add(1)
	r.mu.Unlock()

	go r.runScheduled(ctx, task)
}

// Unschedule 取消指定监控的定时任务（若存在）。
func (r *AccountMonitorRunner) Unschedule(id int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	task, ok := r.tasks[id]
	if ok {
		delete(r.tasks, id)
	}
	r.mu.Unlock()
	if ok {
		task.cancel()
	}
}

// Stop 优雅停止：取消所有任务、关闭池。
func (r *AccountMonitorRunner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	r.parentCancel()
	r.tasks = nil
	r.mu.Unlock()

	r.wg.Wait()
	r.pool.StopAndWait()
}

func (r *AccountMonitorRunner) runScheduled(ctx context.Context, task *scheduledAccountMonitor) {
	defer r.wg.Done()

	r.fire(ctx, task)

	timer := time.NewTimer(task.nextDelay())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			r.fire(ctx, task)
			timer.Reset(task.nextDelay())
		}
	}
}

// fire 提交一次探测到 worker 池。功能开关关闭时跳过本次（账号监控随渠道监控总开关启停）。
func (r *AccountMonitorRunner) fire(ctx context.Context, task *scheduledAccountMonitor) {
	if r.settingService != nil && !r.settingService.GetChannelMonitorRuntime(ctx).Enabled {
		return
	}
	if !r.tryAcquireInFlight(task.id) {
		slog.Debug("account_monitor: skip already in-flight", "monitor_id", task.id)
		return
	}
	if _, ok := r.pool.TrySubmit(func() {
		r.runOne(task.id)
	}); !ok {
		r.releaseInFlight(task.id)
		slog.Warn("account_monitor: worker pool full, skip submission", "monitor_id", task.id)
	}
}

func (r *AccountMonitorRunner) tryAcquireInFlight(id int64) bool {
	r.inFlightMu.Lock()
	defer r.inFlightMu.Unlock()
	if _, exists := r.inFlight[id]; exists {
		return false
	}
	r.inFlight[id] = struct{}{}
	return true
}

func (r *AccountMonitorRunner) releaseInFlight(id int64) {
	r.inFlightMu.Lock()
	delete(r.inFlight, id)
	r.inFlightMu.Unlock()
}

func (r *AccountMonitorRunner) runOne(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), monitorRequestTimeout+monitorPingTimeout+monitorRunOneBuffer)
	defer cancel()

	defer r.releaseInFlight(id)

	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("account_monitor: runner panic", "monitor_id", id, "panic", rec)
		}
	}()

	if _, err := r.svc.RunCheck(ctx, id); err != nil {
		slog.Warn("account_monitor: run check failed", "monitor_id", id, "error", err)
	}
}
