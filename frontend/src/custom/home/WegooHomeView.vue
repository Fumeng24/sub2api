<template>
  <div v-if="hasHomeContent" class="custom-home-content min-h-screen">
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

  <!-- Compact Home Page: opt-in setting, default homepage remains unchanged. -->
  <div v-else-if="compactHomeEnabled" data-testid="compact-home" class="compact-home">
    <header class="compact-home__header">
      <div class="compact-home__nav">
        <div class="compact-home__brand">
          <img :src="siteLogo || '/logo.png'" alt="" />
          <span>{{ siteName }}</span>
        </div>
        <div class="compact-home__actions">
          <button
            type="button"
            :aria-label="isDark ? copy.useLightTheme : copy.useDarkTheme"
            :title="isDark ? copy.useLightTheme : copy.useDarkTheme"
            @click="toggleTheme"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
          </button>
          <LocaleSwitcher />
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            :title="copy.docs"
            @click="handleDocsLinkClick"
          >
            <Icon name="book" size="sm" />
          </a>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'">
            {{ isAuthenticated ? copy.console : copy.signIn }}
          </router-link>
        </div>
      </div>
    </header>
    <main class="compact-home__main">
      <img :src="siteLogo || '/logo.png'" alt="" class="compact-home__logo" />
      <h1>{{ siteName }}</h1>
      <p>{{ siteSubtitle || copy.heroLead }}</p>
      <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="compact-home__cta">
        {{ isAuthenticated ? copy.console : copy.getStarted }}
        <Icon name="arrowRight" size="sm" />
      </router-link>
    </main>
    <footer class="compact-home__footer">&copy; {{ currentYear }} {{ siteName }}</footer>
  </div>

  <div v-else class="gateway-home">
    <header class="gateway-nav">
      <nav class="gateway-nav__inner" :aria-label="copy.navLabel">
        <router-link to="/home" class="gateway-brand" :aria-label="siteName">
          <span class="gateway-brand__mark">
            <img :src="siteLogo || '/logo.png'" alt="" />
          </span>
          <span class="gateway-brand__text">
            <strong>{{ siteName }}</strong>
            <em>AI Gateway</em>
          </span>
        </router-link>

        <div class="gateway-nav__links">
          <router-link v-for="item in navItems" :key="item.to" :to="item.to">{{ item.label }}</router-link>
          <a href="#faq">{{ copy.faq }}</a>
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            @click="handleDocsLinkClick"
          >
            {{ copy.docs }}
          </a>
          <router-link v-else to="/docs">{{ copy.docs }}</router-link>
          <template v-for="item in customNavItems" :key="item.key">
            <a
              v-if="item.external"
              :href="item.href"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ item.label }}
            </a>
            <router-link v-else :to="item.route || item.href">
              {{ item.label }}
            </router-link>
          </template>
        </div>

        <div class="gateway-nav__actions">
          <button
            type="button"
            class="gateway-icon-button"
            :aria-label="isDark ? copy.useLightTheme : copy.useDarkTheme"
            :title="isDark ? copy.useLightTheme : copy.useDarkTheme"
            @click="toggleTheme"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
          </button>
          <LocaleSwitcher />
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="gateway-link-button">
            {{ isAuthenticated ? copy.console : copy.signIn }}
          </router-link>
          <router-link :to="isAuthenticated ? '/keys' : '/register'" class="gateway-primary-button gateway-nav-cta">
            {{ copy.getStarted }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section class="gateway-hero" aria-labelledby="gateway-hero-title">
        <div class="gateway-hero__content">
          <p class="gateway-badge">
            <span></span>
            AI Gateway · Multi-model API
          </p>
          <h1 id="gateway-hero-title">{{ siteName }}</h1>
          <p class="gateway-hero__statement">
            <span>{{ copy.heroTitle }}</span>{{ ' ' }}
            <strong>{{ copy.heroAccent }}</strong>
          </p>
          <p class="gateway-hero__lead">{{ copy.heroLead }}</p>

          <div class="gateway-hero__actions">
            <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="gateway-primary-button">
              {{ copy.openConsole }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <a
              v-if="docsLink"
              class="gateway-secondary-button"
              :href="docsLink.href"
              :target="docsLink.external ? '_blank' : undefined"
              :rel="docsLink.external ? 'noopener noreferrer' : undefined"
              @click="handleDocsLinkClick"
            >
              {{ copy.viewDocs }}
            </a>
            <router-link v-else to="/docs" class="gateway-secondary-button">
              {{ copy.viewDocs }}
            </router-link>
          </div>

          <div class="gateway-capability-strip" :aria-label="copy.capabilities">
            <span v-for="item in capabilityPills" :key="item">{{ item }}</span>
          </div>
        </div>

        <div class="gateway-hero__visual" aria-label="API request preview">
          <div class="gateway-code-window gateway-code-window--hero">
            <div class="gateway-code-window__bar">
              <span></span><span></span><span></span>
              <em>curl /v1/chat/completions</em>
            </div>
            <pre><code>{{ heroCode }}</code></pre>
          </div>
          <div class="gateway-hero-metrics">
            <div v-for="metric in heroMetrics" :key="metric.label">
              <span>{{ metric.label }}</span>
              <strong>{{ metric.value }}</strong>
            </div>
          </div>
        </div>
      </section>

      <section id="promises" class="gateway-section gateway-section--compact">
        <div class="gateway-section__heading">
          <p class="gateway-kicker">{{ copy.promiseKicker }}</p>
          <h2>{{ copy.promiseTitle }}</h2>
          <p>{{ copy.promiseLead }}</p>
        </div>
        <div class="gateway-promise-grid">
          <article v-for="item in promises" :key="item.title" class="gateway-card gateway-promise-card">
            <span class="gateway-card__icon">
              <Icon :name="item.icon" size="md" />
            </span>
            <h3>{{ item.title }}</h3>
            <p>{{ item.body }}</p>
          </article>
        </div>
      </section>

      <section id="routing" class="gateway-section gateway-section--flow">
        <div class="gateway-section__heading gateway-section__heading--narrow">
          <p class="gateway-kicker">{{ copy.flowKicker }}</p>
          <h2>{{ copy.flowTitle }}</h2>
          <p>{{ copy.flowLead }}</p>
        </div>
        <div class="gateway-flow" aria-label="Gateway request flow">
          <article v-for="node in flowNodes" :key="node.title" class="gateway-flow__node">
            <span>
              <Icon :name="node.icon" size="md" />
            </span>
            <h3>{{ node.title }}</h3>
            <p>{{ node.body }}</p>
          </article>
        </div>
      </section>

      <section id="pricing" class="gateway-section">
        <div class="gateway-section__heading">
          <p class="gateway-kicker">{{ copy.pricingKicker }}</p>
          <h2>{{ copy.pricingTitle }}</h2>
          <p>{{ copy.pricingLead }}</p>
        </div>
        <div class="gateway-pricing-grid">
          <article v-for="item in pricingCards" :key="item.name" class="gateway-card gateway-pricing-card">
            <div>
              <span class="gateway-status-dot"></span>
              <em>{{ item.family }}</em>
            </div>
            <h3>{{ item.name }}</h3>
            <dl>
              <div>
                <dt>{{ copy.input }}</dt>
                <dd>{{ item.input }}</dd>
              </div>
              <div>
                <dt>{{ copy.output }}</dt>
                <dd>{{ item.output }}</dd>
              </div>
            </dl>
            <p>{{ item.endpoints }}</p>
          </article>
        </div>
      </section>

      <section id="cache" class="gateway-section gateway-cache">
        <div class="gateway-cache__copy">
          <p class="gateway-kicker">{{ copy.cacheKicker }}</p>
          <h2>{{ copy.cacheTitle }}</h2>
          <p>{{ copy.cacheLead }}</p>
        </div>
        <div class="gateway-cache__panel">
          <div v-for="item in cacheRows" :key="item.label" class="gateway-cache__row">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <i :style="{ width: item.width }"></i>
          </div>
        </div>
      </section>

      <section id="docs" class="gateway-section gateway-code-section">
        <div class="gateway-section__heading">
          <p class="gateway-kicker">{{ copy.codeKicker }}</p>
          <h2>{{ copy.codeTitle }}</h2>
          <p>{{ copy.codeLead }}</p>
        </div>

        <div class="gateway-code-tabs">
          <button
            v-for="tab in codeTabs"
            :key="tab.id"
            type="button"
            :class="{ active: activeCodeTab === tab.id }"
            @click="activeCodeTab = tab.id"
          >
            {{ tab.label }}
          </button>
        </div>

        <div class="gateway-code-window">
          <div class="gateway-code-window__bar">
            <span></span><span></span><span></span>
            <em>{{ selectedCodeTab.label }}</em>
            <button type="button" @click="copyCode(selectedCodeTab.code)">
              <Icon name="copy" size="sm" />
              {{ copy.copy }}
            </button>
          </div>
          <pre><code>{{ selectedCodeTab.code }}</code></pre>
        </div>
      </section>

      <section id="status" class="gateway-section gateway-status-section">
        <div class="gateway-status-section__header">
          <div class="gateway-section__heading gateway-section__heading--narrow">
            <p class="gateway-kicker">{{ copy.statusKicker }}</p>
            <h2>{{ copy.statusTitle }}</h2>
            <p>{{ copy.statusLead }}</p>
          </div>
          <router-link to="/status" class="gateway-secondary-button gateway-status-link">
            {{ copy.statusViewAll }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>

        <div v-if="statusLoading" class="gateway-status-message" role="status" aria-live="polite">
          <Icon name="refresh" size="sm" />
          {{ copy.statusLoading }}
        </div>
        <div v-else-if="statusError" class="gateway-status-message gateway-status-message--error" role="alert">
          <span>{{ copy.statusLoadFailed }}</span>
          <button type="button" @click="loadHomepageStatus">{{ copy.statusRetry }}</button>
        </div>
        <div v-else-if="statusItems.length === 0" class="gateway-status-message" role="status">
          {{ copy.statusEmpty }}
        </div>
        <div v-else class="gateway-status-grid" data-testid="homepage-live-status">
          <article v-for="item in statusItems" :key="item.id" class="gateway-card gateway-status-card">
            <div>
              <span :class="['gateway-health', `is-${item.tone}`]"></span>
              <strong>{{ item.name }}</strong>
            </div>
            <p>{{ item.status }} · {{ item.latency }} · {{ item.availability }}</p>
            <div v-if="item.history.length" class="gateway-history" :aria-label="copy.statusHistory">
              <span
                v-for="(status, index) in item.history"
                :key="`${item.id}-${index}`"
                :class="`is-${status}`"
                :title="statusLabel(status)"
              ></span>
            </div>
            <p v-else class="gateway-history-empty">{{ copy.statusNoHistory }}</p>
          </article>
        </div>
      </section>

      <section id="faq" class="gateway-section gateway-faq">
        <div class="gateway-section__heading">
          <p class="gateway-kicker">FAQ</p>
          <h2>{{ copy.faqTitle }}</h2>
        </div>
        <div class="gateway-faq__list">
          <details v-for="item in faqs" :key="item.question">
            <summary>{{ item.question }}</summary>
            <p>{{ item.answer }}</p>
          </details>
        </div>
      </section>
    </main>

    <footer class="gateway-footer">
      <div>
        <router-link to="/home" class="gateway-brand gateway-brand--footer">
          <span class="gateway-brand__mark">
            <img :src="siteLogo || '/logo.png'" alt="" />
          </span>
          <span class="gateway-brand__text">
            <strong>{{ siteName }}</strong>
            <em>{{ siteSubtitle || 'AI Gateway' }}</em>
          </span>
        </router-link>
        <p>&copy; {{ currentYear }} {{ siteName }}</p>
      </div>
      <div>
        <router-link to="/pricing">{{ copy.models }}</router-link>
        <router-link to="/status">{{ copy.serviceStatus }}</router-link>
        <router-link to="/enterprise">{{ copy.enterprise }}</router-link>
        <a href="#faq">{{ copy.faq }}</a>
        <a
          v-if="docsLink"
          :href="docsLink.href"
          :target="docsLink.external ? '_blank' : undefined"
          :rel="docsLink.external ? 'noopener noreferrer' : undefined"
          @click="handleDocsLinkClick"
        >
          {{ copy.docs }}
        </a>
        <router-link v-else to="/docs">{{ copy.docs }}</router-link>
        <template v-for="item in customNavItems" :key="`footer-${item.key}`">
          <a
            v-if="item.external"
            :href="item.href"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ item.label }}
          </a>
          <router-link v-else :to="item.route || item.href">
            {{ item.label }}
          </router-link>
        </template>
        <router-link :to="isAuthenticated ? dashboardPath : '/login'">{{ copy.console }}</router-link>
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
import type { MonitorStatus, UserMonitorView } from '@/api/channelMonitor'
import { useAppStore, useAuthStore } from '@/stores'
import publicGatewayAPI from '@/custom/api/publicGateway'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/custom/utils/docsLink'
import { resolvePublicApiBaseUrls } from '@/custom/utils/publicApiBaseUrl'
import { resolvePublicCustomNavigationItems } from '@/custom/utils/publicNavigation'
import { sanitizeUrl } from '@/utils/url'

type IconName = InstanceType<typeof Icon>['$props']['name']

const { locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Wegoo AI')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const sanitizedHomeContent = computed(() => DOMPurify.sanitize(homeContent.value))
const rawDocUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const sanitizedDocUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const docsLink = computed(() => resolveDocsLink(sanitizedDocUrl.value || rawDocUrl.value, appStore.cachedPublicSettings?.custom_menu_items ?? []))
const isHomeContentUrl = computed(() => /^https?:\/\//i.test(homeContent.value.trim()))
const customNavItems = computed(() => resolvePublicCustomNavigationItems(appStore.cachedPublicSettings?.custom_menu_items ?? []))

const activeCodeTab = ref('openai')
const isDark = ref(false)
const statusRows = ref<UserMonitorView[]>([])
const statusLoading = ref(true)
const statusError = ref(false)
let statusRequestController: AbortController | null = null

const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const currentYear = computed(() => new Date().getFullYear())
const isZh = computed(() => locale.value.toLowerCase().startsWith('zh'))

const copy = computed(() => isZh.value ? zhCopy : enCopy)

const navItems = computed(() => [
  { to: '/pricing', label: copy.value.models },
  { to: '/status', label: copy.value.serviceStatus },
  { to: '/enterprise', label: copy.value.enterprise }
])

const apiBaseUrls = computed(() => resolvePublicApiBaseUrls(appStore.apiBaseUrl))

const capabilityPills = [
  'OpenAI SDK',
  'Anthropic SDK',
  'Claude Code',
  'Codex CLI',
  'Prompt Cache',
  'Streaming',
  'Tool Use',
  'Image API'
]

const heroCode = computed(() => `curl ${apiBaseUrls.value.v1}/chat/completions \\
  -H "Authorization: Bearer $WEGOO_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.5",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'`)

const heroMetrics = computed(() => [
  { label: copy.value.metricOne, value: '1 Key' },
  { label: copy.value.metricTwo, value: 'GPT / Claude / Gemini' },
  { label: copy.value.metricThree, value: '0 Body Logs' }
])

const promises = computed<Array<{ icon: IconName; title: string; body: string }>>(() => [
  { icon: 'swap', title: copy.value.promiseRawTitle, body: copy.value.promiseRawBody },
  { icon: 'shield', title: copy.value.promisePrivacyTitle, body: copy.value.promisePrivacyBody },
  { icon: 'cube', title: copy.value.promiseModelTitle, body: copy.value.promiseModelBody },
  { icon: 'database', title: copy.value.promiseBillingTitle, body: copy.value.promiseBillingBody }
])

const flowNodes = computed<Array<{ icon: IconName; title: string; body: string }>>(() => [
  { icon: 'terminal', title: 'Your App', body: copy.value.flowApp },
  { icon: 'server', title: 'Wegoo Gateway', body: copy.value.flowGateway },
  { icon: 'brain', title: 'Upstream Models', body: copy.value.flowUpstream }
])

const pricingCards = computed(() => [
  { family: 'OpenAI', name: 'Codex Pro / GPT', input: copy.value.priceMetered, output: copy.value.priceMetered, endpoints: '/v1/chat/completions · /v1/responses' },
  { family: 'Anthropic', name: 'Claude Max', input: copy.value.priceMetered, output: copy.value.priceMetered, endpoints: '/v1/messages · Claude Code' },
  { family: 'Google', name: 'Gemini', input: copy.value.priceMetered, output: copy.value.priceMetered, endpoints: 'OpenAI-compatible · native bridge' },
  { family: 'Images', name: 'GPT Image', input: copy.value.imageInput, output: copy.value.imageOutput, endpoints: '/v1/images/generations' }
])

const cacheRows = computed(() => [
  { label: 'Cache Write', value: copy.value.cacheWrite, width: '72%' },
  { label: 'Cache Read', value: copy.value.cacheRead, width: '92%' },
  { label: 'Saved Cost', value: copy.value.cacheSaved, width: '64%' }
])

const codeTabs = computed(() => [
  {
    id: 'openai',
    label: 'OpenAI SDK',
    code: `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.WEGOO_API_KEY,
  baseURL: "${apiBaseUrls.value.v1}"
});

const stream = await client.chat.completions.create({
  model: "gpt-5.5",
  messages: [{ role: "user", content: "Hello" }],
  stream: true
});`
  },
  {
    id: 'anthropic',
    label: 'Anthropic SDK',
    code: `import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic({
  apiKey: process.env.WEGOO_API_KEY,
  baseURL: "${apiBaseUrls.value.root}"
});

await client.messages.create({
  model: "claude-sonnet-4-6",
  max_tokens: 1024,
  messages: [{ role: "user", content: "Hello" }]
});`
  },
  { id: 'curl', label: 'curl', code: heroCode.value },
  {
    id: 'codex',
    label: 'Codex CLI',
    code: `export OPENAI_API_KEY="$WEGOO_API_KEY"
export OPENAI_BASE_URL="${apiBaseUrls.value.v1}"

codex --model gpt-5.5`
  }
])

const selectedCodeTab = computed(() => codeTabs.value.find((tab) => tab.id === activeCodeTab.value) ?? codeTabs.value[0])

const statusItems = computed(() => statusRows.value.slice(0, 4).map((item) => ({
  id: item.id,
  name: item.name,
  tone: item.primary_status,
  status: statusLabel(item.primary_status),
  latency: typeof item.primary_latency_ms === 'number'
    ? `${Math.round(item.primary_latency_ms)}ms`
    : copy.value.statusNoLatency,
  availability: `${formatAvailability(item.availability_7d)} / 7d`,
  history: (item.timeline || []).slice(-30).map((point) => point.status),
})))

const faqs = computed(() => [
  { question: copy.value.faqWhat, answer: copy.value.faqWhatAnswer },
  { question: copy.value.faqRewrite, answer: copy.value.faqRewriteAnswer },
  { question: copy.value.faqSdk, answer: copy.value.faqSdkAnswer },
  { question: copy.value.faqBilling, answer: copy.value.faqBillingAnswer },
  { question: copy.value.faqInvoice, answer: copy.value.faqInvoiceAnswer }
])

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
  } catch {
    // Clipboard access can be unavailable in insecure contexts.
  }
}

function formatAvailability(value: number): string {
  if (!Number.isFinite(value)) return copy.value.statusNoData
  return `${value.toFixed(value >= 99.95 ? 0 : 1)}%`
}

function statusLabel(status: MonitorStatus): string {
  const labels: Record<MonitorStatus, string> = {
    operational: copy.value.statusOperational,
    degraded: copy.value.statusDegraded,
    failed: copy.value.statusFailed,
    error: copy.value.statusError,
  }
  return labels[status]
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

async function loadHomepageStatus() {
  statusRequestController?.abort()
  const controller = new AbortController()
  statusRequestController = controller
  statusLoading.value = true
  statusError.value = false

  try {
    const response = await publicGatewayAPI.getPublicChannelMonitors({ signal: controller.signal })
    if (!controller.signal.aborted) {
      statusRows.value = response.items || []
    }
  } catch {
    if (!controller.signal.aborted) {
      statusRows.value = []
      statusError.value = true
    }
  } finally {
    if (statusRequestController === controller) {
      statusRequestController = null
      statusLoading.value = false
    }
  }
}

onMounted(() => {
  isDark.value = document.documentElement.classList.contains('dark')
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  if (!hasHomeContent.value && !compactHomeEnabled.value) {
    loadHomepageStatus()
  }
})

onUnmounted(() => {
  statusRequestController?.abort()
  statusRequestController = null
})

const zhCopy = {
  navLabel: '公开站点导航',
  docs: '接入教程',
  models: '模型价格',
  routing: '透明转发',
  serviceStatus: '服务状态',
  enterprise: '企业服务',
  faq: 'FAQ',
  console: '控制台',
  signIn: '登录',
  getStarted: '开始使用',
  heroTitle: '一个 Key，接入',
  heroAccent: '全部主流模型',
  heroLead: 'Wegoo AI 为开发者提供统一 API 网关，支持 GPT/Codex、Claude、Gemini 与 AI 生图。按量计费，透明记录，不改写请求。',
  openConsole: '打开控制台',
  viewDocs: '查看接入教程',
  capabilities: '平台能力',
  metricOne: '统一凭证',
  metricTwo: '模型族',
  metricThree: '请求体业务日志',
  promiseKicker: '平台边界',
  promiseTitle: '开发者需要的确定性。',
  promiseLead: '请求、模型、账单和隐私边界都清楚，避免把 API 网关做成黑盒。',
  promiseRawTitle: '原样转发',
  promiseRawBody: '按配置路由请求，不注入、不裁剪、不静默替换模型。',
  promisePrivacyTitle: '隐私边界',
  promisePrivacyBody: '请求体和响应体不作为业务日志展示，后台只保留必要计费记录。',
  promiseModelTitle: '多模型统一',
  promiseModelBody: '一个 Key 调 GPT、Claude、Gemini、生图和常见开发者工具。',
  promiseBillingTitle: '透明账单',
  promiseBillingBody: '按请求记录 token、模型、费用、延迟和分组，方便对账。',
  flowKicker: '请求链路',
  flowTitle: '你的应用到上游模型，中间只做网关该做的事。',
  flowLead: '鉴权、调度、计费和状态记录留在平台，响应流直接返回客户端。',
  flowApp: 'SDK、curl、LangChain、Codex CLI 或 Claude Code 发起请求。',
  flowGateway: '校验 Key、选择分组、调度账号、记录费用和延迟。',
  flowUpstream: '请求进入 GPT、Claude、Gemini 或图像模型，响应流回传。',
  pricingKicker: '价格预览',
  pricingTitle: '按量计费，完整价格以后台配置为准。',
  pricingLead: '首页只展示代表模型族。控制台内可查看完整模型、倍率、可用分组和状态。',
  input: '输入',
  output: '输出',
  priceMetered: '按模型价',
  imageInput: '按图片规格',
  imageOutput: '1 USD / 张',
  cacheKicker: 'Prompt Cache',
  cacheTitle: '长上下文和多轮任务，缓存命中后成本更清楚。',
  cacheLead: '适合 Claude Code、Codex、长文档和 Agent 场景。Cache Write / Read 独立记录，账单可追溯。',
  cacheWrite: '写入计费',
  cacheRead: '命中降本',
  cacheSaved: '可审计',
  codeKicker: '快速接入',
  codeTitle: '沿用熟悉的 SDK 和命令行。',
  codeLead: '把 Base URL 指向 Wegoo AI，保留 OpenAI / Anthropic 常用调用习惯。',
  copy: '复制',
  statusKicker: '实时状态',
  statusTitle: '实时服务状态，以后台监控为准。',
  statusLead: '这里展示最新可用性、延迟和真实检查记录，不暴露敏感账号信息。',
  statusLowLatency: '低延迟可用',
  statusOperational: '运行正常',
  statusDegraded: '服务波动',
  statusFailed: '暂不可用',
  statusError: '监控异常',
  statusNoLatency: '暂无延迟',
  statusNoData: '暂无数据',
  statusLoading: '正在读取实时服务状态…',
  statusLoadFailed: '实时状态暂时无法读取，请稍后重试。',
  statusRetry: '重试',
  statusEmpty: '服务状态正在建立。',
  statusNoHistory: '近期趋势正在建立',
  statusHistory: '近期真实监控记录',
  statusViewAll: '查看全部状态',
  useLightTheme: '切换到浅色模式',
  useDarkTheme: '切换到深色模式',
  faqTitle: '常见问题',
  faqWhat: 'Wegoo AI 是什么？',
  faqWhatAnswer: '一个面向开发者的多模型 API 网关，用统一 Key 接入 GPT、Claude、Gemini 和图像模型。',
  faqRewrite: '平台会改写请求吗？',
  faqRewriteAnswer: '目标是透明转发。除鉴权、调度、计费和必要兼容处理外，不做静默模型替换。',
  faqSdk: '支持哪些 SDK？',
  faqSdkAnswer: '支持 OpenAI SDK、Anthropic SDK、curl、Codex CLI、Claude Code 等常见调用方式。',
  faqBilling: '如何计费？',
  faqBillingAnswer: '按模型、token、缓存和分组倍率生成用量记录，余额和订单可在控制台查看。',
  faqInvoice: '支持发票吗？',
  faqInvoiceAnswer: '支持符合条件的有效充值记录申请发票，赠送余额和邀请返佣不计入可开票金额。'
}

const enCopy = {
  navLabel: 'Public site navigation',
  docs: 'Docs',
  models: 'Pricing',
  routing: 'Gateway',
  serviceStatus: 'Status',
  enterprise: 'Enterprise',
  faq: 'FAQ',
  console: 'Console',
  signIn: 'Sign in',
  getStarted: 'Get started',
  heroTitle: 'One key for',
  heroAccent: 'all leading models',
  heroLead: 'Wegoo AI gives developers a unified API gateway for GPT/Codex, Claude, Gemini, and image generation. Metered billing, transparent records, and request passthrough.',
  openConsole: 'Open console',
  viewDocs: 'View docs',
  capabilities: 'Platform capabilities',
  metricOne: 'Credential',
  metricTwo: 'Model families',
  metricThree: 'Body business logs',
  promiseKicker: 'Platform boundary',
  promiseTitle: 'The certainty developers need.',
  promiseLead: 'Requests, models, billing, and privacy boundaries stay visible instead of becoming a gateway black box.',
  promiseRawTitle: 'Passthrough',
  promiseRawBody: 'Route by configuration without injection, trimming, or silent model replacement.',
  promisePrivacyTitle: 'Privacy boundary',
  promisePrivacyBody: 'Request and response bodies are not exposed as business logs; only billing records are retained.',
  promiseModelTitle: 'Multi-model access',
  promiseModelBody: 'Use one key for GPT, Claude, Gemini, image APIs, and developer tools.',
  promiseBillingTitle: 'Transparent billing',
  promiseBillingBody: 'Record tokens, model, cost, latency, and group for every request.',
  flowKicker: 'Request path',
  flowTitle: 'From your app to upstream models, the gateway only does gateway work.',
  flowLead: 'Authentication, routing, billing, and status stay on the platform. Response streams return to the client.',
  flowApp: 'SDKs, curl, LangChain, Codex CLI, or Claude Code send requests.',
  flowGateway: 'Validate keys, select groups, schedule accounts, and record cost and latency.',
  flowUpstream: 'Requests reach GPT, Claude, Gemini, or image models and stream back.',
  pricingKicker: 'Pricing preview',
  pricingTitle: 'Metered billing. Full pricing follows backend configuration.',
  pricingLead: 'The home page shows representative model families. The console exposes full models, rates, groups, and status.',
  input: 'Input',
  output: 'Output',
  priceMetered: 'By model rate',
  imageInput: 'By image size',
  imageOutput: '1 USD / image',
  cacheKicker: 'Prompt Cache',
  cacheTitle: 'Long-context and multi-turn workloads get clearer cost with cache hits.',
  cacheLead: 'Built for Claude Code, Codex, long documents, and agent workloads. Cache Write / Read are tracked separately.',
  cacheWrite: 'Write billed',
  cacheRead: 'Read savings',
  cacheSaved: 'Auditable',
  codeKicker: 'Quick start',
  codeTitle: 'Keep the SDKs and CLIs you already use.',
  codeLead: 'Point the Base URL to Wegoo AI and keep familiar OpenAI / Anthropic calling patterns.',
  copy: 'Copy',
  statusKicker: 'Live status',
  statusTitle: 'Live service status from backend monitoring.',
  statusLead: 'See current availability, latency, and real checks without exposing sensitive account details.',
  statusLowLatency: 'Low latency',
  statusOperational: 'Operational',
  statusDegraded: 'Degraded',
  statusFailed: 'Unavailable',
  statusError: 'Monitor error',
  statusNoLatency: 'No latency data',
  statusNoData: 'No data',
  statusLoading: 'Loading live service status…',
  statusLoadFailed: 'Live status is temporarily unavailable. Please retry shortly.',
  statusRetry: 'Retry',
  statusEmpty: 'Service status is being established.',
  statusNoHistory: 'Recent history is being established',
  statusHistory: 'Recent live monitor history',
  statusViewAll: 'View all status',
  useLightTheme: 'Switch to light theme',
  useDarkTheme: 'Switch to dark theme',
  faqTitle: 'Frequently asked questions',
  faqWhat: 'What is Wegoo AI?',
  faqWhatAnswer: 'A developer-focused multi-model API gateway for GPT, Claude, Gemini, and image generation with one API key.',
  faqRewrite: 'Does the platform rewrite requests?',
  faqRewriteAnswer: 'The goal is transparent passthrough. Apart from auth, routing, billing, and compatibility handling, there is no silent model replacement.',
  faqSdk: 'Which SDKs are supported?',
  faqSdkAnswer: 'OpenAI SDK, Anthropic SDK, curl, Codex CLI, Claude Code, and common API clients are supported.',
  faqBilling: 'How does billing work?',
  faqBillingAnswer: 'Usage records are generated from model, tokens, cache, and group rates. Balance and orders are visible in the console.',
  faqInvoice: 'Are invoices supported?',
  faqInvoiceAnswer: 'Eligible paid recharge records can be invoiced. Gift balance and referral rebates are excluded.'
}
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
.compact-home {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  background: #f9fafb;
  color: #111827;
}

.compact-home__header {
  border-bottom: 1px solid #e5e7eb;
  padding: 14px 20px;
}

.compact-home__nav {
  display: flex;
  width: min(100%, 960px);
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin: 0 auto;
}

.compact-home__brand,
.compact-home__actions,
.compact-home__cta {
  display: flex;
  min-width: 0;
  align-items: center;
}

.compact-home__brand {
  flex: 1;
  gap: 10px;
  font-size: 15px;
  font-weight: 650;
}

.compact-home__brand img {
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 8px;
  object-fit: contain;
}

.compact-home__brand span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compact-home__actions {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.compact-home__actions > a,
.compact-home__actions > button {
  display: inline-flex;
  min-width: 38px;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 8px 12px;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
}

.compact-home__main {
  display: flex;
  width: min(100% - 32px, 680px);
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin: 0 auto;
  padding: 64px 0;
  text-align: center;
}

.compact-home__logo {
  width: 80px;
  height: 80px;
  margin-bottom: 24px;
  border-radius: 8px;
  object-fit: contain;
}

.compact-home__main h1 {
  max-width: 100%;
  margin: 0;
  overflow-wrap: anywhere;
  font-size: 36px;
  font-weight: 720;
  line-height: 1.2;
}

.compact-home__main p {
  max-width: 620px;
  margin: 16px 0 0;
  color: #64748b;
  font-size: 15px;
  line-height: 1.7;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.compact-home__cta {
  min-height: 42px;
  justify-content: center;
  gap: 8px;
  margin-top: 30px;
  border-radius: 8px;
  background: #0d9488;
  padding: 10px 18px;
  color: #ffffff;
  font-size: 14px;
  font-weight: 700;
}

.compact-home__footer {
  border-top: 1px solid #e5e7eb;
  padding: 18px 20px;
  color: #94a3b8;
  font-size: 13px;
  overflow-wrap: anywhere;
  text-align: center;
}

:global(.dark .compact-home) {
  background: #020617;
  color: #ffffff;
}

:global(.dark .compact-home__header),
:global(.dark .compact-home__footer) {
  border-color: #1e293b;
}

:global(.dark .compact-home__actions > a),
:global(.dark .compact-home__actions > button) {
  border-color: #334155;
  color: #cbd5e1;
}

:global(.dark .compact-home__main p),
:global(.dark .compact-home__footer) {
  color: #94a3b8;
}

@media (max-width: 640px) {
  .compact-home__nav {
    align-items: flex-start;
  }

  .compact-home__actions {
    max-width: 58%;
  }

  .compact-home__main h1 {
    font-size: 30px;
  }
}

.gateway-home {
  --surface: #f9fafb;
  --surface-secondary: #f8fafc;
  --surface-elevated: #f3f4f6;
  --surface-card: #ffffff;
  --surface-card-solid: #ffffff;
  --surface-card-hover: #f8fafc;
  --surface-overlay: #ffffff;
  --text-primary: #111827;
  --text-secondary: #64748b;
  --text-tertiary: #94a3b8;
  --text-muted: #cbd5e1;
  --outline: #e5e7eb;
  --outline-light: #eef2f7;
  --outline-medium: #d1d5db;
  --outline-hover: #99f6e4;
  --accent: #0d9488;
  --accent-hover: #0f766e;
  --accent-muted: #f0fdfa;
  --accent-secondary: #2563eb;
  --accent-secondary-muted: #eff6ff;
  --success: #10b981;
  min-height: 100vh;
  overflow-x: hidden;
  background: var(--surface);
  color: var(--text-primary);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
}

:global(.dark .gateway-home) {
  --surface: #020617;
  --surface-secondary: #0f172a;
  --surface-elevated: #1e293b;
  --surface-card: #0f172a;
  --surface-card-solid: #0f172a;
  --surface-card-hover: #182235;
  --surface-overlay: #0f172a;
  --text-primary: #f8fafc;
  --text-secondary: #cbd5e1;
  --text-tertiary: #94a3b8;
  --text-muted: #64748b;
  --outline: #334155;
  --outline-light: #1e293b;
  --outline-medium: #475569;
  --outline-hover: #0f766e;
  --accent: #0d9488;
  --accent-hover: #14b8a6;
  --accent-muted: rgb(20 184 166 / 15%);
  --accent-secondary: #60a5fa;
  --accent-secondary-muted: rgb(59 130 246 / 15%);
}

.gateway-home,
.gateway-home * {
  box-sizing: border-box;
}

.gateway-nav {
  position: sticky;
  top: 0;
  z-index: 40;
  border-bottom: 1px solid var(--outline-light);
  background: color-mix(in srgb, var(--surface-card-solid) 92%, transparent);
  backdrop-filter: blur(14px);
}

.gateway-nav__inner {
  display: flex;
  height: 64px;
  max-width: 1180px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin: 0 auto;
  padding: 0 22px;
}

.gateway-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  color: var(--text-primary);
}

.gateway-brand__mark {
  display: inline-flex;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--outline-medium);
  border-radius: 8px;
  background: var(--surface-elevated);
}

.gateway-brand__mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.gateway-brand__text {
  display: grid;
  min-width: 0;
  line-height: 1.05;
}

.gateway-brand__text strong {
  max-width: 180px;
  overflow: hidden;
  font-size: 14px;
  font-weight: 720;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gateway-brand__text em {
  margin-top: 3px;
  color: var(--text-tertiary);
  font-size: 11px;
  font-style: normal;
  letter-spacing: 0;
}

.gateway-nav__links,
.gateway-nav__actions {
  display: flex;
  align-items: center;
}

.gateway-nav__links {
  gap: 26px;
}

.gateway-nav__links a,
.gateway-footer a {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 560;
  letter-spacing: 0;
  transition: color 0.18s ease;
}

.gateway-nav__links a:hover,
.gateway-footer a:hover {
  color: var(--text-primary);
}

.gateway-nav__actions {
  flex: 0 0 auto;
  gap: 8px;
}

.gateway-icon-button,
.gateway-link-button,
.gateway-primary-button,
.gateway-secondary-button {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: 8px;
  font-weight: 650;
  letter-spacing: 0;
  transition: background-color 0.18s ease, border-color 0.18s ease, color 0.18s ease, transform 0.18s ease;
}

.gateway-icon-button {
  width: 36px;
  height: 36px;
  color: var(--text-secondary);
}

.gateway-icon-button:hover {
  background: var(--surface-elevated);
  color: var(--text-primary);
}

.gateway-link-button,
.gateway-secondary-button {
  min-height: 38px;
  border: 1px solid var(--outline);
  background: var(--surface-card);
  color: var(--text-secondary);
  font-size: 13px;
  padding: 0 15px;
}

.gateway-link-button:hover,
.gateway-secondary-button:hover {
  border-color: var(--outline-hover);
  background: var(--surface-elevated);
  color: var(--text-primary);
}

.gateway-primary-button {
  min-height: 42px;
  border: 1px solid var(--accent);
  background: var(--accent);
  color: #ffffff;
  font-size: 14px;
  padding: 0 18px;
  box-shadow: 0 1px 2px rgb(15 23 42 / 10%);
}

.gateway-primary-button:hover {
  border-color: var(--accent-hover);
  background: var(--accent-hover);
}

.gateway-nav-cta {
  min-height: 38px;
  font-size: 13px;
  padding: 0 15px;
}

.gateway-hero {
  display: grid;
  max-width: 1180px;
  min-height: 620px;
  grid-template-columns: minmax(0, 0.94fr) minmax(360px, 0.82fr);
  align-items: center;
  gap: 56px;
  margin: 0 auto;
  padding: 72px 22px 56px;
}

.gateway-badge {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  border: 1px solid var(--outline);
  border-radius: 8px;
  background: var(--accent-muted);
  color: var(--accent-hover);
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 0;
  margin: 0;
  padding: 8px 12px;
}

.gateway-badge span,
.gateway-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--success);
}

.gateway-hero h1 {
  max-width: 720px;
  margin: 22px 0 0;
  color: var(--text-primary);
  font-size: clamp(44px, 5vw, 64px);
  font-weight: 750;
  letter-spacing: 0;
  line-height: 1.05;
  overflow-wrap: anywhere;
  text-wrap: balance;
}

.gateway-hero__statement {
  max-width: 680px;
  margin: 18px 0 0;
  color: var(--text-primary);
  font-size: clamp(24px, 2.6vw, 34px);
  font-weight: 700;
  line-height: 1.2;
}

.gateway-hero__statement strong {
  color: var(--accent);
}

.gateway-hero__lead {
  max-width: 660px;
  color: var(--text-secondary);
  font-size: 16px;
  font-weight: 400;
  line-height: 1.62;
  margin: 18px 0 0;
  text-wrap: balance;
}

.gateway-hero__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 30px;
}

