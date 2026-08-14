<template>
  <div class="public-gateway-shell">
    <PublicGatewayHeader />

    <main class="public-gateway-container pb-16">
      <section class="public-gateway-hero">
        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.45fr)] lg:items-end">
          <div class="min-w-0">
            <p class="public-gateway-kicker">{{ copy.kicker }}</p>
            <h1 class="public-gateway-title">{{ copy.title }}</h1>
            <p class="public-gateway-lead">{{ copy.lead }}</p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span v-for="item in trustSignals" :key="item" class="public-gateway-chip">
                {{ item }}
              </span>
            </div>
          </div>

          <aside class="public-gateway-panel p-4">
            <div class="grid grid-cols-2 gap-2">
              <div v-for="item in summaryItems" :key="item.label" class="public-gateway-stat">
                <p>{{ item.label }}</p>
                <strong>{{ item.value }}</strong>
              </div>
            </div>
            <div class="mt-4 flex justify-end">
              <button class="public-gateway-secondary rounded-lg" :disabled="loading" @click="loadStatus">
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                {{ copy.refresh }}
              </button>
            </div>
          </aside>
        </div>
      </section>

      <section class="public-gateway-panel mb-5 p-3 sm:p-4">
        <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
          <div class="relative min-w-0">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--gw-text-3)]" />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="copy.search"
              class="input h-11 pl-10"
            />
          </div>

          <div class="grid min-w-0 gap-2 sm:grid-cols-2 lg:min-w-[420px]">
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.providerFilter }}</span>
              <select v-model="providerFilter" class="input h-11">
                <option v-for="option in providerFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.statusFilter }}</span>
              <select v-model="statusFilter" class="input h-11">
                <option v-for="option in statusFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-[color:var(--gw-border)] pt-3">
          <div class="flex flex-wrap gap-2 text-xs font-semibold text-[var(--gw-text-3)]">
            <span class="public-gateway-chip">{{ copy.filtered }} {{ filteredItems.length }} / {{ items.length }}</span>
            <span class="public-gateway-chip">{{ copy.autoRefresh }} {{ countdownSeconds }}s</span>
            <span v-if="lastUpdatedLabel" data-test="public-last-updated" class="public-gateway-chip">{{ copy.updated }} {{ lastUpdatedLabel }}</span>
          </div>
          <button class="public-gateway-secondary rounded-lg" type="button" @click="resetFilters">
            {{ copy.reset }}
          </button>
        </div>
      </section>

      <section class="mb-5 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-[color:var(--gw-border)] bg-[var(--gw-panel)] p-3">
        <div class="text-sm text-[var(--gw-text-2)]">
          {{ copy.sorting }}
        </div>
        <span
          data-test="public-overall-status"
          :class="[
            'inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold',
            overallStatus === 'operational' ? 'bg-[var(--gw-accent-soft)] text-[var(--gw-success)]' : 'bg-[var(--gw-gold-soft)] text-[var(--gw-warning)]'
          ]"
        >
          <span
            :class="[
              'mr-1.5 h-1.5 w-1.5 rounded-full',
              overallStatus === 'operational' ? 'bg-[var(--gw-success)]' : 'bg-[var(--gw-warning)]'
            ]"
          />
          {{ overallStatus === 'operational' ? copy.globalOperational : copy.globalDegraded }}
        </span>
      </section>

      <section v-if="errorMessage" class="public-gateway-alert mb-5">
        {{ errorMessage }}
      </section>

      <MonitorCardGrid
        :items="sortedItems"
        window="7d"
        :countdown-seconds="countdownSeconds"
        :loading="loading"
        :detail-cache="{}"
        @card-click="noop"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import MonitorCardGrid from '@/custom/user/monitor/WegooMonitorCardGrid.vue'
import publicGatewayAPI from '@/custom/api/publicGateway'
import type { MonitorStatus, Provider, UserMonitorView } from '@/api/channelMonitor'
import { MONITOR_STATUSES, STATUS_OPERATIONAL } from '@/constants/channelMonitor'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import PublicGatewayHeader from './PublicGatewayHeader.vue'
import './publicGateway.css'

const { locale } = useI18n()
const { providerLabel, statusLabel } = useChannelMonitorFormat()

const REFRESH_SECONDS = 30

