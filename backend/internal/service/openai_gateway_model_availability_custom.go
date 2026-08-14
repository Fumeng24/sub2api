package service

import (
	"context"
	"strings"
)

func (s *OpenAIGatewayService) diagnoseModelAvailabilityCustom(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	platform string,
) (ModelAvailabilityDiagnosis, bool) {
	if s == nil || strings.TrimSpace(requestedModel) == "" {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}, true
	}
	platform = normalizeOpenAICompatiblePlatform(platform)
	accounts, err := s.listConfiguredAccountsForModelAvailabilityCustom(ctx, groupID, platform)
	if err != nil {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}, true
	}
	diagnosis := ModelAvailabilityDiagnosis{}
	for i := range accounts {
		diagnosis.HasAccountsInPool = true
		if accounts[i].IsModelSupported(requestedModel) {
			diagnosis.HasModelSupport = true
			return diagnosis, true
		}
	}
	return diagnosis, true
}

func (s *OpenAIGatewayService) listConfiguredAccountsForModelAvailabilityCustom(ctx context.Context, groupID *int64, platform string) ([]Account, error) {
	if s.accountRepo == nil {
		return nil, nil
	}
	var (
		accounts []Account
		err      error
	)
	if groupID != nil && *groupID > 0 {
		accounts, err = s.accountRepo.ListByGroup(ctx, *groupID)
	} else {
		accounts, err = s.accountRepo.ListByPlatform(ctx, platform)
	}
	if err != nil {
		return nil, err
	}
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if normalizeOpenAICompatiblePlatform(strings.TrimSpace(account.Platform)) == platform {
			filtered = append(filtered, account)
		}
	}
	return filtered, nil
}
