<template>
  <!-- Administrators can still replace the public home page with custom content. -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact mode remains available for deployments that explicitly enable it. -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt=""
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt=""
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">
          {{ siteSubtitle }}
        </p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <div v-else class="wegoo-home">
    <header class="home-nav">
      <nav class="home-nav__inner" :aria-label="t('home.landing.nav.primary')">
        <router-link to="/home" class="home-brand" :aria-label="siteName">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </router-link>

        <div class="home-nav__links">
          <a href="#platform">{{ t('home.landing.nav.noDilution') }}</a>
          <a href="#workflow">{{ t('home.landing.nav.billing') }}</a>
          <router-link to="/model-plaza">{{ t('home.providers.title') }}</router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ t('home.landing.nav.docs') }}
          </a>
        </div>

        <div class="home-nav__actions">
          <LocaleSwitcher />
          <button
            type="button"
            class="home-icon-button"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
          </button>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="home-sign-in">
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
          <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="home-primary-button home-nav__cta">
            {{ t('home.landing.hero.primaryAction') }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section class="home-hero" aria-labelledby="home-hero-title">
        <div class="home-hero__copy">
          <p class="home-eyebrow">{{ t('home.landing.hero.eyebrow') }}</p>
          <h1 id="home-hero-title">{{ siteName }}</h1>
          <p class="home-hero__statement">{{ t('home.landing.hero.title') }}</p>
          <p class="home-hero__lead">{{ t('home.landing.hero.lead') }}</p>

          <div class="home-hero__actions">
            <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="home-primary-button">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.landing.hero.primaryAction') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-secondary-button">
              <Icon name="book" size="sm" />
              {{ t('home.landing.nav.docs') }}
            </a>
            <router-link v-else to="/model-plaza" class="home-secondary-button">
              {{ t('home.landing.hero.statusAction') }}
            </router-link>
          </div>
        </div>

        <div class="home-stage" :aria-label="t('home.landing.product.statusValue')">
          <div class="home-stage__topbar">
            <div class="home-stage__status">
              <span></span>
              {{ t('home.landing.product.statusBadge') }}
            </div>
            <code>POST /v1/responses</code>
            <span>200 · streaming</span>
          </div>

          <div class="home-stage__flow">
            <div class="home-stage__endpoint">
              <span class="home-stage__step">01</span>
              <Icon name="terminal" size="lg" />
              <strong>{{ t('home.landing.workflow.application') }}</strong>
              <small>OpenAI SDK · Claude Code · Codex</small>
            </div>

            <div class="home-stage__connector" aria-hidden="true"><span></span></div>

            <div class="home-stage__gateway">
              <span class="home-stage__step">02</span>
              <img :src="siteLogo || '/logo.svg'" alt="" />
              <strong>{{ t('home.landing.workflow.gateway') }}</strong>
              <small>{{ t('home.landing.workflow.gatewayDetail') }}</small>
            </div>

            <div class="home-stage__connector" aria-hidden="true"><span></span></div>

            <div class="home-stage__models">
              <span class="home-stage__step">03</span>
              <strong>{{ t('home.landing.workflow.models') }}</strong>
              <div>
                <span v-for="provider in heroProviders" :key="provider.name" :class="provider.tone">
                  <i></i>{{ provider.name }}
                </span>
              </div>
            </div>
          </div>

          <div class="home-stage__footer">
            <span v-for="claim in heroClaims" :key="claim">
              <Icon name="checkCircle" size="sm" />
              {{ claim }}
            </span>
          </div>
        </div>
      </section>

      <section id="platform" class="home-section home-platform">
        <div class="home-section__heading">
          <p class="home-eyebrow">{{ t('home.landing.platform.kicker') }}</p>
          <h2>{{ t('home.landing.platform.title') }}</h2>
          <p>{{ t('home.landing.platform.lead') }}</p>
        </div>

        <div class="home-platform__grid">
          <article v-for="item in platformItems" :key="item.title">
            <span :class="['home-platform__icon', item.tone]">
              <Icon :name="item.icon" size="md" />
            </span>
            <h3>{{ item.title }}</h3>
            <p>{{ item.detail }}</p>
            <small>{{ item.meta }}</small>
          </article>
        </div>
      </section>

      <section id="workflow" class="home-workflow">
        <div class="home-workflow__inner">
          <div class="home-workflow__copy">
            <p class="home-eyebrow">{{ t('home.landing.integration.kicker') }}</p>
            <h2>{{ t('home.landing.integration.title') }}</h2>
            <p>{{ t('home.landing.integration.lead') }}</p>

            <ol>
              <li v-for="(step, index) in integrationSteps" :key="step">
                <span>0{{ index + 1 }}</span>
                {{ step }}
              </li>
            </ol>

            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-workflow__link">
              {{ t('home.landing.nav.docs') }}
              <Icon name="externalLink" size="sm" />
            </a>
          </div>

          <div class="home-code" aria-label="OpenAI SDK">
            <div class="home-code__bar">
              <span></span><span></span><span></span>
              <em>quickstart.ts</em>
            </div>
            <pre><code>{{ quickstartCode }}</code></pre>
          </div>
        </div>
      </section>

      <section class="home-cta">
        <img :src="siteLogo || '/logo.svg'" alt="" />
        <p class="home-eyebrow">{{ t('home.landing.cta.kicker') }}</p>
        <h2>{{ t('home.landing.cta.title') }}</h2>
        <p>{{ t('home.landing.cta.lead') }}</p>
        <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="home-primary-button">
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.landing.hero.primaryAction') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </section>
    </main>

    <footer class="home-footer">
      <div>
        <router-link to="/home" class="home-brand">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </router-link>
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
      </div>
      <div>
        <router-link to="/model-plaza">{{ t('home.providers.title') }}</router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('home.landing.nav.docs') }}</a>
        <router-link :to="isAuthenticated ? dashboardPath : '/login'">{{ t('home.landing.nav.continue') }}</router-link>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { DEFAULT_SITE_NAME } from '@/utils/branding'
