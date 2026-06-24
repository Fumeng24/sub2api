<template>
  <div class="apple-auth-shell flex min-h-[100dvh] items-center justify-center overflow-x-hidden bg-[var(--apple-bg)] px-3 py-6 text-[var(--apple-text)] sm:px-6 sm:py-10">
    <!-- Content Container -->
    <div class="w-full max-w-[430px]">
      <!-- Logo/Brand -->
      <div class="mb-6 text-center sm:mb-8">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded || siteName">
          <div
            class="mb-3 inline-flex h-11 w-11 items-center justify-center overflow-hidden rounded-lg bg-[var(--apple-surface)] ring-1 ring-[color:var(--apple-border)]"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="break-words text-lg font-semibold tracking-normal text-[var(--apple-text)] sm:text-xl">
            {{ siteName }}
          </h1>
        </template>
      </div>

      <!-- Card Container -->
      <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm sm:p-7">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-6 break-words text-center text-xs leading-5 text-[var(--apple-muted-2)] sm:mt-8">
        &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()
const { t } = useI18n()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.apple-auth-shell {
  padding-top: max(1.5rem, env(safe-area-inset-top));
  padding-bottom: max(1.5rem, env(safe-area-inset-bottom));
}

:deep(.input),
:deep(.btn) {
  border-radius: 0.5rem;
}

:deep(.input) {
  min-height: 2.875rem;
  font-size: 1rem;
}

:deep(.btn) {
  min-height: 2.875rem;
}

:deep(.input-label),
:deep(.input-hint) {
  line-height: 1.45;
}

:deep(.input-hint) {
  margin-top: 0.5rem;
}

@media (min-width: 640px) {
  :deep(.input) {
    font-size: 0.875rem;
  }
}
</style>
