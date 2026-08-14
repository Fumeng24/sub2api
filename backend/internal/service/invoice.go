package service

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
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

	InvoiceMinAmount       = 100.0
	InvoiceTaxRate         = 0.03
	InvoiceMinTaxFee       = 0.0
	InvoiceTaxFeeThreshold = 0.0
)

type InvoiceRequest struct {
	ID               int64               `json:"id"`
	UserID           int64               `json:"user_id"`
	UserEmail        string              `json:"user_email"`
	UserName         string              `json:"user_name"`
	Status           string              `json:"status"`
	InvoiceType      string              `json:"invoice_type"`
	Title            string              `json:"title"`
	TaxID            string              `json:"tax_id"`
	ItemName         string              `json:"item_name"`
	Amount           float64             `json:"amount"`
	TaxRate          float64             `json:"tax_rate"`
	TaxFee           float64             `json:"tax_fee"`
	ReceiverEmail    string              `json:"receiver_email"`
	Note             string              `json:"note"`
	AdminNote        string              `json:"admin_note"`
	InvoiceNo        string              `json:"invoice_no"`
	SourceOrderCount int                 `json:"source_order_count,omitempty"`
	SourceOrdersJSON InvoiceSourceOrders `json:"source_orders_json,omitempty"`
	CompletedAt      *time.Time          `json:"completed_at,omitempty"`
	RejectedAt       *time.Time          `json:"rejected_at,omitempty"`
	ApprovedAt       *time.Time          `json:"approved_at,omitempty"`
	ProcessedBy      *int64              `json:"processed_by,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type InvoiceSourceOrder struct {
	ID               int64   `json:"id"`
	RecordSource     string  `json:"record_source"`
	BusinessCategory string  `json:"business_category"`
	PaymentType      string  `json:"payment_type"`
	OutTradeNo       string  `json:"out_trade_no"`
	Amount           float64 `json:"amount"`
	RefundAmount     float64 `json:"refund_amount"`
	Invoiceable      bool    `json:"invoiceable"`
}

type InvoiceSourceOrders []InvoiceSourceOrder

func (o *InvoiceSourceOrders) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*o = nil
		return nil
	case []byte:
		return scanInvoiceSourceOrdersJSON(v, o)
	case string:
		return scanInvoiceSourceOrdersJSON([]byte(v), o)
	default:
		return fmt.Errorf("unsupported invoice source orders type %T", src)
	}
}

func scanInvoiceSourceOrdersJSON(raw []byte, out *InvoiceSourceOrders) error {
	var structured []InvoiceSourceOrder
	if err := json.Unmarshal(raw, &structured); err == nil {
		*out = structured
		return nil
	}
	var legacyIDs []int64
	if err := json.Unmarshal(raw, &legacyIDs); err != nil {
		return err
	}
	legacy := make([]InvoiceSourceOrder, 0, len(legacyIDs))
	for _, id := range legacyIDs {
		legacy = append(legacy, InvoiceSourceOrder{ID: id})
	}
	*out = legacy
	return nil
}

func (o InvoiceSourceOrders) Value() (driver.Value, error) {
	if o == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]InvoiceSourceOrder(o))
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
	MinTaxFee        float64 `json:"min_tax_fee"`
	TaxFeeThreshold  float64 `json:"tax_fee_threshold"`
	CanApply         bool    `json:"can_apply"`
	CurrentBalance   float64 `json:"current_balance"`
	InvoiceableBasis string  `json:"invoiceable_basis"`
}

type CreateInvoiceRequestInput struct {
	UserID         int64
	InvoiceType    string
	Title          string
	TaxID          string
	ItemName       string
	Amount         float64
	ReceiverEmail  string
	Note           string
	SourceOrderIDs []int64
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
	ListSourceOrders(ctx context.Context, userID int64, sourceOrderIDs []int64) ([]InvoiceSourceOrder, error)
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
