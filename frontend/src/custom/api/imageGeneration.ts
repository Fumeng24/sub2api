export interface OpenAIImageResult {
  id: string
  src: string
  b64_json?: string
  url?: string
  revised_prompt?: string
  output_format?: string
  quality?: string
  size?: string
}

export interface ImageGatewayRequest {
  apiKey: string
  baseUrl?: string
  platform?: string
  prompt: string
  model: string
  count: number
  size?: string
  quality?: string
  referenceImages?: File[]
  signal?: AbortSignal
}

export interface GeminiImageGatewayRequest extends ImageGatewayRequest {}

export const MAX_IMAGE_GENERATION_COUNT = 4

export class ImageGatewayError extends Error {
  status: number
  code?: string
  payload?: unknown

  constructor(message: string, status: number, code?: string, payload?: unknown) {
    super(message)
    this.name = 'ImageGatewayError'
    this.status = status
    this.code = code
    this.payload = payload
  }
}

const imageEndpoint = {
  generations: '/v1/images/generations',
  edits: '/v1/images/edits',
} as const

function gatewayUrl(baseUrl: string | undefined, path: string) {
  const normalizedBase = (baseUrl || '').trim().replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  if (!normalizedBase) {
    return path
  }
  return `${normalizedBase}${path}`
}

function addOptionalImageFields(target: Record<string, unknown>, request: ImageGatewayRequest) {
  const n = Math.max(1, Math.min(MAX_IMAGE_GENERATION_COUNT, Math.floor(Number(request.count) || 1)))
  target.model = request.model
  target.prompt = request.prompt
  target.n = n
  target.response_format = 'b64_json'
  if (request.platform?.trim().toLowerCase() === 'grok') {
    // The configured Grok image upstream only produces 1K output. Keep the
    // request explicit so a saved size choice cannot be interpreted as 2K/4K.
    target.size_tier = '1k'
    return
  }
  if (request.size && request.size !== 'auto') {
    target.size = request.size
  }
  if (request.quality && request.quality !== 'auto') {
    target.quality = request.quality
  }
}

function buildGenerationBody(request: ImageGatewayRequest) {
  const body: Record<string, unknown> = {}
  addOptionalImageFields(body, request)
  return body
}