import { sanitizeUrl } from '@/utils/url'

type IconName = InstanceType<typeof Icon>['$props']['name']

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || DEFAULT_SITE_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const apiBaseUrl = computed(() => (appStore.apiBaseUrl || 'https://api.wegoo.site').replace(/\/$/, ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const isHomeContentUrl = computed(() => /^https?:\/\//i.test(homeContent.value.trim()))

const isDark = ref(document.documentElement.classList.contains('dark'))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const currentYear = computed(() => new Date().getFullYear())

const heroProviders = [
  { name: 'GPT / Codex', tone: 'blue' },
  { name: 'Claude', tone: 'coral' },
  { name: 'Gemini', tone: 'gold' },
  { name: 'Image API', tone: 'green' },
]

const heroClaims = computed(() => [
  t('home.landing.claims.official'),
  t('home.landing.claims.noChats'),
  t('home.landing.claims.billingCovered'),
])

const platformItems = computed<Array<{
  icon: IconName
  tone: string
  title: string
  detail: string
  meta: string
}>>(() => [
  {
    icon: 'cube',
    tone: 'blue',
    title: t('home.landing.product.modelIntegrity'),
    detail: t('home.landing.product.modelIntegrityDetail'),
    meta: 'GPT · Claude · Gemini · Image',
  },
  {
    icon: 'key',
    tone: 'green',
    title: t('home.landing.product.privacyBoundary'),
    detail: t('home.landing.product.privacyBoundaryDetail'),
    meta: 'One key · Multiple endpoints',
  },
  {
    icon: 'chart',
    tone: 'coral',
    title: t('home.landing.product.billingProtection'),
    detail: t('home.landing.product.billingProtectionDetail'),
    meta: 'Token · Cache · Rate · Cost',
  },
  {
    icon: 'shield',
    tone: 'gold',
    title: t('home.landing.product.serviceContinuity'),
    detail: t('home.landing.product.serviceContinuityDetail'),
    meta: 'Availability · Latency · Incidents',
  },
])

const integrationSteps = computed(() => [
  t('home.landing.integration.steps.key'),
  t('home.landing.integration.steps.endpoint'),
  t('home.landing.integration.steps.request'),
])

const quickstartCode = computed(() => `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.WEGOO_API_KEY,
  baseURL: "${apiBaseUrl.value}/v1"
});

const response = await client.responses.create({
  model: "gpt-5.5",
  input: "Hello, Wegoo"
});`)

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
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

<style scoped>
.wegoo-home,
.wegoo-home * {
  box-sizing: border-box;
  letter-spacing: 0;
}

.wegoo-home {
  --home-bg: #f5f7f6;
  --home-surface: #ffffff;
  --home-ink: #121923;
  --home-muted: #627080;
  --home-line: #dfe5e4;
  --home-teal: #0f9f91;
  --home-teal-strong: #087d73;
  --home-blue: #2f6fed;
  --home-coral: #df744f;
  --home-gold: #c58b20;
  min-height: 100vh;
  overflow-x: hidden;
  background: var(--home-bg);
  color: var(--home-ink);
}

:global(.dark) .wegoo-home {
  --home-bg: #0b1016;
  --home-surface: #121922;
  --home-ink: #f5f7fa;
  --home-muted: #9ca9b7;
  --home-line: #27313d;
  --home-teal: #35c5b6;
  --home-teal-strong: #68dbcf;
  --home-blue: #76a0ff;
  --home-coral: #f19673;
  --home-gold: #e3b454;
}

.home-nav {
  position: sticky;
  top: 0;
  z-index: 30;
  border-bottom: 1px solid var(--home-line);
  background: color-mix(in srgb, var(--home-bg) 94%, transparent);
  backdrop-filter: blur(16px);
}

.home-nav__inner {
  display: flex;
  width: min(100% - 40px, 1240px);
  height: 72px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin: 0 auto;
}

.home-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 11px;
  color: var(--home-ink);
  text-decoration: none;
}

.home-brand img {
  width: 38px;
  height: 38px;
  flex: none;
  border-radius: 8px;
  object-fit: contain;
}

.home-brand span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 750;
}

.home-nav__links,
.home-nav__actions,
.home-hero__actions,
.home-primary-button,
.home-secondary-button,
.home-sign-in {
  display: flex;
  align-items: center;
}

.home-nav__links {
  justify-content: center;
  gap: 26px;
}

.home-nav__links a,
.home-sign-in,
.home-footer a {
  color: var(--home-muted);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  transition: color 0.18s ease;
}

.home-nav__links a:hover,
.home-sign-in:hover,
.home-footer a:hover {
  color: var(--home-ink);
}

.home-nav__actions {
  flex: none;
  gap: 10px;
}

.home-icon-button {
  display: inline-flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--home-muted);
}

