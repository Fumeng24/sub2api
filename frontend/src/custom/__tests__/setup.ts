import { config } from '@vue/test-utils'
import { vi } from 'vitest'

vi.mock('@/api/admin/affiliates', () => {
  const affiliatesAPI = {
    listUsers: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    lookupUsers: vi.fn().mockResolvedValue([]),
    updateUserSettings: vi.fn().mockResolvedValue(undefined),
    clearUserSettings: vi.fn().mockResolvedValue(undefined),
    batchSetRate: vi.fn().mockResolvedValue(undefined)
  }

  return { affiliatesAPI, default: affiliatesAPI }
})

type VueTestConfig = {
  global: {
    stubs: Record<string, unknown>
  }
}

const originalConsoleWarn = console.warn.bind(console)
const originalConsoleError = console.error.bind(console)

const expectedErrorPrefixes = [
  'Table load error:',
  'Failed to parse saved user data:',
  'Failed to fetch active subscriptions:',
  '[OpsOpenAITokenStatsCard] Failed to load data'
]

function installConsoleWarnFilter() {
  console.warn = (...args: unknown[]) => {
    const message = typeof args[0] === 'string' ? args[0] : ''
    if (message.startsWith('[intlify] The message format compilation is not supported in this build.')) {
      return
    }
    originalConsoleWarn(...args)
  }
}

function installConsoleErrorFilter() {
  console.error = (...args: unknown[]) => {
    const message = typeof args[0] === 'string' ? args[0] : ''
    if (expectedErrorPrefixes.some(prefix => message.startsWith(prefix))) {
      return
    }
    originalConsoleError(...args)
  }
}

function createMediaQueryList(query: string, matches = false): MediaQueryList {
  const listeners = new Set<(event: MediaQueryListEvent) => void>()
  return {
    media: query,
    matches,
    onchange: null,
    addEventListener: vi.fn((_type: string, listener: EventListenerOrEventListenerObject) => {
      if (typeof listener === 'function') {
        listeners.add(listener as (event: MediaQueryListEvent) => void)
      }
    }),
    removeEventListener: vi.fn((_type: string, listener: EventListenerOrEventListenerObject) => {
      if (typeof listener === 'function') {
        listeners.delete(listener as (event: MediaQueryListEvent) => void)
      }
    }),
    addListener: vi.fn(listener => listeners.add(listener)),
    removeListener: vi.fn(listener => listeners.delete(listener)),
    dispatchEvent: vi.fn(event => {
      listeners.forEach(listener => listener(event as MediaQueryListEvent))
      return true
    })
  } as unknown as MediaQueryList
}

export function applyWegooTestSetup(config: VueTestConfig) {
  installConsoleWarnFilter()
  installConsoleErrorFilter()
  if (typeof window !== 'undefined') {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      writable: true,
      value: vi.fn((query: string) => createMediaQueryList(query))
    })
  }

  config.global.stubs = {
    RouterLink: {
      name: 'RouterLinkStub',
      inheritAttrs: false,
      template: '<a><slot /></a>'
    },
    AmountInput: {
      name: 'AmountInputStub',
      inheritAttrs: false,
      props: [
        'modelValue',
        'amounts',
        'min',
        'max',
        'prefix',
        'amountLabel',
        'amountDescription',
        'disabledReason'
      ],
      emits: ['update:modelValue'],
      template: '<div class="amount-input-stub" />'
    }
  }
}

applyWegooTestSetup(config)
