package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterCustomPublicRoutes registers unauthenticated site fact sources.
func RegisterCustomPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	public := v1.Group("/public")
	if h.AvailableChannel != nil {
		public.GET("/model-pricing", h.AvailableChannel.ListPublic)
		public.GET("/channels/available", h.AvailableChannel.ListPublic)
	}
	if h.ChannelMonitor != nil {
		public.GET("/channel-monitors", h.ChannelMonitor.List)
		public.GET("/channel-monitors/:id/status", h.ChannelMonitor.GetStatus)
	}
}
