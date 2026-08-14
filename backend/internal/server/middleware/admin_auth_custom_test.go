//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminOrSupportAuthRoleBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	users := map[int64]*service.User{
		1: {ID: 1, Email: "support@example.com", Role: service.RoleSupport, Status: service.StatusActive, TokenVersion: 2, Concurrency: 1},
		2: {ID: 2, Email: "user@example.com", Role: service.RoleUser, Status: service.StatusActive, TokenVersion: 2, Concurrency: 1},
	}
	userRepo := &stubUserRepo{getByID: func(_ context.Context, id int64) (*service.User, error) {
		user, ok := users[id]
		if !ok {
			return nil, service.ErrUserNotFound
		}
		clone := *user
		return &clone, nil
	}}
	userService := service.NewUserService(userRepo, nil, nil, nil)

	request := func(t *testing.T, middleware gin.HandlerFunc, user *service.User, websocket bool) *httptest.ResponseRecorder {
		t.Helper()
		token, err := authService.GenerateToken(context.Background(), user)
		require.NoError(t, err)

		router := gin.New()
		router.Use(middleware)
		router.GET("/t", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		if websocket {
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		router.ServeHTTP(w, req)
		return w
	}

	supportMiddleware := gin.HandlerFunc(NewAdminOrSupportAuthMiddleware(authService, userService, nil))
	adminMiddleware := gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, nil, nil))

	require.Equal(t, http.StatusOK, request(t, supportMiddleware, users[1], false).Code)
	require.Equal(t, http.StatusForbidden, request(t, supportMiddleware, users[2], false).Code)
	require.Equal(t, http.StatusForbidden, request(t, adminMiddleware, users[1], false).Code)
	require.Equal(t, http.StatusForbidden, request(t, supportMiddleware, users[1], true).Code)
}
