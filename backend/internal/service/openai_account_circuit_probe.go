package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
)

const (
	openAIAccountCircuitProbeBodyMax = 4096
	openAIAccountCircuitProbeModel   = "gpt-4.1-mini"
)

var (
	openAIAccountCircuitProbeTimeout    = 30 * time.Second
	openAIAccountCircuitProbeRetryDelay = 5 * time.Second
)

type openAIAccountCircuitProbe struct {
	cancel context.CancelFunc
}

type openAIAccountCircuitProbeAdapter struct {
	service *OpenAIGatewayService
}

func (s *OpenAIGatewayService) maybeStartOpenAIAccountCircuitProbe(accountID int64, model, endpoint, category string) {
	if s == nil || accountID <= 0 || !shouldStartOpenAIAccountCircuitProbe(model, endpoint, category) {
		return
	}
	if s.accountRepo == nil || s.httpUpstream == nil {
		return
	}
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	ctx, cancel := context.WithCancel(context.Background())
	probe := &openAIAccountCircuitProbe{cancel: cancel}
	if _, loaded := s.openaiAccountCircuitProbes.LoadOrStore(key, probe); loaded {
		cancel()
		return
	}
	runnerCfg := s.openAIAccountCircuitProbeRunnerConfig()
	slog.Info("account_circuit_probe_started",
		"account_id", accountID,
		"model", key.Model,
		"endpoint", key.Endpoint,
		"timeout", runnerCfg.timeout.String(),
		"retry_delay", runnerCfg.retryDelay.String(),
		"max_concurrency", runnerCfg.maxConcurrency,
		"healthy_ttft", schedulerHealthyTTFTThreshold.String(),
		"category", category,
	)
	go s.runOpenAIAccountCircuitProbe(ctx, key, category)
}

func shouldStartOpenAIAccountCircuitProbe(model, endpoint, category string) bool {
	switch strings.TrimSpace(category) {
	case "rate_limit", "transient", "transient_transport", "transient_timeout", "unknown":
	default:
		return false
	}
	model = strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(model, "image") {
		return false
	}
	endpoint = normalizeSchedulerDimension(endpoint, defaultSchedulerEndpoint)
	return schedulerTTFTPolicyEndpointAllowed(endpoint)
}

func (s *OpenAIGatewayService) stopOpenAIAccountCircuitProbe(accountID int64, model, endpoint string) {
	if s == nil || accountID <= 0 {
		return
	}
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	if value, loaded := s.openaiAccountCircuitProbes.LoadAndDelete(key); loaded {
		if probe, ok := value.(*openAIAccountCircuitProbe); ok && probe != nil && probe.cancel != nil {
			probe.cancel()
		}
	}
}

func (s *OpenAIGatewayService) runOpenAIAccountCircuitProbe(ctx context.Context, key accountSchedulerHealthKey, initialCategory string) {
	defer s.openaiAccountCircuitProbes.Delete(key)
	runnerCfg := s.openAIAccountCircuitProbeRunnerConfig()
	runner := schedulerProbeRunner{
		health:     s.schedulerHealth,
		classifier: schedulerClassifierForPlatform(PlatformOpenAI),
		adapter:    openAIAccountCircuitProbeAdapter{service: s},
		timeout:    runnerCfg.timeout,
		retryDelay: runnerCfg.retryDelay,
		limiter:    s.openaiAccountCircuitProbeLimiter.get(runnerCfg.maxConcurrency),
	}
	runner.run(ctx, key, initialCategory)
}

func (s *OpenAIGatewayService) openAIAccountCircuitProbeRunnerConfig() schedulerProbeRunnerConfig {
	if s == nil {
		return schedulerProbeRunnerConfigFromConfig(nil, openAIAccountCircuitProbeTimeout, openAIAccountCircuitProbeRetryDelay)
	}
	return schedulerProbeRunnerConfigFromConfig(s.cfg, openAIAccountCircuitProbeTimeout, openAIAccountCircuitProbeRetryDelay)
}

func (a openAIAccountCircuitProbeAdapter) Probe(ctx context.Context, key schedulerProbeKey) (int, []byte, int, error) {
	if a.service == nil {
		return 0, nil, 0, fmt.Errorf("openai circuit probe dependencies unavailable")
	}
	return a.service.probeOpenAIAccountCircuit(ctx, key)
}

func (a openAIAccountCircuitProbeAdapter) OnRecovered(key schedulerProbeKey) {
	if a.service != nil {
		a.service.recoverOpenAIAccountCircuit(key.AccountID)
	}
}

