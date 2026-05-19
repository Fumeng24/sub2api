package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Ticket struct {
	ID                 int64           `json:"id"`
	TicketNo           string          `json:"ticket_no"`
	UserID             int64           `json:"user_id"`
	UserEmail          string          `json:"user_email"`
	UserName           string          `json:"user_name"`
	Subject            string          `json:"subject"`
	Category           string          `json:"category"`
	Priority           string          `json:"priority"`
	Status             string          `json:"status"`
	Source             string          `json:"source"`
	TemplateKey        string          `json:"template_key"`
	ContextType        string          `json:"context_type"`
	ContextID          string          `json:"context_id"`
	ContextData        map[string]any  `json:"context_data,omitempty"`
	AssigneeID         *int64          `json:"assignee_id,omitempty"`
	EscalatedAt        *time.Time      `json:"escalated_at,omitempty"`
	EscalatedBy        *int64          `json:"escalated_by,omitempty"`
	EscalationReason   string          `json:"escalation_reason,omitempty"`
	SLADueAt           *time.Time      `json:"sla_due_at,omitempty"`
	SLARemindedAt      *time.Time      `json:"sla_reminded_at,omitempty"`
	LastMessageAt      time.Time       `json:"last_message_at"`
	LastUserMessageAt  *time.Time      `json:"last_user_message_at,omitempty"`
	LastAdminMessageAt *time.Time      `json:"last_admin_message_at,omitempty"`
	ResolvedAt         *time.Time      `json:"resolved_at,omitempty"`
	ClosedAt           *time.Time      `json:"closed_at,omitempty"`
	UnreadCount        int             `json:"unread_count"`
	Messages           []TicketMessage `json:"messages,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type TicketMessage struct {
	ID          int64                      `json:"id"`
	TicketID    int64                      `json:"ticket_id"`
	SenderType  string                     `json:"sender_type"`
	SenderID    *int64                     `json:"sender_id,omitempty"`
	SenderName  string                     `json:"sender_name"`
	Visibility  string                     `json:"visibility"`
	Body        string                     `json:"body"`
	Attachments []service.TicketAttachment `json:"attachments,omitempty"`
	EditedAt    *time.Time                 `json:"edited_at,omitempty"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
}

func TicketFromService(t *service.Ticket) *Ticket {
	if t == nil {
		return nil
	}
	out := &Ticket{
		ID:                 t.ID,
		TicketNo:           t.TicketNo,
		UserID:             t.UserID,
		UserEmail:          t.UserEmail,
		UserName:           t.UserName,
		Subject:            t.Subject,
		Category:           t.Category,
		Priority:           t.Priority,
		Status:             t.Status,
		Source:             t.Source,
		TemplateKey:        t.TemplateKey,
		ContextType:        t.ContextType,
		ContextID:          t.ContextID,
		ContextData:        t.ContextData,
		AssigneeID:         t.AssigneeID,
		EscalatedAt:        t.EscalatedAt,
		EscalatedBy:        t.EscalatedBy,
		EscalationReason:   t.EscalationReason,
		SLADueAt:           t.SLADueAt,
		SLARemindedAt:      t.SLARemindedAt,
		LastMessageAt:      t.LastMessageAt,
		LastUserMessageAt:  t.LastUserMessageAt,
		LastAdminMessageAt: t.LastAdminMessageAt,
		ResolvedAt:         t.ResolvedAt,
		ClosedAt:           t.ClosedAt,
		UnreadCount:        t.UnreadCount,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
	if len(t.Messages) > 0 {
		out.Messages = make([]TicketMessage, 0, len(t.Messages))
		for i := range t.Messages {
			out.Messages = append(out.Messages, *TicketMessageFromService(&t.Messages[i]))
		}
	}
	return out
}

func TicketMessageFromService(m *service.TicketMessage) *TicketMessage {
	if m == nil {
		return nil
	}
	return &TicketMessage{
		ID:          m.ID,
		TicketID:    m.TicketID,
		SenderType:  m.SenderType,
		SenderID:    m.SenderID,
		SenderName:  m.SenderName,
		Visibility:  m.Visibility,
		Body:        m.Body,
		Attachments: m.Attachments,
		EditedAt:    m.EditedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
