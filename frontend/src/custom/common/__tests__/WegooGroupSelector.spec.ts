import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import WegooGroupSelector from '@/custom/common/WegooGroupSelector.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const GroupBadgeStub = defineComponent({
  name: 'GroupBadge',
  props: { groupId: Number },
  template: '<span class="group-badge-stub">{{ groupId }}</span>',
})

describe('WegooGroupSelector', () => {
  it('passes the selected group id to the pricing badge', () => {
    const wrapper = mount(WegooGroupSelector, {
      props: {
        modelValue: [],
        groups: [
          {
            id: 42,
            name: 'Codex Plus',
            platform: 'openai',
            subscription_type: 'standard',
            rate_multiplier: 0.2,
            account_count: 2,
          } as any,
        ],
      },
      global: {
        stubs: {
          GroupBadge: GroupBadgeStub,
          Icon: true,
        },
      },
    })

    expect(wrapper.findComponent(GroupBadgeStub).props('groupId')).toBe(42)
  })
})
