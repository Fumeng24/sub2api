import { describe, expect, it } from 'vitest'

import { providerGradient } from '@/custom/composables/useChannelMonitorFormat'

describe('custom channel monitor presentation', () => {
  it('uses restrained solid provider backgrounds', () => {
    expect(providerGradient('openai')).toBe('bg-emerald-500/10 dark:bg-emerald-500/15')
    expect(providerGradient('anthropic')).toBe('bg-orange-500/10 dark:bg-orange-500/15')
    expect(providerGradient('gemini')).toBe('bg-blue-500/10 dark:bg-blue-500/15')
    expect(providerGradient('unknown')).toBe('bg-gray-100 dark:bg-dark-700')
  })

  it('does not reintroduce gradient utility classes', () => {
    for (const provider of ['openai', 'anthropic', 'gemini', 'unknown']) {
      expect(providerGradient(provider)).not.toContain('gradient')
    }
  })
})
