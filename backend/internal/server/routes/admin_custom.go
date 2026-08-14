package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterCustomAdminRoutes registers site-specific admin and support routes.
func RegisterCustomAdminRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	custom *handler.CustomHandlers,
	adminAuth middleware.AdminAuthMiddleware,
	adminOrSupportAuth middleware.AdminOrSupportAuthMiddleware,
	settingService *service.SettingService,
) {
	supportAdmin := v1.Group("/admin")
	supportAdmin.Use(gin.HandlerFunc(adminOrSupportAuth))
	registerCustomTicketRoutes(supportAdmin, custom)

	admin := v1.Group("/admin")
	admin.Use(gin.HandlerFunc(adminAuth))
	admin.Use(middleware.AdminComplianceGuard(settingService))
	registerCustomInvoiceRoutes(admin, custom)
	registerCustomGroupRoutes(admin, h)
	registerCustomAccountRoutes(admin, h, custom)
	registerCustomUpstreamRoutes(admin, custom)
	registerCustomSubscriptionRoutes(admin, h)
	registerCustomMonitorRoutes(admin, h, custom)
}

func registerCustomUpstreamRoutes(admin *gin.RouterGroup, h *handler.CustomHandlers) {
	upstreams := admin.Group("/upstreams")
	upstreams.GET("", h.Admin.Upstream.List)
	upstreams.POST("", h.Admin.Upstream.Create)
	upstreams.GET("/account-status", h.Admin.Upstream.AccountStatuses)
	upstreams.POST("/account-status/refresh", h.Admin.Upstream.RefreshAccountStatuses)
	upstreams.POST("/accounts/rename-preview", h.Admin.Upstream.RenameAccountsPreview)
	upstreams.POST("/accounts/rename-apply", h.Admin.Upstream.RenameAccountsApply)
	upstreams.GET("/:id", h.Admin.Upstream.Get)
	upstreams.PUT("/:id", h.Admin.Upstream.Update)
	upstreams.DELETE("/:id", h.Admin.Upstream.Delete)
	upstreams.POST("/:id/probe", h.Admin.Upstream.Probe)
	upstreams.POST("/:id/model-test", h.Admin.Upstream.TestModel)
	upstreams.POST("/:id/models/probe", h.Admin.Upstream.ProbeModels)
	upstreams.GET("/:id/bind-candidates", h.Admin.Upstream.ListBindCandidates)
	upstreams.POST("/:id/bind", h.Admin.Upstream.BindAccounts)
	upstreams.POST("/:id/unbind", h.Admin.Upstream.UnbindAccounts)
	upstreams.POST("/:id/accounts/preview", h.Admin.Upstream.PreviewAccounts)
	upstreams.POST("/:id/accounts/generate", h.Admin.Upstream.GenerateAccounts)
	upstreams.PUT("/:id/accounts/:account_id/upstream-group", h.Admin.Upstream.ChangeAccountUpstreamGroup)
}

func registerCustomTicketRoutes(admin *gin.RouterGroup, h *handler.CustomHandlers) {
	tickets := admin.Group("/tickets")
	tickets.GET("", h.Admin.Ticket.List)
	tickets.GET("/unread-summary", h.Admin.Ticket.UnreadSummary)
	tickets.GET("/stats", h.Admin.Ticket.Stats)
	tickets.GET("/capabilities", h.Admin.Ticket.Capabilities)
	tickets.POST("/batch-update", h.Admin.Ticket.BatchUpdate)
	tickets.POST("/auto-close-resolved", h.Admin.Ticket.AutoCloseResolved)
	tickets.GET("/:id", h.Admin.Ticket.GetByID)
	tickets.PUT("/:id", h.Admin.Ticket.Update)
	tickets.POST("/:id/claim", h.Admin.Ticket.Claim)
	tickets.POST("/:id/escalate", h.Admin.Ticket.Escalate)
	tickets.POST("/:id/balance-adjust", h.Admin.Ticket.AdjustBalance)
	tickets.POST("/:id/messages", h.Admin.Ticket.AddMessage)
}

func registerCustomInvoiceRoutes(admin *gin.RouterGroup, h *handler.CustomHandlers) {
	invoices := admin.Group("/invoices")
	invoices.GET("", h.Admin.Invoice.List)
	invoices.GET("/:id", h.Admin.Invoice.GetByID)
	invoices.POST("/:id/approve", h.Admin.Invoice.Approve)
	invoices.POST("/:id/reject", h.Admin.Invoice.Reject)
	invoices.POST("/:id/complete", h.Admin.Invoice.Complete)
}

func registerCustomGroupRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	groups := admin.Group("/groups")
	groups.GET("/:id/relative-rate-multipliers", h.Admin.Group.GetRelativeRateMultipliers)
	groups.PUT("/:id/relative-rate-multipliers", h.Admin.Group.UpdateRelativeRateMultipliers)
	groups.DELETE("/:id/relative-rate-multipliers", h.Admin.Group.ClearRelativeRateMultipliers)
	groups.GET("/:id/account-scheduling", h.Admin.Group.GetAccountScheduling)
	groups.GET("/:id/account-scheduling/history", h.Admin.Group.History)
	groups.PUT("/:id/account-scheduling", h.Admin.Group.UpdateAccountScheduling)
	groups.POST("/:id/rate-change-notification/preview", h.Admin.Group.PreviewGroupRateChangeNotification)
	groups.POST("/:id/rate-change-notification/send", h.Admin.Group.SendGroupRateChangeNotification)
}

func registerCustomAccountRoutes(admin *gin.RouterGroup, h *handler.Handlers, custom *handler.CustomHandlers) {
	accounts := admin.Group("/accounts")
	accounts.GET("/upstream-sub2api-status", h.Admin.Account.GetUpstreamSub2APIStatus)
	accounts.POST("/bulk-test-models", h.Admin.Account.BulkTestModels)
	accounts.POST("/bulk-verify", custom.Admin.BulkVerify.Start)
	accounts.GET("/bulk-verify/:id", custom.Admin.BulkVerify.Get)
	accounts.POST("/bulk-verify/:id/apply", custom.Admin.BulkVerify.Apply)
	accounts.DELETE("/bulk-verify/:id", custom.Admin.BulkVerify.Cancel)
	accounts.PUT("/:id/scheduler-config", h.Admin.Account.UpdateSchedulerConfig)
	accounts.POST("/:id/copy", h.Admin.Account.Copy)
}

func registerCustomSubscriptionRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	admin.POST("/subscriptions/:id/reset-with-cost", h.Admin.Subscription.ResetWithCost)
}

func registerCustomMonitorRoutes(admin *gin.RouterGroup, h *handler.Handlers, custom *handler.CustomHandlers) {
	admin.PUT("/channel-monitors/sort-order", h.Admin.ChannelMonitor.UpdateSortOrder)

	accountMonitors := admin.Group("/account-monitors")
	accountMonitors.GET("", custom.Admin.AccountMonitor.List)
	accountMonitors.GET("/status", custom.Admin.AccountMonitor.Status)
	accountMonitors.POST("", custom.Admin.AccountMonitor.Create)
	accountMonitors.PUT("/:id", custom.Admin.AccountMonitor.Update)
	accountMonitors.DELETE("/:id", custom.Admin.AccountMonitor.Delete)
	accountMonitors.POST("/:id/run", custom.Admin.AccountMonitor.Run)
}
