import { describe, expect, it } from 'vitest'
import { isOpenAICacheReadOnlyUsage } from '@/custom/utils/cacheUsage'

describe('isOpenAICacheReadOnlyUsage', () => {
  it('detects GPT/OpenAI model names', () => {
    expect(isOpenAICacheReadOnlyUsage({ model: 'gpt-5.5' })).toBe(true)
    expect(isOpenAICacheReadOnlyUsage({ model: 'o3' })).toBe(true)
    expect(isOpenAICacheReadOnlyUsage({ model: 'openai/gpt-5.5' })).toBe(true)
    expect(isOpenAICacheReadOnlyUsage({ upstream_model: 'gpt-5.4-mini' })).toBe(true)
  })

  it('uses the OpenAI group platform when model name is mapped', () => {
    expect(isOpenAICacheReadOnlyUsage({
      model: 'claude-sonnet-4-5',
      upstream_model: 'custom-alias',
      group: { platform: 'openai' },
    })).toBe(true)
  })

  it('does not classify Claude models as OpenAI cache read-only usage', () => {
    expect(isOpenAICacheReadOnlyUsage({ model: 'claude-sonnet-4-5' })).toBe(false)
  })
})
