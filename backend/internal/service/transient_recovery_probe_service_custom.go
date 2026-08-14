package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	transientRecoveryInitialDelay = 10 * time.Second
	transientRecoveryLease        = 2 * time.Minute
	transientRecoveryProbeTimeout = monitorRequestTimeout + 5*time.Second
	transientRecoveryConcurrency  = 4
)

type transientRecoveryAccountReader interface {
	GetByID(ctx context.Context, id int64) (*Account, error)
	ListActive(ctx context.Context) ([]Account, error)
}

type transientRecoveryProber interface {
	ProbeTransientRecovery(ctx context.Context, accountID int64, requestedModel string) (*CheckResult, error)
}

type transientRecoveryCooldownController interface {
	RenewTransient5xxCooldown(ctx context.Context, accountID int64, until time.Time) (bool, error)
	ClearTransient5xxCooldown(ctx context.Context, accountID int64) (bool, error)
}

// TransientRecoveryProbeService actively rechecks accounts quarantined by the
// transient 5xx policy. It is separate from persistent admin account monitors:
// a cooldown must be self-healing even when no monitor row exists.
type TransientRecoveryProbeService struct {
	accounts  transientRecoveryAccountReader
	prober    transientRecoveryProber
	cooldowns transientRecoveryCooldownController

	parentCtx    context.Context
	parentCancel context.CancelFunc
	probeSlots   chan struct{}

	mu      sync.Mutex
	tasks   map[int64]*transientRecoveryTask
	wg      sync.WaitGroup
	started bool
	stopped bool
}

type transientRecoveryTask struct {
	accountID      int64
	requestedModel string
	cancel         context.CancelFunc
}

// NewTransientRecoveryProbeService creates the active recovery worker. Start
// is intentionally separate so dependency wiring can install the scheduler on
// RateLimitService before scanning persisted cooldowns.
func NewTransientRecoveryProbeService(accounts AccountRepository, prober *AccountMonitorService, cooldowns *RateLimitService) *TransientRecoveryProbeService {
	return newTransientRecoveryProbeService(accounts, prober, cooldowns)
}

func newTransientRecoveryProbeService(accounts transientRecoveryAccountReader, prober transientRecoveryProber, cooldowns transientRecoveryCooldownController) *TransientRecoveryProbeService {
	ctx, cancel := context.WithCancel(context.Background())
	return &TransientRecoveryProbeService{
		accounts:     accounts,
		prober:       prober,
		cooldowns:    cooldowns,
		parentCtx:    ctx,
		parentCancel: cancel,
		probeSlots:   make(chan struct{}, transientRecoveryConcurrency),
		tasks:        make(map[int64]*transientRecoveryTask),
	}
}

// Start restores persisted transient 5xx episodes before the HTTP server is
// exposed. Expired episodes are probed immediately so they cannot silently
// re-enter the scheduler after a process restart.
func (s *TransientRecoveryProbeService) Start() {
	if s == nil || s.accounts == nil {
		return
	}
	s.mu.Lock()
	if s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), monitorStartupLoadTimeout)
	defer cancel()
	accounts, err := s.accounts.ListActive(ctx)
	if err != nil {
		slog.Error("transient_recovery: load persisted cooldowns failed", "error", err)
		return
	}

	now := time.Now()
	scheduled := 0
	for i := range accounts {
		account := &accounts[i]
		if !isTransientRecoveryProbeAccount(account) || !isCurrentTransient5xxCooldown(account) {
			continue
		}
		delay := transientRecoveryInitialDelay
		if account.TempUnschedulableUntil == nil || !account.TempUnschedulableUntil.After(now.Add(delay)) {
			delay = 0
		}
		if s.schedule(account.ID, "", delay) {
			scheduled++
		}
	}
	slog.Info("transient_recovery: runner started", "scheduled_tasks", scheduled)
}

// ScheduleTransientRecoveryProbe is called from the request error path. It
// only updates an in-memory task table; all database reads and network work
// happen in the background, so applying a cooldown never adds request latency.
func (s *TransientRecoveryProbeService) ScheduleTransientRecoveryProbe(accountID int64, requestedModel string) {
	if s == nil || accountID <= 0 {
		return
	}
	s.schedule(accountID, requestedModel, transientRecoveryInitialDelay)
}

