import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

import AppHeader from '../AppHeader.vue'

const push = vi.fn()
const route = ref({
  name: 'Dashboard',
  meta: {
    title: '',
  },
  params: {},
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
  useRoute: () => route.value,
  RouterLink: {
    props: ['to'],
    template: '<a><slot /></a>',
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'dashboard.balanceApproxCny') {
        return `≈ ${params?.amount ?? ''} CNY`
      }
      return key
    },
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    contactInfo: '',
    docUrl: '',
    backendModeEnabled: false,
    cachedPublicSettings: null,
    toggleMobileSidebar: vi.fn(),
  }),
  useAuthStore: () => ({
    user: {
      id: 1,
      username: 'demo',
      email: 'demo@example.com',
      role: 'user',
      balance: 12.5,
    },
    isAdmin: false,
    isSimpleMode: false,
    canAccessTicketAdmin: false,
    logout: vi.fn(),
  }),
  useOnboardingStore: () => ({
    replay: vi.fn(),
  }),
  usePaymentStore: () => ({
    config: {
      balance_recharge_multiplier: 6.8,
    },
    fetchConfig: vi.fn().mockResolvedValue(null),
  }),
  useTicketStore: () => ({
    adminUnreadCount: 0,
    userUnreadCount: 0,
    fetchUnreadSummary: vi.fn(),
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

vi.mock('@/components/common/LocaleSwitcher.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/common/SubscriptionProgressMini.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/common/AnnouncementBell.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' },
}))

describe('AppHeader balance display', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    push.mockReset()
  })

  it('shows the user balance with CNY conversion', () => {
    const wrapper = mount(AppHeader, {
      global: {
        stubs: {
          'router-link': { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.text()).toContain('$12.50')
    expect(wrapper.text()).toContain('≈ ¥85.00 CNY')
  })
})
