package admin

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newUpstreamHandlerTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(10)
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	driver := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(driver)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func createUpstreamHandlerTestUpstream(t *testing.T, client *dbent.Client, name, baseURL string) *dbent.Upstream {
	t.Helper()
	item, err := client.Upstream.Create().
		SetName(name).
		SetBaseURL(baseURL).
		SetKind(entupstream.KindNewapi).
		Save(context.Background())
	require.NoError(t, err)
	return item
}

func TestBuildUpstreamViewDoesNotExposeCredentialValues(t *testing.T) {
	lastProbeError := "POST https://upstream.example/v1 returned api_key=sk-secret-value username=user@example.com"
	item := &dbent.Upstream{
		ID:      1,
		Name:    "primary",
		BaseURL: "https://upstream.example",
		Kind:    entupstream.KindNewapi,
		Status:  entupstream.StatusHealthy,
		Credentials: map[string]any{
			upstreamCredentialAPIKey:                "sk-secret-value",
			upstreamCredentialManagementAccessToken: "management-secret-value",
			upstreamCredentialManagementUserID:      "123456",
			upstreamCredentialUsername:              "user@example.com",
			upstreamCredentialPassword:              "password-secret-value",
		},
		LastProbeError: &lastProbeError,
		Metadata: map[string]any{
			"legacy_error": "https://upstream.example/v1?api_key=sk-secret-value",
			"nested": map[string]any{
				"username": "user@example.com",
				"message":  "management_access_token=management-secret-value",
			},
		},
	}

	encoded, err := json.Marshal(buildUpstreamView(item, true, 0))
	require.NoError(t, err)
	body := string(encoded)
	require.NotContains(t, body, "sk-secret-value")
	require.NotContains(t, body, "management-secret-value")
	require.NotContains(t, body, "123456")
	require.NotContains(t, body, "user@example.com")
	require.NotContains(t, body, "password-secret-value")
	require.Contains(t, body, "legacy_error")
	require.Contains(t, body, `"has_api_key":true`)
	require.Contains(t, body, `"has_management_access_token":true`)
}

func TestListUpstreamsIncludesLocalGroupsSortedByName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	upstream := createUpstreamHandlerTestUpstream(t, client, "primary", "https://upstream.example")
	first, err := client.Account.Create().
		SetName("first").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).
		Save(ctx)
	require.NoError(t, err)
	second, err := client.Account.Create().
		SetName("second").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Group.Create().SetName("Zulu").SetPlatform(service.PlatformOpenAI).AddAccounts(first).Save(ctx)
	require.NoError(t, err)
	_, err = client.Group.Create().SetName("alpha").SetPlatform(service.PlatformOpenAI).AddAccounts(first, second).Save(ctx)
	require.NoError(t, err)
	_, err = client.Group.Create().SetName("Beta").SetPlatform(service.PlatformOpenAI).AddAccounts(second).Save(ctx)
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client}
	router := gin.New()
	router.GET("/upstreams", handler.List)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/upstreams?page=1&page_size=20", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var envelope struct {
		Data struct {
			Items []upstreamView `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Len(t, envelope.Data.Items, 1)
	require.Equal(t, []upstreamLocalGroupSummary{
		{Name: "alpha", Platform: service.PlatformOpenAI, ID: envelope.Data.Items[0].LocalGroups[0].ID},
		{Name: "Beta", Platform: service.PlatformOpenAI, ID: envelope.Data.Items[0].LocalGroups[1].ID},
		{Name: "Zulu", Platform: service.PlatformOpenAI, ID: envelope.Data.Items[0].LocalGroups[2].ID},
	}, envelope.Data.Items[0].LocalGroups)
}

func TestAutoUpstreamProbeDetectsNewAPIFromPublicKeyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/usage/token/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-public" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"object":"token_usage","name":"public-key","total_available":250000,"unlimited_quota":false}`)
	}))
	defer server.Close()

	handler := &UpstreamHandler{panelClient: newUpstreamSub2APIStatusClient()}
	item := &dbent.Upstream{
		ID:      17,
		Name:    "auto-public",
		BaseURL: server.URL,
		Kind:    entupstream.KindAuto,
		Credentials: map[string]any{
			upstreamCredentialAPIKey: "sk-public",
		},
	}

	outcome := handler.probeUpstream(context.Background(), item)
	require.Equal(t, entupstream.KindNewapi.String(), outcome.metadata.DetectedKind)
	require.True(t, outcome.detectedKindVerified)
	require.Equal(t, "api_key", outcome.metadata.ProbeSource)
	require.NotNil(t, outcome.metadata.Key)
	require.Equal(t, "public-key", outcome.metadata.Key.Name)
	require.NotNil(t, outcome.metadata.Key.Remaining)
	require.Equal(t, 0.5, *outcome.metadata.Key.Remaining)
}

