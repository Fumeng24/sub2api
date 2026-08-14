<template>
  <AppLayout>
    <div class="custom-page-view">
      <UserPageHero
        :kicker="t('customPage.gateway.kicker')"
        :title="pageTitle"
        :description="t('customPage.gateway.description')"
      />

      <section
        class="custom-page-panel"
        :class="{
          'custom-page-panel--markdown': isMarkdownMode,
          'custom-page-panel--embedded': isValidUrl,
        }"
      >
        <div v-if="loading" class="custom-state custom-state--loading">
          <div class="custom-spinner" aria-hidden="true"></div>
          <p class="custom-state-copy">{{ loadingLabel }}</p>
        </div>

        <div
          v-else-if="!menuItem"
          class="custom-state"
        >
          <div class="custom-state-inner">
            <div class="custom-state-icon">
              <Icon name="document" size="lg" />
            </div>
            <h2 class="custom-state-title">
              {{ unavailableTitle }}
            </h2>
            <p class="custom-state-copy">
              {{ unavailableDesc }}
            </p>
          </div>
        </div>

        <div v-else-if="isMarkdownMode" class="markdown-shell">
          <aside
            v-show="tocVisible && tocItems.length > 0"
            class="toc-sidebar"
          >
            <div class="toc-header">
              <span class="toc-title">{{ tocLabel }}</span>
              <button
                class="toc-close-btn"
                type="button"
                :aria-label="collapseTocLabel"
                :title="collapseTocLabel"
                @click="tocVisible = false"
              >
                <Icon name="chevronLeft" size="sm" :stroke-width="2" />
              </button>
            </div>
            <nav class="toc-nav">
              <a
                v-for="item in tocItems"
                :key="item.id"
                :href="'#' + item.id"
                class="toc-item"
                :class="[
                  `toc-level-${item.level}`,
                  { 'toc-active': activeHeadingId === item.id }
                ]"
                @click.prevent="scrollToHeading(item.id)"
              >
                {{ item.text }}
              </a>
            </nav>
          </aside>

          <button
            v-show="!tocVisible && tocItems.length > 0"
            class="toc-toggle-btn"
            type="button"
            :aria-label="openTocLabel"
            :title="openTocLabel"
            @click="tocVisible = true"
          >
            <Icon name="menu" size="sm" :stroke-width="2" />
          </button>

          <div
            ref="markdownContainer"
            class="markdown-page-content"
            v-html="renderedHtml"
            @scroll="onContentScroll"
          ></div>
        </div>

        <div v-else-if="!isValidUrl" class="custom-state">
          <div class="custom-state-inner">
            <div class="custom-state-icon">
              <Icon name="link" size="lg" />
            </div>
            <h2 class="custom-state-title">
              {{ unavailableTitle }}
            </h2>
            <p class="custom-state-copy">
              {{ unavailableDesc }}
            </p>
          </div>
        </div>

        <div v-else class="custom-embed-shell">
          <a
            :href="embeddedUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="custom-open-fab"
          >
            <Icon name="externalLink" size="sm" :stroke-width="2" />
            {{ t('customPage.openInNewTab') }}
          </a>
          <iframe
            :src="embeddedUrl"
            class="custom-embed-frame"
            allowfullscreen
          ></iframe>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import { buildApiUrl } from '@/api/client'
import { buildEmbeddedUrl, detectTheme } from '@/custom/utils/embedded-url'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

interface TocItem {
  id: string
  text: string
  level: number
}

const { t, locale } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()

const loading = ref(false)
const pageTheme = ref<'light' | 'dark'>('light')
const renderedHtml = ref('')
const markdownContainer = ref<HTMLElement | null>(null)
const tocItems = ref<TocItem[]>([])
const tocVisible = ref(typeof window !== 'undefined' ? window.innerWidth > 768 : true)
const activeHeadingId = ref('')
let themeObserver: MutationObserver | null = null

const menuItemId = computed(() => route.params.id as string)

