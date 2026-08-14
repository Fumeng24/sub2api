package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type modelCapabilityHTTPStub struct {
	status      int
	body        string
	err         error
	responseFn  func(*http.Request, []byte) (int, string)
	lastRequest *http.Request
	lastBody    []byte
}

func (s *modelCapabilityHTTPStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return s.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (s *modelCapabilityHTTPStub) DoWithTLS(
	req *http.Request,
	_ string,
	_ int64,
	_ int,
	_ *tlsfingerprint.Profile,
) (*http.Response, error) {
	s.lastRequest = req
	s.lastBody, _ = io.ReadAll(req.Body)
	if s.err != nil {
		return nil, s.err
	}
	status := s.status
	body := s.body
	if s.responseFn != nil {
		status, body = s.responseFn(req, s.lastBody)
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

var modelCapabilityTestChallengePattern = regexp.MustCompile(`(\d+)\s*([+-])\s*(\d+)`)

func modelCapabilityTestSuccessBody(platform string, requestBody []byte) string {
	promptPath := "input"
	switch platform {
	case PlatformAnthropic:
		promptPath = "messages.0.content.0.text"
	case PlatformGemini:
		promptPath = "contents.0.parts.0.text"
	case PlatformGrok:
		promptPath = "messages.0.content"
	}
	match := modelCapabilityTestChallengePattern.FindStringSubmatch(gjson.GetBytes(requestBody, promptPath).String())
	if len(match) != 4 {
		return `{"error":{"message":"missing arithmetic challenge"}}`
	}
	left, _ := strconv.Atoi(match[1])
	right, _ := strconv.Atoi(match[3])
	answer := left + right
	if match[2] == "-" {
		answer = left - right
	}
	quoted := strconv.Quote(strconv.Itoa(answer))
	switch platform {
	case PlatformAnthropic:
		return `{"id":"msg-1","type":"message","content":[{"type":"text","text":` + quoted + `}]}`
	case PlatformGemini:
		return `{"candidates":[{"content":{"parts":[{"text":` + quoted + `}]},"finishReason":"STOP"}]}`
	case PlatformGrok:
		return `{"id":"chatcmpl-1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":` + quoted + `},"finish_reason":"stop"}]}`
	default:
		return `{"id":"resp-1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":` + quoted + `}]}]}`
	}
}

func newModelCapabilityProbeService(upstream HTTPUpstream) *AccountTestService {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	return NewAccountTestService(nil, nil, nil, nil, nil, upstream, cfg, nil)
}

func newModelCapabilityAccount(platform string) *Account {
	return &Account{
		ID:       -7,
		Platform: platform,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-model-capability-test",
			"base_url": "https://upstream.example",
		},
		Extra:       map[string]any{},
		Concurrency: 1,
	}
}

func TestProbeAPIKeyModelUsesProtocolSpecificRealRequests(t *testing.T) {
	tests := []struct {
		name             string
		platform         string
		model            string
		wantPath         string
		wantAuthHeader   string
		wantAuthValue    string
		wantBodyMaxField string
	}{
		{
			name: "openai responses", platform: PlatformOpenAI, model: "gpt-test",
			wantPath: "/v1/responses", wantAuthHeader: "Authorization", wantAuthValue: "Bearer sk-model-capability-test",
			wantBodyMaxField: "max_output_tokens",
		},
		{
			name: "anthropic messages", platform: PlatformAnthropic, model: "claude-test",
			wantPath: "/v1/messages", wantAuthHeader: "x-api-key", wantAuthValue: "sk-model-capability-test",
			wantBodyMaxField: "max_tokens",
		},
		{
			name: "gemini generate content", platform: PlatformGemini, model: "gemini-test",
			wantPath: "/v1beta/models/gemini-test:generateContent", wantAuthHeader: "x-goog-api-key", wantAuthValue: "sk-model-capability-test",
			wantBodyMaxField: "generationConfig.maxOutputTokens",
		},
		{
			name: "grok chat completions", platform: PlatformGrok, model: "grok-test",
			wantPath: "/v1/chat/completions", wantAuthHeader: "Authorization", wantAuthValue: "Bearer sk-model-capability-test",
			wantBodyMaxField: "max_tokens",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstream := &modelCapabilityHTTPStub{responseFn: func(req *http.Request, body []byte) (int, string) {
				return http.StatusOK, modelCapabilityTestSuccessBody(test.platform, body)
			}}
			result := newModelCapabilityProbeService(upstream).ProbeAPIKeyModel(
				t.Context(), newModelCapabilityAccount(test.platform), test.model,
			)

			require.Equal(t, ModelCapabilityStatusOK, result.Status, result.Message)
			require.Equal(t, http.StatusOK, result.StatusCode)
			require.Equal(t, test.wantPath, upstream.lastRequest.URL.Path)
			require.Equal(t, test.wantAuthValue, upstream.lastRequest.Header.Get(test.wantAuthHeader))
			require.Equal(t, float64(modelCapabilityProbeMaxTokens), gjson.GetBytes(upstream.lastBody, test.wantBodyMaxField).Float())
			require.False(t, gjson.GetBytes(upstream.lastBody, "stream").Bool())
		})
	}
}

func TestProbeAPIKeyModelAcceptsOpenAICompatibleTextContentBlock(t *testing.T) {
	upstream := &modelCapabilityHTTPStub{responseFn: func(_ *http.Request, body []byte) (int, string) {
		return http.StatusOK, strings.ReplaceAll(
			modelCapabilityTestSuccessBody(PlatformOpenAI, body),
			`"output_text"`,
			`"text"`,
		)
	}}

	result := newModelCapabilityProbeService(upstream).ProbeAPIKeyModel(
		t.Context(), newModelCapabilityAccount(PlatformOpenAI), "gpt-test",
	)

	require.Equal(t, ModelCapabilityStatusOK, result.Status, result.Message)
}

func TestProbeAPIKeyModelUsesGrokChatCompletionsWhenResponsesIsUnavailable(t *testing.T) {
	requestCount := 0
	upstream := &modelCapabilityHTTPStub{responseFn: func(req *http.Request, body []byte) (int, string) {
		requestCount++
		if req.URL.Path == "/v1/responses" {
			return http.StatusGatewayTimeout, `{"error":{"message":"responses endpoint timed out"}}`
		}
		return http.StatusOK, modelCapabilityTestSuccessBody(PlatformGrok, body)
	}}

	result := newModelCapabilityProbeService(upstream).ProbeAPIKeyModel(
		t.Context(), newModelCapabilityAccount(PlatformGrok), "grok-4.5",
	)

	require.Equal(t, ModelCapabilityStatusOK, result.Status, result.Message)
	require.Equal(t, 1, requestCount)
	require.Equal(t, "/v1/chat/completions", upstream.lastRequest.URL.Path)
	require.Equal(t, "user", gjson.GetBytes(upstream.lastBody, "messages.0.role").String())
	require.NotEmpty(t, gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
}

func TestProbeAPIKeyModelClassifiesOnlyExplicitModelErrorsAsUnsupported(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{name: "explicit unsupported", statusCode: http.StatusBadRequest, body: `{"error":{"message":"The requested model is not supported"}}`, want: ModelCapabilityStatusUnsupported},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, body: `{"error":{"message":"rate limit exceeded"}}`, want: ModelCapabilityStatusUnconfirmed},
		{name: "upstream unavailable", statusCode: http.StatusServiceUnavailable, body: `{"error":{"message":"Current group is unavailable"}}`, want: ModelCapabilityStatusUnconfirmed},
		{name: "5xx model message", statusCode: http.StatusServiceUnavailable, body: `{"error":{"message":"The requested model is not supported"}}`, want: ModelCapabilityStatusUnconfirmed},
		{name: "insufficient balance", statusCode: http.StatusForbidden, body: `{"error":{"message":"insufficient balance for model request"}}`, want: ModelCapabilityStatusUnconfirmed},
		{name: "authentication", statusCode: http.StatusUnauthorized, body: `{"error":{"message":"invalid api key"}}`, want: ModelCapabilityStatusUnconfirmed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstream := &modelCapabilityHTTPStub{status: test.statusCode, body: test.body}
			result := newModelCapabilityProbeService(upstream).ProbeAPIKeyModel(
				t.Context(), newModelCapabilityAccount(PlatformOpenAI), "gpt-test",
			)
			require.Equal(t, test.want, result.Status)
			require.Equal(t, test.statusCode, result.StatusCode)
			if test.statusCode == http.StatusTooManyRequests {
				require.Contains(t, result.Message, "temporarily rate-limited")
				require.Contains(t, result.Message, "support remains unconfirmed")
			}
		})
	}
}

