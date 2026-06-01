import type { PaymentOrder } from '@/types/payment'
import { formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import {
  formatSettlementCurrencyAmount,
  getCurrentSettlementCnyPerCredit,
  getCurrentSettlementCurrency,
} from '@/composables/useSettlementCurrency'

export function orderCurrency(order: Pick<PaymentOrder, 'currency'>): string {
  return normalizePaymentCurrency(order.currency)
}

export function formatOrderPaymentAmount(
  order: Pick<PaymentOrder, 'currency'>,
  amount: number,
  locale?: string,
): string {
  return formatPaymentAmount(amount, orderCurrency(order), locale)
}

export function formatCreditedBalance(amount: number): string {
  return formatSettlementCurrencyAmount(
    amount,
    getCurrentSettlementCurrency(),
    getCurrentSettlementCnyPerCredit(),
    undefined,
    2,
  )
}

export function shouldShowCreditedBalance(order: Pick<PaymentOrder, 'order_type'>): boolean {
  return order.order_type === 'balance'
}
