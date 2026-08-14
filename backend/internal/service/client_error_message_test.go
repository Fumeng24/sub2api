package service

import (
	"net/http"
	"strings"
	"testing"
)

func TestClientFacingErrorMessage_RedactsInternalDetails(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		errType string
		msg     string
		want    string
	}{
		{
			name:    "upstream forbidden with routing details",
			status:  http.StatusForbidden,
			errType: "forbidden_error",
			msg:     "unexpected status 403 Forbidden: Upstream access forbidden, please contact administrator, url: https://distribute.wegoo.site/responses, cf-ray: abc-NRT, request id: req_123",
			want:    clientFacingTemporaryUnavailableMessage,
		},
		{
			name:    "selection failure",
			status:  http.StatusServiceUnavailable,
			errType: "api_error",
			msg:     "No available accounts: no available accounts supporting model: gpt-5 (candidate_accounts=[1 2], excluded_account_count=2)",
			want:    clientFacingModelGroupUnavailableMessage,
		},
		{
			name:    "openai selection failure without model detail",
			status:  http.StatusServiceUnavailable,
			errType: "api_error",
			msg:     "no available OpenAI accounts",
			want:    clientFacingGroupUnavailableMessage,
		},
		{
			name:    "upstream model unsupported is account capacity",
			status:  http.StatusServiceUnavailable,
			errType: "api_error",
			msg:     "Requested model is not supported by upstream",
			want:    clientFacingModelGroupUnavailableMessage,
		},
		{
			name:    "endpoint unsupported",
			status:  http.StatusServiceUnavailable,
			errType: "endpoint_not_supported",
			msg:     "No accounts in this group support the requested endpoint",
			want:    "This API key does not support the requested endpoint",
		},
		{
			name:    "account concurrency detail",
			status:  http.StatusTooManyRequests,
			errType: "rate_limit_error",
			msg:     "Concurrency limit exceeded for account, please retry later",
			want:    "Service rate limit reached, please retry later",
		},
		{
			name:    "chinese account detail",
			status:  http.StatusBadGateway,
			errType: "upstream_error",
			msg:     "账号 123 上游异常，请联系管理员",
			want:    clientFacingTemporaryUnavailableMessage,
		},
		{
			name:    "upstream response too large",
			status:  http.StatusBadGateway,
			errType: "upstream_error",
			msg:     "Upstream response too large",
			want:    clientFacingTemporaryUnavailableMessage,
		},
		{
			name:    "cloudflare origin error page",
			status:  http.StatusBadGateway,
			errType: "upstream_error",
			msg:     "The origin web server returned an invalid or incomplete response to Cloudflare. This typically indicates the origin is overloaded or misconfigured.",
			want:    clientFacingTemporaryUnavailableMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClientFacingErrorMessage(tt.status, tt.errType, tt.msg)
			if got != tt.want {
				t.Fatalf("ClientFacingErrorMessage() = %q, want %q", got, tt.want)
			}
			lower := strings.ToLower(got)
			for _, leaked := range []string{"account", "https://", "cf-ray", "cloudflare", "request id", "candidate"} {
				if strings.Contains(lower, leaked) {
					t.Fatalf("sanitized message leaked %q: %q", leaked, got)
				}
			}
		})
	}
}

func TestClientFacingErrorMessage_PreservesActionableClientErrors(t *testing.T) {
	msg := "gpt-5.5 context window is 272k tokens. Your input exceeds this limit; please use gpt-5.4 to compress the context first, then retry with gpt-5.5."
	got := ClientFacingErrorMessage(http.StatusBadRequest, "invalid_request_error", msg)
	if got != msg {
		t.Fatalf("ClientFacingErrorMessage() = %q, want %q", got, msg)
	}
}

func TestClientFacingErrorMessage_RewritesContextWindowErrors(t *testing.T) {
	msg := "prompt is too long: 2318541 tokens > 1000000 maximum"
	got := ClientFacingErrorMessage(http.StatusBadRequest, "invalid_request_error", msg)
	if got != clientFacingContextWindowExceededMessage {
		t.Fatalf("ClientFacingErrorMessage() = %q, want %q", got, clientFacingContextWindowExceededMessage)
	}
}

