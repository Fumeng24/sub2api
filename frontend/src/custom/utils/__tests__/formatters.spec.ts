import { describe, expect, it } from 'vitest'

import { formatMultiplier } from '../formatters'

describe('custom formatters', () => {
  it('keeps finite multiplier precision without unnecessary trailing zeros', () => {
    expect(formatMultiplier(1)).toBe('1.00')
    expect(formatMultiplier(0.3)).toBe('0.30')
    expect(formatMultiplier(0.1234)).toBe('0.1234')
    expect(formatMultiplier(0.0012)).toBe('0.0012')
  })

  it('uses a stable fallback for non-finite multipliers', () => {
    expect(formatMultiplier(Number.NaN)).toBe('1')
    expect(formatMultiplier(Number.POSITIVE_INFINITY)).toBe('1')
  })
})
