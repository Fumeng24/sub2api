package service

import (
	"math"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 6.8

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

func calculateCreditedBalance(paymentAmount, cnyPerBalanceCredit float64, currency string) float64 {
	amount := decimal.NewFromFloat(paymentAmount)
	if !shouldApplyCNYBalanceRechargeRate(currency) {
		return amount.Round(2).InexactFloat64()
	}
	return amount.
		Div(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(cnyPerBalanceCredit))).
		Round(2).
		InexactFloat64()
}

func shouldApplyCNYBalanceRechargeRate(currency string) bool {
	normalized, err := payment.NormalizePaymentCurrency(currency)
	if err != nil {
		normalized = payment.DefaultPaymentCurrency
	}
	return normalized == payment.DefaultPaymentCurrency
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(payAmount).Round(fractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(fractionDigits).
		InexactFloat64()
}
