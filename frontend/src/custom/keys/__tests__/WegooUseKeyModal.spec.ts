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

import UseKeyModal from '@/custom/keys/WegooUseKeyModal.vue'

describe('UseKeyModal', () => {
  it('shows the selected service tier and all configured endpoints before examples', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-created',
        baseUrl: 'https://api.example.com/v1',
        platform: 'openai',
        groupName: 'Codex Pro',
        customEndpoints: [
          {
            name: 'Hong Kong Relay',
            endpoint: 'https://hk.example.com/v1',
            description: 'Low latency for Asia traffic'
          },
          {
            name: 'Backup',
            endpoint: 'https://backup.example.com/v1',
            description: ''
          }
        ]
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

    expect(wrapper.find('[data-testid="use-key-group-context"]').text()).toContain('Codex Pro')
    expect(wrapper.find('[data-testid="use-key-platform-context"]').text()).toContain('OpenAI')

    const endpointContext = wrapper.find('[data-testid="use-key-endpoint-context"]')
    expect(endpointContext.exists()).toBe(true)
    expect(endpointContext.text()).toContain('https://api.example.com/v1')
    expect(endpointContext.text()).toContain('Hong Kong Relay')
    expect(endpointContext.text()).toContain('https://hk.example.com/v1')
    expect(endpointContext.text()).toContain('Low latency for Asia traffic')
    expect(endpointContext.text()).toContain('Backup')
    expect(endpointContext.text()).toContain('https://backup.example.com/v1')
  })

  it('renders OpenAI and Anthropic SDK examples with the active key and Base URL', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-created',
        baseUrl: 'https://example.com/v1',
        platform: 'openai',
        allowMessagesDispatch: true
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

    const openaiSdkTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.openaiSdk')
    )
    expect(openaiSdkTab).toBeDefined()
    await openaiSdkTab!.trigger('click')
    await nextTick()

    let codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('import OpenAI from "openai"')
    expect(codeBlock.text()).toContain('apiKey: "sk-created"')
    expect(codeBlock.text()).toContain('baseURL: "https://example.com/v1"')
    expect(codeBlock.text()).toContain('model: "gpt-5.6-sol"')

    const anthropicSdkTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.anthropicSdk')
    )
    expect(anthropicSdkTab).toBeDefined()
    await anthropicSdkTab!.trigger('click')
    await nextTick()

    codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('import Anthropic from "@anthropic-ai/sdk"')
    expect(codeBlock.text()).toContain('apiKey: "sk-created"')
    expect(codeBlock.text()).toContain('baseURL: "https://example.com"')
    expect(codeBlock.text()).toContain('model: "claude-sonnet-5"')
  })

  it('renders GPT-5.6 Sol and goals feature in OpenAI Codex config', () => {
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
    expect(configToml).toContain('model = "gpt-5.6-sol"')
    expect(configToml).toContain('review_model = "gpt-5.6-sol"')
    expect(configToml).toContain('model_reasoning_effort = "xhigh"')
    expect(configToml).toContain('model_context_window = 1050000')
    expect(configToml).toContain('model_auto_compact_token_limit = 880000')
    expect(configToml).not.toContain('model = "gpt-5.4"')
    expect(configToml).toContain('[features]\ngoals = true')
  })

  it('renders GPT-5.6 Sol and goals feature in OpenAI Codex WebSocket config', async () => {
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
    expect(configToml).toContain('model = "gpt-5.6-sol"')
    expect(configToml).toContain('review_model = "gpt-5.6-sol"')
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
      'gpt-5.6',
      'gpt-5.6-sol',
      'gpt-5.6-terra',
      'gpt-5.6-luna',
      'gpt-5.5',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.3-codex-spark',
      'codex-mini-latest'
    ])
    expect(openaiProvider.models['gpt-5.4-mini'].name).toBe('GPT-5.4 Mini')
    expect(openaiProvider.models['gpt-5.4-mini'].limit).toEqual({
      context: 400000,
      output: 128000
    })
    expect(openaiProvider.models['gpt-5.4-mini'].variants).toHaveProperty('xhigh')
    expect(config.agent.build.model).toBe('openai/gpt-5.6-sol')
    expect(config.agent.build.variant).toBe('xhigh')
    expect(config.agent.plan.model).toBe('openai/gpt-5.6-sol')
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
      'claude-sonnet-5',
      'claude-opus-5',
      'claude-opus-4-6-thinking',
      'claude-sonnet-4-6'
    ])
    expect(anthropicProvider.models['claude-fable-5'].options.thinking).toEqual({
      type: 'adaptive'
    })
    expect(anthropicProvider.models['claude-sonnet-5'].limit).toEqual({
      context: 1048576,
      output: 128000
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
    expect(codeBlock.text()).toContain('GEMINI_MODEL="gemini-3.5-flash"')
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
    expect(codeBlock.text()).toContain('GEMINI_MODEL="gemini-3.5-flash"')
  })

  it('renders GPT-5.6 alias and max variants in OpenCode config', async () => {
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

    const parsed = JSON.parse(wrapper.find('pre code').text())
    const models = parsed.provider.openai.models
    for (const model of ['gpt-5.6', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']) {
      expect(models[model]).toBeDefined()
      expect(models[model].variants).toHaveProperty('max')
      expect(models[model].variants).toHaveProperty('xhigh')
    }
    expect(models['gpt-5.6'].name).toBe('GPT-5.6 (Sol)')
  })

  it('renders GPT-5.6 alias and max variants in OpenCode config', async () => {
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

    const parsed = JSON.parse(wrapper.find('pre code').text())
    const models = parsed.provider.openai.models
    for (const model of ['gpt-5.6', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']) {
      expect(models[model]).toBeDefined()
      expect(models[model].variants).toHaveProperty('max')
      expect(models[model].variants).toHaveProperty('xhigh')
    }
    expect(models['gpt-5.6'].name).toBe('GPT-5.6 (Sol)')
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
