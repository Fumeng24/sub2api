import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../AuthLayout.vue'),
  'utf8',
)

describe('AuthLayout branded experience', () => {
  it('keeps the greeting and model route visual beside the auth form', () => {
    expect(source).toContain('class="auth-story"')
    expect(source).toContain("t('auth.gateway.title')")
    expect(source).toContain('class="auth-signal"')
    expect(source).toContain('class="auth-form-panel"')
  })

  it('provides a compact branded header and greeting on mobile', () => {
    expect(source).toContain('class="auth-mobile-brand"')
    expect(source).toContain('class="auth-mobile-greeting"')
    expect(source).toContain('@media (max-width: 900px)')
  })
})
