package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	TicketStatusOpen     = domain.TicketStatusOpen
	TicketStatusPending  = domain.TicketStatusPending
	TicketStatusResolved = domain.TicketStatusResolved
	TicketStatusClosed   = domain.TicketStatusClosed
)

const (
	TicketPriorityLow    = domain.TicketPriorityLow
	TicketPriorityNormal = domain.TicketPriorityNormal
	TicketPriorityHigh   = domain.TicketPriorityHigh
	TicketPriorityUrgent = domain.TicketPriorityUrgent
)

const (
	TicketCategoryGeneral   = domain.TicketCategoryGeneral
	TicketCategoryBilling   = domain.TicketCategoryBilling
	TicketCategoryUsage     = domain.TicketCategoryUsage
	TicketCategoryTechnical = domain.TicketCategoryTechnical
	TicketCategoryAccount   = domain.TicketCategoryAccount
)

const (
	TicketMessageSenderUser   = domain.TicketMessageSenderUser
	TicketMessageSenderAdmin  = domain.TicketMessageSenderAdmin
	TicketMessageSenderSystem = domain.TicketMessageSenderSystem
)

const (
	TicketMessageVisibilityPublic   = domain.TicketMessageVisibilityPublic
	TicketMessageVisibilityInternal = domain.TicketMessageVisibilityInternal
)

const (
	TicketReadActorUser  = domain.TicketReadActorUser
	TicketReadActorAdmin = domain.TicketReadActorAdmin
)

const (
	TicketTemplateFieldText         = "text"
	TicketTemplateFieldTextarea     = "textarea"
	TicketTemplateFieldSelect       = "select"
	TicketTemplateFieldGroupSelect  = "group_select"
	TicketTemplateFieldRecentOrders = "recent_orders"
	TicketTemplateFieldAmount       = "amount"
	TicketTemplateFieldImage        = "image"
	TicketTemplateFieldAttachments  = "attachments"
)

var (
	ErrTicketNotFound             = domain.ErrTicketNotFound
	ErrTicketMessageNotFound      = domain.ErrTicketMessageNotFound
	ErrTicketNilInput             = infraerrors.BadRequest("TICKET_INPUT_REQUIRED", "ticket input is required")
	ErrTicketSubjectInvalid       = infraerrors.BadRequest("TICKET_SUBJECT_INVALID", "ticket subject is invalid")
	ErrTicketBodyRequired         = infraerrors.BadRequest("TICKET_BODY_REQUIRED", "ticket message body is required")
	ErrTicketStatusInvalid        = infraerrors.BadRequest("TICKET_STATUS_INVALID", "ticket status is invalid")
	ErrTicketPriorityInvalid      = infraerrors.BadRequest("TICKET_PRIORITY_INVALID", "ticket priority is invalid")
	ErrTicketCategoryInvalid      = infraerrors.BadRequest("TICKET_CATEGORY_INVALID", "ticket category is invalid")
	ErrTicketTemplateInvalid      = infraerrors.BadRequest("TICKET_TEMPLATE_INVALID", "ticket template is invalid")
	ErrTicketTemplateFieldInvalid = infraerrors.BadRequest(
		"TICKET_TEMPLATE_FIELD_INVALID",
		"ticket template field is invalid",
	)
	ErrTicketPermissionDenied = infraerrors.Forbidden(
		"TICKET_PERMISSION_DENIED",
		"ticket permission denied",
	)
	ErrTicketAssigneeInvalid = infraerrors.BadRequest(
		"TICKET_ASSIGNEE_INVALID",
		"ticket assignee is invalid",
	)
	ErrTicketClosed            = infraerrors.Forbidden("TICKET_CLOSED", "ticket is closed")
	ErrTicketAttachmentInvalid = infraerrors.BadRequest(
		"TICKET_ATTACHMENT_INVALID",
		"ticket attachment is invalid",
	)
	ErrTicketIDsRequired              = infraerrors.BadRequest("TICKET_IDS_REQUIRED", "ticket ids are required")
	ErrTicketEscalationReasonRequired = infraerrors.BadRequest(
		"TICKET_ESCALATION_REASON_REQUIRED",
		"ticket escalation reason is required",
	)
)

