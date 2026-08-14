<template>
  <button
    data-test="user-monitor-card"
    :data-monitor-id="item.id"
    type="button"
    class="card card-hover group flex min-h-[260px] min-w-0 w-full flex-col p-5 text-left"
    @click="emit('click')"
  >
    <div class="flex items-start gap-3">
      <span
        class="grid h-9 w-9 flex-shrink-0 place-items-center rounded-lg ring-1 ring-black/5 dark:ring-white/10"
        :class="['bg-[var(--apple-surface-elevated)]', providerTintClass]"
      >
        <ProviderIcon :provider="item.provider" :size="20" />
      </span>
      <div class="min-w-0 flex-1">
        <div class="break-words text-base font-semibold leading-5 text-[var(--apple-text)]">
          {{ item.name }}
        </div>
        <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1.5">
          <span
            class="monitor-provider-badge inline-flex flex-shrink-0 items-center rounded-md px-1.5 py-0.5 text-[10px] font-medium"
            :class="providerBadgeClass(item.provider)"
          >
            {{ providerLabel(item.provider) }}
          </span>
          <span class="min-w-0 break-all font-mono text-xs text-[var(--apple-muted)]">
            {{ item.primary_model }}
          </span>
          <span
            v-if="item.group_name"
            class="inline-flex max-w-full flex-shrink-0 items-center rounded-md border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--apple-muted)]"
          >
            <span class="truncate">{{ item.group_name }}</span>
          </span>
        </div>
      </div>
      <span
        class="monitor-status-badge flex-shrink-0 rounded-md px-2.5 py-1 text-xs font-semibold"
        :class="statusBadgeClass(item.primary_status)"
      >
        {{ statusLabel(item.primary_status) }}
      </span>
    </div>

    <div class="mt-5 grid grid-cols-2 overflow-hidden rounded-lg border border-[color:var(--apple-border-soft)]">
      <div class="min-w-0 border-r border-[color:var(--apple-border-soft)] px-3 py-2.5">
        <div class="flex items-center gap-1.5 text-[10px] font-semibold text-[var(--apple-muted)]">
          <Icon name="bolt" size="xs" />
          <span class="truncate">{{ t('monitorCommon.dialogLatency') }}</span>
        </div>
        <div class="mt-1 font-mono text-base font-semibold tabular-nums text-[var(--apple-text)]">
          {{ formatLatency(item.primary_latency_ms) }}<span class="ml-0.5 text-xs font-normal text-[var(--apple-muted)]">ms</span>
        </div>
      </div>
      <div class="min-w-0 px-3 py-2.5">
        <div class="flex items-center gap-1.5 text-[10px] font-semibold text-[var(--apple-muted)]">
          <Icon name="globe" size="xs" />
          <span class="truncate">{{ t('monitorCommon.endpointPing') }}</span>
        </div>
        <div class="mt-1 font-mono text-base font-semibold tabular-nums text-[var(--apple-text)]">
          {{ formatLatency(item.primary_ping_latency_ms) }}<span class="ml-0.5 text-xs font-normal text-[var(--apple-muted)]">ms</span>
        </div>
      </div>
    </div>

    <div class="mt-4 border-t border-[color:var(--apple-border-soft)]"></div>

    <MonitorAvailabilityRow
      :window-label="availabilityLabel"
      :value="availabilityValue"
      :samples-label="extraModelsCountLabel"
    />

    <MonitorTimeline
      :buckets="item.timeline"
      :countdown-seconds="countdownSeconds"
    />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView } from '@/api/channelMonitor'
import type { MonitorStatus } from '@/custom/api/admin/channelMonitor'
import { useChannelMonitorFormat } from '@/custom/composables/useChannelMonitorFormat'
import {
  PROVIDER_OPENAI,
  PROVIDER_ANTHROPIC,
  PROVIDER_GEMINI,
  PROVIDER_GROK,
  STATUS_OPERATIONAL,
  STATUS_DEGRADED,
  STATUS_FAILED,
  STATUS_ERROR,
} from '@/constants/channelMonitor'
import Icon from '@/components/icons/Icon.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import MonitorAvailabilityRow from './WegooMonitorAvailabilityRow.vue'
import MonitorTimeline from './WegooMonitorTimeline.vue'

defineOptions({ name: 'MonitorCard' })

