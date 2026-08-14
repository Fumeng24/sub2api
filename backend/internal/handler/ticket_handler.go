package handler

import (
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	ticketService  *service.TicketService
	paymentService *service.PaymentService
	apiKeyService  *service.APIKeyService
}

func NewTicketHandler(
	ticketService *service.TicketService,
	paymentService *service.PaymentService,
	apiKeyService *service.APIKeyService,
) *TicketHandler {
	return &TicketHandler{
		ticketService:  ticketService,
		paymentService: paymentService,
		apiKeyService:  apiKeyService,
	}
}

type CreateTicketRequest struct {
	Subject     string                     `json:"subject" binding:"required"`
	Body        string                     `json:"body" binding:"required"`
	Category    string                     `json:"category"`
	Priority    string                     `json:"priority"`
	TemplateKey string                     `json:"template_key"`
	ContextType string                     `json:"context_type"`
	ContextID   string                     `json:"context_id"`
	ContextData map[string]any             `json:"context_data"`
	Attachments []service.TicketAttachment `json:"attachments"`
}

type AddTicketMessageRequest struct {
	Body        string                     `json:"body" binding:"required"`
	Attachments []service.TicketAttachment `json:"attachments"`
}

func (h *TicketHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "last_message_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	filters := ticketFiltersFromQuery(c)
	items, paginationResult, err := h.ticketService.ListForUser(c.Request.Context(), subject.UserID, params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Ticket, 0, len(items))
	for i := range items {
		out = append(out, *dto.TicketFromService(&items[i]))
	}
	response.Paginated(c, out, paginationResult.Total, page, pageSize)
}

func (h *TicketHandler) UnreadSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	summary, err := h.ticketService.UnreadSummaryForUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *TicketHandler) Templates(c *gin.Context) {
	templates, err := h.ticketService.ListTemplates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, templates)
}

func (h *TicketHandler) Prefill(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	groups, _ := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	out := gin.H{"groups": groups}
	if h.paymentService != nil {
		orders, _, err := h.paymentService.GetUserOrders(c.Request.Context(), subject.UserID, service.OrderListParams{Page: 1, PageSize: 5})
		if err == nil {
			out["recent_orders"] = safeTicketPaymentOrders(orders)
		}
	}
	response.Success(c, out)
}

type ticketPrefillPaymentOrder struct {
	ID          int64     `json:"id"`
	OrderNo     string    `json:"order_no"`
	Amount      float64   `json:"amount"`
	PayAmount   float64   `json:"pay_amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	OrderType   string    `json:"order_type"`
	PaymentType string    `json:"payment_type"`
	OutTradeNo  string    `json:"out_trade_no"`
	CreatedAt   time.Time `json:"created_at"`
}

func safeTicketPaymentOrders(orders []*dbent.PaymentOrder) []ticketPrefillPaymentOrder {
	out := make([]ticketPrefillPaymentOrder, 0, len(orders))
	for _, order := range orders {
		if order == nil {
			continue
		}
		orderNo := order.OutTradeNo
		if orderNo == "" {
			orderNo = strconv.FormatInt(order.ID, 10)
		}
		out = append(out, ticketPrefillPaymentOrder{
			ID:          order.ID,
			OrderNo:     orderNo,
			Amount:      order.Amount,
			PayAmount:   order.PayAmount,
			Currency:    service.PaymentOrderCurrency(order),
			Status:      order.Status,
			OrderType:   order.OrderType,
			PaymentType: order.PaymentType,
			OutTradeNo:  order.OutTradeNo,
			CreatedAt:   order.CreatedAt,
		})
	}
	return out
}

func (h *TicketHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseTicketIDParam(c)
	if !ok {
		return
	}

	t, err := h.ticketService.GetForUser(c.Request.Context(), ticketID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func (h *TicketHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	t, err := h.ticketService.CreateForUser(c.Request.Context(), &service.CreateTicketInput{
		UserID:      subject.UserID,
		Subject:     req.Subject,
		Body:        req.Body,
		Category:    req.Category,
		Priority:    req.Priority,
		TemplateKey: req.TemplateKey,
		ContextType: req.ContextType,
		ContextID:   req.ContextID,
		ContextData: req.ContextData,
		Attachments: req.Attachments,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func (h *TicketHandler) AddMessage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseTicketIDParam(c)
	if !ok {
		return
	}

	var req AddTicketMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	msg, err := h.ticketService.AddUserMessage(c.Request.Context(), ticketID, subject.UserID, &service.AddTicketMessageInput{
		ActorID:     subject.UserID,
		Body:        req.Body,
		Attachments: req.Attachments,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketMessageFromService(msg))
}

func (h *TicketHandler) Close(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseTicketIDParam(c)
	if !ok {
		return
	}

	t, err := h.ticketService.CloseForUser(c.Request.Context(), ticketID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func (h *TicketHandler) Reopen(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseTicketIDParam(c)
	if !ok {
		return
	}

	t, err := h.ticketService.ReopenForUser(c.Request.Context(), ticketID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func parseTicketIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ticket ID")
		return 0, false
	}
	return id, true
}

func ticketFiltersFromQuery(c *gin.Context) service.TicketListFilters {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}
	return service.TicketListFilters{
		Status:     strings.TrimSpace(c.Query("status")),
		Priority:   strings.TrimSpace(c.Query("priority")),
		Category:   strings.TrimSpace(c.Query("category")),
		Search:     search,
		UnreadOnly: parseTicketBoolQuery(c.Query("unread_only")),
	}
}

func parseTicketBoolQuery(raw string) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && v
}
