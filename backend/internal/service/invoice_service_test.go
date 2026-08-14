package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type invoiceRepoStub struct {
	createInput   CreateInvoiceRequestInput
	createErr     error
	templateInput SaveInvoiceTemplateInput
}

func (s *invoiceRepoStub) GetSummary(context.Context, int64) (*InvoiceSummary, error) {
	return nil, nil
}

func (s *invoiceRepoStub) Create(_ context.Context, input CreateInvoiceRequestInput) (*InvoiceRequest, error) {
	s.createInput = input
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &InvoiceRequest{ID: 1, Amount: input.Amount, InvoiceType: input.InvoiceType}, nil
}

func (s *invoiceRepoStub) ListSourceOrders(context.Context, int64, []int64) ([]InvoiceSourceOrder, error) {
	return nil, nil
}

func (s *invoiceRepoStub) ListByUser(context.Context, int64, pagination.PaginationParams) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *invoiceRepoStub) ListForAdmin(context.Context, pagination.PaginationParams, InvoiceListFilters) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *invoiceRepoStub) GetForUser(context.Context, int64, int64) (*InvoiceRequest, error) {
	return nil, nil
}

func (s *invoiceRepoStub) GetForAdmin(context.Context, int64) (*InvoiceRequest, error) {
	return nil, nil
}

func (s *invoiceRepoStub) Approve(context.Context, int64, ApproveInvoiceInput) (*InvoiceRequest, error) {
	return nil, nil
}

func (s *invoiceRepoStub) Reject(context.Context, int64, RejectInvoiceInput) (*InvoiceRequest, error) {
	return nil, nil
}

func (s *invoiceRepoStub) Cancel(context.Context, int64, int64) (*InvoiceRequest, error) {
	return nil, nil
}

func (s *invoiceRepoStub) Complete(context.Context, int64, CompleteInvoiceInput) (*InvoiceRequest, error) {
	return nil, nil
}

func (s *invoiceRepoStub) ListTemplates(context.Context, int64) ([]InvoiceTemplate, error) {
	return nil, nil
}

func (s *invoiceRepoStub) CreateTemplate(_ context.Context, input SaveInvoiceTemplateInput) (*InvoiceTemplate, error) {
	s.templateInput = input
	return &InvoiceTemplate{ID: 1, Name: input.Name, InvoiceType: input.InvoiceType}, nil
}

func (s *invoiceRepoStub) UpdateTemplate(_ context.Context, _ int64, _ int64, input SaveInvoiceTemplateInput) (*InvoiceTemplate, error) {
	s.templateInput = input
	return &InvoiceTemplate{ID: 1, Name: input.Name, InvoiceType: input.InvoiceType}, nil
}

func (s *invoiceRepoStub) DeleteTemplate(context.Context, int64, int64) error {
	return nil
}

func (s *invoiceRepoStub) SetDefaultTemplate(context.Context, int64, int64) (*InvoiceTemplate, error) {
	return nil, nil
}

func TestInvoiceServiceCreateNormalizesAndValidatesInput(t *testing.T) {
	t.Parallel()

	repo := &invoiceRepoStub{}
	svc := NewInvoiceService(repo)

	got, err := svc.Create(context.Background(), CreateInvoiceRequestInput{
		UserID:        42,
		InvoiceType:   "",
		Title:         "  Wegoo AI  ",
		TaxID:         "  abcd1234  ",
		ItemName:      "  信息技术服务费  ",
		Amount:        100.005,
		ReceiverEmail: " billing@example.com ",
		Note:          "  please issue e-invoice  ",
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), got.ID)
	require.Equal(t, InvoiceTypeCompanyVATGeneral, repo.createInput.InvoiceType)
	require.Equal(t, "Wegoo AI", repo.createInput.Title)
	require.Equal(t, "ABCD1234", repo.createInput.TaxID)
	require.Equal(t, "信息技术服务费", repo.createInput.ItemName)
	require.Equal(t, 100.01, repo.createInput.Amount)
	require.Equal(t, "billing@example.com", repo.createInput.ReceiverEmail)
	require.Equal(t, "please issue e-invoice", repo.createInput.Note)
}

