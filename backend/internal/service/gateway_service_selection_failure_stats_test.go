package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type gatewayCircuitProbeUpstreamStub struct {
	mu       sync.Mutex
	calls    int
	callCh   chan int
	lastReq  *http.Request
	lastBody string
}

func (u *gatewayCircuitProbeUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	body := ""
	if req != nil && req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		body = string(raw)
		req.Body = io.NopCloser(strings.NewReader(body))
	}
	u.mu.Lock()
	u.calls++
	call := u.calls
	u.lastReq = req
	u.lastBody = body
	u.mu.Unlock()
	if u.callCh != nil {
		select {
		case u.callCh <- call:
		default:
		}
	}
	responseBody := "event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"1\"}}\n\n"
	if req != nil {
		reqURL := req.URL.String()
		if strings.Contains(reqURL, "generativelanguage.googleapis.com") ||
			strings.Contains(reqURL, "cloudcode-pa.googleapis.com") ||
			strings.Contains(reqURL, "aiplatform.googleapis.com") {
			responseBody = "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"1\"}]}}]}\n\n"
		} else if strings.Contains(reqURL, "v1internal:streamGenerateContent") {
			responseBody = "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"1\"}]}}]}}\n\n"
		}
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}, nil
}

func (u *gatewayCircuitProbeUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func (u *gatewayCircuitProbeUpstreamStub) callCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.calls
}

func (u *gatewayCircuitProbeUpstreamStub) lastRequest() (*http.Request, string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.lastReq, u.lastBody
}

func TestCollectSelectionFailureStats(t *testing.T) {
	svc := &GatewayService{}
	model := "gpt-5.4"
	resetAt := time.Now().Add(2 * time.Minute).Format(time.RFC3339)

	accounts := []*Account{
		// excluded
		{
			ID:          1,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
		// unschedulable
		{
			ID:          2,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: false,
		},
		// platform filtered
		{
			ID:          3,
			Platform:    PlatformAntigravity,
			Status:      StatusActive,
			Schedulable: true,
		},
		// model unsupported
		{
			ID:          4,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-image": "gpt-image",
				},
			},
		},
		// model rate limited
		{
			ID:          5,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"model_rate_limits": map[string]any{
					model: map[string]any{
						"rate_limit_reset_at": resetAt,
					},
				},
			},
		},
		// eligible
		{
			ID:          6,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
	}

	excluded := map[int64]struct{}{1: {}}
	stats := svc.collectSelectionFailureStats(context.Background(), accounts, model, PlatformOpenAI, excluded, false)

	if stats.Total != 6 {
		t.Fatalf("total=%d want=6", stats.Total)
	}
	if stats.Excluded != 1 {
		t.Fatalf("excluded=%d want=1", stats.Excluded)
	}
	if stats.Unschedulable != 1 {
		t.Fatalf("unschedulable=%d want=1", stats.Unschedulable)
	}
	if stats.PlatformFiltered != 1 {
		t.Fatalf("platform_filtered=%d want=1", stats.PlatformFiltered)
	}
	if stats.ModelUnsupported != 1 {
		t.Fatalf("model_unsupported=%d want=1", stats.ModelUnsupported)
	}
	if stats.ModelRateLimited != 1 {
		t.Fatalf("model_rate_limited=%d want=1", stats.ModelRateLimited)
	}
	if stats.Eligible != 1 {
		t.Fatalf("eligible=%d want=1", stats.Eligible)
	}
}

func TestDiagnoseSelectionFailure_UnschedulableDetail(t *testing.T) {
	svc := &GatewayService{}
	acc := &Account{
		ID:          7,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: false,
	}

	diagnosis := svc.diagnoseSelectionFailure(context.Background(), acc, "gpt-5.4", PlatformOpenAI, map[int64]struct{}{}, false)
	if diagnosis.Category != "unschedulable" {
		t.Fatalf("category=%s want=unschedulable", diagnosis.Category)
	}
	if diagnosis.Detail != "generic_unschedulable" {
		t.Fatalf("detail=%s want=generic_unschedulable", diagnosis.Detail)
	}
}

