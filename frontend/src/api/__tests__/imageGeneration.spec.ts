import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  ImageGatewayError,
  MAX_IMAGE_GENERATION_COUNT,
  normalizeOpenAIImageResults,
  submitGeminiImageGatewayRequest,
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

  it('submits Gemini image generation through Chat Completions', async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({
      choices: [{
        message: {
          images: [{
            image_url: { url: 'data:image/png;base64,aGVsbG8=' },
          }],
        },
      }],
    }), { status: 200 })))
    vi.stubGlobal('fetch', fetchMock)

    await submitGeminiImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'draw a tomato',
      model: 'models/gemini-3.1-flash-image',
      count: 1,
      size: '1024x1536',
    })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('https://ai.example/v1/chat/completions')
    expect(init.headers.get('Authorization')).toBe('Bearer test-key')
    const body = JSON.parse(init.body as string)
    expect(body.model).toBe('gemini-3.1-flash-image')
    expect(body.messages[0].content).toContain('draw a tomato')
    expect(body.messages[0].content).toContain('aspect ratio 2:3')
    expect(body.messages[0].content).toContain('1K resolution')
    expect(body.aspect_ratio).toBe('2:3')
    expect(body.image_size).toBe('1K')
  })

  it('maps Gemini 2K and 4K image sizes into Chat Completions fields', async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({
      choices: [{
        message: {
          images: [{
            image_url: { url: 'data:image/png;base64,aGVsbG8=' },
          }],
        },
      }],
    }), { status: 200 })))
    vi.stubGlobal('fetch', fetchMock)

    await submitGeminiImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'draw a tomato',
      model: 'gemini-3.1-flash-image',
      count: 1,
      size: '2048x2048',
    })
    await submitGeminiImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'draw a tomato',
      model: 'gemini-3.1-flash-image',
      count: 1,
      size: '4096x4096',
    })

    const firstBody = JSON.parse(fetchMock.mock.calls[0][1].body as string)
    const secondBody = JSON.parse(fetchMock.mock.calls[1][1].body as string)
    expect(firstBody.aspect_ratio).toBe('1:1')
    expect(firstBody.image_size).toBe('2K')
    expect(secondBody.aspect_ratio).toBe('1:1')
    expect(secondBody.image_size).toBe('4K')
  })

  it('normalizes Chat Completions image_url data URL responses', () => {
    const images = normalizeOpenAIImageResults({
      choices: [{
        message: {
          images: [{
            image_url: { url: 'data:image/jpeg;base64,aGVsbG8=' },
          }],
        },
      }],
    })

    expect(images).toHaveLength(1)
    expect(images[0].url).toBe('data:image/jpeg;base64,aGVsbG8=')
    expect(images[0].src).toBe('data:image/jpeg;base64,aGVsbG8=')
  })

  it('normalizes Gemini inlineData image responses', () => {
    const images = normalizeOpenAIImageResults({
      candidates: [{
        content: {
          parts: [{
            inlineData: {
              mimeType: 'image/webp',
              data: 'aGVsbG8=',
            },
          }],
        },
      }],
    })

    expect(images).toHaveLength(1)
    expect(images[0].b64_json).toBe('aGVsbG8=')
    expect(images[0].src).toBe('data:image/webp;base64,aGVsbG8=')
  })
})
