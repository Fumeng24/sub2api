package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetOpenAISchedulerStatus returns OpenAI scheduler runtime status.
// GET /api/v1/admin/ops/openai-scheduler
func (h *OpsHandler) GetOpenAISchedulerStatus(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var groupID *int64
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}

	status, err := h.opsService.GetOpenAISchedulerStatus(
		c.Request.Context(),
		c.Query("model"),
		c.Query("endpoint"),
		groupID,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}
