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

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIStreamingPassthroughImageBridgeErrorDoesNotDisableOrFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	repo := &snapshotUpdateAccountRepo{updateExtraCalls: make(chan map[string]any, 1)}
	svc := &OpenAIGatewayService{cfg: cfg, accountRepo: repo}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	account := &Account{
		ID:       38808,
		Platform: PlatformOpenAI,
		Name:     "acc",
		Extra:    map[string]any{featureKeyCodexImageGenerationBridge: true},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.created",
			`data: {"type":"response.created","response":{"id":"resp_1"}}`,
			"",
			"event: response.failed",
			`data: {"type":"response.failed","error":{"message":"Image generation is not enabled for this group"}}`,
			"",
		}, "\n"))),
		Header: http.Header{"X-Request-Id": []string{"rid-passthrough-image-bridge-unsupported"}},
	}

	_, err := svc.handleStreamingResponsePassthrough(c.Request.Context(), resp, c, account, time.Now(), "", "")
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.NotErrorAs(t, err, &failoverErr)
	require.True(t, c.Writer.Written())
	require.Contains(t, rec.Body.String(), "response.failed")
	require.Equal(t, true, account.Extra[featureKeyCodexImageGenerationBridge])

	select {
	case <-repo.updateExtraCalls:
		t.Fatal("passthrough request did not apply the bridge and must not update account configuration")
	default:
	}
}

func TestOpenAIStreamingPassthroughBusinessResponseFailedReturnsTerminal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	svc := &OpenAIGatewayService{cfg: cfg}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.created",
			`data: {"type":"response.created","response":{"id":"resp_1"}}`,
			"",
			"event: response.failed",
			`data: {"type":"response.failed","error":{"message":"Image generation is not enabled for this group"}}`,
			"",
		}, "\n"))),
		Header: http.Header{"X-Request-Id": []string{"rid-passthrough-business-failed"}},
	}

	_, err := svc.handleStreamingResponsePassthrough(c.Request.Context(), resp, c, &Account{ID: 1, Platform: PlatformOpenAI, Name: "acc"}, time.Now(), "", "")
	require.Error(t, err)
	var terminalErr *UpstreamTerminalError
	require.ErrorAs(t, err, &terminalErr)
	require.Equal(t, http.StatusBadRequest, terminalErr.StatusCode)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))
	require.True(t, c.Writer.Written())
	require.Contains(t, rec.Body.String(), "response.failed")
}

func TestOpenAIResponseFailedInsufficientQuotaPayloadFailsOver(t *testing.T) {
	payload := []byte(`{"type":"response.failed","error":{"code":"insufficient_quota"}}`)

	require.True(t, openAIStreamFailedEventShouldFailover(payload, ""))
}

func TestSanitizeOpenAIResponseFailedEventForClientRemovesVerboseResponseFields(t *testing.T) {
	payload := []byte(`{
		"type": "response.failed",
		"response": {
			"id": "resp_1",
			"instructions": "private",
			"output": [{"type":"message","content":[{"text":"secret"}]}],
			"usage": {"input_tokens": 1},
			"metadata": {"user": "secret"},
			"tools": [{"name":"tool"}],
			"error": {"message": "failed"}
		}
	}`)

	sanitized, ok := sanitizeOpenAIResponseFailedEventForClient(payload, "response.failed", false)
	require.True(t, ok)
	require.JSONEq(t, `"failed"`, gjson.GetBytes(sanitized, "response.error.message").Raw)
	require.False(t, gjson.GetBytes(sanitized, "response.instructions").Exists())
	require.False(t, gjson.GetBytes(sanitized, "response.output").Exists())
	require.False(t, gjson.GetBytes(sanitized, "response.usage").Exists())
	require.False(t, gjson.GetBytes(sanitized, "response.metadata").Exists())
	require.False(t, gjson.GetBytes(sanitized, "response.tools").Exists())
}

func TestHandleSSEToJSON_CompactContextWindowFailedReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	body := []byte(strings.Join([]string{
		`data: {"type":"response.failed","error":{"message":"Your input exceeds the context window."}}`,
		`data: [DONE]`,
	}, "\n"))

	usage, err := svc.handleSSEToJSONWithContext(context.Background(), resp, c, nil, body, "gpt-5.5", "gpt-5.5")
	require.Nil(t, usage)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, isOpenAIContextWindowError("", failoverErr.ResponseBody))
	require.Empty(t, rec.Body.String())
}

func TestHandlePassthroughSSEToJSON_CompactContextWindowFailedReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/backend-api/codex/responses/compact", nil)

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	account := &Account{ID: 12, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Name: "openai-oauth"}
	body := []byte(strings.Join([]string{
		`data: {"type":"response.failed","error":{"message":"too many input tokens for the context window"}}`,
		`data: [DONE]`,
	}, "\n"))

	result, err := svc.handlePassthroughSSEToJSONWithContext(context.Background(), resp, c, account, body, "gpt-5.5", "gpt-5.5")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, isOpenAIContextWindowError("", failoverErr.ResponseBody))
	require.Empty(t, rec.Body.String())
}

func TestHandleNonStreamingResponse_CompactJSONResponseFailedReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"type":"response.failed","error":{"message":"maximum context length exceeded"}}`)),
	}
	account := &Account{ID: 13, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Name: "openai-apikey"}

	result, err := svc.handleNonStreamingResponse(context.Background(), resp, c, account, "gpt-5.5", "gpt-5.5")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, isOpenAIContextWindowError("", failoverErr.ResponseBody))
	require.Empty(t, rec.Body.String())
}

func TestHandleNonStreamingResponsePassthrough_CompactJSONResponseFailedReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/backend-api/codex/responses/compact", nil)

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"type":"response.failed","error":{"message":"input is too long for the context window"}}`)),
	}
	account := &Account{ID: 14, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Name: "openai-oauth"}

	result, err := svc.handleNonStreamingResponsePassthroughForAccount(context.Background(), resp, c, account, "gpt-5.5", "gpt-5.5")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, isOpenAIContextWindowError("", failoverErr.ResponseBody))
	require.Empty(t, rec.Body.String())
}