function buildEditBody(request: ImageGatewayRequest) {
  const form = new FormData()
  const fields: Record<string, unknown> = {}
  addOptionalImageFields(fields, request)
  Object.entries(fields).forEach(([key, value]) => {
    form.append(key, String(value))
  })
  for (const [index, image] of (request.referenceImages || []).entries()) {
    form.append('image', image, image.name || `reference-${index + 1}.png`)
  }
  return form
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

function geminiAspectRatioFromSize(size: string | undefined) {
  switch (size) {
    case '1024x1536':
    case '2048x3072':
    case '4096x6144':
      return '2:3'
    case '1536x1024':
    case '3072x2048':
    case '6144x4096':
      return '3:2'
    case '1024x1365':
    case '2048x2730':
    case '4096x5460':
      return '3:4'
    case '1365x1024':
    case '2730x2048':
    case '5460x4096':
      return '4:3'
    case '1088x1920':
    case '2176x3840':
    case '4352x7680':
      return '9:16'
    case '1920x1088':
    case '3840x2176':
    case '7680x4352':
      return '16:9'
    case '1024x1024':
    case '2048x2048':
    case '4096x4096':
      return '1:1'
    default:
      return ''
  }
}

function geminiImageSizeFromSize(size: string | undefined) {
  switch (size) {
    case '1024x1536':
    case '1536x1024':
    case '1024x1365':
    case '1365x1024':
    case '1088x1920':
    case '1920x1088':
    case '1024x1024':
    case 'auto':
      return '1K'
    case '2048x3072':
    case '3072x2048':
    case '2048x2730':
    case '2730x2048':
    case '2176x3840':
    case '3840x2176':
    case '2048x2048':
      return '2K'
    case '4096x6144':
    case '6144x4096':
    case '4096x5460':
    case '5460x4096':
    case '4352x7680':
    case '7680x4352':
    case '4096x4096':
      return '4K'
    default:
      return ''
  }
}

function geminiImageOptionsFromSize(size: string | undefined) {
  return {
    aspectRatio: geminiAspectRatioFromSize(size),
    imageSize: geminiImageSizeFromSize(size),
  }
}

function geminiImagePrompt(request: GeminiImageGatewayRequest) {
  const options = geminiImageOptionsFromSize(request.size)
  const instructions: string[] = []
  if (options.aspectRatio) {
    instructions.push(`aspect ratio ${options.aspectRatio}`)
  }
  if (options.imageSize) {
    instructions.push(`${options.imageSize} resolution`)
  }
  if (instructions.length === 0) {
    return request.prompt
  }
  return `${request.prompt}\n\nImage requirements: ${instructions.join(', ')}.`
}

function geminiInlineDataFromDataURL(dataURL: string) {
  const match = dataURL.match(/^data:([^;,]+);base64,(.+)$/i)
  if (!match) return null
  return {
    mimeType: match[1],
    data: match[2],
  }
}

async function buildGeminiGenerateContentBody(request: GeminiImageGatewayRequest) {
  const promptText = geminiImagePrompt(request)
  const parts: Array<Record<string, unknown>> = [{
    type: 'text',
    text: promptText,
  }]
  for (const image of request.referenceImages || []) {
    const dataURL = await readFileAsDataURL(image)
    const inlineData = geminiInlineDataFromDataURL(dataURL)
    if (inlineData) {
      parts.push({ inlineData })
    }
  }

  const imageOptions = geminiImageOptionsFromSize(request.size)
  const body: Record<string, unknown> = {
    contents: [{
      role: 'user',
      parts,
    }],
    generationConfig: {
      responseModalities: ['IMAGE'],
    },
  }
  if (imageOptions.aspectRatio || imageOptions.imageSize) {
    body.generationConfig = {
      responseModalities: ['IMAGE'],
      imageConfig: {
        ...(imageOptions.aspectRatio ? { aspectRatio: imageOptions.aspectRatio } : {}),
        ...(imageOptions.imageSize ? { imageSize: imageOptions.imageSize } : {}),
      },
    }
  }
  return body
}

async function readResponse(response: Response) {
  const text = await response.text()
  if (!text) {
    return null
  }
  return parseJSONValue(text) ?? text
}

function parseJSONValue(text: string): unknown | undefined {
  const trimmed = text.trim()
  if (!trimmed) {
    return undefined
  }
  try {
    return JSON.parse(trimmed)
  } catch {
    return undefined
  }
}

function parseSSEPayloads(text: string): unknown[] {
  const events: unknown[] = []
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  let dataLines: string[] = []

  const flush = () => {
    const data = dataLines.join('\n').trim()
    dataLines = []
    if (!data || data === '[DONE]') {
      return
    }
    events.push(parseJSONValue(data) ?? data)
  }

  for (const rawLine of lines) {
    const line = rawLine.trimEnd()
    if (line === '') {
      flush()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart())
    }
  }
  flush()
  return events
}

function normalizeGatewayPayload(payload: unknown): unknown {
  if (typeof payload !== 'string') {
    return payload
  }
  const parsed = parseJSONValue(payload)
  if (parsed !== undefined) {
    return parsed
  }
  const events = parseSSEPayloads(payload)
  return events.length > 0 ? events : payload
}

function readErrorMessage(value: unknown) {
  if (typeof value === 'string' && value.trim()) {
    return value.trim()
  }
  if (!isRecord(value)) {
    return ''
  }
  for (const key of ['message', 'detail', 'error_description']) {
    const message = value[key]
    if (typeof message === 'string' && message.trim()) {
      return message.trim()
    }
  }
  return ''
}

function readErrorCode(value: unknown) {
  if (!isRecord(value)) {
    return undefined
  }
  const code = value.code || value.type
  return typeof code === 'string' && code.trim() ? code.trim() : undefined
}

function extractGatewayPayloadError(payload: unknown): { message: string; code?: string } | null {
  const visited = new WeakSet<object>()

  function fromErrorValue(value: unknown): { message: string; code?: string } | null {
    const message = readErrorMessage(value)
    if (!message) {
      return null
    }
    return { message, code: readErrorCode(value) }
  }

  function walk(value: unknown): { message: string; code?: string } | null {
    const normalized = normalizeGatewayPayload(value)
    if (normalized !== value) {
      return walk(normalized)
    }

    if (Array.isArray(value)) {
      for (const item of value) {
        const found = walk(item)
        if (found) return found
      }
      return null
    }
    if (!isRecord(value)) {
      return null
    }
    if (visited.has(value)) {
      return null
    }
    visited.add(value)

    const type = typeof value.type === 'string' ? value.type.toLowerCase() : ''
    const status = typeof value.status === 'string' ? value.status.toLowerCase() : ''

    if (value.error) {
      const found = fromErrorValue(value.error)
      if (found && (type.includes('error') || type === 'response.failed' || status === 'failed' || isRecord(value.error))) {
        return found
      }
    }
    if (type === 'response.failed' || status === 'failed') {
      const direct = fromErrorValue(value)
      if (direct) return direct
    }

    for (const child of Object.values(value)) {
      if (Array.isArray(child) || isRecord(child) || typeof child === 'string') {
        const found = walk(child)
        if (found) return found
      }
    }
    return null
  }

  return walk(payload)
}

