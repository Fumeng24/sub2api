package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// NewAdminOrSupportAuthMiddleware is used only by explicitly support-enabled
// routes. Admin API keys and WebSocket authentication stay on the official
// admin-only middleware path.
func NewAdminOrSupportAuthMiddleware(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
) AdminOrSupportAuthMiddleware {
	// Support routes retain the custom role checks; audit logging is attached to
	// the official admin middleware and is not required for this compatibility path.
	adminOnly := adminAuth(authService, userService, settingService, nil)
	return AdminOrSupportAuthMiddleware(func(c *gin.Context) {
		// Preserve the official precedence for WebSocket subprotocols and API keys.
		if isWebSocketUpgradeRequest(c) || c.GetHeader("x-api-key") != "" {
			adminOnly(c)
			return
		}

		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			adminOnly(c)
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			AbortWithError(c, 401, "UNAUTHORIZED", "Authorization required")
			return
		}
		if validateJWTForAdminOrSupport(c, token, authService, userService) {
			c.Next()
		}
	})
}

func validateJWTForAdminOrSupport(
	c *gin.Context,
	token string,
	authService *service.AuthService,
	userService *service.UserService,
) bool {
	claims, err := authService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
			return false
		}
		AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
		return false
	}

	user, err := userService.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
		return false
	}
	if !user.IsActive() {
		AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
		return false
	}
	if claims.TokenVersion != user.TokenVersion {
		AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
		return false
	}
	if !user.IsAdmin() && !user.IsSupport() {
		AbortWithError(c, 403, "FORBIDDEN", "Admin access required")
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      user.ID,
		Concurrency: user.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set("auth_method", "jwt")
	return true
}
