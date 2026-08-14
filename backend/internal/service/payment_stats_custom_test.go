package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestPaymentDashboardStatsCustomIncludesRedeemsAndAdminAdjustments(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	ignoredUser, err := client.User.Create().
		SetEmail("ignored@example.com").
		SetPasswordHash("hash").
		SetUsername("ignored").
		Save(ctx)
	require.NoError(t, err)
	require.Equal(t, ignoredPaymentDashboardUserID, ignoredUser.ID)
	user, err := client.User.Create().
		SetEmail("dashboard@example.com").
		SetPasswordHash("hash").
		SetUsername("dashboard").
		Save(ctx)
	require.NoError(t, err)
	require.NotEqual(t, ignoredPaymentDashboardUserID, user.ID)

	now := time.Now().Truncate(time.Second)
	old := now.AddDate(0, 0, -15)

	createOrder := func(userID int64, userEmail, username, code, status string, amount, refund float64, at time.Time, refundAt *time.Time) {
		t.Helper()
		order := client.PaymentOrder.Create().
			SetUserID(userID).
			SetUserEmail(userEmail).
			SetUserName(username).
			SetAmount(amount).
			SetPayAmount(amount).
			SetFeeRate(0).
			SetRechargeCode(code).
			SetOutTradeNo("out-" + code).
			SetPaymentType(payment.TypeAlipay).
			SetPaymentTradeNo("trade-" + code).
			SetOrderType(payment.OrderTypeBalance).
			SetStatus(status).
			SetExpiresAt(at.Add(time.Hour)).
			SetPaidAt(at).
			SetCompletedAt(at).
			SetClientIP("127.0.0.1").
			SetSrcHost("api.example.com").
			SetCreatedAt(at).
			SetUpdatedAt(at)
		if refund > 0 {
			order.SetRefundAmount(refund)
		}
		if refundAt != nil {
			order.SetRefundAt(*refundAt)
			order.SetUpdatedAt(*refundAt)
		}
		_, err := order.Save(ctx)
		require.NoError(t, err)
	}
	createRedeem := func(userID int64, code, typ, category string, value float64, at time.Time) {
		t.Helper()
		builder := client.RedeemCode.Create().
			SetCode(code).
			SetType(typ).
			SetValue(value).
			SetStatus(StatusUsed).
			SetUsedBy(userID).
			SetUsedAt(at).
			SetCreatedAt(at)
		if category != "" {
			builder.SetBusinessCategory(category)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}

	yesterday := now.AddDate(0, 0, -1)
	createOrder(user.ID, user.Email, user.Username, "PAY-100", OrderStatusCompleted, 100, 0, now, nil)
	createOrder(user.ID, user.Email, user.Username, "PAY-PARTIAL", OrderStatusPartiallyRefunded, 80, 30, now, &now)
	createOrder(user.ID, user.Email, user.Username, "PAY-REFUNDED", OrderStatusRefunded, 40, 40, now, &now)
	createOrder(user.ID, user.Email, user.Username, "PAY-PENDING", OrderStatusPending, 999, 0, now, nil)
	createOrder(user.ID, user.Email, user.Username, "PAY-OLD", OrderStatusCompleted, 15, 0, old, nil)
	createOrder(user.ID, user.Email, user.Username, "PAY-YESTERDAY-REFUND-TODAY", OrderStatusRefunded, 90, 90, yesterday, &now)
	createOrder(ignoredUser.ID, ignoredUser.Email, ignoredUser.Username, "PAY-IGNORED", OrderStatusCompleted, 9999, 0, now, nil)

	createRedeem(user.ID, "PAY-100", RedeemTypeBalance, BalanceBusinessCategoryRecharge, 100, now)
	createRedeem(user.ID, "CARD-200", RedeemTypeBalance, BalanceBusinessCategoryRecharge, 200, now)
	createRedeem(user.ID, "ADMIN-ADD", AdjustmentTypeAdminBalance, BalanceBusinessCategoryManualCollection, 50, now)
	createRedeem(user.ID, "ADMIN-DEDUCT", AdjustmentTypeAdminBalance, BalanceBusinessCategoryManualRefund, -20, now)
	createRedeem(user.ID, "ADMIN-LEGACY", AdjustmentTypeAdminBalance, "", 70, now)
	createRedeem(user.ID, "ADMIN-GIFT", AdjustmentTypeAdminBalance, BalanceBusinessCategoryGiftCompensation, 30, now)
	createRedeem(user.ID, "INV-FEE", AdjustmentTypeAdminBalance, BalanceBusinessCategorySystemServiceFee, -10, now)
	createRedeem(ignoredUser.ID, "CARD-IGNORED", RedeemTypeBalance, BalanceBusinessCategoryRecharge, 8888, now)
	createRedeem(ignoredUser.ID, "ADMIN-IGNORED", AdjustmentTypeAdminBalance, BalanceBusinessCategoryManualCollection, 7777, now)

	svc := &PaymentService{entClient: client}
	got, err := svc.GetDashboardStats(ctx, 7)
	require.NoError(t, err)

	require.InDelta(t, 395, got.TotalAmount[payment.DefaultPaymentCurrency], 0.000001)
	require.InDelta(t, 290, got.TodayAmount[payment.DefaultPaymentCurrency], 0.000001)
	require.Equal(t, 8, got.TotalCount)
	require.Equal(t, 6, got.TodayCount)
	require.Equal(t, 1, got.PendingOrders)
	require.InDelta(t, 49.38, got.AvgAmount[payment.DefaultPaymentCurrency], 0.000001)

	var today DailyStats
	for _, item := range got.DailySeries {
		if item.Date == now.Format("2006-01-02") {
			today = item
			break
		}
	}
	require.InDelta(t, 290, today.Amount[payment.DefaultPaymentCurrency], 0.000001)
	require.Equal(t, 6, today.Count)

	var yesterdayStats DailyStats
	for _, item := range got.DailySeries {
		if item.Date == yesterday.Format("2006-01-02") {
			yesterdayStats = item
			break
		}
	}
	require.InDelta(t, 90, yesterdayStats.Amount[payment.DefaultPaymentCurrency], 0.000001)
	require.Equal(t, 1, yesterdayStats.Count)

	methods := map[string]PaymentMethodStat{}
	for _, method := range got.PaymentMethods {
		methods[method.Type] = method
	}
	require.InDelta(t, 165, methods[payment.TypeAlipay].Amount[payment.DefaultPaymentCurrency], 0.000001)
	require.Equal(t, 5, methods[payment.TypeAlipay].Count)
	require.InDelta(t, 200, methods["redeem_code"].Amount[payment.DefaultPaymentCurrency], 0.000001)
	require.Equal(t, 1, methods["redeem_code"].Count)
	require.InDelta(t, 30, methods["admin_balance"].Amount[payment.DefaultPaymentCurrency], 0.000001)
	require.Equal(t, 2, methods["admin_balance"].Count)

	topUsers := got.TopUsers[payment.DefaultPaymentCurrency]
	require.Len(t, topUsers, 1)
	require.Equal(t, user.ID, topUsers[0].UserID)
	require.InDelta(t, 395, topUsers[0].Amount, 0.000001)
}
