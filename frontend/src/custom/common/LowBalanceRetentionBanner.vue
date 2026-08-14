<template>
  <div
    v-if="shouldShow"
    :class="[
      'relative w-full overflow-hidden rounded-lg border bg-[var(--apple-surface)] px-4 py-3 text-[var(--apple-text)] shadow-sm',
      rootToneClass,
    ]"
    :role="alertRole"
    :aria-live="liveMode"
  >
    <div class="absolute inset-y-0 left-0 w-1" :class="accentToneClass" />
    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex min-w-0 gap-3">
        <div
          :class="[
            'mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ring-1',
            iconToneClass,
          ]"
        >
          <Icon :name="isCritical ? 'exclamationCircle' : 'exclamationTriangle'" size="lg" />
        </div>

        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <p class="text-sm font-semibold">
              {{ title }}
            </p>
            <span
              class="rounded-md px-2 py-0.5 text-xs font-semibold"
              :class="badgeToneClass"
            >
              {{ balanceLabel }}
            </span>
          </div>
          <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
            {{ message }}
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2 self-start lg:self-center">
        <a
          v-if="docsLink"
          :href="docsLink.href"
          :target="docsLink.external ? '_blank' : undefined"
          :rel="docsLink.external ? 'noopener noreferrer' : undefined"
          class="btn btn-secondary btn-sm gap-1.5"
          @click="handleDocsLinkClick"
        >
          <Icon name="book" size="sm" />
          {{ secondaryLabel }}
        </a>

        <RouterLink v-else class="btn btn-secondary btn-sm gap-1.5" to="/usage">
          <Icon name="chartBar" size="sm" />
          {{ secondaryLabel }}
        </RouterLink>

        <RouterLink class="btn btn-primary btn-sm gap-1.5" to="/purchase">
          <Icon name="creditCard" size="sm" />
          {{ purchaseLabel }}
        </RouterLink>

        <button
          type="button"
          class="btn btn-secondary btn-icon btn-sm shrink-0"
          :title="closeLabel"
          :aria-label="closeLabel"
          @click="dismissForAwhile"
        >
          <Icon name="x" size="sm" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useMinuteNow } from '@/custom/composables/useMinuteNow'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { safeGetStorageItem, safeRemoveStorageItem, safeSetStorageItem } from '@/custom/utils/browserStorage'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/custom/utils/docsLink'

const DISMISS_DURATION_MS = 24 * 60 * 60 * 1000

const appStore = useAppStore()
const authStore = useAuthStore()
const paymentCheckoutStore = usePaymentCheckoutStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { formatSettlementAmount } = useSettlementCurrency()
const now = useMinuteNow()
const dismissalToken = ref(0)
const dismissedUntil = ref<number | null>(null)

const user = computed(() => authStore.user)
const balance = computed(() => Number(user.value?.balance ?? NaN))
const threshold = computed(() => {
  const rawThreshold = appStore.cachedPublicSettings?.balance_low_notify_threshold
  const parsedThreshold = Number(rawThreshold)
  return Number.isFinite(parsedThreshold) && parsedThreshold > 0 ? parsedThreshold : 1
})
const lowBalanceNotifyEnabled = computed(() => appStore.cachedPublicSettings?.balance_low_notify_enabled === true)
const paymentEnabled = computed(() => appStore.cachedPublicSettings?.payment_enabled !== false && paymentCheckoutStore.canAccessPurchase)
const routeAllowsBanner = computed(() => route.meta.requiresAdmin !== true && !route.path.startsWith('/purchase'))
const balanceIsValid = computed(() => Number.isFinite(balance.value))
const isLowBalance = computed(() => balanceIsValid.value && balance.value <= threshold.value)
const isCritical = computed(() => balanceIsValid.value && balance.value <= 0)
const balanceBucket = computed(() => {
  if (!user.value || !balanceIsValid.value) return ''
  return `${user.value.id}:${balance.value.toFixed(2)}`
})
const storageKey = computed(() =>
  balanceBucket.value ? `low_balance_retention_banner:${balanceBucket.value}` : ''
)

const docsLink = computed(() =>
  resolveDocsLink(
    appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '',
    appStore.cachedPublicSettings?.custom_menu_items ?? [],
  )
)

