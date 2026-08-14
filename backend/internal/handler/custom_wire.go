package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/google/wire"
)

var CustomProviderSet = wire.NewSet(
	NewTicketHandler,
	NewInvoiceHandler,
	admin.NewTicketHandler,
	admin.NewAccountMonitorHandler,
	admin.NewBulkVerifyHandler,
	admin.NewInvoiceHandler,
	admin.NewUpstreamHandler,
	admin.ApplyGroupHandlerCustomization,
	ProvideCustomAdminHandlers,
	ProvideCustomHandlers,
)
