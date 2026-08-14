package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayService_Forward_CompactBadOutputReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.4","stream":false,"instructions":"compact-test","input":"large context"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-compact-bad"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_bad","status":"completed","model":"gpt-5.4","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"tiny compact summary"}]}],"usage":{"input_tokens":418912,"output_tokens":4}}`)),
	}}

	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          4,
		Name:        "openai-compact-bad",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), openAICompactBadOutputCode)
	require.Empty(t, strings.TrimSpace(rec.Body.String()), "bad compact output must not be written to the client before failover")
}

func TestOpenAIGatewayService_Forward_CompactContextWindowReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.5","stream":false,"instructions":"compact-test","input":"large context"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := `{"error":{"type":"invalid_request_error","code":"context_length_exceeded","message":"Your input exceeds the context window."}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"rid-compact-context"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}

	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          5,
		Name:        "openai-compact-context",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadRequest, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "context window")
	require.Equal(t, "rid-compact-context", failoverErr.ResponseHeaders.Get("x-request-id"))
	require.Empty(t, strings.TrimSpace(rec.Body.String()), "compact context-window errors must be held for model fallback")
}

func TestOpenAIGatewayService_OAuthPassthrough_CompactContextWindowReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", bytes.NewReader(nil))
	c.Request.Header.Set("User-Agent", "codex_cli_rs/0.1.0")
	c.Request.Header.Set("Content-Type", "application/json")

	originalBody := []byte(`{"model":"gpt-5.5","stream":true,"instructions":"compact-test","input":[{"type":"text","text":"large context"}]}`)
	upstreamBody := `{"error":{"type":"invalid_request_error","message":"Your input exceeds the context window."}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"rid-compact-pass-context"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}

	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          6,
		Name:        "openai-compact-pass-context",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra:       map[string]any{"openai_passthrough": true},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, originalBody)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadRequest, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "context window")
	require.Equal(t, "rid-compact-pass-context", failoverErr.ResponseHeaders.Get("x-request-id"))
	require.Empty(t, strings.TrimSpace(rec.Body.String()), "passthrough compact context-window errors must not be written before fallback")
}

func TestOpenAICompactBadOutputReason_AllowsEncryptedContentCompaction(t *testing.T) {
	body := []byte(`{"object":"response.compaction","output":[{"type":"compaction","encrypted_content":"User requested: Respond with OK. Next assistant should reply OK."}],"usage":{"input_tokens":12,"output_tokens":18}}`)

	require.Empty(t, openAICompactBadOutputReason(body))
}

func TestOpenAICompactBadOutputReason_RejectsResponseWithoutCompactionItem(t *testing.T) {
	body := []byte(`{"id":"resp_bad","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"a sufficiently long response that is not a compaction item"}]},{"type":"message","content":[{"type":"output_text","text":"another normal output"}]},{"type":"message","content":[{"type":"output_text","text":"third normal output"}]}]}`)

	require.Equal(t, "missing_compaction_output_item", openAICompactBadOutputReason(body))
}

func TestOpenAICompactBadOutputReason_RejectsMultipleCompactionItems(t *testing.T) {
	body := []byte(`{"id":"resp_bad","status":"completed","output":[{"type":"compaction","encrypted_content":"one"},{"type":"compaction_summary","encrypted_content":"two"}]}`)

	require.Equal(t, "multiple_compaction_output_items", openAICompactBadOutputReason(body))
}

func TestIsOpenAIContextWindowError(t *testing.T) {
	require.True(t, IsOpenAIContextWindowErrorForTest(`{"error":{"message":"Your input exceeds the context window."}}`))
	require.True(t, isOpenAIContextWindowError("", []byte(`{"response":{"error":{"message":"too many input tokens"}}}`)))
	require.False(t, IsOpenAIContextWindowErrorForTest(`{"error":{"message":"model not found"}}`))
}
