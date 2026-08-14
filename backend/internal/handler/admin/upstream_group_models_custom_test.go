package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestProbeModelsSkipsNewAPIGroupCatalogueForExplicitBatch(t *testing.T) {
	var catalogueCalls atomic.Int32
	var actualCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "newapi-session", Path: "/"})
			writeJSON(t, w, `{"success":true,"data":{"id":7,"username":"admin"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/user/models":
			require.Equal(t, "7", r.Header.Get("New-Api-User"))
			require.Equal(t, "vip", r.URL.Query().Get("group"))
			catalogueCalls.Add(1)
			writeJSON(t, w, `{"success":true,"data":["gpt-vip","gpt-shared"]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamHandlerTestClient(t)
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("newapi-model-catalogue").
		SetBaseURL(server.URL).
		SetKind(entupstream.KindNewapi).
		SetCredentials(map[string]any{
			upstreamCredentialUsername: "admin",
			upstreamCredentialPassword: "secret",
			upstreamCredentialAPIKey:   "sk-vip",
		}).
		SetMetadata(metadata).
		Save(t.Context())
	require.NoError(t, err)

	testConfig := &config.Config{}
	testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{responseFn: func(req *http.Request, body []byte) (int, string) {
			if req.Method != http.MethodPost || req.URL.Path != "/v1/responses" {
				return http.StatusInternalServerError, `{"error":{"message":"unexpected probe request"}}`
			}
			actualCalls.Add(1)
			var payload struct {
				Model string `json:"model"`
			}
			if json.Unmarshal(body, &payload) != nil {
				return http.StatusBadRequest, `{"error":{"message":"invalid request"}}`
			}
			if payload.Model == "gpt-nope" {
				return http.StatusBadRequest, `{"error":{"message":"The requested model is not supported"}}`
			}
			return http.StatusOK, upstreamModelProbeSuccessBody(body)
		}},
		testConfig, nil,
	)
	handler := &UpstreamHandler{
		client: client, panelClient: newUpstreamSub2APIStatusClient(), accountTestService: accountTestService,
	}
	router := gin.New()
	router.POST("/upstreams/:id/models/probe", handler.ProbeModels)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/models/probe", upstream.ID), bytes.NewBufferString(`{"platform":"openai","group_name":"vip","models":["gpt-vip","gpt-nope"]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var envelope struct {
		Data upstreamModelsProbeResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, int32(0), catalogueCalls.Load())
	require.Equal(t, int32(2), actualCalls.Load())
	require.Empty(t, envelope.Data.AvailableModels)
	require.Equal(t, "partial", envelope.Data.Status)
	results := make(map[string]upstreamModelProbeResult, len(envelope.Data.Results))
	for _, result := range envelope.Data.Results {
		results[result.Model] = result
	}
	require.True(t, results["gpt-vip"].Success)
	require.Equal(t, "unsupported", results["gpt-nope"].Status)
}

func TestProbeUpstreamModelWithRetryRecoversAfterTransient429(t *testing.T) {
	var calls atomic.Int32
	testConfig := &config.Config{}
	testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{responseFn: func(_ *http.Request, body []byte) (int, string) {
			if calls.Add(1) == 1 {
				return http.StatusTooManyRequests, `{"error":{"message":"rate limit exceeded"}}`
			}
			return http.StatusOK, upstreamModelProbeSuccessBody(body)
		}},
		testConfig, nil,
	)
	handler := &UpstreamHandler{accountTestService: accountTestService}
	account := &service.Account{
		ID:       -1,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-probe",
			"base_url": "https://upstream.example",
		},
		Concurrency: 1,
	}

	result := handler.probeUpstreamModelWithRetry(t.Context(), account, "gpt-test")

	require.Equal(t, service.ModelCapabilityStatusOK, result.Status, result.Message)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, int32(2), calls.Load())
}

func TestUpstreamModelProbeRetryDelay(t *testing.T) {
	delay, retry := upstreamModelProbeRetryDelay(0, 1)
	require.True(t, retry)
	require.Equal(t, upstreamModelProbeRetryBaseDelay, delay)

	delay, retry = upstreamModelProbeRetryDelay(time.Second, 1)
	require.True(t, retry)
	require.Equal(t, time.Second, delay)

	delay, retry = upstreamModelProbeRetryDelay(3*time.Second, 1)
	require.False(t, retry)
	require.Zero(t, delay)
}

