package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIHandleErrorResponseImageBridgeUnsupportedAutoDisablesAndFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &snapshotUpdateAccountRepo{updateExtraCalls: make(chan map[string]any, 1)}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: repo}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	account := &Account{
		ID:       38808,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Name:     "acc",
		Extra:    map[string]any{featureKeyCodexImageGenerationBridge: true},
	}
	respBody := []byte(`{"error":{"message":"Image generation is not enabled for this group"}}`)
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-image-bridge-http"}},
	}

	bridgeCtx := withOpenAICodexImageBridgeApplied(context.Background())
	result, err := svc.handleErrorResponse(bridgeCtx, resp, c, account, []byte(`{"model":"gpt-5.4"}`))
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusForbidden, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "Image generation is not enabled")
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.Body.String())
	require.Equal(t, false, account.Extra[featureKeyCodexImageGenerationBridge])

	select {
	case updates := <-repo.updateExtraCalls:
		require.Equal(t, false, updates[featureKeyCodexImageGenerationBridge])
	case <-time.After(2 * time.Second):
		t.Fatal("expected UpdateExtra to be called")
	}
}

func TestWriteCustomOpenAIClassifiedErrorLeavesUnknownClassToOfficialWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	require.False(t, (&OpenAIGatewayService{}).writeCustomOpenAIClassifiedError(c, http.StatusNotFound, "resource missing", []byte(`{"error":{"message":"resource missing"}}`)))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestWriteCustomOpenAIClassifiedErrorHandlesBillingClass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	require.True(t, (&OpenAIGatewayService{}).writeCustomOpenAIClassifiedError(c, http.StatusPaymentRequired, "insufficient balance", nil))
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Equal(t, "upstream_error", gjson.Get(rec.Body.String(), "error.type").String())
}

func TestOpenAIHandleErrorResponseGroupDisabledFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1}}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitSvc.SetOpenAI403CounterCache(counter)
	svc := &OpenAIGatewayService{cfg: &config.Config{}, rateLimitService: rateLimitSvc}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)

	account := &Account{
		ID:       38808,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Name:     "group-disabled",
	}
	respBody := []byte(`{"code":"GROUP_DISABLED","message":"API Key 所属分组已停用"}`)
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-group-disabled"}},
	}

	result, err := svc.handleErrorResponse(context.Background(), resp, c, account, []byte(`{"model":"gpt-5.5"}`))
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusForbidden, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "GROUP_DISABLED")
	require.Empty(t, failoverErr.SchedulerCategory)
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.Body.String())
	require.Zero(t, repo.tempCalls)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Contains(t, repo.lastErrorMsg, "API Key 所属分组已停用")
	require.Zero(t, counter.lastCount)
}

func TestOpenAIHandleErrorResponseContextWindowIsClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc := &OpenAIGatewayService{cfg: &config.Config{}, rateLimitService: rateLimitSvc}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)

	account := &Account{
		ID:       38809,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Name:     "context-window",
	}
	respBody := []byte(`{"error":{"type":"invalid_request_error","message":"prompt is too long: 2318541 tokens > 1000000 maximum"}}`)
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-context-window"}},
	}

	result, err := svc.handleErrorResponse(context.Background(), resp, c, account, []byte(`{"model":"gpt-5.5"}`), "gpt-5.5")
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr), "context window is a client request error, not account failover")
	var terminalErr *UpstreamTerminalError
	require.ErrorAs(t, err, &terminalErr)
	require.Equal(t, http.StatusBadRequest, terminalErr.StatusCode)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Contains(t, rec.Body.String(), "272k tokens")
	require.Zero(t, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
	require.True(t, HasOpsClientBusinessLimited(c))
}

func TestOpenAIHandleErrorResponseTemporary403UsesProbeCircuit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1}}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitSvc.SetOpenAI403CounterCache(counter)
	svc := &OpenAIGatewayService{cfg: &config.Config{}, rateLimitService: rateLimitSvc}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)

	account := &Account{
		ID:       38804,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Name:     "temporary-403",
	}
	respBody := []byte(`{"error":{"message":"403错误，请稍后再试"}}`)
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-temporary-403"}},
	}

	result, err := svc.handleErrorResponse(context.Background(), resp, c, account, []byte(`{"model":"gpt-5.5"}`))
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusForbidden, failoverErr.StatusCode)
	require.Equal(t, "transient", failoverErr.SchedulerCategory)
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.Body.String())
	require.Zero(t, repo.tempCalls)
	require.Zero(t, repo.setErrorCalls)
	require.Equal(t, int64(1), counter.lastCount)
}

func TestOpenAIHandleErrorResponseUpstreamAccessForbiddenFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)

	account := &Account{
		ID:       38819,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Name:     "forbidden-upstream",
	}
	respBody := []byte(`{"error":{"message":"Upstream access forbidden, please contact administrator","type":"upstream_error"}}`)
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-upstream-forbidden"}},
	}

	result, err := svc.handleErrorResponse(context.Background(), resp, c, account, []byte(`{"model":"gpt-5.4"}`))
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "Upstream access forbidden")
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.Body.String())
}

func TestOpenAIHandleCompatErrorResponseGroupDisabledFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1}}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitSvc.SetOpenAI403CounterCache(counter)
	svc := &OpenAIGatewayService{cfg: &config.Config{}, rateLimitService: rateLimitSvc}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	account := &Account{
		ID:       38808,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Name:     "group-disabled",
	}
	respBody := []byte(`{"code":"GROUP_DISABLED","message":"API Key 所属分组已停用"}`)
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-group-disabled-compat"}},
	}
	writeCalled := false

	result, err := svc.handleCompatErrorResponse(
		resp,
		c,
		account,
		func(c *gin.Context, statusCode int, errType, message string) {
			writeCalled = true
		},
		"gpt-5.5",
	)
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusForbidden, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "GROUP_DISABLED")
	require.Empty(t, failoverErr.SchedulerCategory)
	require.False(t, writeCalled)
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.Body.String())
	require.Zero(t, repo.tempCalls)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Contains(t, repo.lastErrorMsg, "API Key 所属分组已停用")
}

func TestOpenAIHandleCompatErrorResponseUpstreamAccessForbiddenFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	account := &Account{
		ID:       38819,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Name:     "forbidden-upstream",
	}
	respBody := []byte(`{"error":{"message":"Upstream access forbidden, please contact administrator","type":"upstream_error"}}`)
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{"X-Request-Id": []string{"rid-upstream-forbidden-compat"}},
	}
	writeCalled := false

	result, err := svc.handleCompatErrorResponse(
		resp,
		c,
		account,
		func(c *gin.Context, statusCode int, errType, message string) {
			writeCalled = true
		},
		"gpt-5.4",
	)
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "Upstream access forbidden")
	require.False(t, writeCalled)
	require.False(t, c.Writer.Written())
	require.Empty(t, rec.Body.String())
}
