package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type openAIAccountCircuitProbeUpstreamStub struct {
	mu          sync.Mutex
	statuses    []int
	delays      []time.Duration
	emptyStream []bool
	calls       int
	bodies      [][]byte
	callCh      chan int
	allowSecond chan struct{}
}

func (u *openAIAccountCircuitProbeUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	var body []byte
	if req != nil && req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	u.mu.Lock()
	u.calls++
	call := u.calls
	status := http.StatusOK
	if len(u.statuses) > 0 {
		status = u.statuses[0]
		u.statuses = u.statuses[1:]
	}
	delay := time.Duration(0)
	if len(u.delays) > 0 {
		delay = u.delays[0]
		u.delays = u.delays[1:]
	}
	emptyStream := false
	if len(u.emptyStream) > 0 {
		emptyStream = u.emptyStream[0]
		u.emptyStream = u.emptyStream[1:]
	}
	u.bodies = append(u.bodies, append([]byte(nil), body...))
	u.mu.Unlock()

	select {
	case u.callCh <- call:
	default:
	}
	if call == 2 && u.allowSecond != nil {
		<-u.allowSecond
	}
	if delay > 0 {
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
	}

	respBody := "event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"1\"}\n\n"
	contentType := "text/event-stream"
	if emptyStream {
		respBody = "data: [DONE]\n\n"
	}
	if status >= 400 {
		respBody = `{"error":{"message":"bad gateway"}}`
		contentType = "application/json"
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}, nil
}

func (u *openAIAccountCircuitProbeUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func (u *openAIAccountCircuitProbeUpstreamStub) callCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.calls
}

