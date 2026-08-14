import type { Group } from '@/types'

export const DEFAULT_OPENAI_IMAGE_MODELS = ['gpt-image-2', 'gpt-image-1.5', 'gpt-image-1']
export const DEFAULT_GEMINI_IMAGE_MODELS = ['gemini-3.1-flash-image', 'gemini-2.5-flash-image']
export const DEFAULT_GROK_IMAGE_MODELS = ['grok-imagine-image']
export const DEFAULT_IMAGE_MODEL = DEFAULT_OPENAI_IMAGE_MODELS[0]

const SUPPORTED_IMAGE_PLATFORMS = new Set(['openai', 'gemini', 'grok'])

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
  if (platform === 'grok') {
    return DEFAULT_GROK_IMAGE_MODELS
  }
  return []
}

export function resolveSupportedImageModels(group: Pick<Group, 'platform' | 'models_list_config'>) {
  if (group.models_list_config?.enabled) {
    const models = group.models_list_config.models
    if (!Array.isArray(models)) return []
    return uniqueImageModels(models).filter(isImageGenerationModelName)
  }
  // Fallback only applies when the current group does not enable a custom
  // user-visible model list. Enabled lists remain strict to avoid cross-group leakage.
  return defaultImageModelsForGroup(group)
}

export function isImageGenerationGroup(group: Pick<Group, 'allow_image_generation' | 'platform' | 'models_list_config'>) {
  if (!group.allow_image_generation || !isSupportedImagePlatform(group.platform)) {
    return false
  }

  // Grok text groups can also have the image-generation capability enabled for
  // API routing. Only expose them in the image workbench when their explicit
  // user-facing model list actually contains an image model.
  if (group.platform.trim().toLowerCase() === 'grok' && group.models_list_config?.enabled) {
    return resolveSupportedImageModels(group).length > 0
  }

  return true
}
