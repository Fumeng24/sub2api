import type { OrderStatus, PaymentOrder, PaymentRecordSource } from '@/types/payment'

export const invoiceablePaymentStatuses = new Set<OrderStatus>(['COMPLETED', 'PARTIALLY_REFUNDED', 'REFUNDED'])

export function isPaymentOrder(order: Pick<PaymentOrder, 'record_source'>): boolean {
  return !order.record_source || order.record_source === 'payment_order'
}

export function normalizedRecordSource(order: Pick<PaymentOrder, 'record_source'>): PaymentRecordSource {
  return order.record_source || 'payment_order'
}

export function inferredBusinessCategory(order: Pick<PaymentOrder, 'record_source' | 'business_category'>): string {
  const businessCategory = String(order.business_category || '')
  if (businessCategory === 'affiliate_rebate') return 'affiliate_reward'
  if (businessCategory) return businessCategory
  const source = normalizedRecordSource(order)
  if (source === 'payment_order' || source === 'redeem_code') return 'recharge'
  if (source === 'affiliate_rebate') return 'affiliate_reward'
  return ''
}

export function recordSourceI18nKey(order: Pick<PaymentOrder, 'record_source'>): string {
  return `payment.admin.sources.${normalizedRecordSource(order)}`
}

export function businessCategoryI18nKey(order: Pick<PaymentOrder, 'record_source' | 'business_category'>): string {
  const category = inferredBusinessCategory(order) || 'uncategorized'
  return `payment.businessCategories.${category}`
}

export function isInvoiceableBalanceSource(
  order: Pick<PaymentOrder, 'record_source' | 'business_category'>
): boolean {
  if (isPaymentOrder(order)) return true

  const source = normalizedRecordSource(order)
  const category = inferredBusinessCategory(order)

  if (source === 'redeem_code') return category === 'recharge'
  if (source === 'admin_balance') return category === 'manual_collection' || category === 'manual_refund'
  return false
}

export function orderRefundAmount(order: Pick<PaymentOrder, 'refund_amount'>): number {
  return roundMoney(Math.max(Number(order.refund_amount) || 0, 0))
}

export function orderNetInvoiceAmount(
  order: Pick<PaymentOrder, 'amount' | 'refund_amount' | 'record_source'>
): number {
  const amount = roundMoney(Math.max(Number(order.amount) || 0, 0))
  if (!isPaymentOrder(order)) return amount
  return roundMoney(Math.max(amount - orderRefundAmount(order), 0))
}

export function invoiceUnavailableReasonKey(
  order: Pick<PaymentOrder, 'order_type' | 'record_source' | 'business_category' | 'status' | 'amount' | 'refund_amount'>,
  amount = orderNetInvoiceAmount(order)
): string {
  if (order.order_type !== 'balance') return 'notBalance'
  if (!isInvoiceableBalanceSource(order)) return 'notInvoiceableSource'
  if (!invoiceablePaymentStatuses.has(order.status)) return 'notCompleted'
  if (amount <= 0 && isPaymentOrder(order) && orderRefundAmount(order) > 0) return 'fullyRefunded'
  if (amount <= 0) return 'zeroAmount'
  return ''
}

export function isOrderInvoiceable(
  order: Pick<PaymentOrder, 'order_type' | 'record_source' | 'business_category' | 'status' | 'amount' | 'refund_amount'>
): boolean {
  return invoiceUnavailableReasonKey(order) === ''
}

function roundMoney(value: number): number {
  return Math.round(value * 100) / 100
}
