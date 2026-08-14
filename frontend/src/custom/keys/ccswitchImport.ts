import type { GroupPlatform } from '@/types'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-5.6-sol'
export const GROK_CC_SWITCH_MODEL = 'grok-4.5'
export const CLAUDE_CC_SWITCH_HAIKU_MODEL = 'claude-haiku-4-5-20251001'
export const CLAUDE_CC_SWITCH_SONNET_MODEL = 'claude-sonnet-5'
export const CLAUDE_CC_SWITCH_OPUS_MODEL = 'claude-opus-5'
export const CLAUDE_CC_SWITCH_MODEL = CLAUDE_CC_SWITCH_SONNET_MODEL
export const GEMINI_CC_SWITCH_MODEL = 'gemini-3.5-flash'

export type CcSwitchClientType = 'claude' | 'gemini'

export type CcSwitchTargetId =
  | 'claude-code'
  | 'claude-desktop'
  | 'codex'
  | 'gemini-cli'
  | 'opencode'
  | 'openclaw'
  | 'hermes'

export type CcSwitchDeepLinkApp =
  | 'claude'
  | 'codex'
  | 'gemini'
  | 'opencode'
  | 'openclaw'
  | 'hermes'

export type CcSwitchEndpointMode = 'base' | 'v1' | 'v1beta'

export type CcSwitchProtocolMode =
  | 'anthropic-messages'
  | 'openai-responses'
  | 'gemini-native'
  | 'openai-compatible'
  | 'openai-completions'
  | 'chat-completions'

export type CcSwitchProtocolSupport = 'native' | 'compatibility'

export type CcSwitchReasoningEffort = 'high'

export interface CcSwitchClaudeRoleModels {
  haiku: string
  sonnet: string
  opus: string
}

export interface CcSwitchProtocolProfile {
  mode: CcSwitchProtocolMode
  support: CcSwitchProtocolSupport
}

export interface CcSwitchTargetDefinition {
  targetId: CcSwitchTargetId
  app: CcSwitchDeepLinkApp
  endpointMode: CcSwitchEndpointMode
  protocol: CcSwitchProtocolProfile
  model?: string
  reasoningEffort?: CcSwitchReasoningEffort
  claudeRoleModels?: CcSwitchClaudeRoleModels
  icon?: string
}

export interface CcSwitchImportConfig extends CcSwitchTargetDefinition {
  endpoint: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | null
  clientType?: CcSwitchClientType
  targetId?: CcSwitchTargetId
  providerName: string
  apiKey: string
  usageScript?: string
  usageBaseUrl?: string
}

export interface CcSwitchTargetListInput {
  baseUrl: string
  platform?: GroupPlatform | null
}

const CLAUDE_ROLE_MODELS: CcSwitchClaudeRoleModels = {
  haiku: CLAUDE_CC_SWITCH_HAIKU_MODEL,
  sonnet: CLAUDE_CC_SWITCH_SONNET_MODEL,
  opus: CLAUDE_CC_SWITCH_OPUS_MODEL
}

export const CC_SWITCH_TARGET_DEFINITIONS: Record<CcSwitchTargetId, CcSwitchTargetDefinition> = {
  'claude-code': {
    targetId: 'claude-code',
    app: 'claude',
    endpointMode: 'base',
    protocol: {
      mode: 'anthropic-messages',
      support: 'native'
    },
    model: CLAUDE_CC_SWITCH_MODEL,
    claudeRoleModels: CLAUDE_ROLE_MODELS,
    icon: 'claude'
  },
  'claude-desktop': {
    targetId: 'claude-desktop',
    app: 'claude',
    endpointMode: 'base',
    protocol: {
      mode: 'anthropic-messages',
      support: 'native'
    },
    model: CLAUDE_CC_SWITCH_MODEL,
    claudeRoleModels: CLAUDE_ROLE_MODELS,
    icon: 'claude'
  },
  codex: {
    targetId: 'codex',
    app: 'codex',
    endpointMode: 'base',
    protocol: {
      mode: 'openai-responses',
      support: 'native'
    },
    model: OPENAI_CC_SWITCH_CODEX_MODEL,
    reasoningEffort: 'high',
    icon: 'openai'
  },
  'gemini-cli': {
    targetId: 'gemini-cli',
    app: 'gemini',
    endpointMode: 'v1beta',
    protocol: {
      mode: 'gemini-native',
      support: 'native'
    },
    model: GEMINI_CC_SWITCH_MODEL,
    icon: 'gemini'
  },
  opencode: {
    targetId: 'opencode',
    app: 'opencode',
    endpointMode: 'v1',
    protocol: {
      mode: 'openai-compatible',
      support: 'compatibility'
    },
    model: OPENAI_CC_SWITCH_CODEX_MODEL,
    icon: 'opencode'
  },
  openclaw: {
    targetId: 'openclaw',
    app: 'openclaw',
    endpointMode: 'v1',
    protocol: {
      mode: 'openai-completions',
      support: 'compatibility'
    },
    model: OPENAI_CC_SWITCH_CODEX_MODEL,
    icon: 'claw'
  },
  hermes: {
    targetId: 'hermes',
    app: 'hermes',
    endpointMode: 'v1',
    protocol: {
      mode: 'chat-completions',
      support: 'compatibility'
    },
    model: OPENAI_CC_SWITCH_CODEX_MODEL,
    icon: 'hermes'
  }
}

