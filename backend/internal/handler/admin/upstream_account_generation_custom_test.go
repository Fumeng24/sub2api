package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type newAPIGeneratedKeyTestServer struct {
	t           *testing.T
	keyResponse func(http.ResponseWriter, int32)

	nameMu        sync.Mutex
	generatedName string
	createCalls   atomic.Int32
	keyCalls      atomic.Int32
	deleteCalls   atomic.Int32
}

func (s *newAPIGeneratedKeyTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.t.Helper()
	require.Equal(s.t, "Bearer management-token", r.Header.Get("Authorization"))
	require.Equal(s.t, "7", r.Header.Get("New-Api-User"))
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/token/":
		var payload struct {
			Name string `json:"name"`
		}
		require.NoError(s.t, json.NewDecoder(r.Body).Decode(&payload))
		require.NotEmpty(s.t, payload.Name)
		s.nameMu.Lock()
		s.generatedName = payload.Name
		s.nameMu.Unlock()
		s.createCalls.Add(1)
		writeNewAPIGeneratedKeyTestJSON(s.t, w, http.StatusOK, map[string]any{"success": true, "data": true})
	case r.Method == http.MethodGet && r.URL.Path == "/api/token/search":
		s.nameMu.Lock()
		name := s.generatedName
		s.nameMu.Unlock()
		require.Equal(s.t, name, r.URL.Query().Get("keyword"))
		writeNewAPIGeneratedKeyTestJSON(s.t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"items": []map[string]any{{"id": 42, "name": name, "group": "grok"}},
				"total": 1,
			},
		})
	case r.Method == http.MethodPost && r.URL.Path == "/api/token/42/key":
		attempt := s.keyCalls.Add(1)
		s.keyResponse(w, attempt)
	case r.Method == http.MethodDelete && r.URL.Path == "/api/token/42":
		s.deleteCalls.Add(1)
		writeNewAPIGeneratedKeyTestJSON(s.t, w, http.StatusOK, map[string]any{"success": true, "data": true})
	default:
		s.t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}
}

func writeNewAPIGeneratedKeyTestJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

func newAPIGeneratedKeyTestTarget(baseURL string) (*UpstreamHandler, *dbent.Upstream, normalizedUpstreamAccountGenerationSpec) {
	handler := &UpstreamHandler{panelClient: newUpstreamSub2APIStatusClient()}
	item := &dbent.Upstream{
		ID:      31,
		Name:    "mosshub",
		BaseURL: baseURL,
		Kind:    entupstream.KindNewapi,
		Credentials: map[string]any{
			upstreamCredentialManagementAccessToken: "management-token",
			upstreamCredentialManagementUserID:      "7",
		},
	}
	spec := normalizedUpstreamAccountGenerationSpec{upstreamAccountGenerationSpec: upstreamAccountGenerationSpec{
		Platform:          service.PlatformGrok,
		UpstreamGroupName: "grok",
	}}
	return handler, item, spec
}

func TestCreateNewAPIRemoteGroupKeyRetriesTransientRead429WithoutRecreating(t *testing.T) {
	upstream := &newAPIGeneratedKeyTestServer{t: t}
	upstream.keyResponse = func(w http.ResponseWriter, attempt int32) {
		if attempt == 1 {
			writeNewAPIGeneratedKeyTestJSON(t, w, http.StatusTooManyRequests, map[string]any{"success": false, "message": "rate limited"})
			return
		}
		writeNewAPIGeneratedKeyTestJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data":    map[string]any{"key": "sk-grok-generated"},
		})
	}
	server := httptest.NewServer(upstream)
	defer server.Close()
	handler, item, spec := newAPIGeneratedKeyTestTarget(server.URL)

	created, err := handler.createNewAPIRemoteGroupKey(t.Context(), item, spec)

	require.NoError(t, err)
	require.Equal(t, "sk-grok-generated", created.APIKey)
	require.Equal(t, int64(42), created.ID)
	require.Equal(t, int32(1), upstream.createCalls.Load())
	require.Equal(t, int32(2), upstream.keyCalls.Load())
	require.Zero(t, upstream.deleteCalls.Load())
}

func TestCreateNewAPIRemoteGroupKeyCleansUpAfterExhaustedRead429(t *testing.T) {
	upstream := &newAPIGeneratedKeyTestServer{t: t}
	upstream.keyResponse = func(w http.ResponseWriter, _ int32) {
		writeNewAPIGeneratedKeyTestJSON(t, w, http.StatusTooManyRequests, map[string]any{"success": false, "message": "rate limited"})
	}
	server := httptest.NewServer(upstream)
	defer server.Close()
	handler, item, spec := newAPIGeneratedKeyTestTarget(server.URL)

	_, err := handler.createNewAPIRemoteGroupKey(t.Context(), item, spec)

	require.ErrorContains(t, err, "HTTP 429")
	require.Equal(t, int32(1), upstream.createCalls.Load())
	require.Equal(t, int32(3), upstream.keyCalls.Load())
	require.Equal(t, int32(1), upstream.deleteCalls.Load())
}

func TestCreateNewAPIRemoteGroupKeyCancellationStopsRetryAndCleansUp(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	upstream := &newAPIGeneratedKeyTestServer{t: t}
	upstream.keyResponse = func(w http.ResponseWriter, _ int32) {
		writeNewAPIGeneratedKeyTestJSON(t, w, http.StatusTooManyRequests, map[string]any{"success": false, "message": "rate limited"})
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
	}
	server := httptest.NewServer(upstream)
	defer server.Close()
	handler, item, spec := newAPIGeneratedKeyTestTarget(server.URL)
	startedAt := time.Now()

	_, err := handler.createNewAPIRemoteGroupKey(ctx, item, spec)

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err)
	require.Less(t, time.Since(startedAt), upstreamNewAPIGeneratedKeyReadRetryBaseDelay)
	require.Equal(t, int32(1), upstream.createCalls.Load())
	require.Equal(t, int32(1), upstream.keyCalls.Load())
	require.Equal(t, int32(1), upstream.deleteCalls.Load())
}

func TestCreateNewAPIRemoteGroupKeyCleansUpEmptyRead(t *testing.T) {
	upstream := &newAPIGeneratedKeyTestServer{t: t}
	upstream.keyResponse = func(w http.ResponseWriter, _ int32) {
		writeNewAPIGeneratedKeyTestJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data":    map[string]any{"key": "  "},
		})
	}
	server := httptest.NewServer(upstream)
	defer server.Close()
	handler, item, spec := newAPIGeneratedKeyTestTarget(server.URL)

	_, err := handler.createNewAPIRemoteGroupKey(t.Context(), item, spec)

	require.ErrorContains(t, err, "empty generated key")
	require.Equal(t, int32(1), upstream.createCalls.Load())
	require.Equal(t, int32(1), upstream.keyCalls.Load())
	require.Equal(t, int32(1), upstream.deleteCalls.Load())
}
