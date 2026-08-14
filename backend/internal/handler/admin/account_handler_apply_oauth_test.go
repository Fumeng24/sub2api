package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPreserveOAuthCredentialSettings_PreservesModelMappingsOnlyWhenIncomingOmitsThem(t *testing.T) {
	existing := map[string]any{
		"access_token": "old-access",
		"model_mapping": map[string]any{
			"gpt-5.5": "gpt-5.5",
		},
		"compact_model_mapping": map[string]any{
			"gpt-5.5": "gpt-5.5-openai-compact",
		},
		"base_url": "https://old.example.com",
	}
	incoming := map[string]any{
		"access_token": "new-access",
		"model_mapping": map[string]any{
			"gpt-5.4": "gpt-5.4",
		},
	}

	got := preserveOAuthCredentialSettings(existing, incoming)

	require.Equal(t, "new-access", got["access_token"])
	require.Equal(t, map[string]any{"gpt-5.4": "gpt-5.4"}, got["model_mapping"])
	require.Equal(t, map[string]any{"gpt-5.5": "gpt-5.5-openai-compact"}, got["compact_model_mapping"])
	require.NotContains(t, got, "base_url")
	require.NotContains(t, got, "refresh_token")
}

func TestAccountHandlerApplyOAuthCredentials_PreservesModelMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:       42,
			Name:     "openai-oauth",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Credentials: map[string]any{
				"access_token": "old-access",
				"model_mapping": map[string]any{
					"gpt-5.5": "gpt-5.5",
				},
				"compact_model_mapping": map[string]any{
					"gpt-5.5": "gpt-5.5-openai-compact",
				},
			},
		},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/apply-oauth-credentials", handler.ApplyOAuthCredentials)

	body := map[string]any{
		"type": "oauth",
		"credentials": map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/42/apply-oauth-credentials", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.updatedAccounts, 1)

	credentials := adminSvc.updatedAccounts[0].Credentials
	require.Equal(t, "new-access", credentials["access_token"])
	require.Equal(t, "new-refresh", credentials["refresh_token"])
	require.Equal(t, map[string]any{"gpt-5.5": "gpt-5.5"}, credentials["model_mapping"])
	require.Equal(t, map[string]any{"gpt-5.5": "gpt-5.5-openai-compact"}, credentials["compact_model_mapping"])
}
