package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	maxTicketAttachments              = 5
	maxTicketAttachmentNameLen        = 120
	maxTicketAttachmentURLLen         = 1000
	maxTicketAttachmentTypeLen        = 100
	ticketNotificationTimeout         = 30 * time.Second
	defaultTicketNotificationSiteName = "Sub2API"
)

type TicketService struct {
	ticketRepo   TicketRepository
	userRepo     UserRepository
	emailService *EmailService
	settingRepo  SettingRepository
}

func NewTicketService(ticketRepo TicketRepository, userRepo UserRepository, emailService *EmailService, settingRepo SettingRepository) *TicketService {
	return &TicketService{
		ticketRepo:   ticketRepo,
		userRepo:     userRepo,
		emailService: emailService,
		settingRepo:  settingRepo,
	}
}

func (s *TicketService) CreateForUser(ctx context.Context, input *CreateTicketInput) (*Ticket, error) {
	if input == nil {
		return nil, ErrTicketNilInput
	}
	if input.UserID <= 0 {
		return nil, ErrUserNotFound
	}

	subject := strings.TrimSpace(input.Subject)
	body := strings.TrimSpace(input.Body)
	if subject == "" || len(subject) > 200 {
		return nil, ErrTicketSubjectInvalid
	}
	if body == "" {
		return nil, ErrTicketBodyRequired
	}

	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	templates, err := s.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	template := findTicketTemplate(templates, input.TemplateKey)
	if input.TemplateKey != "" && template == nil {
		return nil, ErrTicketTemplateInvalid
	}

	categoryInput := input.Category
	priorityInput := input.Priority
	if template != nil {
		if categoryInput == "" {
			categoryInput = template.Category
		}
		if priorityInput == "" {
			priorityInput = template.Priority
		}
	}
	category, err := normalizeTicketCategory(categoryInput)
	if err != nil {
		return nil, err
	}
	priority, err := normalizeTicketPriority(priorityInput)
	if err != nil {
		return nil, err
	}
	attachments, err := normalizeTicketAttachments(input.Attachments)
	if err != nil {
		return nil, err
	}
	contextData := normalizeTicketContextData(input.ContextData)
	if template != nil {
		if err := validateTicketTemplateSubmission(template, body, contextData, attachments); err != nil {
			return nil, err
		}
	}
	assigneeID, err := s.initialTicketAssignee(ctx, template)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	t := &Ticket{
		TicketNo:          generateTicketNo(now),
		UserID:            user.ID,
		UserEmail:         user.Email,
		UserName:          displayTicketActorName(user),
		Subject:           subject,
		Category:          category,
		Priority:          priority,
		Status:            TicketStatusOpen,
		Source:            "user",
		TemplateKey:       trimTicketContext(input.TemplateKey, 80),
		ContextType:       trimTicketContext(firstNonEmpty(input.ContextType, templateContextType(template)), 50),
		ContextID:         trimTicketContext(input.ContextID, 128),
		ContextData:       contextData,
		AssigneeID:        assigneeID,
		LastMessageAt:     now,
		LastUserMessageAt: &now,
	}
	if template != nil && template.RequiresSuperAdmin {
		t.EscalatedAt = &now
		t.EscalationReason = "requires super admin"
	}
	senderID := user.ID
	msg := &TicketMessage{
		SenderType:  TicketMessageSenderUser,
		SenderID:    &senderID,
		SenderName:  displayTicketActorName(user),
		Visibility:  TicketMessageVisibilityPublic,
		Body:        body,
		Attachments: attachments,
	}

	if err := s.ticketRepo.CreateWithMessage(ctx, t, msg); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	t.Messages = []TicketMessage{*msg}
	_ = s.ticketRepo.MarkRead(ctx, t.ID, TicketReadActorUser, user.ID, &msg.ID)
	s.notifyAdminsForUserTicket(ctx, t, msg, "created")
	return t, nil
}

func (s *TicketService) ListForUser(ctx context.Context, userID int64, params pagination.PaginationParams, filters TicketListFilters) ([]Ticket, *pagination.PaginationResult, error) {
	filters.ReadActorType = TicketReadActorUser
	filters.ReadActorID = userID
	filters.IncludeInternal = false
	items, page, err := s.ticketRepo.ListByUser(ctx, userID, params, filters)
	if err != nil {
		return nil, nil, err
	}
	if err := s.populateUnreadCounts(ctx, items, TicketReadActorUser, userID, false); err != nil {
		return nil, nil, err
	}
	return items, page, nil
}

