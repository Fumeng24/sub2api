package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type geminiCompatHTTPUpstreamStub struct {
	response *http.Response
	err      error
	calls    int
	lastReq  *http.Request
}

func (s *geminiCompatHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	s.calls++
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	if s.response == nil {
		return nil, fmt.Errorf("missing stub response")
	}
	resp := *s.response
	return &resp, nil
}

func (s *geminiCompatHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

type geminiCompatBillingRepoStub struct {
	AccountRepository
	setErrCalls  int
	lastErrorMsg string
}

func (r *geminiCompatBillingRepoStub) SetError(_ context.Context, _ int64, errorMsg string) error {
	r.setErrCalls++
	r.lastErrorMsg = errorMsg
	return nil
}

func TestGeminiForwardAsChatCompletions_UpstreamNetworkErrorReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpStub := &geminiCompatHTTPUpstreamStub{err: errors.New("dial tcp 10.0.0.1:443: i/o timeout")}
	svc := &GeminiMessagesCompatService{
		httpUpstream: httpStub,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}
	account := &Account{
		ID:          38802,
		Platform:    PlatformGemini,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "gemini-key"},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, 0, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "i/o timeout")
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Equal(t, geminiMaxRetries, httpStub.calls)
	require.Equal(t, 0, rec.Body.Len(), "service failover path must not write the client response")
}

func TestGeminiForwardAsChatCompletions_BillingExhaustionReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusPaymentRequired,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"gemini-billing"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":402,"message":"insufficient balance","status":"PAYMENT_REQUIRED"}}`)),
		},
	}
	repo := &geminiCompatBillingRepoStub{}
	svc := &GeminiMessagesCompatService{
		httpUpstream:     httpStub,
		rateLimitService: NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
		cfg:              &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}
	account := &Account{
		ID:          38803,
		Platform:    PlatformGemini,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "gemini-key"},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusPaymentRequired, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "insufficient balance")
	require.Equal(t, 1, httpStub.calls)
	require.Equal(t, 1, repo.setErrCalls)
	require.False(t, c.Writer.Written())
	require.NotContains(t, rec.Body.String(), "insufficient")
	require.NotContains(t, rec.Body.String(), "balance")
}

func TestGeminiForwardNative_SkippedCustomErrorCodePreservesClientBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := `{"error":{"code":400,"message":"Request contains an invalid argument.","status":"INVALID_ARGUMENT"}}`
	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"gemini-rid"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}
	repo := &geminiCompatBillingRepoStub{}
	svc := &GeminiMessagesCompatService{
		httpUpstream:     httpStub,
		rateLimitService: NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
		cfg:              &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}
	account := &Account{
		ID:          38831,
		Platform:    PlatformGemini,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":                    "gemini-key",
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(http.StatusTooManyRequests)},
		},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3.5-flash:generateContent", bytes.NewReader(body))

	result, err := svc.ForwardNative(context.Background(), c, account, "gemini-3.5-flash", "generateContent", false, body)

	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "Invalid request", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	require.NotContains(t, rec.Body.String(), "invalid argument")
	require.NotContains(t, rec.Body.String(), "gemini-rid")
	require.Equal(t, 1, httpStub.calls)
}

func TestGeminiChatCompletionsMappedErrorHidesSensitiveUpstreamStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus int
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"code":401,"message":"Incorrect API key provided: sk-test, request id: req_123","status":"UNAUTHENTICATED"}}`,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "rate_limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"code":429,"message":"Upstream rate limit exceeded, cf-ray: abc","status":"RESOURCE_EXHAUSTED"}}`,
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			svc := &GeminiMessagesCompatService{}
			err := svc.writeGeminiChatCompletionsMappedError(c, &Account{ID: 1, Platform: PlatformGemini}, tt.statusCode, "rid", []byte(tt.body))

			require.Error(t, err)
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, "upstream_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
			require.Equal(t, clientFacingTemporaryUnavailableMessage, gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
			for _, leaked := range []string{"Incorrect API key", "sk-test", "request id", "cf-ray", "rate limit"} {
				require.NotContains(t, rec.Body.String(), leaked)
			}
		})
	}
}

