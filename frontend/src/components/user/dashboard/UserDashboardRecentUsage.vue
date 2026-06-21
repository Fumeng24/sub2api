<template>
  <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.recentUsage') }}</h2>
      <span class="badge badge-gray w-fit">{{ t('dashboard.last7Days') }}</span>
    </div>
    <div class="p-4 sm:p-5">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
      <div v-else-if="data.length === 0" class="py-8">
        <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
      </div>
      <div v-else class="space-y-3">
        <div v-for="log in data" :key="log.id" class="flex flex-col gap-3 rounded-lg border border-gray-100 bg-gray-50 p-4 transition-colors hover:bg-gray-100 dark:border-dark-700 dark:bg-dark-900/45 dark:hover:bg-dark-900 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-100 dark:bg-primary-900/30">
              <Icon name="beaker" size="md" class="text-primary-600 dark:text-primary-400" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ log.model }}</p>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(log.created_at) }}</p>
            </div>
          </div>
          <div class="shrink-0 text-left sm:text-right">
            <p class="text-sm font-semibold">
              <span class="text-green-600 dark:text-green-400" :title="t('dashboard.actual')">{{ formatSettlementAmount(log.actual_cost, 4) }}</span>
              <span class="font-normal text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / {{ formatSettlementAmount(log.total_cost, 4) }}</span>
            </p>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ (log.input_tokens + log.output_tokens).toLocaleString() }} tokens</p>
          </div>
        </div>

        <router-link to="/usage" class="flex items-center justify-center gap-2 py-3 text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300">
          {{ t('dashboard.viewAllUsage') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { useSettlementCurrency } from '@/composables/useSettlementCurrency'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const { formatSettlementAmount } = useSettlementCurrency()
</script>
