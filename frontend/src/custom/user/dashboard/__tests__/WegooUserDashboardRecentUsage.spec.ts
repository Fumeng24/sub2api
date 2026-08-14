import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import RecentUsage from '../WegooUserDashboardRecentUsage.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/custom/composables/useSettlementCurrency', () => ({
  useSettlementCurrency: () => ({
    formatSettlementAmount: (value: number) => String(value),
  }),
}))

describe('WegooUserDashboardRecentUsage', () => {
  it('shows the upstream model when a request was mapped', () => {
    const wrapper = mount(RecentUsage, {
      props: {
        loading: false,
        data: [{
          id: 1,
          model: 'gpt-5.6-sol',
          upstream_model: 'gpt-5.6-luna',
          created_at: '2026-08-12T00:00:00Z',
          actual_cost: 0,
          total_cost: 0,
          input_tokens: 1,
          output_tokens: 1,
        } as any],
      },
      global: {
        stubs: {
          Icon: true,
          EmptyState: true,
          LoadingSpinner: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.text()).toContain('gpt-5.6-sol')
    expect(wrapper.text()).toContain('gpt-5.6-luna')
  })
})
