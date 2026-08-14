//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type updateAPIKeyRepoStub struct {
	key     *APIKey
	updated *APIKey
}

func (s *updateAPIKeyRepoStub) Create(context.Context, *APIKey) error {
	panic("unexpected Create call")
}

func (s *updateAPIKeyRepoStub) GetByID(context.Context, int64) (*APIKey, error) {
	if s.key == nil {
		return nil, ErrAPIKeyNotFound
	}
	clone := *s.key
	clone.IPWhitelist = append([]string(nil), s.key.IPWhitelist...)
	clone.IPBlacklist = append([]string(nil), s.key.IPBlacklist...)
	return &clone, nil
}

func (s *updateAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected GetKeyAndOwnerID call")
}

func (s *updateAPIKeyRepoStub) GetByKey(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKey call")
}

func (s *updateAPIKeyRepoStub) GetByKeyForAuth(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKeyForAuth call")
}

func (s *updateAPIKeyRepoStub) Update(_ context.Context, key *APIKey, _ APIKeyUpdateFields) error {
	clone := *key
	clone.IPWhitelist = append([]string(nil), key.IPWhitelist...)
	clone.IPBlacklist = append([]string(nil), key.IPBlacklist...)
	s.updated = &clone
	return nil
}

func (s *updateAPIKeyRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *updateAPIKeyRepoStub) DeleteWithAudit(context.Context, int64) error {
	panic("unexpected DeleteWithAudit call")
}

func (s *updateAPIKeyRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}

func (s *updateAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected VerifyOwnership call")
}

func (s *updateAPIKeyRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}

func (s *updateAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	panic("unexpected ExistsByKey call")
}

func (s *updateAPIKeyRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}

func (s *updateAPIKeyRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}

func (s *updateAPIKeyRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}

func (s *updateAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}

func (s *updateAPIKeyRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}

func (s *updateAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}

func (s *updateAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}

func (s *updateAPIKeyRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *updateAPIKeyRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *updateAPIKeyRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}

func (s *updateAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}

func (s *updateAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

func TestAPIKeyService_UpdatePreservesIPRestrictionsWhenOmitted(t *testing.T) {
	repo := &updateAPIKeyRepoStub{
		key: &APIKey{
			ID:          10,
			UserID:      7,
			Key:         "sk-preserve-ip-acl",
			Status:      StatusActive,
			IPWhitelist: []string{"1.2.3.4"},
			IPBlacklist: []string{"5.6.7.8"},
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}

	resetQuota := true
	got, err := svc.Update(context.Background(), 10, 7, UpdateAPIKeyRequest{
		ResetQuota: &resetQuota,
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, []string{"1.2.3.4"}, got.IPWhitelist)
	require.Equal(t, []string{"5.6.7.8"}, got.IPBlacklist)
	require.NotNil(t, repo.updated)
	require.Equal(t, []string{"1.2.3.4"}, repo.updated.IPWhitelist)
	require.Equal(t, []string{"5.6.7.8"}, repo.updated.IPBlacklist)
}

func TestAPIKeyService_UpdateClearsIPRestrictionsWhenExplicitlyEmpty(t *testing.T) {
	repo := &updateAPIKeyRepoStub{
		key: &APIKey{
			ID:          10,
			UserID:      7,
			Key:         "sk-clear-ip-acl",
			Status:      StatusActive,
			IPWhitelist: []string{"1.2.3.4"},
			IPBlacklist: []string{"5.6.7.8"},
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}

	emptyWhitelist := []string{}
	emptyBlacklist := []string{}
	got, err := svc.Update(context.Background(), 10, 7, UpdateAPIKeyRequest{
		IPWhitelist: &emptyWhitelist,
		IPBlacklist: &emptyBlacklist,
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got.IPWhitelist)
	require.Empty(t, got.IPBlacklist)
	require.NotNil(t, repo.updated)
	require.Empty(t, repo.updated.IPWhitelist)
	require.Empty(t, repo.updated.IPBlacklist)
}
