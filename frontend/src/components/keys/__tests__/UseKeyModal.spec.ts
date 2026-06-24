import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true)
  })
}))

import UseKeyModal from '../UseKeyModal.vue'

describe('UseKeyModal', () => {
  it('renders GPT-5.5 and goals feature in OpenAI Codex config', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    const configToml = codeBlocks.find((content) => content.includes('model_provider = "OpenAI"'))

    expect(configToml).toBeDefined()
    expect(configToml).toContain('model = "gpt-5.5"')
    expect(configToml).toContain('review_model = "gpt-5.5"')
    expect(configToml).toContain('model_reasoning_effort = "xhigh"')
    expect(configToml).toContain('model_context_window = 1050000')
    expect(configToml).toContain('model_auto_compact_token_limit = 880000')
    expect(configToml).not.toContain('model = "gpt-5.4"')
    expect(configToml).toContain('[features]\ngoals = true')
  })

  it('renders GPT-5.5 and goals feature in OpenAI Codex WebSocket config', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const wsTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.codexCliWs')
    )

    expect(wsTab).toBeDefined()
    await wsTab!.trigger('click')
    await nextTick()

    const codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    const configToml = codeBlocks.find((content) => content.includes('supports_websockets = true'))

    expect(configToml).toBeDefined()
    expect(configToml).toContain('model = "gpt-5.5"')
    expect(configToml).toContain('review_model = "gpt-5.5"')
    expect(configToml).toContain('model_reasoning_effort = "xhigh"')
    expect(configToml).toContain('model_context_window = 1050000')
    expect(configToml).toContain('model_auto_compact_token_limit = 880000')
    expect(configToml).not.toContain('model = "gpt-5.4"')
    expect(configToml).toContain('[features]\nresponses_websockets_v2 = true\ngoals = true')
  })

  it('renders current Codex models and provider options in OpenCode config', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    const config = JSON.parse(codeBlock.text())
    const openaiProvider = config.provider.openai

    expect(openaiProvider.npm).toBe('@ai-sdk/openai')
    expect(openaiProvider.name).toBe('Tomato Codex / GPT')
    expect(openaiProvider.options).toMatchObject({
      baseURL: 'https://example.com/v1',
      apiKey: 'sk-test'
    })
    expect(Object.keys(openaiProvider.models)).toEqual([
      'gpt-5.5',
      'gpt-5.4',
      'gpt-5.4-mini'
    ])
    expect(openaiProvider.models['gpt-5.4-mini'].name).toBe('GPT-5.4 Mini')
    expect(openaiProvider.models['gpt-5.4-mini'].limit).toEqual({
      context: 400000,
      output: 128000
    })
    expect(openaiProvider.models['gpt-5.4-mini'].variants).toHaveProperty('xhigh')
    expect(config.agent.build.model).toBe('openai/gpt-5.5')
    expect(config.agent.build.variant).toBe('xhigh')
    expect(config.agent.plan.model).toBe('openai/gpt-5.5')
    expect(config.agent.plan.variant).toBe('xhigh')
    expect(config.agent.build.options.store).toBe(false)
    expect(config.agent.plan.options.store).toBe(false)
  })

  it('renders Claude models in OpenCode Anthropic config', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-claude',
        baseUrl: 'https://example.com/v1',
        platform: 'anthropic'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    const config = JSON.parse(codeBlock.text())
    const anthropicProvider = config.provider.anthropic

    expect(anthropicProvider.npm).toBe('@ai-sdk/anthropic')
    expect(anthropicProvider.name).toBe('Tomato Claude')
    expect(anthropicProvider.options).toMatchObject({
      baseURL: 'https://example.com/v1',
      apiKey: 'sk-claude'
    })
    expect(Object.keys(anthropicProvider.models)).toEqual([
      'claude-fable-5',
      'claude-opus-4-6-thinking',
      'claude-sonnet-4-6'
    ])
    expect(anthropicProvider.models['claude-fable-5'].options.thinking).toEqual({
      type: 'adaptive'
    })
    expect(anthropicProvider.models['claude-sonnet-4-6'].limit).toEqual({
      context: 200000,
      output: 64000
    })
    expect(anthropicProvider.models['claude-opus-4-6-thinking'].options.thinking).toEqual({
      budgetTokens: 24576,
      type: 'enabled'
    })
    expect(config.agent).toBeUndefined()
  })

  it('renders Gemini CLI base URL with v1beta', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-gemini',
        baseUrl: 'https://example.com/v1',
        platform: 'gemini'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('GOOGLE_GEMINI_BASE_URL="https://example.com/v1beta"')
    expect(codeBlock.text()).toContain('GEMINI_MODEL="gemini-2.0-flash"')
  })

  it('renders Antigravity Gemini CLI base URL with antigravity v1beta', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-antigravity',
        baseUrl: 'https://example.com/v1',
        platform: 'antigravity'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const geminiTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.geminiCli')
    )

    expect(geminiTab).toBeDefined()
    await geminiTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('GOOGLE_GEMINI_BASE_URL="https://example.com/antigravity/v1beta"')
    expect(codeBlock.text()).toContain('GEMINI_MODEL="gemini-2.0-flash"')
  })

  it('renders Claude Fable 5 OpenCode config with adaptive thinking', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'antigravity'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const claudeConfig = wrapper.findAll('pre code')
      .map((code) => code.text())
      .find((content) => content.includes('"antigravity-claude"'))

    expect(claudeConfig).toBeDefined()
    const parsed = JSON.parse(claudeConfig!)
    const fable = parsed.provider['antigravity-claude'].models['claude-fable-5']

    expect(fable.name).toBe('Claude Fable 5')
    expect(fable.limit).toEqual({ context: 1048576, output: 128000 })
    expect(fable.options.thinking).toEqual({ type: 'adaptive' })
    expect(fable.options.thinking).not.toHaveProperty('budgetTokens')
  })
})
