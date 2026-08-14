package service

import (
	"strings"
	"sync"
	"time"
)

const (
	// openAIModelTransientStreakTTL bounds how long a failure streak survives
	// without a new failure. It exists only so the map does not keep state for
	// account+model pairs that stopped being used; a streak is otherwise reset
	// by recordSuccess alone.
	//
	// It must stay well above the cooldowns. Resetting the streak on a short
	// wall-clock window makes the breaker's sensitivity depend on request rate:
	// a gateway called less often than the window never reaches streak 2, so a
	// broken upstream is never cooled down and every request pays a failed
	// attempt plus a failover before reaching a healthy account. Low-traffic
	// deployments were hit hardest, which is the opposite of what a breaker
	// should do.
	openAIModelTransientStreakTTL = 30 * time.Minute
	// A visible upstream failure must remove the account+model from the next
	// request immediately.  The first two strikes use a short quarantine so a
	// recovered account can return without causing sticky-session churn; repeat
	// failures back off much further.
	openAIModelTransientShortCooldown  = 2 * time.Minute
	openAIModelTransientRepeatCooldown = 5 * time.Minute
	openAIModelTransientLongCooldown   = 15 * time.Minute
	openAIModelTransientDefaultMax     = 4096
	openAIModelTransientMaxModelBytes  = 512
)

type openAIAccountModelKey struct {
	AccountID int64
	Model     string
}

type openAIAccountModelTransientEntry struct {
	failureStreak int
	lastFailure   time.Time
	blockUntil    time.Time
	lastTouched   time.Time
}

type openAIAccountModelTransientDecision struct {
	FailureStreak int
	Cooldown      time.Duration
	BlockUntil    time.Time
}

type openAIAccountModelTransientState struct {
	mu         sync.Mutex
	entries    map[openAIAccountModelKey]openAIAccountModelTransientEntry
	maxEntries int
	onChange   func(accountID int64, model string)
}

func newOpenAIAccountModelTransientState(maxEntries int) *openAIAccountModelTransientState {
	if maxEntries <= 0 {
		maxEntries = openAIModelTransientDefaultMax
	}
	return &openAIAccountModelTransientState{
		entries:    make(map[openAIAccountModelKey]openAIAccountModelTransientEntry),
		maxEntries: maxEntries,
	}
}

func (s *openAIAccountModelTransientState) setChangeNotifier(notifier func(accountID int64, model string)) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.onChange = notifier
	s.mu.Unlock()
}

func normalizeOpenAIAccountModelTransientModel(model string) string {
	model = strings.TrimSpace(model)
	if len(model) > openAIModelTransientMaxModelBytes {
		return ""
	}
	return strings.ToLower(model)
}

func openAIAccountModelTransientKey(accountID int64, model string) (openAIAccountModelKey, bool) {
	model = normalizeOpenAIAccountModelTransientModel(model)
	if accountID <= 0 || model == "" {
		return openAIAccountModelKey{}, false
	}
	return openAIAccountModelKey{AccountID: accountID, Model: model}, true
}

