package provider

import "github.com/Wei-Shaw/sub2api/internal/payment"

// CreateProviderWithExtensions dispatches site providers before delegating to
// the upstream provider factory.
func CreateProviderWithExtensions(providerKey, instanceID string, config map[string]string) (payment.Provider, error) {
	if providerKey == payment.TypeGMPay {
		return NewGMPay(instanceID, config)
	}
	return CreateProvider(providerKey, instanceID, config)
}