.gateway-capability-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 28px;
}

.gateway-capability-strip span {
  border: 1px solid var(--outline);
  border-radius: 6px;
  background: var(--surface-card);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 620;
  padding: 7px 10px;
}

.gateway-hero__visual {
  display: grid;
  gap: 14px;
}

.gateway-code-window {
  overflow: hidden;
  border: 1px solid #334155;
  border-radius: 8px;
  background: #0f172a;
  box-shadow: 0 10px 30px rgb(15 23 42 / 14%);
}

.gateway-code-window--hero {
  transform: translateY(8px);
}

.gateway-code-window__bar {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid #334155;
  background: #111c30;
  padding: 0 14px;
}

.gateway-code-window__bar span {
  width: 10px;
  height: 10px;
  flex: 0 0 auto;
  border-radius: 999px;
}

.gateway-code-window__bar span:nth-child(1) {
  background: #e8a87c;
}

.gateway-code-window__bar span:nth-child(2) {
  background: var(--gw-gold);
}

.gateway-code-window__bar span:nth-child(3) {
  background: #2dd4bf;
}

.gateway-code-window__bar em {
  min-width: 0;
  overflow: hidden;
  color: #94a3b8;
  font-size: 12px;
  font-style: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gateway-code-window__bar button {
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  gap: 6px;
  border: 1px solid #475569;
  border-radius: 6px;
  color: #cbd5e1;
  font-size: 12px;
  margin-left: auto;
  padding: 0 10px;
}

.gateway-code-window pre {
  overflow-x: auto;
  margin: 0;
  padding: 22px;
}

.gateway-code-window code {
  color: #e2e8f0;
  font-family: "JetBrains Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 13px;
  line-height: 1.75;
  white-space: pre;
}

.gateway-hero-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.gateway-hero-metrics div {
  border: 1px solid var(--outline);
  border-radius: 8px;
  background: var(--surface-card);
  padding: 14px;
}

.gateway-hero-metrics span,
.gateway-hero-metrics strong {
  display: block;
}

.gateway-hero-metrics span {
  color: var(--text-tertiary);
  font-size: 11px;
  font-weight: 620;
}

.gateway-hero-metrics strong {
  color: var(--text-primary);
  font-size: 14px;
  margin-top: 8px;
}

.gateway-section {
  max-width: 1180px;
  margin: 0 auto;
  padding: 68px 22px;
}

.gateway-section--compact {
  padding-top: 42px;
}

.gateway-section__heading {
  max-width: 760px;
  margin-bottom: 34px;
}

.gateway-section__heading--narrow {
  max-width: 680px;
}

.gateway-kicker {
  color: var(--accent-secondary);
  font-size: 12px;
  font-weight: 760;
  letter-spacing: 0;
  margin: 0 0 12px;
  text-transform: uppercase;
}

.gateway-section h2,
.gateway-cache__copy h2 {
  color: var(--text-primary);
  font-size: clamp(30px, 3.4vw, 42px);
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.08;
  margin: 0;
  text-wrap: balance;
}

.gateway-section__heading > p:not(.gateway-kicker),
.gateway-cache__copy > p:not(.gateway-kicker) {
  color: var(--text-secondary);
  font-size: 15px;
  line-height: 1.65;
  margin: 16px 0 0;
}

.gateway-promise-grid,
.gateway-pricing-grid,
.gateway-status-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.gateway-card {
  border: 1px solid var(--outline-light);
  border-radius: 8px;
  background: var(--surface-card);
  box-shadow: 0 1px 2px rgb(15 23 42 / 5%);
  transition: transform 0.18s ease, border-color 0.18s ease, background-color 0.18s ease;
}

.gateway-card:hover {
  border-color: var(--outline-hover);
  background: var(--surface-card-hover);
}

.gateway-promise-card {
  min-height: 232px;
  padding: 22px;
}

.gateway-card__icon {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--accent-muted);
  color: var(--accent);
}

