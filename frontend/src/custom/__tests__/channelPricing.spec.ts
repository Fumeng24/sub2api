import { describe, expect, it } from 'vitest'
import { formatScaledCny } from '@/custom/channels/pricing'

describe('channel CNY pricing', () => {
  it('preserves upstream scaling precision with a CNY symbol', () => {
    expect(formatScaledCny(0.00000015, 1_000_000)).toBe('¥0.15')
    expect(formatScaledCny(0.5, 1)).toBe('¥0.5')
    expect(formatScaledCny(0, 1_000_000)).toBe('¥0')
  })

  it('keeps missing prices distinct from zero', () => {
    expect(formatScaledCny(null, 1_000_000)).toBe('-')
  })
})