func TestGeminiForwardAsChatCompletions_OAuthRoutesToGeminiAndReturnsChatFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := `data: {"response":{"candidates":[{"content":{"parts":[{"text":"hello from gemini"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":7,"candidatesTokenCount":3}}}` + "\n\n" +
		"data: [DONE]\n\n"
	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}
	svc := &GeminiMessagesCompatService{
		tokenProvider: &GeminiTokenProvider{},
		httpUpstream:  httpStub,
		cfg:           &config.Config{},
	}
	account := &Account{
		ID:       101,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "ya29.test-token",
			"project_id":   "project-1",
		},
		Concurrency: 1,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gemini-2.5-flash", result.Model)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.OutputTokens)

	require.NotNil(t, httpStub.lastReq)
	require.Contains(t, httpStub.lastReq.URL.String(), "/v1internal:streamGenerateContent?alt=sse")
	require.Equal(t, "Bearer ya29.test-token", httpStub.lastReq.Header.Get("Authorization"))
	require.Empty(t, httpStub.lastReq.Header.Get("x-api-key"))
	require.Empty(t, httpStub.lastReq.Header.Get("anthropic-version"))

	var sent map[string]any
	sentBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(sentBody, &sent))
	require.Equal(t, "gemini-2.5-flash", sent["model"])
	require.Equal(t, "project-1", sent["project"])
	require.Contains(t, fmt.Sprint(sent["request"]), "hi")

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "chat.completion", got["object"])
	require.Equal(t, "gemini-2.5-flash", got["model"])
	choices, ok := got["choices"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, choices)
	choice, ok := choices[0].(map[string]any)
	require.True(t, ok)
	message, ok := choice["message"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "assistant", message["role"])
	require.Equal(t, "hello from gemini", message["content"])
	usage, ok := got["usage"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(7), usage["prompt_tokens"])
	require.Equal(t, float64(3), usage["completion_tokens"])
	require.Equal(t, float64(10), usage["total_tokens"])
}

func TestGeminiForwardAsChatCompletions_StreamsOpenAIChunksFromGeminiSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := `data: {"candidates":[{"content":{"parts":[{"text":"hel"}]}}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":1}}` + "\n\n" +
		`data: {"candidates":[{"content":{"parts":[{"text":"hello"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":2}}` + "\n\n" +
		"data: [DONE]\n\n"
	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}
	svc := &GeminiMessagesCompatService{
		httpUpstream: httpStub,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       102,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "gemini-api-key",
		},
		Concurrency: 1,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gemini-2.5-flash","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hi"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, result.Stream)
	require.Equal(t, 2, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)

	require.NotNil(t, httpStub.lastReq)
	require.Contains(t, httpStub.lastReq.URL.String(), "/v1beta/models/gemini-2.5-flash:streamGenerateContent?alt=sse")
	require.Equal(t, "gemini-api-key", httpStub.lastReq.Header.Get("x-goog-api-key"))

	out := rec.Body.String()
	require.Contains(t, out, `"object":"chat.completion.chunk"`)
	require.Contains(t, out, `"role":"assistant"`)
	require.Contains(t, out, `"content":"hel"`)
	require.Contains(t, out, `"content":"lo"`)
	require.Contains(t, out, `"usage":{"prompt_tokens":2,"completion_tokens":2,"total_tokens":4}`)
	require.Contains(t, out, "data: [DONE]")
}

