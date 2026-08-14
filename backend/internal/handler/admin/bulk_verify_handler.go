package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// BulkVerifyHandler exposes the lightweight bulk-verify (wham/usage) endpoints.
type BulkVerifyHandler struct {
	svc *service.WhamVerifyService
}

// NewBulkVerifyHandler wires the handler.
func NewBulkVerifyHandler(svc *service.WhamVerifyService) *BulkVerifyHandler {
	return &BulkVerifyHandler{svc: svc}
}

// Start POST /admin/accounts/bulk-verify
func (h *BulkVerifyHandler) Start(c *gin.Context) {
	if h.svc == nil {
		response.InternalError(c, "bulk-verify service not configured")
		return
	}
	var req service.WhamVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, err.Error())
		return
	}
	job, err := h.svc.Start(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrJobAlreadyRunning) {
			// Return the in-flight job alongside a 409 so the frontend can
			// switch to polling it instead of offering another "Start".
			c.JSON(http.StatusConflict, gin.H{
				"code":    "job_already_running",
				"message": err.Error(),
				"data":    job,
			})
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, job)
}

// Get GET /admin/accounts/bulk-verify/:id
func (h *BulkVerifyHandler) Get(c *gin.Context) {
	if h.svc == nil {
		response.InternalError(c, "bulk-verify service not configured")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.BadRequest(c, "missing job id")
		return
	}
	job := h.svc.Snapshot(id)
	if job == nil {
		response.NotFound(c, "job not found or expired")
		return
	}
	response.Success(c, job)
}

// Apply POST /admin/accounts/bulk-verify/:id/apply
// Executes post-verify actions (refresh expired tokens, mark exhausted as
// rate-limited) based on the targets collected during the verify run.
func (h *BulkVerifyHandler) Apply(c *gin.Context) {
	if h.svc == nil {
		response.InternalError(c, "bulk-verify service not configured")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.BadRequest(c, "missing job id")
		return
	}
	var req service.WhamApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.Apply(c.Request.Context(), id, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

// Cancel DELETE /admin/accounts/bulk-verify/:id
func (h *BulkVerifyHandler) Cancel(c *gin.Context) {
	if h.svc == nil {
		response.InternalError(c, "bulk-verify service not configured")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.BadRequest(c, "missing job id")
		return
	}
	if !h.svc.Cancel(id) {
		response.NotFound(c, "job not found")
		return
	}
	response.Success(c, gin.H{"cancelled": true})
}
