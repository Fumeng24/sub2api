package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestCustomPublicRoutesRegistered(t *testing.T) {
	router := newAuthRoutesTestRouter(nil)
	RegisterCustomPublicRoutes(
		router.Group("/api/v1-extra"),
		&handler.Handlers{
			AvailableChannel: handler.NewAvailableChannelHandler(nil, nil, nil),
			ChannelMonitor:   handler.NewChannelMonitorUserHandler(nil, nil),
		},
	)

	registered := make(map[string]bool)
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet {
			registered[route.Path] = true
		}
	}
	for _, path := range []string{
		"/api/v1-extra/public/model-pricing",
		"/api/v1-extra/public/channels/available",
		"/api/v1-extra/public/channel-monitors",
		"/api/v1-extra/public/channel-monitors/:id/status",
	} {
		require.True(t, registered[path], "GET %s should be registered", path)
	}

	for _, path := range []string{
		"/api/v1-extra/public/model-pricing",
		"/api/v1-extra/public/channels/available",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "path=%s", path)
		require.JSONEq(t, `{"code":0,"data":[],"message":"success"}`, w.Body.String(), "path=%s", path)
	}
}