func TestDiagnoseSelectionFailure_ModelRateLimitedDetail(t *testing.T) {
	svc := &GatewayService{}
	model := "gpt-5.4"
	resetAt := time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
	acc := &Account{
		ID:          8,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				model: map[string]any{
					"rate_limit_reset_at": resetAt,
				},
			},
		},
	}

	diagnosis := svc.diagnoseSelectionFailure(context.Background(), acc, model, PlatformOpenAI, map[int64]struct{}{}, false)
	if diagnosis.Category != "model_rate_limited" {
		t.Fatalf("category=%s want=model_rate_limited", diagnosis.Category)
	}
	if !strings.Contains(diagnosis.Detail, "remaining=") {
		t.Fatalf("detail=%s want contains remaining=", diagnosis.Detail)
	}
}

func TestGatewaySelectionDiagnostics_CircuitOpen(t *testing.T) {
	groupID := int64(11)
	model := "claude-opus-4-7"
	endpoint := "/v1/messages"
	account := &Account{
		ID:            38800,
		Platform:      PlatformAnthropic,
		Status:        StatusActive,
		Schedulable:   true,
		Concurrency:   5,
		AccountGroups: []AccountGroup{{GroupID: groupID}},
		Credentials: map[string]any{
			"model_mapping": map[string]any{model: model},
		},
	}
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	svc.schedulerHealth.reportFailure(account.ID, model, endpoint, "transient_transport", time.Minute)

	err := svc.newGatewayNoAvailableError(
		context.Background(),
		&groupID,
		model,
		PlatformAnthropic,
		endpoint,
		[]*Account{account},
		nil,
		false,
		&Group{ID: groupID, Platform: PlatformAnthropic, Status: StatusActive},
		false,
		map[int64]*AccountLoadInfo{account.ID: {AccountID: account.ID, LoadRate: 0}},
	)

	var noAvailable *GatewayNoAvailableAccountsError
	if !errors.As(err, &noAvailable) {
		t.Fatalf("expected GatewayNoAvailableAccountsError, got %T", err)
	}
	diag := noAvailable.Diagnostics
	if !diag.Collected {
		t.Fatal("expected diagnostics to be collected")
	}
	if diag.ModelSupportedCount != 1 || diag.EndpointSupportedCount != 1 || diag.StateAllowedCount != 1 {
		t.Fatalf("unexpected pre-circuit counts: model=%d endpoint=%d state=%d", diag.ModelSupportedCount, diag.EndpointSupportedCount, diag.StateAllowedCount)
	}
	if diag.CircuitAllowedCount != 0 || diag.FinalCandidateCount != 0 {
		t.Fatalf("expected circuit to remove final candidate, circuit_allowed=%d final=%d", diag.CircuitAllowedCount, diag.FinalCandidateCount)
	}
	if len(diag.CircuitFilteredAccountIDs) != 1 || diag.CircuitFilteredAccountIDs[0] != account.ID {
		t.Fatalf("expected account %d circuit-filtered, got %v", account.ID, diag.CircuitFilteredAccountIDs)
	}
	if diag.FilterReasonCounts["scheduler_circuit_open"] != 1 {
		t.Fatalf("expected scheduler_circuit_open skip reason, got %v", diag.FilterReasonCounts)
	}
	if len(diag.SkippedAccounts) != 1 || diag.SkippedAccounts[0].CircuitState != schedulerCircuitOpen || diag.SkippedAccounts[0].CircuitEndpoint != endpoint {
		t.Fatalf("expected scoped circuit details in skipped account, got %+v", diag.SkippedAccounts)
	}
	skipped := diag.SkippedAccounts[0]
	if skipped.Reason != "scheduler_circuit_open" || skipped.CircuitReason != "transient_transport" || skipped.CircuitModel != model {
		t.Fatalf("expected scheduler circuit reason details in skipped account, got %+v", skipped)
	}
	if skipped.CircuitRetryAt == nil {
		t.Fatalf("expected circuit retry time in skipped account, got %+v", skipped)
	}
	if skipped.CircuitRetryRemainingSec == nil || *skipped.CircuitRetryRemainingSec <= 0 {
		t.Fatalf("expected positive circuit retry remaining seconds, got %+v", skipped)
	}
}

