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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

const (
	openAIStableTTFTProbeInterval     = 5 * time.Minute
	openAIStableTTFTProbeTimeout      = 30 * time.Second
	openAIStableTTFTProbeStartupDelay = 15 * time.Second
	openAIStableTTFTProbeConcurrency  = 3
	openAIStableTTFTProbeBodyMax      = 4096
	openAIStableTTFTProbeModel        = "gpt-5.4-mini"
	openAIStableTTFTProbeEndpoint     = "/v1/responses"
)

var errOpenAIStableTTFTProbeUnschedulable = errors.New("stable ttft probe account not active or schedulable")

func (s *OpenAIGatewayService) Start() {
	if s == nil {
		return
	}
	s.openaiStableTTFTProbeOnce.Do(func() {
		if s.schedulerSnapshot == nil || s.accountRepo == nil || s.httpUpstream == nil {
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.openaiStableTTFTProbeCancel = cancel
		s.openaiStableTTFTProbeStopCh = make(chan struct{})
		s.openaiStableTTFTProbeWG.Add(1)
		go func() {
			defer s.openaiStableTTFTProbeWG.Done()
			s.runOpenAIStableTTFTProbeWorker(ctx)
		}()
	})
}

func (s *OpenAIGatewayService) Stop() {
	if s == nil {
		return
	}
	s.openaiStableTTFTProbeStopOnce.Do(func() {
		if s.openaiStableTTFTProbeCancel != nil {
			s.openaiStableTTFTProbeCancel()
		}
		if s.openaiStableTTFTProbeStopCh != nil {
			close(s.openaiStableTTFTProbeStopCh)
		}
	})
	s.openaiStableTTFTProbeWG.Wait()
}

func (s *OpenAIGatewayService) runOpenAIStableTTFTProbeWorker(ctx context.Context) {
	timer := time.NewTimer(openAIStableTTFTProbeStartupDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.openaiStableTTFTProbeStopCh:
			return
		case <-timer.C:
		}

		probeCtx, cancel := context.WithTimeout(ctx, openAIStableTTFTProbeInterval)
		s.runOpenAIStableTTFTProbeOnce(probeCtx)
		cancel()

		timer.Reset(openAIStableTTFTProbeInterval)
	}
}

func (s *OpenAIGatewayService) runOpenAIStableTTFTProbeOnce(ctx context.Context) {
	if s == nil || s.schedulerSnapshot == nil {
		return
	}
	groups, err := s.schedulerSnapshot.ListActiveGroupsByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		slog.Warn("openai_stable_ttft_probe_list_groups_failed", "error", err)
		return
	}
	for i := range groups {
		group := groups[i]
		if !isOpenAIStableLowTTFTGroup(&group) {
			continue
		}
		if err := s.probeOpenAIStableTTFTGroup(ctx, &group); err != nil {
			slog.Warn("openai_stable_ttft_probe_group_failed",
				"group_id", group.ID,
				"error", err,
			)
		}
	}
}

func (s *OpenAIGatewayService) probeOpenAIStableTTFTGroup(ctx context.Context, group *Group) error {
	if s == nil || group == nil || group.ID <= 0 {
		return nil
	}
	groupID := group.ID
	accounts, err := s.listFreshSchedulableAccounts(ctx, &groupID)
	if err != nil {
		return err
	}
	accounts = filterOpenAIStableTTFTProbeAccounts(ctx, accounts, group)
	if len(accounts) == 0 {
		return nil
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		return isAccountBetterByCurrentGroupOrder(accounts[i], accounts[j], &groupID)
	})

	var wg sync.WaitGroup
	sem := make(chan struct{}, openAIStableTTFTProbeConcurrency)
	for _, account := range accounts {
		if account == nil {
			continue
		}
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(account *Account) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.probeOpenAIStableTTFTAccount(ctx, account); err != nil {
				slog.Warn("openai_stable_ttft_probe_account_failed",
					"group_id", groupID,
					"account_id", account.ID,
					"error", err,
				)
			}
		}(account)
	}
	wg.Wait()
	return nil
}

func filterOpenAIStableTTFTProbeAccounts(ctx context.Context, accounts []*Account, group *Group) []*Account {
	if len(accounts) == 0 {
		return nil
	}
	req := OpenAIAccountScheduleRequest{
		GroupID:           &group.ID,
		RequestedModel:    defaultSchedulerModel,
		SchedulerEndpoint: openAIStableTTFTProbeEndpoint,
	}
	out := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || !account.IsOpenAI() {
			continue
		}
		if reason := openAIAccountStatusFilterReason(ctx, account, req, group); reason != "" {
			continue
		}
		if resolveOpenAIStableTTFTProbeModel(account) == "" {
			continue
		}
		if !accountSupportsOpenAICapabilities(account, OpenAIEndpointCapabilityChatCompletions, "") {
			continue
		}
		out = append(out, account)
	}
	return out
}

