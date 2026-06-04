package service

import (
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

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
)

const (
	openAIAccountCircuitProbeTimeout = 15 * time.Second
	openAIAccountCircuitProbeBodyMax = 4096
	openAIAccountCircuitProbeModel   = "gpt-4.1-mini"
)

var openAIAccountCircuitProbeInterval = 5 * time.Second

var errOpenAIAccountCircuitProbeUnschedulable = errors.New("probe account not active or schedulable")

type openAIAccountCircuitProbe struct {
	cancel context.CancelFunc
}

func (s *OpenAIGatewayService) maybeStartOpenAIAccountCircuitProbe(accountID int64, model, endpoint, category string) {
	if s == nil || accountID <= 0 || !shouldStartOpenAIAccountCircuitProbe(model, endpoint, category) {
		return
	}
	key := makeAccountSchedulerHealthKey(accountID, model, endpoint)
	ctx, cancel := context.WithCancel(context.Background())
	probe := &openAIAccountCircuitProbe{cancel: cancel}
	if _, loaded := s.openaiAccountCircuitProbes.LoadOrStore(key, probe); loaded {
		cancel()
		return
	}
	slog.Info("account_circuit_probe_started",
		"account_id", accountID,
		"model", key.Model,
		"endpoint", key.Endpoint,
		"interval", openAIAccountCircuitProbeInterval.String(),
		"category", category,
	)
	go s.runOpenAIAccountCircuitProbe(ctx, key, category)
}

func shouldStartOpenAIAccountCircuitProbe(model, endpoint, category string) bool {
	switch strings.TrimSpace(category) {
	case "transient", "transient_transport", "transient_timeout", "unknown":
	default:
		return false
	}
	model = strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(model, "image") {
		return false
	}
	endpoint = normalizeSchedulerDimension(endpoint, defaultSchedulerEndpoint)
	return endpoint == "/v1/chat/completions" || endpoint == "/v1/responses" || endpoint == defaultSchedulerEndpoint
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

	timer := time.NewTimer(openAIAccountCircuitProbeInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		statusCode, body, err := s.probeOpenAIAccountCircuit(ctx, key)
		if errors.Is(err, errOpenAIAccountCircuitProbeUnschedulable) {
			if s.schedulerHealth != nil {
				s.schedulerHealth.clear(key.AccountID, key.Model, key.Endpoint)
			}
			s.recoverOpenAIAccountCircuit(key.AccountID)
			slog.Info("account_circuit_probe_stopped",
				"account_id", key.AccountID,
				"model", key.Model,
				"endpoint", key.Endpoint,
				"category", "manual_unschedulable",
				"error", err,
			)
			return
		}
		if err == nil && statusCode >= 200 && statusCode < 400 {
			if s.schedulerHealth != nil {
				s.schedulerHealth.clear(key.AccountID, key.Model, key.Endpoint)
			}
			s.recoverOpenAIAccountCircuit(key.AccountID)
			slog.Info("account_circuit_probe_recovered",
				"account_id", key.AccountID,
				"model", key.Model,
				"endpoint", key.Endpoint,
				"status_code", statusCode,
			)
			return
		}

		category := schedulerFailureCategory(statusCode, body)
		if err != nil && statusCode == 0 {
			category = schedulerFailureCategory(0, []byte(err.Error()))
		}
		if category == "" || category == "unknown" {
			category = strings.TrimSpace(initialCategory)
			if category == "" || category == "unknown" {
				category = "error"
			}
		}
		if s.schedulerHealth != nil {
			s.schedulerHealth.reportFailure(key.AccountID, key.Model, key.Endpoint, category, schedulerCooldownForCategory(category, nil))
		}

		slog.Warn("account_circuit_probe_failed",
			"account_id", key.AccountID,
			"model", key.Model,
			"endpoint", key.Endpoint,
			"status_code", statusCode,
			"category", category,
			"error", err,
		)

		if !shouldStartOpenAIAccountCircuitProbe(key.Model, key.Endpoint, category) {
			slog.Info("account_circuit_probe_stopped",
				"account_id", key.AccountID,
				"model", key.Model,
				"endpoint", key.Endpoint,
				"category", category,
			)
			return
		}

		timer.Reset(openAIAccountCircuitProbeInterval)
	}
}

func (s *OpenAIGatewayService) probeOpenAIAccountCircuit(ctx context.Context, key accountSchedulerHealthKey) (int, []byte, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return 0, nil, fmt.Errorf("openai circuit probe dependencies unavailable")
	}
	probeCtx, cancel := context.WithTimeout(ctx, openAIAccountCircuitProbeTimeout)
	defer cancel()

	account, err := s.accountRepo.GetByID(probeCtx, key.AccountID)
	if err != nil {
		return 0, nil, fmt.Errorf("load probe account: %w", err)
	}
	if account == nil {
		return 0, nil, fmt.Errorf("probe account not found")
	}
	if account.Platform != PlatformOpenAI || account.Status != StatusActive || !account.Schedulable {
		return 0, nil, fmt.Errorf("%w: status=%s schedulable=%t", errOpenAIAccountCircuitProbeUnschedulable, account.Status, account.Schedulable)
	}

	token, _, err := s.GetAccessToken(probeCtx, account)
	if err != nil {
		return 0, nil, fmt.Errorf("get probe token: %w", err)
	}
	req, err := s.buildOpenAIAccountCircuitProbeRequest(probeCtx, account, key.Model, key.Endpoint, token)
	if err != nil {
		return 0, nil, err
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, openAIAccountCircuitProbeBodyMax))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, body, fmt.Errorf("probe upstream HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, body, nil
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
		"stream":     false,
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
	req.Header.Set("Accept", "application/json")
	if ua := account.GetOpenAIUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	return req, nil
}

func (s *OpenAIGatewayService) buildOpenAIAccountCircuitResponsesProbeRequest(ctx context.Context, account *Account, model, token string) (*http.Request, error) {
	reqBody := map[string]any{
		"model":             model,
		"instructions":      "Health check. Reply with 1.",
		"input":             "Reply with 1.",
		"max_output_tokens": 1,
		"stream":            false,
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
	if account.Type == AccountTypeOAuth {
		req.Host = "chatgpt.com"
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", "codex_cli_rs")
		req.Header.Set("User-Agent", codexCLIUserAgent)
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
		if ua := account.GetOpenAIUserAgent(); ua != "" {
			req.Header.Set("User-Agent", ua)
		}
	}
	return req, nil
}
