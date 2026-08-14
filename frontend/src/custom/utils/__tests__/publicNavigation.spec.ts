import { describe, expect, it } from 'vitest'
import { resolvePublicCustomNavigationItems } from '../publicNavigation'
import type { CustomMenuItem } from '@/types'

const item = (overrides: Partial<CustomMenuItem>): CustomMenuItem => ({
  id: overrides.id || 'item',
  label: overrides.label || 'Item',
  icon_svg: '',
  url: overrides.url || '',
  visibility: overrides.visibility || 'user',
  sort_order: overrides.sort_order ?? 0,
  page_slug: overrides.page_slug,
})

describe('resolvePublicCustomNavigationItems', () => {
  it('keeps public FAQ-style custom pages and filters ticket shortcuts', () => {
    const result = resolvePublicCustomNavigationItems([
      item({ id: 'faq', label: 'FAQ', url: 'md:faq', page_slug: 'faq', sort_order: 1 }),
      item({ id: 'tickets', label: '工单', url: '/tickets', sort_order: 2 }),
      item({ id: 'admin', label: '后台', url: '/admin/tickets', visibility: 'admin', sort_order: 3 }),
    ])

    expect(result).toEqual([
      {
        key: 'faq',
        label: 'FAQ',
        href: '/custom/faq',
        route: '/custom/faq',
        external: false,
      },
    ])
  })
})