.home-icon-button:hover {
  border-color: var(--home-line);
  color: var(--home-ink);
}

.home-sign-in {
  min-height: 38px;
  justify-content: center;
  padding: 0 8px;
}

.home-primary-button,
.home-secondary-button {
  min-height: 44px;
  justify-content: center;
  gap: 9px;
  border-radius: 8px;
  padding: 0 18px;
  font-size: 14px;
  font-weight: 700;
  text-decoration: none;
  transition: transform 0.18s ease, background 0.18s ease, border-color 0.18s ease;
}

.home-primary-button {
  border: 1px solid var(--home-teal);
  background: var(--home-teal);
  color: #ffffff;
  box-shadow: 0 10px 30px rgba(15, 159, 145, 0.2);
}

.home-primary-button:hover {
  background: var(--home-teal-strong);
  border-color: var(--home-teal-strong);
  transform: translateY(-1px);
}

.home-secondary-button {
  border: 1px solid var(--home-line);
  background: var(--home-surface);
  color: var(--home-ink);
}

.home-secondary-button:hover {
  border-color: color-mix(in srgb, var(--home-teal) 52%, var(--home-line));
  transform: translateY(-1px);
}

.home-nav__cta {
  min-height: 38px;
  padding: 0 14px;
  font-size: 13px;
}

