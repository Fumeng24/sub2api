package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/stretchr/testify/require"
)

func swapAccountMonitorProbeHTTPClient(t *testing.T) {
	t.Helper()
	orig := monitorHTTPClient
	monitorHTTPClient = &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(func() { monitorHTTPClient = orig })
}

type accountMonitorProbeCapture struct {
	lastPath    string
	lastBody    map[string]any
	lastHeaders http.Header
	response    string
}

func (h *accountMonitorProbeCapture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.lastPath = r.URL.Path
	h.lastHeaders = r.Header.Clone()
	defer func() { _ = r.Body.Close() }()
	_ = json.NewDecoder(r.Body).Decode(&h.lastBody)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := h.response
	if response == "" {
		response = `{"ok":true}`
	}
	_, _ = w.Write([]byte(response))
}

func TestAccountMonitorProbeOpenAIUsesChannelMonitorChallenge(t *testing.T) {
	swapAccountMonitorProbeHTTPClient(t)
	h := &accountMonitorProbeCapture{}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	res := runAccountMonitorCheckForModel(context.Background(), MonitorProviderOpenAI, srv.URL, "sk-openai", "gpt-test")

	require.Equal(t, MonitorStatusFailed, res.Status, res.Message)
	require.Equal(t, providerOpenAIResponsesPath, h.lastPath)
	require.Equal(t, "gpt-test", h.lastBody["model"])
	require.Equal(t, float64(monitorChallengeMaxTokens), h.lastBody["max_output_tokens"])
	require.Equal(t, false, h.lastBody["stream"])
	require.Equal(t, ".", h.lastBody["instructions"])
	require.Contains(t, h.lastBody, "input")
	require.NotContains(t, h.lastBody, "messages")
	require.NotContains(t, h.lastBody, "max_tokens")
	require.Equal(t, "Bearer sk-openai", h.lastHeaders.Get("Authorization"))
}

func TestAccountMonitorProbeAnthropicUsesClaudeCodeLivenessBody(t *testing.T) {
	swapAccountMonitorProbeHTTPClient(t)
	h := &accountMonitorProbeCapture{}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	res := runAccountMonitorCheckForModel(context.Background(), MonitorProviderAnthropic, srv.URL, "sk-ant", "claude-test")

	require.Equal(t, MonitorStatusFailed, res.Status, res.Message)
	require.Equal(t, providerAnthropicPath, h.lastPath)
	require.Equal(t, "claude-test", h.lastBody["model"])
	require.Equal(t, float64(16), h.lastBody["max_tokens"])
	require.Equal(t, false, h.lastBody["stream"])
	require.Contains(t, h.lastHeaders.Get("User-Agent"), "claude-cli/")
	require.Equal(t, "cli", h.lastHeaders.Get("X-App"))
	require.Equal(t, "application/json", h.lastHeaders.Get("Accept"))
	require.Equal(t, "sk-ant", h.lastHeaders.Get("x-api-key"))
	require.Equal(t, monitorAnthropicAPIVersion, h.lastHeaders.Get("anthropic-version"))
	require.Contains(t, h.lastHeaders.Get("anthropic-beta"), claude.BetaClaudeCode)
	require.Contains(t, h.lastHeaders.Get("anthropic-beta"), claude.BetaOAuth)

	system, ok := h.lastBody["system"].([]any)
	require.True(t, ok)
	require.Len(t, system, 1)
	systemBlock, ok := system[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, claudeCodeSystemPrompt, systemBlock["text"])

	metadata, ok := h.lastBody["metadata"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, metadata["user_id"])

	messages, ok := h.lastBody["messages"].([]any)
	require.True(t, ok)
	require.Len(t, messages, 1)
	first, ok := messages[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user", first["role"])
	content, ok := first["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 1)
	textBlock, ok := content[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "text", textBlock["type"])
	promptText, ok := textBlock["text"].(string)
	require.True(t, ok)
	require.Contains(t, accountMonitorAnthropicSafeGreetingPromptsForTest(), promptText)
	require.NotContains(t, promptText, "%!")
}

func TestAccountMonitorProbeAnthropicGreetingChallengeAcceptsHelpText(t *testing.T) {
	swapAccountMonitorProbeHTTPClient(t)
	h := &accountMonitorProbeCapture{
		response: `{"type":"message","content":[{"type":"text","text":"Hi! How can I help you today?"}]}`,
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	res := runAccountMonitorCheckForModel(context.Background(), MonitorProviderAnthropic, srv.URL, "sk-ant", "claude-test")

	require.Equal(t, MonitorStatusOperational, res.Status, res.Message)
}

func TestAccountMonitorProbeAnthropicGreetingChallengeAcceptsThanksText(t *testing.T) {
	require.True(t, isAccountMonitorAnthropicGreetingResponse("You're welcome.", []string{"welcome", "help", "assist", "glad", "sure"}))
	require.False(t, isAccountMonitorAnthropicGreetingResponse("unrelated", []string{"welcome", "help", "assist", "glad", "sure"}))
}

func TestAccountMonitorProbeAnthropicGreetingChallengeVariesPrompt(t *testing.T) {
	seen := make(map[string]struct{})
	for range 20 {
		challenge := accountMonitorAnthropicGreetingChallenge()
		require.NotEmpty(t, challenge.AcceptAny)
		require.Contains(t, accountMonitorAnthropicSafeGreetingPromptsForTest(), challenge.Prompt)
		require.NotContains(t, challenge.Prompt, "%!")
		seen[challenge.Prompt] = struct{}{}
	}
	require.GreaterOrEqual(t, len(seen), 2)
}

func accountMonitorAnthropicSafeGreetingPromptsForTest() []string {
	return []string{
		"Hi",
		"Hi.",
		"Hi!",
		"Hi there",
		"Hello",
		"Hello.",
		"Hello!",
		"Hello there",
		"Hey",
		"Hey!",
		"Hey there",
		"Yo",
		"hi",
		"hello",
		"Good morning",
		"Morning",
		"Good day",
		"Thanks",
		"Thanks!",
		"Ping",
	}
}