const shouldShow = computed(() => {
  if (!authStore.isAuthenticated) return false
  if (!routeAllowsBanner.value) return false
  if (!lowBalanceNotifyEnabled.value) return false
  if (!paymentEnabled.value) return false
  if (!isLowBalance.value) return false
  return dismissedUntil.value === null || dismissedUntil.value <= now.value
})

const formattedBalance = computed(() => formatSettlementAmount(balance.value, 2))
const formattedThreshold = computed(() => formatSettlementAmount(threshold.value, 2))

const balanceLabel = computed(() => {
  if (!balanceIsValid.value) return ''
  return t('dashboard.retention.lowBalanceBanner.balanceLabel', {
    balance: formattedBalance.value,
  })
})

const title = computed(() => {
  return isCritical.value
    ? t('dashboard.retention.lowBalanceBanner.criticalTitle')
    : t('dashboard.retention.lowBalanceBanner.lowTitle')
})

const message = computed(() => {
  if (isCritical.value) {
    return t('dashboard.retention.lowBalanceBanner.criticalMessage')
  }
  return t('dashboard.retention.lowBalanceBanner.lowMessage', {
    threshold: formattedThreshold.value,
  })
})

const purchaseLabel = computed(() => t('dashboard.retention.lowBalanceBanner.primaryAction'))
const secondaryLabel = computed(() =>
  docsLink.value
    ? t('dashboard.retention.lowBalanceBanner.docsAction')
    : t('dashboard.retention.lowBalanceBanner.usageAction')
)
const closeLabel = computed(() => t('dashboard.retention.lowBalanceBanner.dismiss24h'))
const alertRole = computed(() => (isCritical.value ? 'alert' : 'status'))
const liveMode = computed(() => (isCritical.value ? 'assertive' : 'polite'))
const rootToneClass = computed(() =>
  isCritical.value
    ? 'border-red-200/80 dark:border-red-500/30'
    : 'border-amber-200/80 dark:border-amber-500/30'
)
const accentToneClass = computed(() => (isCritical.value ? 'bg-red-500 dark:bg-red-400' : 'bg-amber-500 dark:bg-amber-400'))
const iconToneClass = computed(() =>
  isCritical.value
    ? 'bg-red-50 text-red-600 ring-red-100 dark:bg-red-500/10 dark:text-red-300 dark:ring-red-500/20'
    : 'bg-amber-50 text-amber-700 ring-amber-100 dark:bg-amber-500/10 dark:text-amber-300 dark:ring-amber-500/20'
)
const badgeToneClass = computed(() =>
  isCritical.value
    ? 'bg-red-50 text-red-700 ring-1 ring-red-100 dark:bg-red-500/10 dark:text-red-200 dark:ring-red-500/20'
    : 'bg-amber-50 text-amber-700 ring-1 ring-amber-100 dark:bg-amber-500/10 dark:text-amber-200 dark:ring-amber-500/20'
)

function readDismissal(): void {
  if (!storageKey.value) {
    dismissedUntil.value = null
    return
  }

  const raw = safeGetStorageItem('localStorage', storageKey.value)
  if (!raw) {
    dismissedUntil.value = null
    return
  }

  const parsed = Number(raw)
  if (!Number.isFinite(parsed) || parsed <= now.value) {
    safeRemoveStorageItem('localStorage', storageKey.value)
    dismissedUntil.value = null
    return
  }

  dismissedUntil.value = parsed
}

function dismissForAwhile(): void {
  if (!storageKey.value) return
  const expiresAt = Date.now() + DISMISS_DURATION_MS
  safeSetStorageItem('localStorage', storageKey.value, String(expiresAt))
  dismissedUntil.value = expiresAt
  dismissalToken.value += 1
}

function handleDocsLinkClick(event: MouseEvent): void {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

watch(storageKey, readDismissal, { immediate: true })
watch([now, dismissalToken], readDismissal)
watch(
  [
    () => authStore.isAuthenticated,
    () => authStore.user?.id,
    () => appStore.cachedPublicSettings?.payment_enabled,
  ],
  () => {
    if (!authStore.isAuthenticated || appStore.cachedPublicSettings?.payment_enabled === false) return
    paymentCheckoutStore.fetchCheckoutInfo().catch((error) => {
      console.error('Failed to preload checkout info:', error)
    })
  },
  { immediate: true },
)
</script>
