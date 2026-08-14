import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import InvoicesView from '../InvoicesView.vue'

const invoiceSummary = vi.hoisted(() => vi.fn())
const invoiceList = vi.hoisted(() => vi.fn())
const invoiceTemplates = vi.hoisted(() => vi.fn())
const myOrders = vi.hoisted(() => vi.fn())

vi.mock('@/custom/api/invoices', () => ({
  default: {
    getSummary: invoiceSummary,
    list: invoiceList,
    listTemplates: invoiceTemplates,
    create: vi.fn(),
    cancel: vi.fn(),
  },
  invoicesAPI: {
    getSummary: invoiceSummary,
    list: invoiceList,
    listTemplates: invoiceTemplates,
    create: vi.fn(),
    cancel: vi.fn(),
  },
}))

vi.mock('@/custom/api/payment', () => ({
  paymentAPI: {
    getMyOrders: myOrders,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({ query: {} }),
    useRouter: () => ({ replace: vi.fn() }),
  }
})

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

function mountView() {
  setActivePinia(createPinia())
  return shallowMount(InvoicesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { template: '<div><slot /></div>' },
        DataTable: { template: '<div />' },
        Pagination: { template: '<div />' },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue', 'change'],
          template: '<select />',
        },
        Icon: true,
      },
    },
  })
}

describe('InvoicesView', () => {
  beforeEach(() => {
    invoiceSummary.mockReset().mockResolvedValue({
      data: {
        recharge_amount: 1500,
        invoiced_amount: 0,
        locked_amount: 0,
        available_amount: 1500,
        min_amount: 100,
        tax_rate: 0.03,
        tax_rate_percent: 3,
        min_tax_fee: 0,
        tax_fee_threshold: 0,
        can_apply: true,
        current_balance: 1288.88,
        invoiceable_basis: 'net_recharge',
      },
    })
    invoiceList.mockReset().mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 10, pages: 0 } })
    invoiceTemplates.mockReset().mockResolvedValue({ data: [] })
    myOrders.mockReset().mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 } })
  })

  it('loads invoiceable orders by default and can switch to unavailable orders', async () => {
    const wrapper = mountView()
    await flushPromises()
    await flushPromises()

    expect(myOrders).toHaveBeenCalled()
    expect(myOrders.mock.calls[0][0]).toMatchObject({ invoiceable: true })

    await wrapper.setData?.({})
    const instance = wrapper.vm as unknown as { orderFilters: { invoiceability: string }; handleOrderServerFilterChange: () => Promise<void> | void }
    instance.orderFilters.invoiceability = 'unavailable'
    await instance.handleOrderServerFilterChange()

    expect(myOrders.mock.calls.at(-1)?.[0]).toMatchObject({ invoiceable: false })
  })
})
