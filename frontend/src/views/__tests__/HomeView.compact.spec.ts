import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '@/custom/home/WegooHomeView.vue'

const { appStore, authStore, getPublicChannelMonitors } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    apiBaseUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn(),
  },
  getPublicChannelMonitors: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('@/custom/api/publicGateway', () => ({
  default: {
    getPublicChannelMonitors,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh-CN' } }),
  }
})

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: () => ({ push: vi.fn() }),
  }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = {
    site_name: 'Test site',
    site_subtitle: 'Test subtitle',
    ...settings,
  }

  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' },
      },
    },
  })
}

function compactDestination(wrapper: ReturnType<typeof mountHome>) {
  return wrapper.get('[data-testid="compact-home"]').findComponent(RouterLinkStub).props('to')
}

describe('HomeView compact mode', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    authStore.user = null
    authStore.checkAuth.mockClear()
    appStore.fetchPublicSettings.mockClear()
    appStore.apiBaseUrl = ''
    getPublicChannelMonitors.mockReset()
    getPublicChannelMonitors.mockResolvedValue({ items: [] })
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })

  it('renders custom HTML ahead of compact mode', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      home_content: '<section id="custom-home">Custom home</section>',
    })

    expect(wrapper.get('#custom-home').text()).toBe('Custom home')
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('renders custom URL content ahead of compact mode', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      home_content: ' https://example.com/home ',
    })

    expect(wrapper.get('iframe').attributes('src')).toBe('https://example.com/home')
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('treats whitespace-only custom content as empty and selects compact mode', () => {
    const wrapper = mountHome({ compact_home_enabled: true, home_content: ' \n\t ' })

    expect(wrapper.get('[data-testid="compact-home"]').text()).toContain('Test site')
  })

  it.each([undefined, false])('selects the default home when compact mode is %s', (enabled) => {
    const settings = enabled === undefined ? {} : { compact_home_enabled: enabled }
    const wrapper = mountHome(settings)

    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
    expect(wrapper.find('.gateway-home').exists()).toBe(true)
    expect(wrapper.find('.gateway-code-window--hero').exists()).toBe(true)
  })

  it('links unauthenticated visitors to login', () => {
    expect(compactDestination(mountHome({ compact_home_enabled: true }))).toBe('/login')
  })

  it('links authenticated users to their dashboard', () => {
    authStore.isAuthenticated = true

    expect(compactDestination(mountHome({ compact_home_enabled: true }))).toBe('/dashboard')
  })

  it('links administrators to the admin dashboard', () => {
    authStore.isAuthenticated = true
    authStore.isAdmin = true

    const wrapper = mountHome({ compact_home_enabled: true })
    expect(compactDestination(wrapper)).toBe('/admin/dashboard')
    expect(authStore.checkAuth).toHaveBeenCalledOnce()
    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
  })

  it('builds every OpenAI example from the configured Base URL without duplicate v1', () => {
    appStore.apiBaseUrl = 'https://configured.example/v1/'

    const wrapper = mountHome()
    const heroExample = wrapper.get('.gateway-code-window--hero code').text()

    expect(heroExample).toContain('https://configured.example/v1/chat/completions')
    expect(heroExample).not.toContain('/v1/v1')
    expect(wrapper.html()).not.toContain('api.wegoo.site')
  })

  it('renders live monitor data instead of static provider health', async () => {
    getPublicChannelMonitors.mockResolvedValue({
      items: [{
        id: 7,
        name: 'Codex realtime',
        provider: 'openai',
        group_name: 'codex',
        primary_model: 'gpt-5.5',
        primary_status: 'degraded',
        primary_latency_ms: 438,
        primary_ping_latency_ms: 51,
        availability_7d: 98.6,
        extra_models: [],
        timeline: [{
          status: 'failed',
          latency_ms: 900,
          ping_latency_ms: 51,
          checked_at: '2026-08-11T00:00:00Z',
        }],
      }],
    })

    const wrapper = mountHome()
    await flushPromises()

    const status = wrapper.get('[data-testid="homepage-live-status"]')
    expect(status.text()).toContain('Codex realtime')
    expect(status.text()).toContain('服务波动')
    expect(status.text()).toContain('438ms')
    expect(status.find('.gateway-history .is-failed').exists()).toBe(true)
  })

  it('does not overwrite the globally initialized theme on mount', () => {
    localStorage.setItem('theme', 'light')

    mountHome()

    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
  })

  it('persists theme changes from the homepage toggle', async () => {
    const wrapper = mountHome()

    const toggle = wrapper.get('button[aria-label="切换到深色模式"]')
    await toggle.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(toggle.attributes('aria-label')).toBe('切换到浅色模式')
  })
})