func TestModelCapabilityRetryAfter(t *testing.T) {
	now := time.Date(2026, time.August, 4, 4, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "seconds", raw: "2", want: 2 * time.Second},
		{name: "http date", raw: now.Add(3 * time.Second).Format(http.TimeFormat), want: 3 * time.Second},
		{name: "expired", raw: now.Add(-time.Second).Format(http.TimeFormat)},
		{name: "invalid", raw: "later"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := http.Header{"Retry-After": []string{test.raw}}
			require.Equal(t, test.want, modelCapabilityRetryAfter(header, now))
		})
	}
}

func TestProbeAPIKeyModelAcceptsAnySuccessfulHTTPResponseAndReportsTransportFailure(t *testing.T) {
	incomplete := &modelCapabilityHTTPStub{status: http.StatusOK, body: `{"id":"resp-empty","status":"completed","output":[]}`}
	result := newModelCapabilityProbeService(incomplete).ProbeAPIKeyModel(
		t.Context(), newModelCapabilityAccount(PlatformOpenAI), "gpt-test",
	)
	require.Equal(t, ModelCapabilityStatusOK, result.Status)
	require.Equal(t, http.StatusOK, result.StatusCode)

	transportFailure := &modelCapabilityHTTPStub{err: errors.New("dial timeout")}
	result = newModelCapabilityProbeService(transportFailure).ProbeAPIKeyModel(
		context.Background(), newModelCapabilityAccount(PlatformOpenAI), "gpt-test",
	)
	require.Equal(t, ModelCapabilityStatusUnconfirmed, result.Status)
	require.Contains(t, result.Message, "dial timeout")
}

