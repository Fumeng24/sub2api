<template>
  <header class="public-gateway-nav">
    <nav class="public-gateway-nav__inner" :aria-label="copy.navLabel">
      <router-link to="/home" class="public-gateway-brand" :aria-label="siteName">
        <span class="public-gateway-brand__mark">
          <img :src="siteLogo || '/logo.png'" alt="" />
        </span>
        <span class="public-gateway-brand__text">
          <strong>{{ siteName }}</strong>
          <em>AI Gateway</em>
        </span>
      </router-link>

      <div class="public-gateway-nav__links">
        <router-link
          to="/pricing"
          class="public-gateway-link"
          :class="{ 'public-gateway-link--active': isActive('/pricing') }"
          :aria-current="isActive('/pricing') ? 'page' : undefined"
        >
          {{ copy.models }}
        </router-link>
        <router-link
          to="/status"
          class="public-gateway-link"
          :class="{ 'public-gateway-link--active': isActive('/status') }"
          :aria-current="isActive('/status') ? 'page' : undefined"
        >
          {{ copy.status }}
        </router-link>
        <router-link
          to="/enterprise"
          class="public-gateway-link"
          :class="{ 'public-gateway-link--active': isActive('/enterprise') }"
          :aria-current="isActive('/enterprise') ? 'page' : undefined"
        >
          {{ copy.enterprise }}
        </router-link>
        <a
          v-if="docsLink"
          :href="docsLink.href"
          :target="docsLink.external ? '_blank' : undefined"
          :rel="docsLink.external ? 'noopener noreferrer' : undefined"
          class="public-gateway-link"
          :class="{ 'public-gateway-link--active': isDocsActive }"
          :aria-current="isDocsActive ? 'page' : undefined"
          @click="handleDocsLinkClick"
        >
          {{ copy.docs }}
        </a>
        <router-link
          v-else
          to="/docs"
          class="public-gateway-link"
          :class="{ 'public-gateway-link--active': isDocsActive }"
          :aria-current="isDocsActive ? 'page' : undefined"
        >
          {{ copy.docs }}
        </router-link>
        <template v-for="item in customNavItems" :key="item.key">
          <a
            v-if="item.external"
            :href="item.href"
            target="_blank"
            rel="noopener noreferrer"
            class="public-gateway-link"
          >
            {{ item.label }}
          </a>
          <router-link
            v-else
            :to="item.route || item.href"
            class="public-gateway-link"
            :class="{ 'public-gateway-link--active': isCustomItemActive(item) }"
            :aria-current="isCustomItemActive(item) ? 'page' : undefined"
          >
            {{ item.label }}
          </router-link>
        </template>
      </div>

      <div class="public-gateway-nav__actions">
        <button
          type="button"
          class="public-gateway-icon-button"
          :aria-label="isDark ? copy.useLightTheme : copy.useDarkTheme"
          :title="isDark ? copy.useLightTheme : copy.useDarkTheme"
          @click="toggleTheme"
        >
          <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
        </button>
        <LocaleSwitcher />
        <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="public-gateway-secondary">
          {{ isAuthenticated ? copy.console : copy.signIn }}
        </router-link>
        <router-link :to="isAuthenticated ? '/keys' : '/register'" class="public-gateway-primary">
          {{ copy.start }}
        </router-link>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/custom/utils/docsLink'
import {
  resolvePublicCustomNavigationItems,
  type PublicNavigationItem,
} from '@/custom/utils/publicNavigation'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()
const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()
const { locale } = useI18n()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Wegoo AI')
const siteLogo = computed(() => sanitizeUrl(
  appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '',
  { allowRelative: true, allowDataUrl: true },
))
const rawDocUrl = computed(() => appStore.cachedPublicSettings?.doc_url || '')
const docsLink = computed(() => resolveDocsLink(rawDocUrl.value, appStore.cachedPublicSettings?.custom_menu_items ?? []))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const isDocsActive = computed(() => isActive('/docs'))
const customNavItems = computed(() => resolvePublicCustomNavigationItems(appStore.cachedPublicSettings?.custom_menu_items ?? []))
const isDark = ref(false)

const copy = computed(() => locale.value.startsWith('zh')
  ? {
    models: '模型价格',
    docs: '接入文档',
    status: '服务状态',
    enterprise: '企业服务',
    console: '控制台',
    signIn: '登录',
    start: '开始使用',
    navLabel: '首页导航',
    useLightTheme: '切换到浅色模式',
    useDarkTheme: '切换到深色模式',
  }
  : {
    models: 'Models',
    docs: 'Docs',
    status: 'Status',
    enterprise: 'Enterprise',
    console: 'Console',
    signIn: 'Sign in',
    start: 'Get Started',
    navLabel: 'Home navigation',
    useLightTheme: 'Switch to light theme',
    useDarkTheme: 'Switch to dark theme',
  })

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(`${path}/`)
}

function isCustomItemActive(item: PublicNavigationItem): boolean {
  return Boolean(item.route && isActive(item.route))
}

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link) || !link?.route) return
  event.preventDefault()
  router.push(link.route)
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  isDark.value = document.documentElement.classList.contains('dark')
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