func TestProbeModelsFallsBackToUnverifiedLocalCandidatesWhenCatalogueFails(t *testing.T) {
	client := newUpstreamHandlerTestClient(t)
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("catalogue-unavailable").
		SetBaseURL("https://upstream.example").
		SetKind(entupstream.KindNewapi).
		SetMetadata(metadata).
		Save(t.Context())
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client, adminService: newStubAdminService()}
	result := handler.probeUpstreamModels(t.Context(), upstream, upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI}},
	}, upstreamModelsProbeRequest{
		Platform: service.PlatformOpenAI, GroupName: "vip",
	})

	require.True(t, result.Success)
	require.Equal(t, "candidates", result.Status)
	require.Equal(t, "local_candidates", result.Source)
	require.ElementsMatch(t, []string{"gpt-5.5", "gpt-5.4"}, result.AvailableModels)
	require.Empty(t, result.Results)
	require.Contains(t, result.Message, "actual model request")
}

func TestExplicitProbeWithGeneratedGroupKeySkipsModelsEndpointAndSendsChallenge(t *testing.T) {
	var createdKeys atomic.Int32
	var deletedKeys atomic.Int32
	var actualCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			writeJSON(t, w, `{"code":0,"data":{"access_token":"panel-token","expires_in":3600}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/api-keys":
			createdKeys.Add(1)
			writeJSON(t, w, `{"code":0,"data":{"id":42,"key":"sk-codefree-generated","name":"generated","group_id":9}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/api-keys/42":
			deletedKeys.Add(1)
			writeJSON(t, w, `{"code":0,"data":true}`)
		default:
			t.Fatalf("unexpected management request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamHandlerTestClient(t)
	upstream, err := client.Upstream.Create().
		SetName("generated-key-real-probe").
		SetBaseURL(server.URL).
		SetKind(entupstream.KindSub2api).
		SetCredentials(map[string]any{
			upstreamCredentialUsername: "admin@example.com",
			upstreamCredentialPassword: "secret",
		}).
		Save(t.Context())
	require.NoError(t, err)
	groupID := int64(9)
	metadata := upstreamProbeMetadata{
		DetectedKind: entupstream.KindSub2api.String(),
		Groups: []upstreamProbeGroup{{
			ID: &groupID, Name: "codefree", Platform: service.PlatformOpenAI,
		}},
	}
	testConfig := &config.Config{}
	testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{responseFn: func(req *http.Request, body []byte) (int, string) {
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, "/v1/responses", req.URL.Path)
			require.Equal(t, "gpt-codefree", gjson.GetBytes(body, "model").String())
			actualCalls.Add(1)
			return http.StatusOK, upstreamModelProbeSuccessBody(body)
		}},
		testConfig, nil,
	)
	handler := &UpstreamHandler{
		client: client, panelClient: newUpstreamSub2APIStatusClient(), accountTestService: accountTestService,
	}

	result := handler.probeUpstreamModels(t.Context(), upstream, metadata, upstreamModelsProbeRequest{
		Platform: service.PlatformOpenAI, GroupName: "codefree", Models: []string{"gpt-codefree"},
	})

	require.True(t, result.Success, result.Message)
	require.Equal(t, "ok", result.Status)
	require.Equal(t, "generated_group_key", result.Source)
	require.Equal(t, int32(1), actualCalls.Load())
	require.Equal(t, int32(1), createdKeys.Load())
	require.Equal(t, int32(0), deletedKeys.Load())
	stored, err := client.Upstream.Get(t.Context(), upstream.ID)
	require.NoError(t, err)
	require.Equal(t, "sk-codefree-generated", lookupStoredGeneratedKey(stored.Credentials, "codefree").APIKey)
}

func TestApplyNewAPIGroupModelCatalogueOverridesKeyScopedProtocolFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/user/models", r.URL.Path)
		require.Equal(t, "7", r.Header.Get("New-Api-User"))
		switch r.URL.Query().Get("group") {
		case "available":
			writeJSON(t, w, `{"success":true,"data":["gpt-a","gpt-b"]}`)
		case "empty":
			writeJSON(t, w, `{"success":true,"data":[]}`)
		default:
			t.Fatalf("unexpected group: %q", r.URL.Query().Get("group"))
		}
	}))
	defer server.Close()

	handler := &UpstreamHandler{panelClient: newUpstreamSub2APIStatusClient()}
	item := &dbent.Upstream{
		BaseURL: server.URL,
		Kind:    entupstream.KindNewapi,
		Credentials: map[string]any{
			upstreamCredentialManagementAccessToken: "management-token",
			upstreamCredentialManagementUserID:      "7",
		},
	}
	metadata := upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "available"}, {Name: "empty"}},
		Protocols: []upstreamProtocolCapability{{
			Platform: service.PlatformOpenAI,
			Status:   "error",
			Message:  "Upstream returned no supported models",
		}},
	}

	summary := handler.applyNewAPIGroupModelCatalogue(t.Context(), item, &metadata)
	applyNewAPIProtocolFromGroupCatalogue(&metadata, summary)

	require.Equal(t, upstreamNewAPIGroupCatalogueSummary{
		TotalGroups: 2, AvailableGroups: 1, EmptyGroups: 1, Models: []string{"gpt-a", "gpt-b"},
	}, summary)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, metadata.Groups[0].Models)
	require.Empty(t, metadata.Groups[1].Models)
	require.Equal(t, "ok", metadata.Protocols[0].Status)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, metadata.Protocols[0].Models)
	require.Empty(t, metadata.Protocols[0].Message)
}

