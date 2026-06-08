export interface CacheUsageModelLike {
  model?: string | null
  upstream_model?: string | null
  group?: {
    platform?: string | null
  } | null
}

const OPENAI_MODEL_PREFIX_RE = /^(gpt|chatgpt|codex|o[1-9]|o\d|text-embedding|dall-e)(?:[-_.:]|$)/i

function normalizeModelName(model: string): string {
  return model.replace(/^openai[/:]/i, '')
}

export function isOpenAICacheReadOnlyUsage(row: CacheUsageModelLike | null | undefined): boolean {
  if (!row) return false
  if (row.group?.platform === 'openai') return true

  const modelNames = [row.upstream_model, row.model]
    .map((value) => String(value || '').trim())
    .filter(Boolean)

  return modelNames.some((model) => OPENAI_MODEL_PREFIX_RE.test(normalizeModelName(model)))
}
