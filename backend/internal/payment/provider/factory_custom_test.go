package provider

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestCreateProviderWithExtensionsDispatchesGMPay(t *testing.T) {
	provider, err := CreateProviderWithExtensions(payment.TypeGMPay, "7", map[string]string{
		"apiBase": "https://pay.example.com", "pid": "merchant", "secretKey": "secret", "notifyUrl": "https://site.example/webhook",
	})
	if err != nil {
		t.Fatalf("CreateProviderWithExtensions: %v", err)
	}
	if _, ok := provider.(*GMPay); !ok {
		t.Fatalf("provider type=%T want *GMPay", provider)
	}
}

func TestCreateProviderWithExtensionsPreservesUpstreamResults(t *testing.T) {
	for _, key := range []string{payment.TypeEasyPay, payment.TypeAlipay, payment.TypeWxpay, payment.TypeStripe, payment.TypeAirwallex, "unknown"} {
		t.Run(key, func(t *testing.T) {
			want, wantErr := CreateProvider(key, "1", map[string]string{})
			got, gotErr := CreateProviderWithExtensions(key, "1", map[string]string{})
			if reflect.TypeOf(got) != reflect.TypeOf(want) {
				t.Fatalf("provider type=%T want=%T", got, want)
			}
			if (gotErr == nil) != (wantErr == nil) || (gotErr != nil && gotErr.Error() != wantErr.Error()) {
				t.Fatalf("error=%v want=%v", gotErr, wantErr)
			}
		})
	}
}
