<template>
  <span
    :class="[
      'inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium transition-colors',
      badgeClass
    ]"
  >
    <!-- Platform logo -->
    <PlatformIcon v-if="platform" :platform="platform" size="sm" />
    <!-- Group name -->
    <span class="truncate">{{ name }}</span>
    <!-- Right side label -->
    <span v-if="showLabel" :class="labelClass" :title="discountTitle || undefined">
        <template v-if="discountDisplay">
          <template v-if="discountDisplay.status === 'active'">
            <span class="font-bold">{{ formatRateMultiplier(discountDisplay.discountedRate) }}x</span>
            <span class="ml-0.5 font-semibold">{{ formatDiscountLabel(discountDisplay.multiplier) }}</span>
        </template>
        <template v-else>
          <span class="font-bold">{{ formatRateMultiplier(originalDisplayRate) }}x</span>
          <span class="ml-0.5 opacity-80">{{ upcomingDiscountPrefix }} {{ formatRateMultiplier(discountDisplay.discountedRate) }}x</span>
        </template>
      </template>
      <template v-else-if="hasCustomRate">
        <span class="font-bold">{{ formatRateMultiplier(userRateMultiplier) }}x</span>
      </template>
      <template v-else-if="isSubscription && !alwaysShowRate && rateMultiplier !== undefined">
        <span class="mr-0.5 text-[10px] opacity-75">{{ labelText }}</span>
        <span class="font-bold">{{ effectiveRateLabel }}</span>
      </template>
      <template v-else>
        {{ labelText }}
      </template>
    </span>
    <span v-if="hasPeakRate" :class="peakRateClass" :title="peakRateTitle">
      {{ peakRateText }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useMinuteNow } from '@/custom/composables/useMinuteNow'
import type { SubscriptionType, GroupPlatform } from '@/types'
import {
  formatDiscountLabel,
  formatDiscountSchedule,
  formatRateMultiplier,
  resolveGroupRateDiscount,
  resolvePublicGroupRateDiscount,
} from '@/custom/utils/groupRateDiscount'
import { formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

interface Props {
  name: string
  platform?: GroupPlatform
  subscriptionType?: SubscriptionType
  rateMultiplier?: number
  userRateMultiplier?: number | null // 用户专属倍率
  groupId?: number | null
  discountMultiplier?: number | null
  discountedRateMultiplier?: number | null
  discountName?: string | null
  discountScheduleMode?: string | null
  discountStartAt?: string | null
  discountEndAt?: string | null
  discountWeekdays?: number[] | null
  discountDailyStartTime?: string | null
  discountDailyEndTime?: string | null
  discountTimezone?: string | null
  peakRateEnabled?: boolean
  peakStart?: string
  peakEnd?: string
  peakRateMultiplier?: number
  showRate?: boolean
  daysRemaining?: number | null // 剩余天数（订阅类型时使用）
  /**
   * 订阅分组默认在右侧 label 展示"订阅"或剩余天数；
   * 开启后订阅分组也改为显示倍率（保留订阅主题色 label，配合可用渠道这类
   * 只关心费率、不关心有效期的场景）。
   */
  alwaysShowRate?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  subscriptionType: 'standard',
  showRate: true,
  daysRemaining: null,
  userRateMultiplier: null,
  peakRateEnabled: false,
  alwaysShowRate: false
})

const { t, locale } = useI18n()
const appStore = useAppStore()
const now = useMinuteNow()

const isSubscription = computed(() => props.subscriptionType === 'subscription')
const originalDisplayRate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const publicDiscountSummary = computed(() => resolvePublicGroupRateDiscount(
  appStore.cachedPublicSettings?.group_rate_discount ?? null,
  appStore.cachedPublicSettings?.upcoming_group_rate_discount ?? null,
  now.value,
))
const discountDisplay = computed(() => resolveGroupRateDiscount(
  props.groupId,
  originalDisplayRate.value,
  publicDiscountSummary.value?.discount ?? null,
  {
    multiplier: props.discountMultiplier,
    discountedRate: props.discountedRateMultiplier,
    name: props.discountName,
    scheduleMode: props.discountScheduleMode,
    startAt: props.discountStartAt,
    endAt: props.discountEndAt,
    weekdays: props.discountWeekdays,
    dailyStartTime: props.discountDailyStartTime,
    dailyEndTime: props.discountDailyEndTime,
    timezone: props.discountTimezone
  },
  publicDiscountSummary.value?.status === 'upcoming',
  now.value,
))
const discountTitle = computed(() => {
  if (!discountDisplay.value) return ''
  const schedule = formatDiscountSchedule(discountDisplay.value, locale.value, discountDisplay.value.status)
  const status = discountDisplay.value.status === 'active'
    ? (locale.value.startsWith('zh') ? '进行中' : 'Active')
    : (locale.value.startsWith('zh') ? '即将开始' : 'Upcoming')
  return schedule ? `${status} · ${discountDisplay.value.name} · ${schedule}` : `${status} · ${discountDisplay.value.name}`
})
const upcomingDiscountPrefix = computed(() => locale.value.startsWith('zh') ? '待生效' : 'scheduled')

