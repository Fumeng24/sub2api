<template>
  <AppLayout>
    <div class="redeem-page mx-auto max-w-2xl space-y-6">
      <UserPageHero
        :title="t('redeem.title')"
        :description="t('redeem.description')"
      />

      <div class="card p-6 text-center">
        <div>
          <div
            class="mb-4 inline-flex h-14 w-14 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)]"
          >
            <Icon name="creditCard" size="xl" class="text-[var(--apple-blue)]" />
          </div>
          <p class="text-sm font-medium text-[var(--apple-muted)]">{{ t('redeem.currentBalance') }}</p>
          <p class="mt-2 text-4xl font-semibold text-[var(--apple-text)]">
            {{ formatCreditedBalance(user?.balance || 0) }}
          </p>
          <p class="mt-2 text-sm text-[var(--apple-muted)]">
            {{ t('redeem.concurrency') }}: {{ user?.concurrency || 0 }} {{ t('redeem.requests') }}
          </p>
        </div>
      </div>

      <div class="card">
        <div class="p-6">
          <form @submit.prevent="handleRedeem" class="space-y-5">
            <div>
              <label for="code" class="input-label">
                {{ t('redeem.redeemCodeLabel') }}
              </label>
              <div class="relative mt-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="gift" size="md" class="text-[var(--apple-muted)]" />
                </div>
                <input
                  id="code"
                  v-model="redeemCode"
                  type="text"
                  required
                  :placeholder="t('redeem.redeemCodePlaceholder')"
                  :disabled="submitting"
                  class="input py-3 pl-12 text-lg"
                />
              </div>
              <p class="input-hint">
                {{ t('redeem.redeemCodeHint') }}
              </p>
            </div>

            <button
              type="submit"
              :disabled="!redeemCode || submitting"
              class="btn btn-primary w-full py-3"
            >
              <Icon
                :name="submitting ? 'refresh' : 'checkCircle'"
                size="md"
                :class="submitting ? 'animate-spin' : ''"
              />
              {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
            </button>
          </form>
        </div>
      </div>

      <transition name="fade">
        <div
          v-if="redeemResult"
          class="card border-[color:color-mix(in_srgb,var(--apple-success)_28%,var(--apple-border))]"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-[color:color-mix(in_srgb,var(--apple-success)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_10%,var(--apple-surface))]"
              >
                <Icon name="checkCircle" size="md" class="text-[var(--apple-success)]" />
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-semibold text-[var(--apple-success)]">
                  {{ t('redeem.redeemSuccess') }}
                </h3>
                <div class="mt-2 break-words text-sm text-[var(--apple-muted)]">
                  <p>{{ redeemResult.message }}</p>
                  <div class="mt-3 space-y-1">
                    <p v-if="redeemResult.type === 'balance'" class="font-medium">
                      {{ t('redeem.added') }}: {{ formatCreditedBalance(redeemResult.value) }}
                    </p>
                    <p v-else-if="redeemResult.type === 'concurrency'" class="font-medium">
                      {{ t('redeem.added') }}: {{ redeemResult.value }}
                      {{ t('redeem.concurrentRequests') }}
                    </p>
                    <p v-else-if="redeemResult.type === 'subscription'" class="font-medium">
                      {{ t('redeem.subscriptionAssigned') }}
                      <span v-if="redeemResult.group_name"> - {{ redeemResult.group_name }}</span>
                      <span v-if="redeemResult.validity_days">
                        ({{
                          t('redeem.subscriptionDays', { days: redeemResult.validity_days })
                        }})</span
                      >
                    </p>
                    <p v-if="redeemResult.new_balance !== undefined">
                      {{ t('redeem.newBalance') }}:
                      <span class="font-semibold">{{ formatCreditedBalance(redeemResult.new_balance) }}</span>
                    </p>
                    <p v-if="redeemResult.new_concurrency !== undefined">
                      {{ t('redeem.newConcurrency') }}:
                      <span class="font-semibold"
                        >{{ redeemResult.new_concurrency }} {{ t('redeem.requests') }}</span
                      >
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <transition name="fade">
        <div
          v-if="errorMessage"
          class="card border-[color:color-mix(in_srgb,var(--apple-danger)_28%,var(--apple-border))]"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-[color:color-mix(in_srgb,var(--apple-danger)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_10%,var(--apple-surface))]"
              >
                <Icon
                  name="exclamationCircle"
                  size="md"
                  class="text-[var(--apple-danger)]"
                />
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-semibold text-[var(--apple-danger)]">
                  {{ t('redeem.redeemFailed') }}
                </h3>
                <p class="mt-2 break-words text-sm text-[var(--apple-muted)]">
                  {{ errorMessage }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <div
        class="card"
      >
        <div class="p-6">
          <div class="flex items-start gap-4">
            <div
              class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)]"
            >
              <Icon name="infoCircle" size="md" class="text-[var(--apple-blue)]" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-sm font-semibold text-[var(--apple-text)]">
                {{ t('redeem.aboutCodes') }}
              </h3>
              <div class="mt-3 grid gap-2 text-sm sm:grid-cols-2">
                <div
                  v-for="item in redeemAssuranceItems"
                  :key="item"
                  class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] px-3 py-2 text-[var(--apple-muted)]"
                >
                  {{ item }}
                </div>
                <div
                  v-if="contactInfo"
                  class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] px-3 py-2 text-[var(--apple-muted)]"
                >
                  {{ t('redeem.contactLine') }}
                  <span
                    class="ml-1.5 inline-flex max-w-full items-center rounded-md border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-2 py-0.5 text-xs font-medium text-[var(--apple-text)]"
                  >
                    {{ contactInfo }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-header">
          <h2 class="text-lg font-semibold text-[var(--apple-text)]">
            {{ t('redeem.recentActivity') }}
          </h2>
        </div>
        <div class="p-6">
          <div v-if="loadingHistory" class="flex items-center justify-center py-8">
            <Icon name="refresh" size="md" class="animate-spin text-[var(--apple-blue)]" />
          </div>

          <div v-else-if="history.length > 0" class="space-y-3">
            <div
              v-for="item in history"
              :key="item.id"
              class="flex min-w-0 flex-col gap-3 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="flex min-w-0 items-center gap-4">
                <div
                  :class="[
                    'flex h-10 w-10 items-center justify-center rounded-lg border',
                    isBalanceType(item.type)
                      ? item.value >= 0
                        ? 'border-[color:color-mix(in_srgb,var(--apple-success)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_10%,var(--apple-surface))]'
                        : 'border-[color:color-mix(in_srgb,var(--apple-danger)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_10%,var(--apple-surface))]'
                      : isSubscriptionType(item.type)
                        ? 'border-[color:color-mix(in_srgb,var(--apple-blue)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-blue)_10%,var(--apple-surface))]'
                        : item.value >= 0
                          ? 'border-[color:color-mix(in_srgb,var(--apple-blue)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-blue)_10%,var(--apple-surface))]'
                          : 'border-[color:color-mix(in_srgb,var(--apple-warning)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_10%,var(--apple-surface))]'
                  ]"
                >
                  <Icon
                    v-if="isBalanceType(item.type)"
                    name="dollar"
                    size="md"
                    :class="
                      item.value >= 0
                        ? 'text-[var(--apple-success)]'
                        : 'text-[var(--apple-danger)]'
                    "
                  />
                  <Icon
                    v-else-if="isSubscriptionType(item.type)"
                    name="badge"
                    size="md"
                    class="text-[var(--apple-blue)]"
                  />
                  <Icon
                    v-else
                    name="bolt"
                    size="md"
                    :class="
                      item.value >= 0
                        ? 'text-[var(--apple-blue)]'
                        : 'text-[var(--apple-warning)]'
                    "
                  />
                </div>
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-[var(--apple-text)]">
                    {{ getHistoryItemTitle(item) }}
                  </p>
                  <p class="text-xs text-[var(--apple-muted)]">
                    {{ formatDateTime(item.used_at) }}
                  </p>
                </div>
              </div>
              <div class="min-w-0 text-left sm:text-right">
                <p
                  :class="[
                    'text-sm font-semibold',
                    isBalanceType(item.type)
                      ? item.value >= 0
                        ? 'text-[var(--apple-success)]'
                        : 'text-[var(--apple-danger)]'
                      : isSubscriptionType(item.type)
                        ? 'text-[var(--apple-blue)]'
                        : item.value >= 0
                          ? 'text-[var(--apple-blue)]'
                          : 'text-[var(--apple-warning)]'
                  ]"
                >
                  {{ formatHistoryValue(item) }}
                </p>
                <p
                  v-if="!isAdminAdjustment(item.type)"
                  class="font-mono text-xs text-[var(--apple-muted)]"
                >
                  {{ item.code.slice(0, 8) }}...
                </p>
                <p v-else class="text-xs text-[var(--apple-muted)]">
                  {{ t('redeem.adminAdjustment') }}
                </p>
                <p
                  v-if="item.notes"
                  class="mt-1 max-w-full truncate text-xs italic text-[var(--apple-muted)] sm:max-w-[220px]"
                  :title="item.notes"
                >
                  {{ item.notes }}
                </p>
              </div>
            </div>
          </div>

          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-14 w-14 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)]"
            >
              <Icon name="clock" size="xl" class="text-[var(--apple-muted)]" />
            </div>
            <p class="text-sm text-[var(--apple-muted)]">
              {{ t('redeem.historyWillAppear') }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'
import { redeemAPI, authAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import { formatDateTime } from '@/utils/format'
import { formatCreditedBalance } from '@/custom/payment/orderAmounts'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()
const subscriptionCapabilityStore = useSubscriptionCapabilityStore()

const user = computed(() => authStore.user)
const redeemAssuranceItems = computed(() => [
  t('redeem.assurance.official'),
  t('redeem.assurance.singleUse'),
  t('redeem.assurance.coverage'),
  t('redeem.assurance.privacy'),
  t('redeem.assurance.instantUpdate')
])

const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
  group_name?: string
  validity_days?: number
} | null>(null)
const errorMessage = ref('')

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const contactInfo = ref('')

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${formatCreditedBalance(item.value)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async () => {
  loadingHistory.value = true
  try {
    history.value = await redeemAPI.getHistory()
  } catch (error) {
    console.error('Failed to fetch history:', error)
  } finally {
    loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (!redeemCode.value.trim()) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null

  try {
    const result = await redeemAPI.redeem(redeemCode.value.trim())

    redeemResult.value = result

    // Refresh user data to get updated balance/concurrency
    await authStore.refreshUser()

    // If subscription type, immediately refresh subscription status
    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true) // force refresh
        await subscriptionCapabilityStore.fetchSubscriptionCapability(true)
      } catch (error) {
        console.error('Failed to refresh subscriptions after redeem:', error)
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    // Clear the input
    redeemCode.value = ''

    // Refresh history
    await fetchHistory()

    // Show success toast
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('redeem.failedToRedeem')

    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
  } catch (error) {
    console.error('Failed to load contact info:', error)
  }
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
