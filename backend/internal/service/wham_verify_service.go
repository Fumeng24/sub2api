package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Lightweight bulk account verify via ChatGPT backend wham/usage endpoint.
// Does not consume quota, only reports rate-limit windows and plan_type.
// Results are written back to account Extra so that the existing frontend
// usage columns (codex_5h_used_percent, codex_7d_used_percent, etc.) stay
// in sync after bulk runs.
//
// Multi-instance caveat: job state is in-memory on the instance that started
// it. If the admin panel is fronted by a load balancer, the poll request must
// land on the same backend instance or it returns 404. Deploy behind a single
// instance or add sticky-session routing before using this feature at scale.

const (
	whamUsageURL          = "https://chatgpt.com/backend-api/wham/usage"
	whamVerifyUA          = "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal"
	whamVerifyDefaultConc = 32
	whamVerifyMaxConc     = 128
	whamVerifyHTTPTimeout = 15 * time.Second
	whamVerifyJobTTL      = 1 * time.Hour
	whamWindow5hSeconds   = 5 * 60 * 60
	whamWindow7dSeconds   = 7 * 24 * 60 * 60
	whamErrorMsgMax       = 500
)

// Status bucket labels — shared by service, handler, and i18n.
const (
	whamStatusHigh      = "high"
	whamStatusMedium    = "medium"
	whamStatusLow       = "low"
	whamStatusExhausted = "exhausted"
	whamStatusExpired   = "expired"
	whamStatusMissing   = "missing"
	whamStatusUnknown   = "unknown"
	whamStatusError     = "error"
)

// ErrJobAlreadyRunning is returned when a second job is started while one is
// still in flight. Handler maps this to HTTP 409.
var ErrJobAlreadyRunning = errors.New("bulk-verify job already running")

// WhamVerifyRequest is the start-job payload.
type WhamVerifyRequest struct {
	Concurrency int     `json:"concurrency"`
	GroupIDs    []int64 `json:"group_ids"`
}

// whamVerifyJob is the internal state holder. Concurrent fields are touched
// via atomics; struct fields (Running, Cancelled, FinishedAt) are read/written
// under WhamVerifyService.mu. Apply targets are guarded by targetsMu.
type whamVerifyJob struct {
	ID        string
	Total     int32
	Done      int32
	High      int32
	Medium    int32
	Low       int32
	Exhausted int32
	Expired   int32
	Missing   int32
	Unknown   int32
	Errors    int32

	startedAt time.Time

	// Fields guarded by WhamVerifyService.mu:
	running    bool
	cancelled  bool
	finishedAt *time.Time
	cancel     context.CancelFunc

	// Populated during verify, read during Apply. Guarded by targetsMu so
	// workers don't race on slice/map growth.
	targetsMu        sync.Mutex
	expiredIDs       []int64
	exhaustedTargets map[int64]time.Time // id -> 7d reset_at (empty if no reset data)
	applying         bool                // guards re-entrant Apply
}