type Ticket = domain.Ticket

type TicketMessage = domain.TicketMessage

type TicketAttachment = domain.TicketAttachment

type TicketUnreadSummary struct {
	Total    int `json:"total"`
	Open     int `json:"open"`
	Pending  int `json:"pending"`
	Resolved int `json:"resolved"`
	Closed   int `json:"closed"`
}

type TicketStats struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	Pending      int `json:"pending"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	Unassigned   int `json:"unassigned"`
	AssignedToMe int `json:"assigned_to_me"`
	HandledByMe  int `json:"handled_by_me"`
	Escalated    int `json:"escalated"`
	SLAOverdue   int `json:"sla_overdue"`
	Unread       int `json:"unread"`
}

type TicketListFilters struct {
	Status          string
	Priority        string
	Category        string
	Search          string
	AssigneeID      *int64
	SupportActorID  *int64
	TemplateKey     string
	EscalatedOnly   bool
	Queue           string
	UnreadOnly      bool
	ReadActorType   string
	ReadActorID     int64
	IncludeInternal bool
}

type CreateTicketInput struct {
	UserID      int64
	Subject     string
	Body        string
	Category    string
	Priority    string
	TemplateKey string
	ContextType string
	ContextID   string
	ContextData map[string]any
	Attachments []TicketAttachment
}

type AddTicketMessageInput struct {
	ActorID     int64
	Body        string
	Internal    bool
	Attachments []TicketAttachment
}

type UpdateTicketInput struct {
	ActorID    int64
	Status     *string
	Priority   *string
	Category   *string
	AssigneeID **int64
}

type TicketTemplateOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type TicketTemplateField struct {
	Key         string                 `json:"key"`
	Label       string                 `json:"label"`
	Type        string                 `json:"type"`
	Required    bool                   `json:"required"`
	MinLength   int                    `json:"min_length,omitempty"`
	MaxLength   int                    `json:"max_length,omitempty"`
	MinValue    *float64               `json:"min_value,omitempty"`
	Options     []TicketTemplateOption `json:"options,omitempty"`
	Description string                 `json:"description,omitempty"`
	Placeholder string                 `json:"placeholder,omitempty"`
}

type TicketTemplate struct {
	Key                  string                `json:"key"`
	Name                 string                `json:"name"`
	Description          string                `json:"description,omitempty"`
	Category             string                `json:"category"`
	Priority             string                `json:"priority"`
	SubjectTemplate      string                `json:"subject_template,omitempty"`
	BodyMinLength        int                   `json:"body_min_length,omitempty"`
	RequiresSuperAdmin   bool                  `json:"requires_super_admin"`
	AutoAssignSuperAdmin bool                  `json:"auto_assign_super_admin"`
	ContextType          string                `json:"context_type,omitempty"`
	Fields               []TicketTemplateField `json:"fields,omitempty"`
}

type TicketSupportPermissions struct {
	CanViewAll             bool `json:"can_view_all"`
	CanViewEscalated       bool `json:"can_view_escalated"`
	CanInternalNote        bool `json:"can_internal_note"`
	CanClose               bool `json:"can_close"`
	CanTransfer            bool `json:"can_transfer"`
	CanBatchUpdate         bool `json:"can_batch_update"`
	CanUpdatePriority      bool `json:"can_update_priority"`
	CanUpdateCategory      bool `json:"can_update_category"`
	CanReplyUnassigned     bool `json:"can_reply_unassigned"`
	CanReplyAssignedToSelf bool `json:"can_reply_assigned_to_self"`
	CanEscalate            bool `json:"can_escalate"`
}

type TicketSLASettings struct {
	Enabled                   bool `json:"enabled"`
	FirstResponseMinutes      int  `json:"first_response_minutes"`
	ReminderBeforeMinutes     int  `json:"reminder_before_minutes"`
	AutoEscalateAfterMinutes  int  `json:"auto_escalate_after_minutes"`
	ReminderNotifications     bool `json:"reminder_notifications"`
	AutoEscalateNotifications bool `json:"auto_escalate_notifications"`
	AutoCloseResolvedDays     int  `json:"auto_close_resolved_days"`
	WorkerIntervalSeconds     int  `json:"worker_interval_seconds"`
}

type TicketSystemSettings struct {
	Templates          []TicketTemplate         `json:"templates"`
	SupportPermissions TicketSupportPermissions `json:"support_permissions"`
	SLA                TicketSLASettings        `json:"sla"`
}

type TicketAdminCapabilities struct {
	Role                 string                   `json:"role"`
	IsSuperAdmin         bool                     `json:"is_super_admin"`
	SupportPermissions   TicketSupportPermissions `json:"support_permissions"`
	CanViewAll           bool                     `json:"can_view_all"`
	CanViewEscalated     bool                     `json:"can_view_escalated"`
	CanInternalNote      bool                     `json:"can_internal_note"`
	CanClose             bool                     `json:"can_close"`
	CanTransfer          bool                     `json:"can_transfer"`
	CanBatchUpdate       bool                     `json:"can_batch_update"`
	CanUpdatePriority    bool                     `json:"can_update_priority"`
	CanUpdateCategory    bool                     `json:"can_update_category"`
	CanReplyUnassigned   bool                     `json:"can_reply_unassigned"`
	CanReplyAssignedSelf bool                     `json:"can_reply_assigned_to_self"`
	CanEscalate          bool                     `json:"can_escalate"`
	CanAdjustBalance     bool                     `json:"can_adjust_balance"`
}

type TicketBalanceAdjustmentInput struct {
	Amount    float64
	Operation string
	Notes     string
}

type BatchUpdateTicketInput struct {
	ActorID    int64
	IDs        []int64
	Status     *string
	Priority   *string
	Category   *string
	AssigneeID **int64
}

type TicketRepository interface {
	CreateWithMessage(ctx context.Context, t *Ticket, msg *TicketMessage) error
	CreateMessage(ctx context.Context, msg *TicketMessage) error
	AddMessageAndUpdateTicket(ctx context.Context, msg *TicketMessage, t *Ticket) error
	GetByID(ctx context.Context, id int64) (*Ticket, error)
	GetByIDForUser(ctx context.Context, id, userID int64) (*Ticket, error)
	Update(ctx context.Context, t *Ticket) error
	List(ctx context.Context, params pagination.PaginationParams, filters TicketListFilters) ([]Ticket, *pagination.PaginationResult, error)
	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, filters TicketListFilters) ([]Ticket, *pagination.PaginationResult, error)
	ListMessages(ctx context.Context, ticketID int64, includeInternal bool) ([]TicketMessage, error)
	MarkRead(ctx context.Context, ticketID int64, actorType string, actorID int64, lastReadMessageID *int64) error
	UnreadCounts(ctx context.Context, ticketIDs []int64, actorType string, actorID int64, includeInternal bool) (map[int64]int, error)
	UnreadSummary(ctx context.Context, userID *int64, actorType string, actorID int64, includeInternal bool, filters ...TicketListFilters) (*TicketUnreadSummary, error)
	Stats(ctx context.Context, filters ...TicketListFilters) (*TicketStats, error)
	StatsForAssignee(ctx context.Context, assigneeID int64) (*TicketStats, error)
	AutoCloseResolved(ctx context.Context, before time.Time) (int, error)
	ListSLAActionable(ctx context.Context, before time.Time, limit int) ([]Ticket, error)
}

func isValidTicketStatus(status string) bool {
	switch status {
	case TicketStatusOpen, TicketStatusPending, TicketStatusResolved, TicketStatusClosed:
		return true
	default:
		return false
	}
}

func isValidTicketPriority(priority string) bool {
	switch priority {
	case TicketPriorityLow, TicketPriorityNormal, TicketPriorityHigh, TicketPriorityUrgent:
		return true
	default:
		return false
	}
}

func isValidTicketCategory(category string) bool {
	switch category {
	case TicketCategoryGeneral, TicketCategoryBilling, TicketCategoryUsage, TicketCategoryTechnical, TicketCategoryAccount:
		return true
	default:
		return false
	}
}

func latestTicketMessageID(messages []TicketMessage) *int64 {
	if len(messages) == 0 {
		return nil
	}
	id := messages[len(messages)-1].ID
	return &id
}

func ticketNowPtr(t time.Time) *time.Time {
	return &t
}
