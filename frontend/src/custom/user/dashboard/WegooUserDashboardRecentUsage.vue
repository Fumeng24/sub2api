<template>
  <section data-testid="dashboard-recent-usage" class="card">
    <div class="card-header flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-[var(--apple-text)]">{{ t('dashboard.recentUsage') }}</h2>
      </div>
      <span class="badge badge-gray w-fit">{{ t('dashboard.last7Days') }}</span>
    </div>
    <div class="card-body p-4 sm:p-5">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
      <div v-else-if="data.length === 0" class="py-8">
        <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
      </div>
      <div v-else class="divide-y divide-[color:var(--apple-border-soft)]">
        <div v-for="log in data" :key="log.id" class="flex flex-col gap-3 py-3 first:pt-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--apple-radius)] bg-[var(--apple-surface-elevated)] text-[var(--apple-blue)] ring-1 ring-[color:var(--apple-border)]">
              <Icon name="sparkles" size="md" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <div class="min-w-0 text-sm">
                <p class="truncate font-medium text-[var(--apple-text)]">{{ log.model }}</p>
                <p
                  v-if="log.upstream_model && log.upstream_model !== log.model"
                  class="truncate text-xs text-[var(--apple-muted)]"
                  :title="log.upstream_model"
                >
                  <span class="mr-0.5">↳</span>{{ log.upstream_model }}
                </p>
              </div>
              <p class="text-xs text-[var(--apple-muted)]">{{ formatDateTime(log.created_at) }}</p>
            </div>
          </div>
          <div class="shrink-0 text-left sm:text-right">
            <p class="text-sm font-semibold tabular-nums text-[var(--apple-text)]">
              <span class="text-[var(--apple-success)]" :title="t('dashboard.actual')">{{ formatSettlementAmount(log.actual_cost, 4) }}</span>
              <span class="font-normal text-[var(--apple-muted-2)]" :title="t('dashboard.standard')"> / {{ formatSettlementAmount(log.total_cost, 4) }}</span>
            </p>
            <p class="text-xs text-[var(--apple-muted)]">
              {{ (log.input_tokens + log.output_tokens).toLocaleString() }} {{ t('dashboard.tokens') }}
            </p>
          </div>
        </div>

        <router-link to="/usage" class="mt-2 flex items-center justify-center gap-2 rounded-[var(--apple-radius)] bg-[var(--apple-surface-elevated)] py-3 text-sm font-medium text-[var(--apple-blue)] transition-colors hover:bg-[color-mix(in_srgb,var(--apple-blue) 5%,var(--apple-surface-elevated))] hover:text-[var(--apple-blue-hover)]">
          {{ t('dashboard.viewAllUsage') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/custom/common/WegooEmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineOptions({ name: 'UserDashboardRecentUsage' })

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const { formatSettlementAmount } = useSettlementCurrency()
</script>
