package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRefreshUpstreamClearsManagedValuesWhenProbeFails(t *testing.T) {
	ctx := context.Background()
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"success":false,"message":"temporary upstream failure"}`))
			return
		}
		require.Equal(t, "Bearer management-token", r.Header.Get("Authorization"))
		require.Equal(t, "7", r.Header.Get("New-Api-User"))
		switch r.URL.Path {
		case "/api/user/self":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":7,"username":"admin","group":"default","quota":1500000,"used_quota":0}}`))
		case "/api/user/self/groups":
			_, _ = w.Write([]byte(`{"success":true,"data":{"vip":{"ratio":0.25,"desc":"VIP"}}}`))
		case "/api/token/search":
			require.Equal(t, "sk-runtime", r.URL.Query().Get("token"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"page":1,"page_size":100,"total":1,"items":[{"id":12,"user_id":7,"status":1,"name":"runtime-key","remain_quota":1000000,"used_quota":0,"unlimited_quota":false,"group":"vip"}]}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newUpstreamHandlerTestClient(t)
	initialMetadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		Protocols: []upstreamProtocolCapability{{Platform: service.PlatformOpenAI, Status: "ok", Models: []string{"gpt-test"}}},
	})
	require.NoError(t, err)
	upstream, err := client.Upstream.Create().
		SetName("managed-newapi").
		SetBaseURL(server.URL).
		SetKind(entupstream.KindNewapi).
		SetCredentials(map[string]any{
			upstreamCredentialManagementAccessToken: "management-token",
			upstreamCredentialManagementUserID:      "7",
		}).
		SetMetadata(initialMetadata).
		Save(ctx)
	require.NoError(t, err)
	account, err := client.Account.Create().
		SetName("runtime-openai").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetCredentials(map[string]any{"api_key": "sk-runtime", "model_mapping": map[string]any{"gpt-test": "gpt-test"}}).
		SetExtra(map[string]any{"keep": "unchanged"}).
		SetUpstreamID(upstream.ID).
		Save(ctx)
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client, panelClient: newUpstreamSub2APIStatusClient()}
	refreshed, err := handler.refreshUpstreamByID(ctx, upstream.ID, false)
	require.NoError(t, err)
	// Ent may normalize the location and strip monotonic clock data on reload;
	// compare the persisted instant instead of the time.Time representation.
	require.Equal(t, upstream.UpdatedAt.UnixNano(), refreshed.UpdatedAt.UnixNano())
	metadata, err := parseUpstreamProbeMetadata(refreshed.Metadata)
	require.NoError(t, err)
	require.Equal(t, "ok", metadata.ManagementStatus)
	require.NotNil(t, metadata.Wallet)
	require.NotNil(t, metadata.Wallet.Balance)
	require.Equal(t, 3.0, *metadata.Wallet.Balance)
	require.Len(t, metadata.Protocols, 1, "lightweight refresh must preserve full model capabilities")
	billing := metadata.AccountBilling[strconv.FormatInt(account.ID, 10)]
	require.Equal(t, "ok", billing.Status)
	require.False(t, billing.Stale)
	require.NotNil(t, billing.GroupEffectiveRateMultiplier)
	require.Equal(t, 0.25, *billing.GroupEffectiveRateMultiplier)
	require.NotNil(t, metadata.Refresh)
	require.Equal(t, "ok", metadata.Refresh.Status)

	storedAccount, err := client.Account.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"keep": "unchanged"}, storedAccount.Extra)
	require.Equal(t, "sk-runtime", storedAccount.Credentials["api_key"])

	fail.Store(true)
	refreshed, err = handler.refreshUpstreamByID(ctx, upstream.ID, false)
	require.NoError(t, err)
	metadata, err = parseUpstreamProbeMetadata(refreshed.Metadata)
	require.NoError(t, err)
	require.Nil(t, metadata.Wallet, "failed refresh must not retain the last verified wallet")
	require.Nil(t, metadata.Key, "failed refresh must not retain the last verified key")
	require.Empty(t, metadata.Groups, "failed refresh must not retain the last verified groups")
	billing = metadata.AccountBilling[strconv.FormatInt(account.ID, 10)]
	require.True(t, billing.Stale)
	require.Nil(t, billing.KeyRemaining, "failed refresh must not retain the last verified quota")
	require.Nil(t, billing.GroupDefaultRateMultiplier, "failed refresh must not retain the last verified default rate")
	require.Nil(t, billing.GroupEffectiveRateMultiplier, "failed refresh must not retain the last verified rate")
	require.NotNil(t, metadata.Refresh)
	require.True(t, metadata.Refresh.Stale)
	require.Greater(t, metadata.Refresh.FailureCount, 0)
	require.True(t, metadata.Refresh.NextRefreshAt.Before(metadata.Refresh.LastAttemptAt.Add(upstreamRefreshInterval+upstreamRefreshJitter(upstream.ID))))
	require.NotNil(t, refreshed.LastProbeError)
	require.Contains(t, *refreshed.LastProbeError, "management probe:")
	require.Contains(t, *refreshed.LastProbeError, "account #"+strconv.FormatInt(account.ID, 10))
	require.NotContains(t, *refreshed.LastProbeError, server.URL)
}

