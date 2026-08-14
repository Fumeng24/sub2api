//go:build unit

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// normalizeAPIContractResponseCustom removes site-only response extensions before
// the unchanged upstream snapshots are compared. The extensions are asserted
// independently below so upstream snapshot updates remain directly mergeable.
func normalizeAPIContractResponseCustom(t *testing.T, testName, body string) string {
	t.Helper()

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &response))
	data := response["data"]

	switch testName {
	case "POST /api/v1/keys":
		delete(data.(map[string]any), "category")
	case "GET /api/v1/keys (paginated)":
		for _, item := range data.(map[string]any)["items"].([]any) {
			delete(item.(map[string]any), "category")
		}
	case "GET /api/v1/groups/available":
		for _, item := range data.([]any) {
			group := item.(map[string]any)
			delete(group, "force_openai_priority")
			delete(group, "openai_stable_low_ttft")
			delete(group, "models_list_config")
		}
	case "GET /api/v1/subscriptions":
		for _, item := range data.([]any) {
			delete(item.(map[string]any), "auto_reset_daily")
		}
	case "GET /api/v1/redeem/history":
		for _, item := range data.([]any) {
			delete(item.(map[string]any), "business_category")
		}
	case "GET /api/v1/admin/settings", "GET /api/v1/admin/settings falls back to config oauth defaults":
		settings := data.(map[string]any)
		delete(settings, "affiliate_bind_bonus_amount")
		delete(settings, "payment_balance_recharge_unlock_threshold")
		delete(settings, "group_rate_discount_settings")
		delete(settings, "ticket_system_config")
		settings["fallback_model_openai"] = "gpt-4o"
		settings["login_agreement_updated_at"] = "2026-03-31"
		for _, document := range settings["login_agreement_documents"].([]any) {
			document.(map[string]any)["content_md"] = ""
		}
	}

	normalized, err := json.Marshal(response)
	require.NoError(t, err)
	return string(normalized)
}

func TestAPIContractsCustomResponseExtensions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("api key category", func(t *testing.T) {
		deps := newContractDeps(t)
		status, body := doRequest(t, deps.router, http.MethodPost, "/api/v1/keys", `{"name":"Key One","custom_key":"sk_custom_1234567890"}`, map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "other", responseDataMap(t, body)["category"])
	})

	t.Run("group scheduling fields", func(t *testing.T) {
		deps := newContractDeps(t)
		deps.groupRepo.SetActive([]service.Group{{
			ID: 10, Name: "Group One", Platform: service.PlatformAnthropic,
			Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard,
		}})
		status, body := doRequest(t, deps.router, http.MethodGet, "/api/v1/groups/available", "", nil)
		require.Equal(t, http.StatusOK, status)
		group := responseDataSlice(t, body)[0].(map[string]any)
		require.Equal(t, false, group["force_openai_priority"])
		require.Equal(t, false, group["openai_stable_low_ttft"])
		require.Equal(t, map[string]any{"enabled": false}, group["models_list_config"])
	})

	t.Run("subscription auto reset", func(t *testing.T) {
		deps := newContractDeps(t)
		deps.userSubRepo.SetByUserID(1, []service.UserSubscription{{
			ID: 501, UserID: 1, GroupID: 10, StartsAt: deps.now,
			ExpiresAt: time.Date(2099, 1, 2, 3, 4, 5, 0, time.UTC),
			Status:    service.SubscriptionStatusActive,
		}})
		status, body := doRequest(t, deps.router, http.MethodGet, "/api/v1/subscriptions", "", nil)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, false, responseDataSlice(t, body)[0].(map[string]any)["auto_reset_daily"])
	})

	t.Run("redeem business category", func(t *testing.T) {
		deps := newContractDeps(t)
		deps.redeemRepo.SetByUser(1, []service.RedeemCode{{
			ID: 900, Code: "CODE-123", Type: service.RedeemTypeBalance,
			Value: 1.25, Status: service.StatusUsed, UsedBy: ptr(int64(1)), CreatedAt: deps.now,
		}})
		status, body := doRequest(t, deps.router, http.MethodGet, "/api/v1/redeem/history", "", nil)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "", responseDataSlice(t, body)[0].(map[string]any)["business_category"])
	})

	t.Run("admin site settings", func(t *testing.T) {
		deps := newContractDeps(t)
		deps.settingRepo.SetAll(map[string]string{
			service.SettingKeyLoginAgreementDocuments: `[{"id":"terms","title":"Terms","content_md":"terms test"}]`,
		})
		status, body := doRequest(t, deps.router, http.MethodGet, "/api/v1/admin/settings", "", nil)
		require.Equal(t, http.StatusOK, status)
		settings := responseDataMap(t, body)
		require.Equal(t, "gpt-5.4-mini", settings["fallback_model_openai"])
		require.Contains(t, settings, "affiliate_bind_bonus_amount")
		require.Contains(t, settings, "payment_balance_recharge_unlock_threshold")
		require.Contains(t, settings, "group_rate_discount_settings")
		require.Contains(t, settings, "ticket_system_config")
		documents := settings["login_agreement_documents"].([]any)
		require.Equal(t, "terms test", documents[0].(map[string]any)["content_md"])
	})
}

func TestAPIContractCustomSubscriptionCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deps := newContractDeps(t)
	deps.groupRepo.SetActive([]service.Group{{
		ID: 21, Name: "Internal Subscription Group", Platform: service.PlatformAnthropic,
		Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeSubscription,
	}})

	apiKeyService := service.NewAPIKeyService(
		deps.apiKeyRepo, &stubUserRepo{}, deps.groupRepo, deps.userSubRepo,
		nil, stubApiKeyCache{}, deps.cfg,
	)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	router := gin.New()
	router.GET("/api/v1/groups/subscription-capability", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		apiKeyHandler.GetSubscriptionCapability(c)
	})

	status, body := doRequest(t, router, http.MethodGet, "/api/v1/groups/subscription-capability", "", nil)
	require.Equal(t, http.StatusOK, status)
	require.JSONEq(t, `{"code":0,"message":"success","data":{"has_subscription_groups":true}}`, body)
}

func responseDataMap(t *testing.T, body string) map[string]any {
	t.Helper()
	return decodeAPIContractResponse(t, body)["data"].(map[string]any)
}

func responseDataSlice(t *testing.T, body string) []any {
	t.Helper()
	return decodeAPIContractResponse(t, body)["data"].([]any)
}

func decodeAPIContractResponse(t *testing.T, body string) map[string]any {
	t.Helper()
	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &response))
	return response
}

// Local repository interfaces extend their upstream counterparts. Keeping the
// extra stub methods here leaves the upstream contract fixture merge-friendly.
func (stubRedeemCodeRepo) ListByIDs(context.Context, []int64) ([]service.RedeemCode, error) {
	return nil, errors.New("not implemented")
}

func (stubUserSubscriptionRepo) UpdateAutoResetDaily(context.Context, int64, bool) error {
	return errors.New("not implemented")
}

func (stubUserSubscriptionRepo) ShortenExpiryAndResetDaily(context.Context, int64, time.Time, time.Time, time.Time) (bool, error) {
	return false, nil
}

func (*stubUsageLogRepo) GetRecentAccountFirstTokenStats(context.Context, []int64, int64, time.Time) (map[int64]service.AccountRecentFirstTokenStats, error) {
	return map[int64]service.AccountRecentFirstTokenStats{}, nil
}

func (*stubUsageLogRepo) ListRecentGroupUsers(context.Context, int64, time.Time, int) ([]service.GroupRecentUserUsage, error) {
	return nil, nil
}
