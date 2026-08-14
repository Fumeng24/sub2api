package service

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const openAISlowReserveDefaultMaxEntries = 4096

type openAISlowReserveConfig struct {
	enabled    bool
	ttft       time.Duration
	ttl        time.Duration
	maxEntries int
}

type openAIAccountSlowReserveKey struct {
	AccountID int64
	Model     string
}

type openAIAccountSlowReserveEntry struct {
	MarkedAt    time.Time
	LastTouched time.Time
	ExpiresAt   time.Time
	Reason      string
	TTFTMs      int
}

// openAIAccountSlowReserveState is deliberately separate from runtime blocks:
// entries only affect candidate ordering and never mutate account scheduling
// state, account status, or cooldown fields.
type openAIAccountSlowReserveState struct {
	mu       sync.Mutex
	entries  map[openAIAccountSlowReserveKey]openAIAccountSlowReserveEntry
	onChange func(accountID int64, model string)
}

func newOpenAIAccountSlowReserveState() *openAIAccountSlowReserveState {
	return &openAIAccountSlowReserveState{
		entries: make(map[openAIAccountSlowReserveKey]openAIAccountSlowReserveEntry),
	}
}

func (s *openAIAccountSlowReserveState) setChangeNotifier(notifier func(accountID int64, model string)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.onChange = notifier
	s.mu.Unlock()
}

func openAIAccountSlowReserveKeyFor(accountID int64, model string) (openAIAccountSlowReserveKey, bool) {
	model = normalizeOpenAIAccountModelTransientModel(model)
	if accountID <= 0 || model == "" {
		return openAIAccountSlowReserveKey{}, false
	}
	return openAIAccountSlowReserveKey{AccountID: accountID, Model: model}, true
}

func (s *openAIAccountSlowReserveState) mark(accountID int64, model, reason string, ttftMs int, now time.Time, cfg openAISlowReserveConfig) (openAIAccountSlowReserveEntry, bool) {
	key, ok := openAIAccountSlowReserveKeyFor(accountID, model)
	if s == nil || !ok || !cfg.enabled || cfg.ttl <= 0 {
		return openAIAccountSlowReserveEntry{}, false
	}
	if now.IsZero() {
		now = time.Now()
	}

	s.mu.Lock()
	if s.entries == nil {
		s.entries = make(map[openAIAccountSlowReserveKey]openAIAccountSlowReserveEntry)
	}
	s.purgeExpiredLocked(now)
	entry, exists := s.entries[key]
	if !exists {
		s.evictOldestLocked(cfg.maxEntries)
		entry.MarkedAt = now
	}
	entry.LastTouched = now
	entry.ExpiresAt = now.Add(cfg.ttl)
	entry.Reason = strings.TrimSpace(reason)
	entry.TTFTMs = ttftMs
	s.entries[key] = entry
	notifier := s.onChange
	s.mu.Unlock()
	if notifier != nil {
		notifier(key.AccountID, key.Model)
	}
	return entry, !exists
}

func (s *openAIAccountSlowReserveState) clear(accountID int64, model string) (openAIAccountSlowReserveEntry, bool) {
	key, ok := openAIAccountSlowReserveKeyFor(accountID, model)
	if s == nil || !ok {
		return openAIAccountSlowReserveEntry{}, false
	}
	s.mu.Lock()
	entry, exists := s.entries[key]
	if exists {
		delete(s.entries, key)
	}
	notifier := s.onChange
	s.mu.Unlock()
	if exists && notifier != nil {
		notifier(key.AccountID, key.Model)
	}
	return entry, exists
}

func (s *openAIAccountSlowReserveState) isReserved(accountID int64, model string, now time.Time) bool {
	entry, ok := s.lookup(accountID, model, now)
	if !ok {
		return false
	}
	// The first slow sample or retryable timeout is only a pending signal. It is
	// kept in the same durable state table, but must not reorder candidates or
	// break an active sticky session until a second incident arrives.
	return !isOpenAISlowReservePendingReason(entry.Reason)
}

