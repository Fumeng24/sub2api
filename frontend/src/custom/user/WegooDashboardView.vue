<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <div
        v-else-if="loadError"
        class="card empty-state border-[color:var(--apple-border)]"
      >
        <div
          class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-[var(--apple-radius)] bg-[var(--apple-surface-elevated)] text-[var(--apple-danger)] ring-1 ring-[color:var(--apple-border)]"
        >
          <Icon name="exclamationTriangle" size="lg" :stroke-width="2" />
        </div>
        <h2 class="empty-state-title">
          {{ t('dashboard.loadFailed') }}
        </h2>
        <p class="empty-state-description">
          {{ loadError }}
        </p>
        <button type="button" class="btn btn-primary mt-5" @click="refreshAll">
          <Icon name="refresh" size="sm" class="mr-2" />
          {{ t('dashboard.retryLoad') }}
        </button>
      </div>

      <template v-else-if="stats">
        <UserPageHero
          :kicker="t('dashboard.gateway.kicker')"
          :title="t('dashboard.gateway.title')"
        >
          <template #actions>
          <button type="button" class="btn btn-secondary w-full shrink-0 sm:w-auto" :disabled="isRefreshing" @click="refreshAll">
            <Icon
              name="refresh"
              size="sm"
              :class="isRefreshing ? 'animate-spin' : ''"
            />
            {{ t('common.refresh') }}
          </button>
          </template>
        </UserPageHero>

        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          mode="balance"
          :platform-quotas="platformQuotas"
          :balance-cny-per-credit="balanceCnyPerCredit"
          :balance-groups="balanceGroups"
          :user-group-rates="userGroupRates"
        />

        <section
          v-if="dashboardNextActions.length"
          class="grid gap-3 md:grid-cols-3"
          :aria-label="t('dashboard.nextSteps.label')"
        >
          <article
            v-for="item in dashboardNextActions"
            :key="item.key"
            :class="[
              'group flex min-w-0 flex-col justify-between rounded-[var(--apple-radius)] border p-4 shadow-sm',
              item.panelClass
            ]"
          >
            <div class="min-w-0">
              <div class="flex items-start justify-between gap-3">
                <span :class="['rounded-lg p-2 ring-1', item.iconClass]">
                  <Icon :name="item.icon" size="sm" :stroke-width="2" />
                </span>
                <span class="rounded-full bg-[rgb(255_255_255_/_0.055)] px-2 py-1 text-[10px] font-semibold uppercase text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]">
                  {{ item.badge }}
                </span>
              </div>
              <h2 class="mt-3 text-base font-semibold text-[var(--apple-text)]">{{ item.title }}</h2>
              <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">{{ item.description }}</p>
            </div>
            <router-link
              :to="item.to"
              class="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-[var(--apple-blue)] transition-colors hover:text-[var(--apple-blue-hover)]"
            >
              {{ item.action }}
              <Icon name="arrowRight" size="xs" />
            </router-link>
          </article>
        </section>

        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          mode="usage"
          :platform-quotas="platformQuotas"
          :balance-cny-per-credit="balanceCnyPerCredit"
          :balance-groups="balanceGroups"
          :user-group-rates="userGroupRates"
        />
        <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
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
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/custom/api/usage'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/custom/user/dashboard/WegooUserDashboardStats.vue'
import UserDashboardCharts from '@/custom/user/dashboard/WegooUserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/custom/user/dashboard/WegooUserDashboardRecentUsage.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem, UserErrorRequest } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { paymentAPI } from '@/custom/api/payment'
import { setSettlementCnyPerCredit } from '@/custom/composables/useSettlementCurrency'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import type { Group } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { formatDateLocalInput } from '@/utils/format'

type IconName = InstanceType<typeof Icon>['$props']['name']

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const appStore = useAppStore()
const { formatSettlementAmount } = useSettlementCurrency()
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const loadError = ref('')
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const recentErrorTotal = ref(0)
const latestError = ref<UserErrorRequest | null>(null)
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const balanceCnyPerCredit = ref(6.8)
const balanceGroups = ref<Group[]>([])
const userGroupRates = ref<Record<number, number>>({})
const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref('day')
const isRefreshing = computed(() => loading.value || loadingCharts.value || loadingUsage.value)
const currentBalance = computed(() => Number(user.value?.balance ?? NaN))
const lowBalanceThreshold = computed(() => {
  const parsed = Number(appStore.cachedPublicSettings?.balance_low_notify_threshold)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 1
})
const dashboardErrorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests === true)