func (s *TransientRecoveryProbeService) schedule(accountID int64, requestedModel string, delay time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped || !s.started || accountID <= 0 {
		return false
	}
	if existing, ok := s.tasks[accountID]; ok {
		if existing.requestedModel == "" && requestedModel != "" {
			existing.requestedModel = requestedModel
		}
		return false
	}

	ctx, cancel := context.WithCancel(s.parentCtx)
	task := &transientRecoveryTask{
		accountID:      accountID,
		requestedModel: requestedModel,
		cancel:         cancel,
	}
	s.tasks[accountID] = task
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runTask(ctx, task, delay)
	}()
	return true
}

func (s *TransientRecoveryProbeService) runTask(ctx context.Context, task *transientRecoveryTask, delay time.Duration) {
	defer func() {
		s.mu.Lock()
		if current, ok := s.tasks[task.accountID]; ok && current == task {
			delete(s.tasks, task.accountID)
		}
		s.mu.Unlock()
	}()

	attempt := 0
	for {
		if !waitTransientRecovery(ctx, delay) {
			return
		}
		retry := s.probeOnce(ctx, task)
		if !retry {
			return
		}
		delay = transientRecoveryRetryDelay(attempt)
		attempt++
	}
}

func (s *TransientRecoveryProbeService) probeOnce(ctx context.Context, task *transientRecoveryTask) bool {
	if s == nil || task == nil || s.accounts == nil || s.prober == nil || s.cooldowns == nil {
		return false
	}
	account, err := s.accounts.GetByID(ctx, task.accountID)
	if err != nil || !isTransientRecoveryProbeAccount(account) || !isCurrentTransient5xxCooldown(account) {
		return false
	}

	// Renew only when the durable lease is close to expiring. A 503's original
	// ten-minute window remains the fallback while early probes run; a 502's
	// shorter window is extended before it can expire during a probe request.
	if account.TempUnschedulableUntil == nil || account.TempUnschedulableUntil.Before(time.Now().Add(transientRecoveryLease)) {
		kept, err := s.cooldowns.RenewTransient5xxCooldown(ctx, task.accountID, time.Now().Add(transientRecoveryLease))
		if err != nil {
			slog.Warn("transient_recovery: cooldown renewal failed", "account_id", task.accountID, "error", err)
			return true
		}
		if !kept {
			return false
		}
	}

	select {
	case s.probeSlots <- struct{}{}:
		defer func() { <-s.probeSlots }()
	case <-ctx.Done():
		return false
	}

	probeCtx, cancel := context.WithTimeout(ctx, transientRecoveryProbeTimeout)
	defer cancel()
	check, err := s.prober.ProbeTransientRecovery(probeCtx, task.accountID, s.requestedModel(task))
	if err != nil {
		// Configuration/credential changes are observed on the next iteration;
		// network failures remain retryable and are represented by the existing
		// fixed cooldown until a probe succeeds.
		slog.Debug("transient_recovery: probe could not start", "account_id", task.accountID, "error", err)
		return true
	}
	if check == nil || !isSuccessfulTransientRecoveryCheck(check.Status) {
		if check != nil {
			slog.Info("transient_recovery: probe failed", "account_id", task.accountID, "status", check.Status)
		}
		return true
	}

	cleared, err := s.cooldowns.ClearTransient5xxCooldown(ctx, task.accountID)
	if err != nil {
		slog.Warn("transient_recovery: cooldown clear failed", "account_id", task.accountID, "error", err)
		return true
	}
	if cleared {
		slog.Info("transient_recovery: account recovered", "account_id", task.accountID, "model", check.Model)
	}
	return false
}

func (s *TransientRecoveryProbeService) requestedModel(task *transientRecoveryTask) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.tasks[task.accountID]; ok && current == task {
		return current.requestedModel
	}
	return task.requestedModel
}

func (s *TransientRecoveryProbeService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.parentCancel()
	for _, task := range s.tasks {
		task.cancel()
	}
	s.tasks = nil
	s.mu.Unlock()
	s.wg.Wait()
}

func isTransientRecoveryProbeAccount(account *Account) bool {
	return account != nil &&
		account.Status == StatusActive &&
		account.Schedulable &&
		account.Type == AccountTypeAPIKey &&
		account.GetCredential("api_key") != ""
}

func isSuccessfulTransientRecoveryCheck(status string) bool {
	return status == MonitorStatusOperational || status == MonitorStatusDegraded
}

func transientRecoveryRetryDelay(attempt int) time.Duration {
	switch attempt {
	case 0:
		return 20 * time.Second
	case 1:
		return 40 * time.Second
	default:
		return time.Minute
	}
}

func waitTransientRecovery(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
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