func (s *openAIAccountSlowReserveState) snapshot(accountID int64, model string) (openAIAccountSlowReserveEntry, bool) {
	key, ok := openAIAccountSlowReserveKeyFor(accountID, model)
	if s == nil || !ok {
		return openAIAccountSlowReserveEntry{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists {
		return openAIAccountSlowReserveEntry{}, false
	}
	return entry, true
}

func (s *openAIAccountSlowReserveState) restore(accountID int64, model string, entry openAIAccountSlowReserveEntry) {
	key, ok := openAIAccountSlowReserveKeyFor(accountID, model)
	if s == nil || !ok || entry.ExpiresAt.IsZero() || !entry.ExpiresAt.After(time.Now()) {
		return
	}
	if entry.MarkedAt.IsZero() {
		entry.MarkedAt = entry.ExpiresAt.Add(-5 * time.Minute)
	}
	if entry.LastTouched.IsZero() {
		entry.LastTouched = entry.MarkedAt
	}
	s.mu.Lock()
	if s.entries == nil {
		s.entries = make(map[openAIAccountSlowReserveKey]openAIAccountSlowReserveEntry)
	}
	if existing, exists := s.entries[key]; !exists || existing.ExpiresAt.Before(entry.ExpiresAt) {
		s.entries[key] = entry
	}
	s.mu.Unlock()
}

func (s *openAIAccountSlowReserveState) lookup(accountID int64, model string, now time.Time) (openAIAccountSlowReserveEntry, bool) {
	key, ok := openAIAccountSlowReserveKeyFor(accountID, model)
	if s == nil || !ok {
		return openAIAccountSlowReserveEntry{}, false
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists {
		return openAIAccountSlowReserveEntry{}, false
	}
	if !entry.ExpiresAt.After(now) {
		delete(s.entries, key)
		return openAIAccountSlowReserveEntry{}, false
	}
	entry.LastTouched = now
	s.entries[key] = entry
	return entry, true
}

func (s *openAIAccountSlowReserveState) size(now time.Time) int {
	if s == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	return len(s.entries)
}

func (s *openAIAccountSlowReserveState) purgeExpiredLocked(now time.Time) {
	for key, entry := range s.entries {
		if !entry.ExpiresAt.After(now) {
			delete(s.entries, key)
		}
	}
}

func (s *openAIAccountSlowReserveState) evictOldestLocked(maxEntries int) {
	if maxEntries <= 0 {
		maxEntries = openAISlowReserveDefaultMaxEntries
	}
	if len(s.entries) < maxEntries {
		return
	}
	var oldestKey openAIAccountSlowReserveKey
	var oldest time.Time
	found := false
	for key, entry := range s.entries {
		if !found || entry.LastTouched.Before(oldest) {
			oldestKey = key
			oldest = entry.LastTouched
			found = true
		}
	}
	if found {
		delete(s.entries, oldestKey)
	}
}

func (s *OpenAIGatewayService) openAISlowReserveConfig() openAISlowReserveConfig {
	if s == nil || s.cfg == nil {
		return openAISlowReserveConfig{
			enabled:    true,
			ttft:       15 * time.Second,
			ttl:        3 * time.Minute,
			maxEntries: openAISlowReserveDefaultMaxEntries,
		}
	}
	configured := s.cfg.Gateway.OpenAIScheduler.SlowReserve
	// Unit-level service construction often bypasses config.Load. Treat an
	// entirely zero nested struct as the documented defaults, while an explicit
	// enabled=false config loaded by Viper retains its non-zero defaults.
	if !configured.Enabled && configured.TTFTMs == 0 && configured.TTLSeconds == 0 && configured.MaxEntries == 0 {
		configured.Enabled = true
	}
	if configured.TTFTMs <= 0 {
		configured.TTFTMs = 15000
	}
	if configured.TTLSeconds <= 0 {
		configured.TTLSeconds = 180
	}
	if configured.MaxEntries <= 0 {
		configured.MaxEntries = openAISlowReserveDefaultMaxEntries
	}
	return openAISlowReserveConfig{
		enabled:    configured.Enabled,
		ttft:       time.Duration(configured.TTFTMs) * time.Millisecond,
		ttl:        time.Duration(configured.TTLSeconds) * time.Second,
		maxEntries: configured.MaxEntries,
	}
}

func (s *OpenAIGatewayService) getOpenAIAccountSlowReserveState() *openAIAccountSlowReserveState {
	if s == nil {
		return nil
	}
	s.openaiSlowReserveOnce.Do(func() {
		if s.openaiSlowReserve == nil {
			s.openaiSlowReserve = newOpenAIAccountSlowReserveState()
		}
	})
	return s.openaiSlowReserve
}

func (s *OpenAIGatewayService) isOpenAIAccountSlowReserve(accountID int64, mappedModel string) bool {
	cfg := s.openAISlowReserveConfig()
	if !cfg.enabled {
		return false
	}
	state := s.getOpenAIAccountSlowReserveState()
	return state != nil && state.isReserved(accountID, mappedModel, time.Now())
}

func (s *OpenAIGatewayService) isOpenAIAccountSlowReserveForRequest(account *Account, requestedModel string, requireCompact bool) bool {
	if s == nil || account == nil {
		return false
	}
	return s.isOpenAIAccountSlowReserve(account.ID, resolveOpenAIAccountUpstreamModelForRequest(account, requestedModel, requireCompact))
}

func (s *OpenAIGatewayService) shouldEscapeOpenAIStickySlowReserve(account *Account, requestedModel string, requireCompact bool) bool {
	if s == nil || account == nil || !s.openAISlowReserveConfig().enabled {
		return false
	}
	mappedModel := resolveOpenAIAccountUpstreamModelForRequest(account, requestedModel, requireCompact)
	state := s.getOpenAIAccountSlowReserveState()
	entry, ok := state.lookup(account.ID, mappedModel, time.Now())
	if !ok {
		return false
	}
	if isOpenAISlowReservePendingReason(entry.Reason) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(entry.Reason), "ttft") {
		return true
	}
	thresholdMs := 25000
	if s.cfg != nil && s.cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs > 0 {
		thresholdMs = s.cfg.Gateway.OpenAIScheduler.StickyEscapeTTFTMs
	}
	return entry.TTFTMs >= thresholdMs
}

// prioritizeOpenAISlowReserveAccounts keeps the caller's existing ordering
// (priority, compact tier, rate, etc.) and moves only model-scoped slow
// candidates behind normal candidates in the same compact-support tier.
func (s *OpenAIGatewayService) prioritizeOpenAISlowReserveAccounts(accounts []*Account, requestedModel string, requireCompact bool) []*Account {
	if len(accounts) <= 1 {
		return accounts
	}
	appendPartition := func(out []*Account, pool []*Account) []*Account {
		normal := make([]*Account, 0, len(pool))
		reserve := make([]*Account, 0, len(pool))
		for _, account := range pool {
			if s.isOpenAIAccountSlowReserveForRequest(account, requestedModel, requireCompact) {
				reserve = append(reserve, account)
				continue
			}
			normal = append(normal, account)
		}
		out = append(out, normal...)
		return append(out, reserve...)
	}
	if !requireCompact {
		return appendPartition(make([]*Account, 0, len(accounts)), accounts)
	}
	ordered := make([]*Account, 0, len(accounts))
	for _, tier := range []int{2, 1, 0} {
		pool := make([]*Account, 0, len(accounts))
		for _, account := range accounts {
			if openAICompactSupportTier(account) == tier {
				pool = append(pool, account)
			}
		}
		ordered = appendPartition(ordered, pool)
	}
	return ordered
}

func (s *OpenAIGatewayService) prioritizeOpenAISlowReserveAccountLoads(items []accountWithLoad, requestedModel string, requireCompact bool) []accountWithLoad {
	if len(items) <= 1 {
		return items
	}
	appendPartition := func(out []accountWithLoad, pool []accountWithLoad) []accountWithLoad {
		normal := make([]accountWithLoad, 0, len(pool))
		reserve := make([]accountWithLoad, 0, len(pool))
		for _, item := range pool {
			if s.isOpenAIAccountSlowReserveForRequest(item.account, requestedModel, requireCompact) {
				reserve = append(reserve, item)
				continue
			}
			normal = append(normal, item)
		}
		out = append(out, normal...)
		return append(out, reserve...)
	}
	if !requireCompact {
		return appendPartition(make([]accountWithLoad, 0, len(items)), items)
	}
	ordered := make([]accountWithLoad, 0, len(items))
	for _, tier := range []int{2, 1, 0} {
		pool := make([]accountWithLoad, 0, len(items))
		for _, item := range items {
			if openAICompactSupportTier(item.account) == tier {
				pool = append(pool, item)
			}
		}
		ordered = appendPartition(ordered, pool)
	}
	return ordered
}

func (s *OpenAIGatewayService) reportOpenAIAccountSlowReserveSuccess(accountID int64, mappedModel string, firstTokenMs *int) {
	if s == nil || firstTokenMs == nil || *firstTokenMs <= 0 {
		return
	}
	cfg := s.openAISlowReserveConfig()
	if !cfg.enabled {
		return
	}
	state := s.getOpenAIAccountSlowReserveState()
	if state == nil {
		return
	}
	if time.Duration(*firstTokenMs)*time.Millisecond > cfg.ttft {
		previous, exists := state.snapshot(accountID, mappedModel)
		reason := "ttft_pending"
		if exists && strings.EqualFold(strings.TrimSpace(previous.Reason), "ttft_pending") {
			// Two slow samples in the short reserve window are evidence of a
			// sustained tail, while one sample remains cache-friendly.
			reason = "ttft"
		} else if exists && strings.EqualFold(strings.TrimSpace(previous.Reason), "ttft") {
			// Keep an already active reserve alive while the upstream remains slow.
			reason = "ttft"
		} else if exists && strings.TrimSpace(previous.Reason) != "" {
			// A recent hard upstream failure remains the stronger signal until its
			// short TTL expires; a later slow success must not downgrade it to a
			// harmless pending sample.
			reason = previous.Reason
		}
		entry, created := state.mark(accountID, mappedModel, reason, *firstTokenMs, time.Now(), cfg)
		promoted := reason == "ttft" && exists && strings.EqualFold(strings.TrimSpace(previous.Reason), "ttft_pending")
		if created || promoted {
			event := "openai_slow_reserve_pending"
			if reason == "ttft" {
				event = "openai_slow_reserve_marked"
			}
			slog.Info(event,
				"account_id", accountID,
				"model", normalizeOpenAIAccountModelTransientModel(mappedModel),
				"reason", entry.Reason,
				"ttft_ms", *firstTokenMs,
				"expires_at", entry.ExpiresAt,
			)
		}
		return
	}
	// A fast response clears only a not-yet-promoted pending sample. An active
	// reserve keeps its short TTL so one fast response cannot cause flapping.
	if entry, exists := state.snapshot(accountID, mappedModel); exists && strings.EqualFold(strings.TrimSpace(entry.Reason), "ttft_pending") {
		state.clear(accountID, mappedModel)
	}
}

func (s *OpenAIGatewayService) reportOpenAIAccountSlowReserveFailure(accountID int64, mappedModel string, failoverErr *UpstreamFailoverError) {
	if s == nil {
		return
	}
	reason, ok := openAISlowReserveFailureReason(failoverErr)
	if !ok {
		return
	}
	cfg := s.openAISlowReserveConfig()
	if !cfg.enabled {
		return
	}
	state := s.getOpenAIAccountSlowReserveState()
	if state == nil {
		return
	}
	previous, exists := state.snapshot(accountID, mappedModel)
	storedReason := reason
	promoted := false
	// A retryable timeout already trips the account+model runtime cooldown for
	// the next request. Keep the cache-friendly sticky binding on the first
	// incident; only a second timeout inside the short reserve window upgrades
	// it to a durable candidate reserve and moves existing sessions. This avoids
	// turning one recovered failover into a three-minute cache split.
	if isOpenAISlowReserveFailureReason(reason) {
		switch {
		case exists && strings.EqualFold(strings.TrimSpace(previous.Reason), "failure_pending"):
			promoted = true
		case exists && strings.EqualFold(strings.TrimSpace(previous.Reason), "ttft_pending"):
			promoted = true
		case !exists:
			storedReason = "failure_pending"
		}
	}
	entry, created := state.mark(accountID, mappedModel, storedReason, 0, time.Now(), cfg)
	if created || promoted {
		event := "openai_slow_reserve_pending"
		if promoted {
			event = "openai_slow_reserve_marked"
		}
		slog.Info(event,
			"account_id", accountID,
			"model", normalizeOpenAIAccountModelTransientModel(mappedModel),
			"reason", entry.Reason,
			"expires_at", entry.ExpiresAt,
		)
	}
}

// markOpenAIAccountSlowReserveTerminalFailure promotes a user-visible upstream
// failure immediately.  Retryable failures stay pending until a second sample
// so a single recovered failover does not disturb sticky/cache affinity, but a
// request that has exhausted failover has already cost the user an error.  The
// account+model must therefore leave the normal candidate pool right away.
func (s *OpenAIGatewayService) markOpenAIAccountSlowReserveTerminalFailure(accountID int64, mappedModel string, failoverErr *UpstreamFailoverError) {
	if s == nil {
		return
	}
	reason, ok := openAISlowReserveTerminalFailureReason(failoverErr)
	if !ok {
		return
	}
	cfg := s.openAISlowReserveConfig()
	if !cfg.enabled {
		return
	}
	state := s.getOpenAIAccountSlowReserveState()
	if state == nil {
		return
	}
	entry, created := state.mark(accountID, mappedModel, reason, 0, time.Now(), cfg)
	if created || !isOpenAISlowReserveFailureReason(reason) {
		slog.Warn("openai_slow_reserve_terminal_failure",
			"account_id", accountID,
			"model", normalizeOpenAIAccountModelTransientModel(mappedModel),
			"reason", entry.Reason,
			"expires_at", entry.ExpiresAt,
		)
	}
}

// MarkOpenAIAccountModelTerminalFailure is called by handlers only when the
// current request can no longer fail over.  It deliberately does not record a
// second runtime transient streak: the upstream response path already did
// that before returning the failover error.
func (s *OpenAIGatewayService) MarkOpenAIAccountModelTerminalFailure(accountID int64, mappedModel string, failoverErr *UpstreamFailoverError) {
	s.markOpenAIAccountSlowReserveTerminalFailure(accountID, mappedModel, failoverErr)
}

func isOpenAISlowReservePendingReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "ttft_pending", "failure_pending":
		return true
	default:
		return false
	}
}

func isOpenAISlowReserveFailureReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "first_output_timeout", "upstream_timeout", "transport_timeout":
		return true
	default:
		return false
	}
}

func openAISlowReserveFailureReason(failoverErr *UpstreamFailoverError) (string, bool) {
	if failoverErr == nil {
		return "", false
	}
	if failoverErr.StatusCode == http.StatusGatewayTimeout || failoverErr.StatusCode == 524 {
		if strings.Contains(strings.ToLower(string(failoverErr.ResponseBody)), "first_output_timeout") {
			return "first_output_timeout", true
		}
		return "upstream_timeout", true
	}
	if strings.EqualFold(strings.TrimSpace(failoverErr.SchedulerCategory), "transient_timeout") {
		return "transport_timeout", true
	}
	return "", false
}

func openAISlowReserveTerminalFailureReason(failoverErr *UpstreamFailoverError) (string, bool) {
	if failoverErr == nil {
		return "", false
	}
	if reason, ok := openAISlowReserveFailureReason(failoverErr); ok {
		return reason, true
	}
	switch failoverErr.StatusCode {
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusTooManyRequests, 520, 521, 522, 523:
		return "terminal_upstream_failure", true
	default:
		return "", false
	}
}

// ReportOpenAIAccountScheduleFailure preserves existing scheduler statistics
// and additionally feeds timeout-like no-output failures into slow reserve.
func (s *OpenAIGatewayService) ReportOpenAIAccountScheduleFailure(accountID int64, mappedModel string, failoverErr *UpstreamFailoverError) {
	if s == nil {
		return
	}
	s.ReportOpenAIAccountScheduleResult(accountID, mappedModel, false, nil)
	s.reportOpenAIAccountSlowReserveFailure(accountID, mappedModel, failoverErr)
}

func (s *OpenAIGatewayService) openAIAccountSlowReserveActiveCount() int {
	if s == nil || !s.openAISlowReserveConfig().enabled {
		return 0
	}
	state := s.getOpenAIAccountSlowReserveState()
	if state == nil {
		return 0
	}
	return state.size(time.Now())
}