.gateway-card h3 {
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 720;
  letter-spacing: 0;
  line-height: 1.25;
  margin: 18px 0 0;
}

.gateway-card p {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.62;
  margin: 12px 0 0;
}

.gateway-section--flow {
  position: relative;
}

.gateway-flow {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
}

.gateway-flow::before {
  position: absolute;
  top: 50%;
  right: 12%;
  left: 12%;
  height: 1px;
  background: var(--outline-medium);
  content: "";
}

.gateway-flow__node {
  position: relative;
  z-index: 1;
  min-height: 210px;
  border: 1px solid var(--outline);
  border-radius: 8px;
  background: var(--surface-overlay);
  padding: 24px;
}

.gateway-flow__node span {
  display: inline-flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--accent-secondary-muted);
  color: var(--accent-secondary);
}

.gateway-flow__node h3 {
  margin: 18px 0 0;
  font-size: 20px;
  font-weight: 720;
}

.gateway-flow__node p {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.62;
  margin: 10px 0 0;
}

.gateway-pricing-card {
  min-height: 236px;
  padding: 22px;
}

.gateway-pricing-card > div:first-child {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.gateway-pricing-card em {
  color: var(--text-tertiary);
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
}

.gateway-pricing-card dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 18px 0 0;
}

.gateway-pricing-card dl div {
  border: 1px solid var(--outline-light);
  border-radius: 6px;
  background: var(--surface-secondary);
  padding: 12px;
}

