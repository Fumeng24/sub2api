package repository

import "github.com/google/wire"

// CustomProviderSet contains repositories maintained by this deployment.
var CustomProviderSet = wire.NewSet(
	NewAccountMonitorRepository,
	NewTicketRepository,
	NewInvoiceRepository,
	NewTransientErrorCounterCache,
	NewSlotPoolCache,
)