func TestOpenAIAccountCircuitProbe_RecoveryRequiresSuccessfulProbe(t *testing.T) {
	oldRetryDelay := openAIAccountCircuitProbeRetryDelay
	oldTimeout := openAIAccountCircuitProbeTimeout
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	openAIAccountCircuitProbeRetryDelay = 10 * time.Millisecond
	openAIAccountCircuitProbeTimeout = time.Second
	schedulerHealthyTTFTThreshold = time.Second
	defer func() {
		openAIAccountCircuitProbeRetryDelay = oldRetryDelay
		openAIAccountCircuitProbeTimeout = oldTimeout
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	account := Account{
		ID:          9001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    0,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	upstream := &openAIAccountCircuitProbeUpstreamStub{
		statuses:    []int{http.StatusBadGateway, http.StatusOK},
		callCh:      make(chan int, 4),
		allowSecond: make(chan struct{}),
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		httpUpstream:       upstream,
		schedulerHealth:    newAccountSchedulerHealthStats(),
	}
	defer svc.stopOpenAIAccountCircuitProbe(account.ID, "gpt-5.1", "/v1/responses")

	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.1", "/v1/responses", &UpstreamFailoverError{
		StatusCode:   http.StatusBadGateway,
		ResponseBody: []byte(`{"error":{"message":"bad gateway"}}`),
	})

	selection, _, err := svc.SelectAccountWithScheduler(context.Background(), nil, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.True(t, selection.WeakFallback)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}

	require.Eventually(t, func() bool {
		return upstream.callCount() >= 1
	}, time.Second, time.Millisecond)

	selection, _, err = svc.SelectAccountWithScheduler(context.Background(), nil, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.True(t, selection.WeakFallback, "failed probe keeps normal scheduling closed, but same-group weak fallback may still try the account")
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}

	close(upstream.allowSecond)
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.1", "/v1/responses", false)
		return snap.CircuitState == schedulerCircuitClosed
	}, time.Second, time.Millisecond)
	require.GreaterOrEqual(t, upstream.callCount(), 2)

	selection, _, err = svc.SelectAccountWithScheduler(context.Background(), nil, "", "", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.False(t, selection.WeakFallback)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestShouldStartOpenAIAccountCircuitProbe_StartsForRateLimit(t *testing.T) {
	require.True(t, shouldStartOpenAIAccountCircuitProbe("gpt-5.5", "/v1/responses", "rate_limit"))
	require.True(t, shouldStartOpenAIAccountCircuitProbe("gpt-5.4", "/v1/chat/completions", "rate_limit"))
	require.False(t, shouldStartOpenAIAccountCircuitProbe("gpt-image-2", "/v1/responses", "rate_limit"))
}

func TestOpenAIAccountCircuitProbe_FailedProbeWaitsRetryDelayBeforeNextAttempt(t *testing.T) {
	oldRetryDelay := openAIAccountCircuitProbeRetryDelay
	oldTimeout := openAIAccountCircuitProbeTimeout
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	openAIAccountCircuitProbeRetryDelay = 80 * time.Millisecond
	openAIAccountCircuitProbeTimeout = time.Second
	schedulerHealthyTTFTThreshold = time.Second
	defer func() {
		openAIAccountCircuitProbeRetryDelay = oldRetryDelay
		openAIAccountCircuitProbeTimeout = oldTimeout
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	account := Account{
		ID:          9003,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	upstream := &openAIAccountCircuitProbeUpstreamStub{
		statuses: []int{http.StatusTooManyRequests, http.StatusOK},
		callCh:   make(chan int, 4),
	}
	svc := &OpenAIGatewayService{
		accountRepo:     schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:             &config.Config{},
		httpUpstream:    upstream,
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	defer svc.stopOpenAIAccountCircuitProbe(account.ID, "gpt-5.5", "/v1/responses")

	started := time.Now()
	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.5", "/v1/responses", &UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		ResponseBody: []byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected first probe to start immediately")
	}
	require.Less(t, time.Since(started), 50*time.Millisecond)

	select {
	case <-upstream.callCh:
		t.Fatal("second probe ran before retry delay elapsed")
	case <-time.After(40 * time.Millisecond):
	}

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected second probe after retry delay")
	}
	require.GreaterOrEqual(t, time.Since(started), 70*time.Millisecond)
}

func TestOpenAIAccountCircuitProbe_StopsWhenAccountManuallyUnschedulable(t *testing.T) {
	oldRetryDelay := openAIAccountCircuitProbeRetryDelay
	oldTimeout := openAIAccountCircuitProbeTimeout
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	openAIAccountCircuitProbeRetryDelay = 10 * time.Millisecond
	openAIAccountCircuitProbeTimeout = time.Second
	schedulerHealthyTTFTThreshold = time.Second
	defer func() {
		openAIAccountCircuitProbeRetryDelay = oldRetryDelay
		openAIAccountCircuitProbeTimeout = oldTimeout
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	account := Account{
		ID:          9002,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: false,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	upstream := &openAIAccountCircuitProbeUpstreamStub{
		callCh: make(chan int, 4),
	}
	svc := &OpenAIGatewayService{
		accountRepo:     schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		httpUpstream:    upstream,
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	defer svc.stopOpenAIAccountCircuitProbe(account.ID, "gpt-5.4", "/v1/responses")

	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.4", "/v1/responses", newNetworkUpstreamFailoverError("openai_request_error: use of closed network connection"))

	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.4", "/v1/responses", false)
		_, probeRunning := svc.openaiAccountCircuitProbes.Load(makeAccountSchedulerHealthKey(account.ID, "gpt-5.4", "/v1/responses"))
		return snap.CircuitState == schedulerCircuitClosed && !probeRunning
	}, time.Second, time.Millisecond)
	require.Zero(t, upstream.callCount(), "manual unschedulable accounts must not be probed upstream")
}

func TestOpenAIAccountCircuitProbe_SlowTTFTDoesNotRecover(t *testing.T) {
	oldRetryDelay := openAIAccountCircuitProbeRetryDelay
	oldTimeout := openAIAccountCircuitProbeTimeout
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	openAIAccountCircuitProbeRetryDelay = 20 * time.Millisecond
	openAIAccountCircuitProbeTimeout = time.Second
	schedulerHealthyTTFTThreshold = 15 * time.Millisecond
	defer func() {
		openAIAccountCircuitProbeRetryDelay = oldRetryDelay
		openAIAccountCircuitProbeTimeout = oldTimeout
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	account := Account{
		ID:          9004,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	upstream := &openAIAccountCircuitProbeUpstreamStub{
		statuses: []int{http.StatusOK, http.StatusOK},
		delays:   []time.Duration{30 * time.Millisecond, 30 * time.Millisecond},
		callCh:   make(chan int, 4),
	}
	svc := &OpenAIGatewayService{
		accountRepo:     schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:             &config.Config{},
		httpUpstream:    upstream,
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	defer svc.stopOpenAIAccountCircuitProbe(account.ID, "gpt-5.5", "/v1/responses")

	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.5", "/v1/responses", &UpstreamFailoverError{
		StatusCode:   http.StatusGatewayTimeout,
		ResponseBody: []byte(`{"error":{"message":"timeout"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected first probe")
	}
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.5", "/v1/responses", false)
		return snap.CircuitState == schedulerCircuitOpen && snap.LastFailureReason == "transient_timeout"
	}, time.Second, time.Millisecond)

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected slow probe to retry")
	}
	snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.5", "/v1/responses", false)
	require.Equal(t, schedulerCircuitOpen, snap.CircuitState)
}

func TestOpenAIAccountCircuitProbe_EmptyStreamDoesNotStopProbe(t *testing.T) {
	oldRetryDelay := openAIAccountCircuitProbeRetryDelay
	oldTimeout := openAIAccountCircuitProbeTimeout
	oldHealthyTTFT := schedulerHealthyTTFTThreshold
	openAIAccountCircuitProbeRetryDelay = 20 * time.Millisecond
	openAIAccountCircuitProbeTimeout = time.Second
	schedulerHealthyTTFTThreshold = time.Second
	defer func() {
		openAIAccountCircuitProbeRetryDelay = oldRetryDelay
		openAIAccountCircuitProbeTimeout = oldTimeout
		schedulerHealthyTTFTThreshold = oldHealthyTTFT
	}()

	account := Account{
		ID:          9005,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	upstream := &openAIAccountCircuitProbeUpstreamStub{
		statuses:    []int{http.StatusOK, http.StatusOK},
		emptyStream: []bool{true, false},
		callCh:      make(chan int, 4),
	}
	svc := &OpenAIGatewayService{
		accountRepo:     schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
		cfg:             &config.Config{},
		httpUpstream:    upstream,
		schedulerHealth: newAccountSchedulerHealthStats(),
	}
	defer svc.stopOpenAIAccountCircuitProbe(account.ID, "gpt-5.5", "/v1/responses")

	svc.ReportOpenAIAccountScheduleFailure(account.ID, "gpt-5.5", "/v1/responses", &UpstreamFailoverError{
		StatusCode:   http.StatusGatewayTimeout,
		ResponseBody: []byte(`{"error":{"message":"timeout"}}`),
	})

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected first probe")
	}
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.5", "/v1/responses", false)
		return snap.CircuitState == schedulerCircuitOpen && snap.LastFailureReason == "transient_timeout"
	}, time.Second, time.Millisecond)

	select {
	case <-upstream.callCh:
	case <-time.After(time.Second):
		t.Fatal("expected retry after empty stream")
	}
	require.Eventually(t, func() bool {
		snap := svc.schedulerHealth.snapshot(account.ID, "gpt-5.5", "/v1/responses", false)
		return snap.CircuitState == schedulerCircuitClosed
	}, time.Second, time.Millisecond)
}