func TestProbeModelsDoesNotMixSub2APIOtherGroupModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			writeJSON(t, w, `{"code":0,"data":{"access_token":"panel-token","expires_in":3600}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/channels/available":
			require.Equal(t, "Bearer panel-token", r.Header.Get("Authorization"))
			writeJSON(t, w, `{"code":0,"data":[{"platforms":[{"platform":"openai","groups":[{"id":1,"name":"vip","platform":"openai","supported_models":[{"name":"gpt-vip"}]},{"id":2,"name":"other","platform":"openai","supported_models":[{"name":"gpt-other"}]}],"supported_models":[{"name":"gpt-platform-wide"}]}]}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamHandlerTestClient(t)
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{Name: "vip", Platform: service.PlatformOpenAI}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("sub2api-model-catalogue").
		SetBaseURL(server.URL).
		SetKind(entupstream.KindSub2api).
		SetCredentials(map[string]any{
			upstreamCredentialUsername: "admin@example.com",
			upstreamCredentialPassword: "secret",
		}).
		SetMetadata(metadata).
		Save(t.Context())
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client, panelClient: newUpstreamSub2APIStatusClient()}
	router := gin.New()
	router.POST("/upstreams/:id/models/probe", handler.ProbeModels)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/upstreams/%d/models/probe", upstream.ID), bytes.NewBufferString(`{"platform":"openai","group_name":"vip","models":[]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var envelope struct {
		Data upstreamModelsProbeResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, []string{"gpt-vip"}, envelope.Data.AvailableModels)
}

func TestModelsForSub2APIGroupSupportsLegacySectionCatalogue(t *testing.T) {
	channels := []upstreamSub2APIAvailableChannel{
		{Platforms: []upstreamSub2APIAvailableChannelPlatform{{
			Platform: "openai",
			Groups: []upstreamSub2APIAvailableChannelGroup{
				{Name: "vip"},
				{Name: "standard"},
			},
			SupportedModels: []upstreamSub2APIAvailableChannelModel{{Name: "gpt-section"}},
		}}},
		{Platforms: []upstreamSub2APIAvailableChannelPlatform{{
			Platform:        "openai",
			Groups:          []upstreamSub2APIAvailableChannelGroup{{Name: "other"}},
			SupportedModels: []upstreamSub2APIAvailableChannelModel{{Name: "gpt-other-channel"}},
		}}},
	}

	require.Equal(t, []string{"gpt-section"}, modelsForSub2APIGroup(channels, "openai", "vip"))
	require.Equal(t, []string{"gpt-other-channel"}, modelsForSub2APIGroup(channels, "openai", "other"))
}

