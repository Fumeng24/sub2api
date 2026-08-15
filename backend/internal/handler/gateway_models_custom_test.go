package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayModels_OpenAIGroupFallsBackToDefaultModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(28)
	h := newGatewayModelsHandlerForTest(
		&gatewayModelsAccountRepoStub{
			byGroup: map[int64][]service.Account{
				groupID: {
					{ID: 1, Platform: service.PlatformOpenAI},
				},
			},
		},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{
		"codex-auto-review",
		"gpt-5.3-codex-spark",
		"gpt-5.6-sol",
		"gpt-5.6",
		"gpt-5.6-terra",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.5",
	}, modelIDsForTest(got.Data))
}
