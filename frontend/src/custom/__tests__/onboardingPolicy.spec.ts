import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const tourSource = readFileSync(
  resolve(__dirname, '../composables/useOnboardingTour.ts'),
  'utf8',
)
const englishLocaleSource = readFileSync(
  resolve(__dirname, '../i18n/locales/en-custom.ts'),
  'utf8',
)
const chineseLocaleSource = readFileSync(
  resolve(__dirname, '../i18n/locales/zh-custom.ts'),
  'utf8',
)

describe('site onboarding policy', () => {
  it('keeps the production tour admin-only even when upstream exports user steps', () => {
    expect(tourSource).toContain("import { getAdminSteps } from '@/components/Guide/steps'")
    expect(tourSource).not.toContain('getUserSteps')
    expect(tourSource).toContain('if (!isAdmin || isSimpleMode) return')
  })

  it('does not expose the upstream product name in the first-run guide title', () => {
    expect(englishLocaleSource).toContain('"title": "Setup Guide"')
    expect(chineseLocaleSource).toContain('"title": "配置向导"')
    expect(englishLocaleSource).not.toContain('Sub2API Setup Guide')
    expect(chineseLocaleSource).not.toContain('Sub2API 配置向导')
  })
})
