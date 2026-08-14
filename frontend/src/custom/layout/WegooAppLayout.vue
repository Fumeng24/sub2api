<template>
  <div class="app-layout gateway-console apple-runtime min-h-screen text-[var(--apple-text)]">
    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen overflow-x-hidden transition-[margin] duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="mx-auto w-full max-w-[1680px] min-w-0 px-3 py-4 sm:px-4 md:px-6 lg:px-8 lg:py-6">
        <LowBalanceRetentionBanner class="mb-4" />
        <GroupRateDiscountNotice v-if="showGroupDiscountNotice" class="mb-4" />
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/custom/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/custom/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from '@/custom/layout/WegooAppSidebar.vue'
import AppHeader from '@/custom/layout/WegooAppHeader.vue'
import LowBalanceRetentionBanner from '@/custom/common/LowBalanceRetentionBanner.vue'
import GroupRateDiscountNotice from '@/custom/common/GroupRateDiscountNotice.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const route = useRoute()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const showGroupDiscountNotice = computed(() => authStore.isAuthenticated && route.meta.requiresAdmin !== true)

const { replayTour } = useOnboardingTour({
  storageKey: 'admin_guide',
  autoStart: false
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style scoped>
.app-layout {
  background: var(--apple-bg);
  color: var(--apple-text);
}

.gateway-console {
  background: var(--apple-bg);
  color: var(--apple-text);
}

.gateway-console main {
  position: relative;
}

:global(.dark) .gateway-console {
  --apple-bg: var(--gw-bg);
  --apple-surface: var(--gw-panel);
  --apple-surface-solid: var(--gw-panel-solid);
  --apple-surface-elevated: var(--gw-panel-elevated);
  --apple-surface-elevated-solid: #131926;
  --apple-text: var(--gw-text);
  --apple-muted: var(--gw-text-2);
  --apple-muted-2: var(--gw-text-3);
  --apple-border: var(--gw-border);
  --apple-border-soft: var(--gw-border-soft);
  --apple-blue: var(--gw-accent);
  --apple-blue-hover: var(--gw-accent-hover);
  --apple-danger: var(--gw-danger);
  --apple-success: var(--gw-success);
  --apple-warning: var(--gw-warning);
  --apple-radius: var(--gw-radius-xs);
  --apple-hover: rgb(255 255 255 / 7%);
  --apple-focus-ring: var(--gw-focus-ring);
  --apple-shadow-sm: var(--gw-shadow-sm);
  --apple-shadow-md: var(--gw-shadow-md);
  background:
    linear-gradient(rgb(255 255 255 / 3%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(255 255 255 / 3%) 1px, transparent 1px),
    linear-gradient(180deg, var(--gw-bg-soft) 0%, var(--gw-bg) 42%, #05070d 100%);
  background-size: 44px 44px, 44px 44px, auto;
}
</style>
