package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	InvoiceStatusPending   = "pending"
	InvoiceStatusApproved  = "approved"
	InvoiceStatusRejected  = "rejected"
	InvoiceStatusCompleted = "completed"
	InvoiceStatusCancelled = "cancelled"

	InvoiceTypeCompanyVATGeneral = "company_vat_general"
	InvoiceTypeCompanyVATSpecial = "company_vat_special"
	InvoiceTypePersonal          = "personal"

	InvoiceMinAmount = 500.0
	InvoiceTaxRate   = 0.02
)

type InvoiceRequest struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	UserEmail     string     `json:"user_email"`
	UserName      string     `json:"user_name"`
	Status        string     `json:"status"`
	InvoiceType   string     `json:"invoice_type"`
	Title         string     `json:"title"`
	TaxID         string     `json:"tax_id"`
	ItemName      string     `json:"item_name"`
	Amount        float64    `json:"amount"`
	TaxRate       float64    `json:"tax_rate"`
	TaxFee        float64    `json:"tax_fee"`
	ReceiverEmail string     `json:"receiver_email"`
	Note          string     `json:"note"`
	AdminNote     string     `json:"admin_note"`
	InvoiceNo     string     `json:"invoice_no"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	RejectedAt    *time.Time `json:"rejected_at,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	ProcessedBy   *int64     `json:"processed_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type InvoiceTemplate struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Name          string    `json:"name"`
	InvoiceType   string    `json:"invoice_type"`
	Title         string    `json:"title"`
	TaxID         string    `json:"tax_id"`
	ItemName      string    `json:"item_name"`
	ReceiverEmail string    `json:"receiver_email"`
	Note          string    `json:"note"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type InvoiceSummary struct {
	RechargeAmount   float64 `json:"recharge_amount"`
	InvoicedAmount   float64 `json:"invoiced_amount"`
	LockedAmount     float64 `json:"locked_amount"`
	AvailableAmount  float64 `json:"available_amount"`
	MinAmount        float64 `json:"min_amount"`
	TaxRate          float64 `json:"tax_rate"`
	TaxRatePercent   float64 `json:"tax_rate_percent"`
	CanApply         bool    `json:"can_apply"`
	CurrentBalance   float64 `json:"current_balance"`
	InvoiceableBasis string  `json:"invoiceable_basis"`
}

type CreateInvoiceRequestInput struct {
	UserID        int64
	InvoiceType   string
	Title         string
	TaxID         string
	ItemName      string
	Amount        float64
	ReceiverEmail string
	Note          string
}

type SaveInvoiceTemplateInput struct {
	UserID        int64
	Name          string
	InvoiceType   string
	Title         string
	TaxID         string
	ItemName      string
	ReceiverEmail string
	Note          string
	IsDefault     bool
}

type InvoiceListFilters struct {
	UserID int64
	Status string
	Search string
}

type ApproveInvoiceInput struct {
	AdminID   int64
	AdminNote string
}

type RejectInvoiceInput struct {
	AdminID   int64
	AdminNote string
}

type CompleteInvoiceInput struct {
	AdminID   int64
	InvoiceNo string
	AdminNote string
}

type InvoiceRepository interface {
	GetSummary(ctx context.Context, userID int64) (*InvoiceSummary, error)
	Create(ctx context.Context, input CreateInvoiceRequestInput) (*InvoiceRequest, error)
	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]InvoiceRequest, *pagination.PaginationResult, error)
	ListForAdmin(ctx context.Context, params pagination.PaginationParams, filters InvoiceListFilters) ([]InvoiceRequest, *pagination.PaginationResult, error)
	GetForUser(ctx context.Context, id, userID int64) (*InvoiceRequest, error)
	GetForAdmin(ctx context.Context, id int64) (*InvoiceRequest, error)
	Approve(ctx context.Context, id int64, input ApproveInvoiceInput) (*InvoiceRequest, error)
	Reject(ctx context.Context, id int64, input RejectInvoiceInput) (*InvoiceRequest, error)
	Cancel(ctx context.Context, id, userID int64) (*InvoiceRequest, error)
	Complete(ctx context.Context, id int64, input CompleteInvoiceInput) (*InvoiceRequest, error)
	ListTemplates(ctx context.Context, userID int64) ([]InvoiceTemplate, error)
	CreateTemplate(ctx context.Context, input SaveInvoiceTemplateInput) (*InvoiceTemplate, error)
	UpdateTemplate(ctx context.Context, id, userID int64, input SaveInvoiceTemplateInput) (*InvoiceTemplate, error)
	DeleteTemplate(ctx context.Context, id, userID int64) error
	SetDefaultTemplate(ctx context.Context, id, userID int64) (*InvoiceTemplate, error)
}