func TestUpstreamRefreshErrorSummaryKeepsManagementAndAccountReasons(t *testing.T) {
	summary := upstreamRefreshErrorSummary(
		"upstream login failed: turnstile verification failed",
		false,
		map[string]upstreamAccountBillingMetadata{
			"43": {AccountID: 43, Status: "request_failed", Message: "HTTP 429", Stale: true},
			"41": {AccountID: 41, Status: "key_not_found", Message: "key was not found", Stale: true},
			"42": {AccountID: 42, Status: "ok", Message: "group rate was unavailable", Stale: true},
		},
		3,
	)

	require.Contains(t, summary, "management probe: upstream login failed: turnstile verification failed")
	require.Contains(t, summary, "account #41 [key_not_found]: key was not found")
	require.Contains(t, summary, "account #42 [rate_unavailable]: group rate was unavailable")
	require.Contains(t, summary, "1 more bound account failure(s)")
}

func TestManagedAccountStatusUsesUpstreamMetadata(t *testing.T) {
	rate := 0.4
	wallet := 12.5
	now := time.Now().UTC()
	upstream := &dbent.Upstream{
		ID:      9,
		Name:    "managed",
		BaseURL: "https://upstream.example",
		Kind:    entupstream.KindNewapi,
		Metadata: mustUpstreamMetadataMap(t, upstreamProbeMetadata{
			ManagementStatus: "ok",
			Wallet:           &upstreamProbeWallet{Balance: &wallet, Unit: "USD"},
			AccountBilling: map[string]upstreamAccountBillingMetadata{
				"41": {
					AccountID:                    41,
					Status:                       "ok",
					FetchedAt:                    now,
					UpstreamGroupName:            "vip",
					GroupEffectiveRateMultiplier: &rate,
				},
			},
		}),
	}
	account := &dbent.Account{ID: 41, Name: "runtime", Platform: service.PlatformOpenAI}
	account.Edges.Upstream = upstream

	status := managedAccountStatus(account)
	require.Equal(t, "upstream_manager", status.ProbeSource)
	require.Equal(t, "ok", status.Status)
	require.False(t, status.Stale)
	require.NotNil(t, status.UserBalance)
	require.Equal(t, wallet, *status.UserBalance)
	require.NotNil(t, status.UpstreamGroupEffectiveRateMultiplier)
	require.Equal(t, rate, *status.UpstreamGroupEffectiveRateMultiplier)
}

func TestLightweightMetadataMergePreservesGroupModelCapabilities(t *testing.T) {
	rate := 0.25
	previous := upstreamProbeMetadata{
		Groups: []upstreamProbeGroup{{
			Name:     "vip",
			Platform: service.PlatformOpenAI,
			Models:   []string{"gpt-old"},
		}},
		Protocols: []upstreamProtocolCapability{{
			Platform: service.PlatformOpenAI,
			Status:   "ok",
			Models:   []string{"gpt-platform-wide"},
		}},
	}
	current := upstreamProbeMetadata{
		ManagementStatus: "ok",
		Groups: []upstreamProbeGroup{{
			Name:           "vip",
			RateMultiplier: &rate,
		}},
	}

	merged := mergeUpstreamManagementMetadata(previous, current, false)

	require.Len(t, merged.Groups, 1)
	require.Equal(t, service.PlatformOpenAI, merged.Groups[0].Platform)
	require.Equal(t, []string{"gpt-old"}, merged.Groups[0].Models, "group-specific models must win over a protocol-wide catalogue")
	require.Equal(t, rate, *merged.Groups[0].RateMultiplier)
}