func (s *TicketService) ListForAdmin(ctx context.Context, adminID int64, params pagination.PaginationParams, filters TicketListFilters) ([]Ticket, *pagination.PaginationResult, error) {
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, nil, err
	}
	if !admin.CanHandleTickets() || !admin.IsActive() {
		return nil, nil, ErrTicketPermissionDenied
	}
	if admin.IsSupport() {
		if filters.Queue == "super_admin" || filters.EscalatedOnly {
			return []Ticket{}, emptyTicketPagination(params), nil
		}
		requestedQueue := filters.Queue
		filters.Queue = "support"
		if filters.AssigneeID == nil {
			switch requestedQueue {
			case "mine":
				id := admin.ID
				filters.AssigneeID = &id
			case "all":
				// Support agents can see all non-escalated tickets to help triage.
			default:
				// Default support queue includes unassigned and their own tickets.
				id := admin.ID
				filters.SupportActorID = &id
			}
		}
	}
	filters.ReadActorType = TicketReadActorAdmin
	filters.ReadActorID = adminID
	filters.IncludeInternal = true
	items, page, err := s.ticketRepo.List(ctx, params, filters)
	if err != nil {
		return nil, nil, err
	}
	if err := s.populateUnreadCounts(ctx, items, TicketReadActorAdmin, adminID, true); err != nil {
		return nil, nil, err
	}
	return items, page, nil
}

func (s *TicketService) GetForUser(ctx context.Context, ticketID, userID int64) (*Ticket, error) {
	t, err := s.ticketRepo.GetByIDForUser(ctx, ticketID, userID)
	if err != nil {
		return nil, err
	}
	messages, err := s.ticketRepo.ListMessages(ctx, ticketID, false)
	if err != nil {
		return nil, err
	}
	t.Messages = messages
	if latestID := latestTicketMessageID(messages); latestID != nil {
		_ = s.ticketRepo.MarkRead(ctx, ticketID, TicketReadActorUser, userID, latestID)
	}
	return t, nil
}

func (s *TicketService) GetForAdmin(ctx context.Context, ticketID, adminID int64) (*Ticket, error) {
	t, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if !admin.CanHandleTickets() || !admin.IsActive() {
		return nil, ErrTicketPermissionDenied
	}
	if admin.IsSupport() && t.EscalatedAt != nil {
		return nil, ErrTicketPermissionDenied
	}
	messages, err := s.ticketRepo.ListMessages(ctx, ticketID, true)
	if err != nil {
		return nil, err
	}
	t.Messages = messages
	if latestID := latestTicketMessageID(messages); latestID != nil {
		_ = s.ticketRepo.MarkRead(ctx, ticketID, TicketReadActorAdmin, adminID, latestID)
	}
	return t, nil
}

