package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// This file contains local channel-monitor extensions so the main service can
// continue to track upstream with only narrow integration hooks.

type channelMonitorServiceCustomFields struct {
	apiKeyRepo APIKeyRepository
}

func validateChannelMonitorCreateCustom(p ChannelMonitorCreateParams) error {
	if err := validateSortOrder(p.SortOrder); err != nil {
		return err
	}
	if p.APIKeyID != nil && *p.APIKeyID <= 0 {
		return ErrChannelMonitorInvalidAPIKeyID
	}
	return nil
}

func applyChannelMonitorUpdateCustom(existing *ChannelMonitor, p ChannelMonitorUpdateParams) error {
	if p.ClearAPIKeyID {
		existing.APIKeyID = nil
	} else if p.APIKeyID != nil {
		existing.APIKeyID = cloneInt64Ptr(p.APIKeyID)
	}
	if p.SortOrder != nil {
		if err := validateSortOrder(*p.SortOrder); err != nil {
			return err
		}
		existing.SortOrder = *p.SortOrder
	}
	return nil
}

func (s *ChannelMonitorService) resolveCreateAPIKey(ctx context.Context, p ChannelMonitorCreateParams) (string, *int64, error) {
	if p.APIKeyID != nil {
		plain, err := s.resolveLinkedAPIKey(ctx, *p.APIKeyID)
		return plain, cloneInt64Ptr(p.APIKeyID), err
	}
	plain := strings.TrimSpace(p.APIKey)
	if plain == "" {
		return "", nil, ErrChannelMonitorMissingAPIKey
	}
	return plain, s.lookupAPIKeyIDByPlain(ctx, plain), nil
}

// UpdateSortOrders updates the public channel-monitor display order in one batch.
func (s *ChannelMonitorService) UpdateSortOrders(ctx context.Context, updates []ChannelMonitorSortOrderUpdate) error {
	for _, update := range updates {
		if update.ID <= 0 {
			return ErrChannelMonitorNotFound
		}
		if err := validateSortOrder(update.SortOrder); err != nil {
			return err
		}
	}
	return s.repo.UpdateSortOrders(ctx, updates)
}

// applyAPIKeyUpdate supports both a direct key snapshot and a live API-key binding.
func (s *ChannelMonitorService) applyAPIKeyUpdateCustom(ctx context.Context, existing *ChannelMonitor, p ChannelMonitorUpdateParams) (plain string, updated bool, err error) {
	if !p.ClearAPIKeyID && p.APIKeyID != nil {
		plain, err = s.resolveLinkedAPIKey(ctx, *p.APIKeyID)
		if err != nil {
			return "", false, err
		}
		return s.applyAPIKeyUpdate(existing, &plain)
	}
	if p.APIKey == nil || strings.TrimSpace(*p.APIKey) == "" {
		return "", false, nil
	}
	plain = strings.TrimSpace(*p.APIKey)
	existing.APIKeyID = s.lookupAPIKeyIDByPlain(ctx, plain)
	return s.applyAPIKeyUpdate(existing, &plain)
}

func (s *ChannelMonitorService) lookupAPIKeyIDByPlain(ctx context.Context, plain string) *int64 {
	if s.apiKeyRepo == nil || strings.TrimSpace(plain) == "" {
		return nil
	}
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, strings.TrimSpace(plain))
	if err != nil || apiKey == nil || !apiKey.IsActive() {
		return nil
	}
	id := apiKey.ID
	return &id
}

func buildChannelMonitorErrorResults(m *ChannelMonitor, message string) []*CheckResult {
	if m == nil {
		return nil
	}
	models := append([]string{m.PrimaryModel}, m.ExtraModels...)
	now := time.Now()
	results := make([]*CheckResult, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		results = append(results, &CheckResult{
			Model:     model,
			Status:    MonitorStatusError,
			Message:   truncateMessage(sanitizeErrorMessage(message)),
			CheckedAt: now,
		})
	}
	return results
}

func channelMonitorRuntimeKeyErrorKind(err error) string {
	switch {
	case errors.Is(err, ErrChannelMonitorAPIKeyDecryptFailed):
		return "decrypt_failed"
	case errors.Is(err, ErrChannelMonitorMissingAPIKey), errors.Is(err, ErrChannelMonitorInvalidAPIKeyID):
		return "api_key_missing"
	case errors.Is(err, ErrAPIKeyNotFound), errors.Is(err, ErrAPIKeyExpired), errors.Is(err, ErrAPIKeyQuotaExhausted):
		return "linked_api_key_unavailable"
	default:
		return "api_key_unavailable"
	}
}

// SetAPIKeyRepository enables monitors to follow the current state of a bound key.
func (s *ChannelMonitorService) SetAPIKeyRepository(repo APIKeyRepository) {
	s.apiKeyRepo = repo
}

func (s *ChannelMonitorService) resolveDisplayAPIKeyInPlace(ctx context.Context, monitor *ChannelMonitor) {
	if monitor == nil {
		return
	}
	monitor.APIKeyDecryptFailed = false
	if monitor.APIKeyID != nil && *monitor.APIKeyID > 0 && s.apiKeyRepo != nil {
		apiKey, err := s.apiKeyRepo.GetByID(ctx, *monitor.APIKeyID)
		if err == nil && apiKey != nil {
			monitor.APIKey = apiKey.Key
			return
		}
		slog.Warn("channel_monitor: linked api key unavailable",
			"monitor_id", monitor.ID, "api_key_id", *monitor.APIKeyID, "error", err)
		monitor.APIKey = ""
		monitor.APIKeyDecryptFailed = true
		return
	}
	s.decryptInPlace(monitor)
}

func (s *ChannelMonitorService) resolveRuntimeAPIKeyInPlace(ctx context.Context, monitor *ChannelMonitor) error {
	if monitor == nil {
		return ErrChannelMonitorNotFound
	}
	monitor.APIKeyDecryptFailed = false
	if monitor.APIKeyID != nil && *monitor.APIKeyID > 0 && s.apiKeyRepo != nil {
		plain, err := s.resolveLinkedAPIKey(ctx, *monitor.APIKeyID)
		if err != nil {
			return err
		}
		monitor.APIKey = plain
		return nil
	}
	s.decryptInPlace(monitor)
	if monitor.APIKeyDecryptFailed {
		return ErrChannelMonitorAPIKeyDecryptFailed
	}
	return nil
}

func (s *ChannelMonitorService) resolveLinkedAPIKey(ctx context.Context, id int64) (string, error) {
	if id <= 0 {
		return "", ErrChannelMonitorInvalidAPIKeyID
	}
	if s.apiKeyRepo == nil {
		return "", infraerrors.InternalServer("CHANNEL_MONITOR_API_KEY_REPO_UNAVAILABLE", "api key repository is unavailable")
	}
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get linked api key: %w", err)
	}
	if !apiKey.IsActive() {
		return "", infraerrors.Unauthorized("API_KEY_INACTIVE", "api key is not active")
	}
	if apiKey.IsExpired() {
		return "", ErrAPIKeyExpired
	}
	if apiKey.IsQuotaExhausted() {
		return "", ErrAPIKeyQuotaExhausted
	}
	plain := strings.TrimSpace(apiKey.Key)
	if plain == "" {
		return "", ErrChannelMonitorMissingAPIKey
	}
	return plain, nil
}
