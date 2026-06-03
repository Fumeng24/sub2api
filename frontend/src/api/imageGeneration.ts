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
  prompt: string
  model: string
  count: number
  size?: string
  quality?: string
  referenceImages?: File[]
  signal?: AbortSignal
}

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
  const n = Math.max(1, Math.min(10, Math.floor(Number(request.count) || 1)))
  target.model = request.model
  target.prompt = request.prompt
  target.n = n
  target.response_format = 'b64_json'
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

async function readResponse(response: Response) {
  const text = await response.text()
  if (!text) {
    return null
  }
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function resolveGatewayErrorMessage(payload: unknown, fallback: string) {
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
  const payload = await readResponse(response)
  if (!response.ok) {
    throw new ImageGatewayError(
      resolveGatewayErrorMessage(payload, `Image request failed with HTTP ${response.status}`),
      response.status,
      resolveGatewayErrorCode(payload),
      payload,
    )
  }
  return payload
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
  return type.includes('image') || Boolean(value.output_format || value.revised_prompt || value.b64_json || value.url)
}

function collectCandidate(value: Record<string, unknown>): Omit<OpenAIImageResult, 'id' | 'src'> | null {
  const outputFormat = value.output_format
  const b64 = normalizeBase64(value.b64_json || value.partial_image_b64)
  const result = normalizeBase64(value.result)
  const url = readImageURL(value.url || value.image_url || value.download_url)
  const imageB64 = b64 || (looksLikeImageGenerationItem(value) ? result : '')
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
    if (Array.isArray(value)) {
      value.forEach(walk)
      return
    }
    if (!isRecord(value)) {
      return
    }
    push(collectCandidate(value))
    if (isRecord(value.response)) {
      walk(value.response)
    }
    if (isRecord(value.item)) {
      walk(value.item)
    }
    if (Array.isArray(value.data)) {
      walk(value.data)
    }
    if (Array.isArray(value.output)) {
      walk(value.output)
    }
  }

  walk(payload)
  return results
}
