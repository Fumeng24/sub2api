package handler

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

type CreateInvoiceHTTPRequest struct {
	InvoiceType    string  `json:"invoice_type"`
	Title          string  `json:"title" binding:"required"`
	TaxID          string  `json:"tax_id"`
	ItemName       string  `json:"item_name" binding:"required"`
	Amount         float64 `json:"amount" binding:"required,gt=0"`
	ReceiverEmail  string  `json:"receiver_email" binding:"required"`
	Note           string  `json:"note"`
	SourceOrderIDs []int64 `json:"source_order_ids"`
}

type SaveInvoiceTemplateHTTPRequest struct {
	Name          string `json:"name"`
	InvoiceType   string `json:"invoice_type"`
	Title         string `json:"title" binding:"required"`
	TaxID         string `json:"tax_id"`
	ItemName      string `json:"item_name" binding:"required"`
	ReceiverEmail string `json:"receiver_email" binding:"required"`
	Note          string `json:"note"`
	IsDefault     bool   `json:"is_default"`
}

func (h *InvoiceHandler) Summary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	summary, err := h.invoiceService.GetSummary(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *InvoiceHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, pageResult, err := h.invoiceService.ListByUser(c.Request.Context(), subject.UserID, pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, pageResult.Total, pageResult.Page, pageResult.PageSize)
}

func (h *InvoiceHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseInvoiceIDParam(c)
	if !ok {
		return
	}
	inv, err := h.invoiceService.GetForUser(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func (h *InvoiceHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	var req CreateInvoiceHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inv, err := h.invoiceService.Create(c.Request.Context(), service.CreateInvoiceRequestInput{
		UserID:         subject.UserID,
		InvoiceType:    req.InvoiceType,
		Title:          req.Title,
		TaxID:          req.TaxID,
		ItemName:       req.ItemName,
		Amount:         req.Amount,
		ReceiverEmail:  req.ReceiverEmail,
		Note:           req.Note,
		SourceOrderIDs: req.SourceOrderIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func (h *InvoiceHandler) Cancel(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseInvoiceIDParam(c)
	if !ok {
		return
	}
	inv, err := h.invoiceService.Cancel(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, inv)
}

func (h *InvoiceHandler) ListTemplates(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	items, err := h.invoiceService.ListTemplates(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *InvoiceHandler) CreateTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	var req SaveInvoiceTemplateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tmpl, err := h.invoiceService.CreateTemplate(c.Request.Context(), service.SaveInvoiceTemplateInput{
		UserID:        subject.UserID,
		Name:          req.Name,
		InvoiceType:   req.InvoiceType,
		Title:         req.Title,
		TaxID:         req.TaxID,
		ItemName:      req.ItemName,
		ReceiverEmail: req.ReceiverEmail,
		Note:          req.Note,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *InvoiceHandler) UpdateTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseInvoiceTemplateIDParam(c)
	if !ok {
		return
	}
	var req SaveInvoiceTemplateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tmpl, err := h.invoiceService.UpdateTemplate(c.Request.Context(), id, subject.UserID, service.SaveInvoiceTemplateInput{
		UserID:        subject.UserID,
		Name:          req.Name,
		InvoiceType:   req.InvoiceType,
		Title:         req.Title,
		TaxID:         req.TaxID,
		ItemName:      req.ItemName,
		ReceiverEmail: req.ReceiverEmail,
		Note:          req.Note,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *InvoiceHandler) DeleteTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseInvoiceTemplateIDParam(c)
	if !ok {
		return
	}
	if err := h.invoiceService.DeleteTemplate(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *InvoiceHandler) SetDefaultTemplate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, ok := parseInvoiceTemplateIDParam(c)
	if !ok {
		return
	}
	tmpl, err := h.invoiceService.SetDefaultTemplate(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func parseInvoiceIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid invoice request ID")
		return 0, false
	}
	return id, true
}

func parseInvoiceTemplateIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("template_id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid invoice template ID")
		return 0, false
	}
	return id, true
}
