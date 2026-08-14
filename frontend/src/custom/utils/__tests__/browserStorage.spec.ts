import { afterEach, describe, expect, it, vi } from 'vitest'

import { createSafeStorageFacade } from '@/custom/utils/browserStorage'

describe('createSafeStorageFacade', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('preserves the Storage get/set/remove contract', () => {
    const storage = createSafeStorageFacade('localStorage')

    storage.setItem('token', 'value')
    expect(storage.getItem('token')).toBe('value')

    storage.removeItem('token')
    expect(storage.getItem('token')).toBeNull()
  })

  it('does not throw when a webview blocks storage access', () => {
    vi.spyOn(window, 'localStorage', 'get').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError')
    })
    const storage = createSafeStorageFacade('localStorage')

    expect(() => storage.setItem('token', 'value')).not.toThrow()
    expect(storage.getItem('token')).toBeNull()
    expect(() => storage.removeItem('token')).not.toThrow()
  })
})
