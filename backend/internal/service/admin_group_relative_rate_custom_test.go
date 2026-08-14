//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

type relativeRateGroupRepoStub struct {
	GroupRepository
	group *Group
}

func (s *relativeRateGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	return s.group, nil
}

type relativeRateUserRepoStub struct {
	UserRepository
}

func (s *relativeRateUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	return &User{ID: id, Email: "user@example.com"}, nil
}

type relativeRateRepoStub struct {
	UserGroupRateRepository
	entries []UserGroupRateEntry
	synced  []GroupRelativeRateMultiplierInput
}

func (s *relativeRateRepoStub) GetByGroupID(context.Context, int64) ([]UserGroupRateEntry, error) {
	return s.entries, nil
}

func (s *relativeRateRepoStub) SyncGroupRelativeRateMultipliers(_ context.Context, _ int64, entries []GroupRelativeRateMultiplierInput) error {
	s.synced = append([]GroupRelativeRateMultiplierInput(nil), entries...)
	return nil
}

func TestAdminServiceSyncGroupRelativeRateMultipliersInvalidatesBothGatewayCaches(t *testing.T) {
	repo := &relativeRateRepoStub{}
	groupID := int64(17)
	gatewayCache := gocache.New(time.Minute, time.Minute)
	openAICache := gocache.New(time.Minute, time.Minute)
	gatewayCache.Set("1:17:0.5", 0.4, time.Minute)
	openAICache.Set("1:17:0.5", 0.4, time.Minute)

	svc := &adminServiceImpl{
		groupRepo:         &relativeRateGroupRepoStub{group: &Group{ID: groupID, SubscriptionType: SubscriptionTypeStandard}},
		userRepo:          &relativeRateUserRepoStub{},
		userGroupRateRepo: repo,
		AdminServiceCustomDependencies: AdminServiceCustomDependencies{
			gatewayService: &GatewayService{
				userGroupRateResolver: newUserGroupRateResolver(repo, gatewayCache, time.Minute, nil, "test.gateway"),
			},
			openAIGatewayService: &OpenAIGatewayService{
				userGroupRateResolver: newUserGroupRateResolver(repo, openAICache, time.Minute, nil, "test.openai"),
			},
		},
	}

	err := svc.SyncGroupRelativeRateMultipliers(context.Background(), groupID, []GroupRelativeRateMultiplierInput{{
		UserID:     1,
		Multiplier: 1.25,
	}})
	require.NoError(t, err)
	require.Equal(t, []GroupRelativeRateMultiplierInput{{UserID: 1, Multiplier: 1.25}}, repo.synced)
	require.Empty(t, gatewayCache.Items())
	require.Empty(t, openAICache.Items())
}

func TestAdminServiceSyncGroupRelativeRateMultipliersRejectsFixedRateConflict(t *testing.T) {
	fixedRate := 0.3
	svc := &adminServiceImpl{
		groupRepo: &relativeRateGroupRepoStub{group: &Group{ID: 17, SubscriptionType: SubscriptionTypeStandard}},
		userRepo:  &relativeRateUserRepoStub{},
		userGroupRateRepo: &relativeRateRepoStub{entries: []UserGroupRateEntry{{
			UserID:         1,
			RateMultiplier: &fixedRate,
		}}},
	}

	err := svc.SyncGroupRelativeRateMultipliers(context.Background(), 17, []GroupRelativeRateMultiplierInput{{
		UserID:     1,
		Multiplier: 0.8,
	}})
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, infraerrors.Code(err))
	require.Equal(t, "FIXED_RATE_CONFLICT", infraerrors.Reason(err))
}

func TestAdminServiceSyncGroupRelativeRateMultipliersRejectsValueBelowStoragePrecision(t *testing.T) {
	svc := &adminServiceImpl{
		groupRepo:         &relativeRateGroupRepoStub{group: &Group{ID: 17, SubscriptionType: SubscriptionTypeStandard}},
		userRepo:          &relativeRateUserRepoStub{},
		userGroupRateRepo: &relativeRateRepoStub{},
	}

	err := svc.SyncGroupRelativeRateMultipliers(context.Background(), 17, []GroupRelativeRateMultiplierInput{{
		UserID:     1,
		Multiplier: 0.00001,
	}})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "INVALID_RELATIVE_MULTIPLIER", infraerrors.Reason(err))
}

func TestAdminServiceSyncGroupRelativeRateMultipliersRejectsSubscriptionGroup(t *testing.T) {
	svc := &adminServiceImpl{
		groupRepo: &relativeRateGroupRepoStub{group: &Group{
			ID:               17,
			SubscriptionType: SubscriptionTypeSubscription,
		}},
		userRepo:          &relativeRateUserRepoStub{},
		userGroupRateRepo: &relativeRateRepoStub{},
	}

	err := svc.SyncGroupRelativeRateMultipliers(context.Background(), 17, []GroupRelativeRateMultiplierInput{{
		UserID:     1,
		Multiplier: 0.8,
	}})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "UNSUPPORTED_GROUP_TYPE", infraerrors.Reason(err))
}