function resolveGatewayErrorMessage(payload: unknown, fallback: string) {
  const payloadError = extractGatewayPayloadError(payload)
  if (payloadError?.message) {
    return payloadError.message
  }
  if (!payload || typeof payload !== 'object') {
    return fallback
  }
  const record = payload as Record<string, unknown>
  const direct = record.message || record.detail
  if (typeof direct === 'string' && direct.trim()) {
    return direct
  }
  const error = record.error
  if (error && typeof error === 'object') {
    const errorRecord = error as Record<string, unknown>
    const nested = errorRecord.message || errorRecord.detail
    if (typeof nested === 'string' && nested.trim()) {
      return nested
    }
  }
  return fallback
}

function resolveGatewayErrorCode(payload: unknown) {
  const payloadError = extractGatewayPayloadError(payload)
  if (payloadError?.code) {
    return payloadError.code
  }
  if (!payload || typeof payload !== 'object') {
    return undefined
  }
  const record = payload as Record<string, unknown>
  if (typeof record.code === 'string') {
    return record.code
  }
  if (record.error && typeof record.error === 'object') {
    const code = (record.error as Record<string, unknown>).code
    return typeof code === 'string' ? code : undefined
  }
  return undefined
}

export async function submitImageGatewayRequest(request: ImageGatewayRequest): Promise<unknown> {
  const references = request.referenceImages || []
  if (request.platform?.trim().toLowerCase() === 'grok' && references.length > 0) {
    throw new ImageGatewayError(
      'This Grok image model does not support reference images',
      400,
      'invalid_request_error',
    )
  }
  const isEdit = references.length > 0
  const endpoint = isEdit ? imageEndpoint.edits : imageEndpoint.generations
  const url = gatewayUrl(request.baseUrl, endpoint)
  const headers = new Headers({
    Authorization: `Bearer ${request.apiKey}`,
    Accept: 'application/json',
  })
  let body: BodyInit
  if (isEdit) {
    body = buildEditBody(request)
  } else {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(buildGenerationBody(request))
  }

  const response = await fetch(url, {
    method: 'POST',
    headers,
    body,
    signal: request.signal,
  })
  const payload = normalizeGatewayPayload(await readResponse(response))
  if (!response.ok) {
    throw new ImageGatewayError(
      resolveGatewayErrorMessage(payload, `Image request failed with HTTP ${response.status}`),
      response.status,
      resolveGatewayErrorCode(payload),
      payload,
    )
  }
  const payloadError = extractGatewayPayloadError(payload)
  if (payloadError) {
    throw new ImageGatewayError(payloadError.message, response.status, payloadError.code, payload)
  }
  return payload
}

export async function submitGeminiImageGatewayRequest(request: GeminiImageGatewayRequest): Promise<unknown> {
  const model = request.model.trim().replace(/^models\//, '')
  const url = gatewayUrl(request.baseUrl, `/v1beta/models/${encodeURIComponent(model)}:generateContent`)
  const headers = new Headers({
    Authorization: `Bearer ${request.apiKey}`,
    Accept: 'application/json',
    'Content-Type': 'application/json',
  })
  const requestedCount = Math.max(1, Math.min(MAX_IMAGE_GENERATION_COUNT, Math.floor(Number(request.count) || 1)))
  const payloads: unknown[] = []

  for (let index = 0; index < requestedCount; index += 1) {
    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify(await buildGeminiGenerateContentBody({ ...request, count: 1 })),
      signal: request.signal,
    })
    const payload = normalizeGatewayPayload(await readResponse(response))
    if (!response.ok) {
      throw new ImageGatewayError(
        resolveGatewayErrorMessage(payload, `Gemini image request failed with HTTP ${response.status}`),
        response.status,
        resolveGatewayErrorCode(payload),
        payload,
      )
    }
    const payloadError = extractGatewayPayloadError(payload)
    if (payloadError) {
      throw new ImageGatewayError(payloadError.message, response.status, payloadError.code, payload)
    }
    payloads.push(payload)
  }

  return requestedCount === 1 ? payloads[0] : payloads
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function normalizeBase64(value: unknown) {
  if (typeof value !== 'string') {
    return ''
  }
  const trimmed = value.trim()
  const match = trimmed.match(/^data:image\/[^;]+;base64,(.+)$/i)
  return match ? match[1] : trimmed
}

