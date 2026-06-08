import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  MAX_IMAGE_GENERATION_COUNT,
  submitImageGatewayRequest,
} from '@/api/imageGeneration'

describe('imageGeneration API', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('caps generation count at the upstream maximum', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [] }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await submitImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'draw a tomato',
      model: 'gpt-image-2',
      count: 10,
    })

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body as string).n).toBe(MAX_IMAGE_GENERATION_COUNT)
  })

  it('caps edit count at the upstream maximum', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [] }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await submitImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'make it brighter',
      model: 'gpt-image-2',
      count: 10,
      referenceImages: [new File(['x'], 'reference.png', { type: 'image/png' })],
    })

    const [, init] = fetchMock.mock.calls[0]
    expect(init.body).toBeInstanceOf(FormData)
    expect((init.body as FormData).get('n')).toBe(String(MAX_IMAGE_GENERATION_COUNT))
  })
})
