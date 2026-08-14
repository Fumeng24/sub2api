package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
)

var ErrAPIKeyCategory = infraerrors.BadRequest("API_KEY_CATEGORY_INVALID", "api key category must be openai, anthropic, or other")

type apiKeyServiceCustomFields struct {
	accountRepo AccountRepository
}

func (s *APIKeyService) SetAccountRepository(repo AccountRepository) {
	s.accountRepo = repo
}

func prepareCreateAPIKeyRequestCustom(req *CreateAPIKeyRequest) error {
	category, ok := NormalizeAPIKeyCategory(req.Category)
	if !ok {
		return ErrAPIKeyCategory
	}
	req.Category = category
	return nil
}

func validateUpdateAPIKeyRequestCustom(req UpdateAPIKeyRequest) error {
	if req.IPWhitelist != nil && len(*req.IPWhitelist) > 0 {
		if invalid := ip.ValidateIPPatterns(*req.IPWhitelist); len(invalid) > 0 {
			return fmt.Errorf("%w: %v", ErrInvalidIPPattern, invalid)
		}
	}
	if req.IPBlacklist != nil && len(*req.IPBlacklist) > 0 {
		if invalid := ip.ValidateIPPatterns(*req.IPBlacklist); len(invalid) > 0 {
			return fmt.Errorf("%w: %v", ErrInvalidIPPattern, invalid)
		}
	}
	if req.Category != nil {
		if _, ok := NormalizeAPIKeyCategory(*req.Category); !ok {
			return ErrAPIKeyCategory
		}
	}
	return nil
}

func applyAPIKeyCategoryUpdateCustom(apiKey *APIKey, req UpdateAPIKeyRequest) {
	if req.Category == nil {
		return
	}
	apiKey.Category, _ = NormalizeAPIKeyCategory(*req.Category)
}

func applyAPIKeyIPRestrictionsUpdateCustom(apiKey *APIKey, req UpdateAPIKeyRequest) {
	if req.IPWhitelist != nil {
		apiKey.IPWhitelist = *req.IPWhitelist
	}
	if req.IPBlacklist != nil {
		apiKey.IPBlacklist = *req.IPBlacklist
	}
}

func (s *APIKeyService) HasActiveSubscriptionGroups(ctx context.Context) (bool, error) {
	if s == nil || s.groupRepo == nil {
		return false, nil
	}

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return false, fmt.Errorf("list active groups: %w", err)
	}
	for i := range groups {
		if groups[i].IsSubscriptionType() {
			return true, nil
		}
	}
	return false, nil
}

func (s *APIKeyService) resolveAvailableGroupModelsCustom(ctx context.Context, group Group) Group {
	group.ModelsListConfig = normalizeGroupModelsListConfig(group.ModelsListConfig)
	// A manually configured group list is the published model contract. It
	// must remain stable while accounts are temporarily cooling down, overloaded
	// or rate limited. The scheduler still validates account-level model support
	// when a request is made.
	if group.ModelsListConfig.Enabled || s.accountRepo == nil || strings.TrimSpace(group.Platform) == "" {
		return group
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, group.ID, group.Platform)
	if err != nil {
		return group
	}
	group.ModelsListConfig = resolveAvailableGroupModelsListConfig(group, accounts)
	return group
}

func resolveAvailableGroupModelsListConfig(group Group, accounts []Account) GroupModelsListConfig {
	config := normalizeGroupModelsListConfig(group.ModelsListConfig)
	if config.Enabled {
		return config
	}
	platform := group.Platform
	if len(accounts) == 0 {
		return GroupModelsListConfig{Enabled: true}
	}

	models, enumerable := modelsFromGroupAccounts(platform, accounts)
	if !enumerable {
		return config
	}
	if isUserVisibleImageGenerationGroup(&group) {
		models = filterImageModelNames(models)
	}
	return GroupModelsListConfig{Enabled: true, Models: models}
}

func groupVisibleModelsListForDiagnostics(group *Group, accounts []Account) (GroupModelsListConfig, bool) {
	if group == nil {
		return GroupModelsListConfig{}, false
	}
	return resolveAvailableGroupModelsListConfig(*group, accounts), true
}

func groupAvailableModelsForDiagnostics(platform string, accounts []Account) ([]string, bool) {
	return modelsFromGroupAccounts(platform, accounts)
}

