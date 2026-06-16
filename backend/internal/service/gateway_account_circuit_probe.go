package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const (
	gatewayAccountCircuitProbeBodyMax        = 4096
	gatewayAccountCircuitProbeAnthropicModel = "claude-haiku-4-5"
	gatewayAccountCircuitProbeGeminiModel    = "gemini-3.1-flash"
)

var (
	gatewayAccountCircuitProbeTimeout    = 30 * time.Second
	gatewayAccountCircuitProbeRetryDelay = 5 * time.Second
)

type gatewayAccountCircuitProbe struct {
	cancel context.CancelFunc
}

type gatewayAccountCircuitProbeAdapter struct {
	service *GatewayService
}

func (s *GatewayService) maybeStartGatewayAccountCircuitProbe(accountID int64, model, endpoint, category string) {
	if s == nil || accountID <= 0 || !shouldStartGatewayAccountCircuitProbe(model, endpoint, category) {
		return
	}
	if s.accountRepo == nil || s.httpUpstream == nil {
		return
	}
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	ctx, cancel := context.WithCancel(context.Background())
	probe := &gatewayAccountCircuitProbe{cancel: cancel}
	if _, loaded := s.gatewayAccountCircuitProbes.LoadOrStore(key, probe); loaded {
		cancel()
		return
	}
	slog.Info("gateway_account_circuit_probe_started",
		"account_id", accountID,
		"model", key.Model,
		"endpoint", key.Endpoint,
		"timeout", gatewayAccountCircuitProbeTimeout.String(),
		"retry_delay", gatewayAccountCircuitProbeRetryDelay.String(),
		"healthy_ttft", schedulerHealthyTTFTThreshold.String(),
		"category", category,
	)
	go s.runGatewayAccountCircuitProbe(ctx, key, category)
}

func shouldStartGatewayAccountCircuitProbe(model, endpoint, category string) bool {
	switch strings.TrimSpace(category) {
	case "rate_limit", "transient", "transient_transport", "transient_timeout", "unknown":
	default:
		return false
	}
	model = strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(model, "image") {
		return false
	}
	return schedulerTTFTPolicyEndpointAllowed(endpoint)
}

func (s *GatewayService) stopGatewayAccountCircuitProbe(accountID int64, model, endpoint string) {
	if s == nil || accountID <= 0 {
		return
	}
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	if value, loaded := s.gatewayAccountCircuitProbes.LoadAndDelete(key); loaded {
		if probe, ok := value.(*gatewayAccountCircuitProbe); ok && probe != nil && probe.cancel != nil {
			probe.cancel()
		}
	}
}

func (s *GatewayService) runGatewayAccountCircuitProbe(ctx context.Context, key accountSchedulerHealthKey, initialCategory string) {
	defer s.gatewayAccountCircuitProbes.Delete(key)
	runner := schedulerProbeRunner{
		health:     s.schedulerHealth,
		classifier: defaultSchedulerErrorClassifier{},
		adapter:    gatewayAccountCircuitProbeAdapter{service: s},
		timeout:    gatewayAccountCircuitProbeTimeout,
		retryDelay: gatewayAccountCircuitProbeRetryDelay,
	}
	runner.run(ctx, key, initialCategory)
}

func (a gatewayAccountCircuitProbeAdapter) Probe(ctx context.Context, key schedulerProbeKey) (int, []byte, int, error) {
	if a.service == nil {
		return 0, nil, 0, fmt.Errorf("gateway circuit probe dependencies unavailable")
	}
	return a.service.probeGatewayAccountCircuit(ctx, key)
}

func (a gatewayAccountCircuitProbeAdapter) OnRecovered(key schedulerProbeKey) {}

func (a gatewayAccountCircuitProbeAdapter) OnUnschedulable(key schedulerProbeKey) {}

func (a gatewayAccountCircuitProbeAdapter) ShouldContinue(key schedulerProbeKey, category string) bool {
	return shouldStartGatewayAccountCircuitProbe(key.Model, key.Endpoint, category)
}

func (a gatewayAccountCircuitProbeAdapter) LogAttrs(key schedulerProbeKey) []any {
	return []any{
		"account_id", key.AccountID,
		"model", key.Model,
		"endpoint", key.Endpoint,
	}
}

