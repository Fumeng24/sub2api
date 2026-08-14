type LocaleTree = Record<string, unknown>

function isLocaleTree(value: unknown): value is LocaleTree {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function mergeLocale<T extends LocaleTree>(base: T, overlay: LocaleTree): T {
  const result: LocaleTree = { ...base }
  for (const [key, value] of Object.entries(overlay)) {
    const current = result[key]
    if (isLocaleTree(current) && isLocaleTree(value)) {
      result[key] = mergeLocale(current, value)
      continue
    }
    result[key] = value
  }
  return result as T
}
