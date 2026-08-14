package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/tidwall/gjson"
)

const (
	ModelCapabilityStatusOK          = "ok"
	ModelCapabilityStatusUnsupported = "unsupported"
	ModelCapabilityStatusUnconfirmed = "unconfirmed"

	// 200 tokens leaves sufficient room for reasoning and compatibility relays
	// to return the arithmetic challenge result without making each probe costly.
	modelCapabilityProbeMaxTokens       = 200
	modelCapabilityProbeMaxResponseSize = 16 << 20
	modelCapabilityImagePrompt          = "A single black dot on a white background."
)

// ModelCapabilityProbeResult is a read-only model invocation result. It never
// changes account status, scheduler state, cooldowns, or billing snapshots.
type ModelCapabilityProbeResult struct {
	Status     string
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

type modelCapabilityProbeRequest struct {
	request  *http.Request
	validate func([]byte) bool
}

// ProbeAPIKeyModel invokes one model through the same outbound transport,
// proxy, TLS profile, endpoint, and authentication settings as an API-key
// account. A catalogue entry alone is intentionally not treated as support.
func (s *AccountTestService) ProbeAPIKeyModel(ctx context.Context, account *Account, model string) ModelCapabilityProbeResult {
	if s == nil || s.httpUpstream == nil {
		return unconfirmedModelCapability("Model capability probe is unavailable")
	}
	if account == nil || account.Type != AccountTypeAPIKey {
		return unconfirmedModelCapability("An API-key account is required for model verification")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return unconfirmedModelCapability("Model is required")
	}

	probe, err := s.buildModelCapabilityProbeRequest(ctx, account, model)
	if err != nil {
		return unconfirmedModelCapability(err.Error())
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var tlsProfile *tlsfingerprint.Profile
	if s.tlsFPProfileService != nil {
		tlsProfile = s.tlsFPProfileService.ResolveTLSProfile(account)
	}
	resp, err := s.httpUpstream.DoWithTLS(probe.request, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return unconfirmedModelCapability("Upstream request failed: " + sanitizeErrorMessage(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	// A capability probe only needs to know whether the real model request was
	// accepted. Do not parse a successful body: compatible gateways may return
	// different but valid response shapes, and a body parser must not turn HTTP
	// 200 into a false negative.
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return ModelCapabilityProbeResult{
			Status:     ModelCapabilityStatusOK,
			StatusCode: resp.StatusCode,
			Message:    "Model request completed successfully",
		}
	}

	body, tooLarge, readErr := readModelCapabilityProbeBody(resp.Body)
	if readErr != nil {
		return ModelCapabilityProbeResult{
			Status:     ModelCapabilityStatusUnconfirmed,
			StatusCode: resp.StatusCode,
			Message:    "Failed to read the complete upstream response",
		}
	}
	if tooLarge {
		return ModelCapabilityProbeResult{
			Status:     ModelCapabilityStatusUnconfirmed,
			StatusCode: resp.StatusCode,
			Message:    "Upstream response exceeded the verification size limit",
		}
	}
	status := ModelCapabilityStatusUnconfirmed
	if isDefinitiveModelCapabilityUnsupported(resp.StatusCode, body) {
		status = ModelCapabilityStatusUnsupported
	}
	return ModelCapabilityProbeResult{
		Status:     status,
		StatusCode: resp.StatusCode,
		Message:    modelCapabilityHTTPError(resp.StatusCode, body),
		RetryAfter: modelCapabilityRetryAfter(resp.Header, time.Now()),
	}
}

func (s *AccountTestService) buildModelCapabilityProbeRequest(
	ctx context.Context,
	account *Account,
	model string,
) (modelCapabilityProbeRequest, error) {
	switch account.Platform {
	case PlatformOpenAI:
		return s.buildOpenAIModelCapabilityProbe(ctx, account, model)
	case PlatformAnthropic:
		return s.buildAnthropicModelCapabilityProbe(ctx, account, model)
	case PlatformGemini:
		return s.buildGeminiModelCapabilityProbe(ctx, account, model)
	case PlatformGrok:
		return s.buildGrokModelCapabilityProbe(ctx, account, model)
	default:
		return modelCapabilityProbeRequest{}, fmt.Errorf("unsupported model protocol %q", account.Platform)
	}
}

func (s *AccountTestService) buildOpenAIModelCapabilityProbe(
	ctx context.Context,
	account *Account,
	model string,
) (modelCapabilityProbeRequest, error) {
	apiKey := strings.TrimSpace(account.GetOpenAIApiKey())
	if apiKey == "" {
		return modelCapabilityProbeRequest{}, errors.New("OpenAI API key is missing")
	}
	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	baseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return modelCapabilityProbeRequest{}, fmt.Errorf("invalid OpenAI base URL: %w", err)
	}

	var target string
	var payload map[string]any
	var validate func([]byte) bool
	if isOpenAIImageModel(model) {
		target = buildOpenAIImagesURL(baseURL, openAIImagesGenerationsEndpoint)
		payload = map[string]any{
			"model":  model,
			"prompt": modelCapabilityImagePrompt,
			"n":      1,
		}
		validate = validOpenAIImageCapabilityResponse
	} else {
		challenge := generateChallenge()
		target = buildOpenAIResponsesURL(baseURL)
		payload = map[string]any{
			"model":             model,
			"instructions":      ".",
			"input":             challenge.Prompt,
			"max_output_tokens": modelCapabilityProbeMaxTokens,
			"stream":            false,
		}
		validate = func(body []byte) bool {
			return validOpenAITextCapabilityResponse(body, challenge.Expected)
		}
	}
	return buildJSONModelCapabilityRequest(ctx, account, target, payload, validate, func(header http.Header) {
		header.Set("Authorization", "Bearer "+apiKey)
		applyOpenAICodexProbeHeaders(header)
	})
}

func (s *AccountTestService) buildAnthropicModelCapabilityProbe(
	ctx context.Context,
	account *Account,
	model string,
) (modelCapabilityProbeRequest, error) {
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return modelCapabilityProbeRequest{}, errors.New("Anthropic API key is missing")
	}
	baseURL := account.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	baseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return modelCapabilityProbeRequest{}, fmt.Errorf("invalid Anthropic base URL: %w", err)
	}
	target := buildOpenAIEndpointURL(baseURL, "/v1/messages") + "?beta=true"

	challenge := generateChallenge()
	options := anthropicAccountMonitorClaudeCodeProbeOptions(model, challenge.Prompt)
	payload := options.BodyOverride
	payload["max_tokens"] = modelCapabilityProbeMaxTokens
	payload["stream"] = false
	return buildJSONModelCapabilityRequest(ctx, account, target, payload, func(body []byte) bool {
		return validAnthropicCapabilityResponse(body, challenge.Expected)
	}, func(header http.Header) {
		for key, value := range options.ExtraHeaders {
			header.Set(key, value)
		}
		header.Set("anthropic-version", monitorAnthropicAPIVersion)
		header.Set("anthropic-beta", strings.Join(claude.FullClaudeCodeMimicryBetas(), ","))
		setAnthropicAPIKeyAuthHeader(header, account, apiKey)
	})
}

func (s *AccountTestService) buildGeminiModelCapabilityProbe(
	ctx context.Context,
	account *Account,
	model string,
) (modelCapabilityProbeRequest, error) {
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return modelCapabilityProbeRequest{}, errors.New("Gemini API key is missing")
	}
	baseURL := account.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return modelCapabilityProbeRequest{}, fmt.Errorf("invalid Gemini base URL: %w", err)
	}
	modelPath := url.PathEscape(strings.TrimPrefix(strings.TrimSpace(model), "models/"))
	target := strings.TrimRight(buildGeminiModelsURL(baseURL), "/") + "/" + modelPath + ":generateContent"

	var payload map[string]any
	var validate func([]byte) bool
	if isImageGenerationModel(model) {
		payload = map[string]any{
			"contents": []map[string]any{{
				"role":  "user",
				"parts": []map[string]any{{"text": modelCapabilityImagePrompt}},
			}},
			"generationConfig": map[string]any{
				"responseModalities": []string{"TEXT", "IMAGE"},
				"imageConfig":        map[string]any{"aspectRatio": "1:1"},
			},
		}
		validate = validGeminiImageCapabilityResponse
	} else {
		challenge := generateChallenge()
		payload = map[string]any{
			"contents": []map[string]any{{
				"role":  "user",
				"parts": []map[string]any{{"text": challenge.Prompt}},
			}},
			"generationConfig": map[string]any{"maxOutputTokens": modelCapabilityProbeMaxTokens},
		}
		validate = func(body []byte) bool {
			return validGeminiTextCapabilityResponse(body, challenge.Expected)
		}
	}
	return buildJSONModelCapabilityRequest(ctx, account, target, payload, validate, func(header http.Header) {
		header.Set("x-goog-api-key", apiKey)
	})
}

