package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// DashboardGroups handles user-visible group usage statistics.
// GET /api/v1/usage/dashboard/groups
func (h *UsageHandler) DashboardGroups(c *gin.Context) {
	parsed, ok := h.parseUserUsageFilters(c, true)
	if !ok {
		return
	}

	stats, err := h.usageService.GetGroupStatsWithFilters(c.Request.Context(), parsed.StartTime, parsed.EndTime, parsed.Filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"groups":     stats,
		"start_date": parsed.StartTime.Format("2006-01-02"),
		"end_date":   parsed.EndTime.AddDate(0, 0, -1).Format("2006-01-02"),
	})
}

// DashboardEndpoints handles user-visible inbound endpoint usage statistics.
// GET /api/v1/usage/dashboard/endpoints
func (h *UsageHandler) DashboardEndpoints(c *gin.Context) {
	parsed, ok := h.parseUserUsageFilters(c, true)
	if !ok {
		return
	}

	stats, err := h.usageService.GetEndpointStatsWithFilters(c.Request.Context(), parsed.StartTime, parsed.EndTime, parsed.Filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"endpoints":  stats,
		"start_date": parsed.StartTime.Format("2006-01-02"),
		"end_date":   parsed.EndTime.AddDate(0, 0, -1).Format("2006-01-02"),
	})
}
