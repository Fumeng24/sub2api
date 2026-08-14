package service

import (
	"context"
	"fmt"
	"math"
	"net/mail"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrInvoiceNotFound            = infraerrors.NotFound("INVOICE_NOT_FOUND", "invoice request not found")
	ErrInvoiceInvalidStatus       = infraerrors.BadRequest("INVOICE_INVALID_STATUS", "invoice request status is invalid for this operation")
	ErrInvoiceAmountTooSmall      = infraerrors.BadRequest("INVOICE_AMOUNT_TOO_SMALL", "invoice amount must be at least 100")
	ErrInvoiceAmountUnavailable   = infraerrors.BadRequest("INVOICE_AMOUNT_UNAVAILABLE", "invoice amount exceeds available invoice amount")
	ErrInvoiceBalanceInsufficient = infraerrors.BadRequest("INVOICE_BALANCE_INSUFFICIENT", "user balance is insufficient for invoice tax fee")
	ErrInvoiceTemplateNotFound    = infraerrors.NotFound("INVOICE_TEMPLATE_NOT_FOUND", "invoice template not found")
	ErrInvoiceTemplateNameTaken   = infraerrors.Conflict("INVOICE_TEMPLATE_NAME_TAKEN", "invoice template name already exists")
)

type InvoiceService struct {
	repo InvoiceRepository
}

func NewInvoiceService(repo InvoiceRepository) *InvoiceService {
	return &InvoiceService{repo: repo}
}

func (s *InvoiceService) GetSummary(ctx context.Context, userID int64) (*InvoiceSummary, error) {
	return s.repo.GetSummary(ctx, userID)
}

func (s *InvoiceService) Create(ctx context.Context, input CreateInvoiceRequestInput) (*InvoiceRequest, error) {
	normalized, err := normalizeCreateInvoiceInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, normalized)
}

func (s *InvoiceService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return s.repo.ListByUser(ctx, userID, normalizeInvoicePagination(params, "created_at", pagination.SortOrderDesc))
}

func (s *InvoiceService) ListForAdmin(ctx context.Context, params pagination.PaginationParams, filters InvoiceListFilters) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	filters.Status = strings.TrimSpace(strings.ToLower(filters.Status))
	filters.Search = strings.TrimSpace(filters.Search)
	if len(filters.Search) > 200 {
		filters.Search = filters.Search[:200]
	}
	return s.repo.ListForAdmin(ctx, normalizeInvoicePagination(params, "created_at", pagination.SortOrderDesc), filters)
}

func (s *InvoiceService) GetForUser(ctx context.Context, id, userID int64) (*InvoiceRequest, error) {
	return s.repo.GetForUser(ctx, id, userID)
}

func (s *InvoiceService) GetForAdmin(ctx context.Context, id int64) (*InvoiceRequest, error) {
	return s.repo.GetForAdmin(ctx, id)
}

func (s *InvoiceService) Approve(ctx context.Context, id int64, input ApproveInvoiceInput) (*InvoiceRequest, error) {
	input.AdminNote = trimInvoiceText(input.AdminNote, 2000)
	return s.repo.Approve(ctx, id, input)
}

func (s *InvoiceService) Reject(ctx context.Context, id int64, input RejectInvoiceInput) (*InvoiceRequest, error) {
	input.AdminNote = trimInvoiceText(input.AdminNote, 2000)
	if strings.TrimSpace(input.AdminNote) == "" {
		return nil, infraerrors.BadRequest("INVOICE_REJECT_REASON_REQUIRED", "reject reason is required")
	}
	return s.repo.Reject(ctx, id, input)
}

func (s *InvoiceService) Cancel(ctx context.Context, id, userID int64) (*InvoiceRequest, error) {
	return s.repo.Cancel(ctx, id, userID)
}

func (s *InvoiceService) Complete(ctx context.Context, id int64, input CompleteInvoiceInput) (*InvoiceRequest, error) {
	input.InvoiceNo = trimInvoiceText(input.InvoiceNo, 128)
	input.AdminNote = trimInvoiceText(input.AdminNote, 2000)
	return s.repo.Complete(ctx, id, input)
}

func (s *InvoiceService) ListTemplates(ctx context.Context, userID int64) ([]InvoiceTemplate, error) {
	return s.repo.ListTemplates(ctx, userID)
}

func (s *InvoiceService) CreateTemplate(ctx context.Context, input SaveInvoiceTemplateInput) (*InvoiceTemplate, error) {
	normalized, err := normalizeSaveInvoiceTemplateInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateTemplate(ctx, normalized)
}

func (s *InvoiceService) UpdateTemplate(ctx context.Context, id, userID int64, input SaveInvoiceTemplateInput) (*InvoiceTemplate, error) {
	normalized, err := normalizeSaveInvoiceTemplateInput(input)
	if err != nil {
		return nil, err
	}
	normalized.UserID = userID
	return s.repo.UpdateTemplate(ctx, id, userID, normalized)
}

func (s *InvoiceService) DeleteTemplate(ctx context.Context, id, userID int64) error {
	return s.repo.DeleteTemplate(ctx, id, userID)
}

func (s *InvoiceService) SetDefaultTemplate(ctx context.Context, id, userID int64) (*InvoiceTemplate, error) {
	return s.repo.SetDefaultTemplate(ctx, id, userID)
}

