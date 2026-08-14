//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type affiliateHandlerSettingRepoStub struct {
	service.SettingRepository
}

func (affiliateHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	switch key {
	case service.SettingKeyAffiliateEnabled:
		return "true", nil
	case service.SettingKeyAffiliateBindBonusAmount:
		return "4.25", nil
	default:
		return "", service.ErrSettingNotFound
	}
}

type affiliateHandlerRepoStub struct {
	service.AffiliateRepository

	inviterID   *int64
	claimedAt   *time.Time
	lookup      string
	bindUser    int64
	bindOwner   int64
	claimUser   int64
	claimAmount float64
}

func (r *affiliateHandlerRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*service.AffiliateSummary, error) {
	return &service.AffiliateSummary{
		UserID:    userID,
		AffCode:   "SELF",
		InviterID: r.inviterID,
		AffiliateSummaryCustom: service.AffiliateSummaryCustom{
			BindBonusClaimedAt: r.claimedAt,
			UserCreatedAt:      time.Now().Add(-time.Hour),
		},
	}, nil
}

func (r *affiliateHandlerRepoStub) GetAffiliateByCode(_ context.Context, code string) (*service.AffiliateSummary, error) {
	r.lookup = code
	if code != "OWNER" {
		return nil, service.ErrAffiliateProfileNotFound
	}
	return &service.AffiliateSummary{UserID: 42, AffCode: code}, nil
}

func (r *affiliateHandlerRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	r.bindUser = userID
	r.bindOwner = inviterID
	r.inviterID = &inviterID
	return true, nil
}

func (r *affiliateHandlerRepoStub) ClaimBindBonus(_ context.Context, userID int64, amount float64) (bool, float64, error) {
	r.claimUser = userID
	r.claimAmount = amount
	now := time.Now()
	r.claimedAt = &now
	return true, 12.34, nil
}

func (r *affiliateHandlerRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}

func (r *affiliateHandlerRepoStub) ListInvitees(context.Context, int64, int) ([]service.AffiliateInvitee, error) {
	return nil, nil
}

func newAffiliateHandlerForTest(repo *affiliateHandlerRepoStub) *UserHandler {
	settings := service.NewSettingService(affiliateHandlerSettingRepoStub{}, &config.Config{})
	affiliate := service.NewAffiliateService(repo, settings, nil, nil)
	return NewUserHandler(nil, nil, nil, nil, affiliate, nil)
}

func newAffiliateHandlerContext(method, target, body string, userID int64) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
	}
	return c, recorder
}

func TestUserHandlerBindAffiliateInviterRequiresAuthenticationAndCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newAffiliateHandlerForTest(&affiliateHandlerRepoStub{})

	c, recorder := newAffiliateHandlerContext(http.MethodPost, "/api/v1/user/aff/bind", `{"code":"OWNER"}`, 0)
	handler.BindAffiliateInviter(c)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	c, recorder = newAffiliateHandlerContext(http.MethodPost, "/api/v1/user/aff/bind", `{"code":"  "}`, 7)
	handler.BindAffiliateInviter(c)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "Affiliate code is required", gjson.Get(recorder.Body.String(), "message").String())
}

func TestUserHandlerBindAffiliateInviterReturnsUpdatedDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &affiliateHandlerRepoStub{}
	handler := newAffiliateHandlerForTest(repo)
	c, recorder := newAffiliateHandlerContext(http.MethodPost, "/api/v1/user/aff/bind", `{"code":" owner "}`, 7)

	handler.BindAffiliateInviter(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "OWNER", repo.lookup)
	require.Equal(t, int64(7), repo.bindUser)
	require.Equal(t, int64(42), repo.bindOwner)
	require.Equal(t, int64(42), gjson.Get(recorder.Body.String(), "data.inviter_id").Int())
}

func TestUserHandlerClaimAffiliateBindBonusReturnsBalanceAndDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inviterID := int64(42)
	repo := &affiliateHandlerRepoStub{inviterID: &inviterID}
	handler := newAffiliateHandlerForTest(repo)
	c, recorder := newAffiliateHandlerContext(http.MethodPost, "/api/v1/user/aff/bind-bonus/claim", "", 7)

	handler.ClaimAffiliateBindBonus(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(7), repo.claimUser)
	require.InDelta(t, 4.25, repo.claimAmount, 1e-9)
	require.InDelta(t, 12.34, gjson.Get(recorder.Body.String(), "data.balance").Float(), 1e-9)
	require.Equal(t, int64(42), gjson.Get(recorder.Body.String(), "data.detail.inviter_id").Int())
	require.False(t, gjson.Get(recorder.Body.String(), "data.detail.can_claim_bind_bonus").Bool())
}
