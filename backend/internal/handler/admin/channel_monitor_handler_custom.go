package admin

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type channelMonitorSortOrderRequest struct {
	Updates []struct {
		ID        int64 `json:"id" binding:"required"`
		SortOrder int   `json:"sort_order" binding:"min=0,max=100000"`
	} `json:"updates" binding:"required,min=1"`
}

// UpdateSortOrder PUT /api/v1/admin/channel-monitors/sort-order
func (h *ChannelMonitorHandler) UpdateSortOrder(c *gin.Context) {
	var req channelMonitorSortOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}

	updates := make([]service.ChannelMonitorSortOrderUpdate, 0, len(req.Updates))
	for _, u := range req.Updates {
		updates = append(updates, service.ChannelMonitorSortOrderUpdate{
			ID:        u.ID,
			SortOrder: u.SortOrder,
		})
	}
	if err := h.monitorService.UpdateSortOrders(c.Request.Context(), updates); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Sort order updated successfully"})
}
