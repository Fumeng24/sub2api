package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// RateLimitPolicyExtension owns deployment-specific upstream error policy.
// Returning handled=false delegates to the upstream-derived default flow.
type RateLimitPolicyExtension interface {
	HandleBeforeDefault(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, requestedModel ...string) (disable, handled bool)
}

type rateLimitStatusPolicyExtension interface {
	HandleStatus(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, upstreamMsg string) (disable, handled bool)
}

type rateLimitCorePolicyExtension interface {
	HandleCoreLimits(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, upstreamMsg string) (disable, handled bool)
}

type customRateLimitPolicy struct {
	service *RateLimitService
}

func newCustomRateLimitPolicy(service *RateLimitService) RateLimitPolicyExtension {
	return &customRateLimitPolicy{service: service}
}

func (s *RateLimitService) handlePolicyExtensionsBeforeDefault(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, requestedModel ...string) (bool, bool) {
	if disable, handled := newCustomRateLimitPolicy(s).HandleBeforeDefault(ctx, account, statusCode, headers, responseBody, requestedModel...); handled {
		return disable, true
	}
	if s.policyExtension != nil {
		return s.policyExtension.HandleBeforeDefault(ctx, account, statusCode, headers, responseBody, requestedModel...)
	}
	return false, false
}

func (s *RateLimitService) handlePolicyExtensionsStatus(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, upstreamMsg string) (bool, bool) {
	if s == nil {
		return false, false
	}
	core := newCustomRateLimitPolicy(s).(rateLimitStatusPolicyExtension)
	if disable, handled := core.HandleStatus(ctx, account, statusCode, headers, responseBody, upstreamMsg); handled {
		return disable, true
	}
	if extension, ok := s.policyExtension.(rateLimitStatusPolicyExtension); ok {
		return extension.HandleStatus(ctx, account, statusCode, headers, responseBody, upstreamMsg)
	}
	return false, false
}

func (s *RateLimitService) handlePolicyExtensionsCoreLimits(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, upstreamMsg string) (bool, bool) {
	if s == nil {
		return false, false
	}
	core := newCustomRateLimitPolicy(s).(rateLimitCorePolicyExtension)
	if disable, handled := core.HandleCoreLimits(ctx, account, statusCode, headers, responseBody, upstreamMsg); handled {
		return disable, true
	}
	if extension, ok := s.policyExtension.(rateLimitCorePolicyExtension); ok {
		return extension.HandleCoreLimits(ctx, account, statusCode, headers, responseBody, upstreamMsg)
	}
	return false, false
}

func (s *RateLimitService) handlePolicyExtensionsCoreLimitsFromBodyCustom(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte) (bool, bool) {
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	if upstreamMsg != "" {
		upstreamMsg = truncateForLog([]byte(upstreamMsg), 512)
	}
	return s.handlePolicyExtensionsCoreLimits(ctx, account, statusCode, headers, responseBody, upstreamMsg)
}

func (s *RateLimitService) checkErrorPolicyOverride(ctx context.Context, account *Account, statusCode int, responseBody []byte) (ErrorPolicyResult, bool) {
	if account.IsCustomErrorCodesEnabled() && !account.ShouldHandleErrorCode(statusCode) {
		if isTransient5xxStatus(statusCode) && s.tryTempUnschedulable(ctx, account, statusCode, responseBody) {
			return ErrorPolicyTempUnscheduled, true
		}
		if shouldBypassCustomErrorCodeSkip(account, statusCode, responseBody) {
			return ErrorPolicyMatched, true
		}
	}
	if !account.IsCustomErrorCodesEnabled() && !account.IsPoolMode() && isOpenAI403ProbeCircuitError(account, statusCode, responseBody) {
		return ErrorPolicyNone, true
	}
	return ErrorPolicyNone, false
}