function imageMimeFromFormat(format: unknown) {
  const normalized = typeof format === 'string' ? format.trim().toLowerCase() : ''
  if (normalized.startsWith('image/')) {
    return normalized
  }
  if (normalized === 'jpg') {
    return 'image/jpeg'
  }
  if (normalized === 'jpeg' || normalized === 'png' || normalized === 'webp' || normalized === 'gif') {
    return `image/${normalized}`
  }
  return 'image/png'
}

function dataUrlFromBase64(b64: string, format: unknown) {
  return `data:${imageMimeFromFormat(format)};base64,${normalizeBase64(b64)}`
}

function readImageURL(value: unknown) {
  if (typeof value === 'string') {
    return value.trim()
  }
  if (isRecord(value)) {
    const url = value.url
    return typeof url === 'string' ? url.trim() : ''
  }
  return ''
}

function looksLikeImageGenerationItem(value: Record<string, unknown>) {
  const type = typeof value.type === 'string' ? value.type.toLowerCase() : ''
  return type.includes('image') || Boolean(
    value.output_format ||
    value.revised_prompt ||
    value.b64_json ||
    value.partial_image_b64 ||
    value.image_b64 ||
    value.image_base64 ||
    value.base64_json ||
    value.url ||
    value.image_url
  )
}

function collectCandidate(value: Record<string, unknown>): Omit<OpenAIImageResult, 'id' | 'src'> | null {
  const inlineData = isRecord(value.inlineData)
    ? value.inlineData
    : isRecord(value.inline_data)
      ? value.inline_data
      : null
  if (inlineData) {
    const mimeType = inlineData.mimeType || inlineData.mime_type
    const data = normalizeBase64(inlineData.data)
    if (typeof mimeType === 'string' && mimeType.toLowerCase().startsWith('image/') && data) {
      return {
        b64_json: data,
        output_format: mimeType,
      }
    }
  }

  const directMimeType = value.mimeType || value.mime_type
  const directData = normalizeBase64(value.data)
  if (typeof directMimeType === 'string' && directMimeType.toLowerCase().startsWith('image/') && directData) {
    return {
      b64_json: directData,
      output_format: directMimeType,
    }
  }

  const outputFormat = value.output_format || value.format || value.mime_type || value.mimeType
  const b64 = normalizeBase64(
    value.b64_json ||
    value.partial_image_b64 ||
    value.image_b64 ||
    value.image_base64 ||
    value.base64_json ||
    value.base64
  )
  const result = normalizeBase64(value.result)
  const imageString = normalizeBase64(value.image)
  const dataString = typeof value.data === 'string' ? normalizeBase64(value.data) : ''
  const url = readImageURL(value.url || value.image_url || value.download_url)
  const imageB64 = b64 || (looksLikeImageGenerationItem(value) ? (result || imageString || dataString) : '')
  const src = url || (imageB64 ? dataUrlFromBase64(imageB64, outputFormat) : '')
  if (!src) {
    return null
  }
  return {
    b64_json: imageB64 || undefined,
    url: url || undefined,
    revised_prompt: typeof value.revised_prompt === 'string' ? value.revised_prompt : undefined,
    output_format: typeof outputFormat === 'string' ? outputFormat : undefined,
    quality: typeof value.quality === 'string' ? value.quality : undefined,
    size: typeof value.size === 'string' ? value.size : undefined,
  }
}

export function normalizeOpenAIImageResults(payload: unknown): OpenAIImageResult[] {
  const results: OpenAIImageResult[] = []
  const seen = new Set<string>()
  const visited = new WeakSet<object>()

  function push(candidate: Omit<OpenAIImageResult, 'id' | 'src'> | null) {
    if (!candidate) return
    const src = candidate.url || (candidate.b64_json ? dataUrlFromBase64(candidate.b64_json, candidate.output_format) : '')
    if (!src || seen.has(src)) return
    seen.add(src)
    results.push({
      ...candidate,
      id: `img-${results.length + 1}-${Date.now()}`,
      src,
    })
  }

  function walk(value: unknown) {
    const normalized = normalizeGatewayPayload(value)
    if (normalized !== value) {
      walk(normalized)
      return
    }
    if (Array.isArray(value)) {
      if (visited.has(value)) {
        return
      }
      visited.add(value)
      value.forEach(walk)
      return
    }
    if (!isRecord(value)) {
      return
    }
    if (visited.has(value)) {
      return
    }
    visited.add(value)
    push(collectCandidate(value))
    for (const child of Object.values(value)) {
      if (Array.isArray(child) || isRecord(child) || typeof child === 'string') {
        walk(child)
      }
    }
  }

  walk(payload)
  return results
}
