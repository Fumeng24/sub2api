<template>
  <section class="py-0">
    <div class="flex w-full flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
      <div
        role="tablist"
        class="inline-flex w-full rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-0.5 text-xs sm:w-auto"
      >
        <button
          v-for="opt in windowOptions"
          :key="opt.value"
          type="button"
          role="tab"
          :aria-selected="window === opt.value"
          class="flex-1 rounded-md px-3 py-1 font-medium transition-colors sm:flex-none"
          :class="window === opt.value
            ? 'bg-[var(--apple-surface)] text-[var(--apple-blue)] shadow-sm'
            : 'text-[var(--apple-muted)] hover:text-[var(--apple-text)]'"
          @click="emit('update:window', opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>

      <div class="flex items-center justify-end gap-2">
        <span
          class="monitor-status-chip inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold"
          :class="overallChipClass"
        >
          <span
            class="monitor-status-dot mr-1.5 h-1.5 w-1.5 rounded-full"
            :class="overallDotClass"
          ></span>
          {{ overallLabel }}
        </span>

        <button
          type="button"
          class="btn btn-secondary btn-icon h-8 w-8"
          :disabled="loading"
          :title="t('common.refresh')"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>

        <AutoRefreshButton
          v-if="autoRefresh"
          :enabled="autoRefresh.enabled.value"
          :interval-seconds="autoRefresh.intervalSeconds.value"
          :countdown="autoRefresh.countdown.value"
          :intervals="autoRefresh.intervals"
          @update:enabled="autoRefresh.setEnabled"
          @update:interval="autoRefresh.setInterval"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
export type MonitorWindow = '7d' | '15d' | '30d'
export type OverallStatus = 'operational' | 'degraded'

const props = defineProps<{
  overallStatus: OverallStatus
  window: MonitorWindow
  loading: boolean
  autoRefresh?: {
    enabled: { value: boolean }
    intervalSeconds: { value: number }
    countdown: { value: number }
    intervals: readonly number[]
    setEnabled: (v: boolean) => void
    setInterval: (v: number) => void
  }
}>()

const emit = defineEmits<{
  (e: 'update:window', value: MonitorWindow): void
  (e: 'refresh'): void
}>()

const { t, locale } = useI18n()

const windowOptions = computed<{ value: MonitorWindow; label: string }[]>(() => [
  { value: '7d', label: t('channelStatus.windowTab.7d') },
  { value: '15d', label: t('channelStatus.windowTab.15d') },
  { value: '30d', label: t('channelStatus.windowTab.30d') },
])

const overallLabel = computed(() => {
  const zh = locale.value.startsWith('zh')
  if (props.overallStatus === 'operational') {
    return zh ? '服务可用' : 'Operational'
  }
  return zh ? '部分波动' : 'Degraded'
})

const overallChipClass = computed(() => {
  switch (props.overallStatus) {
    case 'operational':
      return 'monitor-status-chip--operational'
    case 'degraded':
    default:
      return 'monitor-status-chip--degraded'
  }
})

const overallDotClass = computed(() => {
  switch (props.overallStatus) {
    case 'operational':
      return 'monitor-status-dot--operational animate-pulse'
    case 'degraded':
    default:
      return 'monitor-status-dot--degraded animate-pulse'
  }
})

</script>

<style scoped>
.monitor-status-chip {
  border: 1px solid transparent;
}

.monitor-status-chip--operational {
  background: color-mix(in srgb, #16a34a 12%, var(--apple-surface));
  border-color: color-mix(in srgb, #16a34a 38%, var(--apple-border));
  color: #15803d;
}

.monitor-status-chip--degraded {
  background: #fef2f2;
  border-color: #dc2626;
  color: #b91c1c;
}

.monitor-status-dot--operational {
  background: #16a34a;
}

.monitor-status-dot--degraded {
  background: #dc2626;
}

:global(.dark) .monitor-status-chip--operational {
  background: rgba(22, 163, 74, 0.18);
  border-color: rgba(74, 222, 128, 0.5);
  color: #4ade80;
}

:global(.dark) .monitor-status-chip--degraded {
  background: rgba(220, 38, 38, 0.22);
  border-color: rgba(248, 113, 113, 0.68);
  color: #f87171;
}

:global(.dark) .monitor-status-dot--operational {
  background: #4ade80;
}

:global(.dark) .monitor-status-dot--degraded {
  background: #f87171;
}
</style>
