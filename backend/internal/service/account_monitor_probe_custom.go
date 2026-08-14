package service

import (
	"context"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

// runAccountMonitorCheckForModel is the scheduler account monitor probe.
// OpenAI/Codex account monitors call Responses directly so an upstream gateway
// does not have to run its own Chat Completions -> Responses compatibility layer
// for a tiny health probe.
func runAccountMonitorCheckForModel(ctx context.Context, provider, endpoint, apiKey, model string) *CheckResult {
	if provider == MonitorProviderOpenAI {
		return runCheckForModel(ctx, provider, endpoint, apiKey, model, &CheckOptions{
			APIMode: MonitorAPIModeResponses,
		})
	}
	if provider == MonitorProviderAnthropic {
		return runAnthropicAccountMonitorGreetingChallenge(ctx, endpoint, apiKey, model)
	}
	return runCheckForModel(ctx, provider, endpoint, apiKey, model, nil)
}

func runAnthropicAccountMonitorGreetingChallenge(ctx context.Context, endpoint, apiKey, model string) *CheckResult {
	res := &CheckResult{
		Model:     model,
		Status:    MonitorStatusError,
		CheckedAt: time.Now(),
	}

	challenge := accountMonitorAnthropicGreetingChallenge()
	opts := anthropicAccountMonitorClaudeCodeProbeOptions(model, challenge.Prompt)

	start := time.Now()
	respText, rawBody, statusCode, err := callProvider(ctx, MonitorProviderAnthropic, endpoint, apiKey, model, challenge.Prompt, opts)
	latency := time.Since(start)
	latencyMs := int(latency / time.Millisecond)
	res.LatencyMs = &latencyMs

	if err != nil {
		res.Message = truncateMessage(sanitizeErrorMessage(err.Error()))
		return res
	}
	if statusCode < 200 || statusCode >= 300 {
		bodySnippet := truncateForErrorBody(rawBody)
		res.Message = truncateMessage(sanitizeErrorMessage("upstream HTTP " + strconv.Itoa(statusCode) + ": " + bodySnippet))
		return res
	}
	if !isAccountMonitorAnthropicGreetingResponse(respText, challenge.AcceptAny) {
		res.Status = MonitorStatusFailed
		res.Message = truncateMessage(sanitizeErrorMessage("greeting challenge mismatch: " + respText))
		return res
	}
	return finalizeOperationalOrDegraded(res, latency, latencyMs)
}

type accountMonitorAnthropicChallenge struct {
	Prompt    string
	AcceptAny []string
}

func accountMonitorAnthropicGreetingChallenge() accountMonitorAnthropicChallenge {
	acceptGreeting := []string{"hi", "hello", "hey", "help", "assist", "morning", "welcome"}
	type greetingVariant struct {
		Text      string
		AcceptAny []string
	}
	variants := []greetingVariant{
		{Text: "Hi", AcceptAny: acceptGreeting},
		{Text: "Hi.", AcceptAny: acceptGreeting},
		{Text: "Hi!", AcceptAny: acceptGreeting},
		{Text: "Hi there", AcceptAny: acceptGreeting},
		{Text: "Hello", AcceptAny: acceptGreeting},
		{Text: "Hello.", AcceptAny: acceptGreeting},
		{Text: "Hello!", AcceptAny: acceptGreeting},
		{Text: "Hello there", AcceptAny: acceptGreeting},
		{Text: "Hey", AcceptAny: acceptGreeting},
		{Text: "Hey!", AcceptAny: acceptGreeting},
		{Text: "Hey there", AcceptAny: acceptGreeting},
		{Text: "Yo", AcceptAny: acceptGreeting},
		{Text: "hi", AcceptAny: acceptGreeting},
		{Text: "hello", AcceptAny: acceptGreeting},
		{Text: "Good morning", AcceptAny: acceptGreeting},
		{Text: "Morning", AcceptAny: acceptGreeting},
		{Text: "Good day", AcceptAny: acceptGreeting},
		{Text: "Thanks", AcceptAny: []string{"welcome", "help", "assist", "glad", "sure"}},
		{Text: "Thanks!", AcceptAny: []string{"welcome", "help", "assist", "glad", "sure"}},
		{Text: "Ping", AcceptAny: append(acceptGreeting, "pong")},
	}
	variant := variants[rand.IntN(len(variants))] //nolint:gosec // Health-check prompt variance, not security-sensitive.
	return accountMonitorAnthropicChallenge{
		Prompt:    variant.Text,
		AcceptAny: variant.AcceptAny,
	}
}

func isAccountMonitorAnthropicGreetingResponse(text string, acceptAny []string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	for _, token := range acceptAny {
		token = strings.ToLower(strings.TrimSpace(token))
		if token != "" && strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func anthropicAccountMonitorClaudeCodeProbeOptions(model, prompt string) *CheckOptions {
	sessionID, err := generateSessionString()
	if err != nil {
		sessionID = FormatMetadataUserID(
			strings.Repeat("0", 64),
			"",
			"00000000-0000-0000-0000-000000000000",
			ExtractCLIVersion(claude.DefaultHeaders["User-Agent"]),
		)
	}

	headers := make(map[string]string, len(claude.DefaultHeaders)+3)
	for key, value := range claude.DefaultHeaders {
		headers[key] = value
	}
	headers["Accept"] = "application/json"
	headers["anthropic-version"] = monitorAnthropicAPIVersion
	headers["anthropic-beta"] = strings.Join(claude.FullClaudeCodeMimicryBetas(), ",")

	return &CheckOptions{
		BodyOverrideMode: MonitorBodyOverrideModeReplace,
		BodyOverride: map[string]any{
			"model": model,
			"messages": []map[string]any{
				{
					"role": "user",
					"content": []map[string]any{
						{
							"type": "text",
							"text": prompt,
						},
					},
				},
			},
			"system": []map[string]any{
				{
					"type": "text",
					"text": claudeCodeSystemPrompt,
				},
			},
			"metadata": map[string]any{
				"user_id": sessionID,
			},
			"max_tokens": 16,
			"stream":     false,
		},
		ExtraHeaders: headers,
	}
}