func (s *TicketService) AddUserMessage(ctx context.Context, ticketID, userID int64, input *AddTicketMessageInput) (*TicketMessage, error) {
	if input == nil {
		return nil, ErrTicketNilInput
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, ErrTicketBodyRequired
	}
	attachments, err := normalizeTicketAttachments(input.Attachments)
	if err != nil {
		return nil, err
	}

	t, err := s.ticketRepo.GetByIDForUser(ctx, ticketID, userID)
	if err != nil {
		return nil, err
	}
	if t.Status == TicketStatusClosed {
		return nil, ErrTicketClosed
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	t.Status = TicketStatusOpen
	t.LastMessageAt = now
	t.LastUserMessageAt = &now
	t.ResolvedAt = nil
	t.ClosedAt = nil
	senderID := user.ID
	msg := &TicketMessage{
		TicketID:    ticketID,
		SenderType:  TicketMessageSenderUser,
		SenderID:    &senderID,
		SenderName:  displayTicketActorName(user),
		Visibility:  TicketMessageVisibilityPublic,
		Body:        body,
		Attachments: attachments,
	}
	if err := s.ticketRepo.AddMessageAndUpdateTicket(ctx, msg, t); err != nil {
		return nil, fmt.Errorf("add user ticket message: %w", err)
	}
	_ = s.ticketRepo.MarkRead(ctx, ticketID, TicketReadActorUser, userID, &msg.ID)
	s.notifyAdminsForUserTicket(ctx, t, msg, "updated")
	return msg, nil
}

func (s *TicketService) AddAdminMessage(ctx context.Context, ticketID, adminID int64, input *AddTicketMessageInput) (*TicketMessage, error) {
	if input == nil {
		return nil, ErrTicketNilInput
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, ErrTicketBodyRequired
	}
	attachments, err := normalizeTicketAttachments(input.Attachments)
	if err != nil {
		return nil, err
	}

	t, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if !admin.CanHandleTickets() || !admin.IsActive() {
		return nil, ErrTicketPermissionDenied
	}
	if admin.IsSupport() {
		if t.EscalatedAt != nil {
			return nil, ErrTicketPermissionDenied
		}
		if t.AssigneeID == nil || *t.AssigneeID != admin.ID {
			return nil, ErrTicketPermissionDenied
		}
	}

	visibility := TicketMessageVisibilityPublic
	if input.Internal {
		visibility = TicketMessageVisibilityInternal
	}
	senderID := admin.ID
	msg := &TicketMessage{
		TicketID:    ticketID,
		SenderType:  TicketMessageSenderAdmin,
		SenderID:    &senderID,
		SenderName:  displayTicketActorName(admin),
		Visibility:  visibility,
		Body:        body,
		Attachments: attachments,
	}

	if input.Internal {
		if err := s.ticketRepo.CreateMessage(ctx, msg); err != nil {
			return nil, fmt.Errorf("add internal ticket note: %w", err)
		}
		_ = s.ticketRepo.MarkRead(ctx, ticketID, TicketReadActorAdmin, adminID, &msg.ID)
		return msg, nil
	}

	now := time.Now()
	t.Status = TicketStatusPending
	t.LastMessageAt = now
	t.LastAdminMessageAt = &now
	t.ResolvedAt = nil
	t.ClosedAt = nil
	if err := s.ticketRepo.AddMessageAndUpdateTicket(ctx, msg, t); err != nil {
		return nil, fmt.Errorf("add admin ticket message: %w", err)
	}
	_ = s.ticketRepo.MarkRead(ctx, ticketID, TicketReadActorAdmin, adminID, &msg.ID)
	s.notifyUserForAdminReply(ctx, t, msg)
	return msg, nil
}

func (s *TicketService) UpdateForAdmin(ctx context.Context, ticketID int64, input *UpdateTicketInput) (*Ticket, error) {
	if input == nil {
		return nil, ErrTicketNilInput
	}
	t, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	actor, err := s.ticketAdminActor(ctx, input.ActorID)
	if err != nil {
		return nil, err
	}
	if actor.IsSupport() {
		if t.EscalatedAt != nil {
			return nil, ErrTicketPermissionDenied
		}
		if input.AssigneeID != nil || input.Category != nil || input.Priority != nil {
			return nil, ErrTicketPermissionDenied
		}
		if t.AssigneeID == nil || *t.AssigneeID != actor.ID {
			return nil, ErrTicketPermissionDenied
		}
	}

	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !isValidTicketStatus(status) {
			return nil, ErrTicketStatusInvalid
		}
		applyTicketStatus(t, status, time.Now())
	}
	if input.Priority != nil {
		priority, err := normalizeTicketPriority(*input.Priority)
		if err != nil {
			return nil, err
		}
		t.Priority = priority
	}
	if input.Category != nil {
		category, err := normalizeTicketCategory(*input.Category)
		if err != nil {
			return nil, err
		}
		t.Category = category
	}
	if input.AssigneeID != nil {
		assigneeID, err := s.validateTicketAssignee(ctx, *input.AssigneeID)
		if err != nil {
			return nil, err
		}
		t.AssigneeID = assigneeID
	}

	if err := s.ticketRepo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("update ticket: %w", err)
	}
	return t, nil
}

func (s *TicketService) ClaimForAdmin(ctx context.Context, ticketID, adminID int64) (*Ticket, error) {
	t, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if !admin.CanHandleTickets() || !admin.IsActive() {
		return nil, ErrTicketPermissionDenied
	}
	if t.EscalatedAt != nil && !admin.IsAdmin() {
		return nil, ErrTicketPermissionDenied
	}
	t.AssigneeID = &admin.ID
	if err := s.ticketRepo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("claim ticket: %w", err)
	}
	return t, nil
}

func (s *TicketService) EscalateForAdmin(ctx context.Context, ticketID, adminID int64, reason string) (*Ticket, error) {
	t, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if !admin.CanHandleTickets() || !admin.IsActive() {
		return nil, ErrTicketPermissionDenied
	}
	if !admin.IsAdmin() && t.AssigneeID != nil && *t.AssigneeID != admin.ID {
		return nil, ErrTicketPermissionDenied
	}
	now := time.Now()
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	t.EscalatedAt = &now
	t.EscalatedBy = &admin.ID
	t.EscalationReason = reason
	t.AssigneeID = nil
	msg := &TicketMessage{
		TicketID:    ticketID,
		SenderType:  TicketMessageSenderSystem,
		SenderName:  "system",
		Visibility:  TicketMessageVisibilityInternal,
		Body:        firstNonEmpty(reason, "Escalated to super admin"),
		Attachments: nil,
	}
	if err := s.ticketRepo.AddMessageAndUpdateTicket(ctx, msg, t); err != nil {
		return nil, fmt.Errorf("escalate ticket: %w", err)
	}
	s.notifyAdminsForUserTicket(ctx, t, msg, "escalated")
	return t, nil
}

