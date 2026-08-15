package service

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type openAIStickyFailoverContextKey uint8

const (
	openAIStickyFailoverTrackingKey openAIStickyFailoverContextKey = iota + 1
	openAIPreserveStickyBindingKey
	openAIStickyOriginalAccountIDKey
	openAIStickyGroupIDKey
	openAIStickySessionHashKey
)

// WithOpenAIStickyFailoverTracking makes the HTTP handler the owner of
// request-level transient accounting. This keeps pool-mode retries from being
// counted as separate failures before their same-account budget is exhausted.
func WithOpenAIStickyFailoverTracking(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAIStickyFailoverTrackingKey, true)
}

func openAIStickyFailoverTracking(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	tracked, _ := ctx.Value(openAIStickyFailoverTrackingKey).(bool)
	return tracked
}

// WithOpenAIPreserveStickyBinding prevents a fallback selected for this request
// from replacing or deleting the session's existing account binding.
func WithOpenAIPreserveStickyBinding(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAIPreserveStickyBindingKey, true)
}

func openAIPreserveStickyBinding(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	preserve, _ := ctx.Value(openAIPreserveStickyBindingKey).(bool)
	return preserve
}

func withOpenAIStickyOriginalAccountID(ctx context.Context, accountID int64) context.Context {
	if accountID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, openAIStickyOriginalAccountIDKey, accountID)
}

func openAIStickyOriginalAccountID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	accountID, _ := ctx.Value(openAIStickyOriginalAccountIDKey).(int64)
	return accountID
}

func withOpenAIStickyRoute(ctx context.Context, groupID *int64, sessionHash string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if groupID != nil {
		ctx = context.WithValue(ctx, openAIStickyGroupIDKey, *groupID)
	}
	if sessionHash = strings.TrimSpace(sessionHash); sessionHash != "" {
		ctx = context.WithValue(ctx, openAIStickySessionHashKey, sessionHash)
	}
	return ctx
}

func openAIStickyRoute(ctx context.Context) (int64, string) {
	if ctx == nil {
		return 0, ""
	}
	groupID, _ := ctx.Value(openAIStickyGroupIDKey).(int64)
	sessionHash, _ := ctx.Value(openAIStickySessionHashKey).(string)
	return groupID, strings.TrimSpace(sessionHash)
}

// PrepareOpenAIStickyFailoverContext captures the binding that existed before
// this logical request started. A fallback that has not completed successfully
// must not accidentally become the binding protected by a later transient
// failure in the same request.
func (s *OpenAIGatewayService) PrepareOpenAIStickyFailoverContext(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	requestedModel string,
) context.Context {
	ctx = WithOpenAIStickyFailoverTracking(ctx)
	ctx = withOpenAIStickyRoute(ctx, groupID, sessionHash)
	if s == nil || s.cache == nil || strings.TrimSpace(sessionHash) == "" {
		return ctx
	}
	accountID, err := s.getStickySessionAccountID(ctx, groupID, sessionHash)
	if err != nil || accountID <= 0 {
		return ctx
	}
	// A model-level runtime block is a hard veto for the current request.  Do
	// not preserve the old sticky binding while it is cooling down: the next
	// healthy fallback must be allowed to replace it.
	ctx = withOpenAIStickyOriginalAccountID(ctx, accountID)
	return ctx
}

func (s *OpenAIGatewayService) invalidateOpenAIStickyBindingAfterTimeout(
	ctx context.Context,
	accountID int64,
	failure *UpstreamFailoverError,
) {
	if s == nil || s.cache == nil || accountID <= 0 {
		return
	}
	if _, timeoutLike := openAISlowReserveFailureReason(failure); !timeoutLike {
		return
	}
	groupID, sessionHash := openAIStickyRoute(ctx)
	if groupID <= 0 || sessionHash == "" {
		return
	}
	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()
	boundAccountID, err := s.getStickySessionAccountID(stateCtx, &groupID, sessionHash)
	if err != nil || boundAccountID != accountID {
		return
	}
	if err := s.deleteStickySessionAccountID(stateCtx, &groupID, sessionHash); err != nil {
		return
	}
	slog.Info("openai_sticky_timeout_binding_invalidated",
		"account_id", accountID,
		"group_id", groupID,
	)
}

// ShouldPreserveOpenAIStickyBindingAfterFailure records one logical transient
// failure after same-account retries are exhausted.  A visible transient
// failure immediately invalidates the old model binding; a successful fallback
// is allowed to become the new sticky target in the same request.
func (s *OpenAIGatewayService) ShouldPreserveOpenAIStickyBindingAfterFailure(
	ctx context.Context,
	account *Account,
	canonicalModel string,
	failure *UpstreamFailoverError,
) bool {
	if s == nil || account == nil || account.Platform != PlatformOpenAI || failure == nil {
		return false
	}
	s.invalidateOpenAIStickyBindingAfterTimeout(ctx, account.ID, failure)
	if !failure.ShouldReportAccountScheduleFailure() {
		return shouldPreserveOpenAIStickyRequestOriginal(ctx, account.ID)
	}
	if !isOpenAIStickyTransientFailure(failure) {
		return false
	}
	decision := s.recordOpenAIAccountModelTransientFailure(account, canonicalModel, time.Now())
	logOpenAIAccountModelTransientDecision(account.ID, canonicalModel, decision)
	return false
}

func shouldPreserveOpenAIStickyRequestOriginal(ctx context.Context, accountID int64) bool {
	if openAIPreserveStickyBinding(ctx) {
		return true
	}
	originalAccountID := openAIStickyOriginalAccountID(ctx)
	return originalAccountID > 0 && accountID == originalAccountID
}

// ShouldPreserveOpenAIStickyBindingForActiveCooldown is retained as a
// compatibility hook for callers that ask whether a cooldown may pin a
// session. Runtime model cooldowns are hard candidate vetoes, so it always
// allows migration to a healthy fallback.
func (s *OpenAIGatewayService) ShouldPreserveOpenAIStickyBindingForActiveCooldown(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	requestedModel string,
) bool {
	// Active model cooldowns are hard candidate vetoes.  Returning true here
	// would keep a stale session binding alive and let it reappear after the
	// cooldown expires, defeating the first-failure migration policy.
	return false
}

func isOpenAIStickyTransientFailure(failure *UpstreamFailoverError) bool {
	if failure == nil || failure.RequestScopedTransient || isOpenAIStickyHardMigrationFailure(failure) {
		return false
	}
	category := strings.ToLower(strings.TrimSpace(failure.SchedulerCategory))
	// Durable proxy, DNS, or routing faults may be surfaced through a synthetic
	// 502. Their explicit category must win over the HTTP-shaped client status.
	if category == "error" {
		return false
	}
	switch failure.StatusCode {
	case http.StatusRequestTimeout, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 524:
		return true
	case 0:
		return strings.HasPrefix(category, "transient")
	default:
		return strings.HasPrefix(category, "transient")
	}
}

func isOpenAIStickyHardMigrationFailure(failure *UpstreamFailoverError) bool {
	if failure == nil {
		return false
	}
	if failure.IsCredentialFailure() {
		return true
	}
	category := strings.ToLower(strings.TrimSpace(failure.SchedulerCategory))
	switch category {
	case "auth", "authentication", "balance", "billing", "model_unsupported":
		return true
	}
	message := extractUpstreamErrorMessage(failure.ResponseBody)
	switch classifyOpenAIUpstreamError(failure.StatusCode, message, failure.ResponseBody) {
	case openAIUpstreamErrorAuth, openAIUpstreamErrorBilling, openAIUpstreamErrorModelUnsupported:
		return true
	default:
		return false
	}
}
