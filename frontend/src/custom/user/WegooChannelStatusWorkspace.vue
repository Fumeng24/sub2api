<template>
  <div class="mx-auto max-w-7xl space-y-5">
    <UserPageHero
      :kicker="t('channelStatus.gateway.kicker')"
      :title="t('channelStatus.hero.title')"
      :description="t('channelStatus.gateway.description')"
    >
      <template #aside>
        <MonitorHero
          :overall-status="overallStatus"
          :window="window"
          :loading="loading"
          :auto-refresh="autoRefresh"
          @update:window="emit('update:window', $event)"
          @refresh="emit('refresh')"
        />
      </template>
    </UserPageHero>

    <MonitorCardGrid
      :items="items"
      :window="window"
      :countdown-seconds="countdownSeconds"
      :loading="loading"
      :detail-cache="detailCache"
      @card-click="emit('cardClick', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UserMonitorDetail, UserMonitorView } from '@/api/channelMonitor'
import type { useAutoRefresh } from '@/composables/useAutoRefresh'
import MonitorCardGrid from '@/custom/user/monitor/WegooMonitorCardGrid.vue'
import MonitorHero, {
  type MonitorWindow,
  type OverallStatus,
} from '@/custom/user/monitor/WegooMonitorHero.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'

defineProps<{
  items: UserMonitorView[]
  window: MonitorWindow
  countdownSeconds: number
  loading: boolean
  detailCache: Record<number, UserMonitorDetail>
  overallStatus: OverallStatus
  autoRefresh: ReturnType<typeof useAutoRefresh>
}>()

const emit = defineEmits<{
  (event: 'update:window', value: MonitorWindow): void
  (event: 'refresh'): void
  (event: 'cardClick', item: UserMonitorView): void
}>()

const { t } = useI18n()
</script>
