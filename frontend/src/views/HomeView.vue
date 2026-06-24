<template>
  <div v-if="homeContent" class="custom-home-content min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox"
      referrerpolicy="strict-origin-when-cross-origin"
      allowfullscreen
    ></iframe>
    <div v-else class="custom-home-content__body" v-html="sanitizedHomeContent"></div>
  </div>

  <div v-else :class="['home-page min-h-screen', { 'home-page--dark': isDark }]">
    <header class="home-nav">
      <nav class="home-nav__inner">
        <router-link to="/" class="home-brand" :aria-label="t('home.landing.nav.home')">
          <span class="home-brand__mark">
            <img :src="siteLogo || '/logo.png'" alt="" />
          </span>
          <span class="home-brand__name">{{ siteName }}</span>
        </router-link>

        <div class="home-nav__links" :aria-label="t('home.landing.nav.primary')">
          <a v-for="item in navAnchors" :key="item.href" :href="item.href">
            {{ item.label }}
          </a>
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            @click="handleDocsLinkClick"
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
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="home-nav-action">
            {{ isAuthenticated ? t('home.dashboard') : t('home.landing.nav.signIn') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section class="home-hero" aria-labelledby="home-hero-title">
        <div class="home-hero__copy">
          <p class="home-eyebrow">{{ siteName }}</p>
          <h1 id="home-hero-title" :class="{ 'home-hero__title--cjk': heroTitleIsCjk }">
            <span v-for="line in heroTitleLines" :key="line" class="home-hero__title-line">
              {{ line }}
            </span>
          </h1>
          <p class="home-hero__lead">
            {{ heroLead }}
          </p>
          <ul class="home-claim-list" :aria-label="t('home.landing.hero.capabilitiesLabel')">
            <li v-for="claim in heroClaims" :key="claim.label">
              <span :class="claim.accent"></span>
              {{ claim.label }}
            </li>
          </ul>
          <p class="home-hero__meta">
            {{ heroMeta }}
          </p>
        </div>

        <div class="home-hero__visual" aria-hidden="true">
          <div class="home-product-panel">
            <div class="home-product-panel__top">
              <div>
                <span>{{ t('home.landing.product.statusLabel') }}</span>
                <strong>{{ t('home.landing.product.statusValue') }}</strong>
              </div>
              <b>{{ t('home.landing.product.statusBadge') }}</b>
            </div>

            <div class="home-product-rows">
              <div v-for="item in productRows" :key="item.name" class="home-product-row">
                <div>
                  <span :class="['home-product-dot', item.accent]"></span>
                  <strong>{{ item.name }}</strong>
                </div>
                <em>{{ item.detail }}</em>
              </div>
            </div>

            <div class="home-product-metrics">
              <div v-for="item in productMetrics" :key="item.label">
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="experience" class="home-band home-band--light" aria-labelledby="experience-title">
        <div class="home-section-copy home-section-copy--center">
          <p class="home-eyebrow">{{ t('home.landing.sections.models.kicker') }}</p>
          <h2 id="experience-title">
            {{ t('home.landing.sections.models.title') }}
          </h2>
          <p>{{ t('home.landing.sections.models.description') }}</p>
        </div>

        <div class="home-model-lineup" aria-hidden="true">
          <div v-for="item in modelLineup" :key="item.name" class="home-model-lineup__item">
            <span :class="item.accent"></span>
            <strong>{{ item.name }}</strong>
            <em>{{ item.detail }}</em>
          </div>
        </div>
      </section>

      <section id="billing" class="home-band home-band--dark" aria-labelledby="billing-title">
        <div class="home-section-copy">
          <p class="home-eyebrow">{{ t('home.landing.sections.billing.kicker') }}</p>
          <h2 id="billing-title">
            {{ t('home.landing.sections.billing.title') }}
          </h2>
          <p>{{ t('home.landing.sections.billing.description') }}</p>
        </div>

        <div class="home-ledger" aria-hidden="true">
          <div v-for="item in ledgerRows" :key="item.label" class="home-ledger__row">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </section>

      <section id="usage" class="home-band home-band--split" aria-labelledby="usage-title">
        <div class="home-usage-visual" aria-hidden="true">
          <div class="home-usage-visual__summary">
            <span>{{ t('home.landing.privacyVisual.label') }}</span>
            <strong>0</strong>
          </div>
          <div class="home-usage-visual__bars">
            <span v-for="height in usageBars" :key="height" :style="{ height }"></span>
          </div>
        </div>

        <div class="home-section-copy">
          <p class="home-eyebrow">{{ t('home.landing.sections.privacy.kicker') }}</p>
          <h2 id="usage-title">
            {{ t('home.landing.sections.privacy.title') }}
          </h2>
          <p>{{ t('home.landing.sections.privacy.description') }}</p>
        </div>
      </section>

      <section id="stability" class="home-band home-band--final" aria-labelledby="stability-title">
        <div class="home-section-copy home-section-copy--center">
          <p class="home-eyebrow">{{ t('home.landing.sections.stability.kicker') }}</p>
          <h2 id="stability-title">
            {{ t('home.landing.sections.stability.title') }}
          </h2>
          <p>{{ t('home.landing.sections.stability.description') }}</p>
        </div>
      </section>
    </main>

    <footer class="home-footer">
      <div>
        <p>
          &copy; {{ currentYear }} {{ siteName }}<span v-if="siteSubtitle">. {{ siteSubtitle }}</span>
        </p>
        <div>
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            @click="handleDocsLinkClick"
          >
            {{ t('home.landing.nav.docs') }}
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import DOMPurify from 'dompurify'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/utils/docsLink'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const sanitizedHomeContent = computed(() => DOMPurify.sanitize(homeContent.value))
const rawDocUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const docsLink = computed(() => resolveDocsLink(rawDocUrl.value, appStore.cachedPublicSettings?.custom_menu_items ?? []))
const isHomeContentUrl = computed(() => /^https?:\/\//i.test(homeContent.value.trim()))

const isDark = ref(typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
let themeObserver: MutationObserver | null = null

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const currentYear = computed(() => new Date().getFullYear())

const navAnchors = computed(() => [
  { href: '#experience', label: t('home.landing.nav.noDilution') },
  { href: '#billing', label: t('home.landing.nav.billing') },
  { href: '#usage', label: t('home.landing.nav.privacy') },
  { href: '#stability', label: t('home.landing.nav.stability') }
])

const heroTitle = computed(() => t('home.landing.hero.title'))
const heroTitleLines = computed(() => splitHeroTitle(heroTitle.value))
const heroTitleIsCjk = computed(() => /[\u4e00-\u9fff]/.test(heroTitle.value))
const heroLead = computed(() => t('home.landing.hero.lead'))
const heroMeta = computed(() => t('home.landing.hero.meta'))

const heroClaims = computed(() => [
  { label: t('home.landing.claims.official'), accent: 'home-accent--blue' },
  { label: t('home.landing.claims.noDilution'), accent: 'home-accent--green' },
  { label: t('home.landing.claims.noChats'), accent: 'home-accent--pink' },
  { label: t('home.landing.claims.billingCovered'), accent: 'home-accent--orange' }
])

const productRows = computed(() => [
  { name: 'GPT', detail: t('home.landing.product.gpt'), accent: 'home-accent--blue' },
  { name: 'Claude', detail: t('home.landing.product.claude'), accent: 'home-accent--orange' },
  { name: 'Gemini', detail: t('home.landing.product.gemini'), accent: 'home-accent--green' },
  { name: t('home.landing.product.images'), detail: t('home.landing.product.imagesDetail'), accent: 'home-accent--pink' }
])

const productMetrics = computed(() => [
  { label: t('home.landing.metrics.official'), value: '100%' },
  { label: t('home.landing.metrics.storedChats'), value: '0' },
  { label: t('home.landing.metrics.chargeExceptions'), value: t('home.landing.metrics.covered') }
])

const modelLineup = computed(() => [
  { name: 'GPT', detail: t('home.landing.lineup.gpt'), accent: 'home-accent--blue' },
  { name: 'Claude', detail: t('home.landing.lineup.claude'), accent: 'home-accent--orange' },
  { name: 'Gemini', detail: t('home.landing.lineup.gemini'), accent: 'home-accent--green' },
  { name: t('home.landing.product.images'), detail: t('home.landing.lineup.images'), accent: 'home-accent--pink' }
])

const ledgerRows = computed(() => [
  { label: t('home.landing.ledger.modelSource'), value: t('home.landing.ledger.official') },
  { label: t('home.landing.ledger.usageTrail'), value: t('home.landing.ledger.trace') },
  { label: t('home.landing.ledger.billingExceptions'), value: t('home.landing.ledger.covered') },
  { label: t('home.landing.ledger.serviceState'), value: t('home.landing.ledger.online') }
])

const usageBars = ['42%', '56%', '48%', '72%', '62%', '84%', '68%', '78%']

function splitHeroTitle(title: string): string[] {
  const normalized = title.trim()
  if (!normalized) return []

  if (/[\u4e00-\u9fff]/.test(normalized)) {
    const commaSplit = normalized.split(/[，,]/).map((line) => line.trim()).filter(Boolean)
    if (commaSplit.length > 1) return commaSplit
  }

  const sentenceSplit = normalized.split(/(?<=[.!?。！？])\s+/).map((line) => line.trim()).filter(Boolean)
  if (sentenceSplit.length > 1 && sentenceSplit.length <= 3) return sentenceSplit

  return [normalized]
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function syncThemeFromDocument() {
  isDark.value = document.documentElement.classList.contains('dark')
}

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  isDark.value = savedTheme === 'dark' || (!savedTheme && prefersDark)
  document.documentElement.classList.toggle('dark', isDark.value)
}

onMounted(() => {
  initTheme()
  syncThemeFromDocument()
  themeObserver = new MutationObserver(syncThemeFromDocument)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})

onUnmounted(() => {
  themeObserver?.disconnect()
  themeObserver = null
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

.home-page {
  --home-bg: #f5f5f7;
  --home-surface: #ffffff;
  --home-surface-alt: #fbfbfd;
  --home-text: #1d1d1f;
  --home-muted: #6e6e73;
  --home-soft: #86868b;
  --home-border: rgb(0 0 0 / 10%);
  --home-border-soft: rgb(0 0 0 / 6%);
  --home-blue: #0071e3;
  --home-blue-hover: #0077ed;
  min-height: 100vh;
  overflow-x: hidden;
  background: var(--home-bg);
  color: var(--home-text);
}

.home-page--dark {
  --home-bg: #050505;
  --home-surface: #151516;
  --home-surface-alt: #0f0f10;
  --home-text: #f5f5f7;
  --home-muted: #a1a1a6;
  --home-soft: #86868b;
  --home-border: rgb(255 255 255 / 11%);
  --home-border-soft: rgb(255 255 255 / 7%);
  --home-blue: #2997ff;
  --home-blue-hover: #6bbcff;
}

.home-page :deep(*) {
  min-width: 0;
}

.home-nav {
  position: sticky;
  top: 0;
  z-index: 40;
  border-bottom: 1px solid var(--home-border-soft);
  background: rgb(245 245 247 / 84%);
  backdrop-filter: saturate(180%) blur(18px);
}

.home-page--dark .home-nav {
  background: rgb(0 0 0 / 78%);
}

.home-nav__inner {
  display: flex;
  height: 44px;
  max-width: 1180px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin: 0 auto;
  padding: 0 22px;
}

.home-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 9px;
  color: inherit;
}

.home-brand__mark {
  display: inline-flex;
  width: 26px;
  height: 26px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 8px;
  background: var(--home-surface);
  box-shadow: inset 0 0 0 1px var(--home-border-soft);
}

.home-brand__mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.home-brand__name {
  min-width: 0;
  max-width: 220px;
  overflow: hidden;
  color: var(--home-text);
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-nav__links {
  display: flex;
  align-items: center;
  gap: 30px;
}

.home-nav__links a,
.home-footer a {
  color: color-mix(in srgb, var(--home-text) 68%, transparent);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0;
  transition: color 0.18s ease;
}

.home-nav__links a:hover,
.home-footer a:hover {
  color: var(--home-text);
}

.home-nav__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.home-icon-button {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: color-mix(in srgb, var(--home-text) 72%, transparent);
  transition: background-color 0.18s ease, color 0.18s ease;
}

.home-icon-button:hover {
  background: var(--home-border-soft);
  color: var(--home-text);
}

.home-nav-action {
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--home-blue);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0;
  padding: 0 14px;
  transition: background-color 0.18s ease;
}

.home-nav-action:hover {
  background: var(--home-blue-hover);
}

.home-hero {
  display: grid;
  min-height: min(820px, calc(100svh - 64px));
  align-content: start;
  overflow: hidden;
  background: var(--home-bg);
  padding: 94px 22px 42px;
  text-align: center;
}

.home-hero__copy {
  width: min(100%, 1040px);
  margin: 0 auto;
}

.home-eyebrow {
  margin: 0;
  color: var(--home-soft);
  font-size: 17px;
  font-weight: 650;
  letter-spacing: 0;
}

.home-hero .home-eyebrow {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-hero h1 {
  max-width: 960px;
  margin: 10px auto 0;
  color: var(--home-text);
  font-size: clamp(56px, 6.4vw, 82px);
  font-weight: 700;
  letter-spacing: 0;
  line-height: 0.98;
  overflow-wrap: normal;
  text-wrap: balance;
  word-break: keep-all;
}

.home-hero__title-line {
  display: block;
  white-space: nowrap;
}

.home-hero__lead {
  max-width: 720px;
  margin: 20px auto 0;
  color: var(--home-muted);
  font-size: 24px;
  font-weight: 550;
  letter-spacing: 0;
  line-height: 1.32;
  text-wrap: balance;
}

.home-hero__meta {
  max-width: 720px;
  margin: 18px auto 0;
  color: var(--home-soft);
  font-size: 15px;
  font-weight: 500;
  line-height: 1.5;
  text-wrap: balance;
}

.home-claim-list {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin: 28px auto 0;
  padding: 0;
  list-style: none;
}

.home-claim-list li {
  display: inline-flex;
  min-height: 34px;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--home-border-soft);
  border-radius: 8px;
  background: color-mix(in srgb, var(--home-surface) 72%, transparent);
  color: var(--home-muted);
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 0;
  padding: 0 12px;
}

.home-claim-list span {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 8px;
}

.home-hero__visual {
  display: grid;
  width: min(100%, 860px);
  justify-items: center;
  margin: 38px auto 0;
}

.home-product-panel {
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--home-border-soft);
  border-radius: 8px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--home-surface) 96%, transparent), var(--home-surface-alt));
  box-shadow: 0 24px 70px rgb(0 0 0 / 8%);
  text-align: left;
}

.home-page--dark .home-product-panel {
  box-shadow: 0 28px 80px rgb(0 0 0 / 70%);
}

.home-product-panel__top {
  display: flex;
  min-height: 112px;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  border-bottom: 1px solid var(--home-border-soft);
  padding: 24px;
}

.home-product-panel__top span {
  display: block;
  color: var(--home-soft);
  font-size: 13px;
  font-weight: 650;
}

.home-product-panel__top strong {
  display: block;
  margin-top: 10px;
  color: var(--home-text);
  font-size: 34px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.04;
  text-wrap: balance;
}

.home-product-panel__top b {
  display: inline-flex;
  min-height: 28px;
  flex: 0 0 auto;
  align-items: center;
  border-radius: 8px;
  background: color-mix(in srgb, var(--home-blue) 12%, transparent);
  color: var(--home-blue);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  padding: 0 10px;
}

.home-product-rows {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.home-product-row {
  display: grid;
  min-height: 152px;
  align-content: start;
  gap: 16px;
  border-right: 1px solid var(--home-border-soft);
  border-bottom: 1px solid var(--home-border-soft);
  padding: 22px;
}

.home-product-row:last-child {
  border-right: 0;
}

.home-product-row > div {
  display: grid;
  align-content: start;
  gap: 14px;
}

.home-product-row strong {
  display: inline;
  color: var(--home-text);
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.12;
}

.home-product-row em {
  display: block;
  color: var(--home-muted);
  font-size: 13px;
  font-style: normal;
  line-height: 1.42;
  text-wrap: balance;
}

.home-product-dot {
  width: 30px;
  height: 4px;
  border-radius: 8px;
}

.home-product-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.home-product-metrics div {
  min-height: 96px;
  border-right: 1px solid var(--home-border-soft);
  padding: 22px;
}

.home-product-metrics div:last-child {
  border-right: 0;
}

.home-product-metrics span {
  color: var(--home-soft);
  font-size: 12px;
  font-weight: 650;
}

.home-product-metrics strong {
  display: block;
  margin-top: 12px;
  color: var(--home-text);
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 0;
}

.home-band {
  scroll-margin-top: 66px;
}

.home-band--light,
.home-band--final {
  max-width: 1180px;
  margin: 0 auto;
  padding: 104px 22px;
}

.home-band--dark {
  display: grid;
  max-width: none;
  grid-template-columns: minmax(0, 0.94fr) minmax(0, 1.06fr);
  align-items: center;
  gap: 70px;
  background: #000;
  color: #f5f5f7;
  padding: 104px max(22px, calc((100vw - 1180px) / 2 + 22px));
}

.home-band--split {
  display: grid;
  max-width: 1180px;
  grid-template-columns: minmax(0, 1.04fr) minmax(0, 0.96fr);
  align-items: center;
  gap: 70px;
  margin: 0 auto;
  padding: 104px 22px;
}

.home-band--final {
  text-align: center;
}

.home-section-copy--center {
  max-width: 900px;
  margin: 0 auto;
  text-align: center;
}

.home-section-copy h2 {
  margin: 12px 0 0;
  color: var(--home-text);
  font-size: 56px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.06;
  text-wrap: balance;
}

.home-section-copy p:not(.home-eyebrow) {
  max-width: 650px;
  margin: 24px 0 0;
  color: var(--home-muted);
  font-size: 18px;
  font-weight: 500;
  line-height: 1.5;
  text-wrap: balance;
}

.home-section-copy--center p:not(.home-eyebrow) {
  margin-right: auto;
  margin-left: auto;
}

.home-band--dark .home-eyebrow {
  color: rgb(255 255 255 / 52%);
}

.home-band--dark .home-section-copy h2 {
  color: #f5f5f7;
}

.home-band--dark .home-section-copy p:not(.home-eyebrow) {
  color: rgb(255 255 255 / 70%);
}

.home-model-lineup {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
  overflow: hidden;
  border: 1px solid var(--home-border-soft);
  border-radius: 8px;
  margin-top: 54px;
  background: var(--home-surface);
}

.home-page--dark .home-model-lineup {
  background: var(--home-surface-alt);
}

.home-model-lineup__item {
  min-height: 0;
  border-right: 1px solid var(--home-border-soft);
  border-bottom: 1px solid var(--home-border-soft);
  background: transparent;
  padding: 22px 24px 24px;
  text-align: left;
}

.home-model-lineup__item:nth-child(2n) {
  border-right: 0;
}

.home-model-lineup__item:nth-last-child(-n + 2) {
  border-bottom: 0;
}

.home-model-lineup__item span {
  display: block;
  width: 38px;
  height: 4px;
  border-radius: 8px;
}

.home-accent--blue {
  background: #0a84ff;
}

.home-accent--green {
  background: #30d158;
}

.home-accent--pink {
  background: #ff375f;
}

.home-accent--orange {
  background: #ff9f0a;
}

.home-model-lineup__item strong {
  display: block;
  margin-top: 18px;
  color: var(--home-text);
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.12;
}

.home-model-lineup__item em {
  display: block;
  margin-top: 10px;
  color: var(--home-muted);
  font-size: 14px;
  font-style: normal;
  line-height: 1.42;
}

.home-ledger {
  display: grid;
  gap: 0;
  overflow: hidden;
  border-radius: 8px;
  background: rgb(255 255 255 / 5%);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 7%);
}

.home-ledger__row {
  display: grid;
  min-height: 94px;
  grid-template-columns: minmax(0, 0.8fr) minmax(0, 1.2fr);
  align-items: center;
  gap: 20px;
  border-bottom: 1px solid rgb(255 255 255 / 7%);
  padding: 0 26px;
}

.home-ledger__row:last-child {
  border-bottom: 0;
}

.home-ledger__row span {
  color: rgb(255 255 255 / 52%);
  font-size: 14px;
  font-weight: 650;
}

.home-ledger__row strong {
  min-width: 0;
  color: #f5f5f7;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.16;
  overflow-wrap: anywhere;
  text-wrap: balance;
}

.home-usage-visual {
  position: relative;
  min-height: 430px;
  overflow: hidden;
  border-radius: 8px;
  background: var(--home-surface);
  box-shadow: inset 0 0 0 1px var(--home-border-soft);
}

.home-page--dark .home-usage-visual {
  background: var(--home-surface-alt);
}

.home-usage-visual__summary {
  position: absolute;
  top: 34px;
  left: 34px;
  display: grid;
  gap: 6px;
}

.home-usage-visual__summary span {
  color: var(--home-muted);
  font-size: 14px;
  font-weight: 650;
}

.home-usage-visual__summary strong {
  color: var(--home-text);
  font-size: 54px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1;
}

.home-usage-visual__bars {
  position: absolute;
  right: 42px;
  bottom: 42px;
  left: 42px;
  display: grid;
  height: 240px;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  align-items: end;
  gap: 12px;
}

.home-usage-visual__bars span {
  min-height: 36px;
  border-radius: 8px 8px 0 0;
  background: color-mix(in srgb, var(--home-text) 11%, transparent);
}

.home-usage-visual__bars span:nth-child(3n + 1) {
  background: #34c759;
}

.home-usage-visual__bars span:nth-child(3n + 2) {
  background: #64d2ff;
}

.home-usage-visual__bars span:nth-child(3n) {
  background: #ff9f0a;
}

.home-footer {
  background: var(--home-bg);
  color: var(--home-soft);
  padding: 0 22px 32px;
}

.home-footer > div {
  display: flex;
  max-width: 1180px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  border-top: 1px solid var(--home-border-soft);
  margin: 0 auto;
  padding-top: 24px;
  font-size: 12px;
}

.home-footer p {
  margin: 0;
}

.home-footer div div {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
}

@media (max-width: 1020px) {
  .home-nav__links {
    display: none;
  }

  .home-hero h1 {
    font-size: clamp(46px, 7.2vw, 62px);
  }

  .home-hero__lead {
    font-size: 20px;
  }

  .home-hero__meta {
    font-size: 14px;
  }

  .home-band--dark,
  .home-band--split {
    grid-template-columns: 1fr;
    gap: 46px;
    padding-top: 88px;
    padding-bottom: 88px;
  }

  .home-band--dark {
    padding-right: 22px;
    padding-left: 22px;
  }

  .home-section-copy h2 {
    font-size: 46px;
  }

  .home-product-rows {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .home-product-row:nth-child(2n) {
    border-right: 0;
  }

  .home-product-row:nth-last-child(-n + 2) {
    border-bottom: 0;
  }
}

@media (max-width: 640px) {
  .home-nav__inner {
    padding: 0 14px;
  }

  .home-brand__name {
    max-width: 148px;
  }

  .home-hero {
    min-height: auto;
    padding: 62px 16px 28px;
  }

  .home-eyebrow {
    font-size: 15px;
  }

  .home-hero h1 {
    max-width: 358px;
    font-size: clamp(30px, 8vw, 32px);
    line-height: 1.06;
    text-wrap: initial;
  }

  .home-hero h1.home-hero__title--cjk {
    font-size: clamp(36px, 10.2vw, 40px);
  }

  .home-hero__lead {
    max-width: 340px;
    font-size: 17px;
    line-height: 1.42;
  }

  .home-hero__meta {
    max-width: 340px;
    margin-top: 16px;
  }

  .home-claim-list {
    justify-content: center;
    gap: 8px;
    margin-top: 24px;
  }

  .home-claim-list li {
    min-height: 30px;
    flex: 0 1 auto;
    justify-content: center;
    font-size: 12px;
    line-height: 1.2;
    padding: 0 10px;
  }

  .home-hero__visual {
    margin-top: 30px;
  }

  .home-product-panel__top {
    min-height: 0;
    padding: 20px;
  }

  .home-product-panel__top strong {
    max-width: 230px;
    font-size: 26px;
  }

  .home-product-panel__top b {
    min-height: 26px;
    font-size: 11px;
    padding: 0 8px;
  }

  .home-product-rows {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .home-product-row {
    min-height: 132px;
    grid-template-columns: 1fr;
    gap: 14px;
    padding: 18px;
  }

  .home-product-row:nth-child(2n) {
    border-right: 0;
  }

  .home-product-row:nth-last-child(-n + 2) {
    border-bottom: 0;
  }

  .home-product-row > div {
    gap: 12px;
  }

  .home-product-row strong {
    display: block;
    font-size: 18px;
  }

  .home-product-row em {
    font-size: 12px;
    line-height: 1.38;
  }

  .home-product-metrics {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .home-product-metrics div {
    min-height: 78px;
    border-right: 1px solid var(--home-border-soft);
    padding: 16px 14px;
  }

  .home-product-metrics div:last-child {
    border-right: 0;
  }

  .home-product-metrics span {
    font-size: 11px;
  }

  .home-product-metrics strong {
    margin-top: 10px;
    font-size: 24px;
  }

  .home-band--light,
  .home-band--final,
  .home-band--split,
  .home-band--dark {
    gap: 38px;
    padding: 68px 16px;
  }

  .home-section-copy h2 {
    font-size: 36px;
  }

  .home-section-copy p:not(.home-eyebrow) {
    font-size: 16px;
  }

  .home-model-lineup {
    grid-template-columns: 1fr;
    margin-top: 38px;
  }

  .home-model-lineup__item {
    min-height: 0;
    border-right: 0;
    padding: 18px 18px 20px;
  }

  .home-model-lineup__item:nth-last-child(-n + 2) {
    border-bottom: 1px solid var(--home-border-soft);
  }

  .home-model-lineup__item:last-child {
    border-bottom: 0;
  }

  .home-model-lineup__item strong {
    margin-top: 16px;
    font-size: 22px;
  }

  .home-ledger__row {
    min-height: 78px;
    grid-template-columns: 1fr;
    align-content: center;
    gap: 4px;
    padding: 14px 18px;
  }

  .home-ledger__row strong {
    font-size: 20px;
    white-space: normal;
  }

  .home-usage-visual {
    min-height: 300px;
  }

  .home-usage-visual__summary {
    top: 24px;
    left: 24px;
  }

  .home-usage-visual__summary strong {
    font-size: 44px;
  }

  .home-usage-visual__bars {
    right: 24px;
    bottom: 36px;
    left: 24px;
    height: 166px;
    gap: 8px;
  }

  .home-footer > div {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
