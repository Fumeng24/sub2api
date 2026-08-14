package payment

import "testing"

func TestStandaloneProviderKeyForPaymentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		paymentType PaymentType
		want        string
	}{
		{paymentType: TypeStripe, want: TypeStripe},
		{paymentType: TypeAirwallex, want: TypeAirwallex},
		{paymentType: TypeUSDT, want: TypeGMPay},
		{paymentType: TypeAlipay, want: ""},
		{paymentType: TypeWxpay, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.paymentType, func(t *testing.T) {
			t.Parallel()
			if got := standaloneProviderKeyForPaymentType(tt.paymentType); got != tt.want {
				t.Fatalf("standaloneProviderKeyForPaymentType(%q) = %q, want %q", tt.paymentType, got, tt.want)
			}
		})
	}
}
