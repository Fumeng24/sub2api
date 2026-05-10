import type { ActiveGroupRateDiscount, Group } from '@/types'

export interface GroupRateDiscountDisplay {
  name: string
  multiplier: number
  originalRate: number
  discountedRate: number
  status: 'active' | 'upcoming'
  scheduleMode?: string | null
  startAt?: string | null
  endAt?: string | null
  weekdays?: number[] | null
  dailyStartTime?: string | null
  dailyEndTime?: string | null
  timezone?: string | null
}

export interface GroupRateDiscountSummary {
  discount: ActiveGroupRateDiscount
  status: 'active' | 'upcoming'
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

export function formatDiscountSchedule(
  discount: Pick<
    ActiveGroupRateDiscount,
    | 'schedule_mode'
    | 'start_at'
    | 'end_at'
    | 'weekdays'
    | 'daily_start_time'
    | 'daily_end_time'
    | 'timezone'
  > | GroupRateDiscountDisplay | null | undefined,
  locale = 'zh-CN',
  status?: 'active' | 'upcoming' | null,
): string {
  if (!discount) return ''
  const scheduleMode = 'schedule_mode' in discount ? discount.schedule_mode : discount.scheduleMode
  if (scheduleMode === 'weekly') {
    const weekdays = ('weekdays' in discount ? discount.weekdays : null) ?? []
    const startTime = ('daily_start_time' in discount ? discount.daily_start_time : discount.dailyStartTime) || ''
    const endTime = ('daily_end_time' in discount ? discount.daily_end_time : discount.dailyEndTime) || ''
    const timezone = formatDisplayTimezone(('timezone' in discount ? discount.timezone : null) || '')
    const weekdayLabel = formatWeekdays(weekdays, locale)
    if (!weekdayLabel || !startTime || !endTime) return ''
    const timeLabel = formatDailyTimeRange(startTime, endTime, locale)
    return timezone ? `${weekdayLabel} ${timeLabel} ${timezone}` : `${weekdayLabel} ${timeLabel}`
  }
  const startAt = 'start_at' in discount ? discount.start_at : discount.startAt
  const endAt = 'end_at' in discount ? discount.end_at : discount.endAt
  const start = formatDiscountDateTime(startAt, locale)
  if (status === 'upcoming' && start) {
    return locale.startsWith('zh') ? `开始于 ${start}` : `Starts ${start}`
  }
  const end = formatDiscountDateTime(endAt, locale)
  if (end) {
    return locale.startsWith('zh') ? `活动至 ${end}` : `Ends ${end}`
  }
  return start ? (locale.startsWith('zh') ? `开始于 ${start}` : `Starts ${start}`) : ''
}

export function resolvePublicGroupRateDiscount(
  active?: ActiveGroupRateDiscount | null,
  upcoming?: ActiveGroupRateDiscount | null,
  now = Date.now(),
): GroupRateDiscountSummary | null {
  const activeStatus = resolveDiscountStatus(active, now)
  if (active && activeStatus) {
    return { discount: active, status: activeStatus }
  }
  const upcomingStatus = resolveDiscountStatus(upcoming, now)
  if (upcoming && upcomingStatus) {
    return { discount: upcoming, status: upcomingStatus }
  }
  return null
}

export function formatDiscountStatusLabel(status: 'active' | 'upcoming', locale = 'zh-CN'): string {
  if (status === 'active') {
    return locale.startsWith('zh') ? '进行中' : 'Active'
  }
  return locale.startsWith('zh') ? '即将开始' : 'Upcoming'
}

function formatWeekdays(weekdays: number[], locale: string): string {
  const normalized = Array.from(new Set(weekdays.filter((day) => day >= 1 && day <= 7))).sort((a, b) => a - b)
  if (normalized.length === 0) return ''
  if (normalized.length === 7) {
    return locale.startsWith('zh') ? '每天' : 'Every day'
  }
  const zhLabels = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
  const enLabels = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
  const labels = locale.startsWith('zh') ? zhLabels : enLabels
  return normalized.map((day) => labels[day - 1]).join(locale.startsWith('zh') ? '、' : ', ')
}

function formatDisplayTimezone(timezone: string): string {
  return timezone === 'Local' ? '' : timezone
}

function resolveDiscountStatus(
  discount: ActiveGroupRateDiscount | null | undefined,
  now = Date.now(),
): 'active' | 'upcoming' | null {
  if (!discount || discount.discount_multiplier <= 0 || discount.discount_multiplier >= 1 || discount.group_ids.length === 0) {
    return null
  }
  if (discount.schedule_mode === 'weekly') {
    if (
      !discount.weekdays?.length ||
      !discount.daily_start_time ||
      !discount.daily_end_time ||
      discount.daily_start_time === discount.daily_end_time
    ) {
      return null
    }
    return isWeeklyDiscountActive(
      discount.weekdays,
      discount.daily_start_time,
      discount.daily_end_time,
      discount.timezone,
      now,
    )
      ? 'active'
      : 'upcoming'
  }
  const start = parseDiscountTime(discount.start_at)
  const end = parseDiscountTime(discount.end_at)
  if (start === null || end === null || start >= end || now >= end) {
    return null
  }
  return now >= start ? 'active' : 'upcoming'
}

function formatDailyTimeRange(startTime: string, endTime: string, locale: string): string {
  if (!locale.startsWith('zh')) {
    return `${formatTimeOfDay(startTime, locale)}-${formatTimeOfDay(endTime, locale)}`
  }
  const startPeriod = zhTimePeriodLabel(startTime)
  const endPeriod = zhTimePeriodLabel(endTime)
  if (startPeriod && endPeriod) {
    return `${startPeriod} ${startTime} - ${endPeriod} ${endTime}`
  }
  return `${startTime}-${endTime}`
}

function formatTimeOfDay(value: string, locale: string): string {
  if (locale.startsWith('zh')) {
    return value
  }
  const minutes = parseDailyTimeMinutes(value)
  if (minutes === null) return value
  const hour = Math.floor(minutes / 60)
  const minute = minutes % 60
  const suffix = hour < 12 ? 'AM' : 'PM'
  const hour12 = hour % 12 || 12
  return `${hour12}:${String(minute).padStart(2, '0')} ${suffix}`
}

function zhTimePeriodLabel(value: string): string {
  const minutes = parseDailyTimeMinutes(value)
  if (minutes === null) return ''
  if (minutes < 5 * 60) return '凌晨'
  if (minutes < 9 * 60) return '早上'
  if (minutes < 12 * 60) return '上午'
  if (minutes < 14 * 60) return '中午'
  if (minutes < 18 * 60) return '下午'
  return '晚上'
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

function isWeeklyDiscountActive(
  weekdays: number[] | null | undefined,
  dailyStartTime: string | null | undefined,
  dailyEndTime: string | null | undefined,
  timezone: string | null | undefined,
  now = Date.now(),
): boolean {
  const selected = Array.from(new Set((weekdays ?? []).filter((day) => day >= 1 && day <= 7)))
  if (selected.length === 0 || !dailyStartTime || !dailyEndTime || dailyStartTime === dailyEndTime) {
    return false
  }
  const start = parseDailyTimeMinutes(dailyStartTime)
  const end = parseDailyTimeMinutes(dailyEndTime)
  if (start === null || end === null || start === end) return false
  const current = getDiscountLocalParts(now, timezone)
  if (!current) return false
  const currentMinutes = current.hour * 60 + current.minute
  const currentDay = current.weekday
  if (start < end) {
    return selected.includes(currentDay) && currentMinutes >= start && currentMinutes < end
  }
  if (selected.includes(currentDay) && currentMinutes >= start) {
    return true
  }
  const previousDay = currentDay === 1 ? 7 : currentDay - 1
  return selected.includes(previousDay) && currentMinutes < end
}

function parseDailyTimeMinutes(value: string): number | null {
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(value)
  if (!match) return null
  return Number(match[1]) * 60 + Number(match[2])
}

function getDiscountLocalParts(now: number, timezone: string | null | undefined): { weekday: number; hour: number; minute: number } | null {
  const date = new Date(now)
  if (!timezone) {
    return getBrowserLocalParts(date)
  }
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      weekday: 'short',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    }).formatToParts(date)
    const weekday = parts.find((part) => part.type === 'weekday')?.value
    const hour = Number(parts.find((part) => part.type === 'hour')?.value)
    const minute = Number(parts.find((part) => part.type === 'minute')?.value)
    const weekdayMap: Record<string, number> = {
      Mon: 1,
      Tue: 2,
      Wed: 3,
      Thu: 4,
      Fri: 5,
      Sat: 6,
      Sun: 7,
    }
    const normalizedWeekday = weekday ? weekdayMap[weekday] : undefined
    if (!normalizedWeekday || !Number.isFinite(hour) || !Number.isFinite(minute)) {
      return null
    }
    return {
      weekday: normalizedWeekday,
      hour,
      minute,
    }
  } catch {
    return getBrowserLocalParts(date)
  }
}