func TestGeminiForwardAsChatCompletions_AddsToolConfigForWebSearchAndFunctions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"gemini-tools"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2}}`)),
		},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		ID:       1,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "gemini-key",
		},
	}
	body := []byte(`{"model":"gemini-3.5-flash","messages":[{"role":"user","content":"search then call"}],"tools":[{"type":"function","function":{"name":"web_search","description":"Search the web","parameters":{"type":"object","properties":{"query":{"type":"string"}}}}},{"type":"function","function":{"name":"get_weather","description":"Get weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}],"stream":false}`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, httpStub.lastReq)

	postedBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(postedBody, `toolConfig.includeServerSideToolInvocations`).Bool(), string(postedBody))
	require.True(t, gjson.GetBytes(postedBody, `tools.#.google_search`).Exists(), string(postedBody))
	require.True(t, gjson.GetBytes(postedBody, `tools.#.functionDeclarations`).Exists(), string(postedBody))
}

func TestGeminiForwardAsChatCompletions_ImageAPIKeyUsesRawChatCompletions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"gemini-image-raw"}},
			Body: io.NopCloser(strings.NewReader(`{
				"id":"chatcmpl_img",
				"object":"chat.completion",
				"model":"gemini-3.1-flash-image",
				"choices":[{
					"index":0,
					"message":{
						"role":"assistant",
						"content":"",
						"images":[{"image_url":{"url":"data:image/jpeg;base64,aGVsbG8="}}]
					},
					"finish_reason":"stop"
				}],
				"usage":{"prompt_tokens":5,"completion_tokens":7,"total_tokens":12}
			}`)),
		},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		ID:       38847,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "gemini-key",
			"base_url": "https://compat.example/image",
		},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gemini-3.1-flash-image","messages":[{"role":"user","content":"draw a cat"}],"stream":false,"size":"4096x4096","image_size":"4K","aspect_ratio":"1:1","n":1}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gemini-3.1-flash-image", result.Model)
	require.Equal(t, "gemini-3.1-flash-image", result.UpstreamModel)
	require.Equal(t, 5, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, "4K", result.ImageSize)
	require.Equal(t, "4K", result.ImageInputSize)
	require.Contains(t, rec.Body.String(), `"images"`)
	require.Contains(t, rec.Body.String(), `data:image/jpeg;base64,aGVsbG8=`)

	require.NotNil(t, httpStub.lastReq)
	require.Equal(t, "https://compat.example/image/v1/chat/completions", httpStub.lastReq.URL.String())
	require.Equal(t, "Bearer gemini-key", httpStub.lastReq.Header.Get("Authorization"))
	require.Empty(t, httpStub.lastReq.Header.Get("x-goog-api-key"))
	postedBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)
	require.Equal(t, "gemini-3.1-flash-image", gjson.GetBytes(postedBody, "model").String())
	require.Equal(t, "4096x4096", gjson.GetBytes(postedBody, "size").String())
	require.Equal(t, "4K", gjson.GetBytes(postedBody, "image_size").String())
	require.Equal(t, "1:1", gjson.GetBytes(postedBody, "aspect_ratio").String())
	require.False(t, gjson.GetBytes(postedBody, "n").Exists(), string(postedBody))
}

