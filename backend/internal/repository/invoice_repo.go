package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type invoiceRepository struct {
	db                      *sql.DB
	authCacheInvalidator    service.APIKeyAuthCacheInvalidator
	billingCacheInvalidator interface {
		InvalidateUserBalance(context.Context, int64) error
	}
}

func NewInvoiceRepository(db *sql.DB, authCacheInvalidator service.APIKeyAuthCacheInvalidator, billingCacheService *service.BillingCacheService) service.InvoiceRepository {
	return &invoiceRepository{
		db:                      db,
		authCacheInvalidator:    authCacheInvalidator,
		billingCacheInvalidator: billingCacheService,
	}
}

func (r *invoiceRepository) GetSummary(ctx context.Context, userID int64) (*service.InvoiceSummary, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice repository db is nil")
	}
	var s service.InvoiceSummary
	err := r.db.QueryRowContext(ctx, invoiceSummarySQL, userID).Scan(
		&s.RechargeAmount,
		&s.InvoicedAmount,
		&s.LockedAmount,
		&s.AvailableAmount,
		&s.CurrentBalance,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invoice summary: %w", err)
	}
	return normalizeInvoiceSummary(&s), nil
}

func (r *invoiceRepository) Create(ctx context.Context, input service.CreateInvoiceRequestInput) (*service.InvoiceRequest, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var lockedUserID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, input.UserID).Scan(&lockedUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock invoice user for create: %w", err)
	}

	var summary service.InvoiceSummary
	err = tx.QueryRowContext(ctx, invoiceSummarySQL, input.UserID).Scan(
		&summary.RechargeAmount,
		&summary.InvoicedAmount,
		&summary.LockedAmount,
		&summary.AvailableAmount,
		&summary.CurrentBalance,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invoice summary for create: %w", err)
	}
	normalizeInvoiceSummary(&summary)
	if input.Amount > summary.AvailableAmount+0.000001 {
		return nil, service.ErrInvoiceAmountUnavailable.WithMetadata(map[string]string{
			"available_amount": fmt.Sprintf("%.2f", summary.AvailableAmount),
		})
	}

	inv := &service.InvoiceRequest{}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO invoice_requests (
			user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, created_at, updated_at
		)
		SELECT
			u.id, u.email, u.username, $2, $3, $4, $5, $6,
			$7, $8, 0, $9, $10, NOW(), NOW()
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
		RETURNING id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
	`, input.UserID, service.InvoiceStatusPending, input.InvoiceType, input.Title, input.TaxID, input.ItemName,
		input.Amount, service.InvoiceTaxRate, input.ReceiverEmail, input.Note).Scan(invoiceScanDest(inv)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("create invoice request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *invoiceRepository) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.InvoiceRequest, *pagination.PaginationResult, error) {
	filters := InvoiceQueryFilters{UserID: userID}
	return r.list(ctx, params, filters, false)
}

func (r *invoiceRepository) ListForAdmin(ctx context.Context, params pagination.PaginationParams, filters service.InvoiceListFilters) ([]service.InvoiceRequest, *pagination.PaginationResult, error) {
	return r.list(ctx, params, InvoiceQueryFilters{
		UserID: filters.UserID,
		Status: filters.Status,
		Search: filters.Search,
	}, true)
}

func (r *invoiceRepository) GetForUser(ctx context.Context, id, userID int64) (*service.InvoiceRequest, error) {
	return r.get(ctx, "id = $1 AND user_id = $2", id, userID)
}

func (r *invoiceRepository) GetForAdmin(ctx context.Context, id int64) (*service.InvoiceRequest, error) {
	return r.get(ctx, "id = $1", id)
}

func (r *invoiceRepository) Approve(ctx context.Context, id int64, input service.ApproveInvoiceInput) (*service.InvoiceRequest, error) {
	inv := &service.InvoiceRequest{}
	err := r.db.QueryRowContext(ctx, `
		UPDATE invoice_requests
		SET status = $2,
			admin_note = $3,
			processed_by = $4,
			approved_at = COALESCE(approved_at, NOW()),
			updated_at = NOW()
		WHERE id = $1 AND status = $5
		RETURNING id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
	`, id, service.InvoiceStatusApproved, input.AdminNote, nullableAdminID(input.AdminID), service.InvoiceStatusPending).Scan(invoiceScanDest(inv)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.invoiceStatusOrNotFound(ctx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("approve invoice request: %w", err)
	}
	return inv, nil
}

func (r *invoiceRepository) Reject(ctx context.Context, id int64, input service.RejectInvoiceInput) (*service.InvoiceRequest, error) {
	inv := &service.InvoiceRequest{}
	err := r.db.QueryRowContext(ctx, `
		UPDATE invoice_requests
		SET status = $2,
			admin_note = $3,
			processed_by = $4,
			rejected_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND status IN ($5, $6)
		RETURNING id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
	`, id, service.InvoiceStatusRejected, input.AdminNote, nullableAdminID(input.AdminID), service.InvoiceStatusPending, service.InvoiceStatusApproved).Scan(invoiceScanDest(inv)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.invoiceStatusOrNotFound(ctx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("reject invoice request: %w", err)
	}
	return inv, nil
}

func (r *invoiceRepository) Cancel(ctx context.Context, id, userID int64) (*service.InvoiceRequest, error) {
	inv := &service.InvoiceRequest{}
	err := r.db.QueryRowContext(ctx, `
		UPDATE invoice_requests
		SET status = $3,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = $4
		RETURNING id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
	`, id, userID, service.InvoiceStatusCancelled, service.InvoiceStatusPending).Scan(invoiceScanDest(inv)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.invoiceStatusOrNotFoundForUser(ctx, id, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("cancel invoice request: %w", err)
	}
	return inv, nil
}

func (r *invoiceRepository) Complete(ctx context.Context, id int64, input service.CompleteInvoiceInput) (*service.InvoiceRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	inv := &service.InvoiceRequest{}
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
		FROM invoice_requests
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(invoiceScanDest(inv)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock invoice request: %w", err)
	}
	if inv.Status != service.InvoiceStatusPending && inv.Status != service.InvoiceStatusApproved {
		return nil, service.ErrInvoiceInvalidStatus.WithMetadata(map[string]string{"status": inv.Status})
	}

	var currentBalance float64
	err = tx.QueryRowContext(ctx, `
		SELECT balance
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, inv.UserID).Scan(&currentBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock invoice user: %w", err)
	}

	taxFee := roundMoneyRepo(inv.Amount * service.InvoiceTaxRate)
	if currentBalance+0.000001 < taxFee {
		return nil, service.ErrInvoiceBalanceInsufficient.WithMetadata(map[string]string{
			"current_balance": fmt.Sprintf("%.2f", currentBalance),
			"tax_fee":         fmt.Sprintf("%.2f", taxFee),
		})
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, taxFee, inv.UserID); err != nil {
		return nil, fmt.Errorf("deduct invoice tax fee: %w", err)
	}
	if err := r.insertInvoiceTaxBalanceHistory(ctx, tx, inv, taxFee, input); err != nil {
		return nil, err
	}

	completed := &service.InvoiceRequest{}
	err = tx.QueryRowContext(ctx, `
		UPDATE invoice_requests
		SET status = $2,
			tax_rate = $3,
			tax_fee = $4,
			invoice_no = $5,
			admin_note = $6,
			processed_by = $7,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
	`, id, service.InvoiceStatusCompleted, service.InvoiceTaxRate, taxFee, input.InvoiceNo, input.AdminNote, nullableAdminID(input.AdminID)).Scan(invoiceScanDest(completed)...)
	if err != nil {
		return nil, fmt.Errorf("complete invoice request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	r.invalidateUserBillingCaches(ctx, completed.UserID)
	return completed, nil
}

func (r *invoiceRepository) ListTemplates(ctx context.Context, userID int64) ([]service.InvoiceTemplate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice repository db is nil")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, invoice_type, title, tax_id, item_name,
			receiver_email, note, is_default, created_at, updated_at
		FROM invoice_templates
		WHERE user_id = $1
		ORDER BY is_default DESC, updated_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list invoice templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.InvoiceTemplate, 0)
	for rows.Next() {
		var tmpl service.InvoiceTemplate
		if err := rows.Scan(invoiceTemplateScanDest(&tmpl)...); err != nil {
			return nil, err
		}
		out = append(out, tmpl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *invoiceRepository) CreateTemplate(ctx context.Context, input service.SaveInvoiceTemplateInput) (*service.InvoiceTemplate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM invoice_templates
		WHERE user_id = $1
	`, input.UserID).Scan(&count); err != nil {
		return nil, fmt.Errorf("count invoice templates: %w", err)
	}
	isDefault := input.IsDefault || count == 0
	if isDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE invoice_templates SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, input.UserID); err != nil {
			return nil, fmt.Errorf("unset invoice template default: %w", err)
		}
	}

	tmpl := &service.InvoiceTemplate{}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO invoice_templates (
			user_id, name, invoice_type, title, tax_id, item_name, receiver_email, note,
			is_default, created_at, updated_at
		)
		SELECT u.id, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
		RETURNING id, user_id, name, invoice_type, title, tax_id, item_name,
			receiver_email, note, is_default, created_at, updated_at
	`, input.UserID, input.Name, input.InvoiceType, input.Title, input.TaxID, input.ItemName,
		input.ReceiverEmail, input.Note, isDefault).Scan(invoiceTemplateScanDest(tmpl)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		if isInvoiceTemplateNameConflict(err) {
			return nil, service.ErrInvoiceTemplateNameTaken
		}
		return nil, fmt.Errorf("create invoice template: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func (r *invoiceRepository) UpdateTemplate(ctx context.Context, id, userID int64, input service.SaveInvoiceTemplateInput) (*service.InvoiceTemplate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM invoice_templates
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, id, userID).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrInvoiceTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock invoice template: %w", err)
	}
	if input.IsDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE invoice_templates SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
			return nil, fmt.Errorf("unset invoice template default: %w", err)
		}
	}

	tmpl := &service.InvoiceTemplate{}
	err = tx.QueryRowContext(ctx, `
		UPDATE invoice_templates
		SET name = $3,
			invoice_type = $4,
			title = $5,
			tax_id = $6,
			item_name = $7,
			receiver_email = $8,
			note = $9,
			is_default = $10,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, invoice_type, title, tax_id, item_name,
			receiver_email, note, is_default, created_at, updated_at
	`, id, userID, input.Name, input.InvoiceType, input.Title, input.TaxID, input.ItemName,
		input.ReceiverEmail, input.Note, input.IsDefault).Scan(invoiceTemplateScanDest(tmpl)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrInvoiceTemplateNotFound
	}
	if err != nil {
		if isInvoiceTemplateNameConflict(err) {
			return nil, service.ErrInvoiceTemplateNameTaken
		}
		return nil, fmt.Errorf("update invoice template: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func (r *invoiceRepository) DeleteTemplate(ctx context.Context, id, userID int64) error {
	if r == nil || r.db == nil {
		return errors.New("invoice repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var wasDefault bool
	err = tx.QueryRowContext(ctx, `
		DELETE FROM invoice_templates
		WHERE id = $1 AND user_id = $2
		RETURNING is_default
	`, id, userID).Scan(&wasDefault)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrInvoiceTemplateNotFound
	}
	if err != nil {
		return fmt.Errorf("delete invoice template: %w", err)
	}
	if wasDefault {
		if _, err := tx.ExecContext(ctx, `
			WITH next_template AS (
				SELECT id
				FROM invoice_templates
				WHERE user_id = $1
				ORDER BY updated_at DESC, id DESC
				LIMIT 1
			)
			UPDATE invoice_templates
			SET is_default = TRUE, updated_at = NOW()
			WHERE id IN (SELECT id FROM next_template)
		`, userID); err != nil {
			return fmt.Errorf("promote next invoice template: %w", err)
		}
	}
	return tx.Commit()
}

func (r *invoiceRepository) SetDefaultTemplate(ctx context.Context, id, userID int64) (*service.InvoiceTemplate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("invoice repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM invoice_templates
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, id, userID).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrInvoiceTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock invoice template: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invoice_templates SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
		return nil, fmt.Errorf("unset invoice template default: %w", err)
	}

	tmpl := &service.InvoiceTemplate{}
	err = tx.QueryRowContext(ctx, `
		UPDATE invoice_templates
		SET is_default = TRUE,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, invoice_type, title, tax_id, item_name,
			receiver_email, note, is_default, created_at, updated_at
	`, id, userID).Scan(invoiceTemplateScanDest(tmpl)...)
	if err != nil {
		return nil, fmt.Errorf("set invoice template default: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tmpl, nil
}

type InvoiceQueryFilters struct {
	UserID int64
	Status string
	Search string
}

func (r *invoiceRepository) list(ctx context.Context, params pagination.PaginationParams, filters InvoiceQueryFilters, admin bool) ([]service.InvoiceRequest, *pagination.PaginationResult, error) {
	where, args := buildInvoiceWhere(filters)
	whereClause := strings.Join(where, " AND ")
	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM invoice_requests WHERE %s`, whereClause)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count invoice requests: %w", err)
	}

	orderBy := invoiceOrderBy(params, admin)
	pageSize := params.Limit()
	page := params.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
		FROM invoice_requests
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list invoice requests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.InvoiceRequest, 0, pageSize)
	for rows.Next() {
		var inv service.InvoiceRequest
		if err := rows.Scan(invoiceScanDest(&inv)...); err != nil {
			return nil, nil, err
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return out, &pagination.PaginationResult{Total: total, Page: page, PageSize: pageSize, Pages: pages}, nil
}

func (r *invoiceRepository) get(ctx context.Context, predicate string, args ...any) (*service.InvoiceRequest, error) {
	inv := &service.InvoiceRequest{}
	query := fmt.Sprintf(`
		SELECT id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
		FROM invoice_requests
		WHERE %s
	`, predicate)
	err := r.db.QueryRowContext(ctx, query, args...).Scan(invoiceScanDest(inv)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invoice request: %w", err)
	}
	return inv, nil
}

func (r *invoiceRepository) insertInvoiceTaxBalanceHistory(ctx context.Context, tx *sql.Tx, inv *service.InvoiceRequest, taxFee float64, input service.CompleteInvoiceInput) error {
	code, err := invoiceAdjustmentCode()
	if err != nil {
		return err
	}
	note := fmt.Sprintf("发票税点扣除：发票申请 #%d，开票金额 %.2f，税点 %.2f%%", inv.ID, inv.Amount, service.InvoiceTaxRate*100)
	if strings.TrimSpace(input.AdminNote) != "" {
		note += "；" + strings.TrimSpace(input.AdminNote)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes, created_at, validity_days)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6, NOW(), 0)
	`, code, service.AdjustmentTypeAdminBalance, -taxFee, service.StatusUsed, inv.UserID, note)
	if err != nil {
		return fmt.Errorf("insert invoice tax balance history: %w", err)
	}
	return nil
}

func invoiceAdjustmentCode() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate invoice adjustment code: %w", err)
	}
	return "INV" + strings.ToUpper(hex.EncodeToString(b[:])), nil
}

