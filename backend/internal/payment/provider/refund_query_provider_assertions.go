package provider

import "github.com/Wei-Shaw/sub2api/internal/payment"

var (
	_ payment.RefundQueryProvider = (*Airwallex)(nil)
	_ payment.RefundQueryProvider = (*Stripe)(nil)
	_ payment.RefundQueryProvider = (*Wxpay)(nil)
)
