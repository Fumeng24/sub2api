<template>
  <div v-if="usage" class="text-sm">
    <div class="flex items-center gap-1.5">
      <span class="text-gray-500 dark:text-gray-400">{{ t('platformUsage.today') }}:</span>
      <span class="font-medium text-gray-900 dark:text-white">{{ formatSettlementAmount(usage.today_actual_cost, 4) }}</span>
    </div>
    <div class="mt-0.5 flex items-center gap-1.5">
      <span class="text-gray-500 dark:text-gray-400">{{ t('platformUsage.total') }}:</span>
      <span class="font-medium text-gray-900 dark:text-white">{{ formatSettlementAmount(usage.total_actual_cost, 4) }}</span>
    </div>
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-gray-500">—</span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import type { PlatformUsage } from '@/api/admin/dashboard'

defineProps<{
  usage?: PlatformUsage
}>()

const { t } = useI18n()
const { formatSettlementAmount } = useSettlementCurrency()
</script>