func TestClientFacingErrorMessage_PreservesClaudeCodeVersionUpdate(t *testing.T) {
	msg := "Your Claude Code version (2.1.31) is below the minimum required version (2.1.85). Please update: npm update -g @anthropic-ai/claude-code"
	got := ClientFacingErrorMessage(http.StatusBadRequest, "invalid_request_error", msg)
	if got != msg {
		t.Fatalf("ClientFacingErrorMessage() = %q, want %q", got, msg)
	}
}

func TestClientFacingErrorMessage_RedactsCredentialLikeText(t *testing.T) {
	msg := "upstream rejected request api_key=sk-testsecret123456789, Authorization: Bearer ya29.a0AfH6SMDUMMYTOKEN, Cookie: sessionid=secret-cookie"
	got := ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", msg)
	for _, leaked := range []string{"sk-testsecret", "ya29.a0AfH6", "secret-cookie"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("sanitized message leaked %q: %q", leaked, got)
		}
	}
}

func TestClientFacingErrorBody_RedactsCommonJSONMessageFields(t *testing.T) {
	body := []byte(`{"error":{"type":"upstream_error","code":"bad_upstream","message":"unexpected status 403 Forbidden: Upstream access forbidden, url: https://distribute.wegoo.site/responses, cf-ray: abc, request id: req_123"},"status":"failed"}`)

	got := ClientFacingErrorBody(http.StatusForbidden, "upstream_error", body)
	gotText := string(got)
	if !strings.Contains(gotText, `"code":502`) {
		t.Fatalf("expected sensitive upstream code to be normalized, got %s", gotText)
	}
	if !strings.Contains(gotText, clientFacingTemporaryUnavailableMessage) {
		t.Fatalf("expected sanitized message, got %s", gotText)
	}
	for _, leaked := range []string{"distribute.wegoo.site", "cf-ray", "request id", "Upstream access forbidden"} {
		if strings.Contains(gotText, leaked) {
			t.Fatalf("sanitized body leaked %q: %s", leaked, gotText)
		}
	}
}

func TestClientFacingErrorBody_NormalizesSensitiveUpstreamFields(t *testing.T) {
	body := []byte(`{"error":{"type":"authentication_error","code":401,"message":"Incorrect API key provided: sk-live, request id: req_123","status":"UNAUTHENTICATED"}}`)

	got := ClientFacingErrorBody(http.StatusUnauthorized, "authentication_error", body)
	gotText := string(got)

	if !strings.Contains(gotText, `"type":"upstream_error"`) {
		t.Fatalf("expected normalized error type, got %s", gotText)
	}
	if !strings.Contains(gotText, `"code":502`) {
		t.Fatalf("expected normalized code, got %s", gotText)
	}
	if !strings.Contains(gotText, clientFacingTemporaryUnavailableMessage) {
		t.Fatalf("expected sanitized message, got %s", gotText)
	}
	for _, leaked := range []string{"authentication_error", "Incorrect API key", "sk-live", "request id", "UNAUTHENTICATED"} {
		if strings.Contains(gotText, leaked) {
			t.Fatalf("sanitized body leaked %q: %s", leaked, gotText)
		}
	}
}

func TestClientFacingErrorBody_RedactsPlainTextBodies(t *testing.T) {
	body := []byte(`unexpected status 403 Forbidden: Upstream access forbidden, url: https://distribute.wegoo.site/responses, cf-ray: abc, request id: req_123`)

	got := ClientFacingErrorBody(http.StatusForbidden, "upstream_error", body)
	gotText := string(got)
	if gotText != clientFacingTemporaryUnavailableMessage {
		t.Fatalf("expected sanitized fallback, got %q", gotText)
	}
	for _, leaked := range []string{"distribute.wegoo.site", "cf-ray", "request id", "Upstream access forbidden"} {
		if strings.Contains(gotText, leaked) {
			t.Fatalf("sanitized body leaked %q: %s", leaked, gotText)
		}
	}
}
