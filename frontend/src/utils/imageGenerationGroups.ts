import type { Group } from '@/types'

export const DEFAULT_OPENAI_IMAGE_MODELS = ['gpt-image-2', 'gpt-image-1.5', 'gpt-image-1']
export const DEFAULT_GEMINI_IMAGE_MODELS = ['gemini-3.1-flash-image', 'gemini-2.5-flash-image']
export const DEFAULT_IMAGE_MODEL = DEFAULT_OPENAI_IMAGE_MODELS[0]

const SUPPORTED_IMAGE_PLATFORMS = new Set(['openai', 'gemini'])

export function isSupportedImagePlatform(platform: string | undefined | null) {
  return SUPPORTED_IMAGE_PLATFORMS.has(String(platform || '').toLowerCase())
}

export function isImageGenerationModelName(model: string) {
  const normalized = model.trim().toLowerCase().replace(/^models\//, '')
  return normalized.includes('image')
}

export function uniqueImageModels(models: string[]) {
  const seen = new Set<string>()
  const out: string[] = []
  for (const model of models) {
    const normalized = model.trim()
    if (!normalized || seen.has(normalized)) continue
    seen.add(normalized)
    out.push(normalized)
  }
  return out
}

export function defaultImageModelsForGroup(group: Pick<Group, 'platform'>) {
  const platform = group.platform.toLowerCase()
  if (platform === 'gemini') {
    return DEFAULT_GEMINI_IMAGE_MODELS
  }
  if (platform === 'openai') {
    return DEFAULT_OPENAI_IMAGE_MODELS
  }
  return []
}

export function resolveSupportedImageModels(group: Pick<Group, 'platform' | 'models_list_config'>) {
  if (group.models_list_config) {
    const models = group.models_list_config.models
    if (!Array.isArray(models)) return []
    return uniqueImageModels(models).filter(isImageGenerationModelName)
  }
  // Legacy fallback for older group payloads that do not expose supported models.
  // Once models_list_config is present, the page must strictly follow that list.
  return defaultImageModelsForGroup(group)
}

export function isImageGenerationGroup(group: Pick<Group, 'allow_image_generation' | 'name' | 'platform'>) {
  return group.allow_image_generation && group.name.includes('生图') && isSupportedImagePlatform(group.platform)
}
