<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { opsAPI, type OpenAISchedulerAccountStatus, type OpenAISchedulerStatusResponse } from '@/api/admin/ops'
import { useAppStore } from '@/stores/app'

interface Props {
  groupIdFilter?: number | null
  refreshToken: number
}

const props = withDefaults(defineProps<Props>(), {
  groupIdFilter: null
})

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const status = ref<OpenAISchedulerStatusResponse | null>(null)
const model = ref('gpt-5.5')
const endpoint = ref('/v1/responses')

const endpointOptions = computed(() => [
  { value: '/v1/responses', label: '/responses' },
  { value: '/v1/responses/compact', label: '/responses/compact' },
  { value: '/v1/embeddings', label: '/embeddings' }
])

const summary = computed(() => status.value?.summary)
const groups = computed(() => status.value?.groups ?? [])

function safePercent(v: number | undefined): string {
  const n = typeof v === 'number' && Number.isFinite(v) ? v : 0
  return `${Math.round(n * 100)}%`
}

function formatDuration(seconds?: number): string {
  const safe = Math.max(0, Math.floor(seconds || 0))
  if (safe < 60) return `${safe}s`
  const minutes = Math.floor(safe / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  return `${hours}h`
}

function accountBadgeClass(row: OpenAISchedulerAccountStatus): string {
  if (row.is_available) return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
  if (row.circuit_state !== 'closed') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  if (row.load_rate >= 100) return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
}

function accountBadgeText(row: OpenAISchedulerAccountStatus): string {
  if (row.is_available) return t('admin.ops.openAIScheduler.available')
  if (row.runtime_circuit_remaining_sec) return formatDuration(row.runtime_circuit_remaining_sec)
  return translateSchedulerReason(row.block_reason || 'unavailable')
}

function translateSchedulerReason(reason?: string): string {
  const value = (reason || '').trim()
  if (!value) return ''
  const key = `admin.ops.openAIScheduler.reason.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

function accountDetailText(row: OpenAISchedulerAccountStatus): string {
  if (row.temp_unschedulable_reason) return translateSchedulerReason(row.temp_unschedulable_reason)
  if (row.temp_unschedulable_status_code === 0) return translateSchedulerReason('network_or_stream_interruption')
  if (row.scheduler_last_failure_reason) return translateSchedulerReason(row.scheduler_last_failure_reason)
  return ''
}

async function loadData() {
  loading.value = true
  try {
    status.value = await opsAPI.getOpenAISchedulerStatus({
      model: model.value.trim(),
      endpoint: endpoint.value,
      group_id: props.groupIdFilter ?? null
    })
  } catch (err: any) {
    console.error('[OpsOpenAISchedulerStatusCard] Failed to load status', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.openAIScheduler.loadFailed'))
    status.value = null
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.refreshToken, props.groupIdFilter],
  () => loadData(),
  { immediate: true }
)

watch([model, endpoint], () => loadData())
</script>

<template>
  <div class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="min-w-0">
        <h3 class="text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.openAIScheduler.title') }}</h3>
        <div class="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-gray-500 dark:text-gray-400">
          <span>{{ t('admin.ops.openAIScheduler.switchRate') }} {{ safePercent(status?.metrics?.account_switch_rate) }}</span>
          <span>·</span>
          <span>{{ t('admin.ops.openAIScheduler.stickyRate') }} {{ safePercent(status?.metrics?.sticky_hit_ratio) }}</span>
          <span>·</span>
          <span>{{ t('admin.ops.openAIScheduler.latency') }} {{ Math.round(status?.metrics?.scheduler_latency_ms_avg || 0) }}ms</span>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <input
          v-model="model"
          class="h-8 w-36 rounded-lg border border-gray-200 bg-white px-2 text-xs font-medium text-gray-700 outline-none focus:border-blue-500 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200"
          :aria-label="t('admin.ops.openAIScheduler.model')"
        />
        <select
          v-model="endpoint"
          class="h-8 rounded-lg border border-gray-200 bg-white px-2 text-xs font-medium text-gray-700 outline-none focus:border-blue-500 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200"
          :aria-label="t('admin.ops.openAIScheduler.endpoint')"
        >
          <option v-for="option in endpointOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
        <button
          class="flex h-8 items-center gap-1 rounded-lg bg-gray-100 px-2 text-[11px] font-semibold text-gray-700 transition-colors hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600"
          :disabled="loading"
          :title="t('common.refresh')"
          @click="loadData"
        >
          <svg class="h-3 w-3" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>
    </div>

    <div v-if="summary" class="mb-4 grid grid-cols-2 gap-2 md:grid-cols-6">
      <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900">
        <div class="text-[10px] font-semibold uppercase text-gray-400">{{ t('admin.ops.openAIScheduler.accounts') }}</div>
        <div class="mt-1 font-mono text-lg font-bold text-gray-900 dark:text-white">{{ summary.account_count }}</div>
      </div>
      <div class="rounded-xl bg-green-50 p-3 dark:bg-green-900/20">
        <div class="text-[10px] font-semibold uppercase text-green-600 dark:text-green-400">{{ t('admin.ops.openAIScheduler.available') }}</div>
        <div class="mt-1 font-mono text-lg font-bold text-green-700 dark:text-green-300">{{ summary.available_count }}</div>
      </div>
      <div class="rounded-xl bg-amber-50 p-3 dark:bg-amber-900/20">
        <div class="text-[10px] font-semibold uppercase text-amber-600 dark:text-amber-400">{{ t('admin.ops.openAIScheduler.blocked') }}</div>
        <div class="mt-1 font-mono text-lg font-bold text-amber-700 dark:text-amber-300">{{ summary.blocked_count }}</div>
      </div>
      <div class="rounded-xl bg-red-50 p-3 dark:bg-red-900/20">
        <div class="text-[10px] font-semibold uppercase text-red-600 dark:text-red-400">{{ t('admin.ops.openAIScheduler.circuit') }}</div>
        <div class="mt-1 font-mono text-lg font-bold text-red-700 dark:text-red-300">{{ summary.circuit_open_count }}</div>
      </div>
      <div class="rounded-xl bg-blue-50 p-3 dark:bg-blue-900/20">
        <div class="text-[10px] font-semibold uppercase text-blue-600 dark:text-blue-400">{{ t('admin.ops.openAIScheduler.halfOpen') }}</div>
        <div class="mt-1 font-mono text-lg font-bold text-blue-700 dark:text-blue-300">{{ summary.half_open_count }}</div>
      </div>
      <div class="rounded-xl bg-purple-50 p-3 dark:bg-purple-900/20">
        <div class="text-[10px] font-semibold uppercase text-purple-600 dark:text-purple-400">{{ t('admin.ops.openAIScheduler.full') }}</div>
        <div class="mt-1 font-mono text-lg font-bold text-purple-700 dark:text-purple-300">{{ summary.concurrency_full_count }}</div>
      </div>
    </div>

    <div v-if="!status?.enabled" class="rounded-xl border border-dashed border-gray-200 p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
      {{ t('admin.ops.openAIScheduler.disabled') }}
    </div>
    <div v-else-if="groups.length === 0" class="rounded-xl border border-dashed border-gray-200 p-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
      {{ t('admin.ops.openAIScheduler.empty') }}
    </div>
    <div v-else class="space-y-3">
      <div v-for="group in groups" :key="group.group_id" class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
        <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
          <div class="min-w-0">
            <div class="truncate text-xs font-bold text-gray-900 dark:text-white" :title="group.group_name">{{ group.group_name }}</div>
            <div class="mt-0.5 font-mono text-[10px] text-gray-400">#{{ group.group_id }}</div>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-[10px] font-semibold">
            <span class="text-green-600 dark:text-green-400">{{ group.available_count }}/{{ group.total_accounts }}</span>
            <span v-if="group.circuit_open_count > 0" class="text-red-600 dark:text-red-400">{{ t('admin.ops.openAIScheduler.circuit') }} {{ group.circuit_open_count }}</span>
            <span v-if="group.concurrency_full_count > 0" class="text-purple-600 dark:text-purple-400">{{ t('admin.ops.openAIScheduler.full') }} {{ group.concurrency_full_count }}</span>
          </div>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-dark-700">
          <div v-for="row in group.accounts" :key="row.account_id" class="grid grid-cols-1 gap-2 px-3 py-2 md:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.1fr)] md:items-center">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate text-xs font-bold text-gray-900 dark:text-white" :title="row.account_name">{{ row.account_name }}</span>
                <span class="font-mono text-[10px] text-gray-400">#{{ row.account_id }}</span>
              </div>
              <div class="mt-1 flex flex-wrap gap-1 text-[10px] text-gray-500 dark:text-gray-400">
                <span>{{ row.role }}</span>
                <span>sort {{ row.sort_order }}</span>
                <span>w{{ row.weight }}</span>
                <span>{{ row.account_type }}</span>
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-2 text-[10px]">
              <span :class="['rounded px-1.5 py-0.5 font-semibold', accountBadgeClass(row)]">{{ accountBadgeText(row) }}</span>
              <span v-if="accountDetailText(row)" class="truncate text-gray-500 dark:text-gray-400" :title="accountDetailText(row)">{{ accountDetailText(row) }}</span>
            </div>
            <div class="min-w-0">
              <div class="mb-1 flex items-center justify-between text-[10px] text-gray-500 dark:text-gray-400">
                <span>{{ row.current_concurrency }}/{{ row.max_concurrency }}</span>
                <span>{{ row.load_rate }}%</span>
              </div>
              <div class="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                <div class="h-full rounded-full bg-blue-500 transition-all" :style="{ width: `${Math.max(0, Math.min(100, row.load_rate))}%` }"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