.home-hero {
  display: flex;
  width: min(100% - 40px, 1180px);
  min-height: 700px;
  flex-direction: column;
  justify-content: center;
  gap: 44px;
  margin: 0 auto;
  padding: 58px 0 64px;
}

.home-hero__copy {
  max-width: 830px;
  margin: 0 auto;
  text-align: center;
}

.home-eyebrow {
  margin: 0 0 14px;
  color: var(--home-teal-strong);
  font-size: 12px;
  font-weight: 800;
  line-height: 1.4;
  text-transform: uppercase;
}

.home-hero h1 {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--home-ink);
  font-size: 64px;
  font-weight: 760;
  line-height: 1.04;
}

.home-hero__statement {
  margin: 18px 0 0;
  color: var(--home-ink);
  font-size: 30px;
  font-weight: 650;
  line-height: 1.25;
}

.home-hero__lead {
  max-width: 720px;
  margin: 18px auto 0;
  color: var(--home-muted);
  font-size: 16px;
  line-height: 1.75;
}

.home-hero__actions {
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
  margin-top: 28px;
}

.home-stage {
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--home-line);
  border-radius: 8px;
  background: var(--home-surface);
  box-shadow: 0 24px 70px rgba(22, 35, 45, 0.1);
}

.home-stage__topbar,
.home-stage__footer {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 18px;
  padding: 0 18px;
  color: var(--home-muted);
  font-size: 11px;
}

.home-stage__topbar {
  justify-content: space-between;
  border-bottom: 1px solid var(--home-line);
}

.home-stage__status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--home-ink);
  font-weight: 700;
}

.home-stage__status span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #25a979;
  box-shadow: 0 0 0 4px rgba(37, 169, 121, 0.12);
}

.home-stage__topbar code {
  color: var(--home-blue);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.home-stage__flow {
  display: grid;
  min-height: 190px;
  grid-template-columns: minmax(0, 1fr) 72px minmax(0, 1fr) 72px minmax(0, 1.15fr);
  align-items: stretch;
  padding: 18px;
}

.home-stage__endpoint,
.home-stage__gateway,
.home-stage__models {
  position: relative;
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 9px;
  border: 1px solid var(--home-line);
  border-radius: 8px;
  padding: 20px 16px;
  text-align: center;
}

.home-stage__endpoint > svg {
  color: var(--home-blue);
}

.home-stage__gateway {
  border-color: color-mix(in srgb, var(--home-teal) 48%, var(--home-line));
  background: color-mix(in srgb, var(--home-teal) 5%, var(--home-surface));
}

.home-stage__gateway img {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  object-fit: contain;
}

.home-stage__endpoint strong,
.home-stage__gateway strong,
.home-stage__models strong {
  color: var(--home-ink);
  font-size: 14px;
}

.home-stage__endpoint small,
.home-stage__gateway small {
  color: var(--home-muted);
  font-size: 11px;
  line-height: 1.5;
}

.home-stage__step {
  position: absolute;
  top: 11px;
  left: 12px;
  color: var(--home-muted);
  font-size: 10px;
  font-weight: 800;
}

.home-stage__connector {
  display: flex;
  align-items: center;
  padding: 0 10px;
}

.home-stage__connector span {
  position: relative;
  display: block;
  width: 100%;
  height: 1px;
  background: var(--home-line);
}

.home-stage__connector span::after {
  position: absolute;
  top: -3px;
  right: -1px;
  width: 7px;
  height: 7px;
  border-top: 1px solid var(--home-teal);
  border-right: 1px solid var(--home-teal);
  content: '';
  transform: rotate(45deg);
}

.home-stage__models > div {
  display: grid;
  width: 100%;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7px;
}

.home-stage__models > div > span {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--home-line);
  border-radius: 6px;
  padding: 7px 8px;
  color: var(--home-muted);
  font-size: 10px;
  font-weight: 650;
  white-space: nowrap;
}

.home-stage__models i {
  width: 7px;
  height: 7px;
  flex: none;
  border-radius: 2px;
  background: var(--home-blue);
}

.home-stage__models .coral i { background: var(--home-coral); }
.home-stage__models .gold i { background: var(--home-gold); }
.home-stage__models .green i { background: var(--home-teal); }