export const CC_SWITCH_TARGETS_BY_PLATFORM: Record<GroupPlatform, readonly CcSwitchTargetId[]> = {
  openai: ['codex', 'opencode', 'openclaw', 'hermes'],
  anthropic: ['claude-code', 'claude-desktop'],
  gemini: ['gemini-cli'],
  antigravity: ['claude-code', 'claude-desktop', 'gemini-cli'],
  grok: ['opencode', 'openclaw', 'hermes'],
  composite: []
}

const trimTrailingSlash = (value: string): string => value.trim().replace(/\/+$/, '')

export function resolveApiBaseRoot(baseUrl: string): string {
  const trimmed = trimTrailingSlash(baseUrl || window.location.origin)
  return trimmed.replace(/\/v1beta$/i, '').replace(/\/v1$/i, '')
}

export function ensureV1Endpoint(baseUrl: string): string {
  return `${resolveApiBaseRoot(baseUrl)}/v1`
}

export function ensureV1BetaEndpoint(baseUrl: string): string {
  return `${resolveApiBaseRoot(baseUrl)}/v1beta`
}

function resolveDedicatedIdeRoot(baseUrl: string): string {
  return `${resolveApiBaseRoot(baseUrl)}/antigravity`
}

function resolveEndpoint(input: {
  baseUrl: string
  platform?: GroupPlatform | null
  target: CcSwitchTargetDefinition
}): string {
  const platform = input.platform || 'anthropic'
  const baseRoot = resolveApiBaseRoot(input.baseUrl)

  if (platform === 'antigravity') {
    const dedicatedRoot = resolveDedicatedIdeRoot(baseRoot)
    return input.target.endpointMode === 'v1beta'
      ? `${dedicatedRoot}/v1beta`
      : dedicatedRoot
  }

  switch (input.target.endpointMode) {
    case 'v1':
      return ensureV1Endpoint(baseRoot)
    case 'v1beta':
      return ensureV1BetaEndpoint(baseRoot)
    case 'base':
    default:
      return baseRoot
  }
}

function toBase64(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary)
}

export function getDefaultCcSwitchTargetId(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType = 'claude'
): CcSwitchTargetId {
  switch (platform || 'anthropic') {
    case 'antigravity':
      return clientType === 'gemini' ? 'gemini-cli' : 'claude-code'
    case 'openai':
      return 'codex'
    case 'gemini':
      return 'gemini-cli'
    case 'anthropic':
    default:
      return 'claude-code'
  }
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType,
  baseUrl: string
): CcSwitchImportConfig {
  return resolveCcSwitchImportTarget({
    baseUrl,
    platform,
    targetId: getDefaultCcSwitchTargetId(platform, clientType)
  })
}

export function resolveCcSwitchImportTarget(input: {
  baseUrl: string
  platform?: GroupPlatform | null
  targetId: CcSwitchTargetId
}): CcSwitchImportConfig {
  const target = CC_SWITCH_TARGET_DEFINITIONS[input.targetId]
  const model = input.platform === 'grok' ? GROK_CC_SWITCH_MODEL : target.model

  return {
    ...target,
    model,
    endpoint: resolveEndpoint({
      baseUrl: input.baseUrl,
      platform: input.platform,
      target
    })
  }
}

export function listCcSwitchImportTargets(input: CcSwitchTargetListInput): CcSwitchImportConfig[] {
  const platform = input.platform
  if (!platform) {
    return []
  }

  return CC_SWITCH_TARGETS_BY_PLATFORM[platform].map((targetId) =>
    resolveCcSwitchImportTarget({
      baseUrl: input.baseUrl,
      platform,
      targetId
    })
  )
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const targetId = input.targetId ?? getDefaultCcSwitchTargetId(input.platform, input.clientType)
  const config = resolveCcSwitchImportTarget({
    baseUrl: input.baseUrl,
    platform: input.platform,
    targetId
  })

  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', resolveApiBaseRoot(input.baseUrl)],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['enabled', 'true'],
    ['usageEnabled', input.usageScript ? 'true' : 'false'],
    ['usageAutoInterval', input.usageScript ? '30' : '0']
  ]

  if (config.model) {
    entries.splice(2, 0, ['model', config.model])
  }

  if (config.claudeRoleModels) {
    entries.push(['haikuModel', config.claudeRoleModels.haiku])
    entries.push(['sonnetModel', config.claudeRoleModels.sonnet])
    entries.push(['opusModel', config.claudeRoleModels.opus])
  }

  if (config.icon) {
    entries.push(['icon', config.icon])
  }

  if (input.usageScript) {
    entries.push(['usageScript', toBase64(input.usageScript)])
    entries.push(['usageBaseUrl', input.usageBaseUrl || resolveApiBaseRoot(input.baseUrl)])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
