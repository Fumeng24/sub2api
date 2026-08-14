import type { PaymentOrder } from '@/types/payment'
import { formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import {
  formatSettlementCurrencyAmount,
  getCurrentSettlementCnyPerCredit,
  getCurrentSettlementCurrency,
} from '@/custom/composables/useSettlementCurrency'

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

export function formatBalanceCreditAmount(amount: number, locale?: string): string {
  const value = (Number.isFinite(amount) ? amount : 0).toFixed(2)
  return !locale || locale.startsWith('zh') ? `${value} 余额` : `${value} balance`
}

export function shouldShowCreditedBalance(order: Pick<PaymentOrder, 'order_type'>): boolean {
  return order.order_type === 'balance'
}
