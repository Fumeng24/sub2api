import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const apiMocks = vi.hoisted(() => ({
  listGroups: vi.fn(),
  updateUser: vi.fn(),
}))

vi.mock('@/custom/api/admin', () => ({
  adminAPI: {
    groups: {
      list: apiMocks.listGroups,
    },
    users: {
      update: apiMocks.updateUser,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: null,
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (!params) return key
        return key.replace(/\{(\w+)\}/g, (_, k) => String(params[k] ?? ''))
      },
    }),
  }
})

vi.mock('@/custom/common/WegooBaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title', 'width'],
    template: '<div v-if="show"><slot /><slot name="footer" /></div>',
  },
}))

vi.mock('@/components/common/PlatformIcon.vue', () => ({
  default: {
    name: 'PlatformIcon',
    props: ['platform', 'size'],
    template: '<span />',
  },
}))

import UserAllowedGroupsModal from '../WegooUserAllowedGroupsModal.vue'

const user = {
  id: 9,
  email: 'user@example.com',
  username: 'user',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [2],
  group_rates: { 2: 1.7 },
  group_discounts: {},
  notes: '',
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} as any

const groupBase = {
  description: null,
  status: 'active',
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

async function mountAndOpen() {
  const wrapper = mount(UserAllowedGroupsModal, {
    props: { show: false, user },
  })
  await wrapper.setProps({ show: true })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  apiMocks.listGroups.mockResolvedValue({
    items: [
      { ...groupBase, id: 2, name: 'Exclusive', platform: 'openai', rate_multiplier: 2, is_exclusive: true },
      { ...groupBase, id: 3, name: 'Public', platform: 'openai', rate_multiplier: 1.5, is_exclusive: false },
    ],
  })
  apiMocks.updateUser.mockResolvedValue({})
})

describe('UserAllowedGroupsModal', () => {
  it('saves only allowed groups and fixed final rates', async () => {
    const wrapper = await mountAndOpen()

    await wrapper.findAll('button').find((button) => button.text() === 'common.save')!.trigger('click')
    await flushPromises()

    expect(apiMocks.updateUser).toHaveBeenCalledTimes(1)
    expect(apiMocks.updateUser).toHaveBeenCalledWith(9, {
      allowed_groups: [2],
      group_rates: { 2: 1.7 },
    })
  })
})
