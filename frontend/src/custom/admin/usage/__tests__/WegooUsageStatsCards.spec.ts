import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UsageStatsCards from '@/custom/admin/usage/WegooUsageStatsCards.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const stats = {
  total_requests: 1,
  total_input_tokens: 100,
  total_output_tokens: 50,
  total_cache_tokens: 34,
  total_cache_creation_tokens: 12,
  total_cache_read_tokens: 22,
  total_tokens: 184,
  total_cost: 0.002,
  total_actual_cost: 0.001,
  total_account_cost: 0.0008,
  average_duration_ms: 250,
}

describe('WegooUsageStatsCards', () => {
  it('shows account cost by default when the admin response provides it', () => {
    const wrapper = mount(UsageStatsCards, { props: { stats } })

    expect(wrapper.text()).toContain('usage.accountCost $0.0008')
    expect(wrapper.text()).toContain('usage.standardCost $0.0020')
  })

  it('hides account cost and strikes standard cost for the user usage page', () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        stats,
        showAccountCost: false,
        strikeStandardCost: true,
      },
    })

    expect(wrapper.text()).not.toContain('usage.accountCost')
    expect(wrapper.get('.line-through').text()).toBe('$0.0020')
  })

  it('does not invent a zero account cost when the field is absent', () => {
    const { total_account_cost: _accountCost, ...userStats } = stats
    const wrapper = mount(UsageStatsCards, { props: { stats: userStats } })

    expect(wrapper.text()).not.toContain('usage.accountCost')
  })
})
