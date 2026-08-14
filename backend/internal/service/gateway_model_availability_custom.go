package service

import (
	"context"
	"strings"
)

func (s *GatewayService) diagnoseConfiguredModelAvailabilityCustom(ctx context.Context, groupID *int64, platform, requestedModel string) (ModelAvailabilityDiagnosis, bool) {
	if s == nil || s.accountRepo == nil {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}, true
	}
	requestedModel = strings.TrimSpace(requestedModel)
	platform = strings.TrimSpace(platform)
	if requestedModel == "" || platform == "" {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}, true
	}
	accounts, err := s.listConfiguredAccountsForModelAvailabilityCustom(ctx, groupID, platform)
	if err != nil {
		return ModelAvailabilityDiagnosis{HasAccountsInPool: true, HasModelSupport: true}, true
	}
	diagnosis := ModelAvailabilityDiagnosis{}
	for i := range accounts {
		diagnosis.HasAccountsInPool = true
		if s.isModelSupportedByAccountWithContext(ctx, &accounts[i], requestedModel) {
			diagnosis.HasModelSupport = true
			break
		}
	}
	return diagnosis, true
}

func (s *GatewayService) listConfiguredAccountsForModelAvailabilityCustom(ctx context.Context, groupID *int64, platform string) ([]Account, error) {
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
		if strings.TrimSpace(account.Platform) == platform {
			filtered = append(filtered, account)
		}
	}
	return filtered, nil
}
