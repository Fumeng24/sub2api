import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

import AppHeader from '@/custom/layout/WegooAppHeader.vue'
import { setCurrentSettlementCurrency } from '@/custom/composables/useSettlementCurrency'

const storeMocks = vi.hoisted(() => ({
  app: {
    contactInfo: '',
    docUrl: '',
    backendModeEnabled: false,
    cachedPublicSettings: null as { payment_balance_recharge_multiplier?: number } | null,
    toggleMobileSidebar: vi.fn(),
  },
  payment: {
    config: { balance_recharge_multiplier: 1 } as { balance_recharge_multiplier?: number } | null,
    fetchConfig: vi.fn().mockResolvedValue(null),
  },
}))

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

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('en-US'),
      t: (key: string, params?: Record<string, string>) => {
        if (key === 'dashboard.balanceApproxCny') {
          return `≈ ${params?.amount ?? ''} CNY`
        }
        if (key === 'settlementCurrency.baseCredit') return 'balance credit'
        if (key === 'settlementCurrency.cny') return 'CNY'
        if (key === 'settlementCurrency.usd') return 'USD'
        return key
      },
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => storeMocks.app,
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
  usePaymentStore: () => storeMocks.payment,
}))

vi.mock('@/custom/stores/subscriptionCapability', () => ({
  useSubscriptionCapabilityStore: () => ({
    hasSubscriptionGroups: false,
    fetchSubscriptionCapability: vi.fn().mockResolvedValue(false),
  }),
}))

vi.mock('@/custom/stores/tickets', () => ({
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

vi.mock('@/custom/common/WegooSubscriptionProgressMini.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/custom/common/WegooAnnouncementBell.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' },
}))

describe('AppHeader balance display', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setCurrentSettlementCurrency('CNY')
    storeMocks.app.cachedPublicSettings = null
    storeMocks.payment.config = { balance_recharge_multiplier: 1 }
    storeMocks.payment.fetchConfig.mockClear()
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

    expect(wrapper.text()).toContain('¥12.50')
    expect(wrapper.text()).toContain('$12.50 balance credit')
  })

  it('uses the public multiplier while payment config is unavailable', () => {
    storeMocks.payment.config = null
    storeMocks.app.cachedPublicSettings = {
      payment_balance_recharge_multiplier: 1,
    }

    const wrapper = mount(AppHeader)

    expect(wrapper.text()).toContain('¥12.50')
    expect(wrapper.text()).toContain('$12.50 balance credit')
  })

  it('does not invent a CNY conversion when no valid multiplier is available', () => {
    storeMocks.payment.config = null
    storeMocks.app.cachedPublicSettings = {
      payment_balance_recharge_multiplier: 0,
    }

    const wrapper = mount(AppHeader)

    expect(wrapper.text()).toContain('--')
    expect(wrapper.text()).toContain('$12.50 balance credit')
    expect(wrapper.text()).not.toContain('¥85.00')
  })
})