function getBrowserLocalParts(date: Date): { weekday: number; hour: number; minute: number } {
  return {
    weekday: isoWeekday(date.getDay()),
    hour: date.getHours(),
    minute: date.getMinutes(),
  }
}

function isoWeekday(jsDay: number): number {
  return jsDay === 0 ? 7 : jsDay
}

export function resolveGroupRateDiscount(
  groupID: number | null | undefined,
  baseRate: number | null | undefined,
  globalDiscount?: ActiveGroupRateDiscount | null,
  embedded?: {
    multiplier?: number | null
    discountedRate?: number | null
    name?: string | null
    scheduleMode?: string | null
    startAt?: string | null
    endAt?: string | null
    weekdays?: number[] | null
    dailyStartTime?: string | null
    dailyEndTime?: string | null
    timezone?: string | null
  },
  includeUpcoming = false,
  now = Date.now(),
): GroupRateDiscountDisplay | null {
  const originalRate = Number(baseRate)
  if (!Number.isFinite(originalRate)) {
    return null
  }
  if (embedded?.multiplier && embedded.multiplier > 0 && embedded.multiplier < 1) {
    const startAt = embedded.startAt ?? globalDiscount?.start_at
    const endAt = embedded.endAt ?? globalDiscount?.end_at
    const scheduleMode = embedded.scheduleMode ?? globalDiscount?.schedule_mode
    const weekdays = embedded.weekdays ?? globalDiscount?.weekdays
    const dailyStartTime = embedded.dailyStartTime ?? globalDiscount?.daily_start_time
    const dailyEndTime = embedded.dailyEndTime ?? globalDiscount?.daily_end_time
    const timezone = embedded.timezone ?? globalDiscount?.timezone
    const active = scheduleMode === 'weekly'
      ? isWeeklyDiscountActive(weekdays, dailyStartTime, dailyEndTime, timezone, now)
      : isDiscountWindowActive(startAt, endAt, now)
    if (!active) {
      if (!includeUpcoming) {
        return null
      }
    }
    return {
      name: embedded.name || globalDiscount?.name || '限时折扣',
      multiplier: embedded.multiplier,
      originalRate,
      discountedRate: originalRate * embedded.multiplier,
      status: active ? 'active' : 'upcoming',
      scheduleMode,
      startAt,
      endAt,
      weekdays,
      dailyStartTime,
      dailyEndTime,
      timezone,
    }
  }
  if (!globalDiscount || !groupID || !globalDiscount.group_ids?.includes(groupID)) {
    return null
  }
  if (globalDiscount.discount_multiplier <= 0 || globalDiscount.discount_multiplier >= 1) {
    return null
  }
  const active = globalDiscount.schedule_mode === 'weekly'
    ? isWeeklyDiscountActive(globalDiscount.weekdays, globalDiscount.daily_start_time, globalDiscount.daily_end_time, globalDiscount.timezone, now)
    : isDiscountWindowActive(globalDiscount.start_at, globalDiscount.end_at, now)
  if (!active) {
    if (!includeUpcoming) {
      return null
    }
  }
  return {
    name: globalDiscount.name || '限时折扣',
    multiplier: globalDiscount.discount_multiplier,
    originalRate,
    discountedRate: originalRate * globalDiscount.discount_multiplier,
    status: active ? 'active' : 'upcoming',
    scheduleMode: globalDiscount.schedule_mode,
    startAt: globalDiscount.start_at,
    endAt: globalDiscount.end_at,
    weekdays: globalDiscount.weekdays,
    dailyStartTime: globalDiscount.daily_start_time,
    dailyEndTime: globalDiscount.daily_end_time,
    timezone: globalDiscount.timezone,
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
    | 'group_rate_discount_schedule_mode'
    | 'group_rate_discount_start_at'
    | 'group_rate_discount_end_at'
    | 'group_rate_discount_weekdays'
    | 'group_rate_discount_daily_start_time'
    | 'group_rate_discount_daily_end_time'
    | 'group_rate_discount_timezone'
  >,
  baseRate?: number | null,
  globalDiscount?: ActiveGroupRateDiscount | null,
  includeUpcoming = false,
  now = Date.now(),
): GroupRateDiscountDisplay | null {
  return resolveGroupRateDiscount(
    group.id,
    baseRate ?? group.rate_multiplier,
    globalDiscount,
    {
      multiplier: group.group_rate_discount_multiplier,
      discountedRate: group.discounted_rate_multiplier,
      name: group.group_rate_discount_name,
      scheduleMode: group.group_rate_discount_schedule_mode,
      startAt: group.group_rate_discount_start_at,
      endAt: group.group_rate_discount_end_at,
      weekdays: group.group_rate_discount_weekdays,
      dailyStartTime: group.group_rate_discount_daily_start_time,
      dailyEndTime: group.group_rate_discount_daily_end_time,
      timezone: group.group_rate_discount_timezone,
    },
    includeUpcoming,
    now,
  )
}
