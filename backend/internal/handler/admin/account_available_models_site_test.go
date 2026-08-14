package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerGetAvailableModels_OpenAIApiKeyWithoutMappingUsesDefaultModels(t *testing.T) {
	svc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       47,
			Name:     "openai-apikey-default",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeAPIKey,
			Status:   service.StatusActive,
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/47/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	gotIDs := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		gotIDs = append(gotIDs, model.ID)
	}
	require.Equal(t, []string{
		"codex-auto-review",
		"gpt-5.3-codex-spark",
		"gpt-5.6-sol",
		"gpt-5.6",
		"gpt-5.6-terra",
		"gpt-5.6-luna",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.5",
	}, gotIDs)
}

func TestAccountHandlerGetAvailableModels_OpenAIMappingIsSortedAndSkipsBlankKeys(t *testing.T) {
	svc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       46,
			Name:     "openai-mapped",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeAPIKey,
			Status:   service.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"z-model": "gpt-5.1",
					" ":       "gpt-5",
					"a-model": "gpt-5",
				},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/46/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, []struct {
		ID string `json:"id"`
	}{{ID: "a-model"}, {ID: "z-model"}}, resp.Data)
}
