import type { ActiveGroupRateDiscount, Group } from '@/types'

export interface GroupRateDiscountDisplay {
  name: string
  multiplier: number
  originalRate: number
  discountedRate: number
  startAt?: string | null
  endAt?: string | null
}

export function roundRateMultiplier(value: number): number {
  return Number(value.toFixed(4))
}

export function formatRateMultiplier(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return ''
  }
  return String(roundRateMultiplier(value))
}

export function formatDiscountLabel(multiplier: number | null | undefined): string {
  if (multiplier === null || multiplier === undefined || !Number.isFinite(multiplier) || multiplier <= 0) {
    return ''
  }
  const percent = multiplier * 100
  if (Number.isInteger(percent)) {
    return `${percent}%`
  }
  return `${Number(percent.toFixed(2))}%`
}

export function formatDiscountDateTime(
  value: string | null | undefined,
  locale = 'zh-CN',
): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function parseDiscountTime(value: string | null | undefined): number | null {
  if (!value) return null
  const parsed = new Date(value).getTime()
  return Number.isNaN(parsed) ? null : parsed
}

function isDiscountWindowActive(
  startAt: string | null | undefined,
  endAt: string | null | undefined,
  now = Date.now(),
): boolean {
  const start = parseDiscountTime(startAt)
  const end = parseDiscountTime(endAt)
  if (startAt && start === null) return false
  if (endAt && end === null) return false
  if (start !== null && now < start) return false
  if (end !== null && now >= end) return false
  return true
}

export function resolveGroupRateDiscount(
  groupID: number | null | undefined,
  baseRate: number | null | undefined,
  globalDiscount?: ActiveGroupRateDiscount | null,
  embedded?: {
    multiplier?: number | null
    discountedRate?: number | null
    name?: string | null
    startAt?: string | null
    endAt?: string | null
  },
): GroupRateDiscountDisplay | null {
  const originalRate = Number(baseRate)
  if (!Number.isFinite(originalRate)) {
    return null
  }
  const now = Date.now()
  if (embedded?.multiplier && embedded.multiplier > 0 && embedded.multiplier < 1) {
    const startAt = embedded.startAt ?? globalDiscount?.start_at
    const endAt = embedded.endAt ?? globalDiscount?.end_at
    if (!isDiscountWindowActive(startAt, endAt, now)) {
      return null
    }
    return {
      name: embedded.name || globalDiscount?.name || '限时折扣',
      multiplier: embedded.multiplier,
      originalRate,
      discountedRate: originalRate * embedded.multiplier,
      startAt,
      endAt,
    }
  }
  if (!globalDiscount || !groupID || !globalDiscount.group_ids?.includes(groupID)) {
    return null
  }
  if (globalDiscount.discount_multiplier <= 0 || globalDiscount.discount_multiplier >= 1) {
    return null
  }
  if (!isDiscountWindowActive(globalDiscount.start_at, globalDiscount.end_at, now)) {
    return null
  }
  return {
    name: globalDiscount.name || '限时折扣',
    multiplier: globalDiscount.discount_multiplier,
    originalRate,
    discountedRate: originalRate * globalDiscount.discount_multiplier,
    startAt: globalDiscount.start_at,
    endAt: globalDiscount.end_at,
  }
}

export function resolveGroupDiscountFromGroup(
  group: Pick<
    Group,
    | 'id'
    | 'rate_multiplier'
    | 'group_rate_discount_multiplier'
    | 'discounted_rate_multiplier'
    | 'group_rate_discount_name'
    | 'group_rate_discount_start_at'
    | 'group_rate_discount_end_at'
  >,
  baseRate?: number | null,
  globalDiscount?: ActiveGroupRateDiscount | null,
): GroupRateDiscountDisplay | null {
  return resolveGroupRateDiscount(
    group.id,
    baseRate ?? group.rate_multiplier,
    globalDiscount,
    {
      multiplier: group.group_rate_discount_multiplier,
      discountedRate: group.discounted_rate_multiplier,
      name: group.group_rate_discount_name,
      startAt: group.group_rate_discount_start_at,
      endAt: group.group_rate_discount_end_at,
    },
  )
}
