<template>
  <header class="app-header sticky top-0 z-30 border-b backdrop-blur">
    <div class="mx-auto flex h-14 w-full max-w-[1680px] items-center justify-between gap-3 px-3 sm:h-16 sm:px-4 md:px-6 lg:px-8">
      <!-- Left: Mobile Menu Toggle + Page Title -->
      <div class="flex min-w-0 flex-1 items-center gap-2 sm:gap-3">
        <button
          @click="toggleMobileSidebar"
          class="btn-ghost btn-icon lg:hidden"
          aria-label="Toggle Menu"
        >
          <Icon name="menu" size="md" />
        </button>

        <div class="hidden min-w-0 sm:block">
          <h1 class="truncate text-base font-semibold leading-6 text-[var(--apple-text)]">
            {{ pageTitle }}
          </h1>
          <p v-if="pageDescription" class="truncate text-xs text-[var(--apple-muted)]">
            {{ pageDescription }}
          </p>
        </div>
      </div>

      <!-- Right: Announcements + Docs + Language + Subscriptions + Balance + User Dropdown -->
      <div class="flex min-w-0 shrink-0 items-center justify-end gap-1 sm:gap-2 md:gap-2.5">
        <!-- Announcement Bell -->
        <AnnouncementBell v-if="user" />

        <!-- Ticket unread shortcut -->
        <router-link
          v-if="user && showTicketShortcut"
          :to="ticketLink"
          class="header-icon-button relative flex items-center rounded-lg p-2 transition-colors"
          :title="t('tickets.title')"
        >
          <Icon name="chatBubble" size="md" />
          <span
            v-if="ticketUnreadCount > 0"
            class="absolute -right-1 -top-1 min-w-[1.125rem] rounded-full bg-red-500 px-1.5 text-center text-[10px] font-semibold leading-[1.125rem] text-white"
          >
            {{ formatBadgeCount(ticketUnreadCount) }}
          </span>
        </router-link>

        <!-- Docs Link -->
        <a
          v-if="docsLink"
          :href="docsLink.href"
          :target="docsLink.external ? '_blank' : undefined"
          :rel="docsLink.external ? 'noopener noreferrer' : undefined"
          class="header-action-link hidden items-center gap-1.5 rounded-lg border px-3 py-2 text-sm font-medium transition-colors sm:flex"
          @click="handleDocsLinkClick"
        >
          <Icon name="book" size="sm" />
          <span>{{ t('nav.docs') }}</span>
        </a>

        <!-- Model Plaza Entry -->
        <router-link
          v-if="user && modelPlazaEnabled"
          :to="{ path: '/model-plaza', query: { embedded: '1' } }"
          class="hidden items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white sm:flex"
        >
          <Icon name="grid" size="sm" />
          <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
        </router-link>

        <!-- Language Switcher -->
        <div class="hidden sm:block">
          <LocaleSwitcher />
        </div>

        <!-- Subscription Progress (for users with active subscriptions) -->
        <div class="hidden md:block">
          <SubscriptionProgressMini v-if="user" />
        </div>

        <!-- Balance Display -->
        <div
          v-if="user"
          class="header-balance hidden items-center gap-2 rounded-lg border px-3 py-1.5 sm:flex"
        >
          <Icon name="dollar" size="sm" class="text-[var(--apple-blue)]" />
          <div class="flex min-w-[5.75rem] flex-col leading-tight">
            <span class="text-sm font-semibold text-[var(--apple-blue)]">
              {{ formattedBalance }}
            </span>
            <span class="text-[11px] font-medium text-[var(--apple-muted)]">
              {{ formattedBalanceSubtitle }}
            </span>
          <span
            v-if="frozenBalance > 0"
            class="mt-1 text-[11px] font-medium text-amber-600"
          >
            {{ balanceFrozenLabel }}
          </span>
          </div>
        </div>

        <!-- User Dropdown -->
        <div v-if="user" class="relative" ref="dropdownRef">
          <button
            @click="toggleDropdown"
            class="header-user-button flex items-center gap-2 rounded-lg p-1.5 transition-colors"
            aria-label="User Menu"
          >
            <div class="header-avatar flex h-8 w-8 items-center justify-center overflow-hidden rounded-lg text-sm font-medium text-white">
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                :alt="displayName"
                class="h-full w-full object-cover"
              >
              <span v-else>{{ userInitials }}</span>
            </div>
            <div class="hidden min-w-0 text-left md:block">
              <div class="max-w-32 truncate text-sm font-medium text-[var(--apple-text)]">
                {{ displayName }}
              </div>
              <div class="text-xs capitalize text-[var(--apple-muted)]">
                {{ user.role }}
              </div>
            </div>
            <Icon name="chevronDown" size="sm" class="hidden text-[var(--apple-muted-2)] md:block" />
          </button>

          <!-- Dropdown Menu -->
          <transition name="dropdown">
            <div v-if="dropdownOpen" class="dropdown right-0 mt-2 w-56 max-w-[calc(100vw-1rem)]">
              <!-- User Info -->
              <div class="border-b border-[color:var(--apple-border-soft)] px-4 py-3">
                <div class="truncate text-sm font-medium text-[var(--apple-text)]">
                  {{ displayName }}
                </div>
                <div class="truncate text-xs text-[var(--apple-muted)]">{{ user.email }}</div>
              </div>

              <!-- Balance (mobile only) -->
              <div class="border-b border-[color:var(--apple-border-soft)] px-4 py-2 sm:hidden">
                <div class="text-xs text-[var(--apple-muted)]">
                  {{ t('common.balance') }}
                </div>
                <div class="text-sm font-semibold text-[var(--apple-blue)]">
                  {{ formattedBalance }}
                </div>
                <div class="text-xs font-medium text-[var(--apple-muted)]">
                  {{ formattedBalanceSubtitle }}
                </div>
                <div v-if="frozenBalance > 0" class="mt-1 text-xs font-medium text-amber-600">
                  {{ balanceFrozenLabel }}
                </div>
              </div>

              <div class="border-b border-[color:var(--apple-border-soft)] px-4 py-2.5">
                <div class="mb-2 text-xs text-[var(--apple-muted)]">
                  {{ t('settlementCurrency.label') }}
                </div>
                <div class="header-segmented inline-flex rounded-lg p-1">
                  <button
                    v-for="option in settlementCurrencyOptions"
                    :key="option.value"
                    type="button"
                    class="header-segmented-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
                    :class="
                      settlementCurrency === option.value
                        ? 'header-segmented-button-active'
                        : ''
                    "
                    @click.stop="setSettlementCurrency(option.value)"
                  >
                    {{ option.label }}
                  </button>
                </div>
              </div>

              <div class="py-1">
                <a
                  v-if="docsLink"
                  :href="docsLink.href"
                  :target="docsLink.external ? '_blank' : undefined"
                  :rel="docsLink.external ? 'noopener noreferrer' : undefined"
                  @click="handleDocsDropdownClick"
                  class="dropdown-item sm:hidden"
                >
                  <Icon name="book" size="sm" />
                  {{ t('nav.docs') }}
                </a>

                <router-link
                  v-if="showMySubscriptions"
                  to="/subscriptions"
                  @click="closeDropdown"
                  class="dropdown-item md:hidden"
                >
                  <Icon name="creditCard" size="sm" />
                  {{ t('nav.mySubscriptions') }}
                </router-link>

                <router-link to="/profile" @click="closeDropdown" class="dropdown-item">
                  <Icon name="user" size="sm" />
                  {{ t('nav.profile') }}
                </router-link>

                <router-link to="/keys" @click="closeDropdown" class="dropdown-item">
                  <Icon name="key" size="sm" />
                  {{ t('nav.apiKeys') }}
                </router-link>

                <a
                  v-if="authStore.isAdmin"
                  href="https://github.com/Wei-Shaw/sub2api"
                  target="_blank"
                  rel="noopener noreferrer"
                  @click="closeDropdown"
                  class="dropdown-item"
                >
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      fill-rule="evenodd"
                      clip-rule="evenodd"
                      d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
                    />
                  </svg>
                  {{ t('nav.github') }}
                </a>

              </div>

              <div class="border-t border-[color:var(--apple-border-soft)] px-4 py-2.5 sm:hidden">
                <div class="mb-2 text-xs text-[var(--apple-muted)]">
                  {{ t('common.language') }}
                </div>
                <div class="header-segmented grid grid-cols-2 gap-1 rounded-lg p-1">
                  <button
                    v-for="option in availableLocales"
                    :key="option.code"
                    type="button"
                    :disabled="localeSwitching"
                    class="header-segmented-button rounded-md px-2.5 py-1 text-xs font-medium transition-colors disabled:opacity-60"
                    :class="
                      currentLocaleCode === option.code
                        ? 'header-segmented-button-active'
                        : ''
                    "
                    @click.stop="setHeaderLocale(option.code)"
                  >
                    {{ option.name }}
                  </button>
                </div>
              </div>

              <!-- Contact Support (only show if configured) -->
              <div
                v-if="contactInfo"
                class="border-t border-[color:var(--apple-border-soft)] px-4 py-2.5"
              >
                <div class="flex items-center gap-2 text-xs text-[var(--apple-muted)]">
                  <Icon name="chatBubble" size="sm" class="flex-shrink-0" />
                  <span>{{ t('common.contactSupport') }}:</span>
                  <span class="min-w-0 break-all font-medium text-[var(--apple-text)]">{{
                    contactInfo
                  }}</span>
                </div>
              </div>

              <div v-if="showOnboardingButton" class="border-t border-[color:var(--apple-border-soft)] py-1">
                <button @click="handleReplayGuide" class="dropdown-item w-full">
                  <Icon name="questionCircle" size="sm" />
                  {{ $t('onboarding.restartTour') }}
                </button>
              </div>

              <div class="border-t border-[color:var(--apple-border-soft)] py-1">
                <button
                  @click="handleLogout"
                  class="dropdown-item header-danger-item w-full"
                >
                  <Icon name="login" size="sm" />
                  {{ t('nav.logout') }}
                </button>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore, useOnboardingStore, usePaymentStore } from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { useTicketStore } from '@/custom/stores/tickets'
