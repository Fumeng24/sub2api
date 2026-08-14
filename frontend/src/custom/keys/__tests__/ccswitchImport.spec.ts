import { describe, expect, it } from 'vitest'
import {
  CLAUDE_CC_SWITCH_HAIKU_MODEL,
  CLAUDE_CC_SWITCH_MODEL,
  CLAUDE_CC_SWITCH_OPUS_MODEL,
  CLAUDE_CC_SWITCH_SONNET_MODEL,
  GEMINI_CC_SWITCH_MODEL,
  GROK_CC_SWITCH_MODEL,
  OPENAI_CC_SWITCH_CODEX_MODEL,
  buildCcSwitchImportDeeplink,
  listCcSwitchImportTargets,
  resolveApiBaseRoot
} from '@/custom/keys/ccswitchImport'

function paramsFromDeeplink(deeplink: string): URLSearchParams {
  const query = deeplink.split('?')[1] || ''
  return new URLSearchParams(query)
}

function decodeUsageScript(params: URLSearchParams): string {
  const value = params.get('usageScript') || ''
  const binary = atob(value)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

describe('ccswitchImport utils', () => {
  it('defaults OpenAI CC Switch imports to the current Codex model', () => {
    expect(OPENAI_CC_SWITCH_CODEX_MODEL).toBe('gpt-5.6-sol')
  })

  const baseInput = {
    baseUrl: 'https://api.example.com',
    providerName: 'Sub2API',
    apiKey: 'sk-test',
    usageScript: 'return "余额正常"'
  }

  it('builds a Codex provider deeplink for OpenAI imports', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'openai',
        targetId: 'codex'
      })
    )

    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe(baseInput.baseUrl)
    expect(params.get('model')).toBe(OPENAI_CC_SWITCH_CODEX_MODEL)
    expect(params.get('model')).toBe('gpt-5.6-sol')
    expect(params.has('haikuModel')).toBe(false)
    expect(params.get('enabled')).toBe('true')
    expect(params.get('usageEnabled')).toBe('true')
    expect(params.get('usageAutoInterval')).toBe('30')
    expect(params.get('usageBaseUrl')).toBe(baseInput.baseUrl)
    expect(decodeUsageScript(params)).toBe(baseInput.usageScript)
  })

  it('uses the Grok model for Grok-compatible imports', () => {
    const targets = listCcSwitchImportTargets({
      baseUrl: 'https://api.example.com/v1',
      platform: 'grok'
    })

    expect(targets.map((target) => target.model)).toEqual(
      targets.map(() => GROK_CC_SWITCH_MODEL)
    )
  })

  it('normalizes usage base URL independently from app endpoint suffixes', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        baseUrl: 'https://api.example.com/v1',
        platform: 'openai',
        targetId: 'opencode'
      })
    )

    expect(params.get('endpoint')).toBe('https://api.example.com/v1')
    expect(params.get('usageBaseUrl')).toBe('https://api.example.com')
  })

  it('normalizes base URLs that already include an API suffix', () => {
    expect(resolveApiBaseRoot('https://api.example.com/v1')).toBe('https://api.example.com')
    expect(resolveApiBaseRoot('https://api.example.com/v1beta/')).toBe('https://api.example.com')
  })

  it('lists OpenAI-compatible CC Switch targets with the right endpoints', () => {
    const targets = listCcSwitchImportTargets({
      baseUrl: 'https://api.example.com/v1',
      platform: 'openai'
    })

    expect(targets.map((target) => target.targetId)).toEqual([
      'codex',
      'opencode',
      'openclaw',
      'hermes'
    ])
    expect(targets.map((target) => target.app)).toEqual([
      'codex',
      'opencode',
      'openclaw',
      'hermes'
    ])
    expect(targets[0].endpoint).toBe('https://api.example.com')
    expect(targets[0].protocol).toEqual({
      mode: 'openai-responses',
      support: 'native'
    })
    expect(targets[0].reasoningEffort).toBe('high')
    expect(targets.slice(1).map((target) => target.endpoint)).toEqual([
      'https://api.example.com/v1',
      'https://api.example.com/v1',
      'https://api.example.com/v1'
    ])
    expect(targets.slice(1).map((target) => target.protocol)).toEqual([
      {
        mode: 'openai-compatible',
        support: 'compatibility'
      },
      {
        mode: 'openai-completions',
        support: 'compatibility'
      },
      {
        mode: 'chat-completions',
        support: 'compatibility'
      }
    ])
    expect(targets.every((target) => target.model === OPENAI_CC_SWITCH_CODEX_MODEL)).toBe(true)
  })

  it('lists Claude targets using the CC Switch Claude provider app value', () => {
    const targets = listCcSwitchImportTargets({
      baseUrl: 'https://api.example.com',
      platform: 'anthropic'
    })

    expect(targets.map((target) => target.targetId)).toEqual(['claude-code', 'claude-desktop'])
    expect(targets.map((target) => target.app)).toEqual(['claude', 'claude'])
    expect(targets.map((target) => target.endpoint)).toEqual([
      'https://api.example.com',
      'https://api.example.com'
    ])
    expect(targets.map((target) => target.protocol.mode)).toEqual([
      'anthropic-messages',
      'anthropic-messages'
    ])
    expect(targets.map((target) => target.claudeRoleModels)).toEqual([
      {
        haiku: CLAUDE_CC_SWITCH_HAIKU_MODEL,
        sonnet: CLAUDE_CC_SWITCH_SONNET_MODEL,
        opus: CLAUDE_CC_SWITCH_OPUS_MODEL
      },
      {
        haiku: CLAUDE_CC_SWITCH_HAIKU_MODEL,
        sonnet: CLAUDE_CC_SWITCH_SONNET_MODEL,
        opus: CLAUDE_CC_SWITCH_OPUS_MODEL
      }
    ])
    expect(targets.every((target) => target.model === CLAUDE_CC_SWITCH_MODEL)).toBe(true)
  })

  it('emits the latest CC Switch Claude role model parameters', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'anthropic',
        targetId: 'claude-code'
      })
    )

    expect(params.get('app')).toBe('claude')
    expect(params.get('model')).toBe(CLAUDE_CC_SWITCH_MODEL)
    expect(params.get('haikuModel')).toBe(CLAUDE_CC_SWITCH_HAIKU_MODEL)
    expect(params.get('sonnetModel')).toBe(CLAUDE_CC_SWITCH_SONNET_MODEL)
    expect(params.get('opusModel')).toBe(CLAUDE_CC_SWITCH_OPUS_MODEL)
    expect(params.get('sonnetModel')).toBe('claude-sonnet-5')
    expect(params.get('opusModel')).toBe('claude-opus-5')
  })

  it('builds Gemini CLI targets on the v1beta endpoint', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        platform: 'gemini',
        targetId: 'gemini-cli'
      })
    )

    expect(params.get('app')).toBe('gemini')
    expect(params.get('endpoint')).toBe('https://api.example.com/v1beta')
    expect(params.get('model')).toBe(GEMINI_CC_SWITCH_MODEL)
    expect(params.has('haikuModel')).toBe(false)
  })

  it('keeps Antigravity Claude and Gemini targets on their dedicated endpoints', () => {
    const targets = listCcSwitchImportTargets({
      baseUrl: 'https://api.example.com/v1',
      platform: 'antigravity'
    })

    expect(targets.map((target) => [target.targetId, target.app, target.endpoint])).toEqual([
      ['claude-code', 'claude', 'https://api.example.com/antigravity'],
      ['claude-desktop', 'claude', 'https://api.example.com/antigravity'],
      ['gemini-cli', 'gemini', 'https://api.example.com/antigravity/v1beta']
    ])
  })

  it('disables usage polling when no usage script is supplied', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        baseUrl: baseInput.baseUrl,
        providerName: baseInput.providerName,
        apiKey: baseInput.apiKey,
        platform: 'openai',
        targetId: 'opencode'
      })
    )

    expect(params.get('app')).toBe('opencode')
    expect(params.get('usageEnabled')).toBe('false')
    expect(params.get('usageAutoInterval')).toBe('0')
    expect(params.has('usageScript')).toBe(false)
  })
})
