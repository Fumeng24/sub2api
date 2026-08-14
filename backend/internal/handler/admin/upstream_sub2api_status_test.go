package admin

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

func TestProbeSub2APIAccountUsesAccountProxy(t *testing.T) {
	var proxyHits int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&proxyHits, 1)
		if got := r.URL.Host; got != "upstream.example.test" {
			t.Fatalf("expected absolute proxy request for upstream.example.test, got host %q url=%s", got, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			writeJSON(t, w, `{"code":0,"message":"success","data":{"access_token":"proxied-token","expires_in":3600,"token_type":"Bearer"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/auth/me":
			requireBearer(t, r, "proxied-token")
			writeJSON(t, w, `{"code":0,"message":"success","data":{"id":7,"email":"upstream@example.com","balance":9.5}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/available":
			requireBearer(t, r, "proxied-token")
			writeJSON(t, w, `{"code":0,"message":"success","data":[{"id":5,"name":"proxy-group","platform":"openai","rate_multiplier":0.5}]}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/rates":
			requireBearer(t, r, "proxied-token")
			writeJSON(t, w, `{"code":0,"message":"success","data":{"5":0.6}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/keys":
			requireBearer(t, r, "proxied-token")
			writeJSON(t, w, `{"code":0,"message":"success","data":{"items":[{"id":9,"key":"sk-proxied","name":"proxied-key","group_id":5}],"total":1,"page":1,"page_size":100,"pages":1}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/v1/usage":
			requireBearer(t, r, "sk-proxied")
			writeJSON(t, w, `{"mode":"unrestricted","planName":"wallet","remaining":4.2,"balance":8.8,"unit":"USD"}`)

		default:
			t.Fatalf("unexpected proxied request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	host, portText, err := net.SplitHostPort(proxyURL.Host)
	if err != nil {
		t.Fatalf("split proxy host: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse proxy port: %v", err)
	}

	client := newUpstreamSub2APIStatusClient()
	proxyID := int64(100)
	account := &service.Account{
		ID:       44,
		Name:     "proxied-sub2api-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		ProxyID:  &proxyID,
		Proxy: &service.Proxy{
			ID:       proxyID,
			Protocol: proxyURL.Scheme,
			Host:     host,
			Port:     port,
			Status:   service.StatusActive,
		},
		Credentials: map[string]any{
			"base_url":                  "http://upstream.example.test/v1",
			"api_key":                   "sk-proxied",
			"upstream_panel_type":       "sub2api",
			"upstream_sub2api_email":    "upstream@example.com",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" {
		t.Fatalf("expected ok status through proxy, got %s: %s", status.Status, status.Message)
	}
	if !status.ProxyUsed {
		t.Fatalf("expected status to record proxy usage")
	}
	if status.UpstreamGroupName != "proxy-group" {
		t.Fatalf("unexpected upstream group: %q", status.UpstreamGroupName)
	}
	if got := atomic.LoadInt32(&proxyHits); got == 0 {
		t.Fatalf("expected requests to hit account proxy")
	}
}

func TestProbeSub2APIAccountDetectsCloudflareBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Header().Set("cf-ray", "abc123-NRT")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html><title>Access denied</title><body>Cloudflare Error code: 1020</body></html>"))
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       45,
		Name:     "cf-blocked-sub2api-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL + "/v1",
			"api_key":                   "sk-cf",
			"upstream_panel_type":       "sub2api",
			"upstream_sub2api_email":    "upstream@example.com",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "cloudflare_blocked" {
		t.Fatalf("expected cloudflare_blocked status, got %s: %s", status.Status, status.Message)
	}
	if status.ProxyUsed {
		t.Fatalf("expected direct request status to record proxy_used=false")
	}
	if !strings.Contains(status.Message, "Cloudflare") || !strings.Contains(status.Message, "abc123-NRT") {
		t.Fatalf("expected Cloudflare message with ray id, got %q", status.Message)
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

func TestProbeNewAPIAccountEnrichesPublicTokenUsageWithPanelMetadata(t *testing.T) {
	var publicProbeCount int32
	var loginCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/usage/token/":
			atomic.AddInt32(&publicProbeCount, 1)
			requireBearer(t, r, "sk-direct")
			if r.Header.Get("New-Api-User") != "" {
				t.Fatalf("public key probe must not include New-Api-User header")
			}
			if _, err := r.Cookie("session"); err == nil {
				t.Fatalf("public key probe must not include a session cookie")
			}
			writeJSON(t, w, `{"code":true,"message":"ok","data":{"object":"token_usage","name":"direct-key","total_granted":2000000,"total_used":1000000,"total_available":1000000,"unlimited_quota":false,"model_limits_enabled":false,"expires_at":0}}`)

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
			writeJSON(t, w, `{"success":true,"message":"","data":{"id":7,"username":"upstream-user","group":"vip"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self":
			requireNewAPIAuth(t, r)
			writeJSON(t, w, `{"success":true,"message":"","data":{"id":7,"username":"upstream-user","group":"vip","quota":4000000,"used_quota":1000000}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/token/search":
			requireNewAPIAuth(t, r)
			if got := r.URL.Query().Get("token"); got != "sk-direct" {
				t.Fatalf("unexpected token search query: %q", got)
			}
			writeJSON(t, w, `{"success":true,"message":"","data":{"page":1,"page_size":100,"total":1,"items":[{"id":12,"user_id":7,"status":1,"name":"direct-key","remain_quota":1000000,"used_quota":250000,"unlimited_quota":false,"group":"vip"}]}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self/groups":
			requireNewAPIAuth(t, r)
			writeJSON(t, w, `{"success":true,"message":"","data":{"vip":{"ratio":1.25,"desc":"VIP"}}}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       42,
		Name:     "newapi-direct-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL + "/v1",
			"api_key":                   "sk-direct",
			"upstream_panel_type":       "auto",
			"upstream_sub2api_email":    "upstream-user",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" {
		t.Fatalf("expected ok status, got %s: %s", status.Status, status.Message)
	}
	if status.UpstreamKind != "newapi" || status.ProbeSource != "panel_login" {
		t.Fatalf("expected enriched New API status, got kind=%q source=%q", status.UpstreamKind, status.ProbeSource)
	}
	if status.UpstreamKeyName != "direct-key" {
		t.Fatalf("unexpected upstream key name: %q", status.UpstreamKeyName)
	}
	if status.KeyRemaining == nil || *status.KeyRemaining != 2 {
		t.Fatalf("unexpected key remaining: %#v", status.KeyRemaining)
	}
	if status.UsageMode != "key_quota" {
		t.Fatalf("unexpected usage mode: %q", status.UsageMode)
	}
	if status.UpstreamGroupEffectiveRateMultiplier == nil || *status.UpstreamGroupEffectiveRateMultiplier != 1.25 {
		t.Fatalf("unexpected enriched group rate: %#v", status.UpstreamGroupEffectiveRateMultiplier)
	}
	if status.UserBalance == nil || *status.UserBalance != 8 {
		t.Fatalf("unexpected enriched wallet balance: %#v", status.UserBalance)
	}
	if got := atomic.LoadInt32(&publicProbeCount); got != 1 {
		t.Fatalf("expected one public probe, got %d", got)
	}
	if got := atomic.LoadInt32(&loginCount); got != 1 {
		t.Fatalf("expected one metadata-enrichment login, got %d", got)
	}

	cached := client.ProbeAccount(t.Context(), account, false)
	if !cached.Cached {
		t.Fatalf("expected second probe to use status cache")
	}
	if got := atomic.LoadInt32(&publicProbeCount); got != 1 {
		t.Fatalf("expected cached probe to avoid another public request, got %d", got)
	}
}

func TestProbeNewAPIAccountKeepsUnlimitedKeyUsableWithNegativeAvailableQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/api/usage/token/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireBearer(t, r, "sk-unlimited")
		writeJSON(t, w, `{"code":true,"message":"ok","data":{"object":"token_usage","name":"unlimited-key","total_granted":0,"total_used":9000000,"total_available":-9000000,"unlimited_quota":true,"model_limits_enabled":false,"expires_at":0}}`)
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       47,
		Name:     "newapi-unlimited-key",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":            server.URL,
			"api_key":             "sk-unlimited",
			"upstream_panel_type": "newapi",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" || status.UsageMode != "unlimited" {
		t.Fatalf("expected unlimited key to remain usable, got status=%q mode=%q message=%q", status.Status, status.UsageMode, status.Message)
	}
	if status.KeyRemaining != nil {
		t.Fatalf("unlimited key must not expose negative available quota as remaining: %#v", status.KeyRemaining)
	}
	if status.ProbeSource != "api_key" {
		t.Fatalf("expected API-key-only source without panel credentials, got %q", status.ProbeSource)
	}
}

func TestProbeNewAPIAccountEnrichesUnlimitedKeyWithWalletAndGroupRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/usage/token/":
			requireBearer(t, r, "sk-unlimited-enriched")
			writeJSON(t, w, `{"success":true,"message":"","data":{"object":"token_usage","name":"erapikey","total_granted":0,"total_used":9000000,"total_available":-9000000,"unlimited_quota":true,"model_limits_enabled":false,"expires_at":0}}`)

		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			writeJSON(t, w, `{"success":true,"message":"","data":{"id":7,"username":"upstream-user","group":"pro"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self":
			requireNewAPIAuth(t, r)
			writeJSON(t, w, `{"success":true,"message":"","data":{"id":7,"username":"upstream-user","group":"pro","quota":3500000,"used_quota":500000}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/token/search":
			requireNewAPIAuth(t, r)
			if got := r.URL.Query().Get("token"); got != "sk-unlimited-enriched" {
				t.Fatalf("unexpected token search query: %q", got)
			}
			writeJSON(t, w, `{"success":true,"message":"","data":{"page":1,"page_size":100,"total":1,"items":[{"id":23,"user_id":7,"status":1,"name":"erapikey","remain_quota":0,"used_quota":9000000,"unlimited_quota":true,"group":"pro"}]}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/user/self/groups":
			requireNewAPIAuth(t, r)
			writeJSON(t, w, `{"success":true,"message":"","data":{"pro":{"ratio":0.6,"desc":"Pro"}}}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       48,
		Name:     "newapi-unlimited-enriched",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL,
			"api_key":                   "sk-unlimited-enriched",
			"upstream_panel_type":       "newapi",
			"upstream_sub2api_email":    "upstream-user",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" || status.ProbeSource != "panel_login" {
		t.Fatalf("expected enriched New API status, got status=%q source=%q message=%q", status.Status, status.ProbeSource, status.Message)
	}
	if status.UsageMode != "unlimited" || status.KeyRemaining != nil {
		t.Fatalf("unexpected unlimited key state: mode=%q remaining=%#v", status.UsageMode, status.KeyRemaining)
	}
	if status.UpstreamKeyName != "erapikey" {
		t.Fatalf("unexpected upstream key name: %q", status.UpstreamKeyName)
	}
	if status.UserBalance == nil || *status.UserBalance != 7 {
		t.Fatalf("unexpected wallet balance: %#v", status.UserBalance)
	}
	if status.UpstreamGroupName != "pro" {
		t.Fatalf("unexpected upstream group: %q", status.UpstreamGroupName)
	}
	if status.UpstreamGroupEffectiveRateMultiplier == nil || *status.UpstreamGroupEffectiveRateMultiplier != 0.6 {
		t.Fatalf("unexpected group rate: %#v", status.UpstreamGroupEffectiveRateMultiplier)
	}
}

func TestProbeAutoNewAPIUsesSub2APIMetadataFallback(t *testing.T) {
	var publicProbeCount int32
	var newAPILoginCount int32
	var sub2APILoginCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/usage/token/":
			atomic.AddInt32(&publicProbeCount, 1)
			requireBearer(t, r, "sk-hybrid")
			writeJSON(t, w, `{"success":true,"message":"","data":{"object":"token_usage","name":"hybrid-key","total_granted":0,"total_used":9000000,"total_available":-9000000,"unlimited_quota":true,"model_limits_enabled":false,"expires_at":0}}`)

		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			atomic.AddInt32(&newAPILoginCount, 1)
			w.WriteHeader(http.StatusTooManyRequests)
			writeJSON(t, w, `{"success":false,"message":"panel login throttled"}`)

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			atomic.AddInt32(&sub2APILoginCount, 1)
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode Sub2API login body: %v", err)
			}
			if body["email"] != "upstream-user" || body["password"] != "secret" {
				t.Fatalf("unexpected Sub2API login body: %#v", body)
			}
			writeJSON(t, w, `{"code":0,"message":"success","data":{"access_token":"sub2api-token","expires_in":3600,"token_type":"Bearer"}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/auth/me":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":{"id":7,"email":"upstream-user","balance":6}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/available":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":[{"id":5,"name":"hybrid","platform":"openai","rate_multiplier":0.5}]}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/rates":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":{"5":0.4}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/keys":
			requireSub2APIAuth(t, r)
			writeJSON(t, w, `{"code":0,"message":"success","data":{"items":[{"id":5,"key":"sk-hybrid","name":"hybrid-key","group_id":5}],"total":1,"page":1,"page_size":100,"pages":1}}`)

		case r.Method == http.MethodGet && r.URL.Path == "/v1/usage":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(t, w, `{"error":"not found"}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       49,
		Name:     "hybrid-newapi-upstream",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL,
			"api_key":                   "sk-hybrid",
			"upstream_panel_type":       "auto",
			"upstream_sub2api_email":    "upstream-user",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "ok" || status.UpstreamKind != "sub2api" || status.ProbeSource != "panel_login" {
		t.Fatalf("expected Sub2API metadata fallback, got status=%q kind=%q source=%q message=%q", status.Status, status.UpstreamKind, status.ProbeSource, status.Message)
	}
	if status.UsageMode != "unlimited" || status.KeyRemaining != nil {
		t.Fatalf("expected public unlimited-key semantics, got mode=%q remaining=%#v", status.UsageMode, status.KeyRemaining)
	}
	if status.UserBalance == nil || *status.UserBalance != 6 {
		t.Fatalf("unexpected wallet balance: %#v", status.UserBalance)
	}
	if status.UpstreamGroupEffectiveRateMultiplier == nil || *status.UpstreamGroupEffectiveRateMultiplier != 0.4 {
		t.Fatalf("unexpected group rate: %#v", status.UpstreamGroupEffectiveRateMultiplier)
	}
	if got := atomic.LoadInt32(&publicProbeCount); got != 1 {
		t.Fatalf("expected one public probe, got %d", got)
	}
	if got := atomic.LoadInt32(&newAPILoginCount); got != 1 {
		t.Fatalf("expected one New API metadata attempt, got %d", got)
	}
	if got := atomic.LoadInt32(&sub2APILoginCount); got != 1 {
		t.Fatalf("expected one Sub2API metadata fallback, got %d", got)
	}
}

func TestPersistUpstreamMetadataClearsUnavailableValues(t *testing.T) {
	adminSvc := newStubAdminService()
	handler := &AccountHandler{adminService: adminSvc}

	handler.persistUpstreamMetadata(t.Context(), []UpstreamSub2APIAccountStatus{{
		AccountID:    47,
		Status:       "ok",
		UpstreamKind: "newapi",
		ProbeSource:  "api_key",
		UsageMode:    "unlimited",
	}})
	if got := len(adminSvc.updatedExtra); got != 1 {
		t.Fatalf("public New API probe must clear unavailable metadata, got updates %#v", adminSvc.updatedExtra)
	}
	for _, key := range []string{
		"upstream_rate_cached",
		"upstream_rate_cached_at",
		"upstream_wallet_cached",
		"upstream_wallet_cached_at",
		"upstream_wallet_cached_unit",
	} {
		if value, ok := adminSvc.updatedExtra[0][key]; !ok || value != nil {
			t.Fatalf("expected %s to be cleared, got %#v", key, value)
		}
	}
	adminSvc.updatedExtra = nil

	rate := 0.15
	balance := 9.5
	handler.persistUpstreamMetadata(t.Context(), []UpstreamSub2APIAccountStatus{{
		AccountID:                            47,
		Status:                               "ok",
		UpstreamKind:                         "newapi",
		ProbeSource:                          "panel_login",
		BalanceUnit:                          "USD",
		UserBalance:                          &balance,
		UpstreamGroupEffectiveRateMultiplier: &rate,
	}})
	if got := len(adminSvc.updatedExtra); got != 1 {
		t.Fatalf("expected one verified metadata update, got %d", got)
	}
	updates := adminSvc.updatedExtra[0]
	if got, ok := updates["upstream_rate_cached"].(float64); !ok || got != rate {
		t.Fatalf("unexpected cached rate: %#v", updates["upstream_rate_cached"])
	}
	if got, ok := updates["upstream_wallet_cached"].(float64); !ok || got != balance {
		t.Fatalf("unexpected cached wallet: %#v", updates["upstream_wallet_cached"])
	}
	if updates["upstream_wallet_cached_unit"] != "USD" {
		t.Fatalf("unexpected cached wallet unit: %#v", updates["upstream_wallet_cached_unit"])
	}
	for _, key := range []string{"upstream_rate_cached_at", "upstream_wallet_cached_at"} {
		if got, ok := updates[key].(string); !ok || strings.TrimSpace(got) == "" {
			t.Fatalf("expected %s to be persisted, got %#v", key, updates[key])
		}
	}

	adminSvc.updatedExtra = nil
	handler.persistUpstreamMetadata(t.Context(), []UpstreamSub2APIAccountStatus{{
		AccountID: 47,
		Status:    "request_failed",
	}})
	if got := len(adminSvc.updatedExtra); got != 1 {
		t.Fatalf("failed probe must clear unavailable metadata, got %d updates", got)
	}
	if adminSvc.updatedExtra[0]["upstream_rate_cached"] != nil || adminSvc.updatedExtra[0]["upstream_wallet_cached"] != nil {
		t.Fatalf("failed probe retained stale metadata: %#v", adminSvc.updatedExtra[0])
	}
}

func TestProbeNewAPIWithAPIKeyAllowsLoginFallbackForUnsupportedSuccessPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet || r.URL.Path != "/api/usage/token/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireBearer(t, r, "sk-unsupported-payload")
		writeJSON(t, w, `{"code":true,"message":"ok","data":{"id":7}}`)
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	status, fallbackAllowed := client.probeNewAPIWithAPIKey(
		t.Context(),
		UpstreamSub2APIAccountStatus{UpstreamKind: "newapi"},
		server.URL,
		"sk-unsupported-payload",
	)
	if !fallbackAllowed {
		t.Fatal("expected unsupported successful payload to allow panel-login fallback")
	}
	if status.Status != "request_failed" || status.ProbeSource != "api_key" {
		t.Fatalf("unexpected public probe status: %#v", status)
	}
}

func TestProbeNewAPIAccountFallsBackToLoginWhenPublicEndpointUnavailable(t *testing.T) {
	var loginCount int32
	var publicProbeCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/usage/token/":
			atomic.AddInt32(&publicProbeCount, 1)
			w.WriteHeader(http.StatusNotFound)
			writeJSON(t, w, `{"success":false,"message":"not found"}`)

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
	if got := atomic.LoadInt32(&publicProbeCount); got != 1 {
		t.Fatalf("expected one public probe before login fallback, got %d", got)
	}
}

func TestProbeNewAPIAccountSupportsCodeEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/usage/token/":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(t, w, `{"code":404,"message":"not found"}`)

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

func TestProbeNewAPIAccountDoesNotLoginWhenPublicEndpointRejectsKey(t *testing.T) {
	var loginCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/usage/token/":
			requireBearer(t, r, "sk-rejected")
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, `{"success":false,"message":"invalid token"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			atomic.AddInt32(&loginCount, 1)
			t.Fatalf("New API login must not hide a rejected API key")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamSub2APIStatusClient()
	account := &service.Account{
		ID:       46,
		Name:     "newapi-rejected-key",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url":                  server.URL,
			"api_key":                   "sk-rejected",
			"upstream_panel_type":       "auto",
			"upstream_sub2api_email":    "upstream-user",
			"upstream_sub2api_password": "secret",
		},
	}

	status := client.ProbeAccount(t.Context(), account, true)
	if status.Status != "request_failed" {
		t.Fatalf("expected rejected public-key status, got %s: %s", status.Status, status.Message)
	}
	if status.ProbeSource != "api_key" {
		t.Fatalf("expected public probe source, got %q", status.ProbeSource)
	}
	if got := atomic.LoadInt32(&loginCount); got != 0 {
		t.Fatalf("expected no login after API key rejection, got %d", got)
	}
}

func TestShouldProbeUpstreamSub2APIAccountAllowsExplicitNewAPIWithoutLogin(t *testing.T) {
	newAPIWithoutLogin := &service.Account{
		Type: service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":             "sk-direct",
			"upstream_panel_type": "newapi",
		},
	}
	if !shouldProbeUpstreamSub2APIAccount(newAPIWithoutLogin) {
		t.Fatal("explicit New API account without login should be eligible")
	}

	autoWithoutLogin := &service.Account{
		Type:        service.AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-unrelated"},
	}
	if shouldProbeUpstreamSub2APIAccount(autoWithoutLogin) {
		t.Fatal("unconfigured API-key account should not be probed")
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

func TestNormalizeUpstreamSub2APIBaseURLRejectsEmbeddedCredentialsAndSelectors(t *testing.T) {
	tests := []string{
		"https://user:password@upstream.example",
		"https://upstream.example?tenant=one",
		"https://upstream.example#fragment",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			_, err := normalizeUpstreamSub2APIBaseURL(raw)
			if err == nil {
				t.Fatalf("expected %q to be rejected", raw)
			}
		})
	}

	got, err := normalizeUpstreamSub2APIBaseURL("https://upstream.example/v1/")
	if err != nil {
		t.Fatalf("expected normal base URL to be accepted: %v", err)
	}
	if got != "https://upstream.example" {
		t.Fatalf("unexpected normalized base URL: %q", got)
	}
}
