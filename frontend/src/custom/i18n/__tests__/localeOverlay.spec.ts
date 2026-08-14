import { describe, expect, it } from 'vitest'
import { mergeLocale } from '@/custom/i18n/locales/mergeLocale'

describe('mergeLocale', () => {
  it('overlays nested site translations without mutating the official baseline', () => {
    const official = {
      common: { save: 'Save', cancel: 'Cancel' },
      admin: { accounts: { title: 'Accounts' } },
    }
    const site = {
      common: { save: 'Store' },
      admin: { scheduler: { title: 'Scheduler' } },
    }

    const merged = mergeLocale(official, site)

    expect(merged).toEqual({
      common: { save: 'Store', cancel: 'Cancel' },
      admin: {
        accounts: { title: 'Accounts' },
        scheduler: { title: 'Scheduler' },
      },
    })
    expect(official).toEqual({
      common: { save: 'Save', cancel: 'Cancel' },
      admin: { accounts: { title: 'Accounts' } },
    })
  })
})
