<template>
  <section
    class="grid gap-4 xl:items-start"
    :class="isSimple ? 'xl:grid-cols-1' : 'xl:grid-cols-[minmax(300px,0.95fr)_minmax(0,2fr)]'"
  >
    <!-- Balance -->
    <div
      v-if="!isSimple"
      class="relative overflow-hidden rounded-[1.75rem] border border-amber-200/80 bg-gradient-to-br from-amber-50 via-white to-orange-50 p-5 shadow-card dark:border-amber-900/40 dark:from-amber-950/40 dark:via-dark-800/70 dark:to-orange-950/30"
    >
      <div class="pointer-events-none absolute -right-16 -top-20 h-44 w-44 rounded-full bg-amber-300/20 blur-3xl dark:bg-amber-500/10" />
      <div class="pointer-events-none absolute -bottom-20 left-6 h-36 w-36 rounded-full bg-orange-300/20 blur-3xl dark:bg-orange-500/10" />

      <div class="relative flex items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-white/85 px-3 py-1 text-xs font-semibold text-amber-700 shadow-sm ring-1 ring-amber-100 dark:bg-dark-900/70 dark:text-amber-300 dark:ring-amber-900/50">
            <span class="h-1.5 w-1.5 rounded-full bg-amber-500" />
            {{ t('dashboard.balance') }}
          </div>
          <p class="truncate text-3xl font-black tracking-tight text-amber-700 dark:text-amber-300 sm:text-4xl">
            {{ formatSettlementAmount(balance, 2) }}
          </p>
          <p class="mt-2 text-sm font-semibold text-amber-700/80 dark:text-amber-300/80">
            {{ balanceSubtitle }}
          </p>
        </div>
        <div class="rounded-2xl bg-amber-600/10 p-3 text-amber-700 ring-1 ring-amber-500/10 dark:bg-amber-400/10 dark:text-amber-300">
          <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
          </svg>
        </div>
      </div>

      <div class="relative mt-5 grid grid-cols-2 gap-2">
        <div class="rounded-2xl bg-white/75 px-3 py-2 ring-1 ring-amber-100 dark:bg-dark-900/50 dark:ring-amber-900/40">
          <p class="text-[10px] font-semibold uppercase tracking-wide text-amber-700/70 dark:text-amber-300/70">
            {{ t('common.status') }}
          </p>
          <p class="mt-1 text-sm font-bold text-gray-900 dark:text-white">{{ t('common.available') }}</p>
        </div>
        <button
          type="button"
          class="group flex items-center justify-between gap-2 rounded-2xl bg-amber-600 px-3 py-2 text-left text-xs font-bold text-white shadow-lg shadow-amber-600/20 transition hover:-translate-y-0.5 hover:bg-amber-500 dark:bg-amber-500 dark:hover:bg-amber-400"
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
        class="relative mt-4 rounded-2xl border border-amber-100 bg-white/80 p-3 backdrop-blur dark:border-amber-900/40 dark:bg-dark-900/60"
      >
        <div class="mb-3 flex items-start gap-2">
          <div class="rounded-xl bg-amber-100 p-1.5 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
            <Icon name="sparkles" size="xs" :stroke-width="2" />
          </div>
          <div class="min-w-0">
            <p class="text-xs font-bold text-gray-900 dark:text-white">
              {{ t('dashboard.balanceEquivalent.title') }}
            </p>
            <p class="mt-0.5 text-[11px] leading-4 text-gray-500 dark:text-gray-400">
              {{ t('dashboard.balanceEquivalent.description') }}
            </p>
          </div>
        </div>
        <div v-if="balanceEquivalentItems.length > 0" class="max-h-64 space-y-2 overflow-y-auto pr-1">
          <div
            v-for="item in balanceEquivalentItems"
            :key="item.group.id"
            class="rounded-xl bg-amber-50/80 px-3 py-2 text-xs ring-1 ring-amber-100/70 dark:bg-amber-950/20 dark:ring-amber-900/30"
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
                <p class="font-mono text-sm font-black text-amber-700 dark:text-amber-300">
                  {{ item.quotaLabel }}
                </p>
                <p class="text-[10px] text-gray-400">{{ item.quotaUnitLabel }}</p>
              </div>
            </div>
            <p class="mt-1.5 rounded-lg bg-white/60 px-2 py-1 text-[10px] leading-4 text-amber-800/80 dark:bg-dark-900/40 dark:text-amber-200/80">
              {{ item.formulaLabel }}
            </p>
          </div>
        </div>
        <p v-else class="rounded-xl bg-gray-50 px-3 py-2 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
          {{ t('dashboard.balanceEquivalent.empty') }}
        </p>
      </div>
    </div>

    <!-- KPI Matrix -->
    <div class="rounded-[1.75rem] border border-gray-100 bg-white/85 p-3 shadow-card dark:border-dark-700/60 dark:bg-dark-800/60">
      <div class="mb-2 flex items-center justify-between px-1">
        <div>
          <p class="text-sm font-bold text-gray-900 dark:text-white">{{ t('dashboard.todayOverview') }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.accountSnapshot') }}</p>
        </div>
        <span class="rounded-full bg-gray-100 px-2.5 py-1 text-[11px] font-semibold text-gray-500 dark:bg-dark-700 dark:text-gray-300">
          {{ t('dashboard.last7Days') }}
        </span>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 2xl:grid-cols-3">
        <div class="rounded-2xl border border-blue-100 bg-blue-50/60 p-4 dark:border-blue-900/40 dark:bg-blue-950/20">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-blue-700 dark:text-blue-300">{{ t('dashboard.apiKeys') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-blue-600 dark:bg-dark-900/60 dark:text-blue-300">
              <Icon name="key" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-black text-gray-900 dark:text-white">{{ stats?.total_api_keys || 0 }}</p>
          <p class="mt-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">{{ stats?.active_api_keys || 0 }} {{ t('common.active') }}</p>
        </div>

        <div class="rounded-2xl border border-green-100 bg-green-50/60 p-4 dark:border-green-900/40 dark:bg-green-950/20">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-green-700 dark:text-green-300">{{ t('dashboard.todayRequests') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-green-600 dark:bg-dark-900/60 dark:text-green-300">
              <Icon name="chart" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-black text-gray-900 dark:text-white">{{ stats?.today_requests || 0 }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('common.total') }}: {{ formatNumber(stats?.total_requests || 0) }}</p>
        </div>

        <div class="rounded-2xl border border-fuchsia-100 bg-fuchsia-50/60 p-4 dark:border-fuchsia-900/40 dark:bg-fuchsia-950/20">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-fuchsia-700 dark:text-fuchsia-300">{{ t('dashboard.todayCost') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-fuchsia-600 dark:bg-dark-900/60 dark:text-fuchsia-300">
              <Icon name="dollar" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-black text-gray-900 dark:text-white">
            <span class="text-fuchsia-700 dark:text-fuchsia-300" :title="t('dashboard.actual')">{{ formatSettlementAmount(stats?.today_actual_cost || 0, 4) }}</span>
            <span class="text-sm font-semibold text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / {{ formatSettlementAmount(stats?.today_cost || 0, 4) }}</span>
          </p>
          <p class="mt-1 truncate text-xs">
            <span class="text-gray-500 dark:text-gray-400">{{ t('common.total') }}: </span>
            <span class="font-medium text-fuchsia-700 dark:text-fuchsia-300" :title="t('dashboard.actual')">{{ formatSettlementAmount(stats?.total_actual_cost || 0, 4) }}</span>
            <span class="text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / {{ formatSettlementAmount(stats?.total_cost || 0, 4) }}</span>
          </p>
        </div>

        <div class="rounded-2xl border border-amber-100 bg-amber-50/60 p-4 dark:border-amber-900/40 dark:bg-amber-950/20">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-amber-700 dark:text-amber-300">{{ t('dashboard.todayTokens') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-amber-600 dark:bg-dark-900/60 dark:text-amber-300">
              <Icon name="cube" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-black text-gray-900 dark:text-white">{{ formatTokens(stats?.today_tokens || 0) }}</p>
          <p class="mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('dashboard.input') }}: {{ formatTokens(stats?.today_input_tokens || 0) }}</span>
            <span>{{ t('dashboard.output') }}: {{ formatTokens(stats?.today_output_tokens || 0) }}</span>
          </p>
        </div>

        <div class="rounded-2xl border border-indigo-100 bg-indigo-50/60 p-4 dark:border-indigo-900/40 dark:bg-indigo-950/20">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-indigo-700 dark:text-indigo-300">{{ t('dashboard.totalTokens') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-indigo-600 dark:bg-dark-900/60 dark:text-indigo-300">
              <Icon name="database" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-black text-gray-900 dark:text-white">{{ formatTokens(stats?.total_tokens || 0) }}</p>
          <p class="mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('dashboard.input') }}: {{ formatTokens(stats?.total_input_tokens || 0) }}</span>
            <span>{{ t('dashboard.output') }}: {{ formatTokens(stats?.total_output_tokens || 0) }}</span>
          </p>
        </div>

        <div class="rounded-2xl border border-sky-100 bg-sky-50/60 p-4 dark:border-sky-900/40 dark:bg-sky-950/20">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-sky-700 dark:text-sky-300">{{ t('dashboard.cacheToday') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-sky-600 dark:bg-dark-900/60 dark:text-sky-300">
              <Icon name="database" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="text-2xl font-black text-sky-700 dark:text-sky-300">{{ formatTokens(stats?.today_cache_read_tokens || 0) }}</p>
          <div class="mt-1 space-y-0.5 text-xs">
            <p class="font-semibold text-sky-700 dark:text-sky-300">{{ t('usage.cacheHit') }}</p>
            <p class="flex items-center gap-1 text-gray-500 dark:text-gray-400">
              <span>{{ t('usage.cacheCreate') }}: {{ formatTokens(stats?.today_cache_creation_tokens || 0) }}</span>
              <span
                class="inline-flex cursor-help items-center text-gray-400 transition-colors hover:text-sky-600 dark:text-gray-500 dark:hover:text-sky-400"
                :title="t('usage.openaiCacheCreateNote')"
              >
                <Icon name="questionCircle" size="xs" :stroke-width="2" />
              </span>
            </p>
            <p class="font-semibold text-sky-700 dark:text-sky-300">{{ t('usage.cacheHitRate') }}: {{ todayCacheStats.ratePercent }}</p>
          </div>
        </div>

        <div class="rounded-2xl border border-violet-100 bg-violet-50/60 p-4 dark:border-violet-900/40 dark:bg-violet-950/20 sm:col-span-2 2xl:col-span-1">
          <div class="mb-3 flex items-center justify-between">
            <p class="text-xs font-semibold text-violet-700 dark:text-violet-300">{{ t('dashboard.performance') }}</p>
            <div class="rounded-xl bg-white/80 p-2 text-violet-600 dark:bg-dark-900/60 dark:text-violet-300">
              <Icon name="bolt" size="sm" :stroke-width="2" />
            </div>
          </div>
          <div class="grid grid-cols-3 gap-2">
            <div class="rounded-xl bg-white/70 px-2.5 py-2 dark:bg-dark-900/40">
              <p class="text-[10px] font-bold text-gray-400">RPM</p>
              <p class="mt-1 font-mono text-lg font-black text-gray-900 dark:text-white">{{ formatTokens(stats?.rpm || 0) }}</p>
            </div>
            <div class="rounded-xl bg-white/70 px-2.5 py-2 dark:bg-dark-900/40">
              <p class="text-[10px] font-bold text-gray-400">TPM</p>
              <p class="mt-1 font-mono text-lg font-black text-violet-700 dark:text-violet-300">{{ formatTokens(stats?.tpm || 0) }}</p>
            </div>
            <div class="rounded-xl bg-white/70 px-2.5 py-2 dark:bg-dark-900/40">
              <p class="text-[10px] font-bold text-gray-400">{{ t('dashboard.avgResponse') }}</p>
              <p class="mt-1 font-mono text-lg font-black text-rose-600 dark:text-rose-300">{{ formatDuration(stats?.average_duration_ms || 0) }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- Row 3: Per-platform breakdown -->
  <div v-if="!isSimple && platformCards.length > 0" class="card p-4">
    <div class="mb-3 flex items-center justify-between">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.platformBreakdown') }}</h3>
      <span class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('dashboard.platformCount', { count: sortedPlatforms.length }) }}
      </span>
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="item in platformCards"
        :key="item.platform"
        :class="[
          'rounded-lg border p-3',
          item.isOther
            ? 'border-dashed border-gray-300 bg-gray-50 dark:border-dark-500 dark:bg-dark-700/30'
            : 'border-gray-200 dark:border-dark-600'
        ]"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
          </span>
          <span class="font-mono text-sm text-purple-600 dark:text-purple-400" :title="t('dashboard.actual')">
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
        <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-3 space-y-1.5 border-t border-gray-200 pt-2 dark:border-dark-700">
          <p class="text-[10px] uppercase tracking-wide text-gray-400">
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
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
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
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
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
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { formatSettlementCurrencyAmount, useSettlementCurrency } from '@/composables/useSettlementCurrency'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { Group, PlatformQuotaItem } from '@/types'
import { useAppStore } from '@/stores/app'
import { useMinuteNow } from '@/composables/useMinuteNow'
import {
  formatRateMultiplier,
  resolveGroupDiscountFromGroup,
  resolvePublicGroupRateDiscount,
} from '@/utils/groupRateDiscount'

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
  formulaLabel: string
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
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
    formulaLabel: t('dashboard.balanceEquivalent.apiFormula', {
      balance: formatBalanceEquivalentAmount(props.balance),
      rate: rateLabel,
      quota: quotaLabel,
    }),
  }
}

function isBalanceEquivalentImageGroup(group: Group): boolean {
  return group.name.includes('生图')
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

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity']
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
