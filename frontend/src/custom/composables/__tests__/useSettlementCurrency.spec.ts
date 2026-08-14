import { describe, expect, it } from 'vitest'
import {
  convertSettlementAmount,
  convertSettlementAmountToCredits,
  formatSettlementCurrencyAmount,
  getCurrentSettlementCurrency,
  normalizeSettlementCurrency,
  resolveCnyPerCredit,
  setCurrentSettlementCurrency,
  setSettlementCnyPerCredit,
} from '../useSettlementCurrency'
import { formatCreditedBalance } from '@/custom/payment/orderAmounts'

describe('settlement currency helpers', () => {
  it('normalizes unsupported currencies to CNY', () => {
    expect(normalizeSettlementCurrency('usd')).toBe('USD')
    expect(normalizeSettlementCurrency('HKD')).toBe('CNY')
    expect(normalizeSettlementCurrency(null)).toBe('CNY')
  })

  it('converts balance credits to CNY using the configured multiplier', () => {
    expect(convertSettlementAmount(1.25, 'USD', 7.2)).toBe(1.25)
    expect(convertSettlementAmount(1.25, 'CNY', 7.2)).toBe(9)
    expect(resolveCnyPerCredit(0)).toBe(6.8)
  })

  it('converts settlement input amounts back to balance credits', () => {
    expect(convertSettlementAmountToCredits(9, 'CNY', 7.2)).toBe(1.25)
    expect(convertSettlementAmountToCredits(1.25, 'USD', 7.2)).toBe(1.25)
  })

  it('formats with requested precision for small usage costs', () => {
    expect(formatSettlementCurrencyAmount(0.0061, 'USD', 6.8, 'en-US', 4)).toBe('$0.0061')
    expect(formatSettlementCurrencyAmount(0.0061, 'CNY', 6.8, 'zh-CN', 4)).toBe('¥0.0415')
  })

  it('formats credited balances with the active settlement currency', () => {
    setSettlementCnyPerCredit(7.2)

    setCurrentSettlementCurrency('CNY')
    expect(getCurrentSettlementCurrency()).toBe('CNY')
    expect(formatCreditedBalance(2)).toBe('¥14.40')

    setCurrentSettlementCurrency('USD')
    expect(formatCreditedBalance(2)).toBe('$2.00')
  })
})
