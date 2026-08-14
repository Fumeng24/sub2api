package server

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterCustomRoutes(
	r *gin.Engine,
	handlers *handler.Handlers,
	customHandlers *handler.CustomHandlers,
	jwtAuth middleware2.JWTAuthMiddleware,
	adminAuth middleware2.AdminAuthMiddleware,
	adminOrSupportAuth middleware2.AdminOrSupportAuthMiddleware,
	settingService *service.SettingService,
) {
	v1 := r.Group("/api/v1")
	routes.RegisterCustomPublicRoutes(v1, handlers)
	routes.RegisterCustomUserRoutes(v1, handlers, customHandlers, jwtAuth, settingService)
	routes.RegisterCustomAdminRoutes(v1, handlers, customHandlers, adminAuth, adminOrSupportAuth, settingService)
	routes.RegisterCustomPaymentRoutes(v1, handlers.PaymentWebhook)
}
