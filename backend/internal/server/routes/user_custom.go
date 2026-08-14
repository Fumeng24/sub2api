package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterCustomUserRoutes registers authenticated site-specific user routes.
func RegisterCustomUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	custom *handler.CustomHandlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))

	user := authenticated.Group("/user")
	user.POST("/aff/bind", h.User.BindAffiliateInviter)
	user.POST("/aff/bind-bonus/claim", h.User.ClaimAffiliateBindBonus)

	authenticated.GET("/groups/subscription-capability", h.APIKey.GetSubscriptionCapability)
	authenticated.GET("/usage/dashboard/groups", h.Usage.DashboardGroups)
	authenticated.GET("/usage/dashboard/endpoints", h.Usage.DashboardEndpoints)

	registerCustomUserTicketRoutes(authenticated, custom)
	registerCustomUserInvoiceRoutes(authenticated, custom)

	authenticated.POST("/subscriptions/:id/reset", h.Subscription.ResetSubscription)
	authenticated.PATCH("/subscriptions/:id/auto-reset", h.Subscription.SetAutoReset)
}

func registerCustomUserTicketRoutes(authenticated *gin.RouterGroup, h *handler.CustomHandlers) {
	tickets := authenticated.Group("/tickets")
	tickets.GET("", h.Ticket.List)
	tickets.POST("", h.Ticket.Create)
	tickets.GET("/templates", h.Ticket.Templates)
	tickets.GET("/prefill", h.Ticket.Prefill)
	tickets.GET("/unread-summary", h.Ticket.UnreadSummary)
	tickets.GET("/:id", h.Ticket.GetByID)
	tickets.POST("/:id/messages", h.Ticket.AddMessage)
	tickets.POST("/:id/close", h.Ticket.Close)
	tickets.POST("/:id/reopen", h.Ticket.Reopen)
}

func registerCustomUserInvoiceRoutes(authenticated *gin.RouterGroup, h *handler.CustomHandlers) {
	invoices := authenticated.Group("/invoices")
	invoices.GET("/summary", h.Invoice.Summary)

	templates := invoices.Group("/templates")
	templates.GET("", h.Invoice.ListTemplates)
	templates.POST("", h.Invoice.CreateTemplate)
	templates.PUT("/:template_id", h.Invoice.UpdateTemplate)
	templates.DELETE("/:template_id", h.Invoice.DeleteTemplate)
	templates.POST("/:template_id/default", h.Invoice.SetDefaultTemplate)

	invoices.GET("", h.Invoice.List)
	invoices.POST("", h.Invoice.Create)
	invoices.GET("/:id", h.Invoice.GetByID)
	invoices.POST("/:id/cancel", h.Invoice.Cancel)
}