func (s *GatewayService) probeGatewayAccountCircuit(ctx context.Context, key accountSchedulerHealthKey) (int, []byte, int, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return 0, nil, 0, fmt.Errorf("gateway circuit probe dependencies unavailable")
	}

	account, err := s.accountRepo.GetByID(ctx, key.AccountID)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("load probe account: %w", err)
	}
	if account == nil {
		return 0, nil, 0, fmt.Errorf("probe account not found")
	}
	if account.Status != StatusActive || !account.Schedulable {
		return 0, nil, 0, fmt.Errorf("%w: status=%s schedulable=%t", errSchedulerProbeUnschedulable, account.Status, account.Schedulable)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	switch account.Platform {
	case PlatformAnthropic:
		req, err := s.buildGatewayAnthropicCircuitProbeRequest(ctx, account, key.Model, key.Endpoint)
		if err != nil {
			return 0, nil, 0, err
		}
		start := time.Now()
		resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
		if err != nil {
			return 0, nil, 0, err
		}
		defer func() { _ = resp.Body.Close() }()
		ttftMs, body, readErr := readGatewayAnthropicCircuitProbeResponse(resp, start)
		if resp.StatusCode >= 400 {
			return resp.StatusCode, body, ttftMs, fmt.Errorf("probe upstream HTTP %d", resp.StatusCode)
		}
		return resp.StatusCode, body, ttftMs, readErr
	case PlatformGemini:
		req, err := s.buildGatewayGeminiCircuitProbeRequest(ctx, account, key.Model)
		if err != nil {
			return 0, nil, 0, err
		}
		start := time.Now()
		resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, nil)
		if err != nil {
			return 0, nil, 0, err
		}
		defer func() { _ = resp.Body.Close() }()
		ttftMs, body, readErr := readGatewayGeminiCircuitProbeResponse(resp, start)
		if resp.StatusCode >= 400 {
			return resp.StatusCode, body, ttftMs, fmt.Errorf("probe upstream HTTP %d", resp.StatusCode)
		}
		return resp.StatusCode, body, ttftMs, readErr
	case PlatformAntigravity:
		req, err := s.buildGatewayAntigravityCircuitProbeRequest(ctx, account, key.Model)
		if err != nil {
			return 0, nil, 0, err
		}
		start := time.Now()
		resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			return 0, nil, 0, err
		}
		defer func() { _ = resp.Body.Close() }()
		ttftMs, body, readErr := readGatewayAntigravityCircuitProbeResponse(resp, start)
		if resp.StatusCode >= 400 {
			return resp.StatusCode, body, ttftMs, fmt.Errorf("probe upstream HTTP %d", resp.StatusCode)
		}
		return resp.StatusCode, body, ttftMs, readErr
	default:
		return 0, nil, 0, fmt.Errorf("gateway circuit probe unsupported platform: %s", account.Platform)
	}
}

func (s *GatewayService) buildGatewayAnthropicCircuitProbeRequest(ctx context.Context, account *Account, model, endpoint string) (*http.Request, error) {
	model = strings.TrimSpace(model)
	if model == "" || model == defaultSchedulerModel {
		model = gatewayAccountCircuitProbeAnthropicModel
	}
	if account.Type == AccountTypeServiceAccount {
		model = normalizeVertexAnthropicModelID(claude.NormalizeModelID(model))
	} else if account.Type != AccountTypeAPIKey {
		model = claude.NormalizeModelID(model)
	}
	if account.Type == AccountTypeAPIKey {
		model = account.GetMappedModel(model)
	}
	if model == "" {
		model = gatewayAccountCircuitProbeAnthropicModel
	}

	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get probe token: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 1,
		"stream":     true,
		"messages": []map[string]any{
			{"role": "user", "content": "Reply with 1."},
		},
	})
	if err != nil {
		return nil, err
	}

	req, _, err := s.buildUpstreamRequest(ctx, nil, account, payload, token, tokenType, model, true, false)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	return req, nil
}

