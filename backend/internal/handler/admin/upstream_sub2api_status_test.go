package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestProbeNewAPIAccount(t *testing.T) {
	var loginCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			atomic.AddInt32(&loginCount, 1)
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			if body["username"] != "upstream-user" || body["password"] != "secret" {
				t.Fatalf("unexpected login body: %#v", body)
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"id":7,"username":"upstream-user","group":"default"}}`))

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self":
			requireNewAPIAuth(t, r)
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"id":7,"username":"upstream-user","group":"default","quota":1500000,"used_quota":500000}}`))

		case r.Method == http.MethodGet && r.URL.Path == "/api/token/search":
			requireNewAPIAuth(t, r)
			if got := r.URL.Query().Get("token"); got != "sk-test" {
				t.Fatalf("unexpected token search query: %q", got)
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"page":1,"page_size":100,"total":1,"items":[{"id":12,"user_id":7,"status":1,"name":"matched-key","remain_quota":1000000,"used_quota":250000,"unlimited_quota":false,"group":"vip"}]}}`))

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self/groups":
			requireNewAPIAuth(t, r)
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"vip":{"ratio":1.25,"desc":"VIP"}}}`))

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       42,
		Name:     "newapi-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL + "/v1",
			"api_key":                   "sk-test",
			"upstream_panel_type":       "newapi",
			"upstream_sub2api_email":    "upstream-user",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" {
		t.Fatalf("expected ok status, got %s: %s", status.Status, status.Message)
	}
	if status.UpstreamKind != "newapi" {
		t.Fatalf("expected newapi upstream kind, got %q", status.UpstreamKind)
	}
	if status.BaseURL != server.URL {
		t.Fatalf("expected normalized root %q, got %q", server.URL, status.BaseURL)
	}
	if status.UpstreamKeyID == nil || *status.UpstreamKeyID != 12 {
		t.Fatalf("unexpected upstream key id: %#v", status.UpstreamKeyID)
	}
	if status.UpstreamKeyName != "matched-key" {
		t.Fatalf("unexpected upstream key name: %q", status.UpstreamKeyName)
	}
	if status.UpstreamGroupName != "vip" {
		t.Fatalf("unexpected upstream group: %q", status.UpstreamGroupName)
	}
	if status.UpstreamGroupEffectiveRateMultiplier == nil || *status.UpstreamGroupEffectiveRateMultiplier != 1.25 {
		t.Fatalf("unexpected upstream rate: %#v", status.UpstreamGroupEffectiveRateMultiplier)
	}
	if status.UserBalance == nil || *status.UserBalance != 3 {
		t.Fatalf("unexpected user balance: %#v", status.UserBalance)
	}
	if status.KeyRemaining == nil || *status.KeyRemaining != 2 {
		t.Fatalf("unexpected key remaining: %#v", status.KeyRemaining)
	}

	cached := client.ProbeAccount(t.Context(), account, false)
	if !cached.Cached {
		t.Fatalf("expected second probe to use status cache")
	}
	if got := atomic.LoadInt32(&loginCount); got != 1 {
		t.Fatalf("expected one New API login, got %d", got)
	}
}

func requireNewAPIAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("New-Api-User"); got != "7" {
		t.Fatalf("missing New-Api-User header: %q", got)
	}
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value != "abc" {
		t.Fatalf("missing session cookie: %v %#v", err, cookie)
	}
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		t.Fatalf("New API user endpoints should not use bearer auth")
	}
}
