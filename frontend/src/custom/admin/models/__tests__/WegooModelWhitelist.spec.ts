import { describe, expect, it, vi } from 'vitest'

vi.mock('@/custom/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn(),
}))

import { buildModelMappingObject, getModelsByPlatform } from '@/custom/composables/useModelWhitelist'

describe('Wegoo model whitelist policy', () => {
  it('exposes the promoted OpenAI models and removes legacy entries', () => {
    const models = getModelsByPlatform('openai')

    expect(models).toContain('gpt-5.3-codex-spark')
    expect(models).not.toContain('gpt-4o')
    expect(models).not.toContain('gpt-4.1')
    expect(models).not.toContain('gpt-5.2')
    expect(models).not.toContain('gpt-5.3-codex')
    expect(models).not.toContain('gpt-image-2')
  })

  it('keeps the exact GPT-5.5 identity mapping in whitelist mode', () => {
    expect(buildModelMappingObject('whitelist', ['gpt-5.5'], [])).toEqual({
      'gpt-5.5': 'gpt-5.5',
    })
  })
})