// TestConvertClaudeToolsToGeminiTools_CustomType 测试custom类型工具转换
func TestConvertClaudeToolsToGeminiTools_CustomType(t *testing.T) {
	tests := []struct {
		name        string
		tools       any
		expectedLen int
		description string
	}{
		{
			name: "Standard tools",
			tools: []any{
				map[string]any{
					"name":         "get_weather",
					"description":  "Get weather info",
					"input_schema": map[string]any{"type": "object"},
				},
			},
			expectedLen: 1,
			description: "标准工具格式应该正常转换",
		},
		{
			name: "Custom type tool (MCP format)",
			tools: []any{
				map[string]any{
					"type": "custom",
					"name": "mcp_tool",
					"custom": map[string]any{
						"description":  "MCP tool description",
						"input_schema": map[string]any{"type": "object"},
					},
				},
			},
			expectedLen: 1,
			description: "Custom类型工具应该从custom字段读取",
		},
		{
			name: "Mixed standard and custom tools",
			tools: []any{
				map[string]any{
					"name":         "standard_tool",
					"description":  "Standard",
					"input_schema": map[string]any{"type": "object"},
				},
				map[string]any{
					"type": "custom",
					"name": "custom_tool",
					"custom": map[string]any{
						"description":  "Custom",
						"input_schema": map[string]any{"type": "object"},
					},
				},
			},
			expectedLen: 1,
			description: "混合工具应该都能正确转换",
		},
		{
			name: "Custom tool without custom field",
			tools: []any{
				map[string]any{
					"type": "custom",
					"name": "invalid_custom",
					// 缺少 custom 字段
				},
			},
			expectedLen: 0, // 应该被跳过
			description: "缺少custom字段的custom工具应该被跳过",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertClaudeToolsToGeminiTools(tt.tools)

			if tt.expectedLen == 0 {
				if result != nil {
					t.Errorf("%s: expected nil result, got %v", tt.description, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("%s: expected non-nil result", tt.description)
			}

			if len(result) != 1 {
				t.Errorf("%s: expected 1 tool declaration, got %d", tt.description, len(result))
				return
			}

			toolDecl, ok := result[0].(map[string]any)
			if !ok {
				t.Fatalf("%s: result[0] is not map[string]any", tt.description)
			}

			funcDecls, ok := toolDecl["functionDeclarations"].([]any)
			if !ok {
				t.Fatalf("%s: functionDeclarations is not []any", tt.description)
			}

			toolsArr, _ := tt.tools.([]any)
			expectedFuncCount := 0
			for _, tool := range toolsArr {
				toolMap, _ := tool.(map[string]any)
				if toolMap["name"] != "" {
					// 检查是否为有效的custom工具
					if toolMap["type"] == "custom" {
						if toolMap["custom"] != nil {
							expectedFuncCount++
						}
					} else {
						expectedFuncCount++
					}
				}
			}

			if len(funcDecls) != expectedFuncCount {
				t.Errorf("%s: expected %d function declarations, got %d",
					tt.description, expectedFuncCount, len(funcDecls))
			}
		})
	}
}

func TestConvertClaudeToolsToGeminiTools_PreservesWebSearchAlongsideFunctions(t *testing.T) {
	tools := []any{
		map[string]any{
			"name":         "get_weather",
			"description":  "Get weather info",
			"input_schema": map[string]any{"type": "object"},
		},
		map[string]any{
			"type": "web_search_20250305",
			"name": "web_search",
		},
	}

	result := convertClaudeToolsToGeminiTools(tools)
	require.Len(t, result, 2)

	functionDecl, ok := result[0].(map[string]any)
	require.True(t, ok)
	funcDecls, ok := functionDecl["functionDeclarations"].([]any)
	require.True(t, ok)
	require.Len(t, funcDecls, 1)

	searchDecl, ok := result[1].(map[string]any)
	require.True(t, ok)
	googleSearch, ok := searchDecl["googleSearch"].(map[string]any)
	require.True(t, ok)
	require.Empty(t, googleSearch)
}

func TestConvertClaudeMessagesToGeminiGenerateContent_AddsMixedToolConfig(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"tools":[{"name":"get_weather","description":"Get weather info","input_schema":{"type":"object"}},{"type":"web_search_20250305","name":"web_search"}]}`)

	geminiBody, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(geminiBody, `toolConfig.includeServerSideToolInvocations`).Bool(), string(geminiBody))
}

func TestConvertClaudeMessagesToGeminiGenerateContent_DoesNotAddMixedToolConfigForFunctionsOnly(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"tools":[{"name":"get_weather","description":"Get weather info","input_schema":{"type":"object"}}]}`)

	geminiBody, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(geminiBody, `toolConfig.includeServerSideToolInvocations`).Exists(), string(geminiBody))
}

func TestGeminiHandleNativeNonStreamingResponse_DebugDisabledDoesNotEmitHeaderLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logSink, restore := captureStructuredLog(t)
	defer restore()

	svc := &GeminiMessagesCompatService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				GeminiDebugResponseHeaders: false,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type":      []string{"application/json"},
			"X-RateLimit-Limit": []string{"60"},
		},
		Body: io.NopCloser(strings.NewReader(`{"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2}}`)),
	}

	result, err := svc.handleNativeNonStreamingResponse(c, resp, false, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.False(t, logSink.ContainsMessage("[GeminiAPI]"), "debug 关闭时不应输出 Gemini 响应头日志")
}

func TestGeminiMessagesCompatServiceForwardNative_ImageModelEmptyPartsTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3.1-flash-image:generateContent", nil)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1473}}`)),
		},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		ID:       38847,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-key",
		},
	}
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"draw a cat"}]}],"generationConfig":{"responseModalities":["TEXT","IMAGE"],"imageConfig":{"aspectRatio":"1:1","imageSize":"2K"}}}`)

	result, err := svc.ForwardNative(context.Background(), c, account, "gemini-3.1-flash-image", "generateContent", false, body)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr), "empty image response should trigger account failover: %v", err)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Equal(t, "transient", failoverErr.SchedulerCategory)
	require.Equal(t, 0, w.Body.Len(), "empty image response must not be written as a successful response")
}

func TestGeminiMessagesCompatServiceForwardNative_ImageModelCountsInlineDataImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3.1-flash-image:generateContent", nil)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
				"candidates":[{
					"content":{
						"role":"model",
						"parts":[{"inlineData":{"mimeType":"image/png","data":"aGVsbG8="}}]
					},
					"finishReason":"STOP"
				}],
				"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":8}
			}`)),
		},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		ID:       38847,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-key",
		},
	}
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"draw a cat"}]}],"generationConfig":{"responseModalities":["TEXT","IMAGE"],"imageConfig":{"aspectRatio":"1:1","imageSize":"2K"}}}`)

	result, err := svc.ForwardNative(context.Background(), c, account, "gemini-3.1-flash-image", "generateContent", false, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, "2K", result.ImageSize)
	require.Equal(t, "2K", result.ImageInputSize)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"inlineData"`)
}

