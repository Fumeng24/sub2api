import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import type { UserErrorRequest } from '@/types'

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: { value: 'en' },
      t: (key: string) => key,
    },
  }),
  useI18n: () => ({ t: (key: string) => key }),
}))

const row: UserErrorRequest = {
  id: 88,
  created_at: '2026-06-27T12:00:00Z',
  model: 'gpt-5.5',
  inbound_endpoint: '/v1/responses',
  status_code: 503,
  category: 'service_unavailable',
  platform: 'openai',
  message: 'Our servers are currently overloaded. Please try again later.',
  key_name: 'prod-key',
  key_deleted: false,
}

describe('UserErrorRequestsTable', () => {
  beforeEach(() => {
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: true,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
  })

  it('emits the selected error when creating a ticket', async () => {
    const wrapper = mount(UserErrorRequestsTable, {
      props: {
        rows: [row],
        total: 1,
        loading: false,
        page: 1,
        pageSize: 20,
      },
      global: {
        stubs: {
          IpGeoBatchToolbar: true,
          IpGeoCell: true,
          Pagination: true,
          UserErrorDetailModal: true,
          Icon: true,
        },
      },
    })

    await wrapper.get('[data-testid="usage-error-ticket-88"]').trigger('click')

    expect(wrapper.emitted('createTicket')?.[0]).toEqual([row])
  })
})