const menuItem = computed(() => {
  const id = menuItemId.value
  const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
  const found = publicItems.find((item) => item.id === id) ?? null
  if (found) return found
  if (authStore.isAdmin) {
    return adminSettingsStore.customMenuItems.find((item) => item.id === id) ?? null
  }
  return null
})

const pageTitle = computed(() => menuItem.value?.label || t('customPage.defaultTitle'))
const tocLabel = computed(() => t('customPage.toc'))
const openTocLabel = computed(() => t('customPage.openToc'))
const collapseTocLabel = computed(() => t('customPage.collapseToc'))
const loadingLabel = computed(() => t('customPage.loading'))
const unavailableTitle = computed(() => t('customPage.unavailableTitle'))
const unavailableDesc = computed(() => t('customPage.unavailableDesc'))

const markdownSlug = computed(() => {
  const item = menuItem.value
  if (!item) return ''
  if (item.page_slug) return item.page_slug
  if (item.url?.startsWith('md:')) return item.url.slice(3)
  return ''
})

const isMarkdownMode = computed(() => !!markdownSlug.value)

const embeddedUrl = computed(() => {
  if (!menuItem.value || isMarkdownMode.value) return ''
  return buildEmbeddedUrl(
    menuItem.value.url,
    authStore.user?.id,
    authStore.token,
    pageTheme.value,
    locale.value,
  )
})

const isValidUrl = computed(() => {
  if (isMarkdownMode.value) return false
  const url = embeddedUrl.value
  return url.startsWith('http://') || url.startsWith('https://')
})

