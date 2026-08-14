package payment

func standaloneProviderKeyForPaymentType(paymentType PaymentType) string {
	switch normalizeVisibleMethodSupportType(paymentType) {
	case TypeStripe:
		return TypeStripe
	case TypeAirwallex:
		return TypeAirwallex
	case TypeUSDT:
		return TypeGMPay
	default:
		return ""
	}
}
