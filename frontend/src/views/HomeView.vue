<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="custom-home-content min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else class="custom-home-content__body" v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="home-default-shell relative flex min-h-screen w-full max-w-full flex-col overflow-x-hidden bg-gray-50 text-gray-950 dark:bg-dark-950 dark:text-white"
  >
    <header
      class="sticky top-0 z-20 w-full max-w-full border-b border-gray-200/70 bg-white/90 px-3 py-4 backdrop-blur dark:border-dark-800/70 dark:bg-dark-950/85 sm:px-6"
    >
      <nav class="mx-auto flex min-w-0 max-w-6xl items-center justify-between gap-3">
        <div class="flex min-w-0 items-center gap-3">
          <div
            class="h-9 w-9 overflow-hidden rounded-lg bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-800"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="hidden text-sm font-semibold text-gray-900 dark:text-white sm:inline">
            {{ siteName }}
          </span>
        </div>

        <div class="flex shrink-0 items-center gap-1.5 sm:gap-3">
          <LocaleSwitcher />

          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="rounded-md p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
            @click="handleDocsLinkClick"
          >
            <Icon name="book" size="md" />
          </a>

          <button
            class="rounded-md p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-lg bg-gray-950 py-1 pl-1 pr-2.5 transition-colors hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-dark-100"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-md bg-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium text-white dark:text-gray-950">{{ t('home.dashboard') }}</span>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-lg bg-gray-950 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-dark-100"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section
        class="mx-auto flex min-h-[calc(100vh-73px)] w-full max-w-6xl flex-col items-center justify-center px-4 py-14 text-center sm:px-6 sm:py-16 lg:py-20"
      >
        <p class="mb-5 text-sm font-semibold text-gray-500 dark:text-dark-300">
          {{ t('home.heroKicker') }}
        </p>
        <h1
          class="max-w-5xl break-words text-4xl font-semibold leading-[1.08] text-gray-950 dark:text-white sm:text-5xl md:text-7xl md:leading-[1.05]"
        >
          {{ t('home.heroTitle') }}
        </h1>
        <p class="mt-6 max-w-3xl text-lg leading-8 text-gray-600 dark:text-dark-300 md:text-xl md:leading-9">
          {{ t('home.heroDescription') }}
        </p>

        <div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="inline-flex min-h-11 items-center justify-center rounded-lg bg-gray-950 px-6 text-sm font-semibold text-white transition-colors hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-dark-100"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
          </router-link>
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-gray-300 bg-white px-5 text-sm font-semibold text-gray-800 transition-colors hover:border-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-100 dark:hover:border-dark-500"
            @click="handleDocsLinkClick"
          >
            {{ t('home.docs') }}
            <Icon name="arrowRight" size="sm" />
          </a>
        </div>

        <p class="mt-8 text-sm font-medium text-gray-500 dark:text-dark-400">
          {{ t('home.heroModelLine') }}
        </p>

        <dl class="mt-10 grid w-full max-w-5xl border-y border-gray-200 dark:border-dark-800 sm:grid-cols-4">
          <div
            v-for="(item, index) in heroProofs"
            :key="item.value"
            class="px-4 py-5"
            :class="[
              index > 0 ? 'border-t border-gray-200 dark:border-dark-800 sm:border-t-0 sm:border-l' : ''
            ]"
          >
            <dt class="text-2xl font-semibold text-gray-950 dark:text-white">
              {{ item.value }}
            </dt>
            <dd class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-400">
              {{ item.label }}
            </dd>
          </div>
        </dl>
      </section>

      <section class="border-t border-gray-200 bg-white/80 dark:border-dark-800 dark:bg-dark-900/30">
        <div class="mx-auto grid max-w-6xl gap-10 px-4 py-16 sm:px-6 lg:grid-cols-[0.9fr_1.1fr] lg:py-20">
          <div>
            <p class="text-sm font-semibold text-primary-700 dark:text-primary-300">
              {{ t('home.pain.kicker') }}
            </p>
            <h2 class="mt-3 text-4xl font-semibold leading-tight text-gray-950 dark:text-white md:text-5xl">
              {{ t('home.pain.title') }}
            </h2>
            <p class="mt-5 text-base leading-8 text-gray-600 dark:text-dark-300">
              {{ t('home.pain.description') }}
            </p>
          </div>

          <div class="border-y border-gray-200 dark:border-dark-800">
            <div
              v-for="item in painItems"
              :key="item.title"
              class="grid gap-3 border-b border-gray-200 py-6 last:border-b-0 dark:border-dark-800 sm:grid-cols-[10rem_1fr]"
            >
              <h3 class="text-base font-semibold text-gray-950 dark:text-white">
                {{ item.title }}
              </h3>
              <p class="text-sm leading-7 text-gray-600 dark:text-dark-300">
                {{ item.description }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="bg-gray-950 text-white dark:bg-black">
        <div class="mx-auto max-w-6xl px-4 py-16 sm:px-6 lg:py-20">
          <div class="grid gap-8 lg:grid-cols-[0.85fr_1.15fr]">
            <div>
              <p class="text-sm font-semibold text-gray-400">
                {{ t('home.promise.kicker') }}
              </p>
              <h2 class="mt-3 text-4xl font-semibold leading-tight md:text-5xl">
                {{ t('home.promise.title') }}
              </h2>
              <p class="mt-5 text-base leading-8 text-gray-300">
                {{ t('home.promise.description') }}
              </p>
            </div>

            <div class="border-y border-white/15">
              <div
                v-for="item in promiseItems"
                :key="item.title"
                class="flex gap-4 border-b border-white/10 py-5 last:border-b-0"
              >
                <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white text-gray-950">
                  <Icon :name="item.icon" size="sm" :stroke-width="2" />
                </div>
                <div>
                  <h3 class="text-base font-semibold text-white">
                    {{ item.title }}
                  </h3>
                  <p class="mt-1 text-sm leading-6 text-gray-300">
                    {{ item.description }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="border-b border-gray-200 bg-gray-50 dark:border-dark-800 dark:bg-dark-950">
        <div class="mx-auto grid max-w-6xl gap-10 px-4 py-16 sm:px-6 lg:grid-cols-[1fr_1fr] lg:py-20">
          <div>
            <p class="text-sm font-semibold text-primary-700 dark:text-primary-300">
              {{ t('home.cost.kicker') }}
            </p>
            <h2 class="mt-3 text-4xl font-semibold leading-tight text-gray-950 dark:text-white md:text-5xl">
              {{ t('home.cost.title') }}
            </h2>
            <p class="mt-5 text-base leading-8 text-gray-600 dark:text-dark-300">
              {{ t('home.cost.description') }}
            </p>
          </div>

          <dl class="grid content-start gap-4 sm:grid-cols-3 lg:grid-cols-1">
            <div
              v-for="item in costFacts"
              :key="item.value"
              class="border-t border-gray-200 pt-4 dark:border-dark-800"
            >
              <dt class="text-2xl font-semibold text-gray-950 dark:text-white">
                {{ item.value }}
              </dt>
              <dd class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ item.label }}
              </dd>
            </div>
          </dl>
        </div>
      </section>

      <section class="bg-white dark:bg-dark-900/30">
        <div class="mx-auto max-w-6xl px-4 py-16 sm:px-6 lg:py-20">
          <div class="max-w-3xl">
            <p class="text-sm font-semibold text-primary-700 dark:text-primary-300">
              {{ t('home.useCases.kicker') }}
            </p>
            <h2 class="mt-3 text-4xl font-semibold leading-tight text-gray-950 dark:text-white md:text-5xl">
              {{ t('home.useCases.title') }}
            </h2>
            <p class="mt-5 text-base leading-8 text-gray-600 dark:text-dark-300">
              {{ t('home.useCases.description') }}
            </p>
          </div>

          <div class="mt-10 grid gap-x-8 gap-y-8 md:grid-cols-2">
            <div
              v-for="item in useCases"
              :key="item.title"
              class="border-t border-gray-200 pt-5 dark:border-dark-800"
            >
              <div class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-gray-950 text-white dark:bg-white dark:text-gray-950">
                <Icon :name="item.icon" size="sm" :stroke-width="2" />
              </div>
              <h3 class="text-lg font-semibold text-gray-950 dark:text-white">
                {{ item.title }}
              </h3>
              <p class="mt-2 text-sm leading-7 text-gray-600 dark:text-dark-300">
                {{ item.description }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="border-t border-gray-200 bg-gray-50 dark:border-dark-800 dark:bg-dark-950">
        <div class="mx-auto max-w-6xl px-4 py-14 sm:px-6">
          <div class="grid gap-8 lg:grid-cols-[0.8fr_1.2fr]">
            <div>
              <h2 class="text-3xl font-semibold leading-tight text-gray-950 dark:text-white">
                {{ t('home.providers.title') }}
              </h2>
              <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-300">
                {{ t('home.providers.description') }}
              </p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div
                v-for="item in providerModels"
                :key="item.name"
                class="flex items-center justify-between border-b border-gray-200 py-3 dark:border-dark-800"
              >
                <span class="text-base font-semibold text-gray-950 dark:text-white">{{ item.name }}</span>
                <span class="text-xs font-semibold text-primary-700 dark:text-primary-300">
                  {{ item.status }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="bg-white dark:bg-dark-900/30">
        <div class="mx-auto max-w-4xl px-4 py-16 text-center sm:px-6 lg:py-20">
          <h2 class="text-4xl font-semibold leading-tight text-gray-950 dark:text-white md:text-5xl">
            {{ t('home.cta.title') }}
          </h2>
          <p class="mx-auto mt-5 max-w-2xl text-base leading-8 text-gray-600 dark:text-dark-300">
            {{ t('home.cta.description') }}
          </p>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="mt-8 inline-flex min-h-11 items-center justify-center rounded-lg bg-gray-950 px-6 text-sm font-semibold text-white transition-colors hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-dark-100"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
          </router-link>
        </div>
      </section>
    </main>

    <footer class="border-t border-gray-200/70 px-6 py-8 dark:border-dark-800/70">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="text-sm text-gray-500 transition-colors hover:text-gray-800 dark:text-dark-400 dark:hover:text-white"
            @click="handleDocsLinkClick"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-800 dark:text-dark-400 dark:hover:text-white"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore, useAuthStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/utils/docsLink'

const { t } = useI18n()
const router = useRouter()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const rawDocUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const docsLink = computed(() => resolveDocsLink(rawDocUrl.value, appStore.cachedPublicSettings?.custom_menu_items ?? []))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

type HomeIconName = InstanceType<typeof Icon>['$props']['name']

const heroProofs = computed(() => [
  {
    value: t('home.heroProofs.official.value'),
    label: t('home.heroProofs.official.label')
  },
  {
    value: t('home.heroProofs.unwatered.value'),
    label: t('home.heroProofs.unwatered.label')
  },
  {
    value: t('home.heroProofs.privacy.value'),
    label: t('home.heroProofs.privacy.label')
  },
  {
    value: t('home.heroProofs.billing.value'),
    label: t('home.heroProofs.billing.label')
  }
])

const painItems = computed(() => [
  {
    title: t('home.pain.items.watered.title'),
    description: t('home.pain.items.watered.description')
  },
  {
    title: t('home.pain.items.leakage.title'),
    description: t('home.pain.items.leakage.description')
  },
  {
    title: t('home.pain.items.billing.title'),
    description: t('home.pain.items.billing.description')
  },
  {
    title: t('home.pain.items.stability.title'),
    description: t('home.pain.items.stability.description')
  }
])

const promiseItems = computed<Array<{
  title: string
  description: string
  icon: HomeIconName
}>>(() => [
  {
    title: t('home.promise.items.official.title'),
    description: t('home.promise.items.official.description'),
    icon: 'badge'
  },
  {
    title: t('home.promise.items.unwatered.title'),
    description: t('home.promise.items.unwatered.description'),
    icon: 'checkCircle'
  },
  {
    title: t('home.promise.items.noChatLog.title'),
    description: t('home.promise.items.noChatLog.description'),
    icon: 'chat'
  },
  {
    title: t('home.promise.items.noLeak.title'),
    description: t('home.promise.items.noLeak.description'),
    icon: 'lock'
  },
  {
    title: t('home.promise.items.stability.title'),
    description: t('home.promise.items.stability.description'),
    icon: 'shield'
  },
  {
    title: t('home.promise.items.compensation.title'),
    description: t('home.promise.items.compensation.description'),
    icon: 'dollar'
  }
])

const costFacts = computed(() => [
  {
    value: t('home.cost.facts.traceable.value'),
    label: t('home.cost.facts.traceable.label')
  },
  {
    value: t('home.cost.facts.official.value'),
    label: t('home.cost.facts.official.label')
  },
  {
    value: t('home.cost.facts.compensation.value'),
    label: t('home.cost.facts.compensation.label')
  }
])

const useCases = computed<Array<{
  title: string
  description: string
  icon: HomeIconName
}>>(() => [
  {
    title: t('home.useCases.coding.title'),
    description: t('home.useCases.coding.description'),
    icon: 'terminal'
  },
  {
    title: t('home.useCases.writing.title'),
    description: t('home.useCases.writing.description'),
    icon: 'document'
  },
  {
    title: t('home.useCases.automation.title'),
    description: t('home.useCases.automation.description'),
    icon: 'cpu'
  },
  {
    title: t('home.useCases.image.title'),
    description: t('home.useCases.image.description'),
    icon: 'sparkles'
  }
])

const providerModels = computed(() => [
  { name: 'GPT', status: t('home.providers.supported') },
  { name: 'Claude', status: t('home.providers.supported') },
  { name: 'Gemini', status: t('home.providers.supported') },
  { name: t('home.providers.image'), status: t('home.providers.supported') }
])

const currentYear = computed(() => new Date().getFullYear())

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()

  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style>
.custom-home-content,
.custom-home-content__body {
  width: 100% !important;
  max-width: 100vw !important;
  overflow-x: hidden !important;
}

.custom-home-content * {
  box-sizing: border-box !important;
  max-width: 100% !important;
}

.custom-home-content .homex-ambient {
  width: min(800px, 100vw) !important;
  max-width: 100vw !important;
}

.custom-home-content pre,
.custom-home-content code,
.custom-home-content pre span {
  min-width: 0 !important;
  max-width: 100% !important;
  white-space: pre-wrap !important;
  overflow-wrap: anywhere !important;
  word-break: break-word !important;
}

.custom-home-content pre {
  overflow-x: auto !important;
}

.custom-home-content pre > code {
  display: block !important;
  width: 100% !important;
}
</style>

<style scoped>
.home-default-shell,
.home-default-shell :deep(header),
.home-default-shell :deep(main),
.home-default-shell :deep(section),
.home-default-shell :deep(footer) {
  max-width: 100%;
}

.home-default-shell {
  overflow-x: hidden;
}

.home-default-shell :deep(*) {
  min-width: 0;
}

.home-default-shell :deep(h1),
.home-default-shell :deep(h2),
.home-default-shell :deep(p),
.home-default-shell :deep(dt),
.home-default-shell :deep(dd) {
  overflow-wrap: anywhere;
}

.custom-home-content {
  width: 100%;
  max-width: 100vw;
  overflow-x: hidden;
}

.custom-home-content__body {
  min-width: 0;
  max-width: 100vw;
  overflow-x: hidden;
}

.custom-home-content :deep(*) {
  max-width: 100%;
}

.custom-home-content :deep(pre),
.custom-home-content :deep(code) {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.custom-home-content :deep(pre) {
  overflow-x: auto;
}
</style>
