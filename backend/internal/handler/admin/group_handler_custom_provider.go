package admin

import "github.com/Wei-Shaw/sub2api/internal/service"

type GroupHandlerCustomization struct{}

// ApplyGroupHandlerCustomization decorates the upstream-created group handler.
func ApplyGroupHandlerCustomization(h *GroupHandler, concurrencyService *service.ConcurrencyService) GroupHandlerCustomization {
	h.concurrencyService = concurrencyService
	return GroupHandlerCustomization{}
}
