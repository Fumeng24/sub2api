export interface PublicApiBaseUrls {
  root: string
  v1: string
}

export function resolvePublicApiBaseUrls(
  configuredBaseUrl: string | null | undefined,
  fallbackOrigin?: string,
): PublicApiBaseUrls {
  const browserOrigin =
    typeof window !== 'undefined' && window.location.origin !== 'null'
      ? window.location.origin
      : ''
  const normalized = (configuredBaseUrl?.trim() || fallbackOrigin?.trim() || browserOrigin)
    .replace(/\/+$/, '')

  if (!normalized) {
    return { root: '', v1: '/v1' }
  }

  const hasV1Suffix = /\/v1$/i.test(normalized)
  const root = hasV1Suffix ? normalized.slice(0, -3) : normalized

  return {
    root,
    v1: hasV1Suffix ? normalized : `${normalized}/v1`,
  }
}
