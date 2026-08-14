import { describe, expect, it } from 'vitest'
import type { Group } from '@/types'
import {
  DEFAULT_GEMINI_IMAGE_MODELS,
  DEFAULT_GROK_IMAGE_MODELS,
  DEFAULT_OPENAI_IMAGE_MODELS,
  isImageGenerationGroup,
  resolveSupportedImageModels,
} from '@/custom/utils/imageGenerationGroups'

function group(overrides: Partial<Group>): Group {
  return {
    id: 1,
    name: 'GPT生图',
    platform: 'openai',
    allow_image_generation: true,
    models_list_config: { enabled: true, models: [] },
    ...overrides,
  } as Group
}

describe('imageGenerationGroups', () => {
  it('allows OpenAI, Gemini, and Grok image groups when image generation is enabled', () => {
    expect(isImageGenerationGroup(group({ platform: 'openai', name: 'GPT生图' }))).toBe(true)
    expect(isImageGenerationGroup(group({ platform: 'openai', name: 'GPT普通' }))).toBe(true)
    expect(isImageGenerationGroup(group({ platform: 'gemini', name: 'Gemini普通' }))).toBe(true)
    expect(isImageGenerationGroup(group({
      platform: 'grok',
      name: 'Grok生图',
      models_list_config: { enabled: true, models: ['grok-imagine-image'] },
    }))).toBe(true)
    expect(isImageGenerationGroup(group({
      platform: 'grok',
      name: 'Grok Plus',
      models_list_config: { enabled: true, models: ['grok-4.5'] },
    }))).toBe(false)
    expect(isImageGenerationGroup(group({ platform: 'anthropic', name: 'Claude生图' }))).toBe(false)
    expect(isImageGenerationGroup(group({ allow_image_generation: false, name: 'GPT生图' }))).toBe(false)
  })

  it('uses only configured models whose names include image', () => {
    expect(resolveSupportedImageModels(group({
      platform: 'gemini',
      models_list_config: {
        enabled: true,
        models: [
          'gemini-2.5-pro',
          'gemini-2.5-flash-image',
          ' models/gemini-3.1-flash-image ',
          'gemini-2.5-flash-image',
        ],
      },
    }))).toEqual(['gemini-2.5-flash-image', 'models/gemini-3.1-flash-image'])
  })

  it('does not fall back when a configured model list contains no image model', () => {
    expect(resolveSupportedImageModels(group({ platform: 'openai', models_list_config: { enabled: true, models: ['gpt-5'] } })))
      .toEqual([])
    expect(resolveSupportedImageModels(group({ platform: 'gemini', models_list_config: { enabled: true, models: ['gemini-2.5-pro'] } })))
      .toEqual([])
  })

  it('falls back to platform image defaults only when supported models are absent', () => {
    expect(resolveSupportedImageModels(group({ platform: 'openai', models_list_config: undefined })))
      .toEqual(DEFAULT_OPENAI_IMAGE_MODELS)
    expect(resolveSupportedImageModels(group({ platform: 'gemini', models_list_config: undefined })))
      .toEqual(DEFAULT_GEMINI_IMAGE_MODELS)
    expect(resolveSupportedImageModels(group({ platform: 'grok', models_list_config: undefined })))
      .toEqual(DEFAULT_GROK_IMAGE_MODELS)
  })

  it('falls back to platform image defaults when custom model list is disabled', () => {
    expect(resolveSupportedImageModels(group({ platform: 'openai', models_list_config: { enabled: false, models: [] } })))
      .toEqual(DEFAULT_OPENAI_IMAGE_MODELS)
    expect(resolveSupportedImageModels(group({ platform: 'gemini', models_list_config: { enabled: false, models: ['gemini-2.5-pro'] } })))
      .toEqual(DEFAULT_GEMINI_IMAGE_MODELS)
    expect(resolveSupportedImageModels(group({ platform: 'grok', models_list_config: { enabled: false, models: [] } })))
      .toEqual(DEFAULT_GROK_IMAGE_MODELS)
  })
})
