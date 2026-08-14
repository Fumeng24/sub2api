import type { RouteLocationNormalized } from 'vue-router'
import { i18n } from '@/i18n'
import { resolveDocumentTitle } from '@/router/title'
import { DEFAULT_SITE_NAME } from '@/utils/branding'

const DEFAULT_ORIGIN = 'https://ai.wegoo.site'
const DEFAULT_DESCRIPTION =
  'AI 模型服务，支持 GPT、Claude、Gemini、Codex 与图像生成 API。模型目录、访问凭证、用量记录、余额和服务状态集中呈现。'
const DEFAULT_IMAGE_PATH = '/logo.png'
const NOINDEX_ROBOTS = 'noindex,nofollow'
const INDEX_ROBOTS = 'index,follow'

interface ApplyRouteSeoOptions {
  siteName?: string
  title?: string
}

function normalizeSiteName(siteName?: string): string {
  if (typeof siteName !== 'string' || !siteName.trim()) {
    return DEFAULT_SITE_NAME
  }
  const normalized = siteName.trim()
  return normalized === 'Sub2API' ? DEFAULT_SITE_NAME : normalized
}

function textMetaValue(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function translateMetaValue(key: unknown): string | undefined {
  if (typeof key !== 'string' || !key.trim()) {
    return undefined
  }
  const translated = i18n.global.t(key)
  if (!translated || translated === key) {
    return undefined
  }
  return String(translated)
}

function withSiteSuffix(title: string, siteName: string): string {
  if (title.includes(siteName)) {
    return title
  }
  return `${title} - ${siteName}`
}

function resolveSeoTitle(route: RouteLocationNormalized, siteName: string, overrideTitle?: string): string {
  const seoTitle = textMetaValue(route.meta.seoTitle)
  if (seoTitle) {
    return withSiteSuffix(seoTitle, siteName)
  }
  if (overrideTitle) {
    return overrideTitle
  }
  return resolveDocumentTitle(route.meta.title, siteName, textMetaValue(route.meta.titleKey))
}

function resolveSeoDescription(route: RouteLocationNormalized): string {
  return (
    textMetaValue(route.meta.description) ||
    translateMetaValue(route.meta.descriptionKey) ||
    DEFAULT_DESCRIPTION
  )
}

function resolveRobots(route: RouteLocationNormalized): string {
  const explicitRobots = textMetaValue(route.meta.robots)
  if (explicitRobots) {
    return explicitRobots
  }
  if (route.name === 'NotFound' || route.meta.requiresAuth !== false) {
    return NOINDEX_ROBOTS
  }
  return INDEX_ROBOTS
}

function resolveOrigin(): string {
  const configuredOrigin = textMetaValue(import.meta.env.VITE_SITE_ORIGIN)
  if (configuredOrigin) {
    return configuredOrigin.replace(/\/+$/, '')
  }

  if (typeof window !== 'undefined' && window.location.origin) {
    const origin = window.location.origin
    if (!origin.includes('localhost') && !origin.includes('127.0.0.1')) {
      return origin.replace(/\/+$/, '')
    }
  }

  return DEFAULT_ORIGIN
}

function normalizePath(path: string): string {
  if (!path || path === '/') {
    return '/'
  }
  return `/${path.replace(/^\/+/, '').replace(/\/+$/, '')}`
}

function resolveCanonicalUrl(route: RouteLocationNormalized): string {
  const origin = resolveOrigin()
  const canonicalPath = textMetaValue(route.meta.canonicalPath) || route.path || '/'
  return `${origin}${normalizePath(canonicalPath)}`
}

function absolutizeUrl(url: string): string {
  if (/^https?:\/\//i.test(url)) {
    return url
  }
  const origin = resolveOrigin()
  return `${origin}${normalizePath(url)}`
}

function ensureMetaByName(name: string): HTMLMetaElement {
  let element = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)
  if (!element) {
    element = document.createElement('meta')
    element.setAttribute('name', name)
    document.head.appendChild(element)
  }
  return element
}

function ensureMetaByProperty(property: string): HTMLMetaElement {
  let element = document.querySelector<HTMLMetaElement>(`meta[property="${property}"]`)
  if (!element) {
    element = document.createElement('meta')
    element.setAttribute('property', property)
    document.head.appendChild(element)
  }
  return element
}

function setMetaByName(name: string, content: string): void {
  ensureMetaByName(name).setAttribute('content', content)
}

function setMetaByProperty(property: string, content: string): void {
  ensureMetaByProperty(property).setAttribute('content', content)
}

function setCanonical(url: string): void {
  let element = document.querySelector<HTMLLinkElement>('link[rel="canonical"]')
  if (!element) {
    element = document.createElement('link')
    element.setAttribute('rel', 'canonical')
    document.head.appendChild(element)
  }
  element.setAttribute('href', url)
}

export function applyRouteSeo(route: RouteLocationNormalized, options: ApplyRouteSeoOptions = {}): void {
  if (typeof document === 'undefined') {
    return
  }

  const siteName = normalizeSiteName(options.siteName)
  const title = resolveSeoTitle(route, siteName, options.title)
  const description = resolveSeoDescription(route)
  const canonicalUrl = resolveCanonicalUrl(route)
  const imageUrl = absolutizeUrl(textMetaValue(route.meta.ogImage) || DEFAULT_IMAGE_PATH)
  const robots = resolveRobots(route)
  const ogType = textMetaValue(route.meta.ogType) || 'website'

  document.title = title
  setMetaByName('description', description)
  setMetaByName('robots', robots)
  setCanonical(canonicalUrl)

  setMetaByProperty('og:site_name', siteName)
  setMetaByProperty('og:title', title)
  setMetaByProperty('og:description', description)
  setMetaByProperty('og:type', ogType)
  setMetaByProperty('og:url', canonicalUrl)
  setMetaByProperty('og:image', imageUrl)

  setMetaByName('twitter:card', 'summary')
  setMetaByName('twitter:title', title)
  setMetaByName('twitter:description', description)
  setMetaByName('twitter:image', imageUrl)
}