func (s *TicketService) BatchUpdateForAdmin(ctx context.Context, input *BatchUpdateTicketInput) (int, error) {
	if input == nil {
		return 0, ErrTicketNilInput
	}
	if len(input.IDs) == 0 {
		return 0, ErrTicketIDsRequired
	}
	seen := make(map[int64]struct{}, len(input.IDs))
	count := 0
	for _, id := range input.IDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		_, err := s.UpdateForAdmin(ctx, id, &UpdateTicketInput{
			ActorID:    input.ActorID,
			Status:     input.Status,
			Priority:   input.Priority,
			Category:   input.Category,
			AssigneeID: input.AssigneeID,
		})
		if err != nil {
			return count, err
		}
		count++
	}
	if count == 0 {
		return 0, ErrTicketIDsRequired
	}
	return count, nil
}

func (s *TicketService) UnreadSummaryForUser(ctx context.Context, userID int64) (*TicketUnreadSummary, error) {
	if userID <= 0 {
		return &TicketUnreadSummary{}, nil
	}
	return s.ticketRepo.UnreadSummary(ctx, &userID, TicketReadActorUser, userID, false)
}

func (s *TicketService) UnreadSummaryForAdmin(ctx context.Context, adminID int64) (*TicketUnreadSummary, error) {
	if adminID <= 0 {
		return &TicketUnreadSummary{}, nil
	}
	admin, err := s.ticketAdminActor(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if admin.IsSupport() {
		return s.ticketRepo.UnreadSummary(ctx, nil, TicketReadActorAdmin, adminID, true, supportTicketQueueFilters(admin.ID))
	}
	return s.ticketRepo.UnreadSummary(ctx, nil, TicketReadActorAdmin, adminID, true)
}

func (s *TicketService) StatsForAdmin(ctx context.Context, adminID int64) (*TicketStats, error) {
	admin, err := s.ticketAdminActor(ctx, adminID)
	if err != nil {
		return nil, err
	}
	var stats *TicketStats
	if admin.IsSupport() {
		stats, err = s.ticketRepo.Stats(ctx, supportTicketQueueFilters(admin.ID))
	} else {
		stats, err = s.ticketRepo.Stats(ctx)
	}
	if err != nil {
		return nil, err
	}
	if mine, err := s.ticketRepo.StatsForAssignee(ctx, adminID); err == nil && mine != nil {
		stats.AssignedToMe = mine.AssignedToMe
		stats.HandledByMe = mine.HandledByMe
		stats.Escalated = mine.Escalated
	}
	unread, err := s.UnreadSummaryForAdmin(ctx, adminID)
	if err != nil {
		return nil, err
	}
	stats.Unread = unread.Total
	return stats, nil
}

func supportTicketQueueFilters(adminID int64) TicketListFilters {
	return TicketListFilters{
		Queue:          "support",
		SupportActorID: &adminID,
	}
}

func (s *TicketService) AutoCloseResolved(ctx context.Context, adminID int64, days int) (int, error) {
	admin, err := s.ticketAdminActor(ctx, adminID)
	if err != nil {
		return 0, err
	}
	if !admin.IsAdmin() {
		return 0, ErrTicketPermissionDenied
	}
	if days <= 0 {
		days = 7
	}
	if days > 365 {
		days = 365
	}
	before := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	return s.ticketRepo.AutoCloseResolved(ctx, before)
}

func (s *TicketService) CloseForUser(ctx context.Context, ticketID, userID int64) (*Ticket, error) {
	t, err := s.ticketRepo.GetByIDForUser(ctx, ticketID, userID)
	if err != nil {
		return nil, err
	}
	applyTicketStatus(t, TicketStatusClosed, time.Now())
	if err := s.ticketRepo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("close ticket: %w", err)
	}
	return t, nil
}

func (s *TicketService) ReopenForUser(ctx context.Context, ticketID, userID int64) (*Ticket, error) {
	t, err := s.ticketRepo.GetByIDForUser(ctx, ticketID, userID)
	if err != nil {
		return nil, err
	}
	applyTicketStatus(t, TicketStatusOpen, time.Now())
	if err := s.ticketRepo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("reopen ticket: %w", err)
	}
	return t, nil
}

func (s *TicketService) ListTemplates(ctx context.Context) ([]TicketTemplate, error) {
	settings := defaultTicketSystemSettings()
	if s == nil || s.settingRepo == nil {
		return settings.Templates, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyTicketSystemConfig)
	if err != nil || strings.TrimSpace(raw) == "" {
		return settings.Templates, nil
	}
	var configured TicketSystemSettings
	if err := json.Unmarshal([]byte(raw), &configured); err != nil {
		return settings.Templates, nil
	}
	templates := sanitizeTicketTemplates(configured.Templates)
	if len(templates) == 0 {
		return settings.Templates, nil
	}
	return templates, nil
}

