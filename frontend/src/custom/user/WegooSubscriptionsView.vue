<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <UserPageHero
        :title="t('userSubscriptions.title')"
        :description="t('userSubscriptions.description')"
      >
        <template #actions>
          <button v-if="canAccessPurchase" class="btn btn-primary w-full justify-center sm:w-auto" @click="router.push('/purchase')">
            {{ t('payment.result.backToRecharge') }}
          </button>
        </template>

      </UserPageHero>

      <div
        v-if="loading"
        class="flex justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] py-16 shadow-sm"
      >
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"
        ></div>
      </div>

      <div
        v-else-if="subscriptions.length === 0"
        class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-6 py-12 text-center shadow-sm sm:px-12"
      >
        <div
          class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]"
        >
          <Icon name="creditCard" size="xl" class="text-[var(--apple-muted)]" />
        </div>
        <h3 class="text-lg font-semibold text-[var(--apple-text)]">
          {{ t('userSubscriptions.noActiveSubscriptions') }}
        </h3>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-[var(--apple-muted)]">
          {{ t('userSubscriptions.noActiveSubscriptionsDesc') }}
        </p>
        <button v-if="canAccessPurchase" class="btn btn-primary mt-5" @click="router.push('/purchase')">
          {{ t('payment.result.backToRecharge') }}
        </button>
      </div>

      <div v-else class="grid gap-4 lg:grid-cols-2">
        <article
          v-for="subscription in subscriptions"
          :key="subscription.id"
          class="overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm"
        >
          <div class="p-5">
            <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2 text-xs font-medium text-[var(--apple-muted)]">
                  <span
                    :class="[
                      'h-1.5 w-1.5 rounded-full',
                      platformAccentDotClass(subscription.group?.platform || '')
                    ]"
                  />
                  <span class="truncate">
                    {{ platformLabel(subscription.group?.platform || '') }}
                  </span>
                </div>
                <h2 class="mt-2 truncate text-lg font-semibold text-[var(--apple-text)]">
                  {{ subscriptionName(subscription) }}
                </h2>
                <p
                  v-if="subscriptionDescription(subscription)"
                  class="mt-1 text-sm leading-6 text-[var(--apple-muted)]"
                >
                  {{ subscriptionDescription(subscription) }}
                </p>
                <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-gray-400 dark:text-gray-500">
                  <span>{{ t('payment.planCard.rate') }}: ×{{ subscription.group?.rate_multiplier ?? 1 }}</span>
                  <span v-if="subscriptionHasPeakRate(subscription)" class="text-amber-700 dark:text-amber-300">
                    {{ t('payment.planCard.peakRate') }}: {{ subscriptionPeakRateLabel(subscription) }}
                  </span>
                </div>
              </div>

              <div class="grid w-full grid-cols-2 gap-2 sm:flex sm:w-auto sm:flex-wrap sm:items-center sm:justify-end">
                <span
                  :class="[
                    'inline-flex min-h-8 items-center justify-center rounded-full border px-2.5 py-1 text-xs font-medium sm:min-h-0',
                    subscriptionStatusClass(subscription.status)
                  ]"
                >
                  {{ t(`userSubscriptions.status.${subscription.status}`) }}
                </span>
                <button
                  v-if="canReset(subscription)"
                  class="btn btn-secondary btn-sm w-full sm:w-auto"
                  @click="openResetDialog(subscription)"
                >
                  {{ t('userSubscriptions.reset') }}
                </button>
                <button
                  v-if="subscription.status === 'active' && canAccessPurchase"
                  class="btn btn-primary btn-sm w-full sm:w-auto"
                  @click="router.push({ path: '/purchase', query: { tab: 'subscription', group: String(subscription.group_id) } })"
                >
                  {{ t('payment.renewNow') }}
                </button>
              </div>
            </div>

            <dl class="mt-5 border-y border-[color:var(--apple-border-soft)] py-4">
              <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                <dt class="text-sm text-[var(--apple-muted)]">
                  {{ t('userSubscriptions.expires') }}
                </dt>
                <dd v-if="subscription.expires_at" :class="['text-sm', getExpirationClass(subscription.expires_at)]">
                  {{ formatExpirationDate(subscription.expires_at) }}
                </dd>
                <dd v-else class="text-sm font-medium text-[var(--apple-text)]">
                  {{ t('userSubscriptions.noExpiration') }}
                </dd>
              </div>
            </dl>

            <div v-if="hasQuotaLimits(subscription)" class="mt-5 space-y-5">
              <section v-if="subscription.group?.daily_limit_usd" class="space-y-2">
                <div class="flex items-center justify-between gap-3">
                  <h3 class="text-sm font-medium text-[var(--apple-text)]">
                    {{ t('userSubscriptions.daily') }}
                  </h3>
                  <span class="shrink-0 text-sm text-[var(--apple-muted)]">
                    {{
                      formatSettlementAmountPair(
                        subscription.daily_usage_usd || 0,
                        subscription.group.daily_limit_usd,
                        2
                      )
                    }}
                  </span>
                </div>
                <div class="relative h-1.5 overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                    :class="
                      getProgressBarClass(
                        subscription.daily_usage_usd,
                        subscription.group.daily_limit_usd
                      )
                    "
                    :style="{
                      width: getProgressWidth(
                        subscription.daily_usage_usd,
                        subscription.group.daily_limit_usd
                      )
                    }"
                  ></div>
                </div>
                <p
                  v-if="subscription.daily_window_start && formatDailyUsageWindow(subscription)"
                  class="text-xs text-[var(--apple-muted)]"
                >
                  {{ formatDailyUsageWindow(subscription) }}
                </p>

                <div class="flex flex-col gap-3 border-t border-[color:var(--apple-border-soft)] pt-4 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0">
                    <div class="text-sm font-medium text-[var(--apple-text)]">
                      {{ t('userSubscriptions.autoResetLabel') }}
                    </div>
                    <div class="mt-0.5 text-xs leading-5 text-[var(--apple-muted)]">
                      {{ t('userSubscriptions.autoResetHint') }}
                    </div>
                  </div>
                  <button
                    type="button"
                    role="switch"
                    :aria-checked="subscription.auto_reset_daily"
                    :disabled="autoResetSubmitting[subscription.id]"
                    :class="[
                      'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-[color:var(--apple-focus-ring)] focus:ring-offset-2',
                      subscription.auto_reset_daily ? 'bg-[var(--apple-blue)]' : 'bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]',
                      autoResetSubmitting[subscription.id] ? 'cursor-not-allowed opacity-50' : ''
                    ]"
                    @click="toggleAutoReset(subscription, !subscription.auto_reset_daily)"
                  >
                    <span
                      :class="[
                        'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                        subscription.auto_reset_daily ? 'translate-x-5' : 'translate-x-0'
                      ]"
                    ></span>
                  </button>
                </div>
              </section>

              <section v-if="subscription.group?.weekly_limit_usd" class="space-y-2">
                <div class="flex items-center justify-between gap-3">
                  <h3 class="text-sm font-medium text-[var(--apple-text)]">
                    {{ t('userSubscriptions.weekly') }}
                  </h3>
                  <span class="shrink-0 text-sm text-[var(--apple-muted)]">
                    {{
                      formatSettlementAmountPair(
                        subscription.weekly_usage_usd || 0,
                        subscription.group.weekly_limit_usd,
                        2
                      )
                    }}
                  </span>
                </div>
                <div class="relative h-1.5 overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                    :class="
                      getProgressBarClass(
                        subscription.weekly_usage_usd,
                        subscription.group.weekly_limit_usd
                      )
                    "
                    :style="{
                      width: getProgressWidth(
                        subscription.weekly_usage_usd,
                        subscription.group.weekly_limit_usd
                      )
                    }"
                  ></div>
                </div>
                <p
                  v-if="subscription.weekly_window_start && formatResetTime(subscription.weekly_window_start, 168, subscription.expires_at)"
                  class="text-xs text-[var(--apple-muted)]"
                >
                  {{
                    t('userSubscriptions.resetIn', {
                      time: formatResetTime(subscription.weekly_window_start, 168, subscription.expires_at)
                    })
                  }}
                </p>
              </section>

              <section v-if="subscription.group?.monthly_limit_usd" class="space-y-2">
                <div class="flex items-center justify-between gap-3">
                  <h3 class="text-sm font-medium text-[var(--apple-text)]">
                    {{ t('userSubscriptions.monthly') }}
                  </h3>
                  <span class="shrink-0 text-sm text-[var(--apple-muted)]">
                    {{
                      formatSettlementAmountPair(
                        subscription.monthly_usage_usd || 0,
                        subscription.group.monthly_limit_usd,
                        2
                      )
                    }}
                  </span>
                </div>
                <div class="relative h-1.5 overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                    :class="
                      getProgressBarClass(
                        subscription.monthly_usage_usd,
                        subscription.group.monthly_limit_usd
                      )
                    "
                    :style="{
                      width: getProgressWidth(
                        subscription.monthly_usage_usd,
                        subscription.group.monthly_limit_usd
                      )
                    }"
                  ></div>
                </div>
                <p
                  v-if="subscription.monthly_window_start && formatResetTime(subscription.monthly_window_start, 720, subscription.expires_at)"
                  class="text-xs text-[var(--apple-muted)]"
                >
                  {{
                    t('userSubscriptions.resetIn', {
                      time: formatResetTime(subscription.monthly_window_start, 720, subscription.expires_at)
                    })
                  }}
                </p>
              </section>
            </div>

            <div v-else class="mt-5 flex items-center gap-3 border-t border-[color:var(--apple-border-soft)] pt-5">
              <span class="text-3xl leading-none text-[var(--apple-muted-2)]">∞</span>
              <div class="min-w-0">
                <p class="text-sm font-medium text-[var(--apple-text)]">
                  {{ t('userSubscriptions.unlimited') }}
                </p>
                <p class="mt-0.5 text-xs leading-5 text-[var(--apple-muted)]">
                  {{ t('userSubscriptions.unlimitedDesc') }}
                </p>
                <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-gray-400 dark:text-gray-500">
                  <span>{{ t('payment.planCard.rate') }}: ×{{ subscription.group?.rate_multiplier ?? 1 }}</span>
                  <span v-if="subscriptionHasPeakRate(subscription)" class="text-amber-700 dark:text-amber-300">
                    {{ t('payment.planCard.peakRate') }}: {{ subscriptionPeakRateLabel(subscription) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>

    <ConfirmDialog
      :show="showResetDialog"
      :title="t('userSubscriptions.resetTitle')"
      :message="resetDialogMessage"
      :confirm-text="t('userSubscriptions.reset')"
      :cancel-text="t('common.cancel')"
      @confirm="confirmReset"
      @cancel="showResetDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import subscriptionsAPI from '@/custom/api/subscriptions'
import type { UserSubscription } from '@/types'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { platformLabel } from '@/utils/platformColors'
import { getRemainingDurationParts, isOneTimeDailyQuota, type RemainingDurationParts } from '@/utils/subscriptionQuota'

function platformAccentDotClass(p: string): string {
  switch (p) {
    case 'anthropic': return 'bg-[var(--apple-warning)]'
    case 'openai': return 'bg-[var(--apple-success)]'
    case 'antigravity':
    case 'gemini': return 'bg-[var(--apple-blue)]'
    default: return 'bg-[var(--apple-muted-2)]'
  }
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const paymentCheckoutStore = usePaymentCheckoutStore()
const { formatSettlementAmountPair } = useSettlementCurrency()
const canAccessPurchase = computed(() => paymentCheckoutStore.canAccessPurchase)

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)

const showResetDialog = ref(false)
const resettingSubscription = ref<UserSubscription | null>(null)
const resetLoading = ref(false)

const autoResetSubmitting = ref<Record<number, boolean>>({})

async function toggleAutoReset(sub: UserSubscription, next: boolean) {
  if (autoResetSubmitting.value[sub.id]) return
  autoResetSubmitting.value[sub.id] = true
  try {
    const updated = await subscriptionsAPI.setAutoResetDaily(sub.id, next)
    const idx = subscriptions.value.findIndex(s => s.id === sub.id)
    if (idx >= 0) subscriptions.value[idx] = updated
    appStore.showSuccess(
      next
        ? t('userSubscriptions.autoResetEnabled')
        : t('userSubscriptions.autoResetDisabled')
    )
  } catch {
    appStore.showError(t('userSubscriptions.autoResetFailed'))
  } finally {
    autoResetSubmitting.value[sub.id] = false
  }
}

function getRemainingSeconds(expiresAt: string | null | undefined): number {
  if (!expiresAt) return 0
  return Math.max(0, (new Date(expiresAt).getTime() - Date.now()) / 1000)
}

function canReset(sub: UserSubscription): boolean {
  if (sub.status !== 'active') return false
  if (!sub.daily_window_start) return false
  const windowEndMs = new Date(sub.daily_window_start).getTime() + 86400_000
  if (windowEndMs <= Date.now()) return false
  if (getRemainingSeconds(sub.expires_at) <= 86400) return false
  const dailyLimit = sub.group?.daily_limit_usd
  if (!dailyLimit || dailyLimit <= 0) return false
  return (sub.daily_usage_usd || 0) >= dailyLimit
}

function getDaysCeil(expiresAt: string | null | undefined): number {
  return Math.ceil(getRemainingSeconds(expiresAt) / 86400)
}

interface ResetInfo {
  costSeconds: number
  beforeSeconds: number
  afterSeconds: number
}

function computeResetInfo(sub: UserSubscription | null | undefined): ResetInfo {
  if (!sub || !sub.daily_window_start || !sub.expires_at) {
    return { costSeconds: 0, beforeSeconds: 0, afterSeconds: 0 }
  }
  const windowEndMs = new Date(sub.daily_window_start).getTime() + 86400_000
  const nowMs = Date.now()
  const costSeconds = Math.max(0, (windowEndMs - nowMs) / 1000)
  const beforeSeconds = getRemainingSeconds(sub.expires_at)
  const afterSeconds = Math.max(0, beforeSeconds - costSeconds)
  return { costSeconds, beforeSeconds, afterSeconds }
}

const resetDialogMessage = computed(() => {
  if (!resettingSubscription.value) return ''
  const info = computeResetInfo(resettingSubscription.value)
  return t('userSubscriptions.resetConfirm', {
    cost: formatDuration(info.costSeconds),
    before: formatDuration(info.beforeSeconds),
    after: formatDuration(info.afterSeconds)
  })
})

function openResetDialog(sub: UserSubscription) {
  resettingSubscription.value = sub
  showResetDialog.value = true
}

async function confirmReset() {
  if (!resettingSubscription.value || resetLoading.value) return
  resetLoading.value = true
  try {
    const updated = await subscriptionsAPI.resetSubscription(resettingSubscription.value.id)
    const days = getDaysCeil(updated.expires_at)
    appStore.showSuccess(t('userSubscriptions.resetSuccess', { days }))
    showResetDialog.value = false
    resettingSubscription.value = null
    await loadSubscriptions()
  } catch (error: any) {
    const code = error?.code
    let msg = t('userSubscriptions.resetFailed')
    if (code === 'SUBSCRIPTION_TIME_INSUFFICIENT') msg = t('userSubscriptions.resetError.timeInsufficient')
    else if (code === 'SUBSCRIPTION_NOT_OWNED') msg = t('userSubscriptions.resetError.notOwned')
    else if (code === 'SUBSCRIPTION_INACTIVE') msg = t('userSubscriptions.resetError.inactive')
    else if (code === 'SUBSCRIPTION_NOT_FOUND') msg = t('userSubscriptions.resetError.notFound')
    appStore.showError(msg)
    showResetDialog.value = false
    resettingSubscription.value = null
  } finally {
    resetLoading.value = false
  }
}

function subscriptionHasPeakRate(subscription: UserSubscription): boolean {
  return hasPeakRate(subscription.group)
}

function subscriptionPeakRateLabel(subscription: UserSubscription): string {
  return formatPeakRateWindow(subscription.group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

async function loadSubscriptions() {
  try {
    loading.value = true
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch (error) {
    console.error('Failed to load subscriptions:', error)
    appStore.showError(t('userSubscriptions.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function subscriptionName(subscription: UserSubscription): string {
  return subscription.group?.name || `Group #${subscription.group_id}`
}

function subscriptionDescription(subscription: UserSubscription): string {
  return subscription.group?.description || ''
}

function hasQuotaLimits(subscription: UserSubscription): boolean {
  return Boolean(
    subscription.group?.daily_limit_usd ||
      subscription.group?.weekly_limit_usd ||
      subscription.group?.monthly_limit_usd
  )
}

function subscriptionStatusClass(status: UserSubscription['status']): string {
  if (status === 'active') {
    return 'border-[color:color-mix(in_srgb,var(--apple-success)_25%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_8%,var(--apple-surface))] text-[var(--apple-success)]'
  }
  if (status === 'expired') {
    return 'border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]'
  }
  return 'border-[color:color-mix(in_srgb,var(--apple-danger)_25%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_8%,var(--apple-surface))] text-[var(--apple-danger)]'
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return 'bg-[var(--apple-muted-2)]'
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'bg-[var(--apple-danger)]'
  if (percentage >= 70) return 'bg-[var(--apple-warning)]'
  return 'bg-[var(--apple-blue)]'
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return t('userSubscriptions.durationLessThanMinute')
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (days > 0) parts.push(t('userSubscriptions.durationDays', { n: days }))
  if (hours > 0) parts.push(t('userSubscriptions.durationHours', { n: hours }))
  if (minutes > 0 || parts.length === 0) parts.push(t('userSubscriptions.durationMinutes', { n: minutes }))
  return parts.join(' ')
}

function formatLocalDateTime(d: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function formatExpirationDate(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diffMs = expires.getTime() - now.getTime()

  if (diffMs < 0) {
    return t('userSubscriptions.status.expired')
  }

  const dateTimeStr = formatLocalDateTime(expires)
  const remaining = formatDuration(diffMs / 1000)
  return `${dateTimeStr} (${t('userSubscriptions.remainingPrefix')} ${remaining})`
}

function getExpirationClass(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

  if (days <= 0) return 'font-medium text-[var(--apple-danger)]'
  if (days <= 3) return 'text-[var(--apple-danger)]'
  if (days <= 7) return 'text-[var(--apple-warning)]'
  return 'font-medium text-[var(--apple-text)]'
}

function formatDurationParts(parts: RemainingDurationParts): string {
  const chunks: string[] = []
  if (parts.days > 0) chunks.push(t('userSubscriptions.durationDays', { n: parts.days }))
  if (parts.hours > 0) chunks.push(t('userSubscriptions.durationHours', { n: parts.hours }))
  if (parts.minutes > 0 || chunks.length === 0) {
    chunks.push(t('userSubscriptions.durationMinutes', { n: parts.minutes }))
  }
  return chunks.join(' ')
}

function formatResetTime(
  windowStart: string | null,
  windowHours: number,
  expiresAt?: string | null
): string {
  if (!windowStart) return ''

  const start = new Date(windowStart)
  const end = new Date(start.getTime() + windowHours * 60 * 60 * 1000)
  const parts = getRemainingDurationParts(end)

  if (!parts) return ''

  if (expiresAt) {
    const expires = new Date(expiresAt)
    if (end > expires) return ''
  }

  return formatDurationParts(parts)
}

function formatDailyUsageWindow(subscription: UserSubscription): string {
  if (isOneTimeDailyQuota(subscription) && subscription.expires_at) {
    const parts = getRemainingDurationParts(subscription.expires_at)
    if (!parts) return ''
    return t('userSubscriptions.quotaEndsIn', { time: formatDurationParts(parts) })
  }

  const resetTime = formatResetTime(subscription.daily_window_start, 24, subscription.expires_at)
  return resetTime ? t('userSubscriptions.resetIn', { time: resetTime }) : ''
}

onMounted(() => {
  paymentCheckoutStore.fetchCheckoutInfo().catch(() => {})
  loadSubscriptions()
})
</script>
