import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountTestModal from '../WegooAccountTestModal.vue'

const { getAvailableModels, copyToClipboard } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/custom/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.imagePromptDefault': 'Generate a cute orange cat astronaut sticker on a clean pastel background.'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.imageReceived' && params?.count) {
          return `received-${params.count}`
        }
        return messages[key] || key
      }
    })
  }
})

function createStreamResponse(lines: string[]) {
  const encoder = new TextEncoder()
  const chunks = lines.map((line) => encoder.encode(line))
  let index = 0

  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn().mockImplementation(async () => {
          if (index < chunks.length) {
            return { done: false, value: chunks[index++] }
          }
          return { done: true, value: undefined }
        })
      })
    }
  } as Response
}

function mountModal(account: Record<string, unknown> = {
  id: 42,
  name: 'Gemini Image Test',
  platform: 'gemini',
  type: 'apikey',
  status: 'active'
}, show = false) {
  return mount(AccountTestModal, {
    props: {
      show,
      account
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: {
          props: ['modelValue', 'options', 'disabled'],
          template: '<div class="select-stub" :data-option-count="options.length" :data-model-value="modelValue" :data-disabled="disabled"></div>'
        },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

describe('AccountTestModal', () => {
  beforeEach(() => {
    getAvailableModels.mockResolvedValue([
      { id: 'gemini-2.0-flash', display_name: 'Gemini 2.0 Flash' },
      { id: 'gemini-2.5-flash-image', display_name: 'Gemini 2.5 Flash Image' },
      { id: 'gemini-3.1-flash-image', display_name: 'Gemini 3.1 Flash Image' }
    ])
    copyToClipboard.mockReset()
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null)),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn()
      },
      configurable: true
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-2.5-flash-image"}\n',
        'data: {"type":"image","image_url":"data:image/png;base64,QUJD","mime_type":"image/png"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('初始打开挂载时会加载 code=0 data 模型响应并启用测试', async () => {
    getAvailableModels.mockResolvedValueOnce({
      code: 0,
      data: [
        { id: 'gemini-2.5-pro', display_name: 'Gemini 2.5 Pro' },
        { id: 'gemini-3.1-flash-image', display_name: 'Gemini 3.1 Flash Image' }
      ]
    })

    const wrapper = mountModal(undefined, true)
    await flushPromises()

    expect(getAvailableModels).toHaveBeenCalledWith(42)
    expect((wrapper.vm as any).availableModels).toHaveLength(2)

    const modelSelect = wrapper.find('.select-stub')
    expect(modelSelect.attributes('data-option-count')).toBe('2')
    expect(modelSelect.attributes('data-model-value')).toBe('gemini-3.1-flash-image')

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton?.attributes('disabled')).toBeUndefined()
  })

  it('gemini 图片模型测试会携带提示词并渲染图片预览', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    await promptInput.setValue('draw a tiny orange cat astronaut')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gemini-3.1-flash-image',
      prompt: 'draw a tiny orange cat astronaut'
    })

    const preview = wrapper.find('img[alt="test-image-1"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('data:image/png;base64,QUJD')
  })

  it('Grok Imagine 模型测试会显示提示词并发送它', async () => {
    getAvailableModels.mockResolvedValueOnce([
      { id: 'grok-imagine-image', display_name: 'Grok Imagine Image' }
    ])
    const wrapper = mountModal({
      id: 44,
      name: 'Grok Image API',
      platform: 'grok',
      type: 'apikey',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    await promptInput.setValue('draw a Grok image')
    await (wrapper.vm as any).startTest()
    await flushPromises()

    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'grok-imagine-image',
      prompt: 'draw a Grok image'
    })
  })

  it('OpenAI compact 测试会携带 mode 并渲染状态消息', async () => {
    getAvailableModels.mockResolvedValueOnce([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"status","text":"compact probe ok"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 43,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    ;(wrapper.vm as any).testMode = 'compact'
    await (wrapper.vm as any).startTest()
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gpt-5.4',
      mode: 'compact'
    })
    expect(wrapper.text()).toContain('compact probe ok')
  })

  it('兼容包了一层的模型响应', async () => {
    getAvailableModels.mockResolvedValueOnce({
      data: {
        models: [
          { id: 'gemini-2.5-pro', display_name: 'Gemini 2.5 Pro' }
        ]
      }
    })

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect((wrapper.vm as any).availableModels).toEqual([
      expect.objectContaining({
        id: 'gemini-2.5-pro',
        value: 'gemini-2.5-pro',
        label: 'Gemini 2.5 Pro'
      })
    ])
    expect((wrapper.vm as any).selectedModelId).toBe('gemini-2.5-pro')
  })

  it('模型接口失败时使用平台默认模型兜底', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    getAvailableModels.mockRejectedValueOnce(new Error('network failed'))

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect((wrapper.vm as any).availableModels.length).toBeGreaterThan(0)
    expect((wrapper.vm as any).selectedModelId).toBeTruthy()
  })

  it('OpenAI Compact 探测会携带 compact 测试模式', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 42,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'compact'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gpt-5.4',
      prompt: '',
      mode: 'compact'
    })
  })
})