const PROVIDER_TINT: Record<string, string> = {
  [PROVIDER_OPENAI]: 'monitor-provider-tint--openai',
  [PROVIDER_ANTHROPIC]: 'monitor-provider-tint--anthropic',
  [PROVIDER_GEMINI]: 'monitor-provider-tint--gemini',
  [PROVIDER_GROK]: 'monitor-provider-tint--grok',
}

const props = defineProps<{
  item: UserMonitorView
  window: '7d' | '15d' | '30d'
  availabilityValue: number | null
  countdownSeconds: number
}>()

const emit = defineEmits<{
  (e: 'click'): void
}>()

const { t } = useI18n()
const {
  statusLabel,
  providerLabel,
  formatLatency,
} = useChannelMonitorFormat()

const providerTintClass = computed(() =>
  PROVIDER_TINT[props.item.provider] ?? 'text-[var(--apple-muted)]'
)

function statusBadgeClass(status: MonitorStatus | '') {
  switch (status) {
    case STATUS_OPERATIONAL:
      return 'monitor-status-badge--operational'
    case STATUS_DEGRADED:
      return 'monitor-status-badge--degraded'
    case STATUS_FAILED:
    case STATUS_ERROR:
      return 'monitor-status-badge--failed'
    default:
      return 'monitor-status-badge--neutral'
  }
}

function providerBadgeClass(provider: string) {
  switch (provider) {
    case PROVIDER_OPENAI:
      return 'monitor-provider-badge--openai'
    case PROVIDER_ANTHROPIC:
      return 'monitor-provider-badge--anthropic'
    case PROVIDER_GEMINI:
      return 'monitor-provider-badge--gemini'
    default:
      return 'monitor-provider-badge--neutral'
  }
}

const availabilityLabel = computed(() => {
  const win = t(`channelStatus.windowTab.${props.window}`)
  return `${t('monitorCommon.availabilityPrefix')} · ${win}`
})

const extraModelsCountLabel = computed(() => {
  const count = props.item.extra_models?.length ?? 0
  if (count === 0) return undefined
  return t('monitorCommon.extraModelsCount', { n: count })
})
</script>

<style scoped>
.monitor-provider-badge,
.monitor-status-badge {
  border: 1px solid transparent;
}

.monitor-provider-badge--openai {
  background: color-mix(in srgb, var(--apple-success) 11%, transparent);
  border-color: color-mix(in srgb, var(--apple-success) 20%, transparent);
  color: var(--apple-success);
}

.monitor-status-badge--operational {
  background: color-mix(in srgb, #16a34a 12%, var(--apple-surface));
  border-color: color-mix(in srgb, #16a34a 38%, var(--apple-border));
  color: #15803d;
}

.monitor-provider-badge--anthropic {
  background: color-mix(in srgb, var(--apple-warning) 18%, transparent);
  border-color: color-mix(in srgb, var(--apple-warning) 36%, transparent);
  color: var(--apple-warning);
}

.monitor-status-badge--degraded {
  background: #fffbeb;
  border-color: #f59e0b;
  color: #92400e;
}

.monitor-provider-badge--gemini {
  background: color-mix(in srgb, var(--apple-blue) 11%, transparent);
  border-color: color-mix(in srgb, var(--apple-blue) 20%, transparent);
  color: var(--apple-blue);
}

.monitor-status-badge--failed {
  background: #fef2f2;
  border-color: #dc2626;
  color: #b91c1c;
}

.monitor-provider-badge--neutral,
.monitor-status-badge--neutral {
  background: var(--apple-surface-elevated);
  border-color: var(--apple-border-soft);
  color: var(--apple-muted);
}

.monitor-provider-tint--openai {
  color: var(--apple-success);
}

.monitor-provider-tint--anthropic {
  color: var(--apple-warning);
}

.monitor-provider-tint--gemini {
  color: var(--apple-blue);
}

.monitor-provider-tint--grok {
  color: var(--apple-text);
}

:global(.dark) .monitor-status-badge--operational {
  background: rgba(22, 163, 74, 0.18);
  border-color: rgba(74, 222, 128, 0.5);
  color: #4ade80;
}

:global(.dark) .monitor-status-badge--degraded {
  background: rgba(245, 158, 11, 0.2);
  border-color: rgba(251, 191, 36, 0.65);
  color: #fbbf24;
}

:global(.dark) .monitor-status-badge--failed {
  background: rgba(220, 38, 38, 0.22);
  border-color: rgba(248, 113, 113, 0.68);
  color: #f87171;
}
</style>