func normalizeCreateInvoiceInput(input CreateInvoiceRequestInput) (CreateInvoiceRequestInput, error) {
	input.InvoiceType = strings.TrimSpace(input.InvoiceType)
	if input.InvoiceType == "" {
		input.InvoiceType = InvoiceTypeCompanyVATGeneral
	}
	if !isValidInvoiceType(input.InvoiceType) {
		return input, infraerrors.BadRequest("INVOICE_TYPE_INVALID", "invoice type is invalid")
	}
	input.Title = trimInvoiceText(input.Title, 255)
	input.TaxID = strings.ToUpper(trimInvoiceText(input.TaxID, 100))
	input.ItemName = trimInvoiceText(input.ItemName, 100)
	input.ReceiverEmail = trimInvoiceText(input.ReceiverEmail, 255)
	input.Note = trimInvoiceText(input.Note, 2000)
	input.Amount = roundMoney(input.Amount)

	if input.Title == "" {
		return input, infraerrors.BadRequest("INVOICE_TITLE_REQUIRED", "invoice title is required")
	}
	if input.ItemName == "" {
		return input, infraerrors.BadRequest("INVOICE_ITEM_REQUIRED", "invoice item is required")
	}
	if input.ReceiverEmail == "" {
		return input, infraerrors.BadRequest("INVOICE_RECEIVER_EMAIL_REQUIRED", "receiver email is required")
	}
	if _, err := mail.ParseAddress(input.ReceiverEmail); err != nil {
		return input, infraerrors.BadRequest("INVOICE_RECEIVER_EMAIL_INVALID", "receiver email is invalid")
	}
	if input.InvoiceType != InvoiceTypePersonal && input.TaxID == "" {
		return input, infraerrors.BadRequest("INVOICE_TAX_ID_REQUIRED", "tax ID is required for company invoices")
	}
	if input.Amount < InvoiceMinAmount {
		return input, ErrInvoiceAmountTooSmall.WithMetadata(map[string]string{"min_amount": fmt.Sprintf("%.2f", InvoiceMinAmount)})
	}
	return input, nil
}

func normalizeSaveInvoiceTemplateInput(input SaveInvoiceTemplateInput) (SaveInvoiceTemplateInput, error) {
	input.Name = trimInvoiceText(input.Name, 80)
	input.InvoiceType = strings.TrimSpace(input.InvoiceType)
	if input.InvoiceType == "" {
		input.InvoiceType = InvoiceTypeCompanyVATGeneral
	}
	if !isValidInvoiceType(input.InvoiceType) {
		return input, infraerrors.BadRequest("INVOICE_TYPE_INVALID", "invoice type is invalid")
	}
	input.Title = trimInvoiceText(input.Title, 255)
	input.TaxID = strings.ToUpper(trimInvoiceText(input.TaxID, 100))
	input.ItemName = trimInvoiceText(input.ItemName, 100)
	input.ReceiverEmail = trimInvoiceText(input.ReceiverEmail, 255)
	input.Note = trimInvoiceText(input.Note, 2000)

	if input.Name == "" {
		input.Name = defaultInvoiceTemplateName(input)
	}
	if input.Title == "" {
		return input, infraerrors.BadRequest("INVOICE_TITLE_REQUIRED", "invoice title is required")
	}
	if input.ItemName == "" {
		return input, infraerrors.BadRequest("INVOICE_ITEM_REQUIRED", "invoice item is required")
	}
	if input.ReceiverEmail == "" {
		return input, infraerrors.BadRequest("INVOICE_RECEIVER_EMAIL_REQUIRED", "receiver email is required")
	}
	if _, err := mail.ParseAddress(input.ReceiverEmail); err != nil {
		return input, infraerrors.BadRequest("INVOICE_RECEIVER_EMAIL_INVALID", "receiver email is invalid")
	}
	if input.InvoiceType != InvoiceTypePersonal && input.TaxID == "" {
		return input, infraerrors.BadRequest("INVOICE_TAX_ID_REQUIRED", "tax ID is required for company invoices")
	}
	return input, nil
}

func defaultInvoiceTemplateName(input SaveInvoiceTemplateInput) string {
	if input.InvoiceType == InvoiceTypePersonal {
		return "个人发票"
	}
	if input.Title != "" {
		return trimInvoiceText(input.Title, 80)
	}
	return "默认模板"
}

func CalculateInvoiceTaxFee(amount float64) float64 {
	amount = roundMoney(amount)
	if amount <= 0 {
		return 0
	}
	fee := roundMoney(amount * InvoiceTaxRate)
	if fee < InvoiceMinTaxFee {
		fee = InvoiceMinTaxFee
	}
	return roundMoney(fee)
}

func normalizeInvoicePagination(params pagination.PaginationParams, defaultSortBy, defaultSortOrder string) pagination.PaginationParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 1000 {
		params.PageSize = 1000
	}
	params.SortBy = strings.TrimSpace(params.SortBy)
	if params.SortBy == "" {
		params.SortBy = defaultSortBy
	}
	params.SortOrder = pagination.NormalizeSortOrder(params.SortOrder, defaultSortOrder)
	return params
}

func isValidInvoiceType(t string) bool {
	switch t {
	case InvoiceTypeCompanyVATGeneral, InvoiceTypeCompanyVATSpecial, InvoiceTypePersonal:
		return true
	default:
		return false
	}
}

func trimInvoiceText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if maxLen > 0 && len([]rune(s)) > maxLen {
		runes := []rune(s)
		return string(runes[:maxLen])
	}
	return s
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