.gateway-pricing-card dt {
  color: var(--text-tertiary);
  font-size: 11px;
  font-weight: 700;
}

.gateway-pricing-card dd {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 720;
  margin: 6px 0 0;
}

.gateway-cache {
  display: grid;
  grid-template-columns: minmax(0, 0.92fr) minmax(320px, 0.86fr);
  align-items: center;
  gap: 48px;
}

.gateway-cache__panel {
  border: 1px solid var(--outline);
  border-radius: 8px;
  background: var(--surface-card);
  padding: 22px;
}

.gateway-cache__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  border-bottom: 1px solid var(--outline-light);
  padding: 18px 0;
}

.gateway-cache__row:first-child {
  padding-top: 0;
}

.gateway-cache__row:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.gateway-cache__row span {
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 650;
}

.gateway-cache__row strong {
  color: var(--text-primary);
  font-size: 14px;
}

.gateway-cache__row i {
  grid-column: 1 / -1;
  height: 8px;
  border-radius: 4px;
  background: var(--accent);
}

.gateway-code-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.gateway-code-tabs button {
  min-height: 36px;
  border: 1px solid var(--outline);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 650;
  padding: 0 14px;
}

.gateway-code-tabs button.active {
  border-color: var(--outline-hover);
  background: var(--accent-muted);
  color: var(--accent);
}

