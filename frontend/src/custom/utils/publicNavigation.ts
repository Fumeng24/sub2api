import type { CustomMenuItem } from '@/types'

export interface PublicNavigationItem {
  key: string
  label: string
  href: string
  external: boolean
  route?: string
}

const MARKDOWN_DOCS_PREFIX = 'md:'
const MARKDOWN_SLUG_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/
const INTERNAL_PATH_PATTERN = /^\/(?!\/)[^\s\\]*$/
const PRIVATE_PUBLIC_NAV_PREFIXES = [
  '/admin',
  '/tickets',
]

function normalizeValue(value: string | undefined): string {
  return (value || '').trim()
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

function isInternalPath(value: string): boolean {
  return INTERNAL_PATH_PATTERN.test(value)
}

function isPrivatePublicNavPath(path: string): boolean {
  return PRIVATE_PUBLIC_NAV_PREFIXES.some((prefix) => path === prefix || path.startsWith(`${prefix}/`))
}

function markdownSlug(value: string): string | null {
  if (!value.toLowerCase().startsWith(MARKDOWN_DOCS_PREFIX)) return null
  const slug = value.slice(MARKDOWN_DOCS_PREFIX.length).trim()
  return MARKDOWN_SLUG_PATTERN.test(slug) ? slug : null
}

export function resolvePublicCustomNavigationItems(items: CustomMenuItem[] = []): PublicNavigationItem[] {
  return [...items]
    .filter((item) => item.visibility === 'user' && Boolean(item.id) && Boolean(item.label?.trim()))
    .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.label.localeCompare(b.label))
    .map((item) => {
      const label = item.label.trim()
      const rawUrl = normalizeValue(item.url)
      const slug = markdownSlug(rawUrl) || item.page_slug

      if (slug && item.id) {
        const route = `/custom/${encodeURIComponent(item.id)}`
        return {
          key: item.id,
          label,
          href: route,
          route,
          external: false,
        }
      }

      if (isHttpUrl(rawUrl)) {
        return {
          key: item.id,
          label,
          href: new URL(rawUrl).toString(),
          external: true,
        }
      }

      if (isInternalPath(rawUrl) && !isPrivatePublicNavPath(rawUrl)) {
        return {
          key: item.id,
          label,
          href: rawUrl,
          route: rawUrl,
          external: false,
        }
      }

      return null
    })
    .filter((item): item is PublicNavigationItem => Boolean(item))
}