func (a openAIAccountCircuitProbeAdapter) OnUnschedulable(key schedulerProbeKey) {
	if a.service != nil {
		a.service.recoverOpenAIAccountCircuit(key.AccountID)
	}
}

func (a openAIAccountCircuitProbeAdapter) ShouldContinue(key schedulerProbeKey, category string) bool {
	return shouldStartOpenAIAccountCircuitProbe(key.Model, key.Endpoint, category)
}

func (a openAIAccountCircuitProbeAdapter) LogAttrs(key schedulerProbeKey) []any {
	return []any{
		"account_id", key.AccountID,
		"model", key.Model,
		"endpoint", key.Endpoint,
	}
}

func (s *OpenAIGatewayService) probeOpenAIAccountCircuit(ctx context.Context, key accountSchedulerHealthKey) (int, []byte, int, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return 0, nil, 0, fmt.Errorf("openai circuit probe dependencies unavailable")
	}
	probeCtx, cancel := context.WithTimeout(ctx, s.openAIAccountCircuitProbeRunnerConfig().timeout)
	defer cancel()

	account, err := s.accountRepo.GetByID(probeCtx, key.AccountID)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("load probe account: %w", err)
	}
	if account == nil {
		return 0, nil, 0, fmt.Errorf("probe account not found")
	}
	if account.Platform != PlatformOpenAI || account.Status != StatusActive || !account.Schedulable {
		return 0, nil, 0, fmt.Errorf("%w: status=%s schedulable=%t", errSchedulerProbeUnschedulable, account.Status, account.Schedulable)
	}

	token, _, err := s.GetAccessToken(probeCtx, account)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("get probe token: %w", err)
	}
	req, err := s.buildOpenAIAccountCircuitProbeRequest(probeCtx, account, key.Model, key.Endpoint, token)
	if err != nil {
		return 0, nil, 0, err
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	start := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return 0, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	ttftMs, body, readErr := readOpenAIStableTTFTProbeResponse(resp, start)
	if resp.StatusCode >= 400 {
		return resp.StatusCode, body, ttftMs, fmt.Errorf("probe upstream HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, body, ttftMs, readErr
}

func (s *OpenAIGatewayService) buildOpenAIAccountCircuitProbeRequest(ctx context.Context, account *Account, model, endpoint, token string) (*http.Request, error) {
	model = strings.TrimSpace(model)
	if model == "" || model == defaultSchedulerModel {
		model = openAIAccountCircuitProbeModel
	}
	upstreamModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, model, ""))
	endpoint = normalizeSchedulerDimension(endpoint, defaultSchedulerEndpoint)

	if endpoint == "/v1/chat/completions" && account.Type == AccountTypeAPIKey && !openai_compat.ShouldUseResponsesAPI(account.Extra) {
		return s.buildOpenAIAccountCircuitChatProbeRequest(ctx, account, upstreamModel, token)
	}
	return s.buildOpenAIAccountCircuitResponsesProbeRequest(ctx, account, upstreamModel, token)
}

func (s *OpenAIGatewayService) buildOpenAIAccountCircuitChatProbeRequest(ctx context.Context, account *Account, model, token string) (*http.Request, error) {
	payload, err := json.Marshal(map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": "Reply with 1."}},
		"max_tokens": 1,
		"stream":     true,
	})
	if err != nil {
		return nil, err
	}
	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildOpenAIChatCompletionsURL(validatedURL), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if ua := account.GetOpenAIUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	return req, nil
}

func (s *OpenAIGatewayService) buildOpenAIAccountCircuitResponsesProbeRequest(ctx context.Context, account *Account, model, token string) (*http.Request, error) {
	reqBody := map[string]any{
		"model":             model,
		"instructions":      "Health check. Reply with 1.",
		"input":             openAIProbeInputMessage("Reply with 1."),
		"max_output_tokens": 1,
		"stream":            true,
	}
	if account.Type == AccountTypeOAuth {
		applyCodexOAuthTransform(reqBody, true, false)
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	targetURL := openaiPlatformAPIURL
	if account.Type == AccountTypeOAuth {
		targetURL = chatgptCodexURL
	} else if baseURL := account.GetOpenAIBaseURL(); baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = buildOpenAIResponsesURL(validatedURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if account.Type == AccountTypeOAuth {
		req.Host = "chatgpt.com"
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", "codex_cli_rs")
		req.Header.Set("User-Agent", codexCLIUserAgent)
	} else {
		if ua := account.GetOpenAIUserAgent(); ua != "" {
			req.Header.Set("User-Agent", ua)
		}
	}
	return req, nil
}
