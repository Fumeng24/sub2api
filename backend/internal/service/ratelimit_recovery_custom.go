package service

import "context"

func hasCustomRecoverableRuntimeState(account *Account) bool {
	return false
}

func (s *RateLimitService) clearCustomRecoverableRuntimeState(_ context.Context, _ int64) error {
	return nil
}

func (s *RateLimitService) restoreErrorDisabledScheduling(ctx context.Context, account *Account) (bool, error) {
	if account == nil || account.Status != StatusError || account.Schedulable {
		return false, nil
	}
	if err := s.accountRepo.SetSchedulable(ctx, account.ID, true); err != nil {
		return false, err
	}
	return true, nil
}
