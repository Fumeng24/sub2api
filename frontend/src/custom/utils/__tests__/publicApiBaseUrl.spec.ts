import { describe, expect, it } from 'vitest'

import { resolvePublicApiBaseUrls } from '@/custom/utils/publicApiBaseUrl'

describe('resolvePublicApiBaseUrls', () => {
  it.each([
    ['https://api.example.com/', 'https://api.example.com', 'https://api.example.com/v1'],
    ['https://api.example.com/v1/', 'https://api.example.com', 'https://api.example.com/v1'],
    ['https://api.example.com/V1', 'https://api.example.com', 'https://api.example.com/V1'],
  ])('normalizes %s', (configured, root, v1) => {
    expect(resolvePublicApiBaseUrls(configured, 'https://fallback.example')).toEqual({ root, v1 })
  })

  it('uses the supplied origin when configuration is empty', () => {
    expect(resolvePublicApiBaseUrls('', 'https://fallback.example/')).toEqual({
      root: 'https://fallback.example',
      v1: 'https://fallback.example/v1',
    })
  })
})
