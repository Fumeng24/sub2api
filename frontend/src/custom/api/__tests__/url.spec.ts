import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadUrlModule(apiBase?: string) {
  vi.resetModules()
  if (apiBase === undefined) {
    vi.unstubAllEnvs()
  } else {
    vi.stubEnv('VITE_API_BASE_URL', apiBase)
  }
  return import('@/api/url')
}

describe('api url helpers', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('uses /api/v1 by default and avoids double prefixing', async () => {
    const { getAPIBaseURL, buildApiUrl } = await loadUrlModule()

    expect(getAPIBaseURL()).toBe('/api/v1')
    expect(buildApiUrl('/auth/refresh')).toBe('/api/v1/auth/refresh')
    expect(buildApiUrl('/api/v1/auth/refresh')).toBe('/api/v1/auth/refresh')
  })

  it('supports a full backend API base url', async () => {
    const { getAPIBaseURL, buildApiUrl, buildGatewayUrl } = await loadUrlModule(
      'https://api.example.com/api/v1/'
    )

    expect(getAPIBaseURL()).toBe('https://api.example.com/api/v1')
    expect(buildApiUrl('/payment/orders/123')).toBe('https://api.example.com/api/v1/payment/orders/123')
    expect(buildApiUrl('/api/v1/payment/orders/123')).toBe('https://api.example.com/api/v1/payment/orders/123')
    expect(buildGatewayUrl('/setup/status')).toBe('https://api.example.com/setup/status')
  })

  it('normalizes relative API base values', async () => {
    const { getAPIBaseURL, buildApiUrl } = await loadUrlModule('gateway/api/v1/')

    expect(getAPIBaseURL()).toBe('/gateway/api/v1')
    expect(buildApiUrl('pages/readme')).toBe('/gateway/api/v1/pages/readme')
  })
})
