<template>
  <div class="mt-4 border-t border-[color:var(--apple-border-soft)] pt-3">
    <div
      class="mb-2 flex justify-between text-[10px] font-semibold uppercase tracking-wide text-[var(--apple-muted)]"
    >
      <span>{{ t('monitorCommon.history60pts', { n: length }) }}</span>
      <span class="tabular-nums">{{ t('monitorCommon.nextUpdateIn', { n: countdownSeconds }) }}</span>
    </div>

    <div
      v-if="maintenance"
      class="flex h-5 w-full items-center justify-center rounded border border-dashed border-[color:var(--apple-border)] text-[10px] uppercase tracking-wide text-[var(--apple-muted)]"
    >
      {{ t('monitorCommon.maintenancePaused') }}
    </div>
    <div
      v-else
      data-test="monitor-timeline"
      class="grid h-5 w-full items-end gap-px overflow-hidden"
      :style="{ gridTemplateColumns: `repeat(${displayBars.length}, minmax(0, 1fr))` }"
    >
      <div
        v-for="(bar, idx) in displayBars"
        :key="idx"
        data-test="monitor-timeline-bar"
        class="min-w-0 rounded-sm"
        :class="bar.colorClass"
        :style="{ height: bar.heightPct + '%' }"
        :title="bar.title"
      ></div>
    </div>

    <div
      class="mt-1 flex justify-between text-[9px] uppercase tracking-wide text-[var(--apple-muted)]"
    >
      <span>{{ t('monitorCommon.past') }}</span>
      <span>{{ t('monitorCommon.now') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import { useChannelMonitorFormat } from '@/custom/composables/useChannelMonitorFormat'

defineOptions({ name: 'MonitorTimeline' })

const props = withDefaults(defineProps<{
  buckets?: MonitorTimelinePoint[]
  countdownSeconds: number
  length?: number
  maintenance?: boolean
}>(), {
  buckets: () => [],
  length: 60,
  maintenance: false,
})

const { t } = useI18n()
const { statusLabel, formatLatency, formatRelativeTime } = useChannelMonitorFormat()

interface Bar {
  colorClass: string
  heightPct: number
  title: string
}

const STATUS_HEIGHT: Record<string, number> = {
  operational: 100,
  degraded: 65,
  failed: 35,
  error: 35,
  empty: 15,
}

const STATUS_COLOR: Record<string, string> = {
  operational: 'monitor-timeline-bar--operational',
  degraded: 'monitor-timeline-bar--degraded',
  failed: 'monitor-timeline-bar--failed',
  error: 'monitor-timeline-bar--failed',
  empty: 'monitor-timeline-bar--empty',
}

const displayBars = computed<Bar[]>(() => {
  const real = [...(props.buckets ?? [])]
    .slice(0, props.length)
    .reverse()

  const padCount = Math.max(0, props.length - real.length)
  const bars: Bar[] = []

  for (let i = 0; i < padCount; i += 1) {
    bars.push({
      colorClass: STATUS_COLOR.empty,
      heightPct: STATUS_HEIGHT.empty,
      title: '',
    })
  }

  for (const point of real) {
    const status = point.status as keyof typeof STATUS_HEIGHT
    const colorClass = STATUS_COLOR[status] ?? STATUS_COLOR.empty
    const heightPct = STATUS_HEIGHT[status] ?? STATUS_HEIGHT.empty
    const latency = formatLatency(point.latency_ms)
    const relative = formatRelativeTime(point.checked_at)
    const label = statusLabel(point.status)
    bars.push({
      colorClass,
      heightPct,
      title: `${relative} · ${label} · ${latency}ms`,
    })
  }

  return bars
})
</script>

<style scoped>
.monitor-timeline-bar--operational {
  background: #16a34a;
}

.monitor-timeline-bar--degraded {
  background: #f59e0b;
}

.monitor-timeline-bar--failed {
  background: #dc2626;
}

.monitor-timeline-bar--empty {
  background: color-mix(in srgb, var(--apple-muted) 28%, transparent);
}

:global(.dark) .monitor-timeline-bar--operational {
  background: #4ade80;
}

:global(.dark) .monitor-timeline-bar--degraded {
  background: #fbbf24;
}

:global(.dark) .monitor-timeline-bar--failed {
  background: #f87171;
}
</style>