func (s *OpenAIGatewayService) probeOpenAIStableTTFTAccount(ctx context.Context, account *Account) error {
	if s == nil || account == nil {
		return nil
	}
	probeCtx, cancel := context.WithTimeout(ctx, openAIStableTTFTProbeTimeout)
	defer cancel()

	fresh, err := s.accountRepo.GetByID(probeCtx, account.ID)
	if err != nil {
		return fmt.Errorf("load probe account: %w", err)
	}
	if fresh == nil {
		return fmt.Errorf("probe account not found")
	}
	if fresh.Platform != PlatformOpenAI || fresh.Status != StatusActive || !fresh.Schedulable {
		return fmt.Errorf("%w: status=%s schedulable=%t", errOpenAIStableTTFTProbeUnschedulable, fresh.Status, fresh.Schedulable)
	}
	probeModel := resolveOpenAIStableTTFTProbeModel(fresh)
	if probeModel == "" {
		return nil
	}

	token, _, err := s.GetAccessToken(probeCtx, fresh)
	if err != nil {
		return fmt.Errorf("get probe token: %w", err)
	}
	req, err := s.buildOpenAIStableTTFTProbeRequest(probeCtx, fresh, probeModel, token)
	if err != nil {
		return err
	}

	proxyURL := ""
	if fresh.Proxy != nil {
		proxyURL = fresh.Proxy.URL()
	}
	start := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, fresh.ID, fresh.Concurrency)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	ttft, body, err := readOpenAIStableTTFTProbeResponse(resp, start)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("probe upstream HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if ttft <= 0 {
		return fmt.Errorf("probe response did not produce ttft")
	}
	if s.schedulerHealth != nil {
		s.schedulerHealth.reportSuccess(fresh.ID, probeModel, openAIStableTTFTProbeEndpoint, &ttft)
		s.schedulerHealth.reportSuccess(fresh.ID, defaultSchedulerModel, defaultSchedulerEndpoint, &ttft)
	}
	scheduler := s.getOpenAIAccountScheduler(context.Background())
	if scheduler != nil {
		scheduler.ReportResult(fresh.ID, true, &ttft)
	}
	slog.Info("openai_stable_ttft_probe_success",
		"account_id", fresh.ID,
		"model", probeModel,
		"ttft_ms", ttft,
	)
	return nil
}

func resolveOpenAIStableTTFTProbeModel(account *Account) string {
	if account == nil {
		return ""
	}
	preferred := []string{
		openAIStableTTFTProbeModel,
		"gpt-5.5",
		"gpt-5.4",
	}
	for _, model := range preferred {
		if openAIAccountSupportsModelForSchedule(account, model, false, OpenAIAccountScheduleOptions{}) {
			return model
		}
	}
	return ""
}

func openAIProbeInputMessage(text string) []map[string]any {
	return []map[string]any{
		{
			"type": "message",
			"role": "user",
			"content": []map[string]string{
				{"type": "input_text", "text": text},
			},
		},
	}
}

func (s *OpenAIGatewayService) buildOpenAIStableTTFTProbeRequest(ctx context.Context, account *Account, model, token string) (*http.Request, error) {
	upstreamModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, model, ""))
	reqBody := map[string]any{
		"model":             upstreamModel,
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
	} else if ua := account.GetOpenAIUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	return req, nil
}

func readOpenAIStableTTFTProbeResponse(resp *http.Response, start time.Time) (int, []byte, error) {
	if resp == nil || resp.Body == nil {
		return 0, nil, fmt.Errorf("empty probe response")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, openAIStableTTFTProbeBodyMax))
		return 0, body, nil
	}

	reader := bufio.NewReader(resp.Body)
	var body bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if body.Len() < openAIStableTTFTProbeBodyMax {
				remaining := openAIStableTTFTProbeBodyMax - body.Len()
				if len(line) > remaining {
					line = line[:remaining]
				}
				_, _ = body.Write(line)
			}
			trimmed := strings.TrimSpace(string(line))
			if strings.HasPrefix(trimmed, "data:") && isOpenAIStableTTFTProbeOutputEvent(strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))) {
				ttftMs := int(time.Since(start).Milliseconds())
				if ttftMs <= 0 {
					ttftMs = 1
				}
				return ttftMs, body.Bytes(), nil
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

func isOpenAIStableTTFTProbeOutputEvent(data string) bool {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return false
	}
	eventType := strings.TrimSpace(gjson.Get(data, "type").String())
	switch eventType {
	case "response.created", "response.in_progress", "response.queued", "response.output_item.added", "response.content_part.added":
		return false
	case "response.output_text.delta", "response.output_text.done", "response.completed":
		return true
	}
	if delta := strings.TrimSpace(gjson.Get(data, "delta").String()); delta != "" {
		return true
	}
	if content := strings.TrimSpace(gjson.Get(data, "choices.0.delta.content").String()); content != "" {
		return true
	}
	if output := strings.TrimSpace(gjson.Get(data, "output.0.content.0.text").String()); output != "" {
		return true
	}
	return false
}