func defaultTicketSystemSettings() TicketSystemSettings {
	zero := 0.0
	return TicketSystemSettings{Templates: []TicketTemplate{
		{
			Key:             "general",
			Name:            "其他问题",
			Description:     "没有匹配分类时使用",
			Category:        TicketCategoryGeneral,
			Priority:        TicketPriorityNormal,
			SubjectTemplate: "其他问题",
			BodyMinLength:   10,
			ContextType:     "general",
		},
		{
			Key:             "group_connection_issue",
			Name:            "分组连接不上",
			Description:     "请提供正在使用的分组、报错截图和详细现象",
			Category:        TicketCategoryTechnical,
			Priority:        TicketPriorityHigh,
			SubjectTemplate: "分组连接问题",
			BodyMinLength:   15,
			ContextType:     "group",
			Fields: []TicketTemplateField{
				{Key: "group_id", Label: "正在使用的分组", Type: TicketTemplateFieldGroupSelect, Required: true},
				{Key: "error_screenshot", Label: "报错截图", Type: TicketTemplateFieldImage, Required: true},
			},
		},
		{
			Key:                  "billing_missing_payment",
			Name:                 "充值未到账",
			Description:          "普通客服无法补款，会自动升级给超级管理员并邮件通知",
			Category:             TicketCategoryBilling,
			Priority:             TicketPriorityUrgent,
			SubjectTemplate:      "充值未到账",
			BodyMinLength:        15,
			RequiresSuperAdmin:   true,
			AutoAssignSuperAdmin: true,
			ContextType:          "order",
			Fields: []TicketTemplateField{
				{Key: "recent_order_ids", Label: "最近 5 条充值记录", Type: TicketTemplateFieldRecentOrders, Required: true},
				{Key: "missing_amount", Label: "未到账金额", Type: TicketTemplateFieldAmount, Required: true, MinValue: &zero},
				{Key: "payment_screenshot", Label: "支付宝或微信支付截图", Type: TicketTemplateFieldImage, Required: true},
			},
		},
		{
			Key:             "api_key_issue",
			Name:            "API Key 有问题",
			Description:     "请描述 Key 的调用现象和错误信息",
			Category:        TicketCategoryUsage,
			Priority:        TicketPriorityNormal,
			SubjectTemplate: "API Key 使用问题",
			BodyMinLength:   15,
			ContextType:     "api_key",
			Fields: []TicketTemplateField{
				{Key: "api_key_id", Label: "API Key ID", Type: TicketTemplateFieldText, Required: false},
				{Key: "error_message", Label: "错误信息", Type: TicketTemplateFieldTextarea, Required: false, MinLength: 5},
			},
		},
	}}
}

