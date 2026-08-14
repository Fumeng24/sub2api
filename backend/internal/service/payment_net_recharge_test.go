package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestUserNetRechargeAmountCombinesPaymentsRedeemsAndAdjustments(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("net-recharge@example.com").
		SetPasswordHash("hash").
		SetUsername("net-recharge").
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	createOrder := func(code, status string, amount, refund float64) {
		t.Helper()
		order := client.PaymentOrder.Create().
			SetUserID(user.ID).
			SetUserEmail(user.Email).
			SetUserName(user.Username).
			SetAmount(amount).
			SetPayAmount(amount).
			SetFeeRate(0).
			SetRechargeCode(code).
			SetOutTradeNo("out-" + code).
			SetPaymentType(payment.TypeAlipay).
			SetPaymentTradeNo("trade-" + code).
			SetOrderType(payment.OrderTypeBalance).
			SetStatus(status).
			SetExpiresAt(now.Add(time.Hour)).
			SetPaidAt(now).
			SetClientIP("127.0.0.1").
			SetSrcHost("api.example.com")
		if refund > 0 {
			order.SetRefundAmount(refund)
		}
		_, err := order.Save(ctx)
		require.NoError(t, err)
	}
	createRedeem := func(code, typ, category string, value float64) {
		t.Helper()
		builder := client.RedeemCode.Create().
			SetCode(code).
			SetType(typ).
			SetValue(value).
			SetStatus(StatusUsed).
			SetUsedBy(user.ID).
			SetUsedAt(now)
		if category != "" {
			builder.SetBusinessCategory(category)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}

	createOrder("PAY-100", OrderStatusCompleted, 100, 0)
	createOrder("PAY-PARTIAL", OrderStatusPartiallyRefunded, 80, 30)
	createOrder("PAY-REFUNDED", OrderStatusRefunded, 40, 40)
	createOrder("PAY-PENDING", OrderStatusPending, 999, 0)

	createRedeem("PAY-100", RedeemTypeBalance, BalanceBusinessCategoryRecharge, 100)
	createRedeem("CARD-200", RedeemTypeBalance, BalanceBusinessCategoryRecharge, 200)
	createRedeem("ADMIN-ADD", AdjustmentTypeAdminBalance, BalanceBusinessCategoryManualCollection, 50)
	createRedeem("ADMIN-REFUND", AdjustmentTypeAdminBalance, BalanceBusinessCategoryManualRefund, -20)
	createRedeem("ADMIN-LEGACY", AdjustmentTypeAdminBalance, "", 70)
	createRedeem("INV-FEE", AdjustmentTypeAdminBalance, BalanceBusinessCategorySystemServiceFee, -10)

	svc := &PaymentService{entClient: client}
	got, err := svc.UserNetRechargeAmount(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 380, got, 0.000001)
}

func TestNativeBalanceRechargeAvailableUsesNetThreshold(t *testing.T) {
	svc := &PaymentService{entClient: newPaymentConfigServiceTestClient(t)}

	available, net, err := svc.NativeBalanceRechargeAvailable(context.Background(), 1, &PaymentConfig{
		BalanceDisabled:        true,
		BalanceUnlockThreshold: 0,
	})
	require.NoError(t, err)
	require.False(t, available)
	require.Zero(t, net)

	available, _, err = svc.NativeBalanceRechargeAvailable(context.Background(), 1, &PaymentConfig{
		BalanceDisabled: false,
	})
	require.NoError(t, err)
	require.True(t, available)
}

func TestValidateOrderInputBalanceDisabledDoesNotUseUnlockThreshold(t *testing.T) {
	svc := &PaymentService{}

	_, err := svc.validateOrderInput(context.Background(), CreateOrderRequest{
		OrderType: payment.OrderTypeBalance,
		Amount:    500,
		UserID:    1,
	}, &PaymentConfig{
		BalanceDisabled:        true,
		BalanceUnlockThreshold: 200,
		MinAmount:              200,
	})
	require.Error(t, err)
	require.Equal(t, "BALANCE_PAYMENT_DISABLED", infraerrors.FromError(err).Reason)
}

func TestValidateOrderInputUsesConfiguredMinRechargeAmount(t *testing.T) {
	svc := &PaymentService{}

	_, err := svc.validateOrderInput(context.Background(), CreateOrderRequest{
		OrderType: payment.OrderTypeBalance,
		Amount:    199,
		UserID:    1,
	}, &PaymentConfig{
		MinAmount: 200,
	})
	require.Error(t, err)
	appErr := infraerrors.FromError(err)
	require.Equal(t, "INVALID_AMOUNT", appErr.Reason)
	require.Equal(t, "200.00", appErr.Metadata["min"])
}
