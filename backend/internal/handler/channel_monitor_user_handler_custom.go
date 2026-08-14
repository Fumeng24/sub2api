package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type channelMonitorUserOverviewResponse struct {
	Items         []channelMonitorUserListItem `json:"items"`
	LastUpdatedAt *string                      `json:"last_updated_at"`
	TrendPeriod   string                       `json:"trend_period"`
}

func (h *ChannelMonitorUserHandler) listUserOverviewCustom(c *gin.Context) bool {
	if !h.featureEnabled(c) {
		response.Success(c, channelMonitorUserOverviewResponse{
			Items:       []channelMonitorUserListItem{},
			TrendPeriod: "7d",
		})
		return true
	}

	overview, err := h.monitorService.ListUserOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return true
	}
	response.Success(c, channelMonitorUserOverviewResponseCustom(overview))
	return true
}

func channelMonitorUserOverviewResponseCustom(overview *service.UserMonitorOverview) channelMonitorUserOverviewResponse {
	items := make([]channelMonitorUserListItem, 0, len(overview.Items))
	for _, view := range overview.Items {
		items = append(items, userMonitorViewToItem(view))
	}

	var lastUpdatedAt *string
	if overview.LastUpdatedAt != nil {
		value := overview.LastUpdatedAt.UTC().Format(time.RFC3339)
		lastUpdatedAt = &value
	}
	return channelMonitorUserOverviewResponse{
		Items:         items,
		LastUpdatedAt: lastUpdatedAt,
		TrendPeriod:   overview.TrendPeriod,
	}
}