.gateway-status-card {
  padding: 18px;
}

.gateway-status-section__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 32px;
}

.gateway-status-section__header .gateway-section__heading {
  margin-bottom: 0;
}

.gateway-status-link {
  flex: 0 0 auto;
}

.gateway-status-message {
  display: flex;
  min-height: 96px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--outline);
  border-radius: 8px;
  background: var(--surface-card);
  color: var(--text-secondary);
  font-size: 14px;
  padding: 20px;
  text-align: center;
}

.gateway-status-message--error {
  border-color: color-mix(in srgb, #dc2626 32%, var(--outline));
  color: #b91c1c;
}

:global(.dark .gateway-status-message--error) {
  color: #fca5a5;
}

.gateway-status-message button {
  border: 1px solid currentColor;
  border-radius: 6px;
  padding: 5px 10px;
  font-weight: 650;
}

.gateway-status-card > div:first-child {
  display: flex;
  align-items: center;
  gap: 10px;
}

.gateway-status-card strong {
  font-size: 15px;
  font-weight: 720;
}

.gateway-health {
  width: 9px;
  height: 9px;
  border-radius: 999px;
}

.gateway-health.is-operational {
  background: var(--success);
}

.gateway-health.is-degraded {
  background: #d97706;
}

.gateway-health.is-failed,
.gateway-health.is-error {
  background: #dc2626;
}

.gateway-history {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(3px, 1fr));
  gap: 3px;
  margin-top: 18px;
}

