<template>
  <section
    v-if="showBalanceSection || showUsageSection"
    data-testid="dashboard-stats"
    class="grid gap-4 xl:items-start"
    :class="showBalanceSection && showUsageSection && !isSimple ? 'xl:grid-cols-[minmax(300px,0.95fr)_minmax(0,2fr)]' : 'xl:grid-cols-1'"
  >
    <div
      v-if="showBalanceSection"
      class="card overflow-hidden p-5"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="mb-3 inline-flex items-center gap-2 rounded-md bg-[var(--apple-surface-elevated)] px-2.5 py-1 text-xs font-semibold text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border)]">
            <span class="h-1.5 w-1.5 rounded-full bg-[var(--apple-blue)]" />
            {{ t('dashboard.balance') }}
          </div>
          <p class="truncate text-3xl font-semibold tracking-normal text-gray-950 dark:text-white">
            {{ formatSettlementAmount(balance, 2) }}
          </p>
          <p class="mt-2 text-sm font-medium text-gray-500 dark:text-gray-400">
            {{ balanceSubtitle }}
          </p>
        </div>
        <div class="rounded-lg bg-[var(--apple-surface-elevated)] p-3 text-[var(--apple-blue)] ring-1 ring-[color:var(--apple-border)]">
          <Icon name="creditCard" size="lg" :stroke-width="2" />
        </div>
      </div>

      <div class="mt-5 grid grid-cols-2 gap-2">
        <div class="rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2">
          <p class="text-[10px] font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('dashboard.serviceStatus') }}
          </p>
          <p class="mt-1 text-sm font-bold text-emerald-600 dark:text-emerald-300">{{ t('dashboard.serviceStable') }}</p>
        </div>
        <button
          type="button"
          class="group flex items-center justify-between gap-2 rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2 text-left text-xs font-semibold text-[var(--apple-text)] transition-colors hover:text-[var(--apple-blue)]"
          @click="balanceEquivalentExpanded = !balanceEquivalentExpanded"
        >
          <span class="min-w-0">
            <span class="block truncate">{{ balanceEquivalentExpanded ? t('dashboard.balanceEquivalent.hide') : t('dashboard.balanceEquivalent.show') }}</span>
          </span>
          <Icon
            name="chevronDown"
            size="xs"
            :stroke-width="2.5"
            class="shrink-0 transition-transform"
            :class="{ 'rotate-180': balanceEquivalentExpanded }"
          />
        </button>
      </div>

      <div
        v-if="balanceEquivalentExpanded"
        class="mt-4 border-t border-[color:var(--apple-border-soft)] pt-4"
      >
        <div class="mb-3 flex items-start gap-2">
          <div class="rounded-md bg-[var(--apple-surface-elevated)] p-1.5 text-[var(--apple-muted)]">
            <Icon name="sparkles" size="xs" :stroke-width="2" />
          </div>
          <div class="min-w-0">
            <p class="text-xs font-bold text-gray-900 dark:text-white">
              {{ t('dashboard.balanceEquivalent.title') }}
            </p>
          </div>
        </div>
        <div v-if="balanceEquivalentItems.length > 0" class="max-h-64 divide-y divide-[color:var(--apple-border-soft)] overflow-y-auto pr-1">
          <div
            v-for="item in balanceEquivalentItems"
            :key="item.group.id"
            class="py-2 text-xs"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="truncate font-bold text-gray-800 dark:text-gray-100">
                  {{ item.group.name }}
                </p>
                <p class="text-[11px] text-gray-500 dark:text-gray-400">
                  {{ item.metaLabel }}
                </p>
              </div>
              <div class="shrink-0 text-right">
                <p class="font-mono text-sm font-semibold text-amber-700 dark:text-amber-300">
                  {{ item.quotaLabel }}
                </p>
                <p class="text-[10px] text-gray-400">{{ item.quotaUnitLabel }}</p>
              </div>
            </div>
          </div>
        </div>
        <p v-else class="text-xs text-[var(--apple-muted)]">
          {{ t('dashboard.balanceEquivalent.empty') }}
        </p>
      </div>
    </div>

    <div v-if="showUsageSection">
      <div class="grid gap-3 sm:grid-cols-2 2xl:grid-cols-3">
        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('dashboard.keyStatus') }}</p>
            <div class="rounded-md bg-[var(--apple-surface-elevated)] p-2 text-[var(--apple-blue)]">
              <Icon name="key" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-semibold text-gray-950 dark:text-white">{{ stats?.total_api_keys || 0 }}</p>
          <p class="mt-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
            {{ t('dashboard.activeKeys', { count: stats?.active_api_keys || 0 }) }}
          </p>
        </div>

        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('dashboard.todayRequests') }}</p>
            <div class="rounded-md bg-[var(--apple-surface-elevated)] p-2 text-emerald-600 dark:text-emerald-300">
              <Icon name="chart" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-semibold text-gray-950 dark:text-white">{{ stats?.today_requests || 0 }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('common.total') }}: {{ formatNumber(stats?.total_requests || 0) }}</p>
        </div>

        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('dashboard.todayCost') }}</p>
            <div class="rounded-md bg-[var(--apple-surface-elevated)] p-2 text-[var(--apple-blue)]">
              <Icon name="dollar" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-semibold text-gray-950 dark:text-white">
            <span class="text-blue-700 dark:text-blue-300" :title="t('dashboard.actual')">{{ formatSettlementAmount(stats?.today_actual_cost || 0, 4) }}</span>
            <span class="text-sm font-semibold text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / {{ formatSettlementAmount(stats?.today_cost || 0, 4) }}</span>
          </p>
          <p class="mt-1 truncate text-xs">
            <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.totalSpend') }}: </span>
            <span class="font-medium text-blue-700 dark:text-blue-300" :title="t('dashboard.actual')">{{ formatSettlementAmount(stats?.total_actual_cost || 0, 4) }}</span>
            <span class="text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / {{ formatSettlementAmount(stats?.total_cost || 0, 4) }}</span>
          </p>
        </div>

        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('dashboard.todayTokens') }}</p>
            <div class="rounded-md bg-[var(--apple-surface-elevated)] p-2 text-amber-600 dark:text-amber-300">
              <Icon name="cube" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-semibold text-gray-950 dark:text-white">{{ formatTokens(stats?.today_tokens || 0) }}</p>
          <p class="mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('dashboard.input') }}: {{ formatTokens(stats?.today_input_tokens || 0) }}</span>
            <span>{{ t('dashboard.output') }}: {{ formatTokens(stats?.today_output_tokens || 0) }}</span>
          </p>
        </div>

        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('dashboard.totalSpend') }}</p>
            <div class="rounded-md bg-[var(--apple-surface-elevated)] p-2 text-slate-600 dark:text-slate-300">
              <Icon name="calculator" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-semibold text-gray-950 dark:text-white">{{ formatSettlementAmount(stats?.total_actual_cost || 0, 4) }}</p>
          <p class="mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('dashboard.totalUsage') }}: {{ formatNumber(stats?.total_requests || 0) }}</span>
            <span>{{ formatTokens(stats?.total_tokens || 0) }} {{ t('dashboard.tokens') }}</span>
          </p>
        </div>

        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('dashboard.serviceStatus') }}</p>
            <div class="rounded-md bg-[var(--apple-surface-elevated)] p-2 text-emerald-600 dark:text-emerald-300">
              <Icon name="shield" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-semibold text-emerald-700 dark:text-emerald-300">{{ t('dashboard.serviceStable') }}</p>
          <div class="mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('dashboard.avgResponse') }}: {{ formatDuration(stats?.average_duration_ms || 0) }}</span>
            <span>{{ t('usage.cacheHitRate') }}: {{ todayCacheStats.ratePercent }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- Row 3: Per-platform breakdown -->
  <section v-if="showPlatformSection && platformCards.length > 0" class="space-y-3">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <h3 class="text-base font-semibold text-[var(--apple-text)]">{{ t('dashboard.serviceTransparency') }}</h3>
      </div>
      <span class="text-xs text-[var(--apple-muted)]">
        {{ t('dashboard.platformCount', { count: sortedPlatforms.length }) }}
      </span>
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="item in platformCards"
        :key="item.platform"
        :class="[
          'card p-3',
          item.isOther
            ? 'border-dashed'
            : ''
        ]"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
          </span>
          <span class="font-mono text-sm text-[var(--apple-blue)]" :title="t('dashboard.actual')">
            {{ formatSettlementAmount(item.total_actual_cost, 4) }}
          </span>
        </div>
        <div class="mt-2 space-y-1 text-xs">
          <div class="flex items-center justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.todayCost') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">{{ formatSettlementAmount(item.today_actual_cost, 4) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.requests') }}</span>
            <span class="font-mono text-gray-700 dark:text-gray-300">
              {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
            </span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('dashboard.tokens') }}</span>
            <span class="font-mono text-gray-700 dark:text-gray-300">
              {{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}
            </span>
          </div>
        </div>

        <!-- Quota 区：仅当 quota 配置存在、非 __other__ 且至少有一个窗口配了 limit 时显示 -->
        <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-3 space-y-1.5 border-t border-[color:var(--apple-border-soft)] pt-2">
          <p class="text-[10px] uppercase tracking-wide text-[var(--apple-muted)]">
            {{ t('dashboard.platformQuota.title') }}
          </p>
          <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
            <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="space-y-0.5">
              <!-- limit=0：完全禁用 -->
              <template v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-600 dark:text-gray-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="font-mono text-red-500">{{ t('dashboard.platformQuota.disabled') }}</span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full w-full rounded-full bg-red-500" />
                </div>
              </template>
              <!-- limit>0：正常用量进度条 -->
              <template v-else>
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-600 dark:text-gray-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="font-mono text-gray-700 dark:text-gray-200">
                    {{ formatSettlementAmountPair((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number, 2) }}
                  </span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div
                    class="h-full rounded-full transition-all"
                    :class="quotaBarClass(calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number))"
                    :style="{ width: calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number) + '%' }"
                  />
                </div>
                <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="text-[10px] text-gray-400">
                  {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
                </p>
              </template>
            </div>
          </template>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { formatSettlementCurrencyAmount, useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import type { UserDashboardStats as UserStatsType } from '@/custom/api/usage'
import type { Group, PlatformQuotaItem } from '@/types'
import { useAppStore } from '@/stores/app'
import { useMinuteNow } from '@/custom/composables/useMinuteNow'
import {
  formatRateMultiplier,
  resolveGroupDiscountFromGroup,
  resolvePublicGroupRateDiscount,
} from '@/custom/utils/groupRateDiscount'
import { isImageGenerationGroup } from '@/custom/utils/imageGenerationGroups'

defineOptions({ name: 'UserDashboardStats' })

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

interface BalanceEquivalentItem {
  group: Group
  rate: number
  rateLabel: string
  quota: number
  quotaLabel: string
  quotaUnitLabel: string
  metaLabel: string
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  mode?: 'all' | 'balance' | 'usage'
  platformQuotas?: PlatformQuotaItem[] | null
  balanceCnyPerCredit?: number
  balanceGroups?: Group[]
  userGroupRates?: Record<number, number>
}>()
const { t } = useI18n()
const appStore = useAppStore()
const now = useMinuteNow()
const {
  settlementCurrency,
  cnyPerCredit,
  formatSettlementAmount,
  formatSettlementAmountPair,
} = useSettlementCurrency()
const balanceEquivalentExpanded = ref(false)

const showBalanceSection = computed(() => !props.isSimple && (props.mode ?? 'all') !== 'usage')
const showUsageSection = computed(() => (props.mode ?? 'all') !== 'balance')
const showPlatformSection = computed(() => !props.isSimple && (props.mode ?? 'all') !== 'balance')

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity'
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

const publicDiscountSummary = computed(() => resolvePublicGroupRateDiscount(
  appStore.cachedPublicSettings?.group_rate_discount ?? null,
  appStore.cachedPublicSettings?.upcoming_group_rate_discount ?? null,
  now.value,
))

const balanceEquivalentItems = computed<BalanceEquivalentItem[]>(() => {
  const uniqueGroups = new Map<number, Group>()
  for (const group of props.balanceGroups ?? []) {
    if (group.status === 'active') {
      uniqueGroups.set(group.id, group)
    }
  }

  return [...uniqueGroups.values()]
    .map((group) => {
      const userRate = props.userGroupRates?.[group.id]
      const baseRate = Number.isFinite(userRate) ? userRate : group.rate_multiplier
      const discount = resolveGroupDiscountFromGroup(
        group,
        baseRate,
        publicDiscountSummary.value?.discount ?? null,
        false,
        now.value,
      )
      const effectiveRate = Number(discount?.discountedRate ?? baseRate)
      if (!Number.isFinite(effectiveRate)) return null
      if (isBalanceEquivalentImageGroup(group)) return null
      if (effectiveRate <= 0) return null
      return buildApiBalanceEquivalentItem(group, effectiveRate)
    })
    .filter((item): item is NonNullable<typeof item> => item !== null)
    .sort((a, b) => b.quota - a.quota || a.group.name.localeCompare(b.group.name))
})

function buildApiBalanceEquivalentItem(group: Group, effectiveRate: number): BalanceEquivalentItem {
  const quota = props.balance / effectiveRate
  const rateLabel = `${formatRateMultiplier(effectiveRate)}x`
  const quotaLabel = t('dashboard.balanceEquivalent.officialAmount', {
    amount: formatBalanceEquivalentAmount(quota),
  })
  return {
    group,
    rate: effectiveRate,
    rateLabel,
    quota,
    quotaLabel,
    quotaUnitLabel: t('dashboard.balanceEquivalent.officialQuota'),
    metaLabel: `${platformLabel(group.platform)} · ${t('dashboard.balanceEquivalent.rate', { rate: rateLabel })}`,
  }
}

function isBalanceEquivalentImageGroup(group: Group): boolean {
  return isImageGenerationGroup(group)
}

function formatBalanceEquivalentAmount(value: number): string {
  const amount = Number.isFinite(value) ? Math.max(0, value) : 0
  return formatSettlementCurrencyAmount(
    amount,
    'USD',
    balanceCnyPerCredit.value,
    undefined,
    amount >= 100 ? 0 : { minimumFractionDigits: 2, maximumFractionDigits: 2 },
  )
}

const buildCacheStats = (input: number, cacheCreate: number, cacheRead: number) => {
  const totalPromptTokens = input + cacheCreate + cacheRead
  return {
    totalPromptTokens,
    ratePercent: totalPromptTokens > 0 ? `${((cacheRead / totalPromptTokens) * 100).toFixed(1)}%` : '-',
  }
}

const todayCacheStats = computed(() => buildCacheStats(
  props.stats?.today_input_tokens ?? 0,
  props.stats?.today_cache_creation_tokens ?? 0,
  props.stats?.today_cache_read_tokens ?? 0,
))

const sortedPlatforms = computed(() => {
  const list = props.stats?.by_platform ?? []
  return [...list].sort((a, b) => b.total_actual_cost - a.total_actual_cost)
})

// 处理"各平台之和 < 总值"的差值：后端按平台聚合时过滤了无法归属平台的行
// （group 与 account 都缺 platform）。这里把差值作为"其他"卡片显式展示，
// 避免 Row 1 总值与 Row 3 平台拆分加总对不上、用户困惑。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  // 建立 by_platform Map
  const byPlat = new Map<string, (typeof sortedPlatforms.value)[number]>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  // 建立 quota Map
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // union 平台集合。后端 by_platform / quota 接口均不会返回 platform='__other__'，
  // 无需显式排除；__other__ 由下方差值补差逻辑单独追加。
  const platforms = new Set<string>([...byPlat.keys(), ...byQuota.keys()])

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
  const cards: FusedPlatformCard[] = []

  for (const p of platforms) {
    const stat = byPlat.get(p)
    cards.push({
      platform: p,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(p),
    })
  }

  // 排序：按 PLATFORM_ORDER，未知平台按名称排序
  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  // __other__ 补差逻辑：只对 by_platform 有 usage 数据的总和计算
  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const sumTotal = cards.reduce((s, c) => s + c.total_actual_cost, 0)
  const sumToday = cards.reduce((s, c) => s + c.today_actual_cost, 0)
  const diffTotal = Math.max(0, total - sumTotal)
  const diffToday = Math.max(0, today - sumToday)

  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

// Quota helpers

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

function quotaVal(q: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return q?.[key]
}

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function quotaBarClass(p: number): string {
  if (p >= 95) return 'bg-red-500'
  if (p >= 75) return 'bg-amber-500'
  return 'bg-green-500'
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const balanceCnyPerCredit = computed(() => {
  const value = props.balanceCnyPerCredit ?? cnyPerCredit.value
  return Number.isFinite(value) && value > 0 ? value : cnyPerCredit.value
})
const balanceSubtitle = computed(() => {
  if (settlementCurrency.value === 'CNY') {
    const base = formatSettlementCurrencyAmount(props.balance, 'USD', balanceCnyPerCredit.value, undefined, 2)
    return `${base} ${t('settlementCurrency.baseCredit')}`
  }
  return t('dashboard.balanceApproxCny', {
    amount: formatSettlementCurrencyAmount(props.balance, 'CNY', balanceCnyPerCredit.value, undefined, 2),
  })
})

const formatNumber = (n: number) => n.toLocaleString()
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>