func TestApplySub2APIModelCatalogueKeepsGroupModelsSeparate(t *testing.T) {
	metadata := upstreamProbeMetadata{
		DetectedKind: entupstream.KindSub2api.String(),
		FetchedAt:    time.Now().UTC(),
		Groups: []upstreamProbeGroup{
			{Name: "vip", Platform: service.PlatformOpenAI},
			{Name: "other", Platform: service.PlatformOpenAI},
			{Name: "not-exposed", Platform: service.PlatformOpenAI},
		},
		Protocols: []upstreamProtocolCapability{{Platform: service.PlatformOpenAI, Status: "error", Message: "default key rejected"}},
	}
	channels := []upstreamSub2APIAvailableChannel{{Platforms: []upstreamSub2APIAvailableChannelPlatform{{
		Platform: "openai",
		Groups: []upstreamSub2APIAvailableChannelGroup{
			{Name: "vip", SupportedModels: []upstreamSub2APIAvailableChannelModel{{Name: "gpt-vip"}}},
			{Name: "other", SupportedModels: []upstreamSub2APIAvailableChannelModel{{Name: "gpt-other"}}},
		},
	}}}}

	applySub2APIModelCatalogue(&metadata, channels)
	attachProtocolModelsToGroups(&metadata)
	require.Equal(t, []string{"gpt-vip"}, metadata.Groups[0].Models)
	require.Equal(t, []string{"gpt-other"}, metadata.Groups[1].Models)
	require.Empty(t, metadata.Groups[2].Models)
	require.Equal(t, "ok", metadata.Protocols[0].Status)
	require.Equal(t, []string{"gpt-other", "gpt-vip"}, metadata.Protocols[0].Models)
}

func TestSafeUpstreamModelProbeErrorRedactsRequestOverrideKey(t *testing.T) {
	const standardSecret = "sk-request-override-secret"
	message := safeUpstreamModelProbeError(errors.New("upstream rejected "+standardSecret), nil, standardSecret)
	require.NotContains(t, message, standardSecret)

	const opaqueSecret = "temporary-catalogue-credential-123456"
	message = safeUpstreamModelProbeError(errors.New("upstream rejected "+opaqueSecret), nil, opaqueSecret)
	require.NotContains(t, message, opaqueSecret)
	require.Contains(t, message, "[REDACTED]")
}

func TestRedactUpstreamTextKeepsGeneratedGroupName(t *testing.T) {
	const groupName = "codefree"
	const generatedKey = "sk-generated-secret-value"
	item := &dbent.Upstream{
		Credentials: mergeStoredGeneratedKey(map[string]any{}, storedGeneratedUpstreamKey{
			APIKey: generatedKey, GroupName: groupName, Kind: entupstream.KindSub2api.String(), CreatedAt: time.Now().UTC(),
		}),
	}

	message := redactUpstreamText("group "+groupName+" rejected key "+generatedKey, item)

	require.Contains(t, message, "group "+groupName)
	require.NotContains(t, message, generatedKey)
	require.Contains(t, message, "***")
}