func TestLightweightRefreshDoesNotMarkFailedProtocolsHealthy(t *testing.T) {
	status, message := lightweightRefreshStatus(upstreamProbeMetadata{
		Protocols: []upstreamProtocolCapability{{Platform: service.PlatformOpenAI, Status: "error", Message: "HTTP 403 group disabled"}},
	})
	require.Equal(t, entupstream.StatusDegraded, status)
	require.Equal(t, "openai: HTTP 403 group disabled", message)

	status, message = lightweightRefreshStatus(upstreamProbeMetadata{
		Protocols: []upstreamProtocolCapability{{Platform: service.PlatformOpenAI, Status: "ok"}},
	})
	require.Equal(t, entupstream.StatusHealthy, status)
	require.Empty(t, message)

	status, message = lightweightRefreshStatus(upstreamProbeMetadata{
		Protocols: []upstreamProtocolCapability{{Platform: service.PlatformOpenAI, Status: "missing_api_key"}},
	})
	require.Equal(t, entupstream.StatusHealthy, status)
	require.Empty(t, message)
}

func TestHasVerifiedUpstreamProtocol(t *testing.T) {
	require.True(t, hasVerifiedUpstreamProtocol([]upstreamProtocolCapability{
		{Platform: service.PlatformOpenAI, Status: "error"},
		{Platform: service.PlatformAnthropic, Status: "ok"},
	}))
	require.False(t, hasVerifiedUpstreamProtocol([]upstreamProtocolCapability{
		{Platform: service.PlatformOpenAI, Status: "missing_api_key"},
		{Platform: service.PlatformAnthropic, Status: "error"},
	}))
}

func TestPartialAccountBillingDoesNotApplyOldRateToNewGroup(t *testing.T) {
	now := time.Now().UTC()
	oldRemaining := 2.0
	oldRate := 0.25
	currentRemaining := 8.0
	previous := upstreamAccountBillingMetadata{
		AccountID:                    41,
		KeyRemaining:                 &oldRemaining,
		UpstreamGroupName:            "old-group",
		UpstreamGroupPlatform:        service.PlatformOpenAI,
		GroupEffectiveRateMultiplier: &oldRate,
	}
	status := UpstreamSub2APIAccountStatus{
		AccountID:             41,
		Status:                "ok",
		FetchedAt:             now,
		KeyRemaining:          &currentRemaining,
		UpstreamGroupName:     "new-group",
		UpstreamGroupPlatform: service.PlatformOpenAI,
	}

	billing, ok := upstreamAccountBillingFromStatus(&dbent.Upstream{}, status, previous)

	require.False(t, ok)
	require.True(t, billing.Stale)
	require.Equal(t, currentRemaining, *billing.KeyRemaining)
	require.Equal(t, "new-group", billing.UpstreamGroupName)
	require.Nil(t, billing.GroupEffectiveRateMultiplier)

	unlimitedStatus := status
	unlimitedStatus.UsageMode = "unlimited"
	unlimitedStatus.KeyRemaining = nil
	unlimitedBilling, ok := upstreamAccountBillingFromStatus(&dbent.Upstream{}, unlimitedStatus, previous)
	require.False(t, ok)
	require.Nil(t, unlimitedBilling.KeyRemaining)
}

