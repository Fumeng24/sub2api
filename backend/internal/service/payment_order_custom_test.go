package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestBuildPaymentOrderProviderSnapshotCustomIncludesGMPayMerchantIdentity(t *testing.T) {
	t.Parallel()

	snapshot := buildPaymentOrderProviderSnapshot(&payment.InstanceSelection{
		InstanceID:  "gm-88",
		ProviderKey: payment.TypeGMPay,
		Config: map[string]string{
			"pid":      "merchant-88",
			"currency": "usd",
			"key":      "secret",
		},
		PaymentMode: "redirect",
	}, CreateOrderRequest{})

	require.Equal(t, "merchant-88", snapshot["merchant_id"])
	require.Equal(t, "USD", snapshot["currency"])
	require.NotContains(t, snapshot, "key")
}
