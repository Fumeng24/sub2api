<template>
  <div
    class="gateway-not-found dark flex min-h-screen items-center justify-center overflow-hidden px-4 py-10 text-white sm:px-6"
  >
    <div class="gateway-not-found__grid relative z-10 mx-auto grid w-full max-w-5xl items-center gap-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-12">
      <section class="min-w-0 text-center lg:text-left">
        <p class="gateway-not-found__badge mx-auto lg:mx-0">
          <span></span>
          AI Gateway · Route check
        </p>
        <h1 class="mt-6 text-5xl font-semibold leading-none tracking-normal text-white sm:text-7xl">
          404
        </h1>
        <h2 class="mt-5 text-2xl font-semibold tracking-normal text-white sm:text-3xl">
          {{ t('errors.pageNotFound') }}
        </h2>
        <p class="mx-auto mt-4 max-w-xl text-sm leading-6 text-white/60 sm:text-base sm:leading-7 lg:mx-0">
          {{ t('errors.pageNotFoundDescription') }}
        </p>

        <div class="mt-8 flex flex-col justify-center gap-3 sm:flex-row lg:justify-start">
          <router-link to="/home" class="gateway-not-found__primary">
            <Icon name="home" size="md" />
            {{ t('errors.goHome') }}
          </router-link>
          <button type="button" @click="goBack" class="gateway-not-found__secondary">
            <Icon name="arrowLeft" size="md" />
            {{ t('errors.goBack') }}
          </button>
        </div>

        <div class="mt-8 flex flex-wrap justify-center gap-2 lg:justify-start" :aria-label="t('errors.pageNotFound')">
          <router-link v-for="item in suggestions" :key="item.to" :to="item.to" class="gateway-not-found__chip">
            {{ item.label }}
          </router-link>
        </div>
      </section>

      <aside class="gateway-not-found__panel mx-auto w-full max-w-sm rounded-2xl border border-white/10 bg-[#0d1219]/82 p-5 shadow-[0_24px_70px_rgba(0,0,0,0.42)] backdrop-blur-xl sm:p-6">
        <div class="flex h-10 items-center gap-2 border-b border-white/8 pb-4">
          <span class="h-2.5 w-2.5 rounded-full bg-[#e8a87c]"></span>
          <span class="h-2.5 w-2.5 rounded-full bg-[#e0b44c]"></span>
          <span class="h-2.5 w-2.5 rounded-full bg-[#4dd4e6]"></span>
          <span class="ml-2 truncate text-xs text-white/38">route.inspect</span>
        </div>
        <div class="mt-5 space-y-3 font-mono text-xs leading-6 text-white/68">
          <p><span class="text-[#4dd4e6]">$</span> GET {{ currentPath }}</p>
          <p><span class="text-white/38">status</span> <strong class="text-[#e0b44c]">404 not_found</strong></p>
          <p><span class="text-white/38">next</span> /home · /pricing · /docs</p>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const currentPath = computed(() => route.fullPath || '/')
const suggestions = computed(() => [
  { to: '/pricing', label: 'Models' },
  { to: '/docs', label: 'Docs' },
  { to: '/status', label: 'Status' },
  { to: '/enterprise', label: 'Enterprise' },
])

function goBack(): void {
  if (window.history.length > 1) {
    router.back()
    return
  }
  router.push('/home')
}
</script>

<style scoped>
.gateway-not-found {
  position: relative;
  background:
    radial-gradient(circle at 18% 8%, rgba(77, 212, 230, 0.13), transparent 32rem),
    radial-gradient(circle at 82% 18%, rgba(201, 161, 79, 0.11), transparent 30rem),
    linear-gradient(180deg, #070a10 0%, #090d13 52%, #070a10 100%);
}

.gateway-not-found::before {
  pointer-events: none;
  position: absolute;
  inset: 0;
  content: '';
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.025) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.72), transparent 78%);
}

.gateway-not-found__badge,
.gateway-not-found__chip,
.gateway-not-found__primary,
.gateway-not-found__secondary {
  display: inline-flex;
  align-items: center;
}

.gateway-not-found__badge {
  width: fit-content;
  gap: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.10);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.045);
  padding: 0.375rem 0.75rem;
  color: rgba(255, 255, 255, 0.68);
  font-size: 0.75rem;
  font-weight: 600;
}

.gateway-not-found__badge span {
  width: 0.375rem;
  height: 0.375rem;
  border-radius: 999px;
  background: #4dd4e6;
  box-shadow: 0 0 16px rgba(77, 212, 230, 0.75);
}

.gateway-not-found__primary,
.gateway-not-found__secondary {
  min-height: 2.875rem;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 0.75rem;
  padding: 0 1.125rem;
  font-size: 0.875rem;
  font-weight: 700;
  transition: border-color 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.gateway-not-found__primary {
  border: 1px solid rgba(77, 212, 230, 0.34);
  background: linear-gradient(135deg, #4dd4e6, #78e6f4);
  color: #071014;
  box-shadow: 0 14px 34px rgba(77, 212, 230, 0.18);
}

.gateway-not-found__secondary,
.gateway-not-found__chip {
  border: 1px solid rgba(255, 255, 255, 0.10);
  background: rgba(255, 255, 255, 0.045);
  color: rgba(255, 255, 255, 0.78);
}

.gateway-not-found__secondary:hover,
.gateway-not-found__chip:hover {
  border-color: rgba(255, 255, 255, 0.18);
  background: rgba(255, 255, 255, 0.075);
  color: rgba(255, 255, 255, 0.94);
}

.gateway-not-found__chip {
  min-height: 2rem;
  border-radius: 999px;
  padding: 0 0.75rem;
  font-size: 0.75rem;
  font-weight: 700;
}

.gateway-not-found__panel {
  position: relative;
}

.gateway-not-found__panel::before {
  pointer-events: none;
  position: absolute;
  inset: 0;
  border-radius: inherit;
  content: '';
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.06), transparent 42%);
}
</style>
