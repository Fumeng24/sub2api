import type { CustomMenuItem } from '@/types'

export interface ResolvedDocsLink {
  href: string
  external: boolean
  route?: string
}

const MARKDOWN_DOCS_PREFIX = 'md:'
const MARKDOWN_SLUG_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/
const INTERNAL_PATH_PATTERN = /^\/(?!\/)[^\s\\]*$/

function normalizeInput(value: string): string {
  return value.trim()
}

function isHttpUrl(value: string): boolean {
  if (!/^https?:\/\//i.test(value)) return false
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function normalizeHttpUrl(value: string): string | null {
  if (!isHttpUrl(value)) return null
  return new URL(value).toString()
}

function isInternalPath(value: string): boolean {
  return INTERNAL_PATH_PATTERN.test(value)
}

function normalizeMarkdownSlug(value: string): string | null {
  if (!value.toLowerCase().startsWith(MARKDOWN_DOCS_PREFIX)) return null
  const slug = value.slice(MARKDOWN_DOCS_PREFIX.length).trim()
  return MARKDOWN_SLUG_PATTERN.test(slug) ? slug : null
}

function matchesMarkdownSlug(item: CustomMenuItem, slug: string): boolean {
  if (item.page_slug === slug) return true
  const itemSlug = normalizeMarkdownSlug(item.url || '')
  return itemSlug === slug
}

function resolveMarkdownDocsLink(slug: string, customMenuItems: CustomMenuItem[]): ResolvedDocsLink | null {
  const item = customMenuItems.find((menuItem) => matchesMarkdownSlug(menuItem, slug))
  if (!item?.id) return null

  const route = `/custom/${encodeURIComponent(item.id)}`
  return {
    href: route,
    route,
    external: false,
  }
}

export function resolveDocsLink(rawValue: string, customMenuItems: CustomMenuItem[] = []): ResolvedDocsLink | null {
  const value = normalizeInput(rawValue)
  if (!value) return null

  const externalHref = normalizeHttpUrl(value)
  if (externalHref) {
    return {
      href: externalHref,
      external: true,
    }
  }

  if (isInternalPath(value)) {
    return {
      href: value,
      route: value,
      external: false,
    }
  }

  const markdownSlug = normalizeMarkdownSlug(value)
  if (markdownSlug) {
    return resolveMarkdownDocsLink(markdownSlug, customMenuItems)
  }

  return null
}

export function normalizeDocsLinkValue(rawValue: string): string {
  const value = normalizeInput(rawValue)
  if (!value) return ''
  if (isHttpUrl(value) || isInternalPath(value) || normalizeMarkdownSlug(value)) return value
  return ''
}

export function shouldUseClientDocsNavigation(event: MouseEvent, link: ResolvedDocsLink | null): boolean {
  return Boolean(
    link &&
      !link.external &&
      !event.defaultPrevented &&
      event.button === 0 &&
      !event.metaKey &&
      !event.altKey &&
      !event.ctrlKey &&
      !event.shiftKey,
  )
}
