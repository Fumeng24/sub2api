import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  ImageGatewayError,
  MAX_IMAGE_GENERATION_COUNT,
  normalizeOpenAIImageResults,
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

  it('normalizes non-streaming event-stream image responses', () => {
    const payload = [
      'data: {"type":"response.output_item.done","item":{"id":"ig_1","type":"image_generation_call","result":"aGVsbG8=","output_format":"png","revised_prompt":"draw a cat"}}',
      '',
      'data: {"type":"response.completed","response":{"output":[]}}',
      '',
      'data: [DONE]',
      '',
    ].join('\n')

    const images = normalizeOpenAIImageResults(payload)

    expect(images).toHaveLength(1)
    expect(images[0].b64_json).toBe('aGVsbG8=')
    expect(images[0].src).toBe('data:image/png;base64,aGVsbG8=')
    expect(images[0].revised_prompt).toBe('draw a cat')
  })

  it('normalizes nested Responses image output blocks', () => {
    const images = normalizeOpenAIImageResults({
      output: [
        {
          type: 'message',
          content: [
            {
              type: 'output_image',
              image_url: { url: 'https://cdn.example/image.png' },
              size: '1024x1365',
            },
          ],
        },
      ],
    })

    expect(images).toHaveLength(1)
    expect(images[0].url).toBe('https://cdn.example/image.png')
    expect(images[0].size).toBe('1024x1365')
  })

  it('throws upstream errors from 2xx event-stream responses', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response([
      'data: {"type":"response.failed","response":{"error":{"type":"invalid_request_error","code":"invalid_size","message":"Invalid image size"}}}',
      '',
    ].join('\n'), {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(submitImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'draw a tomato',
      model: 'gpt-image-2',
      count: 1,
    })).rejects.toMatchObject({
      name: 'ImageGatewayError',
      message: 'Invalid image size',
      status: 200,
      code: 'invalid_size',
    } satisfies Partial<ImageGatewayError>)
  })
})
