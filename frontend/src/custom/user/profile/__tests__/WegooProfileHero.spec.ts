import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import WegooProfileHero from '@/custom/user/profile/WegooProfileHero.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const user = {
  id: 42,
  username: 'alice',
  email: 'alice@example.com',
  role: 'user' as const,
  balance: 10,
  concurrency: 2,
  status: 'active' as const,
  allowed_groups: null,
  created_at: '2026-04-20T00:00:00Z',
  updated_at: '2026-04-20T00:00:00Z',
}

describe('WegooProfileHero', () => {
  it('renders account identity, status, role, and trust summaries', () => {
    const wrapper = mount(WegooProfileHero, { props: { user } })

    expect(wrapper.text()).toContain('#42')
    expect(wrapper.text()).toContain('alice')
    expect(wrapper.text()).toContain('alice@example.com')
    expect(wrapper.text()).toContain('common.active')
    expect(wrapper.text()).toContain('profile.user')
    expect(wrapper.text()).toContain('profile.trust.accountSecurity')
  })

  it('uses support and disabled labels for a disabled support account', () => {
    const wrapper = mount(WegooProfileHero, {
      props: {
        user: {
          ...user,
          role: 'support',
          status: 'disabled',
        },
      },
    })

    expect(wrapper.text()).toContain('profile.gateway.supportRole')
    expect(wrapper.text()).toContain('common.disabled')
  })
})