// WhamVerifyJobSnapshot is the external, immutable view of job state. Returned
// by Start/Get; safe to serialize concurrently with worker goroutines.
type WhamVerifyJobSnapshot struct {
	ID                 string     `json:"id"`
	Total              int32      `json:"total"`
	Done               int32      `json:"done"`
	High               int32      `json:"high"`
	Medium             int32      `json:"medium"`
	Low                int32      `json:"low"`
	Exhausted          int32      `json:"exhausted"`
	Expired            int32      `json:"expired"`
	Missing            int32      `json:"missing"`
	Unknown            int32      `json:"unknown"`
	Error              int32      `json:"error"`
	Running            bool       `json:"running"`
	Cancelled          bool       `json:"cancelled"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	RefreshableExpired int        `json:"refreshable_expired"` // accounts pending refresh via Apply
	MarkableExhausted  int        `json:"markable_exhausted"`  // exhausted accounts with known reset_at
}

// WhamVerifyService owns the worker pool and job registry for bulk verify.
type WhamVerifyService struct {
	accountRepo AccountRepository
	openaiOAuth *OpenAIOAuthService // for token refresh during Apply
	httpClient  *http.Client

	mu           sync.Mutex
	jobs         map[string]*whamVerifyJob
	stopCleanup  chan struct{}
	cleanupOnce  sync.Once
	shutdownOnce sync.Once
}

// NewWhamVerifyService constructs the service and starts a janitor goroutine.
func NewWhamVerifyService(accountRepo AccountRepository, openaiOAuth *OpenAIOAuthService) *WhamVerifyService {
	s := &WhamVerifyService{
		accountRepo: accountRepo,
		openaiOAuth: openaiOAuth,
		httpClient:  &http.Client{Timeout: whamVerifyHTTPTimeout},
		jobs:        make(map[string]*whamVerifyJob),
		stopCleanup: make(chan struct{}),
	}
	s.cleanupOnce.Do(func() { go s.cleanupLoop() })
	return s
}

// Shutdown stops the janitor goroutine. Does not cancel running jobs.
func (s *WhamVerifyService) Shutdown() {
	s.shutdownOnce.Do(func() { close(s.stopCleanup) })
}

// Start enumerates eligible accounts, spawns the worker pool, and returns a
// snapshot. Returns ErrJobAlreadyRunning if another job is still in flight.
func (s *WhamVerifyService) Start(ctx context.Context, req *WhamVerifyRequest) (*WhamVerifyJobSnapshot, error) {
	if req == nil {
		req = &WhamVerifyRequest{}
	}

	// Reject if another job is already running.
	s.mu.Lock()
	for _, existing := range s.jobs {
		if existing.running {
			snap := s.snapshotLocked(existing)
			s.mu.Unlock()
			return snap, ErrJobAlreadyRunning
		}
	}
	s.mu.Unlock()

	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	groupFilter := make(map[int64]struct{}, len(req.GroupIDs))
	for _, g := range req.GroupIDs {
		if g > 0 {
			groupFilter[g] = struct{}{}
		}
	}

	targets := make([]Account, 0, len(accounts))
	for _, a := range accounts {
		if a.Type != AccountTypeOAuth {
			continue
		}
		if len(groupFilter) > 0 && !accountMatchesGroups(&a, groupFilter) {
			continue
		}
		targets = append(targets, a)
	}

	conc := req.Concurrency
	if conc <= 0 {
		conc = whamVerifyDefaultConc
	}
	if conc > whamVerifyMaxConc {
		conc = whamVerifyMaxConc
	}

	job := &whamVerifyJob{
		ID:               uuid.NewString(),
		Total:            int32(len(targets)),
		startedAt:        time.Now(),
		running:          true,
		exhaustedTargets: make(map[int64]time.Time),
	}
	jobCtx, cancel := context.WithCancel(context.Background())
	job.cancel = cancel

	s.mu.Lock()
	// Double-check nothing raced past the first guard.
	for _, existing := range s.jobs {
		if existing.running {
			s.mu.Unlock()
			cancel()
			return s.Snapshot(existing.ID), ErrJobAlreadyRunning
		}
	}
	s.jobs[job.ID] = job
	s.mu.Unlock()

	go s.runJob(jobCtx, job, targets, conc)
	return s.Snapshot(job.ID), nil
}

// Snapshot returns a snapshot of the job (nil if not found).
func (s *WhamVerifyService) Snapshot(jobID string) *WhamVerifyJobSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return nil
	}
	return s.snapshotLocked(job)
}

// snapshotLocked captures job state while s.mu is held.
func (s *WhamVerifyService) snapshotLocked(job *whamVerifyJob) *WhamVerifyJobSnapshot {
	job.targetsMu.Lock()
	refreshable := len(job.expiredIDs)
	markable := 0
	for _, resetAt := range job.exhaustedTargets {
		if !resetAt.IsZero() {
			markable++
		}
	}
	job.targetsMu.Unlock()

	return &WhamVerifyJobSnapshot{
		ID:                 job.ID,
		Total:              job.Total,
		Done:               atomic.LoadInt32(&job.Done),
		High:               atomic.LoadInt32(&job.High),
		Medium:             atomic.LoadInt32(&job.Medium),
		Low:                atomic.LoadInt32(&job.Low),
		Exhausted:          atomic.LoadInt32(&job.Exhausted),
		Expired:            atomic.LoadInt32(&job.Expired),
		Missing:            atomic.LoadInt32(&job.Missing),
		Unknown:            atomic.LoadInt32(&job.Unknown),
		Error:              atomic.LoadInt32(&job.Errors),
		Running:            job.running,
		Cancelled:          job.cancelled,
		StartedAt:          job.startedAt,
		FinishedAt:         job.finishedAt,
		RefreshableExpired: refreshable,
		MarkableExhausted:  markable,
	}
}

// Cancel requests the job to stop; returns false if job does not exist.
func (s *WhamVerifyService) Cancel(jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return false
	}
	if job.running {
		job.cancelled = true
		if job.cancel != nil {
			job.cancel()
		}
	}
	return true
}

func (s *WhamVerifyService) runJob(ctx context.Context, job *whamVerifyJob, targets []Account, concurrency int) {
	defer func() {
		now := time.Now()
		s.mu.Lock()
		job.finishedAt = &now
		job.running = false
		s.mu.Unlock()
	}()

	logger.L().Info("wham_verify: job started",
		zap.String("job_id", job.ID),
		zap.Int32("total", job.Total),
		zap.Int("concurrency", concurrency),
	)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := range targets {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		acc := targets[i]
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.verifyAccount(ctx, job, &acc)
		}()
	}
	wg.Wait()

	logger.L().Info("wham_verify: job finished",
		zap.String("job_id", job.ID),
		zap.Int32("total", job.Total),
		zap.Int32("done", atomic.LoadInt32(&job.Done)),
		zap.Int32("high", atomic.LoadInt32(&job.High)),
		zap.Int32("medium", atomic.LoadInt32(&job.Medium)),
		zap.Int32("low", atomic.LoadInt32(&job.Low)),
		zap.Int32("exhausted", atomic.LoadInt32(&job.Exhausted)),
		zap.Int32("expired", atomic.LoadInt32(&job.Expired)),
		zap.Int32("missing", atomic.LoadInt32(&job.Missing)),
		zap.Int32("error", atomic.LoadInt32(&job.Errors)),
	)
}

func (s *WhamVerifyService) verifyAccount(ctx context.Context, job *whamVerifyJob, account *Account) {
	// Short-circuit on cancellation before touching Done/writing DB — avoids
	// incrementing counters and poisoning healthy accounts with stale errors.
	if ctx.Err() != nil {
		return
	}
	defer atomic.AddInt32(&job.Done, 1)

	accessToken := account.GetOpenAIAccessToken()
	chatgptAccID := account.GetChatGPTAccountID()

	if accessToken == "" || chatgptAccID == "" {
		atomic.AddInt32(&job.Missing, 1)
		s.writeResult(account.ID, whamStatusMissing, "missing access_token or chatgpt_account_id", nil)
		return
	}

	if exp, ok := whamJWTExpiry(accessToken); ok && exp.Before(time.Now()) {
		atomic.AddInt32(&job.Expired, 1)
		job.addExpiredTarget(account.ID)
		s.writeResult(account.ID, whamStatusExpired, fmt.Sprintf("access_token expired at %s", exp.Format(time.RFC3339)), nil)
		return
	}

	status, body, err := s.callWham(ctx, accessToken, chatgptAccID)
	if err != nil {
		// Suppress writeback when the request was cancelled — the caller cancelled
		// the whole job and we must not overwrite the previous healthy snapshot.
		if ctx.Err() != nil {
			return
		}
		atomic.AddInt32(&job.Errors, 1)
		s.writeResult(account.ID, whamStatusError, err.Error(), nil)
		return
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		atomic.AddInt32(&job.Expired, 1)
		job.addExpiredTarget(account.ID)
		s.writeResult(account.ID, whamStatusExpired, fmt.Sprintf("http %d: %s", status, whamTruncate(string(body), 200)), nil)
		return
	}
	if status >= 400 {
		atomic.AddInt32(&job.Errors, 1)
		s.writeResult(account.ID, whamStatusError, fmt.Sprintf("http %d: %s", status, whamTruncate(string(body), 200)), nil)
		return
	}

	snapshot, classified := parseWhamResponse(body)
	switch classified {
	case whamStatusHigh:
		atomic.AddInt32(&job.High, 1)
	case whamStatusMedium:
		atomic.AddInt32(&job.Medium, 1)
	case whamStatusLow:
		atomic.AddInt32(&job.Low, 1)
	case whamStatusExhausted:
		atomic.AddInt32(&job.Exhausted, 1)
		job.addExhaustedTarget(account.ID, snapshot)
	default:
		classified = whamStatusUnknown
		atomic.AddInt32(&job.Unknown, 1)
	}
	s.writeResult(account.ID, classified, "", snapshot)
}

func (j *whamVerifyJob) addExpiredTarget(accountID int64) {
	j.targetsMu.Lock()
	j.expiredIDs = append(j.expiredIDs, accountID)
	j.targetsMu.Unlock()
}

// addExhaustedTarget records the account along with its 7d reset time (if
// known) so Apply can call SetRateLimited with an accurate expiry. If the 7d
// reset is unknown, fall back to the 5h reset — better to unblock at the
// wrong time than never unblock.
func (j *whamVerifyJob) addExhaustedTarget(accountID int64, snapshot *OpenAICodexUsageSnapshot) {
	resetAt := extractCodexExhaustedResetAt(snapshot, time.Now())
	j.targetsMu.Lock()
	j.exhaustedTargets[accountID] = resetAt
	j.targetsMu.Unlock()
}

// extractCodexExhaustedResetAt prefers the secondary (7d) window reset time
// since that's the window that defines when a fully exhausted account can be
// used again. Returns the zero time if nothing usable is present.
func extractCodexExhaustedResetAt(snapshot *OpenAICodexUsageSnapshot, now time.Time) time.Time {
	if snapshot == nil {
		return time.Time{}
	}
	if snapshot.SecondaryResetAfterSeconds != nil && *snapshot.SecondaryResetAfterSeconds > 0 {
		return now.Add(time.Duration(*snapshot.SecondaryResetAfterSeconds) * time.Second)
	}
	if snapshot.PrimaryResetAfterSeconds != nil && *snapshot.PrimaryResetAfterSeconds > 0 {
		return now.Add(time.Duration(*snapshot.PrimaryResetAfterSeconds) * time.Second)
	}
	return time.Time{}
}

func (s *WhamVerifyService) callWham(ctx context.Context, accessToken, chatgptAccID string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, whamUsageURL, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Chatgpt-Account-Id", chatgptAccID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", whamVerifyUA)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, body, err
	}
	return resp.StatusCode, body, nil
}

func (s *WhamVerifyService) writeResult(accountID int64, status, errMsg string, snapshot *OpenAICodexUsageSnapshot) {
	now := time.Now()
	updates := map[string]any{
		"wham_verify_status":    status,
		"wham_last_verified_at": now.Format(time.RFC3339),
	}
	if errMsg != "" {
		updates["wham_verify_error"] = whamTruncate(errMsg, whamErrorMsgMax)
	} else {
		updates["wham_verify_error"] = ""
	}
	if snapshot != nil {
		for k, v := range buildCodexUsageExtraUpdates(snapshot, now) {
			updates[k] = v
		}
	}
	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.accountRepo.UpdateExtra(updateCtx, accountID, updates); err != nil {
		logger.L().Warn("wham_verify: UpdateExtra failed",
			zap.Int64("account_id", accountID),
			zap.Error(err),
		)
	}
}

// whamUsagePayload captures the shape of https://chatgpt.com/backend-api/wham/usage.
type whamUsagePayload struct {
	PlanType  string `json:"plan_type"`
	RateLimit struct {
		LimitReached    bool             `json:"limit_reached"`
		Allowed         *bool            `json:"allowed"`
		PrimaryWindow   *whamUsageWindow `json:"primary_window"`
		SecondaryWindow *whamUsageWindow `json:"secondary_window"`
	} `json:"rate_limit"`
}

type whamUsageWindow struct {
	UsedPercent        *float64 `json:"used_percent"`
	ResetAt            *float64 `json:"reset_at"`
	ResetAfterSeconds  *float64 `json:"reset_after_seconds"`
	LimitWindowSeconds *float64 `json:"limit_window_seconds"`
}

// parseWhamResponse converts the JSON payload into an OpenAICodexUsageSnapshot
// and classifies usage into high/medium/low/exhausted. Percent is assumed to
// be on 0-100 scale (validated against the reference check_codex_quota.py).
func parseWhamResponse(body []byte) (*OpenAICodexUsageSnapshot, string) {
	var p whamUsagePayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, whamStatusError
	}

	// Prefer explicit limit_window_seconds tags to determine which slot is 5h
	// vs 7d. When absent, fall back to the ParseCodexRateLimitHeaders
	// convention (primary=7d/secondary=5h) so callers see consistent data.
	var fiveH, sevenD *whamUsageWindow
	for _, w := range []*whamUsageWindow{p.RateLimit.PrimaryWindow, p.RateLimit.SecondaryWindow} {
		if w == nil || w.LimitWindowSeconds == nil {
			continue
		}
		switch int(*w.LimitWindowSeconds) {
		case whamWindow5hSeconds:
			fiveH = w
		case whamWindow7dSeconds:
			sevenD = w
		}
	}
	if fiveH == nil && sevenD == nil {
		sevenD = p.RateLimit.PrimaryWindow
		fiveH = p.RateLimit.SecondaryWindow
	}

	snap := &OpenAICodexUsageSnapshot{}
	if fiveH != nil {
		if fiveH.UsedPercent != nil {
			v := *fiveH.UsedPercent
			snap.PrimaryUsedPercent = &v
		}
		if fiveH.ResetAfterSeconds != nil {
			v := int(*fiveH.ResetAfterSeconds)
			snap.PrimaryResetAfterSeconds = &v
		}
		mins := whamWindow5hSeconds / 60
		snap.PrimaryWindowMinutes = &mins
	}
	if sevenD != nil {
		if sevenD.UsedPercent != nil {
			v := *sevenD.UsedPercent
			snap.SecondaryUsedPercent = &v
		}
		if sevenD.ResetAfterSeconds != nil {
			v := int(*sevenD.ResetAfterSeconds)
			snap.SecondaryResetAfterSeconds = &v
		}
		mins := whamWindow7dSeconds / 60
		snap.SecondaryWindowMinutes = &mins
	}

	var used7d, used5h *float64
	if sevenD != nil {
		used7d = sevenD.UsedPercent
	}
	if fiveH != nil {
		used5h = fiveH.UsedPercent
	}

	if p.RateLimit.LimitReached {
		return snap, whamStatusExhausted
	}
	pct := used7d
	if pct == nil {
		pct = used5h
	}
	if pct == nil {
		return snap, whamStatusUnknown
	}
	switch {
	case *pct >= 100:
		return snap, whamStatusExhausted
	case *pct >= 70:
		return snap, whamStatusLow
	case *pct >= 30:
		return snap, whamStatusMedium
	default:
		return snap, whamStatusHigh
	}
}

func whamJWTExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Tolerate padded base64 variants.
		raw, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, false
		}
	}
	var claims struct {
		Exp *float64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp == nil {
		return time.Time{}, false
	}
	return time.Unix(int64(*claims.Exp), 0), true
}

func accountMatchesGroups(a *Account, filter map[int64]struct{}) bool {
	if a == nil {
		return false
	}
	for _, gid := range a.GroupIDs {
		if _, ok := filter[gid]; ok {
			return true
		}
	}
	return false
}

// whamTruncate cuts a string to at most n bytes, preserving UTF-8 boundaries.
// Used for truncating upstream error bodies before persisting / logging.
func whamTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Back off to a valid UTF-8 boundary.
	for n > 0 && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}

// WhamApplyRequest captures which post-verify actions to execute.
type WhamApplyRequest struct {
	RefreshExpired bool `json:"refresh_expired"`
	MarkExhausted  bool `json:"mark_exhausted"`
	DryRun         bool `json:"dry_run"`
}

// WhamApplyFailure records a single account-level failure for UI visibility.
type WhamApplyFailure struct {
	AccountID int64  `json:"account_id"`
	Action    string `json:"action"`
	Error     string `json:"error"`
}

// WhamApplyResult summarises what Apply did.
type WhamApplyResult struct {
	RefreshSucceeded    int                `json:"refresh_succeeded"`
	RefreshFailed       int                `json:"refresh_failed"`
	MarkedRateLimited   int                `json:"marked_rate_limited"`
	MarkRateLimitFailed int                `json:"mark_rate_limit_failed"`
	SkippedNoResetAt    int                `json:"skipped_no_reset_at"`
	DryRun              bool               `json:"dry_run"`
	Failures            []WhamApplyFailure `json:"failures,omitempty"`
}

// Apply executes the post-verify actions against the targets collected during
// verify. Mirrors the gateway's own side-effect paths:
//   - Expired tokens: try OpenAIOAuthService.RefreshAccountToken; on failure
//     fall back to accountRepo.SetError (same as gateway's 401 token-revoked).
//   - Exhausted accounts: accountRepo.SetRateLimited with the 7d reset time
//     (same as gateway's 429 handler via calculateOpenAI429ResetTime).
//
// Concurrent execution is capped at whamVerifyDefaultConc. Apply is not
// re-entrant on the same job.
func (s *WhamVerifyService) Apply(ctx context.Context, jobID string, req *WhamApplyRequest) (*WhamApplyResult, error) {
	if req == nil {
		req = &WhamApplyRequest{}
	}

	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("job not found")
	}
	if job.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("job still running; cancel or wait before applying")
	}
	if job.applying {
		s.mu.Unlock()
		return nil, fmt.Errorf("apply already in progress for this job")
	}
	job.applying = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		job.applying = false
		s.mu.Unlock()
	}()

	job.targetsMu.Lock()
	expiredIDs := append([]int64(nil), job.expiredIDs...)
	exhaustedCopy := make(map[int64]time.Time, len(job.exhaustedTargets))
	for id, r := range job.exhaustedTargets {
		exhaustedCopy[id] = r
	}
	job.targetsMu.Unlock()

	result := &WhamApplyResult{DryRun: req.DryRun}
	var mu sync.Mutex
	recordFailure := func(accountID int64, action string, err error) {
		mu.Lock()
		defer mu.Unlock()
		result.Failures = append(result.Failures, WhamApplyFailure{
			AccountID: accountID,
			Action:    action,
			Error:     whamTruncate(err.Error(), whamErrorMsgMax),
		})
	}

	if req.RefreshExpired && len(expiredIDs) > 0 {
		if s.openaiOAuth == nil {
			return nil, fmt.Errorf("openai oauth service not configured; refresh_expired unavailable")
		}
		sem := make(chan struct{}, whamVerifyDefaultConc)
		var wg sync.WaitGroup
		for _, id := range expiredIDs {
			accountID := id
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				if ctx.Err() != nil {
					return
				}
				if req.DryRun {
					mu.Lock()
					result.RefreshSucceeded++
					mu.Unlock()
					return
				}
				if err := s.refreshOneExpired(ctx, accountID); err != nil {
					mu.Lock()
					result.RefreshFailed++
					mu.Unlock()
					recordFailure(accountID, "refresh", err)
					return
				}
				mu.Lock()
				result.RefreshSucceeded++
				mu.Unlock()
			}()
		}
		wg.Wait()
	}

	if req.MarkExhausted && len(exhaustedCopy) > 0 {
		for id, resetAt := range exhaustedCopy {
			if resetAt.IsZero() {
				result.SkippedNoResetAt++
				continue
			}
			if req.DryRun {
				result.MarkedRateLimited++
				continue
			}
			if err := s.accountRepo.SetRateLimited(ctx, id, resetAt); err != nil {
				result.MarkRateLimitFailed++
				recordFailure(id, "mark_rate_limited", err)
				continue
			}
			result.MarkedRateLimited++
		}
	}

	logger.L().Info("wham_verify: apply finished",
		zap.String("job_id", jobID),
		zap.Bool("dry_run", req.DryRun),
		zap.Int("refresh_succeeded", result.RefreshSucceeded),
		zap.Int("refresh_failed", result.RefreshFailed),
		zap.Int("marked_rate_limited", result.MarkedRateLimited),
		zap.Int("mark_rate_limit_failed", result.MarkRateLimitFailed),
		zap.Int("skipped_no_reset_at", result.SkippedNoResetAt),
	)
	return result, nil
}

// refreshOneExpired loads the account, attempts a fresh OAuth refresh, and on
// failure marks the account as errored (same path gateway uses for
// token_invalidated).
func (s *WhamVerifyService) refreshOneExpired(ctx context.Context, accountID int64) error {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("load account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}
	if _, err := s.openaiOAuth.RefreshAccountToken(ctx, account); err != nil {
		// Token refresh failed — the account is effectively dead. Mark it error
		// so the scheduler stops picking it (gateway does the same in its 401
		// token_invalidated branch via handleAuthError).
		reason := fmt.Sprintf("wham-verify: token refresh failed: %s", whamTruncate(err.Error(), 200))
		if setErr := s.accountRepo.SetError(ctx, accountID, reason); setErr != nil {
			return fmt.Errorf("refresh failed (%v) and set_error failed (%w)", err, setErr)
		}
		return fmt.Errorf("refresh failed (marked error): %w", err)
	}
	return nil
}

// cleanupLoop drops finished jobs after whamVerifyJobTTL so we don't leak memory.
// Exits when Shutdown is called.
func (s *WhamVerifyService) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCleanup:
			return
		case <-ticker.C:
			s.gcFinishedJobs(time.Now())
		}
	}
}

func (s *WhamVerifyService) gcFinishedJobs(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, job := range s.jobs {
		if job.finishedAt == nil {
			continue
		}
		if now.Sub(*job.finishedAt) > whamVerifyJobTTL {
			delete(s.jobs, id)
		}
	}
}
