<template>
  <div class="flex min-h-screen items-center justify-center bg-white px-4 py-10 text-gray-950 dark:bg-dark-950 dark:text-white sm:px-6">
    <!-- Content Container -->
    <div class="w-full max-w-[420px]">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded || siteName">
          <div
            class="mb-3 inline-flex h-11 w-11 items-center justify-center overflow-hidden rounded-lg bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-800"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="text-xl font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ siteName }}
          </h1>
        </template>
      </div>

      <!-- Card Container -->
      <div class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-800 dark:bg-dark-900/80 sm:p-7">
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
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
:deep(.input),
:deep(.btn) {
  border-radius: 0.5rem;
}

:deep(.input) {
  min-height: 2.75rem;
}
</style>
