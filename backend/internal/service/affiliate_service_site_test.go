//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestAffiliateCustomFieldsRemainFlattenedInJSON(t *testing.T) {
	createdAt := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	summaryBody, err := json.Marshal(AffiliateSummary{
		UserID: 7,
		AffiliateSummaryCustom: AffiliateSummaryCustom{
			UserCreatedAt: createdAt,
		},
	})
	require.NoError(t, err)
	var summaryJSON map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(summaryBody, &summaryJSON))
	require.Contains(t, summaryJSON, "user_created_at")
	require.NotContains(t, summaryJSON, "AffiliateSummaryCustom")

	detailBody, err := json.Marshal(AffiliateDetail{
		affiliateDetailCustom: affiliateDetailCustom{
			BindBonusAmount:    2.5,
			CanClaimBindBonus:  true,
			RebateDurationDays: 30,
		},
	})
	require.NoError(t, err)
	var detailJSON map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(detailBody, &detailJSON))
	require.JSONEq(t, `2.5`, string(detailJSON["bind_bonus_amount"]))
	require.JSONEq(t, `true`, string(detailJSON["can_claim_bind_bonus"]))
	require.JSONEq(t, `30`, string(detailJSON["rebate_duration_days"]))
	require.NotContains(t, detailJSON, "affiliateDetailCustom")
}

type affiliateBindSettingRepoStub struct {
	values map[string]string
}

