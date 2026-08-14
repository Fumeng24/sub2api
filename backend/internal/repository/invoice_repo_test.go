package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestInvoiceSummarySQLUsesNetRechargeBasis(t *testing.T) {
	normalized := strings.Join(strings.Fields(invoiceSummarySQL), " ")

	require.Contains(t, normalized, "amount - COALESCE(refund_amount, 0)")
	require.Contains(t, normalized, "status IN ('COMPLETED', 'PARTIALLY_REFUNDED', 'REFUNDED')")
	require.Contains(t, normalized, "FROM redeem_codes rc")
	require.Contains(t, normalized, "rc.used_by = $1")
	require.Contains(t, normalized, "rc.status = 'used'")
	require.Contains(t, normalized, "rc.type = 'balance'")
	require.Contains(t, normalized, "rc.type = 'admin_balance'")
	require.Contains(t, normalized, "rc.business_category IN ('manual_collection', 'manual_refund')")
	require.NotContains(t, normalized, "rc.value > 0")
	require.Contains(t, normalized, "NOT EXISTS")
	require.Contains(t, normalized, "FROM payment_orders po")
	require.Contains(t, normalized, "po.recharge_code = rc.code")
	require.Contains(t, normalized, "payment_recharge.amount + standalone_redeem.amount")
}

func TestInvoiceRequestQueriesIncludeSourceOrderSnapshot(t *testing.T) {
	normalized := strings.Join(strings.Fields(invoiceSummarySQL), " ")
	require.NotContains(t, normalized, "source_orders_json")

	createSnippet := strings.Join(strings.Fields(`
		INSERT INTO invoice_requests (
			user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, source_order_count, source_orders_json, created_at, updated_at
		)
		RETURNING id, user_id, user_email, user_name, status, invoice_type, title, tax_id, item_name,
			amount, tax_rate, tax_fee, receiver_email, note, admin_note, invoice_no,
			source_order_count, source_orders_json,
			completed_at, rejected_at, approved_at, processed_by, created_at, updated_at
	`), " ")
	require.Contains(t, createSnippet, "source_order_count, source_orders_json")
}

func TestInvoiceSourceOrdersScanSupportsLegacyAndStructuredJSON(t *testing.T) {
	t.Run("structured", func(t *testing.T) {
		var out service.InvoiceSourceOrders
		require.NoError(t, out.Scan([]byte(`[{"id":1,"record_source":"payment_order","business_category":"recharge","payment_type":"alipay","out_trade_no":"T1","amount":10,"refund_amount":0,"invoiceable":true}]`)))
		require.Len(t, out, 1)
		require.Equal(t, int64(1), out[0].ID)
		require.Equal(t, "payment_order", out[0].RecordSource)
		require.True(t, out[0].Invoiceable)
	})

	t.Run("legacy ids", func(t *testing.T) {
		var out service.InvoiceSourceOrders
		require.NoError(t, out.Scan([]byte(`[1,2,3]`)))
		require.Len(t, out, 3)
		require.Equal(t, int64(1), out[0].ID)
		require.Equal(t, int64(3), out[2].ID)
	})
}
