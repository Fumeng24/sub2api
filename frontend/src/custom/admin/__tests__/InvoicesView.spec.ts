import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import InvoicesView from '../InvoicesView.vue'

const adminInvoicesList = vi.hoisted(() => vi.fn())

vi.mock('@/custom/api/admin/invoices', () => ({
  default: {
    list: adminInvoicesList,
    getById: vi.fn(),
    approve: vi.fn(),
    reject: vi.fn(),
    complete: vi.fn(),
  },
  adminInvoicesAPI: {
    list: adminInvoicesList,
    getById: vi.fn(),
    approve: vi.fn(),
    reject: vi.fn(),
    complete: vi.fn(),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' },
    }),
  }
})

describe('admin InvoicesView', () => {
  const invoiceRow = {
    id: 7,
    user_id: 11,
    user_email: 'billing@example.com',
    user_name: 'Billing User',
    status: 'pending',
    invoice_type: 'company_vat_general',
    title: 'Wegoo AI',
    tax_id: 'ABCD1234',
    item_name: '信息技术服务费',
    amount: 1000,
    tax_rate: 0.02,
    tax_fee: 50,
    receiver_email: 'receiver@example.com',
    note: '',
    admin_note: '',
    invoice_no: '',
    source_order_count: 2,
    source_orders_json: [
      {
        id: 1,
        record_source: 'payment_order',
        business_category: '用户充值',
        payment_type: '支付宝',
        out_trade_no: 'TRADE-1',
        amount: 700,
        refund_amount: 0,
        invoiceable: true,
      },
    ],
    completed_at: null,
    rejected_at: null,
    approved_at: null,
    processed_by: null,
    created_at: '2026-06-01T12:00:00Z',
    updated_at: '2026-06-01T12:00:00Z',
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    adminInvoicesList.mockReset().mockResolvedValue({
      data: {
        items: [invoiceRow],
        total: 1,
        page: 1,
        page_size: 20,
        pages: 1,
      },
    })
  })

  it('renders structured source order snapshots in admin detail panel', async () => {
    const wrapper = mount(InvoicesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: { template: '<div v-if=\"show\"><slot /></div>', props: ['show', 'title', 'width'] },
          Pagination: true,
          Select: true,
          Icon: true,
        },
      },
    })

    await flushPromises()
    const detailButton = wrapper.findAll('button').find((button) => button.text() === '详情')
    expect(detailButton).toBeTruthy()
    await detailButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('来源订单快照')
    expect(wrapper.text()).toContain('支付订单')
    expect(wrapper.text()).toContain('退款抵扣')
  })
})
