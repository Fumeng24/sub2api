package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestGMPayProviderConfigRegistration(t *testing.T) {
	t.Parallel()

	require.True(t, validProviderKeys[payment.TypeGMPay])
	require.True(t, isSensitiveProviderConfigField(payment.TypeGMPay, "SecretKey"))
	require.Equal(t, map[string]struct{}{
		"secretkey": {},
		"pid":       {},
		"apibase":   {},
		"currency":  {},
		"token":     {},
		"network":   {},
	}, providerPendingOrderProtectedConfigFields[payment.TypeGMPay])
	require.NoError(t, validateProviderRequest(payment.TypeGMPay, "GMPay", payment.TypeUSDT))
}
