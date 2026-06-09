import { describe, expect, it } from 'vitest'
import { normalizeDocsLinkValue, resolveDocsLink } from '../docsLink'
import type { CustomMenuItem } from '@/types'

const menuItems: CustomMenuItem[] = [
  {
    id: 'guide-page',
    label: 'Guide',
    icon_svg: '',
    url: 'md:guide',
    visibility: 'user',
    sort_order: 0,
  },
  {
    id: 'faq-page',
    label: 'FAQ',
    icon_svg: '',
    url: 'https://example.com/faq',
    page_slug: 'faq',
    visibility: 'user',
    sort_order: 1,
  },
]

describe('docsLink', () => {
  it('resolves http(s) documentation URLs as external links', () => {
    expect(resolveDocsLink(' https://docs.example.com/guide ')).toEqual({
      href: 'https://docs.example.com/guide',
      external: true,
    })
  })

  it('resolves same-origin absolute paths as client routes', () => {
    expect(resolveDocsLink('/custom/guide-page?tab=intro')).toEqual({
      href: '/custom/guide-page?tab=intro',
      route: '/custom/guide-page?tab=intro',
      external: false,
    })
  })

  it('resolves markdown slugs through matching custom menu items', () => {
    expect(resolveDocsLink('md:guide', menuItems)).toEqual({
      href: '/custom/guide-page',
      route: '/custom/guide-page',
      external: false,
    })
  })

  it('supports custom menu items that use page_slug', () => {
    expect(resolveDocsLink('md:faq', menuItems)).toEqual({
      href: '/custom/faq-page',
      route: '/custom/faq-page',
      external: false,
    })
  })

  it('rejects unsafe or unresolved documentation values', () => {
    expect(resolveDocsLink('javascript:alert(1)', menuItems)).toBeNull()
    expect(resolveDocsLink('//evil.example.com/docs', menuItems)).toBeNull()
    expect(resolveDocsLink('custom/guide-page', menuItems)).toBeNull()
    expect(resolveDocsLink('md:missing', menuItems)).toBeNull()
  })

  it('normalizes values accepted by admin settings', () => {
    expect(normalizeDocsLinkValue(' https://docs.example.com ')).toBe('https://docs.example.com')
    expect(normalizeDocsLinkValue('/custom/guide-page')).toBe('/custom/guide-page')
    expect(normalizeDocsLinkValue('md:guide')).toBe('md:guide')
    expect(normalizeDocsLinkValue('mailto:support@example.com')).toBe('')
  })
})