type upstreamModelProbeHTTPStub struct {
	status     int
	body       string
	onRequest  func()
	responseFn func(*http.Request, []byte) (int, string)
}

var upstreamModelProbeChallengePattern = regexp.MustCompile(`(\d+)\s*([+-])\s*(\d+)`)

func upstreamModelProbeSuccessBody(requestBody []byte) string {
	match := upstreamModelProbeChallengePattern.FindStringSubmatch(gjson.GetBytes(requestBody, "input").String())
	if len(match) != 4 {
		return `{"error":{"message":"missing arithmetic challenge"}}`
	}
	left, _ := strconv.Atoi(match[1])
	right, _ := strconv.Atoi(match[3])
	answer := left + right
	if match[2] == "-" {
		answer = left - right
	}
	return `{"id":"resp-1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":` + strconv.Quote(strconv.Itoa(answer)) + `}]}]}`
}

func (s *upstreamModelProbeHTTPStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return s.response(req), nil
}

func (s *upstreamModelProbeHTTPStub) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.response(req), nil
}

func (s *upstreamModelProbeHTTPStub) response(req *http.Request) *http.Response {
	if s.onRequest != nil {
		s.onRequest()
	}
	var requestBody []byte
	if req.Body != nil {
		requestBody, _ = io.ReadAll(req.Body)
	}
	status := s.status
	body := s.body
	if s.responseFn != nil {
		status, body = s.responseFn(req, requestBody)
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestTestModelSendsRealRequestWithoutChangingConfiguration(t *testing.T) {
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("model-test").
		SetBaseURL("https://upstream.example/v1").
		SetKind(entupstream.KindNewapi).
		SetCredentials(map[string]any{upstreamCredentialAPIKey: "sk-test"}).
		SetMetadata(metadata).
		Save(ctx)
	require.NoError(t, err)

	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{responseFn: func(_ *http.Request, body []byte) (int, string) {
			return http.StatusOK, upstreamModelProbeSuccessBody(body)
		}},
		&config.Config{}, nil,
	)
	handler := &UpstreamHandler{client: client, accountTestService: accountTestService}
	router := gin.New()
	router.POST("/upstreams/:id/model-test", handler.TestModel)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/model-test", upstream.ID), bytes.NewBufferString(`{"platform":"openai","group_name":"vip","model":"gpt-test"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var envelope struct {
		Data upstreamModelProbeResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.True(t, envelope.Data.Success)
	require.Equal(t, "ok", envelope.Data.Status)
	require.Equal(t, "gpt-test", envelope.Data.Model)

	stored, err := client.Upstream.Get(ctx, upstream.ID)
	require.NoError(t, err)
	require.Equal(t, metadata, stored.Metadata)
}

func TestBindAccountsOnlyChangesUpstreamRelation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	upstream := createUpstreamHandlerTestUpstream(t, client, "primary", "https://same.example")
	account, err := client.Account.Create().
		SetName("legacy-openai").
		SetNotes("keep-note").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetCredentials(map[string]any{"api_key": "sk-legacy", "model_mapping": map[string]any{"gpt-test": "gpt-test"}}).
		SetExtra(map[string]any{"keep": "value"}).
		SetConcurrency(9).
		SetPriority(17).
		SetRateMultiplier(0.42).
		SetStatus(service.StatusError).
		SetSchedulable(false).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("legacy-local-group").
		SetPlatform(service.PlatformOpenAI).
		AddAccounts(account).
		Save(ctx)
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client}
	router := gin.New()
	router.POST("/upstreams/:id/bind", handler.BindAccounts)
	payload := fmt.Sprintf(`{"account_ids":[%d],"allow_rebind":false}`, account.ID)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/bind", upstream.ID), bytes.NewBufferString(payload))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	bound, err := client.Account.Query().Where().WithGroups().Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, bound.UpstreamID)
	require.Equal(t, upstream.ID, *bound.UpstreamID)
	require.Equal(t, account.Name, bound.Name)
	require.Equal(t, account.Notes, bound.Notes)
	require.Equal(t, account.Platform, bound.Platform)
	require.Equal(t, account.Type, bound.Type)
	require.Equal(t, account.Credentials, bound.Credentials)
	require.Equal(t, account.Extra, bound.Extra)
	require.Equal(t, account.Concurrency, bound.Concurrency)
	require.Equal(t, account.Priority, bound.Priority)
	require.Equal(t, account.RateMultiplier, bound.RateMultiplier)
	require.Equal(t, account.Status, bound.Status)
	require.Equal(t, account.Schedulable, bound.Schedulable)
	require.Len(t, bound.Edges.Groups, 1)
	require.Equal(t, group.ID, bound.Edges.Groups[0].ID)
}