func TestGatewaySelectionDiagnostics_HalfOpenPendingOmitsPastRetryAt(t *testing.T) {
	groupID := int64(42)
	model := "claude-opus-4-7"
	endpoint := "/v1/messages"
	account := &Account{
		ID:            38823,
		Platform:      PlatformAnthropic,
		Type:          AccountTypeOAuth,
		Status:        StatusActive,
		Schedulable:   true,
		AccountGroups: []AccountGroup{{GroupID: groupID}},
		Credentials: map[string]any{
			"model_mapping": map[string]any{model: model},
		},
	}
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	svc.schedulerHealth.reportFailure(account.ID, model, endpoint, "transient_transport", time.Millisecond)
	key := makeAccountSchedulerHealthKey(account.ID, model, endpoint)
	entry, ok := svc.schedulerHealth.get(key)
	if !ok {
		t.Fatal("expected scheduler health entry")
	}
	entry.mu.Lock()
	entry.cooldownUntil = time.Now().Add(-time.Second)
	entry.circuitState = schedulerCircuitHalfOpen
	entry.halfOpenInFlight = true
	entry.mu.Unlock()

	err := svc.newGatewayNoAvailableError(
		context.Background(),
		&groupID,
		model,
		PlatformAnthropic,
		endpoint,
		[]*Account{account},
		nil,
		false,
		&Group{ID: groupID, Platform: PlatformAnthropic, Status: StatusActive},
		false,
		nil,
	)

	var noAvailable *GatewayNoAvailableAccountsError
	if !errors.As(err, &noAvailable) {
		t.Fatalf("expected GatewayNoAvailableAccountsError, got %T", err)
	}
	if len(noAvailable.Diagnostics.SkippedAccounts) != 1 {
		t.Fatalf("expected one skipped account, got %+v", noAvailable.Diagnostics.SkippedAccounts)
	}
	skipped := noAvailable.Diagnostics.SkippedAccounts[0]
	if skipped.Reason != "scheduler_half_open_in_flight" {
		t.Fatalf("expected half-open in-flight reason, got %+v", skipped)
	}
	if skipped.CircuitRetryAt != nil || skipped.CircuitRetryRemainingSec != nil {
		t.Fatalf("expected no stale retry time for half-open in-flight account, got %+v", skipped)
	}
}

