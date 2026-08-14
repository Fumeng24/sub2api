import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  ImageGatewayError,
  MAX_IMAGE_GENERATION_COUNT,
  normalizeOpenAIImageResults,
  submitGeminiImageGatewayRequest,
  submitImageGatewayRequest,
} from '@/custom/api/imageGeneration'

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

  it('uses Grok native 1K base64 generation without size or quality fields', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [] }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await submitImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      platform: 'grok',
      prompt: 'draw a tomato',
      model: 'grok-imagine-image',
      count: 1,
      size: '4096x4096',
      quality: 'high',
    })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('https://ai.example/v1/images/generations')
    expect(JSON.parse(init.body as string)).toEqual({
      model: 'grok-imagine-image',
      prompt: 'draw a tomato',
      n: 1,
      response_format: 'b64_json',
      size_tier: '1k',
    })
  })

  it('does not route Grok reference images to the edits endpoint', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await expect(submitImageGatewayRequest({
      apiKey: 'test-key',
      platform: 'grok',
      prompt: 'edit this image',
      model: 'grok-imagine-image',
      count: 1,
      referenceImages: [new File(['x'], 'reference.png', { type: 'image/png' })],
    })).rejects.toMatchObject({
      name: 'ImageGatewayError',
      status: 400,
      code: 'invalid_request_error',
    } satisfies Partial<ImageGatewayError>)
    expect(fetchMock).not.toHaveBeenCalled()
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

  it('submits Gemini image generation through the native generateContent endpoint', async () => {
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
    expect(url).toBe('https://ai.example/v1beta/models/gemini-3.1-flash-image:generateContent')
    expect(init.headers.get('Authorization')).toBe('Bearer test-key')
    const body = JSON.parse(init.body as string)
    expect(body.contents[0].parts[0].text).toContain('draw a tomato')
    expect(body.contents[0].parts[0].text).toContain('aspect ratio 2:3')
    expect(body.contents[0].parts[0].text).toContain('1K resolution')
    expect(body.generationConfig.responseModalities).toEqual(['IMAGE'])
    expect(body.generationConfig.imageConfig).toEqual({ aspectRatio: '2:3', imageSize: '1K' })
  })

  it('maps Gemini 2K and 4K image sizes into native imageConfig fields', async () => {
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
    expect(firstBody.generationConfig.imageConfig).toEqual({ aspectRatio: '1:1', imageSize: '2K' })
    expect(secondBody.generationConfig.imageConfig).toEqual({ aspectRatio: '1:1', imageSize: '4K' })
  })

  it('converts Gemini reference images into inlineData parts', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      candidates: [{ content: { parts: [{ inlineData: { mimeType: 'image/png', data: 'aGVsbG8=' } }] } }],
    }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await submitGeminiImageGatewayRequest({
      apiKey: 'test-key',
      baseUrl: 'https://ai.example/v1',
      prompt: 'edit this image',
      model: 'gemini-3.1-flash-image',
      count: 1,
      referenceImages: [new File(['reference'], 'reference.png', { type: 'image/png' })],
    })

    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string)
    expect(body.contents[0].parts[1].inlineData.mimeType).toBe('image/png')
    expect(body.contents[0].parts[1].inlineData.data).toBeTruthy()
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
