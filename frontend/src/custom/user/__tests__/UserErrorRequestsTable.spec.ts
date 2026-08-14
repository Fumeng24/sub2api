import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UserErrorRequestsTable from '@/custom/user/WegooUserErrorRequestsTable.vue'
import type { UserErrorRequest } from '@/types'

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: { value: 'en' },
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params?.count != null) return `${key}:${params.count}`
      return key
    }
  })
}))

vi.mock('@/api/usage', () => ({
  getMyErrorDetail: vi.fn()
}))

const rows: UserErrorRequest[] = [
  {
    id: 88,
    created_at: '2026-06-27T12:00:00Z',
    model: 'gpt-5.5',
    inbound_endpoint: '/v1/responses',
    status_code: 503,
    category: 'service_unavailable',
    platform: 'openai',
    message: 'Our servers are currently overloaded. Please try again later.',
    key_name: 'prod-key',
    key_deleted: false
  }
]

function mountTable() {
  return mount(UserErrorRequestsTable, {
    props: {
      rows,
      total: rows.length,
      loading: false,
      page: 1,
      pageSize: 20,
      apiKeys: []
    },
    global: {
      stubs: {
        Select: {
          props: ['modelValue', 'options', 'placeholder'],
          emits: ['update:modelValue', 'change'],
          template: '<select><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select>'
        },
        Pagination: {
          template: '<div data-testid="pagination" />'
        },
        IpGeoBatchToolbar: true,
        IpGeoCell: true,
        Icon: {
          template: '<span />'
        },
        UserErrorDetailModal: {
          props: ['show', 'errorId'],
          template: '<div v-if="show" data-testid="stub-error-detail-modal">{{ errorId }}</div>'
        }
      }
    }
  })
}

describe('UserErrorRequestsTable', () => {
  it('renders compact mobile cards with the same issue context as the desktop table', () => {
    const wrapper = mountTable()

    const card = wrapper.get('.data-table-mobile-card')
    expect(card.text()).toContain('gpt-5.5')
    expect(card.text()).toContain('/v1/responses')
    expect(card.text()).toContain('503')
    expect(card.text()).toContain('prod-key')
    expect(card.text()).toContain('usage.errors.categories.service_unavailable')
    expect(card.text()).toContain('openai')
    expect(card.text()).toContain('Our servers are currently overloaded')
  })

  it('opens detail and emits ticket creation from mobile card actions', async () => {
    const wrapper = mountTable()

    await wrapper.get('[data-testid="usage-error-detail-88"]').trigger('click')
    expect(wrapper.get('[data-testid="stub-error-detail-modal"]').text()).toContain('88')

    const buttons = wrapper.findAll('.data-table-mobile-actions button')
    await buttons[1].trigger('click')
    expect(wrapper.emitted('createTicket')?.[0]).toEqual([rows[0]])
  })
})
