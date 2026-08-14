import { afterEach, describe, expect, it, vi } from 'vitest'
import { formatDiscountSchedule, resolveGroupRateDiscount } from '../groupRateDiscount'

describe('groupRateDiscount utils', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

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
        schedule_mode: 'once',
        start_at: '2026-01-01T00:00:00Z',
        end_at: '2026-01-02T00:00:00Z',
        weekdays: [],
        daily_start_time: '',
        daily_end_time: '',
        group_ids: [11],
      },
    )

    expect(discount).toBeNull()
  })

  it('returns weekly discounts during selected weekdays and time windows', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-04T10:00:00'))

    const discount = resolveGroupRateDiscount(
      11,
      2,
      {
        name: 'Monday Promo',
        discount_multiplier: 0.6,
        schedule_mode: 'weekly',
        start_at: '',
        end_at: '',
        weekdays: [1, 3],
        daily_start_time: '09:00',
        daily_end_time: '18:00',
        group_ids: [11],
      },
    )

    expect(discount?.discountedRate).toBe(1.2)
    expect(discount?.weekdays).toEqual([1, 3])
  })

  it('does not return weekly discounts outside selected weekdays', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-05T10:00:00'))

    const discount = resolveGroupRateDiscount(
      11,
      2,
      {
        name: 'Monday Promo',
        discount_multiplier: 0.6,
        schedule_mode: 'weekly',
        start_at: '',
        end_at: '',
        weekdays: [1],
        daily_start_time: '09:00',
        daily_end_time: '18:00',
        group_ids: [11],
      },
    )

    expect(discount).toBeNull()
  })

  it('keeps weekly cross-midnight discounts active after midnight', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-05T07:30:00'))

    const discount = resolveGroupRateDiscount(
      11,
      2,
      {
        name: 'Late Promo',
        discount_multiplier: 0.5,
        schedule_mode: 'weekly',
        start_at: '',
        end_at: '',
        weekdays: [1],
        daily_start_time: '22:00',
        daily_end_time: '08:00',
        group_ids: [11],
      },
    )

    expect(discount?.discountedRate).toBe(1)
  })

  it('evaluates weekly discounts in the server timezone when provided', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-04T14:30:00Z'))

    const discount = resolveGroupRateDiscount(
      11,
      2,
      {
        name: 'Shanghai Evening Promo',
        discount_multiplier: 0.5,
        schedule_mode: 'weekly',
        start_at: '',
        end_at: '',
        weekdays: [1],
        daily_start_time: '22:00',
        daily_end_time: '23:00',
        timezone: 'Asia/Shanghai',
        group_ids: [11],
      },
    )

    expect(discount?.discountedRate).toBe(1)
  })

  it('can return upcoming weekly discounts for previews', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-04T01:00:00'))

    const discount = resolveGroupRateDiscount(
      11,
      0.2,
      {
        name: '深夜限时折扣',
        discount_multiplier: 0.7,
        schedule_mode: 'weekly',
        start_at: '',
        end_at: '',
        weekdays: [1, 2, 3, 4, 5, 6, 7],
        daily_start_time: '02:00',
        daily_end_time: '08:00',
        group_ids: [11],
      },
      undefined,
      true,
    )

    expect(discount?.status).toBe('upcoming')
    expect(discount?.discountedRate).toBeCloseTo(0.14)
  })

  it('uses embedded weekly schedule fields before falling back to public settings', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-04T03:00:00'))

    const discount = resolveGroupRateDiscount(
      11,
      0.2,
      null,
      {
        multiplier: 0.7,
        discountedRate: 0.14,
        name: '深夜限时折扣',
        scheduleMode: 'weekly',
        weekdays: [1, 2, 3, 4, 5, 6, 7],
        dailyStartTime: '02:00',
        dailyEndTime: '08:00',
        timezone: null,
      },
    )

    expect(discount?.status).toBe('active')
    expect(discount?.discountedRate).toBeCloseTo(0.14)
  })

  it('labels early daily schedules as late night and morning in Chinese', () => {
    expect(formatDiscountSchedule({
      name: '深夜限时折扣',
      discount_multiplier: 0.7,
      schedule_mode: 'weekly',
      start_at: '',
      end_at: '',
      weekdays: [1, 2, 3, 4, 5, 6, 7],
      daily_start_time: '02:00',
      daily_end_time: '08:00',
      group_ids: [11],
    }, 'zh-CN')).toBe('每天 凌晨 02:00 - 早上 08:00')
  })

  it('shows the start time for upcoming one-off discounts', () => {
    expect(formatDiscountSchedule({
      name: 'Future Sale',
      discount_multiplier: 0.8,
      schedule_mode: 'once',
      start_at: '2026-05-10T02:00:00+08:00',
      end_at: '2026-05-10T08:00:00+08:00',
      weekdays: [],
      daily_start_time: '',
      daily_end_time: '',
      group_ids: [11],
    }, 'zh-CN', 'upcoming')).toContain('开始于')
  })
})