func TestUnbindAccountsDeletesByDefaultAndCanExplicitlyPreserve(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newUpstreamHandlerTestClient(t)
	upstream := createUpstreamHandlerTestUpstream(t, client, "primary", "https://same.example")
	deleteByDefault, err := client.Account.Create().
		SetName("delete-by-default").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).
		Save(t.Context())
	require.NoError(t, err)
	preserve, err := client.Account.Create().
		SetName("preserve-explicitly").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).
		Save(t.Context())
	require.NoError(t, err)

	adminService := newStubAdminService()
	handler := &UpstreamHandler{client: client, adminService: adminService}
	router := gin.New()
	router.POST("/upstreams/:id/unbind", handler.UnbindAccounts)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/unbind", upstream.ID), bytes.NewBufferString(fmt.Sprintf(`{"account_ids":[%d]}`, deleteByDefault.ID)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, []int64{deleteByDefault.ID}, adminService.deletedAccountIDs)
	require.Contains(t, recorder.Body.String(), `"deleted_account_ids"`)

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/unbind", upstream.ID), bytes.NewBufferString(fmt.Sprintf(`{"account_ids":[%d],"delete_accounts":false}`, preserve.ID)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	stored, err := client.Account.Get(t.Context(), preserve.ID)
	require.NoError(t, err)
	require.Nil(t, stored.UpstreamID)
}

func TestBindCandidatesOnlyListsUnboundAPIKeyAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	upstream := createUpstreamHandlerTestUpstream(t, client, "primary", "https://same.example")
	other := createUpstreamHandlerTestUpstream(t, client, "other", "https://same.example")

	unbound, err := client.Account.Create().
		SetName("unbound").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.Account.Create().
		SetName("already-current").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).
		SetUpstreamID(upstream.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.Account.Create().
		SetName("already-other").SetPlatform(service.PlatformAnthropic).SetType(service.AccountTypeAPIKey).
		SetUpstreamID(other.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.Account.Create().
		SetName("oauth-account").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeOAuth).
		Save(ctx)
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client}
	router := gin.New()
	router.GET("/upstreams/:id/bind-candidates", handler.ListBindCandidates)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/upstreams/%d/bind-candidates?page=1&page_size=100", upstream.ID), nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var envelope struct {
		Data struct {
			Items []upstreamAccountSummary `json:"items"`
			Total int                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 1, envelope.Data.Total)
	require.Len(t, envelope.Data.Items, 1)
	require.Equal(t, unbound.ID, envelope.Data.Items[0].ID)
}

