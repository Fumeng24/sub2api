package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestCalculateCreditedBalanceUsesCNYPerBalanceCredit(t *testing.T) {
	t.Parallel()

	got := calculateCreditedBalance(64, 6.4, payment.DefaultPaymentCurrency)
	if got != 10 {
		t.Fatalf("credited balance = %v, want 10", got)
	}
}

func TestCalculateCreditedBalanceKeepsNonCNYCurrenciesOneToOne(t *testing.T) {
	t.Parallel()

	got := calculateCreditedBalance(12.345, 6.4, "USD")
	if got != 12.35 {
		t.Fatalf("credited balance = %v, want 12.35", got)
	}
}

func TestCalculateCreditedBalanceNormalizesInvalidRate(t *testing.T) {
	t.Parallel()

	got := calculateCreditedBalance(12, 0, payment.DefaultPaymentCurrency)
	if got != 1.76 {
		t.Fatalf("credited balance = %v, want 1.76", got)
	}
}
