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
	"github.com/tidwall/gjson"
)

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

func TestGeminiForwardAsChatCompletions_KeepsNamedWebSearchAsClientFunction(t *testing.T) {
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
	require.False(t, gjson.GetBytes(postedBody, `toolConfig.includeServerSideToolInvocations`).Exists(), string(postedBody))
	require.False(t, gjson.GetBytes(postedBody, `tools.0.google_search`).Exists(), string(postedBody))
	require.True(t, gjson.GetBytes(postedBody, `tools.0.functionDeclarations`).Exists(), string(postedBody))
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
