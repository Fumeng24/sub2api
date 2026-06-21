package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	invoiceService *service.InvoiceService
}

func NewInvoiceHandler(invoiceService *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

type AdminInvoiceNoteRequest struct {
	AdminNote string `json:"admin_note"`
}

type AdminCompleteInvoiceRequest struct {
	InvoiceNo string `json:"invoice_no"`
	AdminNote string `json:"admin_note"`
}

func (h *InvoiceHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	userID := parseOptionalInt64Query(c.Query("user_id"))
	items, pageResult, err := h.invoiceService.ListForAdmin(c.Request.Context(), pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}, service.InvoiceListFilters{
		UserID: userID,
		Status: strings.TrimSpace(c.Query("status")),
		Search: strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, pageResult.Total, pageResult.Page, pageResult.PageSize)
}

func (h *InvoiceHandler) GetByID(c *gin.Context) {
	id, ok := parseAdminInvoiceIDParam(c)
	if !ok {
		return
	}
	inv, err := h.invoiceService.GetForAdmin(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func (h *InvoiceHandler) Approve(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseAdminInvoiceIDParam(c)
	if !ok {
		return
	}
	var req AdminInvoiceNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inv, err := h.invoiceService.Approve(c.Request.Context(), id, service.ApproveInvoiceInput{
		AdminID:   subject.UserID,
		AdminNote: req.AdminNote,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func (h *InvoiceHandler) Reject(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseAdminInvoiceIDParam(c)
	if !ok {
		return
	}
	var req AdminInvoiceNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inv, err := h.invoiceService.Reject(c.Request.Context(), id, service.RejectInvoiceInput{
		AdminID:   subject.UserID,
		AdminNote: req.AdminNote,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func (h *InvoiceHandler) Complete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseAdminInvoiceIDParam(c)
	if !ok {
		return
	}
	var req AdminCompleteInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inv, err := h.invoiceService.Complete(c.Request.Context(), id, service.CompleteInvoiceInput{
		AdminID:   subject.UserID,
		InvoiceNo: req.InvoiceNo,
		AdminNote: req.AdminNote,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func parseAdminInvoiceIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid invoice request ID")
		return 0, false
	}
	return id, true
}

func parseOptionalInt64Query(raw string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if id < 0 {
		return 0
	}
	return id
}
