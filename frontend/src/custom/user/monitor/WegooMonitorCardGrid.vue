<template>
  <div>
    <div
      v-if="loading && items.length === 0"
      class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      <div
        v-for="i in 6"
        :key="i"
        class="card min-h-[260px] animate-pulse p-5"
      >
        <div class="flex items-start gap-3">
          <div class="skeleton h-9 w-9 rounded-lg"></div>
          <div class="flex-1 space-y-2">
            <div class="skeleton h-4 w-2/3"></div>
            <div class="skeleton h-3 w-1/2"></div>
          </div>
          <div class="skeleton h-6 w-16 rounded-full"></div>
        </div>
        <div class="mt-5 grid grid-cols-2 gap-2">
          <div class="skeleton h-14 rounded-lg"></div>
          <div class="skeleton h-14 rounded-lg"></div>
        </div>
        <div class="skeleton mt-6 h-5 w-full"></div>
      </div>
    </div>

    <EmptyState
      v-else-if="items.length === 0"
      :title="emptyCopy.title"
      :description="emptyCopy.description"
    />

    <div
      v-else
      class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      <MonitorCard
        v-for="item in items"
        :key="item.id"
        :item="item"
        :window="window"
        :availability-value="resolveAvailability(item)"
        :countdown-seconds="countdownSeconds"
        @click="emit('cardClick', item)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView, UserMonitorDetail } from '@/api/channelMonitor'
import EmptyState from '@/custom/common/WegooEmptyState.vue'
import MonitorCard from './WegooMonitorCard.vue'

const props = defineProps<{
  items: UserMonitorView[]
  window: '7d' | '15d' | '30d'
  countdownSeconds: number
  loading: boolean
  detailCache: Record<number, UserMonitorDetail>
}>()

const emit = defineEmits<{
  (e: 'cardClick', item: UserMonitorView): void
}>()

const { locale } = useI18n()

const emptyCopy = computed(() => locale.value.startsWith('zh')
  ? {
    title: '状态正在建立',
    description: '有可展示的状态记录后，会在这里呈现可用率、延迟和近期状态。'
  }
  : {
    title: 'Status is being prepared',
    description: 'Availability, latency, and recent checks will appear here once status updates are available.'
  }
)

function resolveAvailability(item: UserMonitorView): number | null {
  if (props.window === '7d') {
    return item.availability_7d ?? null
  }
  const detail = props.detailCache[item.id]
  if (!detail) return null
  const primary = detail.models.find(m => m.model === item.primary_model)
  if (!primary) return null
  return props.window === '15d' ? primary.availability_15d ?? null : primary.availability_30d ?? null
}
</script>
