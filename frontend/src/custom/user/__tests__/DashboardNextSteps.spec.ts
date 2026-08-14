import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'
import enLocale from '@/custom/i18n/locales/en'
import zhLocale from '@/custom/i18n/locales/zh'

const dashboardView = readFileSync(
  resolve(__dirname, '../../../custom/user/WegooDashboardView.vue'),
  'utf8',
)
const usageView = readFileSync(resolve(__dirname, '../WegooUsageView.vue'), 'utf8')

function localeValue(locale: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((current, key) => {
    if (current === null || typeof current !== 'object' || Array.isArray(current)) return undefined
    return (current as Record<string, unknown>)[key]
  }, locale)
}

describe('Dashboard next-step states', () => {
  it('surfaces no-key, low-balance, and recent-error actions from real account facts', () => {
    expect(dashboardView).toContain('dashboardNextActions')
    expect(dashboardView).toContain("(stats.value?.active_api_keys || 0) <= 0")
    expect(dashboardView).toContain('currentBalance.value <= lowBalanceThreshold.value')
    expect(dashboardView).toContain('recentErrorTotal.value > 0')
  })

  it('loads recent error summary only when the user error fact source is enabled', () => {
    expect(dashboardView).toContain('dashboardErrorViewEnabled')
    expect(dashboardView).toContain('appStore.cachedPublicSettings?.allow_user_view_error_requests === true')
    expect(dashboardView).toContain('usageAPI.listMyErrorRequests')
    expect(dashboardView).toContain("to: { path: '/usage', query: { tab: 'errors' } }")
  })

  it('keeps visible copy localized in Chinese and English', () => {
    for (const locale of [zhLocale, enLocale]) {
      for (const path of [
        'dashboard.nextSteps.noKey.title',
        'dashboard.nextSteps.lowBalance.title',
        'dashboard.nextSteps.recentError.title',
      ]) {
        expect(localeValue(locale, path)).toEqual(expect.any(String))
      }
    }
  })
})

describe('Usage error tab deep link', () => {
  it('opens the error tab when Dashboard links to /usage?tab=errors', () => {
    expect(usageView).toContain("get('tab') === 'errors'")
    expect(usageView).toContain('shouldOpenErrorsTab && errorViewEnabled.value')
    expect(usageView).toContain('watch(errorViewEnabled')
    expect(usageView).toContain('switchToErrors()')
  })
})
