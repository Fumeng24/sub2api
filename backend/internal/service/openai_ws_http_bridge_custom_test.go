package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestShouldBridgeOpenAIWSHTTPWithContextPreservesOfficialFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = true
	cfg.Gateway.OpenAIWS.HTTPBridgeThresholdBytes = 100
	svc := &OpenAIGatewayService{cfg: cfg}

	require.False(t, svc.shouldBridgeOpenAIWSHTTPWithContext(context.Background(), 99, ""))
	require.True(t, svc.shouldBridgeOpenAIWSHTTPWithContext(context.Background(), 100, ""))

	forced := WithOpenAIWSForceHTTPBridge(context.Background())
	require.True(t, svc.shouldBridgeOpenAIWSHTTPWithContext(forced, 1, ""))
	require.False(t, svc.shouldBridgeOpenAIWSHTTPWithContext(forced, 100, "resp_existing"))

	cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = false
	require.False(t, svc.shouldBridgeOpenAIWSHTTPWithContext(forced, 100, ""))
}
