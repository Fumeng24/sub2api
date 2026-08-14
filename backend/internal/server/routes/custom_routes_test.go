package routes

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCustomRoutesRegisterAlongsideUpstreamRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")
	h := newRouteTestHandlers(t)
	custom := newCustomRouteTestHandlers(t)
	pass := func(c *gin.Context) { c.Next() }

	RegisterUserRoutes(v1, h, middleware.JWTAuthMiddleware(pass), nil, nil, nil)
	RegisterCustomUserRoutes(v1, h, custom, middleware.JWTAuthMiddleware(pass), nil)
	RegisterAdminRoutes(v1, h, middleware.AdminAuthMiddleware(pass), nil, nil, nil, nil)
	RegisterCustomAdminRoutes(
		v1,
		h,
		custom,
		middleware.AdminAuthMiddleware(pass),
		middleware.AdminOrSupportAuthMiddleware(pass),
		nil,
	)

	registered := make(map[string]struct{}, len(router.Routes()))
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	expected := []string{
		"POST /api/v1/user/aff/bind",
		"POST /api/v1/user/aff/bind-bonus/claim",
		"GET /api/v1/groups/subscription-capability",
		"GET /api/v1/usage/dashboard/groups",
		"GET /api/v1/usage/dashboard/endpoints",
		"GET /api/v1/tickets",
		"POST /api/v1/tickets",
		"GET /api/v1/tickets/templates",
		"GET /api/v1/tickets/prefill",
		"GET /api/v1/tickets/unread-summary",
		"GET /api/v1/tickets/:id",
		"POST /api/v1/tickets/:id/messages",
		"POST /api/v1/tickets/:id/close",
		"POST /api/v1/tickets/:id/reopen",
		"GET /api/v1/invoices/summary",
		"GET /api/v1/invoices/templates",
		"POST /api/v1/invoices/templates",
		"PUT /api/v1/invoices/templates/:template_id",
		"DELETE /api/v1/invoices/templates/:template_id",
		"POST /api/v1/invoices/templates/:template_id/default",
		"GET /api/v1/invoices",
		"POST /api/v1/invoices",
		"GET /api/v1/invoices/:id",
		"POST /api/v1/invoices/:id/cancel",
		"POST /api/v1/subscriptions/:id/reset",
		"PATCH /api/v1/subscriptions/:id/auto-reset",
		"GET /api/v1/admin/tickets",
		"GET /api/v1/admin/tickets/unread-summary",
		"GET /api/v1/admin/tickets/stats",
		"GET /api/v1/admin/tickets/capabilities",
		"POST /api/v1/admin/tickets/batch-update",
		"POST /api/v1/admin/tickets/auto-close-resolved",
		"GET /api/v1/admin/tickets/:id",
		"PUT /api/v1/admin/tickets/:id",
		"POST /api/v1/admin/tickets/:id/claim",
		"POST /api/v1/admin/tickets/:id/escalate",
		"POST /api/v1/admin/tickets/:id/balance-adjust",
		"POST /api/v1/admin/tickets/:id/messages",
		"GET /api/v1/admin/invoices",
		"GET /api/v1/admin/invoices/:id",
		"POST /api/v1/admin/invoices/:id/approve",
		"POST /api/v1/admin/invoices/:id/reject",
		"POST /api/v1/admin/invoices/:id/complete",
		"GET /api/v1/admin/groups/:id/relative-rate-multipliers",
		"PUT /api/v1/admin/groups/:id/relative-rate-multipliers",
		"DELETE /api/v1/admin/groups/:id/relative-rate-multipliers",
		"GET /api/v1/admin/groups/:id/account-scheduling",
		"GET /api/v1/admin/groups/:id/account-scheduling/history",
		"PUT /api/v1/admin/groups/:id/account-scheduling",
		"POST /api/v1/admin/groups/:id/rate-change-notification/preview",
		"POST /api/v1/admin/groups/:id/rate-change-notification/send",
		"GET /api/v1/admin/accounts/upstream-sub2api-status",
		"POST /api/v1/admin/accounts/bulk-test-models",
		"POST /api/v1/admin/accounts/bulk-verify",
		"GET /api/v1/admin/accounts/bulk-verify/:id",
		"POST /api/v1/admin/accounts/bulk-verify/:id/apply",
		"DELETE /api/v1/admin/accounts/bulk-verify/:id",
		"PUT /api/v1/admin/accounts/:id/scheduler-config",
		"POST /api/v1/admin/accounts/:id/copy",
		"POST /api/v1/admin/upstreams/:id/models/probe",
		"POST /api/v1/admin/upstreams/accounts/rename-preview",
		"POST /api/v1/admin/upstreams/accounts/rename-apply",
		"PUT /api/v1/admin/upstreams/:id/accounts/:account_id/upstream-group",
		"POST /api/v1/admin/subscriptions/:id/reset-with-cost",
		"PUT /api/v1/admin/channel-monitors/sort-order",
		"GET /api/v1/admin/account-monitors",
		"GET /api/v1/admin/account-monitors/status",
		"POST /api/v1/admin/account-monitors",
		"PUT /api/v1/admin/account-monitors/:id",
		"DELETE /api/v1/admin/account-monitors/:id",
		"POST /api/v1/admin/account-monitors/:id/run",
	}

	for _, route := range expected {
		_, ok := registered[route]
		require.Truef(t, ok, "custom route is not registered: %s", route)
	}
}

func TestCustomRouteAuthBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "user route uses JWT auth", method: http.MethodGet, path: "/api/v1/tickets", wantStatus: 461},
		{name: "ticket admin route permits support auth", method: http.MethodGet, path: "/api/v1/admin/tickets", wantStatus: 462},
		{name: "invoice admin route requires admin auth", method: http.MethodGet, path: "/api/v1/admin/invoices", wantStatus: 463},
		{name: "scheduler admin route requires admin auth", method: http.MethodGet, path: "/api/v1/admin/groups/1/account-scheduling", wantStatus: 463},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			v1 := router.Group("/api/v1")
			h := newRouteTestHandlers(t)
			custom := newCustomRouteTestHandlers(t)
			abort := func(status int) gin.HandlerFunc {
				return func(c *gin.Context) { c.AbortWithStatus(status) }
			}

			RegisterCustomUserRoutes(v1, h, custom, middleware.JWTAuthMiddleware(abort(461)), nil)
			RegisterCustomAdminRoutes(
				v1,
				h,
				custom,
				middleware.AdminAuthMiddleware(abort(463)),
				middleware.AdminOrSupportAuthMiddleware(abort(462)),
				nil,
			)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(recorder, request)
			require.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func newRouteTestHandlers(t *testing.T) *handler.Handlers {
	t.Helper()

	h := &handler.Handlers{Admin: &handler.AdminHandlers{}}
	initializePointerFields(t, reflect.ValueOf(h).Elem())
	initializePointerFields(t, reflect.ValueOf(h.Admin).Elem())
	return h
}

func newCustomRouteTestHandlers(t *testing.T) *handler.CustomHandlers {
	t.Helper()
	h := &handler.CustomHandlers{Admin: &handler.CustomAdminHandlers{}}
	initializePointerFields(t, reflect.ValueOf(h).Elem())
	initializePointerFields(t, reflect.ValueOf(h.Admin).Elem())
	return h
}

func initializePointerFields(t *testing.T, value reflect.Value) {
	t.Helper()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Ptr && field.IsNil() {
			require.True(t, field.CanSet())
			field.Set(reflect.New(field.Type().Elem()))
		}
	}
}
