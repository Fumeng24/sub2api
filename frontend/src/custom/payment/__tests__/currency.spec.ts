import { describe, expect, it } from 'vitest'
import { paymentAmountPrefix } from '../currency'

describe('paymentAmountPrefix', () => {
  it('uses compact prefixes for supported settlement currencies', () => {
    expect(paymentAmountPrefix('CNY')).toBe('¥')
    expect(paymentAmountPrefix('HKD')).toBe('$')
    expect(paymentAmountPrefix('EUR')).toBe('€')
    expect(paymentAmountPrefix('GBP')).toBe('£')
  })

  it('normalizes input and falls back to the normalized currency code', () => {
    expect(paymentAmountPrefix('usd')).toBe('$')
    expect(paymentAmountPrefix('XYZ')).toBe('XYZ')
    expect(paymentAmountPrefix('')).toBe('¥')
  })
})
