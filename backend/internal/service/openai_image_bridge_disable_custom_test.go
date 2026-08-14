package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIStreamingResponseFailedImageBridgeUnsupportedAutoDisablesAndFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	repo := &snapshotUpdateAccountRepo{updateExtraCalls: make(chan map[string]any, 1)}
	svc := &OpenAIGatewayService{cfg: cfg, accountRepo: repo}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	account := &Account{ID: 38808, Platform: PlatformOpenAI, Name: "acc", Extra: map[string]any{featureKeyCodexImageGenerationBridge: true}}
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
		Header: http.Header{"X-Request-Id": []string{"rid-image-bridge-unsupported"}},
	}

	bridgeCtx := withOpenAICodexImageBridgeApplied(c.Request.Context())
	_, err := svc.handleStreamingResponse(bridgeCtx, resp, c, account, time.Now(), "model", "model")
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
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

func TestAutoDisableCodexImageBridgeRequiresAppliedMarker(t *testing.T) {
	updates := make(chan map[string]any, 1)
	repo := &snapshotUpdateAccountRepo{updateExtraCalls: updates}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 91, Name: "bridge-account", Platform: PlatformOpenAI, Extra: map[string]any{}}

	require.False(t, svc.autoDisableCodexImageBridgeForUnsupportedUpstream(
		context.Background(), account, imageGenerationPermissionMessage, nil,
	))
	select {
	case <-updates:
		t.Fatal("request without an applied bridge must not update account configuration")
	default:
	}
}

func TestAutoDisableCodexImageBridgeAfterAppliedMarker(t *testing.T) {
	updates := make(chan map[string]any, 1)
	repo := &snapshotUpdateAccountRepo{updateExtraCalls: updates}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 92, Name: "bridge-account", Platform: PlatformOpenAI, Extra: map[string]any{}}
	ctx := withOpenAICodexImageBridgeApplied(context.Background())

	require.True(t, svc.autoDisableCodexImageBridgeForUnsupportedUpstream(
		ctx, account, imageGenerationPermissionMessage, nil,
	))
	require.Equal(t, false, (<-updates)[featureKeyCodexImageGenerationBridge])
	require.Equal(t, false, account.Extra[featureKeyCodexImageGenerationBridge])
}

func TestAutoDisableCodexImageBridgeIgnoresDisabledOverride(t *testing.T) {
	updates := make(chan map[string]any, 1)
	repo := &snapshotUpdateAccountRepo{updateExtraCalls: updates}
	svc := &OpenAIGatewayService{accountRepo: repo}
	account := &Account{ID: 93, Name: "bridge-account", Platform: PlatformOpenAI, Extra: map[string]any{
		featureKeyCodexImageGenerationBridge: false,
	}}

	require.False(t, svc.autoDisableCodexImageBridgeForUnsupportedUpstream(
		withOpenAICodexImageBridgeApplied(context.Background()), account, imageGenerationPermissionMessage, nil,
	))
	select {
	case <-updates:
		t.Fatal("disabled override must not be written again")
	default:
	}
}
