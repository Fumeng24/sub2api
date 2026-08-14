package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type managedNewAPIKeyFixture struct {
	mu      sync.Mutex
	token   upstreamNewAPIToken
	updates []map[string]any
}

func (fixture *managedNewAPIKeyFixture) handler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.Equal(t, "Bearer management-token", r.Header.Get("Authorization"))
		require.Equal(t, "7", r.Header.Get("New-Api-User"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/token/search":
			fixture.mu.Lock()
			token := fixture.token
			fixture.mu.Unlock()
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": upstreamNewAPITokensPage{
					Items: []upstreamNewAPIToken{token}, Total: 1, Page: 1, PageSize: 100,
				},
			}))
		case r.Method == http.MethodPut && r.URL.Path == "/api/token/":
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			fixture.mu.Lock()
			fixture.updates = append(fixture.updates, payload)
			fixture.token.Group, _ = payload["group"].(string)
			fixture.mu.Unlock()
			writeJSON(t, w, `{"success":true}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})
}

func (fixture *managedNewAPIKeyFixture) group() string {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.token.Group
}

func newManagedNewAPIUpstream(baseURL string) *dbent.Upstream {
	return &dbent.Upstream{
		ID:      17,
		Name:    "managed-newapi",
		BaseURL: baseURL,
		Kind:    entupstream.KindNewapi,
		Credentials: map[string]any{
			upstreamCredentialManagementAccessToken: "management-token",
			upstreamCredentialManagementUserID:      "7",
		},
	}
}

func TestChangeNewAPIAccountGroupPreservesTokenFieldsAndVerifiesRollback(t *testing.T) {
	allowIPs := "10.0.0.1"
	fixture := &managedNewAPIKeyFixture{token: upstreamNewAPIToken{
		ID: 91, Name: "runtime-key", Status: 1, ExpiredTime: 1234, RemainQuota: 5678,
		UnlimitedQuota: true, ModelLimitsEnabled: true, ModelLimits: "gpt-a,gpt-b",
		AllowIPs: &allowIPs, Group: "standard", CrossGroupRetry: true,
	}}
	server := httptest.NewServer(fixture.handler(t))
	defer server.Close()
	handler := &UpstreamHandler{panelClient: newUpstreamSub2APIStatusClient()}

	rollback, keyID, err := handler.changeNewAPIAccountGroup(t.Context(), newManagedNewAPIUpstream(server.URL), "sk-runtime", "vip")
	require.NoError(t, err)
	require.NotNil(t, rollback)
	require.Equal(t, int64(91), *keyID)
	require.Equal(t, "vip", fixture.group())
	require.Len(t, fixture.updates, 1)
	first := fixture.updates[0]
	require.Equal(t, "runtime-key", first["name"])
	require.Equal(t, float64(1), first["status"])
	require.Equal(t, float64(1234), first["expired_time"])
	require.Equal(t, float64(5678), first["remain_quota"])
	require.Equal(t, true, first["unlimited_quota"])
	require.Equal(t, true, first["model_limits_enabled"])
	require.Equal(t, "gpt-a,gpt-b", first["model_limits"])
	require.Equal(t, "10.0.0.1", first["allow_ips"])
	require.Equal(t, true, first["cross_group_retry"])

	require.NoError(t, rollback(t.Context()))
	require.Equal(t, "standard", fixture.group())
	require.Len(t, fixture.updates, 2)
}

func TestChangeSub2APIAccountGroupUsesDiscoveredPathAndVerifiesRollback(t *testing.T) {
	var mu sync.Mutex
	currentGroupID := int64(3)
	putPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			writeJSON(t, w, `{"code":0,"data":{"access_token":"panel-token","expires_in":3600}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/keys":
			w.WriteHeader(http.StatusNotFound)
			writeJSON(t, w, `{"code":404,"message":"not found"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/api-keys":
			require.Equal(t, "Bearer panel-token", r.Header.Get("Authorization"))
			mu.Lock()
			groupID := currentGroupID
			mu.Unlock()
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": upstreamSub2APIKeysPage{Items: []upstreamSub2APIKey{{
					ID: 42, Key: "sk-runtime", Name: "runtime", GroupID: &groupID,
				}}, Total: 1, Page: 1, PageSize: 100, Pages: 1},
			}))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/api-keys/42":
			require.Equal(t, "Bearer panel-token", r.Header.Get("Authorization"))
			var payload struct {
				GroupID int64 `json:"group_id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			mu.Lock()
			currentGroupID = payload.GroupID
			putPaths = append(putPaths, r.URL.Path)
			mu.Unlock()
			writeJSON(t, w, `{"code":0,"data":true}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	handler := &UpstreamHandler{panelClient: newUpstreamSub2APIStatusClient()}
	item := &dbent.Upstream{
		ID: 18, Name: "managed-sub2api", BaseURL: server.URL, Kind: entupstream.KindSub2api,
		Credentials: map[string]any{
			upstreamCredentialUsername: "admin@example.com",
			upstreamCredentialPassword: "secret",
		},
	}
	rollback, keyID, err := handler.changeSub2APIAccountGroup(t.Context(), item, "sk-runtime", 8)
	require.NoError(t, err)
	require.Equal(t, int64(42), *keyID)
	mu.Lock()
	require.Equal(t, int64(8), currentGroupID)
	mu.Unlock()
	require.NoError(t, rollback(t.Context()))
	mu.Lock()
	require.Equal(t, int64(3), currentGroupID)
	require.Equal(t, []string{"/api/v1/api-keys/42", "/api/v1/api-keys/42"}, putPaths)
	mu.Unlock()
}

func TestChangeAccountUpstreamGroupRollsBackWhenLocalUpdateFails(t *testing.T) {
	fixture := &managedNewAPIKeyFixture{token: upstreamNewAPIToken{ID: 91, Name: "runtime-key", Group: "standard"}}
	server := httptest.NewServer(fixture.handler(t))
	defer server.Close()

	client := newUpstreamHandlerTestClient(t)
	rate := 0.2
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		ManagementStatus: "ok",
		Groups:           []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI, RateMultiplier: &rate}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("managed-newapi").SetBaseURL(server.URL).SetKind(entupstream.KindNewapi).
		SetCredentials(newManagedNewAPIUpstream(server.URL).Credentials).SetMetadata(metadata).Save(t.Context())
	require.NoError(t, err)
	account, err := client.Account.Create().
		SetName("legacy-name").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).
		SetCredentials(map[string]any{"api_key": "sk-runtime", "model_mapping": map[string]any{"gpt-old": "gpt-old"}}).
		SetExtra(map[string]any{"upstream_group_name": "standard"}).SetUpstreamID(upstream.ID).Save(t.Context())
	require.NoError(t, err)

	adminService := newStubAdminService()
	adminService.updateAccountErr = errors.New("local update failed")
	testConfig := &config.Config{}
	testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{status: http.StatusOK, body: `{"data":[{"id":"gpt-vip"}]}`},
		testConfig, nil,
	)
	handler := &UpstreamHandler{
		client: client, adminService: adminService, accountTestService: accountTestService,
		panelClient: newUpstreamSub2APIStatusClient(),
	}
	router := gin.New()
	router.PUT("/upstreams/:id/accounts/:account_id/upstream-group", handler.ChangeAccountUpstreamGroup)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/upstreams/"+jsonNumber(upstream.ID)+"/accounts/"+jsonNumber(account.ID)+"/upstream-group", bytes.NewBufferString(`{"group_name":"vip"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	require.Equal(t, "standard", fixture.group())
	require.Equal(t, 1, adminService.updateAccountCalls)
	stored, err := client.Account.Get(t.Context(), account.ID)
	require.NoError(t, err)
	require.Equal(t, account.Name, stored.Name)
	require.Equal(t, account.Credentials, stored.Credentials)
	require.Equal(t, account.Extra, stored.Extra)
}

func TestChangeAccountUpstreamGroupRollsBackWhenModelVerificationFails(t *testing.T) {
	fixture := &managedNewAPIKeyFixture{token: upstreamNewAPIToken{ID: 91, Name: "runtime-key", Group: "standard"}}
	server := httptest.NewServer(fixture.handler(t))
	defer server.Close()

	client := newUpstreamHandlerTestClient(t)
	rate := 0.2
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		ManagementStatus: "ok",
		Groups:           []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI, RateMultiplier: &rate}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("managed-newapi").SetBaseURL(server.URL).SetKind(entupstream.KindNewapi).
		SetCredentials(newManagedNewAPIUpstream(server.URL).Credentials).SetMetadata(metadata).Save(t.Context())
	require.NoError(t, err)
	account, err := client.Account.Create().
		SetName("legacy-name").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).
		SetCredentials(map[string]any{"api_key": "sk-runtime", "model_mapping": map[string]any{"gpt-old": "gpt-old"}}).
		SetExtra(map[string]any{"upstream_group_name": "standard"}).SetUpstreamID(upstream.ID).Save(t.Context())
	require.NoError(t, err)

	adminService := newStubAdminService()
	testConfig := &config.Config{}
	testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{status: http.StatusOK, body: `{"data":[]}`},
		testConfig, nil,
	)
	handler := &UpstreamHandler{
		client: client, adminService: adminService, accountTestService: accountTestService,
		panelClient: newUpstreamSub2APIStatusClient(),
	}
	router := gin.New()
	router.PUT("/upstreams/:id/accounts/:account_id/upstream-group", handler.ChangeAccountUpstreamGroup)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/upstreams/"+jsonNumber(upstream.ID)+"/accounts/"+jsonNumber(account.ID)+"/upstream-group", bytes.NewBufferString(`{"group_name":"vip"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadGateway, recorder.Code, recorder.Body.String())
	require.Equal(t, "standard", fixture.group())
	require.Zero(t, adminService.updateAccountCalls)
	stored, err := client.Account.Get(t.Context(), account.ID)
	require.NoError(t, err)
	require.Equal(t, account.Name, stored.Name)
	require.Equal(t, account.Credentials, stored.Credentials)
	require.Equal(t, account.Extra, stored.Extra)
}

func TestRunUpstreamAccountRollbackSurvivesParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	err := runUpstreamAccountRollback(parent, func(ctx context.Context) error {
		called = true
		require.NoError(t, ctx.Err())
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(upstreamAccountRollbackTimeout), deadline, 2*time.Second)
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestResolveUpstreamAccountTargetGroupRequiresIDForAmbiguousNames(t *testing.T) {
	firstID := int64(1)
	secondID := int64(2)
	groups := []upstreamProbeGroup{
		{ID: &firstID, Name: "shared", Platform: service.PlatformOpenAI},
		{ID: &secondID, Name: "shared", Platform: service.PlatformOpenAI},
	}
	_, err := resolveUpstreamAccountTargetGroup(groups, service.PlatformOpenAI, "shared", nil)
	require.Error(t, err)
	resolved, err := resolveUpstreamAccountTargetGroup(groups, service.PlatformOpenAI, "shared", &secondID)
	require.NoError(t, err)
	require.Equal(t, secondID, *resolved.ID)
}

func TestRenamePreviewSkipsAccountsWithoutVerifiedGroup(t *testing.T) {
	client := newUpstreamHandlerTestClient(t)
	upstream, err := client.Upstream.Create().
		SetName("Primary").SetBaseURL("https://upstream.example").SetKind(entupstream.KindNewapi).
		Save(t.Context())
	require.NoError(t, err)
	verified, err := client.Account.Create().
		SetName("old verified name").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).Save(t.Context())
	require.NoError(t, err)
	unverified, err := client.Account.Create().
		SetName("keep this name").SetPlatform(service.PlatformAnthropic).SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).Save(t.Context())
	require.NoError(t, err)
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{AccountBilling: map[string]upstreamAccountBillingMetadata{
		jsonNumber(verified.ID): {
			AccountID: verified.ID, Status: "ok", UpstreamGroupName: "vip", UpstreamGroupPlatform: service.PlatformOpenAI,
		},
	}})
	require.NoError(t, err)
	_, err = client.Upstream.UpdateOneID(upstream.ID).SetMetadata(metadata).Save(t.Context())
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client}
	preview, err := handler.buildUpstreamAccountRenamePreview(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, preview.Renames)
	require.Equal(t, 1, preview.Skips)
	require.Len(t, preview.Items, 2)
	items := make(map[int64]upstreamAccountRenameItem, len(preview.Items))
	for _, item := range preview.Items {
		items[item.AccountID] = item
	}
	require.Equal(t, "Primary / vip / OPENAI", items[verified.ID].ProposedName)
	require.Equal(t, "rename", items[verified.ID].Action)
	require.Equal(t, "skip", items[unverified.ID].Action)
	require.Equal(t, "upstream group is not currently verified", items[unverified.ID].Reason)
}

func TestRenameAccountsApplyReturnsUnavailableWithoutAdminService(t *testing.T) {
	handler := &UpstreamHandler{}
	router := gin.New()
	router.POST("/upstreams/accounts/rename-apply", handler.RenameAccountsApply)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/upstreams/accounts/rename-apply", nil)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
}

func jsonNumber(value int64) string {
	data, _ := json.Marshal(value)
	return string(data)
}
