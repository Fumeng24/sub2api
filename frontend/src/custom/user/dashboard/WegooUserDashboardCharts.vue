<template>
  <div data-testid="dashboard-charts" class="space-y-5">
    <!-- Date Range Filter -->
    <div class="card p-4">
      <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto_auto] lg:items-center">
        <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
          <span class="text-sm font-medium text-[var(--apple-text)]">{{ t('dashboard.timeRange') }}:</span>
          <DateRangePicker :start-date="startDate" :end-date="endDate" @update:startDate="$emit('update:startDate', $event)" @update:endDate="$emit('update:endDate', $event)" @change="$emit('dateRangeChange', $event)" />
        </div>
        <div class="flex min-w-0 items-center gap-2 lg:justify-end">
          <span class="text-sm font-medium text-[var(--apple-text)]">{{ t('dashboard.granularity') }}:</span>
          <div class="w-full sm:w-28">
            <Select :model-value="granularity" :options="[{value:'day', label:t('dashboard.day')}, {value:'hour', label:t('dashboard.hour')}]" @update:model-value="$emit('update:granularity', $event)" @change="$emit('granularityChange')" />
          </div>
        </div>
        <button type="button" @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary w-full sm:w-auto">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh') }}
        </button>
      </div>
    </div>

    <!-- Charts Grid -->
    <div class="grid grid-cols-1 gap-5 lg:grid-cols-2">
      <!-- Model Distribution Chart -->
      <div class="card relative overflow-hidden p-4">
        <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/60 backdrop-blur-sm dark:bg-dark-800/60">
          <LoadingSpinner size="md" />
        </div>
        <div class="mb-4">
          <h3 class="text-sm font-semibold text-[var(--apple-text)]">{{ t('dashboard.modelDistribution') }}</h3>
        </div>
        <div class="flex flex-col gap-4 xl:flex-row xl:items-center xl:gap-6">
          <div class="mx-auto h-48 w-48 shrink-0 xl:mx-0">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div v-else class="flex h-full items-center justify-center text-sm text-[var(--apple-muted)]">{{ t('dashboard.noDataAvailable') }}</div>
          </div>
          <div v-if="models.length > 0" class="table-container max-h-48 min-w-0 flex-1 overflow-auto">
            <table class="table text-xs">
              <thead>
                <tr>
                  <th class="text-left">{{ t('dashboard.model') }}</th>
                  <th class="text-right">{{ t('dashboard.requests') }}</th>
                  <th class="text-right">{{ t('dashboard.tokens') }}</th>
                  <th class="text-right">{{ t('dashboard.actual') }}</th>
                  <th class="text-right">{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.model">
                  <td class="max-w-[100px] truncate font-medium" :title="model.model">{{ model.model }}</td>
                  <td class="text-right">{{ formatNumber(model.requests) }}</td>
                  <td class="text-right">{{ formatTokens(model.total_tokens) }}</td>
                  <td class="text-right text-emerald-600 dark:text-emerald-400">{{ formatSettlementAmount(model.actual_cost, 4) }}</td>
                  <td class="text-right text-[var(--apple-muted)]">{{ formatSettlementAmount(model.cost, 4) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="card relative overflow-hidden p-4">
        <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/60 backdrop-blur-sm dark:bg-dark-800/60">
          <LoadingSpinner size="md" />
        </div>
        <div class="mb-4">
          <h3 class="text-sm font-semibold text-[var(--apple-text)]">{{ t('dashboard.usageTrend') }}</h3>
        </div>
        <div class="h-56">
          <Line v-if="trendChartData" :data="trendChartData" :options="trendChartOptions" />
          <div v-else class="flex h-full items-center justify-center text-sm text-[var(--apple-muted)]">{{ t('dashboard.noDataAvailable') }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/custom/common/WegooDateRangePicker.vue'
import Select from '@/custom/common/WegooSelect.vue'
import Icon from '@/components/icons/Icon.vue'
import { Doughnut, Line } from 'vue-chartjs'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'

defineOptions({ name: 'UserDashboardCharts' })

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()
const { formatSettlementAmount } = useSettlementCurrency()

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: ['#3b82f6', '#10b981', '#f59e0b', '#64748b', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16']
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}

const chartPalette = computed(() => {
  return {
    text: readCssColor('--apple-muted', '#a8b0bf'),
    grid: readCssColor('--apple-border-soft', 'rgba(255,255,255,0.08)'),
    actual: readCssColor('--apple-blue', '#d7ad5f'),
    standard: readCssColor('--apple-muted-2', '#727d91'),
    tokens: '#44d7a8',
  }
})

const trendChartData = computed(() => {
  if (!props.trend?.length) return null

  return {
    labels: props.trend.map((d) => d.date),
    datasets: [
      {
        label: t('dashboard.actual'),
        data: props.trend.map((d) => d.actual_cost),
        borderColor: chartPalette.value.actual,
        backgroundColor: `${chartPalette.value.actual}1f`,
        fill: true,
        tension: 0.35,
        pointRadius: 2,
        yAxisID: 'ySpend',
      },
      {
        label: t('dashboard.standard'),
        data: props.trend.map((d) => d.cost),
        borderColor: chartPalette.value.standard,
        backgroundColor: 'transparent',
        borderDash: [4, 4],
        fill: false,
        tension: 0.35,
        pointRadius: 2,
        yAxisID: 'ySpend',
      },
      {
        label: t('dashboard.tokens'),
        data: props.trend.map((d) => d.total_tokens),
        borderColor: chartPalette.value.tokens,
        backgroundColor: `${chartPalette.value.tokens}14`,
        fill: false,
        tension: 0.35,
        pointRadius: 2,
        yAxisID: 'yTokens',
      },
    ],
  }
})

const trendChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const,
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartPalette.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        boxWidth: 8,
        padding: 14,
        font: {
          size: 11,
        },
      },
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const value = Number(context.raw ?? 0)
          if (context.dataset.yAxisID === 'yTokens') {
            return `${context.dataset.label}: ${formatTokens(value)}`
          }
          return `${context.dataset.label}: ${formatSettlementAmount(value, 4)}`
        },
      },
    },
  },
  scales: {
    x: {
      grid: {
        color: chartPalette.value.grid,
      },
      ticks: {
        color: chartPalette.value.text,
        font: {
          size: 10,
        },
      },
    },
    ySpend: {
      position: 'left' as const,
      grid: {
        color: chartPalette.value.grid,
      },
      ticks: {
        color: chartPalette.value.text,
        font: {
          size: 10,
        },
        callback: (value: string | number) => formatSettlementAmount(Number(value), 4),
      },
    },
    yTokens: {
      position: 'right' as const,
      grid: {
        drawOnChartArea: false,
      },
      ticks: {
        color: chartPalette.value.tokens,
        font: {
          size: 10,
        },
        callback: (value: string | number) => formatTokens(Number(value)),
      },
    },
  },
}))

function readCssColor(name: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback
  const host = document.querySelector('.gateway-console') || document.documentElement
  return getComputedStyle(host).getPropertyValue(name).trim() || fallback
}
</script>