func (s *AccountTestService) buildGrokModelCapabilityProbe(
	ctx context.Context,
	account *Account,
	model string,
) (modelCapabilityProbeRequest, error) {
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return modelCapabilityProbeRequest{}, errors.New("Grok API key is missing")
	}

	var target string
	var payload map[string]any
	var validate func([]byte) bool
	var err error
	if isGrokImageGenerationModel(model) {
		target, err = buildGrokMediaURL(account, s.cfg, GrokMediaEndpointImagesGenerations, "")
		payload = map[string]any{
			"model":  model,
			"prompt": modelCapabilityImagePrompt,
			"n":      1,
		}
		validate = validOpenAIImageCapabilityResponse
	} else {
		challenge := generateChallenge()
		target, err = buildGrokChatCompletionsURL(account, s.cfg)
		payload = map[string]any{
			"model": model,
			"messages": []map[string]any{{
				"role":    "user",
				"content": challenge.Prompt,
			}},
			"max_tokens": modelCapabilityProbeMaxTokens,
			"stream":     false,
		}
		validate = func(body []byte) bool {
			return validOpenAIChatCapabilityResponse(body, challenge.Expected)
		}
	}
	if err != nil {
		return modelCapabilityProbeRequest{}, fmt.Errorf("invalid Grok base URL: %w", err)
	}
	return buildJSONModelCapabilityRequest(ctx, account, target, payload, validate, func(header http.Header) {
		header.Set("Authorization", "Bearer "+apiKey)
	})
}

