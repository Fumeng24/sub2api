package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type CustomAdminHandlers struct {
	Ticket         *admin.TicketHandler
	AccountMonitor *admin.AccountMonitorHandler
	BulkVerify     *admin.BulkVerifyHandler
	Invoice        *admin.InvoiceHandler
	Upstream       *admin.UpstreamHandler
}

type CustomHandlers struct {
	Ticket  *TicketHandler
	Invoice *InvoiceHandler
	Admin   *CustomAdminHandlers
}

func ProvideCustomAdminHandlers(ticket *admin.TicketHandler, accountMonitor *admin.AccountMonitorHandler, bulkVerify *admin.BulkVerifyHandler, invoice *admin.InvoiceHandler, upstream *admin.UpstreamHandler) *CustomAdminHandlers {
	return &CustomAdminHandlers{Ticket: ticket, AccountMonitor: accountMonitor, BulkVerify: bulkVerify, Invoice: invoice, Upstream: upstream}
}

func ProvideCustomHandlers(ticket *TicketHandler, invoice *InvoiceHandler, adminHandlers *CustomAdminHandlers, _ admin.GroupHandlerCustomization, _ service.AdminServiceCustomization, _ service.ConcurrencyCacheCustomization, _ service.RateLimitCustomization, _ service.OpsRuntimeCustomization, _ service.APIKeyCustomization, _ service.GatewayCustomization, _ service.ChannelMonitorCustomization) *CustomHandlers {
	return &CustomHandlers{Ticket: ticket, Invoice: invoice, Admin: adminHandlers}
}