func sanitizeTicketTemplates(in []TicketTemplate) []TicketTemplate {
	out := make([]TicketTemplate, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		item.Key = strings.TrimSpace(item.Key)
		item.Name = strings.TrimSpace(item.Name)
		if item.Key == "" || item.Name == "" {
			continue
		}
		if _, ok := seen[item.Key]; ok {
			continue
		}
		category, err := normalizeTicketCategory(item.Category)
		if err != nil {
			category = TicketCategoryGeneral
		}
		priority, err := normalizeTicketPriority(item.Priority)
		if err != nil {
			priority = TicketPriorityNormal
		}
		item.Category = category
		item.Priority = priority
		if item.BodyMinLength < 0 {
			item.BodyMinLength = 0
		}
		item.ContextType = trimTicketContext(item.ContextType, 50)
		item.SubjectTemplate = strings.TrimSpace(item.SubjectTemplate)
		item.Description = strings.TrimSpace(item.Description)
		item.Fields = sanitizeTicketTemplateFields(item.Fields)
		seen[item.Key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func sanitizeTicketTemplateFields(in []TicketTemplateField) []TicketTemplateField {
	out := make([]TicketTemplateField, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, field := range in {
		field.Key = strings.TrimSpace(field.Key)
		field.Label = strings.TrimSpace(field.Label)
		field.Type = strings.TrimSpace(field.Type)
		if field.Key == "" || field.Label == "" || field.Type == "" {
			continue
		}
		if _, ok := seen[field.Key]; ok {
			continue
		}
		if field.MinLength < 0 {
			field.MinLength = 0
		}
		if field.MaxLength < 0 {
			field.MaxLength = 0
		}
		field.Description = strings.TrimSpace(field.Description)
		field.Placeholder = strings.TrimSpace(field.Placeholder)
		field.Options = sanitizeTicketTemplateOptions(field.Options)
		seen[field.Key] = struct{}{}
		out = append(out, field)
	}
	return out
}

func sanitizeTicketTemplateOptions(in []TicketTemplateOption) []TicketTemplateOption {
	out := make([]TicketTemplateOption, 0, len(in))
	for _, option := range in {
		option.Value = strings.TrimSpace(option.Value)
		option.Label = strings.TrimSpace(option.Label)
		if option.Value == "" || option.Label == "" {
			continue
		}
		out = append(out, option)
	}
	return out
}

func findTicketTemplate(templates []TicketTemplate, key string) *TicketTemplate {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	for i := range templates {
		if templates[i].Key == key {
			return &templates[i]
		}
	}
	return nil
}

func validateTicketTemplateSubmission(tpl *TicketTemplate, body string, data map[string]any, attachments []TicketAttachment) error {
	if tpl == nil {
		return nil
	}
	if tpl.BodyMinLength > 0 && len([]rune(strings.TrimSpace(body))) < tpl.BodyMinLength {
		return ErrTicketTemplateFieldInvalid
	}
	for _, field := range tpl.Fields {
		value, ok := data[field.Key]
		if field.Required && isTicketTemplateEmptyValue(value) && !ticketFieldSatisfiedByAttachment(field, attachments) {
			return ErrTicketTemplateFieldInvalid
		}
		if !ok {
			continue
		}
		if err := validateTicketTemplateField(field, value); err != nil {
			return err
		}
	}
	return nil
}

func validateTicketTemplateField(field TicketTemplateField, value any) error {
	if isTicketTemplateEmptyValue(value) {
		return nil
	}
	switch field.Type {
	case TicketTemplateFieldText, TicketTemplateFieldTextarea, TicketTemplateFieldImage:
		text := strings.TrimSpace(fmt.Sprint(value))
		if field.MinLength > 0 && len([]rune(text)) < field.MinLength {
			return ErrTicketTemplateFieldInvalid
		}
		if field.MaxLength > 0 && len([]rune(text)) > field.MaxLength {
			return ErrTicketTemplateFieldInvalid
		}
	case TicketTemplateFieldSelect:
		text := strings.TrimSpace(fmt.Sprint(value))
		if len(field.Options) > 0 {
			for _, option := range field.Options {
				if option.Value == text {
					return nil
				}
			}
			return ErrTicketTemplateFieldInvalid
		}
	case TicketTemplateFieldGroupSelect:
		if _, ok := numericTicketValue(value); !ok {
			return ErrTicketTemplateFieldInvalid
		}
	case TicketTemplateFieldRecentOrders:
		if len(toTicketValueList(value)) == 0 {
			return ErrTicketTemplateFieldInvalid
		}
	case TicketTemplateFieldAmount:
		amount, ok := numericTicketValue(value)
		if !ok {
			return ErrTicketTemplateFieldInvalid
		}
		if field.MinValue != nil && amount <= *field.MinValue {
			return ErrTicketTemplateFieldInvalid
		}
	}
	return nil
}

func isTicketTemplateEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case []string:
		return len(v) == 0
	default:
		return false
	}
}

func ticketFieldSatisfiedByAttachment(field TicketTemplateField, attachments []TicketAttachment) bool {
	if field.Type != TicketTemplateFieldAttachments {
		return false
	}
	return len(attachments) > 0
}

func numericTicketValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func toTicketValueList(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case []string:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case []float64:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func normalizeTicketContextData(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		key = trimTicketContext(key, 80)
		if key == "" {
			continue
		}
		out[key] = normalizeTicketContextValue(value)
	}
	return out
}

func normalizeTicketContextValue(value any) any {
	switch v := value.(type) {
	case string:
		return limitTicketText(v, 2000)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, normalizeTicketContextValue(item))
		}
		return out
	case map[string]any:
		return normalizeTicketContextData(v)
	default:
		return v
	}
}

func (s *TicketService) initialTicketAssignee(ctx context.Context, tpl *TicketTemplate) (*int64, error) {
	if tpl == nil || !tpl.AutoAssignSuperAdmin {
		return nil, nil
	}
	admin, err := s.userRepo.GetFirstAdmin(ctx)
	if err != nil || admin == nil || !admin.IsActive() {
		return nil, nil
	}
	return &admin.ID, nil
}

func (s *TicketService) validateTicketAssignee(ctx context.Context, assigneeID *int64) (*int64, error) {
	if assigneeID == nil {
		return nil, nil
	}
	if *assigneeID <= 0 {
		return nil, nil
	}
	user, err := s.userRepo.GetByID(ctx, *assigneeID)
	if err != nil {
		return nil, ErrTicketAssigneeInvalid
	}
	if user == nil || !user.IsActive() || !user.CanHandleTickets() {
		return nil, ErrTicketAssigneeInvalid
	}
	id := user.ID
	return &id, nil
}

func (s *TicketService) ticketAdminActor(ctx context.Context, actorID int64) (*User, error) {
	if actorID <= 0 {
		return nil, ErrTicketPermissionDenied
	}
	user, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() || !user.CanHandleTickets() {
		return nil, ErrTicketPermissionDenied
	}
	return user, nil
}

func (s *TicketService) populateUnreadCounts(ctx context.Context, items []Ticket, actorType string, actorID int64, includeInternal bool) error {
	if len(items) == 0 || actorID <= 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	counts, err := s.ticketRepo.UnreadCounts(ctx, ids, actorType, actorID, includeInternal)
	if err != nil {
		return err
	}
	for i := range items {
		items[i].UnreadCount = counts[items[i].ID]
	}
	return nil
}

