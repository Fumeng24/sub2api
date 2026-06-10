<template>
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- Background -->
    <div
      class="absolute inset-0 bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
    ></div>

    <!-- Decorative Elements -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <!-- Gradient Orbs -->
      <div
        class="absolute -right-40 -top-40 h-80 w-80 rounded-full bg-primary-400/20 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-80 w-80 rounded-full bg-primary-500/15 blur-3xl"
      ></div>
      <div
        class="absolute left-1/2 top-1/2 h-96 w-96 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary-300/10 blur-3xl"
      ></div>

      <!-- Grid Pattern -->
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(20,184,166,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(20,184,166,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl shadow-lg shadow-primary-500/30"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="text-gradient mb-2 text-3xl font-bold">
            {{ siteName }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Value Props -->
      <div class="mb-4 grid grid-cols-3 gap-2">
        <div
          class="rounded-2xl border border-emerald-200/70 bg-white/75 px-2.5 py-2 text-center shadow-sm backdrop-blur-sm dark:border-emerald-900/40 dark:bg-dark-800/60"
        >
          <Icon name="gift" size="sm" class="mx-auto mb-1 text-emerald-600 dark:text-emerald-300" />
          <p class="text-[11px] font-semibold leading-4 text-gray-700 dark:text-dark-200">
            {{ t('auth.valueProps.trial') }}
          </p>
        </div>
        <div
          class="rounded-2xl border border-sky-200/70 bg-white/75 px-2.5 py-2 text-center shadow-sm backdrop-blur-sm dark:border-sky-900/40 dark:bg-dark-800/60"
        >
          <Icon name="dollar" size="sm" class="mx-auto mb-1 text-sky-600 dark:text-sky-300" />
          <p class="text-[11px] font-semibold leading-4 text-gray-700 dark:text-dark-200">
            {{ t('auth.valueProps.discount') }}
          </p>
        </div>
        <div
          class="rounded-2xl border border-primary-200/70 bg-white/75 px-2.5 py-2 text-center shadow-sm backdrop-blur-sm dark:border-primary-900/40 dark:bg-dark-800/60"
        >
          <Icon name="shield" size="sm" class="mx-auto mb-1 text-primary-600 dark:text-primary-300" />
          <p class="text-[11px] font-semibold leading-4 text-gray-700 dark:text-dark-200">
            {{ t('auth.valueProps.stability') }}
          </p>
        </div>
      </div>

      <!-- Card Container -->
      <div class="card-glass rounded-2xl p-8 shadow-glass">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.text-gradient {
  @apply bg-gradient-to-r from-primary-600 to-primary-500 bg-clip-text text-transparent;
}
</style>
