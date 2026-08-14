import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PlatformCostCell from '@/custom/admin/user/WegooPlatformCostCell.vue'
import PlatformUsageBreakdown from '@/custom/admin/user/WegooPlatformUsageBreakdown.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/custom/composables/useSettlementCurrency', () => ({
  useSettlementCurrency: () => ({
    formatSettlementAmount: (value: number, digits = 2) =>
      `¥${(value * 6.8).toFixed(digits)}`,
  }),
}))

describe('Wegoo platform usage cells', () => {
  it('formats platform totals with the active settlement formatter', () => {
    const wrapper = mount(PlatformCostCell, {
      props: {
        usage: {
          platform: 'openai',
          today_actual_cost: 1,
          total_actual_cost: 2,
        },
      },
    })

    expect(wrapper.text()).toContain('platformUsage.today:¥6.8000')
    expect(wrapper.text()).toContain('platformUsage.total:¥13.6000')
  })

  it('formats every breakdown row and preserves unmatched cost as other', () => {
    const wrapper = mount(PlatformUsageBreakdown, {
      props: {
        today: 3,
        total: 5,
        byPlatform: [
          { platform: 'anthropic', today_actual_cost: 1, total_actual_cost: 2 },
          { platform: 'openai', today_actual_cost: 1, total_actual_cost: 1 },
        ],
      },
      global: { stubs: { Icon: true } },
    })

    expect(wrapper.text()).toContain('platformUsage.other')
    expect(wrapper.text()).toContain('¥6.8000 / ¥13.6000')
    expect(wrapper.text()).toContain('platformUsage.today:¥20.4000')
    expect(wrapper.text()).toContain('platformUsage.total:¥34.0000')
  })
})
