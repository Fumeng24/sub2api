import { afterEach, describe, expect, it } from 'vitest'

import { getPersistedPageSize } from '@/custom/composables/usePersistedPageSize'

describe('usePersistedPageSize', () => {
  afterEach(() => {
    localStorage.clear()
    delete window.__APP_CONFIG__
  })

  it('uses the system table default instead of stale localStorage state', () => {
    window.__APP_CONFIG__ = {
      table_default_page_size: 1000,
      table_page_size_options: [20, 50, 1000]
    } as any
    localStorage.setItem('table-page-size', '50')
    localStorage.setItem('table-page-size-source', 'user')

    expect(getPersistedPageSize()).toBe(1000)
  })

  it('clears legacy page-size state before reading the persisted value', () => {
    localStorage.setItem('table-page-size', '50')
    localStorage.setItem('table-page-size-source', 'user')

    expect(getPersistedPageSize()).toBe(20)
    expect(localStorage.getItem('table-page-size')).toBeNull()
    expect(localStorage.getItem('table-page-size-source')).toBeNull()
  })
})
