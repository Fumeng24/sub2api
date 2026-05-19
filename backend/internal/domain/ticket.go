package domain

import (
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	TicketStatusOpen     = "open"
	TicketStatusPending  = "pending"
	TicketStatusResolved = "resolved"
	TicketStatusClosed   = "closed"
)

const (
	TicketPriorityLow    = "low"
	TicketPriorityNormal = "normal"
	TicketPriorityHigh   = "high"
	TicketPriorityUrgent = "urgent"
)

const (
	TicketCategoryGeneral   = "general"
	TicketCategoryBilling   = "billing"
	TicketCategoryUsage     = "usage"
	TicketCategoryTechnical = "technical"
	TicketCategoryAccount   = "account"
)

const (
	TicketMessageSenderUser   = "user"
	TicketMessageSenderAdmin  = "admin"
	TicketMessageSenderSystem = "system"
)

const (
	TicketMessageVisibilityPublic   = "public"
	TicketMessageVisibilityInternal = "internal"
)

const (
	TicketReadActorUser  = "user"
	TicketReadActorAdmin = "admin"
)

var (
	ErrTicketNotFound        = infraerrors.NotFound("TICKET_NOT_FOUND", "ticket not found")
	ErrTicketMessageNotFound = infraerrors.NotFound("TICKET_MESSAGE_NOT_FOUND", "ticket message not found")
)

type Ticket struct {
	ID                 int64
	TicketNo           string
	UserID             int64
	UserEmail          string
	UserName           string
	Subject            string
	Category           string
	Priority           string
	Status             string
	Source             string
	TemplateKey        string
	ContextType        string
	ContextID          string
	ContextData        map[string]any
	AssigneeID         *int64
	EscalatedAt        *time.Time
	EscalatedBy        *int64
	EscalationReason   string
	SLADueAt           *time.Time
	SLARemindedAt      *time.Time
	LastMessageAt      time.Time
	LastUserMessageAt  *time.Time
	LastAdminMessageAt *time.Time
	ResolvedAt         *time.Time
	ClosedAt           *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Messages    []TicketMessage
	UnreadCount int
}

type TicketAttachment struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type TicketContextData map[string]any

func (d TicketContextData) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]any(d))
}

type TicketMessage struct {
	ID          int64
	TicketID    int64
	SenderType  string
	SenderID    *int64
	SenderName  string
	Visibility  string
	Body        string
	Attachments []TicketAttachment
	EditedAt    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