func (s *GatewayService) buildGatewayGeminiCircuitProbeRequest(ctx context.Context, account *Account, model string) (*http.Request, error) {
	model = strings.TrimSpace(model)
	if model == "" || model == defaultSchedulerModel {
		model = gatewayAccountCircuitProbeGeminiModel
	}
	if account.Type == AccountTypeAPIKey || account.Type == AccountTypeServiceAccount {
		model = account.GetMappedModel(model)
	}
	if model == "" {
		model = gatewayAccountCircuitProbeGeminiModel
	}

	payload := []byte(`{"contents":[{"role":"user","parts":[{"text":"Reply with 1."}]}]}`)
	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return nil, errors.New("gemini api_key not configured")
		}
		req, err := s.newGatewayGeminiAIStudioCircuitProbeRequest(ctx, account, model, payload)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-goog-api-key", apiKey)
		return req, nil
	case AccountTypeOAuth:
		if s.geminiTokenProvider == nil {
			return nil, errors.New("gemini token provider not configured")
		}
		accessToken, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, err
		}
		projectID := strings.TrimSpace(account.GetCredential("project_id"))
		if projectID != "" {
			req, err := s.newGatewayGeminiCodeAssistCircuitProbeRequest(ctx, model, projectID, payload)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+accessToken)
			return req, nil
		}
		req, err := s.newGatewayGeminiAIStudioCircuitProbeRequest(ctx, account, model, payload)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return req, nil
	case AccountTypeServiceAccount:
		if s.geminiTokenProvider == nil {
			return nil, errors.New("gemini token provider not configured")
		}
		accessToken, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, err
		}
		fullURL, err := buildVertexGeminiURL(account.VertexProjectID(), account.VertexLocation(model), model, "streamGenerateContent", true)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(normalizeGeminiRequestForAIStudio(payload)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return req, nil
	default:
		return nil, fmt.Errorf("gateway gemini circuit probe unsupported account type: %s", account.Type)
	}
}

func (s *GatewayService) newGatewayGeminiAIStudioCircuitProbeRequest(ctx context.Context, account *Account, model string, payload []byte) (*http.Request, error) {
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", strings.TrimRight(normalizedBaseURL, "/"), model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(normalizeGeminiRequestForAIStudio(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	return req, nil
}

func (s *GatewayService) newGatewayGeminiCodeAssistCircuitProbeRequest(ctx context.Context, model, projectID string, payload []byte) (*http.Request, error) {
	baseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
	if err != nil {
		return nil, err
	}
	var inner any
	if err := json.Unmarshal(payload, &inner); err != nil {
		return nil, fmt.Errorf("failed to parse gemini probe request: %w", err)
	}
	wrapped, err := json.Marshal(map[string]any{
		"model":   model,
		"project": strings.TrimSpace(projectID),
		"request": inner,
	})
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/v1internal:streamGenerateContent?alt=sse", strings.TrimRight(baseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(wrapped))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
	return req, nil
}

func (s *GatewayService) buildGatewayAntigravityCircuitProbeRequest(ctx context.Context, account *Account, model string) (*http.Request, error) {
	if s.antigravityTokenProvider == nil {
		return nil, errors.New("antigravity token provider not configured")
	}
	token, err := s.antigravityTokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get antigravity probe token: %w", err)
	}

	model = resolveGatewayAntigravityCircuitProbeModel(account, model)
	if model == "" {
		return nil, errors.New("antigravity probe model not configured")
	}
	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	payload, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": "Reply with 1."},
				},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]any{
				{"text": antigravity.GetDefaultIdentityPatch()},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 1,
		},
	})
	if err != nil {
		return nil, err
	}
	wrapped, err := s.wrapGatewayAntigravityCircuitProbeRequest(projectID, model, payload)
	if err != nil {
		return nil, err
	}

	baseURL := resolveAntigravityForwardBaseURL()
	if baseURL == "" {
		return nil, errors.New("no antigravity forward base url configured")
	}
	req, err := antigravity.NewAPIRequestWithURL(ctx, baseURL, "streamGenerateContent", token, wrapped)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	return req, nil
}

func resolveGatewayAntigravityCircuitProbeModel(account *Account, model string) string {
	model = strings.TrimSpace(model)
	if model == "" || model == defaultSchedulerModel {
		model = "gemini-3-flash"
	}
	if isImageGenerationModel(model) {
		model = "gemini-3-flash"
	}
	if account == nil {
		return model
	}
	mappedModel := mapAntigravityModel(account, model)
	if strings.TrimSpace(mappedModel) == "" && model != "gemini-3-flash" {
		mappedModel = mapAntigravityModel(account, "gemini-3-flash")
	}
	return strings.TrimSpace(mappedModel)
}

