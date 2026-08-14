package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

type AdminPaymentOrderRecord struct {
	ID                  int64      `json:"id"`
	SourceID            int64      `json:"source_id"`
	RecordSource        string     `json:"record_source"`
	RecordSourceLabel   string     `json:"record_source_label,omitempty"`
	BusinessCategory    string     `json:"business_category,omitempty"`
	UserID              int64      `json:"user_id"`
	UserEmail           string     `json:"user_email"`
	UserName            string     `json:"user_name"`
	UserNotes           *string    `json:"user_notes,omitempty"`
	Amount              float64    `json:"amount"`
	PayAmount           float64    `json:"pay_amount"`
	Currency            string     `json:"currency,omitempty"`
	FeeRate             float64    `json:"fee_rate"`
	RechargeCode        string     `json:"recharge_code,omitempty"`
	OutTradeNo          string     `json:"out_trade_no"`
	PaymentType         string     `json:"payment_type"`
	PaymentTradeNo      string     `json:"payment_trade_no,omitempty"`
	ProviderInstanceID  string     `json:"provider_instance_id,omitempty"`
	OrderType           string     `json:"order_type"`
	Status              string     `json:"status"`
	RefundAmount        float64    `json:"refund_amount"`
	RefundReason        *string    `json:"refund_reason,omitempty"`
	RefundRequestedAt   *time.Time `json:"refund_requested_at,omitempty"`
	RefundRequestedBy   *string    `json:"refund_requested_by,omitempty"`
	RefundRequestReason *string    `json:"refund_request_reason,omitempty"`
	ExpiresAt           time.Time  `json:"expires_at,omitempty"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Notes               string     `json:"notes,omitempty"`
}

func (s *PaymentService) NativeBalanceRechargeAvailable(ctx context.Context, userID int64, cfg *PaymentConfig) (bool, float64, error) {
	if cfg == nil || !cfg.BalanceDisabled {
		return true, 0, nil
	}
	threshold := cfg.BalanceUnlockThreshold
	if threshold <= 0 {
		return false, 0, nil
	}
	net, err := s.UserNetRechargeAmount(ctx, userID)
	if err != nil {
		return false, 0, err
	}
	return net >= threshold, net, nil
}

func (s *PaymentService) UserNetRechargeAmount(ctx context.Context, userID int64) (float64, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return 0, nil
	}
	var paymentNet sql.NullFloat64
	rows, err := s.entClient.QueryContext(ctx, `
		SELECT COALESCE(SUM(
			CASE
				WHEN amount - COALESCE(refund_amount, 0) > 0 THEN amount - COALESCE(refund_amount, 0)
				ELSE 0
			END
		), 0)
		FROM payment_orders
		WHERE user_id = $1
			AND order_type = 'balance'
			AND status IN ('COMPLETED', 'PARTIALLY_REFUNDED', 'REFUNDED')
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("sum balance payment orders: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&paymentNet); err != nil {
			return 0, fmt.Errorf("scan balance payment orders: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate balance payment orders: %w", err)
	}

	var redeemNet sql.NullFloat64
	rows, err = s.entClient.QueryContext(ctx, `
		SELECT COALESCE(SUM(rc.value), 0)
		FROM redeem_codes rc
		WHERE rc.used_by = $1
			AND rc.status = 'used'
			AND (
				rc.type = 'balance'
				OR (
					rc.type = 'admin_balance'
					AND rc.business_category IN ('manual_collection', 'manual_refund')
				)
			)
			AND NOT EXISTS (
				SELECT 1
				FROM payment_orders po
				WHERE po.recharge_code = rc.code
			)
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("sum standalone balance adjustments: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&redeemNet); err != nil {
			return 0, fmt.Errorf("scan standalone balance adjustments: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate standalone balance adjustments: %w", err)
	}
	return paymentNet.Float64 + redeemNet.Float64, nil
}

func extendPaymentOrderProviderSnapshotCustom(snapshot map[string]any, sel *payment.InstanceSelection) {
	if sel == nil || strings.TrimSpace(sel.ProviderKey) != payment.TypeGMPay {
		return
	}
	if pid := strings.TrimSpace(sel.Config["pid"]); pid != "" {
		snapshot["merchant_id"] = pid
	}
	snapshot["currency"] = paymentProviderConfigCurrency(sel.ProviderKey, sel.Config)
}

func (s *PaymentService) AdminListOrderRecords(ctx context.Context, userID int64, p OrderListParams) ([]AdminPaymentOrderRecord, int64, error) {
	if s == nil || s.entClient == nil {
		return nil, 0, nil
	}
	ps, pg := applyPagination(p.PageSize, p.Page)
	query, args := buildAdminOrderRecordsQuery(userID, p, ps, (pg-1)*ps)
	rows, err := s.entClient.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query admin order records: %w", err)
	}
	defer rows.Close()

	records := make([]AdminPaymentOrderRecord, 0, ps)
	var total int64
	for rows.Next() {
		var rec AdminPaymentOrderRecord
		var userNotes sql.NullString
		var refundReason sql.NullString
		var refundRequestedBy sql.NullString
		var refundRequestReason sql.NullString
		var expiresAt sql.NullTime
		var paidAt sql.NullTime
		var completedAt sql.NullTime
		var notes sql.NullString
		if err := rows.Scan(
			&total,
			&rec.ID,
			&rec.SourceID,
			&rec.RecordSource,
			&rec.RecordSourceLabel,
			&rec.BusinessCategory,
			&rec.UserID,
			&rec.UserEmail,
			&rec.UserName,
			&userNotes,
			&rec.Amount,
			&rec.PayAmount,
			&rec.Currency,
			&rec.FeeRate,
			&rec.RechargeCode,
			&rec.OutTradeNo,
			&rec.PaymentType,
			&rec.PaymentTradeNo,
			&rec.ProviderInstanceID,
			&rec.OrderType,
			&rec.Status,
			&rec.RefundAmount,
			&refundReason,
			&rec.RefundRequestedAt,
			&refundRequestedBy,
			&refundRequestReason,
			&expiresAt,
			&paidAt,
			&completedAt,
			&rec.CreatedAt,
			&rec.UpdatedAt,
			&notes,
		); err != nil {
			return nil, 0, fmt.Errorf("scan admin order record: %w", err)
		}
		if userNotes.Valid {
			rec.UserNotes = &userNotes.String
		}
		if refundReason.Valid {
			rec.RefundReason = &refundReason.String
		}
		if refundRequestedBy.Valid {
			rec.RefundRequestedBy = &refundRequestedBy.String
		}
		if refundRequestReason.Valid {
			rec.RefundRequestReason = &refundRequestReason.String
		}
		if expiresAt.Valid {
			rec.ExpiresAt = expiresAt.Time
		}
		if paidAt.Valid {
			rec.PaidAt = &paidAt.Time
		}
		if completedAt.Valid {
			rec.CompletedAt = &completedAt.Time
		}
		if notes.Valid {
			rec.Notes = notes.String
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate admin order records: %w", err)
	}
	return records, total, nil
}

func (s *PaymentService) UserListOrderRecords(ctx context.Context, userID int64, p OrderListParams) ([]AdminPaymentOrderRecord, int64, error) {
	if userID <= 0 {
		return nil, 0, nil
	}
	return s.AdminListOrderRecords(ctx, userID, p)
}

func buildAdminOrderRecordsQuery(userID int64, p OrderListParams, limit, offset int) (string, []any) {
	clauses := make([]string, 0, 5)
	args := make([]any, 0, 7)
	addArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if userID > 0 {
		clauses = append(clauses, "user_id = "+addArg(userID))
	}
	if p.Status != "" {
		clauses = append(clauses, "status = "+addArg(p.Status))
	}
	if p.OrderType != "" {
		clauses = append(clauses, "order_type = "+addArg(p.OrderType))
	}
	if p.PaymentType != "" {
		clauses = append(clauses, "payment_type = "+addArg(p.PaymentType))
	}
	if p.Invoiceable != nil {
		invoiceableExpr := `(order_type = 'balance'
			AND status IN ('COMPLETED', 'PARTIALLY_REFUNDED', 'REFUNDED')
			AND CASE
				WHEN record_source = 'payment_order' THEN GREATEST(amount - COALESCE(refund_amount, 0), 0)
				ELSE amount
			END > 0
			AND (
				record_source = 'payment_order'
				OR (record_source = 'redeem_code' AND business_category = 'recharge')
				OR (record_source = 'admin_balance' AND business_category IN ('manual_collection', 'manual_refund'))
			))`
		if *p.Invoiceable {
			clauses = append(clauses, invoiceableExpr)
		} else {
			clauses = append(clauses, "NOT "+invoiceableExpr)
		}
	}
	if keyword := strings.TrimSpace(p.Keyword); keyword != "" {
		pattern := "%" + strings.ToLower(keyword) + "%"
		placeholder := addArg(pattern)
		clauses = append(clauses, `(LOWER(out_trade_no) LIKE `+placeholder+`
			OR LOWER(user_email) LIKE `+placeholder+`
			OR LOWER(user_name) LIKE `+placeholder+`
			OR LOWER(recharge_code) LIKE `+placeholder+`
			OR LOWER(notes) LIKE `+placeholder+`
			OR CAST(source_id AS text) LIKE `+placeholder+`)`)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	limitPlaceholder := addArg(limit)
	offsetPlaceholder := addArg(offset)
	query := `
WITH records AS (
	SELECT
		po.id AS id,
		po.id AS source_id,
		'payment_order'::text AS record_source,
		'线上订单'::text AS record_source_label,
		'recharge'::text AS business_category,
		po.user_id,
		COALESCE(po.user_email, '') AS user_email,
		COALESCE(po.user_name, '') AS user_name,
		po.user_notes,
		po.amount::double precision AS amount,
		po.pay_amount::double precision AS pay_amount,
		''::text AS currency,
		po.fee_rate::double precision AS fee_rate,
		COALESCE(po.recharge_code, '') AS recharge_code,
		COALESCE(po.out_trade_no, '') AS out_trade_no,
		COALESCE(po.payment_type, '') AS payment_type,
		COALESCE(po.payment_trade_no, '') AS payment_trade_no,
		COALESCE(po.provider_instance_id, '') AS provider_instance_id,
		COALESCE(po.order_type, 'balance') AS order_type,
		COALESCE(po.status, '') AS status,
		po.refund_amount::double precision AS refund_amount,
		po.refund_reason,
		po.refund_requested_at,
		po.refund_requested_by,
		po.refund_request_reason,
		po.expires_at,
		po.paid_at,
		po.completed_at,
		po.created_at,
		po.updated_at,
		COALESCE(po.failed_reason, '')::text AS notes
	FROM payment_orders po

	UNION ALL

	SELECT
		-rc.id AS id,
		rc.id AS source_id,
		CASE WHEN rc.type = 'admin_balance' THEN 'admin_balance' ELSE 'redeem_code' END AS record_source,
		CASE
			WHEN rc.type = 'admin_balance' AND rc.business_category = 'manual_collection' THEN '管理员收款'
			WHEN rc.type = 'admin_balance' AND rc.business_category = 'manual_refund' THEN '管理员退款'
			WHEN rc.type = 'admin_balance' AND rc.business_category = 'gift_compensation' THEN '赠送补偿'
			WHEN rc.type = 'admin_balance' AND rc.business_category = 'gift_reversal' THEN '赠送冲正'
			WHEN rc.type = 'admin_balance' AND rc.business_category = 'system_service_fee' THEN '系统服务费'
			WHEN rc.type = 'admin_balance' THEN '管理员调整'
			ELSE '卡密兑换'
		END AS record_source_label,
		COALESCE(rc.business_category, '') AS business_category,
		COALESCE(rc.used_by, 0) AS user_id,
		COALESCE(u.email, '') AS user_email,
		COALESCE(u.username, '') AS user_name,
		NULLIF(u.notes, '') AS user_notes,
		rc.value::double precision AS amount,
		0::double precision AS pay_amount,
		''::text AS currency,
		0::double precision AS fee_rate,
		rc.code AS recharge_code,
		rc.code AS out_trade_no,
		CASE WHEN rc.type = 'admin_balance' THEN 'admin_balance' ELSE 'redeem_code' END AS payment_type,
		''::text AS payment_trade_no,
		''::text AS provider_instance_id,
		'balance'::text AS order_type,
		'COMPLETED'::text AS status,
		0::double precision AS refund_amount,
		NULL::text AS refund_reason,
		NULL::timestamptz AS refund_requested_at,
		NULL::text AS refund_requested_by,
		NULL::text AS refund_request_reason,
		COALESCE(rc.used_at, rc.created_at) AS expires_at,
		rc.used_at AS paid_at,
		rc.used_at AS completed_at,
		COALESCE(rc.used_at, rc.created_at) AS created_at,
		COALESCE(rc.used_at, rc.created_at) AS updated_at,
		COALESCE(rc.notes, '')::text AS notes
	FROM redeem_codes rc
	LEFT JOIN users u ON u.id = rc.used_by
	WHERE rc.status = 'used'
		AND rc.used_by IS NOT NULL
		AND rc.type IN ('balance', 'admin_balance')
		AND NOT EXISTS (
			SELECT 1
			FROM payment_orders po
			WHERE po.recharge_code = rc.code
		)

	UNION ALL

	SELECT
		(-1000000000000 - ual.id) AS id,
		ual.id AS source_id,
		'affiliate_rebate'::text AS record_source,
		'邀请返佣'::text AS record_source_label,
		'affiliate_reward'::text AS business_category,
		ual.user_id,
		COALESCE(u.email, '') AS user_email,
		COALESCE(u.username, '') AS user_name,
		NULLIF(u.notes, '') AS user_notes,
		ual.amount::double precision AS amount,
		0::double precision AS pay_amount,
		''::text AS currency,
		0::double precision AS fee_rate,
		''::text AS recharge_code,
		CASE
			WHEN ual.source_order_id IS NOT NULL THEN 'AFF-' || ual.source_order_id::text
			ELSE 'AFF-' || ual.id::text
		END AS out_trade_no,
		'affiliate_rebate'::text AS payment_type,
		''::text AS payment_trade_no,
		''::text AS provider_instance_id,
		'balance'::text AS order_type,
		'COMPLETED'::text AS status,
		0::double precision AS refund_amount,
		NULL::text AS refund_reason,
		NULL::timestamptz AS refund_requested_at,
		NULL::text AS refund_requested_by,
		NULL::text AS refund_request_reason,
		ual.created_at AS expires_at,
		ual.created_at AS paid_at,
		ual.created_at AS completed_at,
		ual.created_at,
		ual.updated_at,
		CASE
			WHEN ual.source_user_id IS NOT NULL THEN '邀请用户 #' || ual.source_user_id::text || ' 充值返佣'
			ELSE '邀请返佣'
		END AS notes
	FROM user_affiliate_ledger ual
	LEFT JOIN users u ON u.id = ual.user_id
	WHERE ual.action = 'accrue'
		AND ual.source_user_id IS NOT NULL
), filtered AS (
	SELECT *
	FROM records
	` + where + `
)
SELECT
	COUNT(*) OVER() AS total,
	id,
	source_id,
	record_source,
	record_source_label,
	business_category,
	user_id,
	user_email,
	user_name,
	user_notes,
	amount,
	pay_amount,
	currency,
	fee_rate,
	recharge_code,
	out_trade_no,
	payment_type,
	payment_trade_no,
	provider_instance_id,
	order_type,
	status,
	refund_amount,
	refund_reason,
	refund_requested_at,
	refund_requested_by,
	refund_request_reason,
	expires_at,
	paid_at,
	completed_at,
	created_at,
	updated_at,
	notes
FROM filtered
ORDER BY created_at DESC, id DESC
LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	return query, args
}