.home-stage__footer {
  flex-wrap: wrap;
  justify-content: center;
  border-top: 1px solid var(--home-line);
}

.home-stage__footer span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.home-stage__footer svg {
  color: var(--home-teal);
}

.home-section {
  width: min(100% - 40px, 1180px);
  margin: 0 auto;
  padding: 92px 0;
}

.home-section__heading {
  max-width: 680px;
  margin-bottom: 48px;
}

.home-section__heading h2,
.home-workflow h2,
.home-cta h2 {
  margin: 0;
  color: var(--home-ink);
  font-size: 38px;
  font-weight: 720;
  line-height: 1.2;
}

.home-section__heading > p:last-child,
.home-workflow__copy > p,
.home-cta > p:not(.home-eyebrow) {
  margin: 16px 0 0;
  color: var(--home-muted);
  font-size: 15px;
  line-height: 1.75;
}

.home-platform {
  border-top: 1px solid var(--home-line);
}

.home-platform__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-top: 1px solid var(--home-line);
  border-bottom: 1px solid var(--home-line);
}

.home-platform__grid article {
  min-width: 0;
  padding: 30px 24px 32px;
  border-right: 1px solid var(--home-line);
}

.home-platform__grid article:last-child {
  border-right: 0;
}

.home-platform__icon {
  display: inline-flex;
  width: 38px;
  height: 38px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: color-mix(in srgb, var(--home-blue) 11%, var(--home-surface));
  color: var(--home-blue);
}

.home-platform__icon.green {
  background: color-mix(in srgb, var(--home-teal) 11%, var(--home-surface));
  color: var(--home-teal);
}

.home-platform__icon.coral {
  background: color-mix(in srgb, var(--home-coral) 11%, var(--home-surface));
  color: var(--home-coral);
}

.home-platform__icon.gold {
  background: color-mix(in srgb, var(--home-gold) 12%, var(--home-surface));
  color: var(--home-gold);
}

.home-platform__grid h3 {
  margin: 22px 0 0;
  color: var(--home-ink);
  font-size: 17px;
  font-weight: 700;
}

.home-platform__grid p {
  min-height: 72px;
  margin: 10px 0 0;
  color: var(--home-muted);
  font-size: 13px;
  line-height: 1.65;
}

.home-platform__grid small {
  display: block;
  margin-top: 20px;
  color: var(--home-teal-strong);
  font-size: 10px;
  font-weight: 750;
  line-height: 1.5;
}

.home-workflow {
  background: #111923;
  color: #ffffff;
}

.home-workflow__inner {
  display: grid;
  width: min(100% - 40px, 1180px);
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
  align-items: center;
  gap: 72px;
  margin: 0 auto;
  padding: 92px 0;
}

.home-workflow h2 {
  color: #ffffff;
}

.home-workflow__copy > p {
  color: #a9b4c0;
}

.home-workflow ol {
  display: grid;
  gap: 0;
  margin: 28px 0 0;
  padding: 0;
  list-style: none;
}

.home-workflow li {
  display: flex;
  align-items: center;
  gap: 14px;
  border-top: 1px solid #2b3541;
  padding: 15px 0;
  color: #d8dee5;
  font-size: 13px;
}

.home-workflow li:last-child {
  border-bottom: 1px solid #2b3541;
}

.home-workflow li span {
  color: #5fd3c8;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}

.home-workflow__link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 24px;
  color: #5fd3c8;
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
}

.home-code {
  overflow: hidden;
  border: 1px solid #303b48;
  border-radius: 8px;
  background: #0a1017;
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.28);
}

.home-code__bar {
  display: flex;
  height: 42px;
  align-items: center;
  gap: 7px;
  border-bottom: 1px solid #27313d;
  padding: 0 14px;
}

.home-code__bar > span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #df744f;
}