func buildJSONModelCapabilityRequest(
	ctx context.Context,
	account *Account,
	target string,
	payload map[string]any,
	validate func([]byte) bool,
	configureHeaders func(http.Header),
) (modelCapabilityProbeRequest, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return modelCapabilityProbeRequest{}, fmt.Errorf("build model probe payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return modelCapabilityProbeRequest{}, fmt.Errorf("build model probe request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if configureHeaders != nil {
		configureHeaders(req.Header)
	}
	account.ApplyHeaderOverrides(req.Header)
	return modelCapabilityProbeRequest{request: req, validate: validate}, nil
}

func readModelCapabilityProbeBody(body io.Reader) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(body, modelCapabilityProbeMaxResponseSize+1))
	if err != nil {
		return nil, false, err
	}
	if len(data) > modelCapabilityProbeMaxResponseSize {
		return data[:modelCapabilityProbeMaxResponseSize], true, nil
	}
	return data, false, nil
}

func validOpenAITextCapabilityResponse(body []byte, expected string) bool {
	if gjson.GetBytes(body, "error").Exists() {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "status").String()))
	if status == "failed" || status == "incomplete" || status == "cancelled" {
		return false
	}
	return validateChallenge(extractOpenAIResponsesText(body), expected)
}

func validOpenAIChatCapabilityResponse(body []byte, expected string) bool {
	if gjson.GetBytes(body, "error").Exists() {
		return false
	}
	choices := gjson.GetBytes(body, "choices")
	if !choices.IsArray() || len(choices.Array()) == 0 {
		return false
	}
	return validateChallenge(choices.Get("0.message.content").String(), expected)
}

