import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import WegooMonitorKeyPickerDialog from '@/custom/admin/monitor/WegooMonitorKeyPickerDialog.vue'

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

describe('WegooMonitorKeyPickerDialog', () => {
  it('passes the API key group id to the pricing badge', () => {
    const wrapper = mount(WegooMonitorKeyPickerDialog, {
      props: {
        show: true,
        loading: false,
        provider: 'openai',
        keys: [
          {
            id: 7,
            name: 'Monitor Key',
            key: 'sk-test-key',
            group: {
              id: 88,
              name: 'Codex Pro',
              platform: 'openai',
              subscription_type: 'standard',
              rate_multiplier: 0.3,
            },
          } as any,
        ],
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>',
          },
          GroupBadge: GroupBadgeStub,
        },
      },
    })

    expect(wrapper.findComponent(GroupBadgeStub).props('groupId')).toBe(88)
  })
})
