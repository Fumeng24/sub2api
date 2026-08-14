package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/model"
)

func TestOpenAIHandleErrorResponse_ContextWindowReturnsInvalidRequestWithoutFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	svc := &OpenAIGatewayService{}
	respBody := []byte(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"upstream_error","code":null}}`)
	resp := &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(bytes.NewReader(respBody)), Header: http.Header{}}
	account := &Account{ID: 14, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "invalid_request_error", errField["type"])
	assert.Equal(t, ContextWindowExceededClientMessage(""), errField["message"])
}

func TestGeminiWriteGeminiMappedError_MapsInvalidArgumentWrappedAs500To400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	svc := &GeminiMessagesCompatService{}
	respBody := []byte(`{"error":{"code":500,"message":"Request contains an invalid argument.","status":"INVALID_ARGUMENT"}}`)
	account := &Account{ID: 14, Platform: PlatformGemini, Type: AccountTypeAPIKey}

	err := svc.writeGeminiMappedError(c, account, http.StatusInternalServerError, "req-invalid", respBody)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "invalid_request_error", errField["type"])
	assert.Equal(t, "Invalid request", errField["message"])
}

func TestApplyErrorPassthroughRule_NormalizesInvalidResponseCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusUnprocessableEntity, "invalid schema", http.StatusProcessing, "联系QQ群群主解决")})
	BindErrorPassthroughService(c, ruleSvc)

	status, errType, errMsg, matched := applyErrorPassthroughRule(c, PlatformOpenAI, http.StatusUnprocessableEntity, []byte(`{"error":{"message":"invalid schema"}}`), http.StatusBadGateway, "upstream_error", "Upstream request failed")
	assert.True(t, matched)
	assert.Equal(t, http.StatusBadGateway, status)
	assert.Equal(t, "upstream_error", errType)
	assert.Equal(t, clientFacingTemporaryUnavailableMessage, errMsg)
}

func TestOpenAIHandleErrorResponse_BillingExhaustionBypassesPassthroughRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusPaymentRequired, "insufficient balance", http.StatusPaymentRequired, "上游余额不足")})
	BindErrorPassthroughService(c, ruleSvc)
	svc := &OpenAIGatewayService{}
	respBody := []byte(`{"error":{"message":"insufficient balance","type":"billing_error"}}`)
	resp := &http.Response{StatusCode: http.StatusPaymentRequired, Body: io.NopCloser(bytes.NewReader(respBody)), Header: http.Header{}}
	account := &Account{ID: 22, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	assert.Equal(t, http.StatusPaymentRequired, failoverErr.StatusCode)
	assert.False(t, c.Writer.Written())
	assert.NotContains(t, rec.Body.String(), "insufficient")
	assert.NotContains(t, rec.Body.String(), "余额")
}

func TestGeminiWriteGeminiMappedError_BillingExhaustionBypassesPassthroughRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusPaymentRequired, "insufficient balance", http.StatusPaymentRequired, "Gemini上游余额不足")})
	BindErrorPassthroughService(c, ruleSvc)
	svc := &GeminiMessagesCompatService{}
	respBody := []byte(`{"error":{"code":402,"message":"insufficient balance","status":"PAYMENT_REQUIRED"}}`)
	account := &Account{ID: 23, Platform: PlatformGemini, Type: AccountTypeAPIKey}

	err := svc.writeGeminiMappedError(c, account, http.StatusPaymentRequired, "req-billing", respBody)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.NotContains(t, rec.Body.String(), "insufficient")
	assert.NotContains(t, rec.Body.String(), "余额")
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, clientFacingTemporaryUnavailableMessage, errField["message"])
}
