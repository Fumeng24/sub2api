package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type apiKeyServiceAccountRepoCustomStub struct {
	AccountRepository
}

type apiKeyServiceEffectiveRateRepoCustomStub struct {
	UserGroupRateRepository
	effective map[int64]float64
}

func (s *apiKeyServiceEffectiveRateRepoCustomStub) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	panic("GetByUserID must not be used for effective rates")
}

func (s *apiKeyServiceEffectiveRateRepoCustomStub) GetEffectiveByUserID(context.Context, int64) (map[int64]float64, error) {
	return s.effective, nil
}

func TestNewAPIKeyServiceCustomPreservesOptionalAccountRepository(t *testing.T) {
	repo := &apiKeyServiceAccountRepoCustomStub{}
	svc := NewAPIKeyService(nil, nil, nil, nil, nil, nil, nil)
	svc.SetAccountRepository(repo)

	require.Same(t, repo, svc.accountRepo)
}

func TestAPIKeyServiceGetUserGroupRatesCustomUsesEffectiveRates(t *testing.T) {
	repo := &apiKeyServiceEffectiveRateRepoCustomStub{effective: map[int64]float64{7: 0.25}}
	svc := NewAPIKeyService(nil, nil, nil, nil, repo, nil, nil)

	rates, err := svc.GetUserGroupRates(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{7: 0.25}, rates)
}

func TestAPIKeyServiceApplyUserVisibleGroupRatesCustomOverwritesBaseRate(t *testing.T) {
	repo := &apiKeyServiceEffectiveRateRepoCustomStub{effective: map[int64]float64{7: 0.25}}
	svc := NewAPIKeyService(nil, nil, nil, nil, repo, nil, nil)

	groups, err := svc.applyUserVisibleGroupRatesCustom(context.Background(), 42, []Group{
		{ID: 7, RateMultiplier: 0.6},
		{ID: 8, RateMultiplier: 0.4},
	})
	require.NoError(t, err)
	require.Equal(t, 0.25, groups[0].RateMultiplier)
	require.Equal(t, 0.4, groups[1].RateMultiplier)
}

func TestAPIKeyServiceApplyUserVisibleRatesToAPIKeysOverwritesOnlyPresentationCopy(t *testing.T) {
	repo := &apiKeyServiceEffectiveRateRepoCustomStub{effective: map[int64]float64{7: 0.25}}
	svc := NewAPIKeyService(nil, nil, nil, nil, repo, nil, nil)
	originalGroup := &Group{ID: 7, RateMultiplier: 0.6}
	keys, err := svc.ApplyUserVisibleRatesToAPIKeys(context.Background(), 42, []APIKey{
		{ID: 1, Group: originalGroup},
		{ID: 2, Group: &Group{ID: 8, RateMultiplier: 0.4}},
	})
	require.NoError(t, err)
	require.Equal(t, 0.25, keys[0].Group.RateMultiplier)
	require.Equal(t, 0.4, keys[1].Group.RateMultiplier)
	require.Equal(t, 0.6, originalGroup.RateMultiplier)
	require.NotSame(t, originalGroup, keys[0].Group)
}

func TestPrepareCreateAPIKeyRequestCustomNormalizesCategory(t *testing.T) {
	req := CreateAPIKeyRequest{}
	require.NoError(t, prepareCreateAPIKeyRequestCustom(&req))
	require.Equal(t, APIKeyCategoryOther, req.Category)

	req.Category = "unsupported"
	require.ErrorIs(t, prepareCreateAPIKeyRequestCustom(&req), ErrAPIKeyCategory)
}

func TestApplyAPIKeyUpdateCustomPreservesOmittedIPRestrictions(t *testing.T) {
	apiKey := &APIKey{
		Category:    APIKeyCategoryOther,
		IPWhitelist: []string{"192.0.2.1"},
		IPBlacklist: []string{"198.51.100.1"},
	}

	applyAPIKeyCategoryUpdateCustom(apiKey, UpdateAPIKeyRequest{})
	applyAPIKeyIPRestrictionsUpdateCustom(apiKey, UpdateAPIKeyRequest{})

	require.Equal(t, APIKeyCategoryOther, apiKey.Category)
	require.Equal(t, []string{"192.0.2.1"}, apiKey.IPWhitelist)
	require.Equal(t, []string{"198.51.100.1"}, apiKey.IPBlacklist)
}

func TestApplyAPIKeyUpdateCustomAppliesExplicitValues(t *testing.T) {
	category := APIKeyCategoryOpenAI
	empty := []string{}
	blacklist := []string{"203.0.113.0/24"}
	req := UpdateAPIKeyRequest{
		Category:    &category,
		IPWhitelist: &empty,
		IPBlacklist: &blacklist,
	}
	require.NoError(t, validateUpdateAPIKeyRequestCustom(req))

	apiKey := &APIKey{
		Category:    APIKeyCategoryOther,
		IPWhitelist: []string{"192.0.2.1"},
	}
	applyAPIKeyCategoryUpdateCustom(apiKey, req)
	applyAPIKeyIPRestrictionsUpdateCustom(apiKey, req)

	require.Equal(t, APIKeyCategoryOpenAI, apiKey.Category)
	require.Empty(t, apiKey.IPWhitelist)
	require.Equal(t, blacklist, apiKey.IPBlacklist)
}
