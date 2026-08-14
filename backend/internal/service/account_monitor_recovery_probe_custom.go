package service

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

var errTransientRecoveryProbeUnavailable = errors.New("transient recovery probe is unavailable for this account")

// ProbeTransientRecovery runs the existing lightweight account probe without
// creating an account-monitor row. It intentionally follows the account
// monitor's API-key/endpoint rules and is independent of the admin monitor
// runtime switch.
func (s *AccountMonitorService) ProbeTransientRecovery(ctx context.Context, accountID int64, requestedModel string) (*CheckResult, error) {
	if s == nil || s.account == nil {
		return nil, errTransientRecoveryProbeUnavailable
	}
	account, err := s.eligibleAccountForProbe(ctx, accountID)
	if err != nil || account == nil {
		return nil, errTransientRecoveryProbeUnavailable
	}
	endpoint := resolveAccountEndpoint(account)
	if err := validateAccountMonitorEndpoint(endpoint); err != nil {
		return nil, err
	}
	model := selectTransientRecoveryProbeModel(account, requestedModel)
	if model == "" {
		return nil, errTransientRecoveryProbeUnavailable
	}
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return nil, errTransientRecoveryProbeUnavailable
	}
	return runAccountMonitorCheckForModel(ctx, inferProviderFromAccount(account), endpoint, apiKey, model), nil
}

func selectTransientRecoveryProbeModel(account *Account, requestedModel string) string {
	if account == nil {
		return ""
	}
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel != "" && account.IsModelSupported(requestedModel) {
		if mapped := strings.TrimSpace(account.GetMappedModel(requestedModel)); mapped != "" {
			return mapped
		}
	}

	// On process restart the original request model is not in memory. Prefer a
	// concrete account mapping so the recovery probe does not revive an account
	// with a model the account explicitly excludes.
	mapping := account.GetModelMapping()
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, "*?") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if mapped := strings.TrimSpace(account.GetMappedModel(key)); mapped != "" {
			return mapped
		}
	}

	switch strings.ToLower(strings.TrimSpace(account.Platform)) {
	case PlatformAnthropic, "claude":
		return claude.DefaultTestModel
	case PlatformGemini, "google":
		return "gemini-2.5-flash"
	case PlatformGrok:
		return MonitorDefaultGrokModel
	default:
		return accountMonitorDefaultModel
	}
}
