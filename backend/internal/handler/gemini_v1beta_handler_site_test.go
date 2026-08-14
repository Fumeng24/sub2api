//go:build unit

package handler

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiGoogleErrorSanitizesInternalMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini:test", nil)

	googleError(c, http.StatusBadGateway, `Upstream request failed: account 38850 via https://upstream.example`)

	require.Equal(t, http.StatusBadGateway, w.Code)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &parsed))
	errorObj, ok := parsed["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, service.ClientFacingTemporaryUnavailableMessage(), errorObj["message"])
}

func TestWriteUpstreamResponse_NormalizesSensitiveUpstreamErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeUpstreamResponse(c, &service.UpstreamHTTPResult{
		StatusCode: http.StatusUnauthorized,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"error":{"code":401,"message":"Incorrect API key provided, request id: req_123","status":"UNAUTHENTICATED"}}`),
	})

	require.Equal(t, http.StatusBadGateway, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	errObj, ok := got["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(http.StatusBadGateway), errObj["code"])
	require.Equal(t, "Service temporarily unavailable, please retry later", errObj["message"])
	require.NotContains(t, w.Body.String(), "Incorrect API key")
	require.NotContains(t, w.Body.String(), "request id")
	require.NotContains(t, w.Body.String(), "UNAUTHENTICATED")
}