const dashboardNextActions = computed<Array<{
  key: string
  icon: IconName
  badge: string
  title: string
  description: string
  action: string
  to: string | { path: string; query?: Record<string, string> }
  panelClass: string
  iconClass: string
}>>(() => {
  const items: Array<{
    key: string
    icon: IconName
    badge: string
    title: string
    description: string
    action: string
    to: string | { path: string; query?: Record<string, string> }
    panelClass: string
    iconClass: string
  }> = []

  if ((stats.value?.active_api_keys || 0) <= 0) {
    items.push({
      key: 'no-key',
      icon: 'key',
      badge: t('dashboard.nextSteps.requiredBadge'),
      title: t('dashboard.nextSteps.noKey.title'),
      description: t('dashboard.nextSteps.noKey.description'),
      action: t('dashboard.nextSteps.noKey.action'),
      to: '/keys',
      panelClass: 'border-[color:color-mix(in_srgb,var(--apple-blue)_24%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-blue)_7%,var(--apple-surface))]',
      iconClass: 'bg-[color-mix(in_srgb,var(--apple-blue)_12%,transparent)] text-[var(--apple-blue)] ring-[color:color-mix(in_srgb,var(--apple-blue)_24%,var(--apple-border))]',
    })
  }

  if (Number.isFinite(currentBalance.value) && currentBalance.value <= lowBalanceThreshold.value) {
    items.push({
      key: 'low-balance',
      icon: 'creditCard',
      badge: t('dashboard.nextSteps.balanceBadge'),
      title: t('dashboard.nextSteps.lowBalance.title'),
      description: t('dashboard.nextSteps.lowBalance.description', {
        threshold: formatSettlementAmount(lowBalanceThreshold.value, 2),
      }),
      action: t('dashboard.nextSteps.lowBalance.action'),
      to: '/purchase',
      panelClass: 'border-[color:color-mix(in_srgb,var(--apple-warning)_26%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_7%,var(--apple-surface))]',
      iconClass: 'bg-[color-mix(in_srgb,var(--apple-warning)_12%,transparent)] text-[var(--apple-warning)] ring-[color:color-mix(in_srgb,var(--apple-warning)_26%,var(--apple-border))]',
    })
  }

  if (dashboardErrorViewEnabled.value && recentErrorTotal.value > 0) {
    items.push({
      key: 'recent-error',
      icon: 'exclamationTriangle',
      badge: t('dashboard.nextSteps.errorBadge'),
      title: t('dashboard.nextSteps.recentError.title'),
      description: t('dashboard.nextSteps.recentError.description', {
        count: recentErrorTotal.value,
        model: latestError.value?.model || t('dashboard.nextSteps.recentError.unknownModel'),
        status: latestError.value?.status_code || '-',
      }),
      action: t('dashboard.nextSteps.recentError.action'),
      to: { path: '/usage', query: { tab: 'errors' } },
      panelClass: 'border-[color:color-mix(in_srgb,var(--apple-danger)_26%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_7%,var(--apple-surface))]',
      iconClass: 'bg-[color-mix(in_srgb,var(--apple-danger)_12%,transparent)] text-[var(--apple-danger)] ring-[color:color-mix(in_srgb,var(--apple-danger)_26%,var(--apple-border))]',
    })
  }

  return items
})

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

const loadDashboardErrors = async () => {
  try {
    await appStore.fetchPublicSettings()
    if (!dashboardErrorViewEnabled.value) {
      recentErrorTotal.value = 0
      latestError.value = null
      return
    }

    const resp = await usageAPI.listMyErrorRequests({
      page: 1,
      page_size: 1,
      start_date: startDate.value,
      end_date: endDate.value,
    })
    recentErrorTotal.value = resp.total || 0
    latestError.value = resp.items?.[0] || null
  } catch (error) {
    console.warn('Failed to load dashboard error summary:', error)
    recentErrorTotal.value = 0
    latestError.value = null
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
  loadDashboardErrors()
  loadPlatformQuotas()
  loadPaymentConfig()
  loadBalanceGroups()
}

onMounted(() => { refreshAll() })

</script>