func TestProbeAPIKeyModelUsesActualImageEndpoint(t *testing.T) {
	upstream := &modelCapabilityHTTPStub{
		status: http.StatusOK,
		body:   `{"created":1,"data":[{"url":"https://images.example/result.png"}]}`,
	}
	result := newModelCapabilityProbeService(upstream).ProbeAPIKeyModel(
		t.Context(), newModelCapabilityAccount(PlatformOpenAI), "gpt-image-2",
	)

	require.Equal(t, ModelCapabilityStatusOK, result.Status, result.Message)
	require.Equal(t, "/v1/images/generations", upstream.lastRequest.URL.Path)
	require.Equal(t, "gpt-image-2", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, float64(1), gjson.GetBytes(upstream.lastBody, "n").Float())
	require.False(t, gjson.GetBytes(upstream.lastBody, "response_format").Exists())
}

func TestProbeAPIKeyModelNormalizesGeminiVersionedBaseURL(t *testing.T) {
	upstream := &modelCapabilityHTTPStub{responseFn: func(req *http.Request, body []byte) (int, string) {
		return http.StatusOK, modelCapabilityTestSuccessBody(PlatformGemini, body)
	}}
	account := newModelCapabilityAccount(PlatformGemini)
	account.Credentials["base_url"] = "https://upstream.example/v1beta"

	result := newModelCapabilityProbeService(upstream).ProbeAPIKeyModel(t.Context(), account, "gemini-test")

	require.Equal(t, ModelCapabilityStatusOK, result.Status, result.Message)
	require.Equal(t, "/v1beta/models/gemini-test:generateContent", upstream.lastRequest.URL.Path)
}