func normalizeTicketAttachments(in []TicketAttachment) ([]TicketAttachment, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if len(in) > maxTicketAttachments {
		return nil, ErrTicketAttachmentInvalid
	}
	out := make([]TicketAttachment, 0, len(in))
	for _, item := range in {
		name := strings.TrimSpace(item.Name)
		rawURL := strings.TrimSpace(item.URL)
		contentType := strings.TrimSpace(item.ContentType)
		if name == "" || rawURL == "" {
			return nil, ErrTicketAttachmentInvalid
		}
		if len(name) > maxTicketAttachmentNameLen ||
			len(rawURL) > maxTicketAttachmentURLLen ||
			len(contentType) > maxTicketAttachmentTypeLen ||
			item.Size < 0 {
			return nil, ErrTicketAttachmentInvalid
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed == nil || parsed.Host == "" {
			return nil, ErrTicketAttachmentInvalid
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, ErrTicketAttachmentInvalid
		}
		out = append(out, TicketAttachment{
			Name:        name,
			URL:         rawURL,
			ContentType: contentType,
			Size:        item.Size,
		})
	}
	return out, nil
}

func (s *TicketService) notifyAdminsForUserTicket(ctx context.Context, t *Ticket, msg *TicketMessage, event string) {
	if s == nil || s.emailService == nil || t == nil || msg == nil {
		return
	}
	recipients := s.adminTicketRecipients(ctx, t)
	if len(recipients) == 0 {
		return
	}
	siteName := s.ticketSiteName(ctx)
	eventLabel := "新工单"
	if event == "updated" {
		eventLabel = "用户回复"
	} else if event == "escalated" {
		eventLabel = "工单升级"
	}
	subject := fmt.Sprintf("[%s] %s %s", sanitizeEmailHeader(siteName), eventLabel, sanitizeEmailHeader(t.TicketNo))
	body := buildTicketNotificationBody(siteName, eventLabel, t, msg, true)
	s.sendTicketNotificationAsync(recipients, subject, body, "ticket_id", t.ID, "event", event)
}

func (s *TicketService) notifyUserForAdminReply(ctx context.Context, t *Ticket, msg *TicketMessage) {
	if s == nil || s.emailService == nil || t == nil || msg == nil {
		return
	}
	recipients := s.userTicketRecipients(ctx, t)
	if len(recipients) == 0 {
		return
	}
	siteName := s.ticketSiteName(ctx)
	subject := fmt.Sprintf("[%s] 工单回复 %s", sanitizeEmailHeader(siteName), sanitizeEmailHeader(t.TicketNo))
	body := buildTicketNotificationBody(siteName, "工单回复", t, msg, false)
	s.sendTicketNotificationAsync(recipients, subject, body, "ticket_id", t.ID, "event", "admin_reply")
}

func (s *TicketService) adminTicketRecipients(ctx context.Context, t *Ticket) []string {
	if t == nil || s.userRepo == nil {
		return nil
	}
	var recipients []string
	if t.AssigneeID != nil && *t.AssigneeID > 0 {
		if admin, err := s.userRepo.GetByID(ctx, *t.AssigneeID); err == nil && admin != nil && admin.IsAdmin() && admin.IsActive() {
			recipients = append(recipients, admin.Email)
		}
	}
	if len(recipients) == 0 {
		if admin, err := s.userRepo.GetFirstAdmin(ctx); err == nil && admin != nil {
			recipients = append(recipients, admin.Email)
		}
	}
	return dedupeTicketEmails(recipients)
}

func (s *TicketService) userTicketRecipients(ctx context.Context, t *Ticket) []string {
	if t == nil {
		return nil
	}
	recipients := []string{t.UserEmail}
	if s.userRepo != nil && t.UserID > 0 {
		if user, err := s.userRepo.GetByID(ctx, t.UserID); err == nil && user != nil {
			recipients = append(recipients, user.Email)
			recipients = append(recipients, filterVerifiedEmails(user.BalanceNotifyExtraEmails)...)
		}
	}
	return dedupeTicketEmails(recipients)
}

func (s *TicketService) ticketSiteName(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return defaultTicketNotificationSiteName
	}
	name, err := s.settingRepo.GetValue(ctx, SettingKeySiteName)
	if err != nil || strings.TrimSpace(name) == "" {
		return defaultTicketNotificationSiteName
	}
	return strings.TrimSpace(name)
}