import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import SubscriptionProgressMini from '@/custom/common/WegooSubscriptionProgressMini.vue'
import AnnouncementBell from '@/custom/common/WegooAnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatSettlementCurrencyAmount, setSettlementCnyPerCredit, useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/custom/utils/docsLink'
import { availableLocales, setLocale } from '@/i18n'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

defineOptions({ name: 'AppHeader' })

const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()
const onboardingStore = useOnboardingStore()
const paymentStore = usePaymentStore()
const ticketStore = useTicketStore()
const subscriptionCapabilityStore = useSubscriptionCapabilityStore()
const {
  settlementCurrency,
  settlementCurrencyOptions,
  setSettlementCurrency,
} = useSettlementCurrency()

const user = computed(() => authStore.user)
const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const localeSwitching = ref(false)
const contactInfo = computed(() => appStore.contactInfo)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))
const docsCustomMenuItems = computed(() => [
  ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
  ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
])
const sanitizedDocUrl = computed(() => sanitizeUrl(appStore.docUrl))
const docsLink = computed(() => resolveDocsLink(sanitizedDocUrl.value || appStore.docUrl, docsCustomMenuItems.value))
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')
const ticketLink = computed(() => (authStore.canAccessTicketAdmin ? '/admin/tickets' : '/tickets'))
const ticketUnreadCount = computed(() => (authStore.canAccessTicketAdmin ? ticketStore.adminUnreadCount : ticketStore.userUnreadCount))
const showTicketShortcut = computed(() => !appStore.backendModeEnabled || authStore.canAccessTicketAdmin)
const showMySubscriptions = computed(() => subscriptionCapabilityStore.hasSubscriptionGroups)
const currentLocaleCode = computed(() => locale.value)
const balanceCredits = computed(() => {
  const value = user.value?.balance ?? 0
  return Number.isFinite(value) ? value : 0
})
const balanceCnyPerCredit = computed(() => {
  const candidates = [
    paymentStore.config?.balance_recharge_multiplier,
    appStore.cachedPublicSettings?.payment_balance_recharge_multiplier,
  ]
  for (const candidate of candidates) {
    const value = Number(candidate)
    if (Number.isFinite(value) && value > 0) return value
  }
  return null
})