func TestGeminiMessagesCompatServiceForward_PreservesRequestedModelAndMappedUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"gemini-req-1"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`)),
		},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		ID:   1,
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-key",
			"model_mapping": map[string]any{
				"claude-sonnet-4": "claude-sonnet-4-20250514",
			},
		},
	}
	body := []byte(`{"model":"claude-sonnet-4","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "claude-sonnet-4", result.Model)
	require.Equal(t, "claude-sonnet-4-20250514", result.UpstreamModel)
	require.Equal(t, 1, httpStub.calls)
	require.NotNil(t, httpStub.lastReq)
	require.Contains(t, httpStub.lastReq.URL.String(), "/models/claude-sonnet-4-20250514:")
}

func TestGeminiMessagesCompatServiceForward_NormalizesWebSearchToolForAIStudio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"gemini-req-2"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`)),
		},
	}
	svc := &GeminiMessagesCompatService{httpUpstream: httpStub, cfg: &config.Config{}}
	account := &Account{
		ID:   1,
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-key",
		},
	}
	body := []byte(`{"model":"claude-sonnet-4","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"tools":[{"name":"get_weather","description":"Get weather info","input_schema":{"type":"object"}},{"type":"web_search_20250305","name":"web_search"}]}`)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, httpStub.lastReq)

	postedBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)

	var posted map[string]any
	require.NoError(t, json.Unmarshal(postedBody, &posted))
	tools, ok := posted["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 2)

	searchTool, ok := tools[1].(map[string]any)
	require.True(t, ok)
	_, hasSnake := searchTool["google_search"]
	_, hasCamel := searchTool["googleSearch"]
	require.True(t, hasSnake)
	require.False(t, hasCamel)
	_, hasFuncDecl := searchTool["functionDeclarations"]
	require.False(t, hasFuncDecl)
}

func TestConvertClaudeMessagesToGeminiGenerateContent_AddsThoughtSignatureForToolUse(t *testing.T) {
	claudeReq := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 10,
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "hi"},
				},
			},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "ok"},
					map[string]any{
						"type":  "tool_use",
						"id":    "toolu_123",
						"name":  "default_api:write_file",
						"input": map[string]any{"path": "a.txt", "content": "x"},
						// no signature on purpose
					},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"name":        "default_api:write_file",
				"description": "write file",
				"input_schema": map[string]any{
					"type":       "object",
					"properties": map[string]any{"path": map[string]any{"type": "string"}},
				},
			},
		},
	}
	b, _ := json.Marshal(claudeReq)

	out, err := convertClaudeMessagesToGeminiGenerateContent(b)
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "\"functionCall\"") {
		t.Fatalf("expected functionCall in output, got: %s", s)
	}
	if !strings.Contains(s, "\"thoughtSignature\":\""+geminiDummyThoughtSignature+"\"") {
		t.Fatalf("expected injected thoughtSignature %q, got: %s", geminiDummyThoughtSignature, s)
	}
}

func TestEnsureGeminiFunctionCallThoughtSignatures_InsertsWhenMissing(t *testing.T) {
	geminiReq := map[string]any{
		"contents": []any{
			map[string]any{
				"role": "user",
				"parts": []any{
					map[string]any{
						"functionCall": map[string]any{
							"name": "default_api:write_file",
							"args": map[string]any{"path": "a.txt"},
						},
					},
				},
			},
		},
	}
	b, _ := json.Marshal(geminiReq)
	out := ensureGeminiFunctionCallThoughtSignatures(b)
	s := string(out)
	if !strings.Contains(s, "\"thoughtSignature\":\""+geminiDummyThoughtSignature+"\"") {
		t.Fatalf("expected injected thoughtSignature %q, got: %s", geminiDummyThoughtSignature, s)
	}
}