func TestProbeSub2APIGroupWithGeneratedKeyKeepsScopedKeyWhenCatalogueUnavailable(t *testing.T) {
	tests := []struct {
		name        string
		modelStatus int
		modelBody   string
		wantModels  []string
		wantErr     bool
	}{
		{name: "verified", modelStatus: http.StatusOK, modelBody: `{"data":[{"id":"gpt-codefree"}]}`, wantModels: []string{"gpt-codefree"}},
		{name: "empty catalogue", modelStatus: http.StatusOK, modelBody: `{"data":[]}`, wantErr: true},
		{name: "probe failed", modelStatus: http.StatusServiceUnavailable, modelBody: `{"error":{"message":"temporarily unavailable"}}`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var createdKeys atomic.Int32
			var deletedKeys atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
					writeJSON(t, w, `{"code":0,"data":{"access_token":"panel-token","expires_in":3600}}`)
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/api-keys":
					require.Equal(t, "Bearer panel-token", r.Header.Get("Authorization"))
					var payload struct {
						GroupID int64 `json:"group_id"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					require.Equal(t, int64(9), payload.GroupID)
					createdKeys.Add(1)
					writeJSON(t, w, `{"code":0,"data":{"id":42,"key":"sk-codefree-generated","name":"generated","group_id":9}}`)
				case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/api-keys/42":
					require.Equal(t, "Bearer panel-token", r.Header.Get("Authorization"))
					deletedKeys.Add(1)
					writeJSON(t, w, `{"code":0,"data":true}`)
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			client := newUpstreamHandlerTestClient(t)
			upstream, err := client.Upstream.Create().
				SetName("hanhegufei").
				SetBaseURL(server.URL).
				SetKind(entupstream.KindSub2api).
				SetCredentials(map[string]any{
					upstreamCredentialUsername: "admin@example.com",
					upstreamCredentialPassword: "secret",
				}).
				Save(t.Context())
			require.NoError(t, err)
			groupID := int64(9)
			metadata := upstreamProbeMetadata{
				DetectedKind:     entupstream.KindSub2api.String(),
				ManagementStatus: "ok",
				Groups: []upstreamProbeGroup{{
					ID: &groupID, Name: "codefree", Platform: service.PlatformOpenAI,
				}},
			}
			testConfig := &config.Config{}
			testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
			accountTestService := service.NewAccountTestService(
				nil, nil, nil, nil, nil,
				&upstreamModelProbeHTTPStub{status: test.modelStatus, body: test.modelBody},
				testConfig, nil,
			)
			handler := &UpstreamHandler{
				client:             client,
				panelClient:        newUpstreamSub2APIStatusClient(),
				accountTestService: accountTestService,
			}

			models, source, probeErr := handler.probeUpstreamGroupWithGeneratedKey(
				t.Context(), upstream, metadata, service.PlatformOpenAI, "codefree",
			)
			if test.wantErr {
				require.Error(t, probeErr)
				require.Empty(t, models)
				require.Empty(t, source)
			} else {
				require.NoError(t, probeErr)
				require.Equal(t, test.wantModels, models)
				require.Equal(t, "generated_group_key", source)
			}
			require.Equal(t, int32(1), createdKeys.Load())
			require.Equal(t, int32(0), deletedKeys.Load())

			stored, err := client.Upstream.Get(t.Context(), upstream.ID)
			require.NoError(t, err)
			generated := lookupStoredGeneratedKey(stored.Credentials, "codefree")
			require.Equal(t, "sk-codefree-generated", generated.APIKey)
			require.Equal(t, int64(42), generated.ID)
		})
	}
}

func TestRunUpstreamGeneratedKeyCleanupSurvivesParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	err := runUpstreamGeneratedKeyCleanup(parent, func(ctx context.Context) error {
		called = true
		require.NoError(t, ctx.Err())
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(upstreamGeneratedKeyCleanupTimeout), deadline, 2*time.Second)
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestGeneratedGroupKeyIsCleanedWhenLocalPersistenceFails(t *testing.T) {
	var deletedKeys atomic.Int32
	client := newUpstreamHandlerTestClient(t)
	var upstreamID int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login":
			writeJSON(t, w, `{"code":0,"data":{"access_token":"panel-token","expires_in":3600}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/api-keys":
			require.NoError(t, client.Upstream.DeleteOneID(upstreamID).Exec(context.Background()))
			writeJSON(t, w, `{"code":0,"data":{"id":42,"key":"sk-generated","name":"generated","group_id":9}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/api-keys/42":
			deletedKeys.Add(1)
			writeJSON(t, w, `{"code":0,"data":true}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	upstream, err := client.Upstream.Create().
		SetName("persistence-cancel").
		SetBaseURL(server.URL).
		SetKind(entupstream.KindSub2api).
		SetCredentials(map[string]any{
			upstreamCredentialUsername: "admin@example.com",
			upstreamCredentialPassword: "secret",
		}).
		Save(t.Context())
	require.NoError(t, err)
	upstreamID = upstream.ID
	groupID := int64(9)
	metadata := upstreamProbeMetadata{
		DetectedKind: entupstream.KindSub2api.String(),
		Groups: []upstreamProbeGroup{{
			ID: &groupID, Name: "codefree", Platform: service.PlatformOpenAI,
		}},
	}
	testConfig := &config.Config{}
	testConfig.Security.URLAllowlist.AllowInsecureHTTP = true
	accountTestService := service.NewAccountTestService(
		nil, nil, nil, nil, nil,
		&upstreamModelProbeHTTPStub{status: http.StatusOK, body: `{"data":[{"id":"gpt-codefree"}]}`},
		testConfig, nil,
	)
	handler := &UpstreamHandler{
		client: client, panelClient: newUpstreamSub2APIStatusClient(), accountTestService: accountTestService,
	}

	models, source, probeErr := handler.probeUpstreamGroupWithGeneratedKey(
		t.Context(), upstream, metadata, service.PlatformOpenAI, "codefree",
	)

	require.Error(t, probeErr)
	require.Empty(t, models)
	require.Empty(t, source)
	require.Equal(t, int32(1), deletedKeys.Load())
	_, err = client.Upstream.Get(t.Context(), upstream.ID)
	require.Error(t, err)
}