const formatBalanceAmount = (amount: number, fractionDigits = 2) => {
  if (settlementCurrency.value === 'CNY' && balanceCnyPerCredit.value === null) {
    return '--'
  }
  return formatSettlementCurrencyAmount(
    amount,
    settlementCurrency.value,
    balanceCnyPerCredit.value ?? 1,
    locale.value,
    fractionDigits,
  )
}

const formattedBalance = computed(() => formatBalanceAmount(balanceCredits.value, 2))
const formattedBalanceSubtitle = computed(() => {
  const base = formatSettlementCurrencyAmount(balanceCredits.value, 'USD', 1, locale.value, 2)
  if (balanceCnyPerCredit.value === null) {
    return settlementCurrency.value === 'USD'
      ? t('settlementCurrency.baseCredit')
      : `${base} ${t('settlementCurrency.baseCredit')}`
  }
  if (settlementCurrency.value === 'CNY') {
    return `${base} ${t('settlementCurrency.baseCredit')}`
  }
  return t('dashboard.balanceApproxCny', {
    amount: formatSettlementCurrencyAmount(balanceCredits.value, 'CNY', balanceCnyPerCredit.value, locale.value, 2),
  })
})
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const balanceFrozenText = computed(() => t('common.frozenBalance') === 'common.frozenBalance' ? '冻结金额' : t('common.frozenBalance'))
const formattedFrozenBalance = computed(() => formatBalanceAmount(frozenBalance.value, 2))
const balanceFrozenLabel = computed(() => `${balanceFrozenText.value} ${formattedFrozenBalance.value}`)