func (s *GatewayService) wrapGatewayAntigravityCircuitProbeRequest(projectID, model string, originalBody []byte) ([]byte, error) {
	var request any
	if err := json.Unmarshal(originalBody, &request); err != nil {
		return nil, fmt.Errorf("parse antigravity probe request: %w", err)
	}
	return json.Marshal(map[string]any{
		"project":     strings.TrimSpace(projectID),
		"requestId":   "probe-" + uuid.New().String(),
		"userAgent":   "antigravity",
		"requestType": "agent",
		"model":       model,
		"request":     request,
	})
}

func readGatewayAnthropicCircuitProbeResponse(resp *http.Response, start time.Time) (int, []byte, error) {
	return readGatewaySSECircuitProbeResponse(resp, start, isGatewayAnthropicCircuitProbeOutputEvent)
}

func readGatewayGeminiCircuitProbeResponse(resp *http.Response, start time.Time) (int, []byte, error) {
	return readGatewaySSECircuitProbeResponse(resp, start, isGatewayGeminiCircuitProbeOutputEvent)
}

func readGatewayAntigravityCircuitProbeResponse(resp *http.Response, start time.Time) (int, []byte, error) {
	return readGatewaySSECircuitProbeResponse(resp, start, isGatewayAntigravityCircuitProbeOutputEvent)
}

func readGatewaySSECircuitProbeResponse(resp *http.Response, start time.Time, isOutput func(string, string) bool) (int, []byte, error) {
	if resp == nil || resp.Body == nil {
		return 0, nil, fmt.Errorf("empty probe response")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, gatewayAccountCircuitProbeBodyMax))
		return 0, body, nil
	}

	reader := bufio.NewReader(resp.Body)
	var body bytes.Buffer
	eventName := ""
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if body.Len() < gatewayAccountCircuitProbeBodyMax {
				remaining := gatewayAccountCircuitProbeBodyMax - body.Len()
				if len(line) > remaining {
					line = line[:remaining]
				}
				body.Write(line)
			}
			trimmed := strings.TrimSpace(string(line))
			if strings.HasPrefix(trimmed, "event:") {
				eventName = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
			}
			if strings.HasPrefix(trimmed, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if isOutput(eventName, data) {
					ttftMs := int(time.Since(start).Milliseconds())
					if ttftMs <= 0 {
						ttftMs = 1
					}
					return ttftMs, body.Bytes(), nil
				}
			}
			if trimmed == "" {
				eventName = ""
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0, body.Bytes(), nil
			}
			return 0, body.Bytes(), err
		}
	}
}

func isGatewayAnthropicCircuitProbeOutputEvent(eventName, data string) bool {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return false
	}
	if eventName == "content_block_delta" {
		if text := strings.TrimSpace(gjson.Get(data, "delta.text").String()); text != "" {
			return true
		}
		if partialJSON := strings.TrimSpace(gjson.Get(data, "delta.partial_json").String()); partialJSON != "" {
			return true
		}
	}
	eventType := strings.TrimSpace(gjson.Get(data, "type").String())
	switch eventType {
	case "content_block_delta", "message_delta", "message_stop":
		return true
	default:
		return false
	}
}

func isGatewayGeminiCircuitProbeOutputEvent(_ string, data string) bool {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return false
	}
	if text := strings.TrimSpace(gjson.Get(data, "candidates.0.content.parts.0.text").String()); text != "" {
		return true
	}
	if text := strings.TrimSpace(gjson.Get(data, "candidates.0.content.parts.0.inlineData.data").String()); text != "" {
		return true
	}
	return false
}

func isGatewayAntigravityCircuitProbeOutputEvent(eventName, data string) bool {
	if isGatewayGeminiCircuitProbeOutputEvent(eventName, data) {
		return true
	}
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return false
	}
	if text := strings.TrimSpace(gjson.Get(data, "response.candidates.0.content.parts.0.text").String()); text != "" {
		return true
	}
	if text := strings.TrimSpace(gjson.Get(data, "response.candidates.0.content.parts.0.inlineData.data").String()); text != "" {
		return true
	}
	return false
}