func TestGatewayServiceReportAccountScheduleSuccessForRequestClearsContextEndpoint(t *testing.T) {
	accountID := int64(38800)
	model := "claude-opus-4-7"
	explicitEndpoint := "anthropic"
	contextEndpoint := "/v1/messages"
	ctx := WithSchedulerEndpoint(context.Background(), contextEndpoint)
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}

	failoverErr := &UpstreamFailoverError{
		StatusCode:   0,
		ResponseBody: []byte("openai_request_error: context canceled"),
	}
	svc.ReportAccountScheduleFailure(ctx, accountID, model, explicitEndpoint, failoverErr)

	if snap := svc.schedulerHealth.snapshot(accountID, model, explicitEndpoint, true); snap.CircuitState != schedulerCircuitOpen {
		t.Fatalf("explicit endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitOpen)
	}
	if snap := svc.schedulerHealth.snapshot(accountID, model, contextEndpoint, true); snap.CircuitState != schedulerCircuitOpen {
		t.Fatalf("context endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitOpen)
	}

	firstTokenMs := 42
	svc.ReportAccountScheduleSuccessForRequest(ctx, accountID, model, explicitEndpoint, &firstTokenMs)

	if snap := svc.schedulerHealth.snapshot(accountID, model, explicitEndpoint, true); snap.CircuitState != schedulerCircuitClosed {
		t.Fatalf("explicit endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitClosed)
	}
	if snap := svc.schedulerHealth.snapshot(accountID, model, contextEndpoint, true); snap.CircuitState != schedulerCircuitClosed {
		t.Fatalf("context endpoint circuit=%s want=%s", snap.CircuitState, schedulerCircuitClosed)
	}
}

func TestGatewayServiceReportAccountScheduleSuccessForRequestSlowTTFTOpensCircuit(t *testing.T) {
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	schedulerHealthyTTFTThreshold = 15 * time.Millisecond
	defer func() {
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	accountID := int64(38801)
	model := "claude-opus-4-7"
	explicitEndpoint := PlatformAnthropic
	contextEndpoint := "/v1/messages"
	ctx := WithSchedulerEndpoint(context.Background(), contextEndpoint)
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}

	firstTokenMs := 16
	svc.ReportAccountScheduleSuccessForRequest(ctx, accountID, model, explicitEndpoint, &firstTokenMs)

	if snap := svc.schedulerHealth.snapshot(accountID, model, explicitEndpoint, true); snap.CircuitState != schedulerCircuitOpen || snap.LastFailureReason != schedulerSlowTTFTCategory {
		t.Fatalf("explicit endpoint state=%s reason=%s want=%s/%s", snap.CircuitState, snap.LastFailureReason, schedulerCircuitOpen, schedulerSlowTTFTCategory)
	}
	if snap := svc.schedulerHealth.snapshot(accountID, model, contextEndpoint, true); snap.CircuitState != schedulerCircuitOpen || snap.LastFailureReason != schedulerSlowTTFTCategory {
		t.Fatalf("context endpoint state=%s reason=%s want=%s/%s", snap.CircuitState, snap.LastFailureReason, schedulerCircuitOpen, schedulerSlowTTFTCategory)
	}
}

func TestGatewayServiceReportAccountScheduleFailureStartsCircuitProbeForRecoverableError(t *testing.T) {
	oldRetryDelay := gatewayAccountCircuitProbeRetryDelay
	oldTimeout := gatewayAccountCircuitProbeTimeout
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	gatewayAccountCircuitProbeRetryDelay = 10 * time.Millisecond
	gatewayAccountCircuitProbeTimeout = time.Second
	schedulerHealthyTTFTThreshold = time.Second
	defer func() {
		gatewayAccountCircuitProbeRetryDelay = oldRetryDelay
		gatewayAccountCircuitProbeTimeout = oldTimeout
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	account := Account{
		ID:          38802,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "anthropic-key"},
	}
	upstream := &gatewayCircuitProbeUpstreamStub{callCh: make(chan int, 4)}
	svc := &GatewayService{
		accountRepo:     schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:             &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:    upstream,
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	defer svc.stopGatewayAccountCircuitProbe(account.ID, "claude-sonnet-4-5", "/v1/messages")

	svc.ReportAccountScheduleFailure(WithSchedulerEndpoint(context.Background(), "/v1/messages"), account.ID, "claude-sonnet-4-5", "/v1/messages", &UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"message":"rate limit"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected gateway circuit probe to start")
	}
	requireEventuallyGatewayCircuitClosed(t, svc, account.ID, "claude-sonnet-4-5", "/v1/messages")
}

func TestGatewayServiceReportAccountScheduleFailureDoesNotProbeBusinessForbidden(t *testing.T) {
	account := Account{
		ID:          38803,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "anthropic-key"},
	}
	upstream := &gatewayCircuitProbeUpstreamStub{callCh: make(chan int, 4)}
	svc := &GatewayService{
		accountRepo:     schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:             &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:    upstream,
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	defer svc.stopGatewayAccountCircuitProbe(account.ID, "claude-sonnet-4-5", "/v1/messages")

	svc.ReportAccountScheduleFailure(context.Background(), account.ID, "claude-sonnet-4-5", "/v1/messages", &UpstreamFailoverError{
		StatusCode:   http.StatusForbidden,
		ResponseBody: []byte(`{"error":{"message":"forbidden"}}`),
	})

	select {
	case <-upstream.callCh:
		t.Fatal("forbidden business/account errors should not start recovery probes")
	case <-time.After(50 * time.Millisecond):
	}
	if upstream.callCount() != 0 {
		t.Fatalf("probe calls=%d want=0", upstream.callCount())
	}
}

func TestGatewayServiceReportAccountScheduleSuccessStopsCircuitProbe(t *testing.T) {
	accountID := int64(38804)
	model := "claude-sonnet-4-5"
	endpoint := "/v1/messages"
	svc := &GatewayService{schedulerHealth: newAccountSchedulerHealthStats()}
	ctx, cancel := context.WithCancel(context.Background())
	svc.gatewayAccountCircuitProbes.Store(makeAccountSchedulerHealthKey(accountID, model, endpoint), &gatewayAccountCircuitProbe{cancel: cancel})
	svc.schedulerHealth.reportFailure(accountID, model, endpoint, "rate_limit", time.Minute)

	firstTokenMs := 10
	svc.ReportAccountScheduleSuccess(accountID, model, endpoint, &firstTokenMs)

	if _, ok := svc.gatewayAccountCircuitProbes.Load(makeAccountSchedulerHealthKey(accountID, model, endpoint)); ok {
		t.Fatal("expected success to stop gateway circuit probe")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("expected probe context to be canceled")
	}
	if snap := svc.schedulerHealth.snapshot(accountID, model, endpoint, false); snap.CircuitState != schedulerCircuitClosed {
		t.Fatalf("circuit=%s want=%s", snap.CircuitState, schedulerCircuitClosed)
	}
}

func TestGatewayServiceGeminiOAuthAIStudioCircuitProbe(t *testing.T) {
	account := Account{
		ID:          38805,
		Platform:    PlatformGemini,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "gemini-token"},
	}
	upstream := &gatewayCircuitProbeUpstreamStub{callCh: make(chan int, 4)}
	svc := &GatewayService{
		accountRepo:         schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:                 &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:        upstream,
		schedulerHealth:     newAccountSchedulerHealthStats(),
		geminiTokenProvider: NewGeminiTokenProvider(schedulerTestOpenAIAccountRepo{accounts: []Account{account}}, nil, nil),
	}
	defer svc.stopGatewayAccountCircuitProbe(account.ID, "gemini-3.1-flash", PlatformGemini)

	svc.ReportAccountScheduleFailure(context.Background(), account.ID, "gemini-3.1-flash", PlatformGemini, &UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"message":"rate limit"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected gemini oauth probe to start")
	}
	req, body := upstream.lastRequest()
	if req == nil {
		t.Fatal("expected probe request")
	}
	if !strings.Contains(req.URL.String(), "generativelanguage.googleapis.com/v1beta/models/gemini-3.1-flash:streamGenerateContent?alt=sse") {
		t.Fatalf("unexpected URL: %s", req.URL.String())
	}
	if got := req.Header.Get("Authorization"); got != "Bearer gemini-token" {
		t.Fatalf("authorization=%q want bearer token", got)
	}
	if !strings.Contains(body, "Reply with 1.") {
		t.Fatalf("unexpected body: %s", body)
	}
	requireEventuallyGatewayCircuitClosed(t, svc, account.ID, "gemini-3.1-flash", PlatformGemini)
}

func TestGatewayServiceGeminiOAuthCodeAssistCircuitProbe(t *testing.T) {
	account := Account{
		ID:          38806,
		Platform:    PlatformGemini,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "gemini-token",
			"project_id":   "project-123",
		},
	}
	upstream := &gatewayCircuitProbeUpstreamStub{callCh: make(chan int, 4)}
	svc := &GatewayService{
		accountRepo:         schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:                 &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:        upstream,
		schedulerHealth:     newAccountSchedulerHealthStats(),
		geminiTokenProvider: NewGeminiTokenProvider(schedulerTestOpenAIAccountRepo{accounts: []Account{account}}, nil, nil),
	}
	defer svc.stopGatewayAccountCircuitProbe(account.ID, "gemini-3.1-flash", PlatformGemini)

	svc.ReportAccountScheduleFailure(context.Background(), account.ID, "gemini-3.1-flash", PlatformGemini, &UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"message":"rate limit"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected gemini code assist probe to start")
	}
	req, body := upstream.lastRequest()
	if req == nil {
		t.Fatal("expected probe request")
	}
	if !strings.Contains(req.URL.String(), "cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse") {
		t.Fatalf("unexpected URL: %s", req.URL.String())
	}
	if got := req.Header.Get("Authorization"); got != "Bearer gemini-token" {
		t.Fatalf("authorization=%q want bearer token", got)
	}
	if got := req.Header.Get("User-Agent"); got == "" {
		t.Fatal("expected gemini cli user-agent")
	}
	if !strings.Contains(body, `"project":"project-123"`) || !strings.Contains(body, `"model":"gemini-3.1-flash"`) {
		t.Fatalf("unexpected code assist body: %s", body)
	}
	requireEventuallyGatewayCircuitClosed(t, svc, account.ID, "gemini-3.1-flash", PlatformGemini)
}

func TestGatewayServiceGeminiServiceAccountCircuitProbe(t *testing.T) {
	account := Account{
		ID:          38807,
		Platform:    PlatformGemini,
		Type:        AccountTypeServiceAccount,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"project_id":           "vertex-project",
			"location":             "global",
			"service_account_json": `{"project_id":"vertex-project","client_email":"probe@example.iam.gserviceaccount.com","private_key":"dummy"}`,
		},
	}
	upstream := &gatewayCircuitProbeUpstreamStub{callCh: make(chan int, 4)}
	svc := &GatewayService{
		accountRepo:         schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:                 &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:        upstream,
		schedulerHealth:     newAccountSchedulerHealthStats(),
		geminiTokenProvider: NewGeminiTokenProvider(schedulerTestOpenAIAccountRepo{accounts: []Account{account}}, &gatewayCircuitProbeTokenCache{token: "vertex-token"}, nil),
	}
	defer svc.stopGatewayAccountCircuitProbe(account.ID, "gemini-3.1-flash", PlatformGemini)

	svc.ReportAccountScheduleFailure(context.Background(), account.ID, "gemini-3.1-flash", PlatformGemini, &UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"message":"rate limit"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected gemini vertex probe to start")
	}
	req, _ := upstream.lastRequest()
	if req == nil {
		t.Fatal("expected probe request")
	}
	if !strings.Contains(req.URL.String(), "aiplatform.googleapis.com/v1/projects/vertex-project/locations/global/publishers/google/models/gemini-3.1-flash:streamGenerateContent?alt=sse") {
		t.Fatalf("unexpected URL: %s", req.URL.String())
	}
	if got := req.Header.Get("Authorization"); got != "Bearer vertex-token" {
		t.Fatalf("authorization=%q want bearer token", got)
	}
	requireEventuallyGatewayCircuitClosed(t, svc, account.ID, "gemini-3.1-flash", PlatformGemini)
}