const items = ref<UserMonitorView[]>([])
const loading = ref(false)
const errorMessage = ref('')
const searchQuery = ref('')
const providerFilter = ref<'all' | Provider>('all')
const statusFilter = ref<'all' | MonitorStatus>('all')
const countdownSeconds = ref(REFRESH_SECONDS)
const lastUpdatedAt = ref<Date | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const copy = computed(() => locale.value.startsWith('zh')
  ? {
    kicker: 'Service Status · Public Health',
    title: '公开服务状态',
    lead: '展示模型服务的可用性、响应速度和近期状态。展示顺序与后台渠道监控排序保持一致。',
    refresh: '刷新',
    total: '服务项',
    healthy: '正常',
    degradedCount: '波动',
    latency: '平均延迟',
    search: '搜索服务或模型',
    providerFilter: '供应商',
    statusFilter: '状态',
    allProviders: '全部供应商',
    allStatuses: '全部状态',
    filtered: '当前展示',
    autoRefresh: '自动刷新',
    updated: '更新于',
    reset: '重置筛选',
    sorting: '排序规则：按后台渠道监控中的展示顺序排列。',
    operational: '服务可用',
    degraded: '部分波动',
    globalOperational: '全站服务可用',
    globalDegraded: '全站部分波动',
    empty: '状态正在建立',
    loadFailed: '公开状态接口暂时不可用，请稍后刷新；登录控制台可查看更完整的状态信息。',
  }
  : {
    kicker: 'Service Status · Public Health',
    title: 'Public Service Status',
    lead: 'View model-service availability, response speed, and recent status. Display order follows the admin channel monitor order.',
    refresh: 'Refresh',
    total: 'Services',
    healthy: 'Healthy',
    degradedCount: 'Degraded',
    latency: 'Avg Latency',
    search: 'Search services or models',
    providerFilter: 'Provider',
    statusFilter: 'Status',
    allProviders: 'All Providers',
    allStatuses: 'All Statuses',
    filtered: 'Showing',
    autoRefresh: 'Auto Refresh',
    updated: 'Updated',
    reset: 'Reset Filters',
    sorting: 'Sorting: follows the admin channel monitor display order.',
    operational: 'Operational',
    degraded: 'Degraded',
    globalOperational: 'All services operational',
    globalDegraded: 'Some services degraded',
    empty: 'Status is being prepared',
    loadFailed: 'The public status endpoint is temporarily unavailable. Refresh later or sign in for more detail.',
  })

const trustSignals = computed(() => locale.value.startsWith('zh')
  ? ['可用性', '30 天趋势', '延迟参考', '后台排序', '状态透明']
  : ['Availability', '30-day trend', 'Latency reference', 'Admin order', 'Transparent status'])

const filteredItems = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (providerFilter.value !== 'all' && item.provider !== providerFilter.value) return false
    if (statusFilter.value !== 'all' && item.primary_status !== statusFilter.value) return false
    if (!q) return true
    return [
      item.name,
      item.provider,
      providerLabel(item.provider),
      item.group_name,
      item.primary_model,
      ...((item.extra_models || []).map((model) => model.model)),
    ]
      .filter(Boolean)
      .some((value) => value.toLowerCase().includes(q))
  })
})

const sortedItems = computed(() => filteredItems.value)

const healthyCount = computed(() => filteredItems.value.filter((item) => item.primary_status === STATUS_OPERATIONAL).length)
const degradedCount = computed(() => Math.max(filteredItems.value.length - healthyCount.value, 0))
const overallStatus = computed(() =>
  items.value.length > 0 && items.value.every((item) => item.primary_status === STATUS_OPERATIONAL) && !errorMessage.value
    ? 'operational'
    : 'degraded'
)
const avgLatency = computed(() => {
  const samples = filteredItems.value
    .map((item) => item.primary_latency_ms)
    .filter((value): value is number => typeof value === 'number' && Number.isFinite(value))
  if (samples.length === 0) return null
  return samples.reduce((sum, value) => sum + value, 0) / samples.length
})

const summaryItems = computed(() => [
  { label: copy.value.total, value: filteredItems.value.length },
  { label: copy.value.healthy, value: healthyCount.value },
  { label: copy.value.degradedCount, value: degradedCount.value },
  { label: copy.value.latency, value: formatLatency(avgLatency.value) },
])

const providerFilterOptions = computed(() => [
  { value: 'all', label: copy.value.allProviders },
  ...Array.from(new Set(items.value.map((item) => item.provider)))
    .sort()
    .map((provider) => ({ value: provider, label: providerLabel(provider) })),
])

const statusFilterOptions = computed(() => [
  { value: 'all', label: copy.value.allStatuses },
  ...MONITOR_STATUSES.map((status) => ({ value: status, label: statusLabel(status) })),
])

const lastUpdatedLabel = computed(() => {
  if (!lastUpdatedAt.value) return ''
  return new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(lastUpdatedAt.value)
})

function parseTimestamp(value: string | null | undefined): Date | null {
  if (!value) return null
  const parsed = new Date(value)
  return Number.isFinite(parsed.getTime()) ? parsed : null
}

function latestTimelineDate(rows: UserMonitorView[]): Date | null {
  let latest: Date | null = null
  for (const item of rows) {
    for (const point of item.timeline || []) {
      const parsed = parseTimestamp(point.checked_at)
      if (parsed && (!latest || parsed > latest)) {
        latest = parsed
      }
    }
  }
  return latest
}

function formatLatency(value: number | null): string {
  if (value === null) return '-'
  return `${Math.round(value)}ms`
}

function noop() {
  // Public status cards are read-only summaries; authenticated users can open full detail in /monitor.
}

function resetFilters() {
  searchQuery.value = ''
  providerFilter.value = 'all'
  statusFilter.value = 'all'
}

async function loadStatus() {
  if (loading.value) return
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await publicGatewayAPI.getPublicChannelMonitors()
    items.value = response.items || []
    lastUpdatedAt.value = parseTimestamp(response.last_updated_at) || latestTimelineDate(items.value)
  } catch {
    errorMessage.value = copy.value.loadFailed
    items.value = []
    lastUpdatedAt.value = null
  } finally {
    countdownSeconds.value = REFRESH_SECONDS
    loading.value = false
  }
}

onMounted(() => {
  loadStatus()
  refreshTimer = setInterval(() => {
    countdownSeconds.value = Math.max(countdownSeconds.value - 1, 0)
    if (countdownSeconds.value === 0) {
      loadStatus()
    }
  }, 1000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>
