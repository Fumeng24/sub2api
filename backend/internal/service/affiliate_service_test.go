//go:build unit

package service

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
	return &AffiliateSummary{UserID: userID, AffCode: "SELF", InviterID: r.inviterID, BindBonusClaimedAt: r.claimedAt, UserCreatedAt: userCreatedAt}, nil
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

// TestResolveRebateRatePercent_PerUserOverride verifies that per-inviter
// AffRebateRatePercent overrides the global rate, that NULL falls back to the
// global rate, and that out-of-range exclusive rates are clamped silently.
//
// SettingService is left nil here so globalRebateRatePercent returns the
// documented default (AffiliateRebateRateDefault = 20%) — this exercises the
// fallback path without spinning up a settings stub.
func TestResolveRebateRatePercent_PerUserOverride(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}

	// nil exclusive rate → falls back to global default (20%)
	require.InDelta(t, AffiliateRebateRateDefault,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{}), 1e-9)

	// exclusive rate set → overrides global
	rate := 50.0
	require.InDelta(t, 50.0,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &rate}), 1e-9)

	// exclusive rate 0 → returns 0 (no rebate, intentional)
	zero := 0.0
	require.InDelta(t, 0.0,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &zero}), 1e-9)

	// exclusive rate above max → clamped to Max
	tooHigh := 250.0
	require.InDelta(t, AffiliateRebateRateMax,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooHigh}), 1e-9)

	// exclusive rate below min → clamped to Min
	tooLow := -5.0
	require.InDelta(t, AffiliateRebateRateMin,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooLow}), 1e-9)
}

// TestIsEnabled_NilSettingServiceReturnsDefault verifies that IsEnabled
// safely handles a nil settingService dependency by returning the default
// (off). This protects callers from nil-pointer crashes in misconfigured
// environments.
func TestIsEnabled_NilSettingServiceReturnsDefault(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}
	require.False(t, svc.IsEnabled(context.Background()))
	require.Equal(t, AffiliateEnabledDefault, svc.IsEnabled(context.Background()))
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

// TestValidateExclusiveRate_BoundaryAndInvalid covers the validator used by
// admin-facing rate setters: nil is always valid (clear), in-range values
// are accepted, NaN/Inf and out-of-range values produce a typed BadRequest.
func TestValidateExclusiveRate_BoundaryAndInvalid(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateExclusiveRate(nil))

	for _, v := range []float64{0, 0.01, 50, 99.99, 100} {
		v := v
		require.NoError(t, validateExclusiveRate(&v), "value %v should be valid", v)
	}

	for _, v := range []float64{-0.01, 100.01, -100, 200} {
		v := v
		require.Error(t, validateExclusiveRate(&v), "value %v should be rejected", v)
	}

	nan := math.NaN()
	require.Error(t, validateExclusiveRate(&nan))
	posInf := math.Inf(1)
	require.Error(t, validateExclusiveRate(&posInf))
	negInf := math.Inf(-1)
	require.Error(t, validateExclusiveRate(&negInf))
}

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	require.Equal(t, "a***@g***.com", maskEmail("alice@gmail.com"))
	require.Equal(t, "x***@d***", maskEmail("x@domain"))
	require.Equal(t, "", maskEmail(""))
}

func TestIsValidAffiliateCodeFormat(t *testing.T) {
	t.Parallel()

	// 邀请码格式校验同时服务于：
	// 1) 系统自动生成的 12 位随机码（A-Z 去 I/O，2-9 去 0/1）
	// 2) 管理员设置的自定义专属码（如 "VIP2026"、"NEW_USER-1"）
	// 因此校验放宽到 [A-Z0-9_-]{4,32}（要求调用方先 ToUpper）。
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"valid canonical 12-char", "ABCDEFGHJKLM", true},
		{"valid all digits 2-9", "234567892345", true},
		{"valid mixed", "A2B3C4D5E6F7", true},
		{"valid admin custom short", "VIP1", true},
		{"valid admin custom with hyphen", "NEW-USER", true},
		{"valid admin custom with underscore", "VIP_2026", true},
		{"valid 32-char max", "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345", true},
		// Previously-excluded chars (I/O/0/1) are now allowed since admins may use them.
		{"letter I now allowed", "IBCDEFGHJKLM", true},
		{"letter O now allowed", "OBCDEFGHJKLM", true},
		{"digit 0 now allowed", "0BCDEFGHJKLM", true},
		{"digit 1 now allowed", "1BCDEFGHJKLM", true},
		{"too short (3 chars)", "ABC", false},
		{"too long (33 chars)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456", false},
		{"lowercase rejected (caller must ToUpper first)", "abcdefghjklm", false},
		{"empty", "", false},
		{"utf8 non-ascii", "ÄÄÄÄÄÄ", false}, // bytes out of charset
		{"ascii punctuation .", "ABCDEFGHJK.M", false},
		{"whitespace", "ABCDEFGHJK M", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isValidAffiliateCodeFormat(tc.in))
		})
	}
}