func (s *openAIAccountModelTransientState) recordFailure(accountID int64, model string, now time.Time) openAIAccountModelTransientDecision {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return openAIAccountModelTransientDecision{}
	}
	if now.IsZero() {
		now = time.Now()
	}

	s.mu.Lock()
	if s.entries == nil {
		s.entries = make(map[openAIAccountModelKey]openAIAccountModelTransientEntry)
	}
	if s.maxEntries <= 0 {
		s.maxEntries = openAIModelTransientDefaultMax
	}

	entry, exists := s.entries[key]
	if !exists {
		s.evictOldestLocked()
	}
	// The streak is cleared by recordSuccess. Only drop it here when the entry
	// is stale beyond the TTL, or when the clock moved backwards.
	if !exists || entry.lastFailure.IsZero() || now.Sub(entry.lastFailure) > openAIModelTransientStreakTTL || now.Before(entry.lastFailure) {
		entry.failureStreak = 0
		entry.blockUntil = time.Time{}
	}
	entry.failureStreak++
	entry.lastFailure = now
	entry.lastTouched = now

	cooldown := openAIModelTransientShortCooldown
	switch {
	case entry.failureStreak == 2:
		cooldown = openAIModelTransientRepeatCooldown
	case entry.failureStreak >= 3:
		cooldown = openAIModelTransientLongCooldown
	}
	if cooldown > 0 {
		entry.blockUntil = now.Add(cooldown)
	} else {
		entry.blockUntil = time.Time{}
	}
	s.entries[key] = entry
	decision := openAIAccountModelTransientDecision{
		FailureStreak: entry.failureStreak,
		Cooldown:      cooldown,
		BlockUntil:    entry.blockUntil,
	}
	notifier := s.onChange
	s.mu.Unlock()
	if notifier != nil {
		notifier(key.AccountID, key.Model)
	}
	return decision
}

func (s *openAIAccountModelTransientState) recordSuccess(accountID int64, model string) {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return
	}
	s.mu.Lock()
	_, existed := s.entries[key]
	delete(s.entries, key)
	notifier := s.onChange
	s.mu.Unlock()
	if existed && notifier != nil {
		notifier(key.AccountID, key.Model)
	}
}

func (s *openAIAccountModelTransientState) snapshot(accountID int64, model string) (openAIAccountModelTransientEntry, bool) {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return openAIAccountModelTransientEntry{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists || entry.lastFailure.IsZero() {
		return openAIAccountModelTransientEntry{}, false
	}
	return entry, true
}

func (s *openAIAccountModelTransientState) restore(accountID int64, model string, failureStreak int, lastFailure, blockUntil *time.Time, updatedAt time.Time) {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok || failureStreak <= 0 || lastFailure == nil {
		return
	}
	now := time.Now()
	if now.Sub(*lastFailure) > openAIModelTransientStreakTTL || now.Before(*lastFailure) {
		return
	}
	entry := openAIAccountModelTransientEntry{
		failureStreak: failureStreak,
		lastFailure:   *lastFailure,
		lastTouched:   updatedAt,
	}
	if entry.lastTouched.IsZero() {
		entry.lastTouched = now
	}
	if blockUntil != nil {
		entry.blockUntil = *blockUntil
	}
	s.mu.Lock()
	if s.entries == nil {
		s.entries = make(map[openAIAccountModelKey]openAIAccountModelTransientEntry)
	}
	if existing, exists := s.entries[key]; !exists || existing.lastFailure.Before(entry.lastFailure) {
		s.evictOldestLocked()
		s.entries[key] = entry
	}
	s.mu.Unlock()
}

func (s *openAIAccountModelTransientState) isBlocked(accountID int64, model string, now time.Time) bool {
	return s.activeFailureStreak(accountID, model, now) > 0
}

func (s *openAIAccountModelTransientState) activeFailureStreak(accountID int64, model string, now time.Time) int {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists {
		return 0
	}
	if !entry.lastFailure.IsZero() && now.Sub(entry.lastFailure) > openAIModelTransientStreakTTL {
		delete(s.entries, key)
		return 0
	}
	entry.lastTouched = now
	s.entries[key] = entry
	if entry.blockUntil.IsZero() || !now.Before(entry.blockUntil) {
		return 0
	}
	return entry.failureStreak
}

func (s *openAIAccountModelTransientState) size() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *openAIAccountModelTransientState) evictOldestLocked() {
	if len(s.entries) < s.maxEntries {
		return
	}
	var oldestKey openAIAccountModelKey
	var oldestTime time.Time
	found := false
	for key, entry := range s.entries {
		if !found || entry.lastTouched.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastTouched
			found = true
		}
	}
	if found {
		delete(s.entries, oldestKey)
	}
}
