import { describe, expect, it, vi } from 'vitest'
import { resolveGroupRateDiscount } from '../groupRateDiscount'

describe('groupRateDiscount utils', () => {
  it('recomputes embedded discounts from the displayed base rate', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T12:00:00Z'))

    const discount = resolveGroupRateDiscount(
      11,
      1.8,
      null,
      {
        multiplier: 0.5,
        discountedRate: 0.7,
        name: 'Promo',
        startAt: '2026-01-01T00:00:00Z',
        endAt: '2026-01-02T00:00:00Z',
      },
    )

    expect(discount?.discountedRate).toBe(0.9)
    vi.useRealTimers()
  })

  it('does not return cached embedded discounts after their end time', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-02T00:00:00Z'))

    const discount = resolveGroupRateDiscount(
      11,
      1.8,
      null,
      {
        multiplier: 0.5,
        discountedRate: 0.9,
        name: 'Promo',
        startAt: '2026-01-01T00:00:00Z',
        endAt: '2026-01-02T00:00:00Z',
      },
    )

    expect(discount).toBeNull()
    vi.useRealTimers()
  })

  it('does not return cached public discounts outside their active window', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-02T00:00:00Z'))

    const discount = resolveGroupRateDiscount(
      11,
      1.8,
      {
        name: 'Promo',
        discount_multiplier: 0.5,
        start_at: '2026-01-01T00:00:00Z',
        end_at: '2026-01-02T00:00:00Z',
        group_ids: [11],
      },
    )

    expect(discount).toBeNull()
    vi.useRealTimers()
  })
})