func TestInvoiceServiceCreateAllowsPersonalInvoiceWithoutTaxID(t *testing.T) {
	t.Parallel()

	repo := &invoiceRepoStub{}
	svc := NewInvoiceService(repo)

	_, err := svc.Create(context.Background(), CreateInvoiceRequestInput{
		UserID:        42,
		InvoiceType:   InvoiceTypePersonal,
		Title:         "Alice",
		ItemName:      "信息技术服务费",
		Amount:        100,
		ReceiverEmail: "alice@example.com",
	})

	require.NoError(t, err)
	require.Empty(t, repo.createInput.TaxID)
}

func TestInvoiceServiceCreateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input CreateInvoiceRequestInput
		want  error
		code  string
	}{
		{
			name: "company_tax_id_required",
			input: CreateInvoiceRequestInput{
				InvoiceType:   InvoiceTypeCompanyVATGeneral,
				Title:         "Company",
				ItemName:      "信息技术服务费",
				Amount:        100,
				ReceiverEmail: "billing@example.com",
			},
			code: "INVOICE_TAX_ID_REQUIRED",
		},
		{
			name: "minimum_amount",
			input: CreateInvoiceRequestInput{
				InvoiceType:   InvoiceTypePersonal,
				Title:         "Alice",
				ItemName:      "信息技术服务费",
				Amount:        99.99,
				ReceiverEmail: "alice@example.com",
			},
			code: "INVOICE_AMOUNT_TOO_SMALL",
		},
		{
			name: "receiver_email_invalid",
			input: CreateInvoiceRequestInput{
				InvoiceType:   InvoiceTypePersonal,
				Title:         "Alice",
				ItemName:      "信息技术服务费",
				Amount:        100,
				ReceiverEmail: "not-an-email",
			},
			code: "INVOICE_RECEIVER_EMAIL_INVALID",
		},
		{
			name: "invoice_type_invalid",
			input: CreateInvoiceRequestInput{
				InvoiceType:   "unknown",
				Title:         "Alice",
				ItemName:      "信息技术服务费",
				Amount:        100,
				ReceiverEmail: "alice@example.com",
			},
			code: "INVOICE_TYPE_INVALID",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewInvoiceService(&invoiceRepoStub{})
			_, err := svc.Create(context.Background(), tt.input)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.code)
		})
	}
}

func TestCalculateInvoiceTaxFeeUsesFlatRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		amount float64
		want   float64
	}{
		{name: "zero", amount: 0, want: 0},
		{name: "minimum_invoice", amount: 100, want: 3},
		{name: "standard", amount: 500, want: 15},
		{name: "large", amount: 3000, want: 90},
		{name: "rounded", amount: 3333.335, want: 100},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, CalculateInvoiceTaxFee(tt.amount))
		})
	}
}

func TestInvoiceServiceRejectRequiresReason(t *testing.T) {
	t.Parallel()

	svc := NewInvoiceService(&invoiceRepoStub{})
	_, err := svc.Reject(context.Background(), 1, RejectInvoiceInput{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "INVOICE_REJECT_REASON_REQUIRED")
}

func TestInvoiceServiceCreateTemplateNormalizesInput(t *testing.T) {
	t.Parallel()

	repo := &invoiceRepoStub{}
	svc := NewInvoiceService(repo)

	got, err := svc.CreateTemplate(context.Background(), SaveInvoiceTemplateInput{
		UserID:        42,
		Name:          "  默认模板  ",
		InvoiceType:   "",
		Title:         "  Wegoo AI  ",
		TaxID:         "  abcd1234  ",
		ItemName:      "  信息技术服务费  ",
		ReceiverEmail: " billing@example.com ",
		Note:          "  remark  ",
		IsDefault:     true,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), got.ID)
	require.Equal(t, "默认模板", repo.templateInput.Name)
	require.Equal(t, InvoiceTypeCompanyVATGeneral, repo.templateInput.InvoiceType)
	require.Equal(t, "Wegoo AI", repo.templateInput.Title)
	require.Equal(t, "ABCD1234", repo.templateInput.TaxID)
	require.Equal(t, "信息技术服务费", repo.templateInput.ItemName)
	require.Equal(t, "billing@example.com", repo.templateInput.ReceiverEmail)
	require.Equal(t, "remark", repo.templateInput.Note)
	require.True(t, repo.templateInput.IsDefault)
}