// 是否有专属倍率（且与默认倍率不同）
const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

const hasPeakRate = computed(() => {
  return Boolean(props.showRate && props.peakRateEnabled && props.peakStart && props.peakEnd)
})

const peakRateText = computed(() => {
  return formatPeakRateWindow(
    {
      peak_rate_enabled: props.peakRateEnabled,
      peak_start: props.peakStart,
      peak_end: props.peakEnd,
      peak_rate_multiplier: props.peakRateMultiplier
    },
    serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)
  )
})

const peakRateTitle = computed(() => {
  return t('common.peakRateTooltip', { window: peakRateText.value })
})

// 是否显示右侧标签
const showLabel = computed(() => {
  if (!props.showRate) return false
  // 订阅类型：显示天数或"订阅"
  if (isSubscription.value) return true
  // 标准类型：显示倍率（包括专属倍率）
  return props.rateMultiplier !== undefined || hasCustomRate.value
})

// Label text
const labelText = computed(() => {
  const rateLabel = effectiveRateLabel.value
  if (isSubscription.value && !props.alwaysShowRate) {
    // 如果有剩余天数，显示天数
    if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
      if (props.daysRemaining <= 0) {
        return t('admin.users.expired')
      }
      return t('admin.users.daysRemaining', { days: props.daysRemaining })
    }
    // 否则显示"订阅"
    return t('groups.subscription')
  }
  return rateLabel
})

const effectiveRateLabel = computed(() => {
  const rate = props.userRateMultiplier ?? props.rateMultiplier
  return rate !== undefined ? `${formatRateMultiplier(rate)}x` : ''
})

// Label style based on type and days remaining
const labelClass = computed(() => {
  const base = 'px-1.5 py-0.5 rounded text-[10px] font-semibold'

  if (!isSubscription.value) {
    // Standard: subtle background (不再为专属倍率使用不同的背景色)
    return `${base} bg-black/10 dark:bg-white/10`
  }

  // 订阅类型：根据剩余天数显示不同颜色
  if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
    if (props.daysRemaining <= 0 || props.daysRemaining <= 3) {
      // 已过期或紧急（<=3天）：红色
      return `${base} bg-red-200/80 text-red-800 dark:bg-red-800/50 dark:text-red-300`
    }
    if (props.daysRemaining <= 7) {
      // 警告（<=7天）：橙色
      return `${base} bg-amber-200/80 text-amber-800 dark:bg-amber-800/50 dark:text-amber-300`
    }
  }

  // 正常状态或无天数：根据平台显示主题色
  if (props.platform === 'anthropic') {
    return `${base} bg-orange-200/60 text-orange-800 dark:bg-orange-800/40 dark:text-orange-300`
  }
  if (props.platform === 'openai') {
    return `${base} bg-emerald-200/60 text-emerald-800 dark:bg-emerald-800/40 dark:text-emerald-300`
  }
  if (props.platform === 'gemini') {
    return `${base} bg-blue-200/60 text-blue-800 dark:bg-blue-800/40 dark:text-blue-300`
  }
  if (props.platform === 'antigravity') {
    return `${base} bg-purple-200/60 text-purple-800 dark:bg-purple-800/40 dark:text-purple-300`
  }
  if (props.platform === 'grok') {
    return `${base} bg-zinc-300/70 text-zinc-800 dark:bg-zinc-700/60 dark:text-zinc-200`
  }
  return `${base} bg-violet-200/60 text-violet-800 dark:bg-violet-800/40 dark:text-violet-300`
})

const peakRateClass = computed(() => {
  return 'px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
})

// Badge color based on platform and subscription type
const badgeClass = computed(() => {
  if (props.platform === 'anthropic') {
    // Claude: orange theme
    return isSubscription.value
      ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
      : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
  } else if (props.platform === 'openai') {
    // OpenAI: green theme
    return isSubscription.value
      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
      : 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400'
  }
  if (props.platform === 'gemini') {
    return isSubscription.value
      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
      : 'bg-sky-50 text-sky-700 dark:bg-sky-900/20 dark:text-sky-400'
  }
  if (props.platform === 'antigravity') {
    return isSubscription.value
      ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
      : 'bg-fuchsia-50 text-fuchsia-700 dark:bg-fuchsia-900/20 dark:text-fuchsia-400'
  }
  if (props.platform === 'grok') {
    return isSubscription.value
      ? 'bg-zinc-200 text-zinc-800 dark:bg-zinc-700 dark:text-zinc-100'
      : 'bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-200'
  }
  // Fallback: original colors
  return isSubscription.value
    ? 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400'
    : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
})
</script>
