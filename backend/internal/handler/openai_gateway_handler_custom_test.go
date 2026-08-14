package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIHandleFailoverExhausted_ModelUnsupportedReturnsGroupUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	h := &OpenAIGatewayHandler{}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusNotFound,
		ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"model not found"}}`),
	}, false)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Equal(t, "api_error", gjson.Get(w.Body.String(), "error.type").String())
	require.Equal(t, service.ClientFacingGroupUnavailableMessage(), gjson.Get(w.Body.String(), "error.message").String())
}

func TestOpenAIWebSocketErrorPayloadSanitizesInternalMessages(t *testing.T) {
	payload := buildOpenAIWebSocketErrorPayload("invalid_request_error", `Upstream request failed: account 38850 via https://upstream.example`)

	require.True(t, gjson.ValidBytes(payload))
	require.Equal(t, "error", gjson.GetBytes(payload, "type").String())
	require.Equal(t, "invalid_request_error", gjson.GetBytes(payload, "error.type").String())
	require.Equal(t, service.ClientFacingTemporaryUnavailableMessage(), gjson.GetBytes(payload, "error.message").String())
	require.NotContains(t, string(payload), "38850")
	require.NotContains(t, string(payload), "https://upstream.example")
	require.NotContains(t, string(payload), "Upstream")
}

func TestOpenAIWSCloseReasonSanitizesInternalMessages(t *testing.T) {
	require.Equal(t, "invalid JSON payload", sanitizeOpenAIWSCloseReason(coderws.StatusPolicyViolation, "invalid JSON payload"))
	require.Equal(t, service.ClientFacingTemporaryUnavailableMessage(), sanitizeOpenAIWSCloseReason(coderws.StatusTryAgainLater, "account is busy, please retry later"))
	require.Equal(t, service.ClientFacingTemporaryUnavailableMessage(), sanitizeOpenAIWSCloseReason(coderws.StatusInternalError, "upstream websocket proxy failed"))

	status, reason := openAIWSFailoverCloseStatusAndReason(&service.UpstreamFailoverError{StatusCode: http.StatusTooManyRequests})
	require.Equal(t, coderws.StatusTryAgainLater, status)
	require.Equal(t, service.ClientFacingTemporaryUnavailableMessage(), reason)

	status, reason = openAIWSFailoverCloseStatusAndReason(&service.UpstreamFailoverError{
		StatusCode:   http.StatusBadRequest,
		ResponseBody: []byte(`{"error":{"message":"model not found"}}`),
	})
	require.Equal(t, coderws.StatusTryAgainLater, status)
	require.Equal(t, service.ClientFacingGroupUnavailableMessage(), reason)
}

func TestOpenAIHandleFailoverExhausted_NilErrorFallsBack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	h := &OpenAIGatewayHandler{}
	h.handleFailoverExhausted(c, nil, false)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Equal(t, "upstream_error", gjson.Get(w.Body.String(), "error.type").String())
	require.NotEmpty(t, gjson.Get(w.Body.String(), "error.message").String())
}

func TestOpenAIHandleAnthropicFailoverExhausted_ModelUnsupportedReturnsGroupUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	h := &OpenAIGatewayHandler{}
	h.handleAnthropicFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusServiceUnavailable,
		ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"No available channel for model gpt-5.5"}}`),
	}, false)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Equal(t, "error", gjson.Get(w.Body.String(), "type").String())
	require.Equal(t, "api_error", gjson.Get(w.Body.String(), "error.type").String())
	require.Equal(t, service.ClientFacingGroupUnavailableMessage(), gjson.Get(w.Body.String(), "error.message").String())
}

func TestOpenAIHandleAnthropicFailoverExhausted_NilErrorFallsBack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	h := &OpenAIGatewayHandler{}
	h.handleAnthropicFailoverExhausted(c, nil, false)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Equal(t, "error", gjson.Get(w.Body.String(), "type").String())
	require.Equal(t, "api_error", gjson.Get(w.Body.String(), "error.type").String())
	require.Equal(t, service.ClientFacingTemporaryUnavailableMessage(), gjson.Get(w.Body.String(), "error.message").String())
}

func TestCloseOpenAIWSFailoverExhausted_ModelUnsupportedUsesRetryableClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			return
		}
		closeOpenAIWSFailoverExhausted(conn, &service.UpstreamFailoverError{
			StatusCode:   http.StatusNotFound,
			ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"model not found"}}`),
		})
	}))
	defer server.Close()

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	cancelDial()
	require.NoError(t, err)
	defer func() {
		_ = clientConn.CloseNow()
	}()

	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	_, _, err = clientConn.Read(readCtx)
	cancelRead()
	require.Error(t, err)
	var closeErr coderws.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, coderws.StatusTryAgainLater, closeErr.Code)
	require.Equal(t, service.ClientFacingGroupUnavailableMessage(), closeErr.Reason)
}
