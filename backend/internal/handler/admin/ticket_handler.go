package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	ticketService *service.TicketService
	adminService  service.AdminService
}

func NewTicketHandler(ticketService *service.TicketService, adminService service.AdminService) *TicketHandler {
	return &TicketHandler{ticketService: ticketService, adminService: adminService}
}

type AdminAddTicketMessageRequest struct {
	Body        string                     `json:"body" binding:"required"`
	Internal    bool                       `json:"internal"`
	Attachments []service.TicketAttachment `json:"attachments"`
}

type AdminUpdateTicketRequest struct {
	Status     *string `json:"status"`
	Priority   *string `json:"priority"`
	Category   *string `json:"category"`
	AssigneeID *int64  `json:"assignee_id"`
}

type AdminBatchUpdateTicketRequest struct {
	IDs        []int64 `json:"ids" binding:"required"`
	Status     *string `json:"status"`
	Priority   *string `json:"priority"`
	Category   *string `json:"category"`
	AssigneeID *int64  `json:"assignee_id"`
}

type AdminAutoCloseResolvedRequest struct {
	Days int `json:"days"`
}

type AdminEscalateTicketRequest struct {
	Reason string `json:"reason"`
}

type AdminBalanceAdjustRequest struct {
	Amount           float64 `json:"amount" binding:"required,gt=0"`
	Operation        string  `json:"operation" binding:"required,oneof=set add subtract"`
	Notes            string  `json:"notes"`
	BusinessCategory string  `json:"business_category"`
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
	filters := adminTicketFiltersFromQuery(c)
	items, paginationResult, err := h.ticketService.ListForAdmin(c.Request.Context(), subject.UserID, params, filters)
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

func (h *TicketHandler) Stats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	stats, err := h.ticketService.StatsForAdmin(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *TicketHandler) UnreadSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	summary, err := h.ticketService.UnreadSummaryForAdmin(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *TicketHandler) Capabilities(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	capabilities, err := h.ticketService.CapabilitiesForAdmin(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, capabilities)
}

func (h *TicketHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseAdminTicketIDParam(c)
	if !ok {
		return
	}

	t, err := h.ticketService.GetForAdmin(c.Request.Context(), ticketID, subject.UserID)
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
	ticketID, ok := parseAdminTicketIDParam(c)
	if !ok {
		return
	}

	var req AdminAddTicketMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	msg, err := h.ticketService.AddAdminMessage(c.Request.Context(), ticketID, subject.UserID, &service.AddTicketMessageInput{
		ActorID:     subject.UserID,
		Body:        req.Body,
		Internal:    req.Internal,
		Attachments: req.Attachments,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketMessageFromService(msg))
}

func (h *TicketHandler) Claim(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseAdminTicketIDParam(c)
	if !ok {
		return
	}
	t, err := h.ticketService.ClaimForAdmin(c.Request.Context(), ticketID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func (h *TicketHandler) Escalate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseAdminTicketIDParam(c)
	if !ok {
		return
	}
	var req AdminEscalateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	t, err := h.ticketService.EscalateForAdmin(c.Request.Context(), ticketID, subject.UserID, req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func (h *TicketHandler) AdjustBalance(c *gin.Context) {
	if h.adminService == nil {
		response.BadRequest(c, "Balance adjustment is unavailable")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	role, _ := middleware2.GetUserRoleFromContext(c)
	if role != service.RoleAdmin {
		response.ErrorFrom(c, service.ErrTicketPermissionDenied)
		return
	}
	ticketID, ok := parseAdminTicketIDParam(c)
	if !ok {
		return
	}
	t, err := h.ticketService.GetForAdmin(c.Request.Context(), ticketID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var req AdminBalanceAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	notes := strings.TrimSpace(req.Notes)
	if notes == "" {
		notes = "ticket " + t.TicketNo
	}
	idempotencyPayload := struct {
		TicketID int64                     `json:"ticket_id"`
		UserID   int64                     `json:"user_id"`
		Body     AdminBalanceAdjustRequest `json:"body"`
		Notes    string                    `json:"notes"`
	}{
		TicketID: ticketID,
		UserID:   t.UserID,
		Body:     req,
		Notes:    notes,
	}
	executeAdminIdempotentJSON(c, "admin.tickets.balance.adjust", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		user, execErr := service.UpdateAdminUserBalance(ctx, h.adminService, t.UserID, req.Amount, req.Operation, notes, req.BusinessCategory)
		if execErr != nil {
			return nil, execErr
		}
		return dto.UserFromServiceAdmin(user), nil
	})
}

func (h *TicketHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	ticketID, ok := parseAdminTicketIDParam(c)
	if !ok {
		return
	}

	var req AdminUpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.UpdateTicketInput{
		ActorID:  subject.UserID,
		Status:   req.Status,
		Priority: req.Priority,
		Category: req.Category,
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID > 0 {
			assigneeID := *req.AssigneeID
			assigneePtr := &assigneeID
			input.AssigneeID = &assigneePtr
		} else {
			var cleared *int64
			input.AssigneeID = &cleared
		}
	}

	t, err := h.ticketService.UpdateForAdmin(c.Request.Context(), ticketID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketFromService(t))
}

func (h *TicketHandler) BatchUpdate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	var req AdminBatchUpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.BatchUpdateTicketInput{
		ActorID:  subject.UserID,
		IDs:      req.IDs,
		Status:   req.Status,
		Priority: req.Priority,
		Category: req.Category,
	}
	if req.AssigneeID != nil {
		input.AssigneeID = adminAssigneeInput(req.AssigneeID)
	}
	count, err := h.ticketService.BatchUpdateForAdmin(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": count})
}

func (h *TicketHandler) AutoCloseResolved(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	var req AdminAutoCloseResolvedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	count, err := h.ticketService.AutoCloseResolved(c.Request.Context(), subject.UserID, req.Days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"closed": count})
}

func adminTicketFiltersFromQuery(c *gin.Context) service.TicketListFilters {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}
	filters := service.TicketListFilters{
		Status:        strings.TrimSpace(c.Query("status")),
		Priority:      strings.TrimSpace(c.Query("priority")),
		Category:      strings.TrimSpace(c.Query("category")),
		Search:        search,
		TemplateKey:   strings.TrimSpace(c.Query("template_key")),
		Queue:         strings.TrimSpace(c.Query("queue")),
		EscalatedOnly: parseAdminTicketBoolQuery(c.Query("escalated_only")),
		UnreadOnly:    parseAdminTicketBoolQuery(c.Query("unread_only")),
	}
	if raw := strings.TrimSpace(c.Query("assignee_id")); raw != "" {
		if raw == "unassigned" {
			id := int64(0)
			filters.AssigneeID = &id
		} else if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filters.AssigneeID = &id
		}
	}
	return filters
}

func adminAssigneeInput(raw *int64) **int64 {
	if raw == nil {
		return nil
	}
	if *raw > 0 {
		assigneeID := *raw
		assigneePtr := &assigneeID
		return &assigneePtr
	}
	var cleared *int64
	return &cleared
}

func parseAdminTicketBoolQuery(raw string) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && v
}

func parseAdminTicketIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ticket ID")
		return 0, false
	}
	return id, true
}