func validAnthropicCapabilityResponse(body []byte, expected string) bool {
	if gjson.GetBytes(body, "error").Exists() || gjson.GetBytes(body, "type").String() != "message" {
		return false
	}
	return validateChallenge(extractAnthropicMessagesText(body), expected)
}

func validGeminiTextCapabilityResponse(body []byte, expected string) bool {
	if gjson.GetBytes(body, "error").Exists() {
		return false
	}
	var texts []string
	candidates := gjson.GetBytes(body, "candidates")
	if !candidates.IsArray() {
		return false
	}
	candidates.ForEach(func(_, candidate gjson.Result) bool {
		parts := candidate.Get("content.parts")
		parts.ForEach(func(_, part gjson.Result) bool {
			if text := strings.TrimSpace(part.Get("text").String()); text != "" {
				texts = append(texts, text)
			}
			return true
		})
		return true
	})
	return validateChallenge(strings.Join(texts, "\n"), expected)
}

func validOpenAIImageCapabilityResponse(body []byte) bool {
	if gjson.GetBytes(body, "error").Exists() {
		return false
	}
	data := gjson.GetBytes(body, "data")
	if !data.IsArray() {
		return false
	}
	valid := false
	data.ForEach(func(_, item gjson.Result) bool {
		valid = strings.TrimSpace(item.Get("b64_json").String()) != "" ||
			strings.TrimSpace(item.Get("url").String()) != "" ||
			strings.TrimSpace(item.Get("image_url").String()) != ""
		return !valid
	})
	return valid
}

func validGeminiImageCapabilityResponse(body []byte) bool {
	if gjson.GetBytes(body, "error").Exists() {
		return false
	}
	valid := false
	candidates := gjson.GetBytes(body, "candidates")
	if !candidates.IsArray() {
		return false
	}
	candidates.ForEach(func(_, candidate gjson.Result) bool {
		candidate.Get("content.parts").ForEach(func(_, part gjson.Result) bool {
			inlineData := part.Get("inlineData")
			if !inlineData.Exists() {
				inlineData = part.Get("inline_data")
			}
			valid = strings.TrimSpace(inlineData.Get("data").String()) != ""
			return !valid
		})
		return !valid
	})
	return valid
}

func modelCapabilityHTTPError(statusCode int, body []byte) string {
	if statusCode == http.StatusTooManyRequests {
		return "Upstream temporarily rate-limited the model verification request (HTTP 429); model support remains unconfirmed"
	}
	message := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	if message == "" {
		message = "upstream request failed"
	}
	return sanitizeErrorMessage(fmt.Sprintf("upstream HTTP %d: %s", statusCode, message))
}

func modelCapabilityRetryAfter(header http.Header, now time.Time) time.Duration {
	raw := strings.TrimSpace(header.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	deadline, err := http.ParseTime(raw)
	if err != nil {
		return 0
	}
	delay := deadline.Sub(now)
	if delay <= 0 {
		return 0
	}
	return delay
}

func isDefinitiveModelCapabilityUnsupported(statusCode int, body []byte) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound:
		return IsUpstreamModelUnsupportedError(statusCode, body)
	default:
		return false
	}
}

func unconfirmedModelCapability(message string) ModelCapabilityProbeResult {
	return ModelCapabilityProbeResult{
		Status:  ModelCapabilityStatusUnconfirmed,
		Message: sanitizeErrorMessage(strings.TrimSpace(message)),
	}
}
