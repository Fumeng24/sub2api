package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestProbeSub2APIAccountUsesKeysEndpoint(t *testing.T) {
	var loginCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			atomic.AddInt32(&loginCount, 1)
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			if body["email"] != "upstream@example.com" || body["password"] != "secret" {
				t.Fatalf("unexpected login body: %#v", body)
			}
			writeJSON(t, w, `{"code":0,"message":"success","data":{"access_token":"sub2api-token","expires_in":3600,"token_type":"Bearer"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/auth/me":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":{"id":7,"email":"upstream@example.com","balance":12.5}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/available":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":[{"id":5,"name":"vip","platform":"openai","rate_multiplier":1.1}]}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/rates":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":{"5":1.35}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/keys":
			requireSub2APIAuth(t, r)
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("unexpected keys page: %q", got)
			}
			if got := r.URL.Query().Get("page_size"); got != "100" {
				t.Fatalf("unexpected keys page_size: %q", got)
			}
			writeJSON(t, w, `{"code":0,"message":"success","data":{"items":[{"id":9,"key":"sk-sub2api","name":"matched-key","group_id":5}],"total":1,"page":1,"page_size":100,"pages":1}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/api-keys":
			t.Fatalf("unexpected legacy api key path request after /api/v1/keys succeeded")

		case r.Method == http.MethodGet && r.URL.Path == "/v1/usage":
			requireBearer(t, r, "sk-sub2api")
			writeJSON(t, w, `{"mode":"unrestricted","planName":"wallet","remaining":4.2,"balance":8.8,"unit":"USD"}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       41,
		Name:     "sub2api-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL + "/v1",
			"api_key":                   "sk-sub2api",
			"upstream_panel_type":       "sub2api",
			"upstream_sub2api_email":    "upstream@example.com",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" {
		t.Fatalf("expected ok status, got %s: %s", status.Status, status.Message)
	}
	if status.UpstreamKind != "sub2api" {
		t.Fatalf("expected sub2api upstream kind, got %q", status.UpstreamKind)
	}
	if status.UpstreamKeyID == nil || *status.UpstreamKeyID != 9 {
		t.Fatalf("unexpected upstream key id: %#v", status.UpstreamKeyID)
	}
	if status.UpstreamKeyName != "matched-key" {
		t.Fatalf("unexpected upstream key name: %q", status.UpstreamKeyName)
	}
	if status.UpstreamGroupID == nil || *status.UpstreamGroupID != 5 {
		t.Fatalf("unexpected upstream group id: %#v", status.UpstreamGroupID)
	}
	if status.UpstreamGroupName != "vip" {
		t.Fatalf("unexpected upstream group name: %q", status.UpstreamGroupName)
	}
	if status.UpstreamGroupDefaultRateMultiplier == nil || *status.UpstreamGroupDefaultRateMultiplier != 1.1 {
		t.Fatalf("unexpected default rate: %#v", status.UpstreamGroupDefaultRateMultiplier)
	}
	if status.UpstreamGroupEffectiveRateMultiplier == nil || *status.UpstreamGroupEffectiveRateMultiplier != 1.35 {
		t.Fatalf("unexpected effective rate: %#v", status.UpstreamGroupEffectiveRateMultiplier)
	}
	if status.UserBalance == nil || *status.UserBalance != 12.5 {
		t.Fatalf("unexpected user balance: %#v", status.UserBalance)
	}
	if status.KeyRemaining == nil || *status.KeyRemaining != 4.2 {
		t.Fatalf("unexpected key remaining: %#v", status.KeyRemaining)
	}
	if got := atomic.LoadInt32(&loginCount); got != 1 {
		t.Fatalf("expected one sub2api login, got %d", got)
	}
}

func TestFindAPIKeyFallsBackToLegacyAPIKeysEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			writeJSON(t, w, `{"code":0,"message":"success","data":{"access_token":"sub2api-token","expires_in":3600,"token_type":"Bearer"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/keys":
			requireSub2APIAuth(t, r)
			w.WriteHeader(http.StatusNotFound)
			writeJSON(t, w, `{"code":404,"message":"not found"}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/api-keys":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":{"items":[{"id":11,"key":"sk-legacy","name":"legacy-key"}],"total":1,"page":1,"page_size":100,"pages":1}}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	matchedKey, err := client.findAPIKey(t.Context(), server.URL, "upstream@example.com", "secret", "sk-legacy")
	if err != nil {
		t.Fatalf("findAPIKey returned error: %v", err)
	}
	if matchedKey == nil || matchedKey.ID != 11 {
		t.Fatalf("unexpected matched key: %#v", matchedKey)
	}
}

func TestUnwrapUpstreamEnvelopes(t *testing.T) {
	tests := []struct {
		name    string
		unwrap  func([]byte) ([]byte, error)
		body    string
		want    string
		wantErr string
	}{
		{
			name:   "sub2api numeric code zero",
			unwrap: unwrapUpstreamSub2APIEnvelope,
			body:   `{"code":0,"message":"success","data":{"access_token":"token"}}`,
			want:   `{"access_token":"token"}`,
		},
		{
			name:   "sub2api success field without code",
			unwrap: unwrapUpstreamSub2APIEnvelope,
			body:   `{"success":true,"message":"","data":[{"id":1}]}`,
			want:   `[{"id":1}]`,
		},
		{
			name:   "sub2api string code 200",
			unwrap: unwrapUpstreamSub2APIEnvelope,
			body:   `{"code":"200","message":"ok","data":{"id":1}}`,
			want:   `{"id":1}`,
		},
		{
			name:   "newapi numeric code zero",
			unwrap: unwrapUpstreamNewAPIEnvelope,
			body:   `{"code":0,"message":"","data":{"id":7}}`,
			want:   `{"id":7}`,
		},
		{
			name:   "newapi numeric code 200",
			unwrap: unwrapUpstreamNewAPIEnvelope,
			body:   `{"code":200,"message":"","data":{"id":7}}`,
			want:   `{"id":7}`,
		},
		{
			name:   "newapi boolean code true",
			unwrap: unwrapUpstreamNewAPIEnvelope,
			body:   `{"code":true,"message":"","data":{"id":7}}`,
			want:   `{"id":7}`,
		},
		{
			name:    "failure message",
			unwrap:  unwrapUpstreamSub2APIEnvelope,
			body:    `{"code":500,"message":"upstream failed"}`,
			wantErr: "upstream failed",
		},
		{
			name:   "raw payload",
			unwrap: unwrapUpstreamSub2APIEnvelope,
			body:   `{"mode":"unrestricted","remaining":2}`,
			want:   `{"mode":"unrestricted","remaining":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.unwrap([]byte(tt.body))
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !jsonEqual(got, []byte(tt.want)) {
				t.Fatalf("unexpected payload: got %s want %s", string(got), tt.want)
			}
		})
	}
}

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

func TestProbeNewAPIAccountSupportsCodeEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode login body: %v", err)
			}
			if body["username"] != "upstream-user" || body["password"] != "secret" {
				t.Fatalf("unexpected login body: %#v", body)
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			writeJSON(t, w, `{"code":0,"message":"","data":{"id":7,"username":"upstream-user","group":"default"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self":
			requireNewAPIAuth(t, r)
			writeJSON(t, w, `{"code":200,"message":"","data":{"id":7,"username":"upstream-user","group":"default","quota":2500000,"used_quota":500000}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/token/search":
			requireNewAPIAuth(t, r)
			if got := r.URL.Query().Get("token"); got != "sk-new-numeric" {
				t.Fatalf("unexpected token search query: %q", got)
			}
			writeJSON(t, w, `{"code":"0","message":"","data":{"page":1,"page_size":100,"total":1,"items":[{"id":22,"user_id":7,"status":1,"name":"numeric-key","remain_quota":1500000,"used_quota":250000,"unlimited_quota":false,"group":"pro"}]}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self/groups":
			requireNewAPIAuth(t, r)
			writeJSON(t, w, `{"code":true,"message":"","data":{"pro":{"ratio":"1.75","desc":"Pro"}}}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       43,
		Name:     "newapi-code-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL + "/v1",
			"api_key":                   "sk-new-numeric",
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
	if status.UpstreamKeyID == nil || *status.UpstreamKeyID != 22 {
		t.Fatalf("unexpected upstream key id: %#v", status.UpstreamKeyID)
	}
	if status.UpstreamKeyName != "numeric-key" {
		t.Fatalf("unexpected upstream key name: %q", status.UpstreamKeyName)
	}
	if status.UpstreamGroupName != "pro" {
		t.Fatalf("unexpected upstream group: %q", status.UpstreamGroupName)
	}
	if status.UpstreamGroupEffectiveRateMultiplier == nil || *status.UpstreamGroupEffectiveRateMultiplier != 1.75 {
		t.Fatalf("unexpected upstream rate: %#v", status.UpstreamGroupEffectiveRateMultiplier)
	}
	if status.UserBalance == nil || *status.UserBalance != 5 {
		t.Fatalf("unexpected user balance: %#v", status.UserBalance)
	}
	if status.KeyRemaining == nil || *status.KeyRemaining != 3 {
		t.Fatalf("unexpected key remaining: %#v", status.KeyRemaining)
	}
}

func requireSub2APIAuth(t *testing.T, r *http.Request) {
	t.Helper()
	requireBearer(t, r, "sub2api-token")
}

func requireBearer(t *testing.T, r *http.Request, token string) {
	t.Helper()
	if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer "+token {
		t.Fatalf("unexpected authorization header: %q", got)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	_, err := w.Write([]byte(body))
	if err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func jsonEqual(got, want []byte) bool {
	return bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want))
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