func (r *invoiceRepository) invalidateUserBillingCaches(ctx context.Context, userID int64) {
	if r.authCacheInvalidator != nil {
		r.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if r.billingCacheInvalidator == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = r.billingCacheInvalidator.InvalidateUserBalance(cacheCtx, userID)
}

func (r *invoiceRepository) invoiceStatusOrNotFound(ctx context.Context, id int64) error {
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM invoice_requests WHERE id = $1`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrInvoiceNotFound
	}
	if err != nil {
		return err
	}
	return service.ErrInvoiceInvalidStatus.WithMetadata(map[string]string{"status": status})
}

func (r *invoiceRepository) invoiceStatusOrNotFoundForUser(ctx context.Context, id, userID int64) error {
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM invoice_requests WHERE id = $1 AND user_id = $2`, id, userID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrInvoiceNotFound
	}
	if err != nil {
		return err
	}
	return service.ErrInvoiceInvalidStatus.WithMetadata(map[string]string{"status": status})
}

func buildInvoiceWhere(filters InvoiceQueryFilters) ([]string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 4)
	if filters.UserID > 0 {
		args = append(args, filters.UserID)
		where = append(where, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if filters.Status != "" {
		args = append(args, filters.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if filters.Search != "" {
		args = append(args, "%"+escapeLike(filters.Search)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(user_email ILIKE $%d OR user_name ILIKE $%d OR title ILIKE $%d OR tax_id ILIKE $%d OR invoice_no ILIKE $%d)", n, n, n, n, n))
	}
	return where, args
}

func invoiceOrderBy(params pagination.PaginationParams, _ bool) string {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := pagination.NormalizeSortOrder(params.SortOrder, pagination.SortOrderDesc)
	dir := "DESC"
	if sortOrder == pagination.SortOrderAsc {
		dir = "ASC"
	}
	field := "created_at"
	switch sortBy {
	case "amount":
		field = "amount"
	case "status":
		field = "status"
	case "updated_at":
		field = "updated_at"
	case "completed_at":
		field = "completed_at"
	case "created_at":
		field = "created_at"
	}
	return fmt.Sprintf("%s %s, id DESC", field, dir)
}

func normalizeInvoiceSummary(s *service.InvoiceSummary) *service.InvoiceSummary {
	s.RechargeAmount = roundMoneyRepo(s.RechargeAmount)
	s.InvoicedAmount = roundMoneyRepo(s.InvoicedAmount)
	s.LockedAmount = roundMoneyRepo(s.LockedAmount)
	s.AvailableAmount = roundMoneyRepo(s.AvailableAmount)
	if s.AvailableAmount < 0 {
		s.AvailableAmount = 0
	}
	s.CurrentBalance = roundMoneyRepo(s.CurrentBalance)
	s.MinAmount = service.InvoiceMinAmount
	s.TaxRate = service.InvoiceTaxRate
	s.TaxRatePercent = service.InvoiceTaxRate * 100
	s.CanApply = s.AvailableAmount >= service.InvoiceMinAmount
	s.InvoiceableBasis = "completed_balance_payment_orders"
	return s
}

func invoiceScanDest(inv *service.InvoiceRequest) []any {
	return []any{
		&inv.ID,
		&inv.UserID,
		&inv.UserEmail,
		&inv.UserName,
		&inv.Status,
		&inv.InvoiceType,
		&inv.Title,
		&inv.TaxID,
		&inv.ItemName,
		&inv.Amount,
		&inv.TaxRate,
		&inv.TaxFee,
		&inv.ReceiverEmail,
		&inv.Note,
		&inv.AdminNote,
		&inv.InvoiceNo,
		&inv.CompletedAt,
		&inv.RejectedAt,
		&inv.ApprovedAt,
		&inv.ProcessedBy,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	}
}

func invoiceTemplateScanDest(tmpl *service.InvoiceTemplate) []any {
	return []any{
		&tmpl.ID,
		&tmpl.UserID,
		&tmpl.Name,
		&tmpl.InvoiceType,
		&tmpl.Title,
		&tmpl.TaxID,
		&tmpl.ItemName,
		&tmpl.ReceiverEmail,
		&tmpl.Note,
		&tmpl.IsDefault,
		&tmpl.CreatedAt,
		&tmpl.UpdatedAt,
	}
}

func isInvoiceTemplateNameConflict(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return string(pqErr.Code) == "23505" && pqErr.Constraint == "idx_invoice_templates_user_name"
}

func nullableAdminID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

func roundMoneyRepo(v float64) float64 {
	return math.Round(v*100) / 100
}

const invoiceSummarySQL = `
	WITH recharge AS (
		SELECT COALESCE(SUM(amount), 0)::float8 AS amount
		FROM payment_orders
		WHERE user_id = $1
			AND status = 'COMPLETED'
			AND order_type = 'balance'
	), invoiced AS (
		SELECT COALESCE(SUM(amount), 0)::float8 AS amount
		FROM invoice_requests
		WHERE user_id = $1
			AND status = 'completed'
	), locked AS (
		SELECT COALESCE(SUM(amount), 0)::float8 AS amount
		FROM invoice_requests
		WHERE user_id = $1
			AND status IN ('pending', 'approved', 'completed')
	)
	SELECT
		recharge.amount,
		invoiced.amount,
		locked.amount,
		GREATEST(recharge.amount - locked.amount, 0)::float8 AS available_amount,
		u.balance::float8
	FROM users u, recharge, invoiced, locked
	WHERE u.id = $1 AND u.deleted_at IS NULL
`
