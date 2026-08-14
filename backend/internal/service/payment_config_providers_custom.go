package service

import "github.com/Wei-Shaw/sub2api/internal/payment"

func init() {
	providerSensitiveConfigFields[payment.TypeGMPay] = map[string]struct{}{
		"secretkey": {},
	}
	providerPendingOrderProtectedConfigFields[payment.TypeGMPay] = map[string]struct{}{
		"secretkey": {},
		"pid":       {},
		"apibase":   {},
		"currency":  {},
		"token":     {},
		"network":   {},
	}
	validProviderKeys[payment.TypeGMPay] = true
}
