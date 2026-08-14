<template>
  <div class="gateway-auth-shell dark min-h-[100dvh] overflow-x-hidden bg-[#070a10] px-4 py-6 text-white sm:px-6 sm:py-10">
    <div class="mx-auto flex min-h-[calc(100dvh-3rem)] w-full max-w-6xl items-center justify-center">
      <div class="grid w-full items-center gap-8 lg:grid-cols-[minmax(0,1fr)_430px] lg:gap-12 xl:gap-16">
        <section class="hidden min-w-0 lg:block">
          <div class="max-w-2xl">
            <div class="mb-6 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs font-medium text-white/70">
              <span class="h-1.5 w-1.5 rounded-full bg-[#4dd4e6] shadow-[0_0_16px_rgba(77,212,230,0.75)]"></span>
              <span>AI Gateway · Multi-model API</span>
            </div>
            <h1 class="max-w-xl text-4xl font-semibold leading-tight tracking-normal text-white xl:text-5xl">
              {{ t('auth.gateway.title') }}
            </h1>
            <p class="mt-5 max-w-xl text-base leading-7 text-white/62">
              {{ t('auth.gateway.subtitle') }}
            </p>

            <div class="mt-8 grid max-w-xl grid-cols-2 gap-3">
              <div
                v-for="item in gatewayHighlights"
                :key="item.label"
                class="rounded-xl border border-white/10 bg-white/[0.045] p-4 shadow-[0_18px_44px_rgba(0,0,0,0.22)]"
              >
                <div class="text-sm font-semibold text-white">{{ item.label }}</div>
                <div class="mt-1 text-xs leading-5 text-white/50">{{ item.detail }}</div>
              </div>
            </div>

            <div class="mt-8 max-w-xl rounded-2xl border border-white/10 bg-[#0d1219]/80 shadow-[0_24px_70px_rgba(0,0,0,0.38)]">
              <div class="flex h-10 items-center gap-2 border-b border-white/8 px-4">
                <span class="h-2.5 w-2.5 rounded-full bg-[#e8a87c]"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-[#e0b44c]"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-[#4dd4e6]"></span>
                <span class="ml-2 text-xs text-white/38">quickstart.sh</span>
              </div>
              <pre class="overflow-hidden px-4 py-4 text-xs leading-6 text-white/72"><code>curl {{ apiBaseUrl }}/v1/chat/completions \
  -H "Authorization: Bearer $WEGOO_API_KEY" \
  -d '{"model":"gpt-5.5","stream":true}'</code></pre>
            </div>
          </div>
        </section>

        <div class="mx-auto w-full max-w-[430px] lg:mx-0">
          <div class="mb-5 text-center sm:mb-6">
            <template v-if="settingsLoaded || siteName">
              <a
                href="/home"
                class="inline-flex items-center gap-3 rounded-full border border-white/10 bg-white/[0.045] px-3 py-2 text-left transition hover:border-white/18 hover:bg-white/[0.07]"
              >
                <span class="inline-flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-full border border-white/10 bg-white/[0.06]">
                  <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
                </span>
                <span class="min-w-0">
                  <span class="block truncate text-sm font-semibold text-white">{{ siteName }}</span>
                  <span class="block text-[11px] font-medium uppercase tracking-[0.18em] text-[#4dd4e6]">Gateway Console</span>
                </span>
              </a>
            </template>
          </div>

          <div class="gateway-auth-card rounded-2xl border border-white/10 bg-[#0d1219]/82 p-5 shadow-[0_24px_70px_rgba(0,0,0,0.42)] backdrop-blur-xl sm:p-7">
            <slot />
          </div>

          <div class="mt-5 text-center text-sm text-white/55">
            <slot name="footer" />
          </div>

          <div class="mt-5 break-words text-center text-xs leading-5 text-white/34 sm:mt-7">
            &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import { DEFAULT_SITE_NAME } from '@/utils/branding'

defineOptions({ name: 'AuthLayout' })

const appStore = useAppStore()
const { t } = useI18n()

const siteName = computed(() => appStore.siteName || DEFAULT_SITE_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())
const apiBaseUrl = computed(() => appStore.apiBaseUrl || 'https://api.wegoo.site')

const gatewayHighlights = computed(() => [
  {
    label: t('auth.gateway.highlights.key.label'),
    detail: t('auth.gateway.highlights.key.detail')
  },
  {
    label: t('auth.gateway.highlights.routing.label'),
    detail: t('auth.gateway.highlights.routing.detail')
  },
  {
    label: t('auth.gateway.highlights.billing.label'),
    detail: t('auth.gateway.highlights.billing.detail')
  },
  {
    label: t('auth.gateway.highlights.status.label'),
    detail: t('auth.gateway.highlights.status.detail')
  }
])

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.gateway-auth-shell {
  padding-top: max(1.5rem, env(safe-area-inset-top));
  padding-bottom: max(1.5rem, env(safe-area-inset-bottom));
  background:
    radial-gradient(circle at 20% 0%, rgba(77, 212, 230, 0.12), transparent 34rem),
    radial-gradient(circle at 88% 10%, rgba(212, 168, 92, 0.10), transparent 28rem),
    linear-gradient(180deg, #070a10 0%, #090d13 52%, #070a10 100%);
}

.gateway-auth-shell::before {
  pointer-events: none;
  position: fixed;
  inset: 0;
  content: '';
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.68), transparent 70%);
}

.gateway-auth-card {
  position: relative;
}

.gateway-auth-card::before {
  pointer-events: none;
  position: absolute;
  inset: 0;
  border-radius: inherit;
  content: '';
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.06), transparent 42%);
}

:deep(.input),
:deep(.btn) {
  border-radius: 0.625rem;
}

:deep(.input) {
  min-height: 2.875rem;
  border-color: rgba(255, 255, 255, 0.10);
  background: rgba(255, 255, 255, 0.045);
  color: rgba(255, 255, 255, 0.95);
  font-size: 1rem;
}

:deep(.input::placeholder) {
  color: rgba(255, 255, 255, 0.34);
}

:deep(.input:focus) {
  border-color: rgba(77, 212, 230, 0.72);
  box-shadow: 0 0 0 3px rgba(77, 212, 230, 0.13);
}

:deep(.btn) {
  min-height: 2.875rem;
}

:deep(.btn-primary) {
  border-color: rgba(77, 212, 230, 0.34);
  background: linear-gradient(135deg, #4dd4e6, #78e6f4);
  color: #071014;
  box-shadow: 0 14px 34px rgba(77, 212, 230, 0.18);
}

:deep(.btn-secondary) {
  border-color: rgba(255, 255, 255, 0.10);
  background: rgba(255, 255, 255, 0.045);
  color: rgba(255, 255, 255, 0.88);
}

:deep(.btn-secondary:hover) {
  border-color: rgba(255, 255, 255, 0.18);
  background: rgba(255, 255, 255, 0.075);
}

:deep(.input-label),
:deep(.input-hint) {
  line-height: 1.45;
}

:deep(.input-label) {
  color: rgba(255, 255, 255, 0.72);
}

:deep(.input-hint) {
  margin-top: 0.5rem;
  color: rgba(255, 255, 255, 0.45);
}

:deep(a) {
  color: #4dd4e6;
}

:deep(a:hover) {
  color: #78e6f4;
}

@media (min-width: 640px) {
  :deep(.input) {
    font-size: 0.875rem;
  }
}
</style>