// TestUnwrapGeminiResponse 测试 unwrapGeminiResponse 的各种输入场景
// 关键区别：只有 response 为 JSON 对象/数组时才解包
func TestUnwrapGeminiResponse(t *testing.T) {
	// 构造 >50KB 的大型 JSON 对象
	largePadding := strings.Repeat("x", 50*1024)
	largeInput := []byte(fmt.Sprintf(`{"response":{"id":"big","pad":"%s"}}`, largePadding))
	largeExpected := fmt.Sprintf(`{"id":"big","pad":"%s"}`, largePadding)

	tests := []struct {
		name     string
		input    []byte
		expected string
		wantErr  bool
	}{
		{
			name:     "正常 response 包装（JSON 对象）",
			input:    []byte(`{"response":{"key":"val"}}`),
			expected: `{"key":"val"}`,
		},
		{
			name:     "无包装直接返回",
			input:    []byte(`{"key":"val"}`),
			expected: `{"key":"val"}`,
		},
		{
			name:     "空 JSON",
			input:    []byte(`{}`),
			expected: `{}`,
		},
		{
			name:     "null response 返回原始 body",
			input:    []byte(`{"response":null}`),
			expected: `{"response":null}`,
		},
		{
			name:     "非法 JSON 返回原始 body",
			input:    []byte(`not json`),
			expected: `not json`,
		},
		{
			name:     "response 为基础类型 string 返回原始 body",
			input:    []byte(`{"response":"hello"}`),
			expected: `{"response":"hello"}`,
		},
		{
			name:     "嵌套 response 只解一层",
			input:    []byte(`{"response":{"response":{"inner":true}}}`),
			expected: `{"response":{"inner":true}}`,
		},
		{
			name:     "大型 JSON >50KB",
			input:    largeInput,
			expected: largeExpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unwrapGeminiResponse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, strings.TrimSpace(string(got)))
		})
	}
}

// ---------------------------------------------------------------------------
// Task 8.1 — extractGeminiUsage 测试
// ---------------------------------------------------------------------------