.home-code__bar > span:nth-child(2) { background: #c58b20; }
.home-code__bar > span:nth-child(3) { background: #0f9f91; }

.home-code__bar em {
  margin-left: 5px;
  color: #7f8c99;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  font-style: normal;
}

.home-code pre {
  min-height: 320px;
  margin: 0;
  overflow: auto;
  padding: 28px;
}

.home-code code {
  color: #d9e1e8;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 1.85;
  white-space: pre;
}

.home-cta {
  width: min(100% - 40px, 880px);
  margin: 0 auto;
  padding: 104px 0;
  text-align: center;
}

.home-cta > img {
  width: 52px;
  height: 52px;
  margin-bottom: 22px;
  border-radius: 8px;
  object-fit: contain;
}

.home-cta > p:not(.home-eyebrow) {
  max-width: 620px;
  margin-right: auto;
  margin-left: auto;
}

.home-cta .home-primary-button {
  display: inline-flex;
  margin-top: 28px;
}

.home-footer {
  display: flex;
  width: min(100% - 40px, 1180px);
  align-items: flex-end;
  justify-content: space-between;
  gap: 28px;
  margin: 0 auto;
  border-top: 1px solid var(--home-line);
  padding: 28px 0 34px;
}

.home-footer > div:last-child {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 22px;
}

.home-footer p {
  margin: 10px 0 0;
  color: var(--home-muted);
  font-size: 11px;
}

@media (max-width: 1020px) {
  .home-nav__links {
    display: none;
  }

  .home-platform__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .home-platform__grid article:nth-child(2) {
    border-right: 0;
  }

  .home-platform__grid article:nth-child(-n + 2) {
    border-bottom: 1px solid var(--home-line);
  }

  .home-workflow__inner {
    grid-template-columns: 1fr;
    gap: 48px;
  }
}

@media (max-width: 720px) {
  .home-nav__inner {
    width: min(100% - 28px, 1180px);
    height: 64px;
    gap: 10px;
  }

  .home-brand span,
  .home-sign-in,
  .home-nav__cta {
    display: none;
  }

  .home-nav__actions {
    gap: 4px;
  }

  .home-hero {
    width: min(100% - 28px, 1180px);
    min-height: 710px;
    gap: 32px;
    padding: 44px 0 50px;
  }

  .home-hero h1 {
    font-size: 42px;
    line-height: 1.08;
  }

  .home-hero__statement {
    font-size: 24px;
  }

  .home-hero__lead {
    font-size: 14px;
    line-height: 1.7;
  }

  .home-hero__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .home-stage__topbar > code {
    display: none;
  }

  .home-stage__flow {
    min-height: 282px;
    grid-template-columns: 1fr 34px 1fr;
    padding: 12px;
  }

  .home-stage__endpoint,
  .home-stage__gateway {
    padding: 18px 10px;
  }

  .home-stage__connector {
    padding: 0 7px;
  }

  .home-stage__models {
    display: none;
  }

  .home-stage__flow > .home-stage__connector:nth-child(4) {
    display: none;
  }

  .home-stage__footer {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
    padding: 12px 14px;
  }

  .home-section,
  .home-workflow__inner,
  .home-cta,
  .home-footer {
    width: min(100% - 28px, 1180px);
  }

  .home-section,
  .home-workflow__inner {
    padding: 70px 0;
  }

  .home-section__heading h2,
  .home-workflow h2,
  .home-cta h2 {
    font-size: 30px;
  }

  .home-platform__grid {
    grid-template-columns: 1fr;
  }

  .home-platform__grid article,
  .home-platform__grid article:nth-child(2) {
    border-right: 0;
    border-bottom: 1px solid var(--home-line);
  }

  .home-platform__grid article:last-child {
    border-bottom: 0;
  }

  .home-platform__grid p {
    min-height: 0;
  }

  .home-code pre {
    min-height: 0;
    padding: 20px;
  }

  .home-code code {
    font-size: 10px;
  }

  .home-cta {
    padding: 78px 0;
  }

  .home-footer {
    align-items: flex-start;
    flex-direction: column;
  }

  .home-footer > div:last-child {
    justify-content: flex-start;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-primary-button,
  .home-secondary-button,
  .home-nav__links a,
  .home-sign-in {
    transition: none;
  }
}
</style>