func TestGatewayServiceAntigravityCircuitProbe(t *testing.T) {
	account := Account{
		ID:          38808,
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "ag-token",
			"project_id":   "ag-project",
		},
	}
	upstream := &gatewayCircuitProbeUpstreamStub{callCh: make(chan int, 4)}
	repo := schedulerTestOpenAIAccountRepo{accounts: []Account{account}}
	svc := &GatewayService{
		accountRepo:              repo,
		cfg:                      &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:             upstream,
		schedulerHealth:          newAccountSchedulerHealthStats(),
		antigravityTokenProvider: NewAntigravityTokenProvider(repo, nil, nil),
	}
	defer svc.stopGatewayAccountCircuitProbe(account.ID, "gemini-3-flash", PlatformGemini)

	svc.ReportAccountScheduleFailure(context.Background(), account.ID, "gemini-3-flash", PlatformGemini, &UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"message":"rate limit"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected antigravity probe to start")
	}
	req, body := upstream.lastRequest()
	if req == nil {
		t.Fatal("expected probe request")
	}
	if !strings.Contains(req.URL.String(), "v1internal:streamGenerateContent?alt=sse") {
		t.Fatalf("unexpected URL: %s", req.URL.String())
	}
	if got := req.Header.Get("Authorization"); got != "Bearer ag-token" {
		t.Fatalf("authorization=%q want bearer token", got)
	}
	if !strings.Contains(body, `"project":"ag-project"`) || !strings.Contains(body, `"model":"gemini-3-flash"`) {
		t.Fatalf("unexpected antigravity body: %s", body)
	}
	requireEventuallyGatewayCircuitClosed(t, svc, account.ID, "gemini-3-flash", PlatformGemini)
}

type gatewayCircuitProbeTokenCache struct {
	token string
}

func (c *gatewayCircuitProbeTokenCache) GetAccessToken(context.Context, string) (string, error) {
	return c.token, nil
}

func (c *gatewayCircuitProbeTokenCache) SetAccessToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (c *gatewayCircuitProbeTokenCache) DeleteAccessToken(context.Context, string) error {
	return nil
}

func (c *gatewayCircuitProbeTokenCache) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return false, nil
}

func (c *gatewayCircuitProbeTokenCache) ReleaseRefreshLock(context.Context, string) error {
	return nil
}

func requireEventuallyGatewayCircuitClosed(t *testing.T, svc *GatewayService, accountID int64, model, endpoint string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snap := svc.schedulerHealth.snapshot(accountID, model, endpoint, false)
		if snap.CircuitState == schedulerCircuitClosed {
			return
		}
		time.Sleep(time.Millisecond)
	}
	snap := svc.schedulerHealth.snapshot(accountID, model, endpoint, false)
	t.Fatalf("circuit=%s reason=%s want=%s", snap.CircuitState, snap.LastFailureReason, schedulerCircuitClosed)
}