func (s *TicketService) sendTicketNotificationAsync(recipients []string, subject, body string, logAttrs ...any) {
	if s == nil || s.emailService == nil || len(recipients) == 0 {
		return
	}
	recipients = dedupeTicketEmails(recipients)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in ticket notification", "recover", r)
			}
		}()
		for _, to := range recipients {
			ctx, cancel := context.WithTimeout(context.Background(), ticketNotificationTimeout)
			err := s.emailService.SendEmail(ctx, to, subject, body)
			cancel()
			if err != nil {
				if errors.Is(err, ErrEmailNotConfigured) {
					slog.Debug("ticket notification skipped: email not configured", logAttrs...)
					return
				}
				attrs := append([]any{"to", to, "error", err}, logAttrs...)
				slog.Error("failed to send ticket notification", attrs...)
			}
		}
	}()
}

func dedupeTicketEmails(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, email := range in {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		key := strings.ToLower(email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, email)
	}
	return out
}

func buildTicketNotificationBody(siteName, title string, t *Ticket, msg *TicketMessage, adminView bool) string {
	viewPath := "/tickets"
	if adminView {
		viewPath = "/admin/tickets"
	}
	attachments := buildTicketAttachmentEmailList(msg.Attachments)
	contextLine := ""
	if t.ContextType != "" || t.ContextID != "" {
		contextLine = fmt.Sprintf(
			"<p><strong>关联上下文：</strong>%s %s</p>",
			html.EscapeString(t.ContextType),
			html.EscapeString(t.ContextID),
		)
	}
	return fmt.Sprintf(`<!doctype html>
<html>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;line-height:1.6;color:#111827">
  <h2>%s</h2>
  <p><strong>站点：</strong>%s</p>
  <p><strong>工单：</strong>%s</p>
  <p><strong>主题：</strong>%s</p>
  <p><strong>用户：</strong>%s</p>
  %s
  <div style="margin:16px 0;padding:12px;border-left:4px solid #2563eb;background:#f8fafc;white-space:pre-wrap">%s</div>
  %s
  <p style="color:#6b7280;font-size:12px">请登录后台或用户中心查看完整工单：%s</p>
</body>
</html>`,
		html.EscapeString(title),
		html.EscapeString(siteName),
		html.EscapeString(t.TicketNo),
		html.EscapeString(t.Subject),
		html.EscapeString(firstNonEmpty(t.UserName, t.UserEmail)),
		contextLine,
		html.EscapeString(limitTicketText(msg.Body, 1200)),
		attachments,
		html.EscapeString(viewPath),
	)
}

func buildTicketAttachmentEmailList(attachments []TicketAttachment) string {
	if len(attachments) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<p><strong>附件：</strong></p><ul>")
	for _, item := range attachments {
		b.WriteString("<li><a href=\"")
		b.WriteString(html.EscapeString(item.URL))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(item.Name))
		b.WriteString("</a></li>")
	}
	b.WriteString("</ul>")
	return b.String()
}

func limitTicketText(v string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(v))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func normalizeTicketCategory(category string) (string, error) {
	category = strings.TrimSpace(category)
	if category == "" {
		category = TicketCategoryGeneral
	}
	if !isValidTicketCategory(category) {
		return "", ErrTicketCategoryInvalid
	}
	return category, nil
}

func normalizeTicketPriority(priority string) (string, error) {
	priority = strings.TrimSpace(priority)
	if priority == "" {
		priority = TicketPriorityNormal
	}
	if !isValidTicketPriority(priority) {
		return "", ErrTicketPriorityInvalid
	}
	return priority, nil
}

func applyTicketStatus(t *Ticket, status string, now time.Time) {
	t.Status = status
	switch status {
	case TicketStatusOpen, TicketStatusPending:
		t.ResolvedAt = nil
		t.ClosedAt = nil
	case TicketStatusResolved:
		t.ResolvedAt = ticketNowPtr(now)
		t.ClosedAt = nil
	case TicketStatusClosed:
		if t.ResolvedAt == nil {
			t.ResolvedAt = ticketNowPtr(now)
		}
		t.ClosedAt = ticketNowPtr(now)
	}
}

func displayTicketActorName(u *User) string {
	if u == nil {
		return ""
	}
	name := strings.TrimSpace(u.Username)
	if name != "" {
		return name
	}
	return strings.TrimSpace(u.Email)
}

func trimTicketContext(v string, maxLen int) string {
	v = strings.TrimSpace(v)
	if len(v) > maxLen {
		return v[:maxLen]
	}
	return v
}

func templateContextType(tpl *TicketTemplate) string {
	if tpl == nil {
		return ""
	}
	return tpl.ContextType
}

func emptyTicketPagination(params pagination.PaginationParams) *pagination.PaginationResult {
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.Limit()
	return &pagination.PaginationResult{
		Total:    0,
		Page:     page,
		PageSize: pageSize,
		Pages:    0,
	}
}

func generateTicketNo(now time.Time) string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("T%s%06d", now.Format("20060102150405"), now.Nanosecond()%1000000)
	}
	return "T" + now.Format("20060102150405") + strings.ToUpper(hex.EncodeToString(b[:]))
}