func (p *customRateLimitPolicy) HandleBeforeDefault(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, requestedModel ...string) (bool, bool) {
	model := tempUnschedulableModel(ctx, requestedModel)
	// A deterministic model rejection must win over the generic pool/transient
	// branches. Retrying the same account cannot make an unsupported model work.
	if p.service != nil && model != "" && p.service.HandleUpstreamModelNotFound(ctx, account, model, statusCode, responseBody) {
		return true, true
	}
	customErrorCodesEnabled := account.IsCustomErrorCodesEnabled()
	if account.IsPoolMode() && !customErrorCodesEnabled && !isTransient5xxStatus(statusCode) {
		slog.Info("pool_mode_error_skipped", "account_id", account.ID, "status_code", statusCode)
		return false, true
	}
	if !account.ShouldHandleErrorCode(statusCode) && !isTransient5xxStatus(statusCode) && !shouldBypassCustomErrorCodeSkip(account, statusCode, responseBody) {
		slog.Info("account_error_code_skipped", "account_id", account.ID, "status_code", statusCode)
		return false, true
	}
	if p.service == nil {
		return false, false
	}
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	if upstreamMsg != "" {
		upstreamMsg = truncateForLog([]byte(upstreamMsg), 512)
	}
	deferCoreLimitsToAnthropicWindow := account.Platform == PlatformAnthropic &&
		statusCode == http.StatusTooManyRequests && account.ShouldHandleErrorCode(statusCode)
	if !deferCoreLimitsToAnthropicWindow {
		if disable, handled := p.HandleCoreLimits(ctx, account, statusCode, headers, responseBody, upstreamMsg); handled {
			return disable, true
		}
	}
	if isTransient5xxStatus(statusCode) {
		if p.service.tryTempUnschedulable(ctx, account, statusCode, responseBody, model) {
			return true, true
		}
		// The OpenAI gateway owns its request-level account+model runtime circuit.
		// Keep pool-mode accounting here, and use account scope only when the
		// caller cannot identify which model failed.
		if account.Platform == PlatformOpenAI && model != "" && !account.IsPoolMode() {
			return false, true
		}
		return p.service.handleTransient5xx(ctx, account, statusCode, upstreamMsg, model), true
	}
	if account.Platform == PlatformOpenAI && statusCode == http.StatusForbidden && !account.ShouldHandleErrorCode(statusCode) {
		return p.service.handleOpenAI403Policy(ctx, account, upstreamMsg, responseBody), true
	}
	return false, false
}

func skipDefaultTempUnschedulableCustom(account *Account, statusCode int, responseBody []byte) bool {
	return isOpenAI403ProbeCircuitError(account, statusCode, responseBody)
}

func (s *RateLimitService) handleOpenAI403PolicyCustom(ctx context.Context, account *Account, upstreamMsg string, responseBody []byte) (bool, bool) {
	return s.handleOpenAI403Policy(ctx, account, upstreamMsg, responseBody), true
}

func (p *customRateLimitPolicy) HandleStatus(ctx context.Context, account *Account, statusCode int, _ http.Header, _ []byte, upstreamMsg string) (bool, bool) {
	if p.service == nil || !isTransient5xxStatus(statusCode) {
		return false, false
	}
	return p.service.handleTransient5xx(ctx, account, statusCode, upstreamMsg), true
}

func (p *customRateLimitPolicy) HandleCoreLimits(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, upstreamMsg string) (bool, bool) {
	if p.service == nil {
		return false, false
	}
	if account.Platform == PlatformOpenAI && isOpenAIImageRateLimitError(statusCode, responseBody) {
		if p.service.HandleOpenAIImageRateLimit(ctx, account, statusCode, headers, responseBody) {
			return false, true
		}
	}
	isOpenAIImageQuota429 := account.Platform == PlatformOpenAI && isOpenAIImageQuotaRateLimitError(statusCode, upstreamMsg, responseBody)
	if isOpenAIImageQuota429 || !isUpstreamBillingExhaustionError(statusCode, upstreamMsg, responseBody) {
		return false, false
	}
	msg := fmt.Sprintf("Upstream billing exhausted (%d): insufficient balance or billing issue", statusCode)
	if upstreamMsg != "" {
		msg = fmt.Sprintf("Upstream billing exhausted (%d): %s", statusCode, upstreamMsg)
	}
	p.service.handleAuthError(ctx, account, msg)
	return true, true
}