func (s *affiliateBindSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *affiliateBindSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *affiliateBindSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *affiliateBindSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *affiliateBindSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *affiliateBindSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *affiliateBindSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type affiliateBindRepoStub struct {
	ensureCalls   []int64
	codeOwners    map[string]int64
	userCreatedAt time.Time
	inviterID     *int64
	claimedAt     *time.Time
	bindCalls     []struct {
		userID    int64
		inviterID int64
	}
	claimCalls []struct {
		userID int64
		amount float64
	}
}

func (r *affiliateBindRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	r.ensureCalls = append(r.ensureCalls, userID)
	userCreatedAt := r.userCreatedAt
	if userCreatedAt.IsZero() {
		userCreatedAt = time.Now()
	}
	return &AffiliateSummary{UserID: userID, AffCode: "SELF", InviterID: r.inviterID, AffiliateSummaryCustom: AffiliateSummaryCustom{BindBonusClaimedAt: r.claimedAt, UserCreatedAt: userCreatedAt}}, nil
}

func (r *affiliateBindRepoStub) GetAffiliateByCode(_ context.Context, code string) (*AffiliateSummary, error) {
	inviterID, ok := r.codeOwners[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return nil, ErrAffiliateProfileNotFound
	}
	return &AffiliateSummary{UserID: inviterID, AffCode: strings.ToUpper(strings.TrimSpace(code))}, nil
}

func (r *affiliateBindRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	r.bindCalls = append(r.bindCalls, struct {
		userID    int64
		inviterID int64
	}{userID: userID, inviterID: inviterID})
	return true, nil
}

func (r *affiliateBindRepoStub) ClaimBindBonus(_ context.Context, userID int64, amount float64) (bool, float64, error) {
	r.claimCalls = append(r.claimCalls, struct {
		userID int64
		amount float64
	}{userID: userID, amount: amount})
	return true, 12.34, nil
}

func (r *affiliateBindRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	panic("unexpected AccrueQuota call")
}

func (r *affiliateBindRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}

func (r *affiliateBindRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}

func (r *affiliateBindRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	panic("unexpected TransferQuotaToBalance call")
}

func (r *affiliateBindRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) {
	return nil, nil
}

func (r *affiliateBindRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}

func (r *affiliateBindRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}

func (r *affiliateBindRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	panic("unexpected SetUserRebateRate call")
}

func (r *affiliateBindRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (r *affiliateBindRepoStub) ListUsersWithCustomSettings(context.Context, AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func (r *affiliateBindRepoStub) ListAffiliateInviteRecords(context.Context, AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}

func (r *affiliateBindRepoStub) ListAffiliateRebateRecords(context.Context, AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}

func (r *affiliateBindRepoStub) ListAffiliateTransferRecords(context.Context, AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}

func (r *affiliateBindRepoStub) GetAffiliateUserOverview(context.Context, int64) (*AffiliateUserOverview, error) {
	panic("unexpected GetAffiliateUserOverview call")
}

type affiliateBindAuthInvalidatorStub struct {
	userIDs []int64
}

func (s *affiliateBindAuthInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *affiliateBindAuthInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string) {
	panic("unexpected InvalidateAuthCacheByKey call")
}

func (s *affiliateBindAuthInvalidatorStub) InvalidateAuthCacheByGroupID(context.Context, int64) {
	panic("unexpected InvalidateAuthCacheByGroupID call")
}

func TestBindInviterByCode_BindsInviterWithoutAutoBonus(t *testing.T) {
	repo := &affiliateBindRepoStub{
		codeOwners: map[string]int64{
			"BONUS123": 99,
		},
	}
	settingSvc := &SettingService{
		settingRepo: &affiliateBindSettingRepoStub{values: map[string]string{
			SettingKeyAffiliateEnabled:            "true",
			SettingKeyAffiliateBindBonusAmount:    "4.25",
			SettingKeyAffiliateRebateDurationDays: "30",
		}},
	}
	authInvalidator := &affiliateBindAuthInvalidatorStub{}
	svc := &AffiliateService{
		repo:                 repo,
		settingService:       settingSvc,
		authCacheInvalidator: authInvalidator,
	}

	require.NoError(t, svc.BindInviterByCode(context.Background(), 7, "bonus123"))
	require.Equal(t, []int64{7}, repo.ensureCalls)
	require.Len(t, repo.bindCalls, 1)
	require.Equal(t, int64(7), repo.bindCalls[0].userID)
	require.Equal(t, int64(99), repo.bindCalls[0].inviterID)
	require.Empty(t, repo.claimCalls)
	require.Empty(t, authInvalidator.userIDs)
}

func TestClaimBindBonus_AppliesConfiguredBonusAndInvalidatesCaches(t *testing.T) {
	inviterID := int64(99)
	repo := &affiliateBindRepoStub{
		userCreatedAt: time.Now(),
		inviterID:     &inviterID,
	}
	settingSvc := &SettingService{
		settingRepo: &affiliateBindSettingRepoStub{values: map[string]string{
			SettingKeyAffiliateEnabled:            "true",
			SettingKeyAffiliateBindBonusAmount:    "4.25",
			SettingKeyAffiliateRebateDurationDays: "30",
		}},
	}
	authInvalidator := &affiliateBindAuthInvalidatorStub{}
	svc := &AffiliateService{
		repo:                 repo,
		settingService:       settingSvc,
		authCacheInvalidator: authInvalidator,
	}

	balance, err := svc.ClaimBindBonus(context.Background(), 7)
	require.NoError(t, err)
	require.InDelta(t, 12.34, balance, 1e-9)
	require.Equal(t, []int64{7}, repo.ensureCalls)
	require.Len(t, repo.claimCalls, 1)
	require.Equal(t, int64(7), repo.claimCalls[0].userID)
	require.InDelta(t, 4.25, repo.claimCalls[0].amount, 1e-9)
	require.Equal(t, []int64{7}, authInvalidator.userIDs)
}

func TestBindInviterByCode_RejectsExpiredRegistrationWindow(t *testing.T) {
	repo := &affiliateBindRepoStub{
		codeOwners: map[string]int64{
			"OLD123": 99,
		},
		userCreatedAt: time.Now().Add(-25 * time.Hour),
	}
	settingSvc := &SettingService{
		settingRepo: &affiliateBindSettingRepoStub{values: map[string]string{
			SettingKeyAffiliateEnabled:            "true",
			SettingKeyAffiliateBindBonusAmount:    "4.25",
			SettingKeyAffiliateRebateDurationDays: "30",
		}},
	}
	authInvalidator := &affiliateBindAuthInvalidatorStub{}
	svc := &AffiliateService{
		repo:                 repo,
		settingService:       settingSvc,
		authCacheInvalidator: authInvalidator,
	}

	require.ErrorIs(t, svc.BindInviterByCode(context.Background(), 7, "old123"), ErrAffiliateBindExpired)
	require.Equal(t, []int64{7}, repo.ensureCalls)
	require.Empty(t, repo.bindCalls)
	require.Empty(t, authInvalidator.userIDs)
}

func TestGetAffiliateDetail_HidesBindBonusAfterRegistrationWindow(t *testing.T) {
	repo := &affiliateBindRepoStub{
		userCreatedAt: time.Now().Add(-25 * time.Hour),
	}
	settingSvc := &SettingService{
		settingRepo: &affiliateBindSettingRepoStub{values: map[string]string{
			SettingKeyAffiliateEnabled:            "true",
			SettingKeyAffiliateBindBonusAmount:    "4.25",
			SettingKeyAffiliateRebateDurationDays: "30",
		}},
	}
	svc := &AffiliateService{
		repo:           repo,
		settingService: settingSvc,
	}

	detail, err := svc.GetAffiliateDetail(context.Background(), 7)
	require.NoError(t, err)
	require.False(t, detail.CanBindInviter)
	require.False(t, detail.CanClaimBindBonus)
	require.Zero(t, detail.BindBonusAmount)
	require.Equal(t, 30, detail.RebateDurationDays)
}
