package service

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// validateCustomProviderSnapshotMetadata extends the upstream snapshot checks
// with the site-specific GMPay identity and status fields.
func validateCustomProviderSnapshotMetadata(order *ent.PaymentOrder, providerKey string, metadata map[string]string) error {
	if err := validateProviderSnapshotMetadata(order, providerKey, metadata); err != nil {
		return err
	}
	if order == nil || len(metadata) == 0 || !strings.EqualFold(strings.TrimSpace(providerKey), payment.TypeGMPay) {
		return nil
	}

	snapshot := psOrderProviderSnapshot(order)
	if snapshot == nil {
		return nil
	}
	if expected := strings.TrimSpace(snapshot.MerchantID); expected != "" {
		actual := strings.TrimSpace(metadata["pid"])
		if actual == "" {
			return fmt.Errorf("gmpay pid missing")
		}
		if !strings.EqualFold(expected, actual) {
			return fmt.Errorf("gmpay pid mismatch: expected %s, got %s", expected, actual)
		}
	}
	if expected := strings.TrimSpace(snapshot.Currency); expected != "" {
		actual := strings.ToUpper(strings.TrimSpace(metadata["currency"]))
		if actual == "" {
			return fmt.Errorf("gmpay notification missing currency")
		}
		if !strings.EqualFold(expected, actual) {
			return fmt.Errorf("gmpay currency mismatch: expected %s, got %s", expected, actual)
		}
	}
	if actual := strings.TrimSpace(metadata["status"]); actual != "" && actual != "2" {
		return fmt.Errorf("gmpay status mismatch: expected 2, got %s", actual)
	}
	return nil
}
