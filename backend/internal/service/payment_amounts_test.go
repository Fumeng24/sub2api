package service

import (
	"testing"
)

func TestCalculateCreditedBalanceCreditsPaymentAmountOneToOne(t *testing.T) {
	t.Parallel()

	got := calculateCreditedBalance(10, 1)
	if got != 10 {
		t.Fatalf("credited balance = %v, want 10", got)
	}
}

func TestCalculateCreditedBalanceRoundsToCents(t *testing.T) {
	t.Parallel()

	got := calculateCreditedBalance(12.345, 1)
	if got != 12.35 {
		t.Fatalf("credited balance = %v, want 12.35", got)
	}
}
