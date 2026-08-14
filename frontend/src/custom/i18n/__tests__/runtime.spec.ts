import { afterEach, describe, expect, it, vi } from 'vitest'

import i18n, {
  LOCALE_CHANGED_EVENT,
  loadLocaleMessages,
  setLocale,
} from '@/i18n'

describe('site i18n runtime', () => {
  afterEach(async () => {
    await setLocale('en')
    localStorage.removeItem('sub2api_locale')
  })

  it('loads the site overlay through the public i18n entry', async () => {
    await loadLocaleMessages('en')
    await loadLocaleMessages('zh')

    expect(i18n.global.getLocaleMessage('en').home.heroSubtitle).toBe(
      'Leading model services in one account',
    )
    expect(i18n.global.getLocaleMessage('zh').home.heroSubtitle).toBe('主流模型统一接入')
  })

  it('persists locale changes and dispatches the site event', async () => {
    const listener = vi.fn()
    window.addEventListener(LOCALE_CHANGED_EVENT, listener)

    await setLocale('zh')

    expect(localStorage.getItem('sub2api_locale')).toBe('zh')
    expect(document.documentElement.getAttribute('lang')).toBe('zh')
    expect(listener).toHaveBeenCalledOnce()

    window.removeEventListener(LOCALE_CHANGED_EVENT, listener)
  })
})
