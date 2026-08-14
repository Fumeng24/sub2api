import { describe, expect, it } from 'vitest'
import type { PaymentOrder } from '@/types/payment'
import {
  businessCategoryI18nKey,
  inferredBusinessCategory,
  invoiceUnavailableReasonKey,
  isOrderInvoiceable,
  orderNetInvoiceAmount,
  recordSourceI18nKey,
} from '../paymentRecordSemantics'

function order(overrides: Partial<PaymentOrder>): PaymentOrder {
  return {
    id: 1,
    user_id: 2,
    amount: 100,
    pay_amount: 100,
    fee_rate: 0,
    payment_type: 'alipay',
    out_trade_no: 'T001',
    status: 'COMPLETED',
    order_type: 'balance',
    created_at: '2026-06-01T00:00:00Z',
    expires_at: '2026-06-01T00:30:00Z',
    refund_amount: 0,
    ...overrides,
  }
}

describe('payment record semantics', () => {
  it('treats online payment top-ups as paid recharge and subtracts refunds', () => {
    const item = order({ record_source: 'payment_order', amount: 100, refund_amount: 35 })

    expect(recordSourceI18nKey(item)).toBe('payment.admin.sources.payment_order')
    expect(inferredBusinessCategory(item)).toBe('recharge')
    expect(businessCategoryI18nKey(item)).toBe('payment.businessCategories.recharge')
    expect(orderNetInvoiceAmount(item)).toBe(65)
    expect(isOrderInvoiceable(item)).toBe(true)
  })

  it('treats redeem-code top-ups as paid recharge when categorized as recharge', () => {
    const item = order({ record_source: 'redeem_code', business_category: 'recharge' })

    expect(inferredBusinessCategory(item)).toBe('recharge')
    expect(isOrderInvoiceable(item)).toBe(true)
  })

  it('keeps gift compensation and affiliate rewards out of invoiceable sources', () => {
    const gift = order({ record_source: 'admin_balance', business_category: 'gift_compensation' })
    const affiliate = order({ record_source: 'affiliate_rebate', business_category: 'affiliate_reward' })
    const legacyAffiliate = order({
      record_source: 'affiliate_rebate',
      business_category: 'affiliate_rebate' as PaymentOrder['business_category'],
    })

    expect(invoiceUnavailableReasonKey(gift)).toBe('notInvoiceableSource')
    expect(inferredBusinessCategory(affiliate)).toBe('affiliate_reward')
    expect(inferredBusinessCategory(legacyAffiliate)).toBe('affiliate_reward')
    expect(invoiceUnavailableReasonKey(affiliate)).toBe('notInvoiceableSource')
  })

  it('allows manual collection but not unfinished or zero amount records', () => {
    const manualCollection = order({ record_source: 'admin_balance', business_category: 'manual_collection' })
    const pending = order({ status: 'PENDING' })
    const refunded = order({ amount: 100, refund_amount: 100 })

    expect(isOrderInvoiceable(manualCollection)).toBe(true)
    expect(invoiceUnavailableReasonKey(pending)).toBe('notCompleted')
    expect(invoiceUnavailableReasonKey(refunded)).toBe('fullyRefunded')
  })
})