func TestPreviewUpstreamAccountCarriesDetectedGroupRateAndModels(t *testing.T) {
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	rate := 0.25
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		ManagementStatus: "ok",
		Groups:           []upstreamProbeGroup{{Name: "vip", RateMultiplier: &rate}},
		Protocols: []upstreamProtocolCapability{{
			Platform: service.PlatformOpenAI,
			Status:   "ok",
			Models:   []string{"gpt-test"},
		}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("primary").
		SetBaseURL("https://upstream.example").
		SetKind(entupstream.KindNewapi).
		SetCredentials(map[string]any{upstreamCredentialAPIKey: "sk-upstream"}).
		SetMetadata(metadata).
		Save(ctx)
	require.NoError(t, err)
	upstream, err = client.Upstream.Query().Where().WithAccounts().Only(ctx)
	require.NoError(t, err)

	adminService := newStubAdminService()
	adminService.groups = []service.Group{{ID: 91, Name: "local-openai", Platform: service.PlatformOpenAI, Status: service.StatusActive}}
	handler := &UpstreamHandler{client: client, adminService: adminService}
	handler.rememberUpstreamModelVerification(upstream, service.PlatformOpenAI, "vip", "gpt-test", "sk-upstream")
	preview := handler.previewUpstreamAccounts(ctx, upstream, []upstreamAccountGenerationSpec{{
		Platform:          service.PlatformOpenAI,
		UpstreamGroupName: "vip",
		Models:            []string{"gpt-test"},
		LocalGroupIDs:     []int64{91},
		Concurrency:       4,
		Priority:          intPointer(21),
	}})

	require.True(t, preview.Valid)
	require.Equal(t, 1, preview.Creates)
	require.Len(t, preview.Items, 1)
	require.Equal(t, []string{"gpt-test"}, preview.Items[0].Models)
	require.Equal(t, []int64{91}, preview.Items[0].LocalGroupIDs)
	require.Equal(t, &rate, preview.Items[0].RateMultiplier)
	require.Equal(t, "stored_default_key", preview.Items[0].KeySource)
}

func TestPreviewUpstreamAccountRequiresMatchingRealRequestVerification(t *testing.T) {
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI}},
		Protocols: []upstreamProtocolCapability{{
			Platform: service.PlatformOpenAI, Status: "ok", Models: []string{"gpt-test"},
		}},
	})
	require.NoError(t, err)
	item := &dbent.Upstream{
		ID: 17, Name: "primary", BaseURL: "https://upstream.example", Kind: entupstream.KindNewapi,
		Credentials: map[string]any{upstreamCredentialAPIKey: "sk-current"}, Metadata: metadata,
	}
	adminService := newStubAdminService()
	adminService.groups = []service.Group{{ID: 91, Name: "local-openai", Platform: service.PlatformOpenAI, Status: service.StatusActive}}
	handler := &UpstreamHandler{adminService: adminService}
	spec := []upstreamAccountGenerationSpec{{
		Platform: service.PlatformOpenAI, UpstreamGroupName: "vip", Models: []string{"gpt-test"}, LocalGroupIDs: []int64{91},
	}}

	preview := handler.previewUpstreamAccounts(t.Context(), item, spec)
	require.False(t, preview.Valid)
	require.Contains(t, strings.Join(preview.Items[0].Errors, " "), "successful current-group request")

	handler.rememberUpstreamModelVerification(item, service.PlatformOpenAI, "vip", "gpt-test", "sk-another-key")
	preview = handler.previewUpstreamAccounts(t.Context(), item, spec)
	require.False(t, preview.Valid, "verification from another key must not be reusable")

	handler.rememberUpstreamModelVerification(item, service.PlatformOpenAI, "vip", "gpt-test", "sk-current")
	preview = handler.previewUpstreamAccounts(t.Context(), item, spec)
	require.True(t, preview.Valid)
}

func TestGenerateUpstreamAccountAutomaticallyBindsAndUsesDetectedRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	rate := 0.3
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		ManagementStatus: "ok",
		Groups:           []upstreamProbeGroup{{Name: "pro", RateMultiplier: &rate}},
		Protocols: []upstreamProtocolCapability{{
			Platform: service.PlatformAnthropic,
			Status:   "ok",
			Models:   []string{"claude-test"},
		}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("multi-protocol").
		SetBaseURL("https://upstream.example").
		SetKind(entupstream.KindNewapi).
		SetCredentials(map[string]any{upstreamCredentialAnthropicAPIKey: "sk-ant"}).
		SetMetadata(metadata).
		Save(ctx)
	require.NoError(t, err)

	adminService := newStubAdminService()
	adminService.groups = []service.Group{{ID: 92, Name: "local-claude", Platform: service.PlatformAnthropic, Status: service.StatusActive}}
	handler := &UpstreamHandler{client: client, adminService: adminService}
	handler.rememberUpstreamModelVerification(upstream, service.PlatformAnthropic, "pro", "claude-test", "sk-ant")
	router := gin.New()
	router.POST("/upstreams/:id/accounts/generate", handler.GenerateAccounts)
	body := `{"accounts":[{"platform":"anthropic","upstream_group_name":"pro","models":["claude-test"],"local_group_ids":[92],"concurrency":5,"priority":18}]}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/accounts/generate", upstream.ID), bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, adminService.createdAccounts, 1)

	created := adminService.createdAccounts[0]
	require.NotNil(t, created.UpstreamID)
	require.Equal(t, upstream.ID, *created.UpstreamID)
	require.Equal(t, service.PlatformAnthropic, created.Platform)
	require.Equal(t, service.AccountTypeAPIKey, created.Type)
	require.Equal(t, []int64{92}, created.GroupIDs)
	require.Equal(t, 5, created.Concurrency)
	require.Equal(t, 18, created.Priority)
	require.Equal(t, &rate, created.RateMultiplier)
	require.Equal(t, "https://upstream.example", created.Credentials["base_url"])
	require.Equal(t, "sk-ant", created.Credentials["api_key"])
	require.Equal(t, map[string]any{"claude-test": "claude-test"}, created.Credentials["model_mapping"])
}

func intPointer(value int) *int { return &value }