func TestExtractGeminiUsage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantUsage *ClaudeUsage
	}{
		{
			name:    "完整 usageMetadata",
			input:   `{"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":50,"cachedContentTokenCount":20}}`,
			wantNil: false,
			wantUsage: &ClaudeUsage{
				InputTokens:          80,
				OutputTokens:         50,
				CacheReadInputTokens: 20,
			},
		},
		{
			name:    "包含 thoughtsTokenCount",
			input:   `{"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"thoughtsTokenCount":50}}`,
			wantNil: false,
			wantUsage: &ClaudeUsage{
				InputTokens:          100,
				OutputTokens:         70,
				CacheReadInputTokens: 0,
			},
		},
		{
			name:    "包含 thoughtsTokenCount 与缓存",
			input:   `{"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"cachedContentTokenCount":30,"thoughtsTokenCount":50}}`,
			wantNil: false,
			wantUsage: &ClaudeUsage{
				InputTokens:          70,
				OutputTokens:         70,
				CacheReadInputTokens: 30,
			},
		},
		{
			name:    "缺失 cachedContentTokenCount",
			input:   `{"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":50}}`,
			wantNil: false,
			wantUsage: &ClaudeUsage{
				InputTokens:          100,
				OutputTokens:         50,
				CacheReadInputTokens: 0,
			},
		},
		{
			name:    "无 usageMetadata",
			input:   `{"candidates":[]}`,
			wantNil: true,
		},
		{
			// gjson 对 null 返回 Exists()=true，因此函数不会返回 nil，
			// 而是返回全零的 ClaudeUsage。
			name:    "null usageMetadata — gjson Exists 为 true",
			input:   `{"usageMetadata":null}`,
			wantNil: false,
			wantUsage: &ClaudeUsage{
				InputTokens:          0,
				OutputTokens:         0,
				CacheReadInputTokens: 0,
			},
		},
		{
			name:    "零值字段",
			input:   `{"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":0,"cachedContentTokenCount":0}}`,
			wantNil: false,
			wantUsage: &ClaudeUsage{
				InputTokens:          0,
				OutputTokens:         0,
				CacheReadInputTokens: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGeminiUsage([]byte(tt.input))
			if tt.wantNil {
				if got != nil {
					t.Fatalf("期望返回 nil，实际返回 %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("期望返回非 nil，实际返回 nil")
			}
			if got.InputTokens != tt.wantUsage.InputTokens {
				t.Errorf("InputTokens: 期望 %d，实际 %d", tt.wantUsage.InputTokens, got.InputTokens)
			}
			if got.OutputTokens != tt.wantUsage.OutputTokens {
				t.Errorf("OutputTokens: 期望 %d，实际 %d", tt.wantUsage.OutputTokens, got.OutputTokens)
			}
			if got.CacheReadInputTokens != tt.wantUsage.CacheReadInputTokens {
				t.Errorf("CacheReadInputTokens: 期望 %d，实际 %d", tt.wantUsage.CacheReadInputTokens, got.CacheReadInputTokens)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 8.2 — estimateGeminiCountTokens 测试
// ---------------------------------------------------------------------------

func TestEstimateGeminiCountTokens(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantGt0   bool // 期望结果 > 0
		wantExact *int // 如果非 nil，期望精确匹配
	}{
		{
			name: "含 systemInstruction 和 contents",
			input: `{
				"systemInstruction":{"parts":[{"text":"You are a helpful assistant."}]},
				"contents":[{"parts":[{"text":"Hello, how are you?"}]}]
			}`,
			wantGt0: true,
		},
		{
			name: "仅 contents，无 systemInstruction",
			input: `{
				"contents":[{"parts":[{"text":"Hello, how are you?"}]}]
			}`,
			wantGt0: true,
		},
		{
			name:      "空 parts",
			input:     `{"contents":[{"parts":[]}]}`,
			wantGt0:   false,
			wantExact: intPtr(0),
		},
		{
			name:      "非文本 parts（inlineData）",
			input:     `{"contents":[{"parts":[{"inlineData":{"mimeType":"image/png"}}]}]}`,
			wantGt0:   false,
			wantExact: intPtr(0),
		},
		{
			name:      "空白文本",
			input:     `{"contents":[{"parts":[{"text":"   "}]}]}`,
			wantGt0:   false,
			wantExact: intPtr(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateGeminiCountTokens([]byte(tt.input))
			if tt.wantExact != nil {
				if got != *tt.wantExact {
					t.Errorf("期望精确值 %d，实际 %d", *tt.wantExact, got)
				}
				return
			}
			if tt.wantGt0 && got <= 0 {
				t.Errorf("期望返回 > 0，实际 %d", got)
			}
			if !tt.wantGt0 && got != 0 {
				t.Errorf("期望返回 0，实际 %d", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 8.3 — ParseGeminiRateLimitResetTime 测试
// ---------------------------------------------------------------------------

func TestParseGeminiRateLimitResetTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantNil     bool
		approxDelta int64 // 预期的 (返回值 - now) 大约是多少秒
	}{
		{
			name:        "正常 quotaResetDelay",
			input:       `{"error":{"details":[{"metadata":{"quotaResetDelay":"12.345s"}}]}}`,
			wantNil:     false,
			approxDelta: 13, // 向上取整 12.345 -> 13
		},
		{
			name:        "daily quota",
			input:       `{"error":{"message":"quota per day exceeded"}}`,
			wantNil:     false,
			approxDelta: -1, // 不检查精确 delta，仅检查非 nil
		},
		{
			name:    "无 details 且无 regex 匹配",
			input:   `{"error":{"message":"rate limit"}}`,
			wantNil: true,
		},
		{
			name:        "regex 回退匹配",
			input:       `Please retry in 30s`,
			wantNil:     false,
			approxDelta: 30,
		},
		{
			name:    "完全无匹配",
			input:   `{"error":{"code":429}}`,
			wantNil: true,
		},
		{
			name:        "非法 JSON 但 regex 回退仍工作",
			input:       `not json but Please retry in 10s`,
			wantNil:     false,
			approxDelta: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().Unix()
			got := ParseGeminiRateLimitResetTime([]byte(tt.input))

			if tt.wantNil {
				if got != nil {
					t.Fatalf("期望返回 nil，实际返回 %d", *got)
				}
				return
			}

			if got == nil {
				t.Fatalf("期望返回非 nil，实际返回 nil")
			}

			// approxDelta == -1 表示只检查非 nil，不检查具体值（如 daily quota 场景）
			if tt.approxDelta == -1 {
				// 仅验证返回的时间戳在合理范围内（未来的某个时间）
				if *got < now {
					t.Errorf("期望返回的时间戳 >= now(%d)，实际 %d", now, *got)
				}
				return
			}

			// 使用 +/-2 秒容差进行范围检查
			delta := *got - now
			if delta < tt.approxDelta-2 || delta > tt.approxDelta+2 {
				t.Errorf("期望 delta 约为 %d 秒（+/-2），实际 delta = %d 秒（返回值=%d, now=%d）",
					tt.approxDelta, delta, *got, now)
			}
		})
	}
}

// TestGeminiMessagesHandleStreamingResponse_ClosesToolBlockBeforeText guards the
// tool→text ordering in the Gemini→Anthropic (messages) streaming bridge. When
// Gemini emits a functionCall part followed by a text part, the tool_use content
// block must be closed before the text block opens; otherwise the Anthropic SSE
// stream contains overlapping content blocks. The chat-completions sibling
// already enforces this via closeOpenTool().
func TestGeminiMessagesHandleStreamingResponse_ClosesToolBlockBeforeText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamBody := `data: {"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"city":"SF"}}}]}}]}` + "\n\n" +
		`data: {"candidates":[{"content":{"parts":[{"text":"All done."}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3}}` + "\n\n" +
		"data: [DONE]\n\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GeminiMessagesCompatService{}
	result, err := svc.handleStreamingResponse(c, resp, time.Now(), "claude-3-5-sonnet")
	require.NoError(t, err)
	require.NotNil(t, result)

	events := parseAnthropicContentBlockEvents(t, rec.Body.String())

	// Anthropic allows at most one content block open at a time: every
	// content_block_start must be matched by a content_block_stop before the
	// next start. Replay the lifecycle and assert there is no overlap.
	open := -1
	blockTypes := map[int]string{}
	textStarted := false
	toolClosed := false
	toolClosedBeforeText := false
	for _, ev := range events {
		switch ev.event {
		case "content_block_start":
			require.Equalf(t, -1, open,
				"content block %d opened while block %d was still open (overlapping blocks)", ev.index, open)
			open = ev.index
			blockTypes[ev.index] = ev.blockType
			if ev.blockType == "text" {
				textStarted = true
				if toolClosed {
					toolClosedBeforeText = true
				}
			}
		case "content_block_stop":
			require.Equalf(t, open, ev.index,
				"content_block_stop index %d does not match the open block %d", ev.index, open)
			if blockTypes[ev.index] == "tool_use" {
				toolClosed = true
			}
			open = -1
		}
	}

	require.True(t, textStarted, "expected a text content block to be emitted after the tool call")
	require.True(t, toolClosedBeforeText, "tool_use block must be closed before the text block starts")
	require.Equal(t, -1, open, "stream ended with a content block still open")
}

type anthropicContentBlockEvent struct {
	event     string
	index     int
	blockType string
}

// parseAnthropicContentBlockEvents extracts content_block_start/stop events (with
// their index and, for starts, the content block type) from an Anthropic SSE body.
func parseAnthropicContentBlockEvents(t *testing.T, raw string) []anthropicContentBlockEvent {
	t.Helper()
	var events []anthropicContentBlockEvent
	for _, chunk := range strings.Split(raw, "\n\n") {
		var eventName, dataLine string
		for _, line := range strings.Split(chunk, "\n") {
			switch {
			case strings.HasPrefix(line, "event:"):
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataLine = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
		if eventName != "content_block_start" && eventName != "content_block_stop" {
			continue
		}
		var payload struct {
			Index        int `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
			} `json:"content_block"`
		}
		require.NoError(t, json.Unmarshal([]byte(dataLine), &payload))
		events = append(events, anthropicContentBlockEvent{
			event:     eventName,
			index:     payload.Index,
			blockType: payload.ContentBlock.Type,
		})
	}
	return events
}