func TestUpstreamRefreshDueAtPrioritizesNeverRefreshedItems(t *testing.T) {
	now := time.Now().UTC()
	earlierRetryAt := now.Add(-2 * time.Minute)
	laterRetryAt := now.Add(-time.Minute)
	earlierMetadata := mustUpstreamMetadataMap(t, upstreamProbeMetadata{
		Refresh: &upstreamRefreshMetadata{NextRefreshAt: earlierRetryAt},
	})
	laterMetadata := mustUpstreamMetadataMap(t, upstreamProbeMetadata{
		Refresh: &upstreamRefreshMetadata{NextRefreshAt: laterRetryAt},
	})

	require.True(t, upstreamRefreshDue(&dbent.Upstream{ID: 1}, now))
	require.True(t, upstreamRefreshDue(&dbent.Upstream{ID: 2, Metadata: laterMetadata}, now))
	require.True(t, upstreamRefreshDueAt(&dbent.Upstream{ID: 1}).IsZero())
	require.Equal(t, laterRetryAt, upstreamRefreshDueAt(&dbent.Upstream{ID: 2, Metadata: laterMetadata}))

	items := []*dbent.Upstream{
		{ID: 2, Metadata: laterMetadata},
		{ID: 4, Metadata: earlierMetadata},
		{ID: 3},
	}
	sortUpstreamsByRefreshDue(items)
	require.Equal(t, []int64{3, 4, 2}, []int64{items[0].ID, items[1].ID, items[2].ID})
}

func TestFailedAutoRefreshDoesNotPersistHistoricalDetectedKind(t *testing.T) {
	ctx := context.Background()
	client := newUpstreamHandlerTestClient(t)
	metadata, err := upstreamMetadataMap(upstreamProbeMetadata{
		DetectedKind: entupstream.KindNewapi.String(),
		Groups:       nil,
		Protocols:    nil,
		FetchedAt:    time.Now().UTC().Add(-time.Minute),
	})
	require.NoError(t, err)
	item, err := client.Upstream.Create().
		SetName("auto-with-stale-detection").
		SetBaseURL("https://upstream.example").
		SetKind(entupstream.KindAuto).
		SetMetadata(metadata).
		Save(ctx)
	require.NoError(t, err)

	handler := &UpstreamHandler{client: client, panelClient: newUpstreamSub2APIStatusClient()}
	refreshed, err := handler.refreshUpstreamByID(ctx, item.ID, false)
	require.NoError(t, err)
	require.Equal(t, entupstream.KindAuto, refreshed.Kind)

	stored, err := parseUpstreamProbeMetadata(refreshed.Metadata)
	require.NoError(t, err)
	require.Equal(t, entupstream.KindAuto.String(), stored.DetectedKind)
	require.NotNil(t, stored.Groups)
	require.Empty(t, stored.Groups)
	require.NotNil(t, stored.Protocols)
	require.Empty(t, stored.Protocols)
}

func TestManagedAccountStatusEndpointsRejectOversizedRequests(t *testing.T) {
	parts := make([]string, managedAccountStatusMaxIDs+1)
	for i := range parts {
		parts[i] = strconv.Itoa(i + 1)
	}
	joined := strings.Join(parts, ",")

	handler := &UpstreamHandler{}
	router := gin.New()
	router.GET("/account-status", handler.AccountStatuses)
	router.POST("/account-status/refresh", handler.RefreshAccountStatuses)

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/account-status?account_ids="+joined, nil)
	router.ServeHTTP(getRecorder, getRequest)
	require.Equal(t, http.StatusBadRequest, getRecorder.Code, getRecorder.Body.String())

	postRecorder := httptest.NewRecorder()
	postRequest := httptest.NewRequest(http.MethodPost, "/account-status/refresh", strings.NewReader(`{"account_ids":[`+joined+`]}`))
	postRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(postRecorder, postRequest)
	require.Equal(t, http.StatusBadRequest, postRecorder.Code, postRecorder.Body.String())

	invalidRecorder := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPost, "/account-status/refresh", strings.NewReader(`{"account_ids":[1,-2]}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(invalidRecorder, invalidRequest)
	require.Equal(t, http.StatusBadRequest, invalidRecorder.Code, invalidRecorder.Body.String())

	largeRecorder := httptest.NewRecorder()
	largeBody := `{"account_ids":[1],"padding":"` + strings.Repeat("x", managedAccountStatusMaxBody) + `"}`
	largeRequest := httptest.NewRequest(http.MethodPost, "/account-status/refresh", strings.NewReader(largeBody))
	largeRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(largeRecorder, largeRequest)
	require.Equal(t, http.StatusBadRequest, largeRecorder.Code, largeRecorder.Body.String())
}

func mustUpstreamMetadataMap(t *testing.T, metadata upstreamProbeMetadata) map[string]any {
	t.Helper()
	result, err := upstreamMetadataMap(metadata)
	require.NoError(t, err)
	return result
}
