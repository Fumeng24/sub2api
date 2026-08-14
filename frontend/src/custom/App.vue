<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { onMounted, onBeforeUnmount, watch } from 'vue'
import Toast from '@/custom/common/WegooToast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import AdminComplianceDialog from '@/components/admin/AdminComplianceDialog.vue'
import { resolveRouteDocumentTitle } from '@/router/title'
import AnnouncementPopup from '@/custom/common/WegooAnnouncementPopup.vue'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore, useAdminComplianceStore, useAdminSettingsStore } from '@/stores'
import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'
import { getSetupStatus } from '@/api/setup'
import { LOCALE_CHANGED_EVENT } from '@/i18n'
import { applyRouteSeo } from '@/custom/utils/seo'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const subscriptionCapabilityStore = useSubscriptionCapabilityStore()
const announcementStore = useAnnouncementStore()
const adminComplianceStore = useAdminComplianceStore()
const adminSettingsStore = useAdminSettingsStore()
const paymentCheckoutStore = usePaymentCheckoutStore()

function updateDocumentTitle() {
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

function currentRouteTitle() {
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  return resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

/**
 * Update favicon dynamically
 * @param logoUrl - URL of the logo to use as favicon
 */
function updateFavicon(logoUrl: string) {
  // Find existing favicon link or create new one
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = logoUrl.endsWith('.svg') ? 'image/svg+xml' : 'image/x-icon'
  link.href = logoUrl
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

watch(
  [
    () => route.fullPath,
    () => route.meta.title,
    () => route.meta.titleKey,
    () => appStore.siteName,
    () => appStore.cachedPublicSettings?.custom_menu_items,
    () => authStore.isAdmin,
    () => adminSettingsStore.customMenuItems,
  ],
  updateDocumentTitle,
  { deep: true }
)

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
    appStore.fetchPublicSettings(true)
  }
}

function onAdminComplianceRequired(event: Event) {
  const detail = (event as CustomEvent<Record<string, string>>).detail || {}
  adminComplianceStore.requireAcknowledgement(detail)
}

function refreshCurrentRouteSeo() {
  applyRouteSeo(route, {
    siteName: appStore.siteName,
    title: currentRouteTitle()
  })
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    if (isAuthenticated) {
      if (authStore.isAdmin) {
        adminComplianceStore.fetchStatus().catch((error) => {
          console.error('Failed to fetch admin compliance status:', error)
        })
      }

      // User logged in: preload subscriptions and start polling
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionCapabilityStore.fetchSubscriptionCapability().catch((error) => {
        console.error('Failed to preload subscription capability:', error)
      })
      subscriptionStore.startPolling()
      subscriptionCapabilityStore.startPolling()

      // Announcements: new login vs page refresh restore
      if (oldValue === false) {
        // New login: delay 3s then force fetch
        setTimeout(() => announcementStore.fetchAnnouncements(true), 3000)
      } else {
        // Page refresh restore (oldValue was undefined)
        announcementStore.fetchAnnouncements()
      }

      // Register visibility change listener
      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      // User logged out: clear data and stop polling
      subscriptionStore.clear()
      subscriptionCapabilityStore.clear()
      paymentCheckoutStore.clear()
      announcementStore.reset()
      adminComplianceStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  },
  { immediate: true }
)

// Route change trigger (throttled by store)
router.afterEach(() => {
  if (authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('admin-compliance-required', onAdminComplianceRequired)
  window.removeEventListener(LOCALE_CHANGED_EVENT, refreshCurrentRouteSeo)
})

onMounted(async () => {
  window.addEventListener('admin-compliance-required', onAdminComplianceRequired)
  window.addEventListener(LOCALE_CHANGED_EVENT, refreshCurrentRouteSeo)

  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()

  // Re-resolve document title and SEO tags now that siteName is available.
  refreshCurrentRouteSeo()
})
</script>

<template>
  <div class="apple-runtime min-h-screen bg-[var(--apple-bg)] text-[var(--apple-text)]">
    <NavigationProgress />
    <RouterView />
    <Toast />
    <AnnouncementPopup />
    <AdminComplianceDialog />
  </div>
</template>