func modelsFromGroupAccounts(platform string, accounts []Account) ([]string, bool) {
	normalizedPlatform := strings.ToLower(strings.TrimSpace(platform))
	seen := make(map[string]struct{})
	models := make([]string, 0)
	considered := 0

	for i := range accounts {
		account := &accounts[i]
		if normalizedPlatform != "" && strings.ToLower(strings.TrimSpace(account.Platform)) != normalizedPlatform {
			continue
		}
		considered++
		mapping := account.GetModelMapping()
		if len(mapping) == 0 {
			return nil, false
		}
		for model := range mapping {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			if strings.HasSuffix(model, "*") {
				return nil, false
			}
			key := strings.ToLower(model)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			models = append(models, model)
		}
	}

	if considered == 0 {
		return nil, true
	}
	sort.Strings(models)
	return models, true
}

func isUserVisibleImageGenerationGroup(group *Group) bool {
	if group == nil || !group.AllowImageGeneration {
		return false
	}
	platform := strings.ToLower(strings.TrimSpace(group.Platform))
	return platform == PlatformOpenAI || platform == PlatformGemini || platform == PlatformGrok
}

func isUserVisibleImageModelName(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	normalized = strings.TrimPrefix(normalized, "models/")
	return strings.Contains(normalized, "image")
}

func filterImageModelNames(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || !isUserVisibleImageModelName(model) {
			continue
		}
		filtered = append(filtered, model)
	}
	return filtered
}

func (s *APIKeyService) getUserGroupRatesCustom(ctx context.Context, userID int64) (map[int64]float64, error) {
	return s.userGroupRateRepo.GetEffectiveByUserID(ctx, userID)
}

// applyUserVisibleGroupRatesCustom overlays a user's final fixed or relative
// rate on group data returned by user-facing selection endpoints. This is a
// presentation copy only; persisted group rates and gateway billing remain
// unchanged.
func (s *APIKeyService) applyUserVisibleGroupRatesCustom(
	ctx context.Context,
	userID int64,
	groups []Group,
) ([]Group, error) {
	if len(groups) == 0 || s.userGroupRateRepo == nil {
		return groups, nil
	}
	rates, err := s.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if rate, ok := rates[groups[i].ID]; ok {
			groups[i].RateMultiplier = rate
		}
	}
	return groups, nil
}

// ApplyUserVisibleRateToAPIKey returns a presentation copy of an API key whose
// attached group exposes the user's effective rate. It deliberately leaves the
// repository-owned API key and group untouched so gateway billing continues to
// use the persisted group default and resolve the user rate independently.
func (s *APIKeyService) ApplyUserVisibleRateToAPIKey(
	ctx context.Context,
	userID int64,
	apiKey *APIKey,
) (*APIKey, error) {
	if apiKey == nil {
		return nil, nil
	}
	keys, err := s.ApplyUserVisibleRatesToAPIKeys(ctx, userID, []APIKey{*apiKey})
	if err != nil || len(keys) == 0 {
		return nil, err
	}
	return &keys[0], nil
}

// ApplyUserVisibleRatesToAPIKeys overlays each bound group's final rate in a
// copy returned by user-facing API-key endpoints. A user must never receive a
// pre-coefficient group rate through a second response path.
func (s *APIKeyService) ApplyUserVisibleRatesToAPIKeys(
	ctx context.Context,
	userID int64,
	apiKeys []APIKey,
) ([]APIKey, error) {
	if len(apiKeys) == 0 || s.userGroupRateRepo == nil {
		return apiKeys, nil
	}

	hasBoundGroup := false
	for i := range apiKeys {
		if apiKeys[i].Group != nil {
			hasBoundGroup = true
			break
		}
	}
	if !hasBoundGroup {
		return apiKeys, nil
	}

	rates, err := s.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rates) == 0 {
		return apiKeys, nil
	}

	out := append([]APIKey(nil), apiKeys...)
	for i := range out {
		if out[i].Group == nil {
			continue
		}
		rate, ok := rates[out[i].Group.ID]
		if !ok {
			continue
		}
		group := *out[i].Group
		group.RateMultiplier = rate
		out[i].Group = &group
	}
	return out, nil
}
