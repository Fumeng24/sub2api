package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAdminOrderRecordsQueryIncludesAllBalanceRecordSources(t *testing.T) {
	query, args := buildAdminOrderRecordsQuery(123, OrderListParams{
		Status:      OrderStatusCompleted,
		OrderType:   "balance",
		PaymentType: "affiliate_rebate",
		Keyword:     "invitee@example.com",
	}, 20, 40)

	normalized := strings.Join(strings.Fields(query), " ")
	require.Contains(t, normalized, "FROM payment_orders po")
	require.Contains(t, normalized, "'recharge'::text AS business_category")
	require.Contains(t, normalized, "FROM redeem_codes rc")
	require.Contains(t, normalized, "FROM user_affiliate_ledger ual")
	require.Contains(t, normalized, "WHERE ual.action = 'accrue' AND ual.source_user_id IS NOT NULL")
	require.Contains(t, normalized, "'affiliate_rebate'::text AS record_source")
	require.Contains(t, normalized, "'affiliate_reward'::text AS business_category")
	require.Contains(t, normalized, "AND NOT EXISTS")
	require.Contains(t, normalized, "WHERE po.recharge_code = rc.code")
	require.Contains(t, normalized, "record_source")
	require.Contains(t, normalized, "business_category")
	require.Contains(t, normalized, "provider_instance_id")
	require.Len(t, args, 7)
	require.Equal(t, int64(123), args[0])
	require.Equal(t, OrderStatusCompleted, args[1])
	require.Equal(t, "balance", args[2])
	require.Equal(t, "affiliate_rebate", args[3])
	require.Equal(t, "%invitee@example.com%", args[4])
	require.Equal(t, 20, args[5])
	require.Equal(t, 40, args[6])
}

func TestBuildAdminOrderRecordsQueryFiltersInvoiceableRecords(t *testing.T) {
	yes := true
	query, args := buildAdminOrderRecordsQuery(123, OrderListParams{
		Invoiceable: &yes,
	}, 20, 0)

	normalized := strings.Join(strings.Fields(query), " ")
	require.Contains(t, normalized, "record_source = 'payment_order'")
	require.Contains(t, normalized, "record_source = 'redeem_code' AND business_category = 'recharge'")
	require.Contains(t, normalized, "record_source = 'admin_balance' AND business_category IN ('manual_collection', 'manual_refund')")
	require.Contains(t, normalized, "status IN ('COMPLETED', 'PARTIALLY_REFUNDED', 'REFUNDED')")
	require.Contains(t, normalized, "GREATEST(amount - COALESCE(refund_amount, 0), 0)")
	require.NotContains(t, normalized, "NOT (order_type = 'balance'")
	require.Len(t, args, 3)

	no := false
	query, args = buildAdminOrderRecordsQuery(123, OrderListParams{
		Invoiceable: &no,
	}, 20, 0)

	normalized = strings.Join(strings.Fields(query), " ")
	require.Contains(t, normalized, "NOT (order_type = 'balance'")
	require.Contains(t, normalized, "record_source = 'payment_order'")
	require.Len(t, args, 3)
}
