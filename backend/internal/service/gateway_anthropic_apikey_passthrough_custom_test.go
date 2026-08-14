package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestGatewayService_AnthropicAPIKeyPassthrough_UpstreamNetworkErrorReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	upstream := &anthropicHTTPUpstreamRecorder{err: errors.New("dial tcp 10.0.0.1:443: i/o timeout")}
	svc := &GatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := newAnthropicAPIKeyAccountForTest()
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4.5", Stream: false}

	result, err := svc.Forward(context.Background(), c, account, parsed)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, 0, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "i/o timeout")
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Equal(t, 0, rec.Body.Len(), "service failover path must not write the client response")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ResponseHeaderTimeoutSwitchesAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-opus-5","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"stream":true}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	upstream := &anthropicHTTPUpstreamRecorder{err: errors.New("http2: timeout awaiting response headers")}
	svc := &GatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := newAnthropicAPIKeyAccountForTest()
	account.Credentials["pool_mode"] = true
	account.Credentials["pool_mode_retry_count"] = float64(2)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-opus-5", Stream: true}

	result, err := svc.Forward(context.Background(), c, account, parsed)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.False(t, failoverErr.RetryableOnSameAccount, "header timeout must switch upstream before the CDN deadline")
	require.Equal(t, "transient_timeout", failoverErr.SchedulerCategory)
	require.Equal(t, 0, rec.Body.Len(), "failover candidate must not receive a committed client error")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_BadResponseStatus400ReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-opus-4-8","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"stream":true}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	upstreamBody := `{"error":{"message":"bad response status code 400 (request id: rid_400)","type":"bad_response_status_code"},"type":"error"}`
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"x-request-id": []string{"rid_400"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}},
			Gateway:  config.GatewayConfig{FailoverOn400: true, LogUpstreamErrorBody: true, LogUpstreamErrorBodyMaxBytes: 2048},
		},
		httpUpstream: upstream,
	}
	account := newAnthropicAPIKeyAccountForTest()
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-opus-4-8", Stream: true}

	result, err := svc.Forward(context.Background(), c, account, parsed)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadRequest, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "bad_response_status_code")
	require.Equal(t, 0, rec.Body.Len(), "failover candidate must not receive a committed client error")
	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "failover_on_400", events[0].Kind)
	require.True(t, events[0].Passthrough)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ClaudeCodeVersion400StaysClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-opus-4-8","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	versionMsg := "Your Claude Code version (2.1.31) is below the minimum required version (2.1.85). Please update: npm update -g @anthropic-ai/claude-code"
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"x-request-id": []string{"rid_version"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"` + versionMsg + `","type":"invalid_request_error"},"type":"error"}`)),
	}}
	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}},
			Gateway:  config.GatewayConfig{FailoverOn400: true, LogUpstreamErrorBody: true, LogUpstreamErrorBodyMaxBytes: 2048},
		},
		httpUpstream: upstream,
	}
	account := newAnthropicAPIKeyAccountForTest()
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-opus-4-8", Stream: false}

	result, err := svc.Forward(context.Background(), c, account, parsed)

	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), versionMsg)
	require.NotContains(t, rec.Body.String(), clientFacingTemporaryUnavailableMessage)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_MissingTerminalEventWithoutUsageReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	svc := &GatewayService{
		cfg:              &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		rateLimitService: &RateLimitService{},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}`,
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-3-7-sonnet-20250219")
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing terminal event")
	require.NotNil(t, result)
}