.gateway-history span {
  height: 26px;
  border-radius: 4px;
}

.gateway-history span.is-operational {
  background: color-mix(in srgb, var(--success) 72%, var(--surface-card));
}

.gateway-history span.is-degraded {
  background: color-mix(in srgb, #d97706 72%, var(--surface-card));
}

.gateway-history span.is-failed,
.gateway-history span.is-error {
  background: color-mix(in srgb, #dc2626 72%, var(--surface-card));
}

.gateway-history-empty {
  margin-top: 18px !important;
  color: var(--text-tertiary) !important;
  font-size: 12px !important;
}

.gateway-faq__list {
  display: grid;
  gap: 10px;
}

.gateway-faq details {
  border: 1px solid var(--outline);
  border-radius: 8px;
  background: var(--surface-card);
  padding: 18px 20px;
}

.gateway-faq summary {
  cursor: pointer;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0;
}

.gateway-faq details p {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.65;
  margin: 12px 0 0;
}

.gateway-footer {
  display: flex;
  max-width: 1180px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  border-top: 1px solid var(--outline-light);
  margin: 0 auto;
  padding: 28px 22px 38px;
}

.gateway-footer > div {
  display: flex;
  align-items: center;
  gap: 18px;
}

.gateway-footer p {
  color: var(--text-tertiary);
  font-size: 12px;
  margin: 0;
}

.gateway-brand--footer .gateway-brand__mark {
  width: 30px;
  height: 30px;
}

@media (max-width: 1040px) {
  .gateway-nav__links {
    display: none;
  }

  .gateway-hero,
  .gateway-cache {
    grid-template-columns: 1fr;
  }

  .gateway-hero {
    min-height: auto;
    padding-top: 64px;
  }

  .gateway-promise-grid,
  .gateway-pricing-grid,
  .gateway-status-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .gateway-nav__inner {
    height: 58px;
    padding: 0 14px;
  }

  .gateway-nav-cta,
  .gateway-link-button {
    display: none;
  }

  .gateway-brand__text strong {
    max-width: 124px;
  }

  .gateway-hero,
  .gateway-section {
    padding-right: 16px;
    padding-left: 16px;
  }

  .gateway-hero h1 {
    font-size: clamp(40px, 11vw, 52px);
  }

  .gateway-hero__lead {
    font-size: 16px;
  }

  .gateway-hero-metrics,
  .gateway-promise-grid,
  .gateway-pricing-grid,
  .gateway-status-grid,
  .gateway-flow {
    grid-template-columns: 1fr;
  }

  .gateway-status-section__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .gateway-flow::before {
    display: none;
  }

  .gateway-code-window pre {
    padding: 18px;
  }

  .gateway-code-window code {
    font-size: 12px;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }

  .gateway-footer,
  .gateway-footer > div {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