// 只在标准模式的管理员下显示配置向导入口
const showOnboardingButton = computed(() => {
  return !authStore.isSimpleMode && user.value?.role === 'admin'
})

const userInitials = computed(() => {
  if (!user.value) return ''
  // Prefer username, fallback to email
  if (user.value.username) {
    return user.value.username.substring(0, 2).toUpperCase()
  }
  if (user.value.email) {
    // Get the part before @ and take first 2 chars
    const localPart = user.value.email.split('@')[0]
    return localPart.substring(0, 2).toUpperCase()
  }
  return ''
})

const displayName = computed(() => {
  if (!user.value) return ''
  return user.value.username || user.value.email?.split('@')[0] || ''
})

const pageTitle = computed(() => {
  // For custom pages, use the menu item's label instead of generic "自定义页面"
  if (route.name === 'CustomPage') {
    const id = route.params.id as string
    const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
    const menuItem = publicItems.find((item) => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find((item) => item.id === id) : undefined)
    if (menuItem?.label) return menuItem.label
  }
  const titleKey = route.meta.titleKey as string
  if (titleKey) {
    return t(titleKey)
  }
  return (route.meta.title as string) || ''
})

const pageDescription = computed(() => {
  const descKey = route.meta.descriptionKey as string
  if (descKey) {
    return t(descKey)
  }
  return (route.meta.description as string) || ''
})

function toggleMobileSidebar() {
  appStore.toggleMobileSidebar()
}

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function closeDropdown() {
  dropdownOpen.value = false
}

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

function handleDocsDropdownClick(event: MouseEvent) {
  handleDocsLinkClick(event)
  closeDropdown()
}

async function setHeaderLocale(code: string) {
  if (localeSwitching.value) return
  localeSwitching.value = true
  try {
    await setLocale(code)
    closeDropdown()
  } finally {
    localeSwitching.value = false
  }
}

async function handleLogout() {
  closeDropdown()
  try {
    await authStore.logout()
  } catch (error) {
    // Ignore logout errors - still redirect to login
    console.error('Logout error:', error)
  }
  await router.push('/login')
}

function handleReplayGuide() {
  closeDropdown()
  onboardingStore.replay()
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    closeDropdown()
  }
}

function formatBadgeCount(count: number) {
  return count > 99 ? '99+' : String(count)
}

function refreshTicketSummary(force = false) {
  if (!user.value) return
  ticketStore.fetchUnreadSummary(authStore.canAccessTicketAdmin ? 'admin' : 'user', force)
}

function refreshPaymentConfig(force = false) {
  if (!user.value) return
  paymentStore.fetchConfig(force).catch((error) => {
    console.warn('Failed to load payment config:', error)
  })
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  refreshTicketSummary()
  refreshPaymentConfig()
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})

watch(
  () => [user.value?.id, user.value?.role],
  () => {
    refreshTicketSummary(true)
    refreshPaymentConfig()
  }
)

watch(
  balanceCnyPerCredit,
  (value) => {
    if (value !== null) setSettlementCnyPerCredit(value)
  },
  { immediate: true }
)
</script>

<style scoped>
.app-header {
  background: color-mix(in srgb, var(--apple-surface) 92%, transparent);
  border-color: var(--apple-border);
  color: var(--apple-text);
}

.header-icon-button,
.header-user-button {
  color: var(--apple-muted);
}

.header-icon-button:hover,
.header-user-button:hover {
  background: var(--apple-hover);
  color: var(--apple-text);
}

.header-action-link,
.header-balance {
  background: var(--apple-surface);
  border-color: var(--apple-border);
  color: var(--apple-text);
  box-shadow: none;
}

.header-action-link:hover {
  background: var(--apple-surface-elevated);
  border-color: var(--apple-border);
  color: var(--apple-blue);
}

.header-balance {
  background: var(--apple-surface-elevated);
}

.header-avatar {
  background: var(--apple-blue);
}

.header-segmented {
  background: var(--apple-surface-elevated);
}

.header-segmented-button {
  min-width: 0;
  color: var(--apple-muted);
}

.header-segmented-button:hover:not(:disabled) {
  color: var(--apple-text);
}

.header-segmented-button-active {
  background: var(--apple-surface);
  color: var(--apple-blue);
  box-shadow: var(--apple-shadow-sm);
}

.header-danger-item {
  color: var(--apple-danger);
}

.header-danger-item:hover {
  background: color-mix(in srgb, var(--apple-danger) 10%, transparent);
  color: var(--apple-danger);
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
</style>