function generateHeadingId(text: string, index: number): string {
  const base = text
    .toLowerCase()
    .replace(/[^\w一-鿿]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return base ? `${base}-${index}` : `heading-${index}`
}

function isRelativeMarkdownAsset(src: string): boolean {
  const trimmed = src.trim()
  if (!trimmed || /^[a-z][a-z0-9+.-]*:/i.test(trimmed) || trimmed.startsWith('//') || trimmed.startsWith('/')) {
    return false
  }
  const [pathPart] = trimmed.split(/([?#].*)/, 2)
  return pathPart
    .split('/')
    .filter((part) => part && part !== '.')
    .every((part) => part !== '..' && !part.includes('\\'))
}

function buildPageImageUrl(slug: string, src: string): string {
  const trimmed = src.trim()
  const [pathPart, suffix = ''] = trimmed.split(/([?#].*)/, 2)
  const encodedPath = pathPart
    .split('/')
    .filter((part) => part && part !== '.')
    .map((part) => encodeURIComponent(part))
    .join('/')
  return buildApiUrl(`/pages/${encodeURIComponent(slug)}/images/${encodedPath}${suffix}`)
}

function markdownStateHtml(kind: 'notFound' | 'loadFailed'): string {
  const title = kind === 'notFound'
    ? t('customPage.unavailableTitle')
    : t('customPage.markdownLoadFailedTitle')
  const description = kind === 'notFound'
    ? t('customPage.markdownNotFoundDesc')
    : t('customPage.markdownLoadFailedDesc')
  return `<div class="markdown-inline-state"><p class="markdown-inline-state-title">${title}</p><p>${description}</p></div>`
}

async function fetchAndRenderMarkdown(slug: string) {
  loading.value = true
  tocItems.value = []
  activeHeadingId.value = ''
  try {
    const resp = await fetch(buildApiUrl(`/pages/${encodeURIComponent(slug)}`), {
      headers: authStore.token ? { Authorization: `Bearer ${authStore.token}` } : {},
    })
    if (!resp.ok) {
      renderedHtml.value = markdownStateHtml('notFound')
      return
    }
    let raw = await resp.text()

    raw = raw.replace(
      /!\[([^\]]*)\]\(([^)]+)\)/g,
      (match, alt, src) => isRelativeMarkdownAsset(src) ? `![${alt}](${buildPageImageUrl(slug, src)})` : match
    )

    const html = marked.parse(raw) as string
    const sanitized = DOMPurify.sanitize(html, {
      ADD_TAGS: ['iframe'],
      ADD_ATTR: ['allowfullscreen', 'frameborder', 'src'],
    })

    // Inject IDs into headings and build TOC
    const toc: TocItem[] = []
    let headingIndex = 0
    const withIds = sanitized.replace(
      /<(h[1-4])[^>]*>(.*?)<\/h[1-4]>/gi,
      (_, tag: string, content: string) => {
        const level = parseInt(tag[1])
        const text = content.replace(/<[^>]+>/g, '').trim()
        const id = generateHeadingId(text, headingIndex++)
        toc.push({ id, text, level })
        return `<${tag} id="${id}">${content}</${tag}>`
      }
    )

    renderedHtml.value = withIds
    tocItems.value = toc
  } catch {
    renderedHtml.value = markdownStateHtml('loadFailed')
  } finally {
    loading.value = false
    await nextTick()
    await nextTick()
    injectCopyButtons()
  }
}

function scrollToHeading(id: string) {
  const container = markdownContainer.value
  if (!container) return
  const el = container.querySelector(`#${CSS.escape(id)}`)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeHeadingId.value = id
    if (window.innerWidth <= 640) {
      tocVisible.value = false
    }
  }
}

let scrollRafId = 0
function onContentScroll() {
  if (scrollRafId) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = 0
    const container = markdownContainer.value
    if (!container || tocItems.value.length === 0) return

    const containerRect = container.getBoundingClientRect()
    let current = ''

    for (const item of tocItems.value) {
      const el = container.querySelector(`#${CSS.escape(item.id)}`) as HTMLElement | null
      if (el) {
        const elRect = el.getBoundingClientRect()
        if (elRect.top - containerRect.top <= 100) {
          current = item.id
        }
      }
    }
    activeHeadingId.value = current
  })
}

function injectCopyButtons() {
  const container = markdownContainer.value
  if (!container) return

  container.querySelectorAll('pre').forEach((pre) => {
    if (pre.querySelector('.copy-btn')) return
    const btn = document.createElement('button')
    btn.className = 'copy-btn'
    btn.textContent = t('customPage.copyCode')
    btn.addEventListener('click', async () => {
      const code = pre.querySelector('code')?.textContent ?? pre.textContent ?? ''
      try {
        await navigator.clipboard.writeText(code)
        btn.textContent = t('customPage.copiedCode')
        setTimeout(() => { btn.textContent = t('customPage.copyCode') }, 2000)
      } catch {
        btn.textContent = t('customPage.copyFailed')
        setTimeout(() => { btn.textContent = t('customPage.copyCode') }, 2000)
      }
    })
    pre.style.position = 'relative'
    pre.appendChild(btn)
  })
}

watch(markdownSlug, (slug) => {
  if (slug) {
    fetchAndRenderMarkdown(slug)
  } else {
    renderedHtml.value = ''
    tocItems.value = []
  }
}, { immediate: true })

onMounted(async () => {
  pageTheme.value = detectTheme()

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      pageTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }

  if (appStore.publicSettingsLoaded) return
  loading.value = true
  try {
    await appStore.fetchPublicSettings()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.custom-page-view {
  --custom-bg: var(--apple-bg, #f5f5f7);
  --custom-surface: var(--apple-surface, #ffffff);
  --custom-surface-elevated: var(--apple-surface-elevated, #fbfbfd);
  --custom-text: var(--apple-text, #1d1d1f);
  --custom-muted: var(--apple-muted, #6e6e73);
  --custom-muted-2: var(--apple-muted-2, #86868b);
  --custom-border: var(--apple-border, rgb(0 0 0 / 10%));
  --custom-border-soft: var(--apple-border-soft, rgb(0 0 0 / 6%));
  --custom-blue: var(--apple-blue, #0071e3);
  --custom-blue-hover: var(--apple-blue-hover, #0077ed);
  --custom-hover: var(--apple-hover, rgb(0 0 0 / 4%));
  --custom-focus-ring: var(--apple-focus-ring, rgb(0 113 227 / 28%));
  --custom-shadow-sm: var(--apple-shadow-sm, 0 1px 2px rgb(0 0 0 / 4%));
  --custom-shadow-md: var(--apple-shadow-md, 0 10px 24px rgb(0 0 0 / 8%));
  --custom-radius: var(--apple-radius, 8px);

  display: flex;
  min-height: calc(100vh - 64px - 3rem);
  min-height: calc(100dvh - 64px - 3rem);
  flex-direction: column;
  gap: 1rem;
  color: var(--custom-text);
}

:global(.dark) .custom-page-view {
  --custom-bg: var(--apple-bg, #000000);
  --custom-surface: var(--apple-surface, #161617);
  --custom-surface-elevated: var(--apple-surface-elevated, #1d1d1f);
  --custom-text: var(--apple-text, #f5f5f7);
  --custom-muted: var(--apple-muted, #a1a1a6);
  --custom-muted-2: var(--apple-muted-2, #86868b);
  --custom-border: var(--apple-border, rgb(255 255 255 / 12%));
  --custom-border-soft: var(--apple-border-soft, rgb(255 255 255 / 8%));
  --custom-blue: var(--apple-blue, #2997ff);
  --custom-blue-hover: var(--apple-blue-hover, #6bbcff);
  --custom-hover: var(--apple-hover, rgb(255 255 255 / 8%));
  --custom-focus-ring: var(--apple-focus-ring, rgb(41 151 255 / 34%));
  --custom-shadow-sm: var(--apple-shadow-sm, 0 1px 2px rgb(0 0 0 / 18%));
  --custom-shadow-md: var(--apple-shadow-md, 0 10px 28px rgb(0 0 0 / 32%));
}

.custom-page-panel {
  position: relative;
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
  border: 1px solid var(--custom-border);
  border-radius: var(--custom-radius);
  background: var(--custom-surface);
  box-shadow: var(--custom-shadow-sm);
}

.custom-page-panel--markdown {
  background: var(--custom-surface);
}

.custom-page-panel--embedded {
  background: var(--custom-surface);
}

.custom-state {
  display: flex;
  min-height: min(560px, calc(100dvh - 12rem));
  width: 100%;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  text-align: center;
}

.custom-state-inner {
  max-width: 28rem;
}

.custom-state-icon {
  display: inline-flex;
  height: 3rem;
  width: 3rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--custom-border-soft);
  border-radius: var(--custom-radius);
  background: var(--custom-surface-elevated);
  color: var(--custom-muted);
}

.custom-state-title {
  margin: 1rem 0 0;
  color: var(--custom-text);
  font-size: 1.125rem;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.4;
}

.custom-state-copy {
  margin: 0.5rem auto 0;
  max-width: 28rem;
  color: var(--custom-muted);
  font-size: 0.875rem;
  line-height: 1.7;
}

.custom-state--loading {
  min-height: min(480px, calc(100dvh - 12rem));
  flex-direction: column;
}

.custom-spinner {
  height: 2rem;
  width: 2rem;
  border: 2px solid var(--custom-border);
  border-top-color: var(--custom-blue);
  border-radius: 999px;
  animation: custom-page-spin 0.75s linear infinite;
}

@keyframes custom-page-spin {
  to {
    transform: rotate(360deg);
  }
}

.markdown-shell {
  position: relative;
  display: flex;
  min-height: 0;
  width: 100%;
  overflow: hidden;
}

.toc-sidebar {
  display: flex;
  width: clamp(12rem, 22vw, 17rem);
  min-width: 12rem;
  height: 100%;
  flex-direction: column;
  border-right: 1px solid var(--custom-border-soft);
  background: var(--custom-surface-elevated);
  overflow: hidden;
}

.toc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-bottom: 1px solid var(--custom-border-soft);
  padding: 0.875rem 1rem;
}

.toc-title {
  color: var(--custom-text);
  font-size: 0.8125rem;
  font-weight: 600;
  letter-spacing: 0;
}

.toc-close-btn {
  display: inline-flex;
  height: 1.75rem;
  width: 1.75rem;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--custom-radius);
  background: transparent;
  color: var(--custom-muted);
  cursor: pointer;
  transition: background-color 0.16s ease, color 0.16s ease;
}

.toc-close-btn:hover {
  background: var(--custom-hover);
  color: var(--custom-text);
}

.toc-close-btn:focus-visible,
.toc-toggle-btn:focus-visible,
.custom-open-fab:focus-visible {
  outline: 2px solid var(--custom-focus-ring);
  outline-offset: 2px;
}

.toc-nav {
  flex: 1 1 auto;
  overflow-y: auto;
  padding: 0.5rem;
}

.toc-item {
  display: block;
  overflow: hidden;
  border-radius: var(--custom-radius);
  padding: 0.45rem 0.625rem;
  color: var(--custom-muted);
  font-size: 0.8125rem;
  line-height: 1.45;
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: background-color 0.16s ease, color 0.16s ease;
}

.toc-item:hover {
  background: var(--custom-hover);
  color: var(--custom-text);
}

.toc-item.toc-active {
  background: color-mix(in srgb, var(--custom-blue) 10%, var(--custom-surface));
  color: var(--custom-blue);
  font-weight: 600;
}

.toc-level-1 { padding-left: 0.625rem; }
.toc-level-2 { padding-left: 1rem; }
.toc-level-3 { padding-left: 1.375rem; }
.toc-level-4 { padding-left: 1.75rem; }

.toc-toggle-btn {
  position: absolute;
  left: 0.875rem;
  top: 0.875rem;
  z-index: 10;
  display: inline-flex;
  height: 2.25rem;
  width: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--custom-border);
  border-radius: var(--custom-radius);
  background: color-mix(in srgb, var(--custom-surface) 92%, transparent);
  color: var(--custom-muted);
  box-shadow: var(--custom-shadow-sm);
  cursor: pointer;
  transition: background-color 0.16s ease, color 0.16s ease;
  backdrop-filter: saturate(180%) blur(16px);
}

.toc-toggle-btn:hover {
  background: var(--custom-surface-elevated);
  color: var(--custom-text);
}

.custom-embed-shell {
  position: relative;
  height: 100%;
  min-height: min(720px, calc(100dvh - 12rem));
  width: 100%;
  overflow: hidden;
  border-radius: var(--custom-radius);
  background: var(--custom-surface);
}

.custom-open-fab {
  position: absolute;
  right: 0.875rem;
  top: 0.875rem;
  z-index: 10;
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  min-height: 2.25rem;
  max-width: calc(100% - 1.75rem);
  border: 1px solid var(--custom-border);
  border-radius: var(--custom-radius);
  background: color-mix(in srgb, var(--custom-surface) 88%, transparent);
  padding: 0.45rem 0.75rem;
  color: var(--custom-text);
  font-size: 0.8125rem;
  font-weight: 500;
  line-height: 1.2;
  text-decoration: none;
  box-shadow: var(--custom-shadow-sm);
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;
  backdrop-filter: saturate(180%) blur(16px);
}

.custom-open-fab:hover {
  background: var(--custom-surface-elevated);
  border-color: var(--custom-border);
  color: var(--custom-blue);
}

.custom-embed-frame {
  display: block;
  margin: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}

@media (max-width: 768px) {
  .custom-page-view {
    min-height: calc(100vh - 64px - 2rem);
    min-height: calc(100dvh - 64px - 2rem);
    gap: 0.75rem;
  }

  .custom-page-panel {
    min-height: calc(100vh - 64px - 7rem);
    min-height: calc(100dvh - 64px - 7rem);
  }

  .toc-sidebar {
    position: absolute;
    inset: 0 auto 0 0;
    z-index: 20;
    width: min(82vw, 20rem);
    max-width: none;
    border-right: 1px solid var(--custom-border);
    box-shadow: var(--custom-shadow-md);
  }

  .custom-state {
    min-height: calc(100vh - 64px - 8rem);
    min-height: calc(100dvh - 64px - 8rem);
    padding: 1.5rem;
  }

  .custom-embed-shell {
    min-height: calc(100vh - 64px - 8rem);
    min-height: calc(100dvh - 64px - 8rem);
  }

  .custom-open-fab {
    right: 0.625rem;
    top: 0.625rem;
    max-width: calc(100% - 1.25rem);
  }
}
</style>

<style>
.markdown-page-content {
  --markdown-bg: var(--apple-bg, #f5f5f7);
  --markdown-surface: var(--apple-surface, #ffffff);
  --markdown-surface-elevated: var(--apple-surface-elevated, #fbfbfd);
  --markdown-text: var(--apple-text, #1d1d1f);
  --markdown-muted: var(--apple-muted, #6e6e73);
  --markdown-muted-2: var(--apple-muted-2, #86868b);
  --markdown-border: var(--apple-border, rgb(0 0 0 / 10%));
  --markdown-border-soft: var(--apple-border-soft, rgb(0 0 0 / 6%));
  --markdown-blue: var(--apple-blue, #0071e3);
  --markdown-blue-hover: var(--apple-blue-hover, #0077ed);
  --markdown-hover: var(--apple-hover, rgb(0 0 0 / 4%));
  --markdown-radius: var(--apple-radius, 8px);

  height: 100%;
  flex: 1 1 auto;
  overflow: auto;
  padding: clamp(1.5rem, 3vw, 3rem);
  color: var(--markdown-text);
  font-size: 1rem;
  line-height: 1.75;
}

.dark .markdown-page-content {
  --markdown-bg: var(--apple-bg, #000000);
  --markdown-surface: var(--apple-surface, #161617);
  --markdown-surface-elevated: var(--apple-surface-elevated, #1d1d1f);
  --markdown-text: var(--apple-text, #f5f5f7);
  --markdown-muted: var(--apple-muted, #a1a1a6);
  --markdown-muted-2: var(--apple-muted-2, #86868b);
  --markdown-border: var(--apple-border, rgb(255 255 255 / 12%));
  --markdown-border-soft: var(--apple-border-soft, rgb(255 255 255 / 8%));
  --markdown-blue: var(--apple-blue, #2997ff);
  --markdown-blue-hover: var(--apple-blue-hover, #6bbcff);
  --markdown-hover: var(--apple-hover, rgb(255 255 255 / 8%));
}

.markdown-page-content > :first-child {
  margin-top: 0;
}

.markdown-page-content > :last-child {
  margin-bottom: 0;
}

.markdown-page-content h1,
.markdown-page-content h2,
.markdown-page-content h3,
.markdown-page-content h4 {
  color: var(--markdown-text);
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.25;
  scroll-margin-top: 1.25rem;
}

.markdown-page-content h1 {
  margin: 0 0 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--markdown-border-soft);
  font-size: clamp(1.875rem, 4vw, 2.75rem);
}

.markdown-page-content h2 {
  margin: 2.25rem 0 0.875rem;
  font-size: clamp(1.45rem, 2.8vw, 2rem);
}

.markdown-page-content h3 {
  margin: 1.75rem 0 0.75rem;
  font-size: 1.25rem;
}

.markdown-page-content h4 {
  margin: 1.5rem 0 0.625rem;
  font-size: 1.0625rem;
}

.markdown-page-content p {
  margin: 0 0 1rem;
  color: var(--markdown-text);
}

.markdown-page-content ul,
.markdown-page-content ol {
  margin: 0 0 1.25rem;
  padding-left: 1.4rem;
}

.markdown-page-content li {
  margin: 0.35rem 0;
  padding-left: 0.15rem;
}

.markdown-page-content li::marker {
  color: var(--markdown-muted-2);
}

.markdown-page-content a {
  color: var(--markdown-blue);
  text-decoration: none;
}

.markdown-page-content a:hover {
  color: var(--markdown-blue-hover);
  text-decoration: underline;
  text-underline-offset: 0.18em;
}

.markdown-page-content blockquote {
  margin: 1.5rem 0;
  border-left: 2px solid var(--markdown-blue);
  padding: 0.15rem 0 0.15rem 1rem;
  color: var(--markdown-muted);
}

.markdown-page-content blockquote p {
  color: inherit;
}

.markdown-page-content img {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 1.5rem 0;
  border-radius: var(--markdown-radius);
}

.markdown-page-content table {
  width: 100%;
  margin: 1.5rem 0;
  border-collapse: separate;
  border-spacing: 0;
  overflow: hidden;
  border: 1px solid var(--markdown-border);
  border-radius: var(--markdown-radius);
  font-size: 0.875rem;
}

.markdown-page-content th,
.markdown-page-content td {
  border-bottom: 1px solid var(--markdown-border-soft);
  padding: 0.75rem 0.875rem;
  text-align: left;
  vertical-align: top;
}

.markdown-page-content th {
  background: var(--markdown-surface-elevated);
  color: var(--markdown-muted);
  font-weight: 600;
}

.markdown-page-content tr:last-child td {
  border-bottom: 0;
}

.markdown-page-content code {
  border: 1px solid var(--markdown-border-soft);
  border-radius: 6px;
  background: var(--markdown-surface-elevated);
  padding: 0.1rem 0.35rem;
  color: var(--markdown-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.88em;
}

.markdown-page-content pre {
  position: relative;
  margin: 1.5rem 0;
  overflow-x: auto;
  border: 1px solid var(--markdown-border);
  border-radius: var(--markdown-radius);
  background: #0b0b0f;
  padding: 1rem;
  color: #f5f5f7;
}

.markdown-page-content pre code {
  border: 0;
  background: transparent;
  padding: 0;
  color: inherit;
  font-size: 0.875rem;
}

.markdown-page-content hr {
  margin: 2rem 0;
  border: 0;
  border-top: 1px solid var(--markdown-border-soft);
}

.markdown-page-content iframe {
  display: block;
  max-width: 100%;
  margin: 1.5rem 0;
  border: 1px solid var(--markdown-border);
  border-radius: var(--markdown-radius);
  background: var(--markdown-surface-elevated);
}

.markdown-inline-state {
  max-width: 32rem;
  margin: 12vh auto 0;
  border: 1px solid var(--markdown-border);
  border-radius: var(--markdown-radius);
  background: var(--markdown-surface-elevated);
  padding: 1.25rem;
  text-align: center;
}

.markdown-inline-state-title {
  margin-bottom: 0.35rem;
  color: var(--markdown-text);
  font-weight: 600;
}

.markdown-inline-state p {
  color: var(--markdown-muted);
}

.markdown-page-content .copy-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  border: 1px solid rgb(255 255 255 / 18%);
  border-radius: 6px;
  background: rgb(255 255 255 / 10%);
  padding: 4px 9px;
  color: #f5f5f7;
  font-size: 12px;
  font-family: inherit;
  line-height: 1.2;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.16s ease, background-color 0.16s ease;
}

.markdown-page-content .copy-btn:hover {
  background: rgb(255 255 255 / 18%);
}

.markdown-page-content pre:hover .copy-btn,
.markdown-page-content .copy-btn:focus-visible {
  opacity: 1;
}

@media (max-width: 768px) {
  .markdown-page-content {
    padding: 3.5rem 1.125rem 1.5rem;
    font-size: 0.9375rem;
  }

  .markdown-page-content h1 {
    font-size: 1.75rem;
  }

  .markdown-page-content h2 {
    font-size: 1.375rem;
  }

  .markdown-page-content table {
    display: block;
    overflow-x: auto;
  }

  .markdown-page-content .copy-btn {
    opacity: 1;
  }
}
</style>
