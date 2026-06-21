<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <div
        v-else-if="loadError"
        class="rounded-lg border border-red-100 bg-white p-8 text-center shadow-sm dark:border-red-900/40 dark:bg-dark-800"
      >
        <div
          class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300"
        >
          <Icon name="exclamationTriangle" size="lg" :stroke-width="2" />
        </div>
        <h2 class="text-lg font-bold text-gray-900 dark:text-white">
          {{ t('dashboard.loadFailed') }}
        </h2>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ loadError }}
        </p>
        <button type="button" class="btn btn-primary mt-5" @click="refreshAll">
          <Icon name="refresh" size="sm" class="mr-2" />
          {{ t('dashboard.retryLoad') }}
        </button>
      </div>

      <template v-else-if="stats">
        <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 sm:flex-row sm:items-end sm:justify-between">
          <div class="min-w-0">
            <h1 class="text-2xl font-semibold tracking-normal text-gray-950 dark:text-white">
              {{ t('dashboard.title') }}
            </h1>
            <p class="mt-1 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t('dashboard.welcomeMessage') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" :disabled="loading || loadingCharts || loadingUsage" @click="refreshAll">
            <Icon
              name="refresh"
              size="sm"
              :class="loading || loadingCharts || loadingUsage ? 'animate-spin' : ''"
            />
            {{ t('common.refresh') }}
          </button>
        </div>
        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          :platform-quotas="platformQuotas"
          :balance-cny-per-credit="balanceCnyPerCredit"
          :balance-groups="balanceGroups"
          :user-group-rates="userGroupRates"
        />
        <UserDashboardCharts
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          v-model:granularity="granularity"
          :loading="loadingCharts"
          :trend="trendData"
          :models="modelStats"
          @dateRangeChange="loadCharts"
          @granularityChange="loadCharts"
          @refresh="refreshAll"
        />
        <div class="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1.7fr)_minmax(280px,0.8fr)]">
          <div>
            <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
          </div>
          <div>
            <UserDashboardQuickActions />
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { paymentAPI } from '@/api/payment'
import { setSettlementCnyPerCredit } from '@/composables/useSettlementCurrency'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import type { Group } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const appStore = useAppStore()
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const loadError = ref('')
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const balanceCnyPerCredit = ref(6.8)
const balanceGroups = ref<Group[]>([])
const userGroupRates = ref<Record<number, number>>({})

const formatLD = (d: Date) => d.toISOString().split('T')[0]
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatLD(new Date()))
const granularity = ref('day')

const loadStats = async () => {
  loading.value = true
  loadError.value = ''
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
    stats.value = null
    loadError.value = extractApiErrorMessage(error, t('dashboard.loadFailedDesc'))
  } finally {
    loading.value = false
  }
}

const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const [trend, models] = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value as any,
      }),
      usageAPI.getDashboardModels({
        start_date: startDate.value,
        end_date: endDate.value,
      }),
    ])
    trendData.value = trend.trend || []
    modelStats.value = models.models || []
  } catch (error) {
    console.error('Failed to load charts:', error)
  } finally {
    loadingCharts.value = false
  }
}

const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const res = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = res.items.slice(0, 5)
  } catch (error) {
    console.error('Failed to load recent usage:', error)
  } finally {
    loadingUsage.value = false
  }
}

const loadPlatformQuotas = async () => {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (error) {
    console.warn('Failed to load platform quotas:', error)
    platformQuotas.value = []
  }
}

const loadPaymentConfig = async () => {
  try {
    const res = await paymentAPI.getConfig()
    const value = res.data.balance_recharge_multiplier
    balanceCnyPerCredit.value = Number.isFinite(value) && value > 0 ? value : 6.8
    setSettlementCnyPerCredit(balanceCnyPerCredit.value)
  } catch (error) {
    console.warn('Failed to load payment config:', error)
    balanceCnyPerCredit.value = 6.8
    setSettlementCnyPerCredit(balanceCnyPerCredit.value)
  }
}

const loadBalanceGroups = async () => {
  try {
    const [groups, rates] = await Promise.all([
      userGroupsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((error) => {
        console.warn('Failed to load user group rates:', error)
        return {} as Record<number, number>
      }),
      appStore.fetchPublicSettings(),
    ])
    balanceGroups.value = groups
    userGroupRates.value = rates
  } catch (error) {
    console.warn('Failed to load balance groups:', error)
    balanceGroups.value = []
    userGroupRates.value = {}
  }
}

const refreshAll = () => {
  loadStats()
  loadCharts()
  loadRecent()
  loadPlatformQuotas()
  loadPaymentConfig()
  loadBalanceGroups()
}

onMounted(() => { refreshAll() })
</script>
