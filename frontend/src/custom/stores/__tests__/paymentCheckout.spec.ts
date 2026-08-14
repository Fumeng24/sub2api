import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { usePaymentStore } from '@/stores/payment'
import type { CheckoutInfoResponse, PaymentConfig, SubscriptionPlan } from '@/types/payment'

const mockGetCheckoutInfo = vi.fn()

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo: (...args: unknown[]) => mockGetCheckoutInfo(...args),
  },
}))

vi.mock('@/custom/payment/providerConfig', () => ({
  CARD_CODE_PURCHASE_URL: '',
}))

function checkoutFixture(
  overrides: Partial<CheckoutInfoResponse> = {},
): CheckoutInfoResponse {
  return {
    methods: {},
    global_min: 0,
    global_max: 0,
    plans: [],
    balance_disabled: true,
    balance_recharge_multiplier: 1,
    subscription_usd_to_cny_rate: 0,
    recharge_fee_rate: 0,
    help_text: '',
    help_image_url: '',
    stripe_publishable_key: '',
    ...overrides,
  }
}

function paymentConfigFixture(): PaymentConfig {
  return {
    payment_enabled: true,
    min_amount: 1,
    max_amount: 1000,
    daily_limit: 0,
    max_pending_orders: 3,
    order_timeout_minutes: 15,
    balance_disabled: true,
    balance_recharge_unlock_threshold: 0,
    balance_recharge_multiplier: 1,
    subscription_usd_to_cny_rate: 0,
    enabled_payment_types: [],
    help_image_url: '',
    help_text: '',
    stripe_publishable_key: '',
  }
}

describe('usePaymentCheckoutStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('deduplicates concurrent requests and caches checkout info', async () => {
    let resolveRequest!: (value: { data: CheckoutInfoResponse }) => void
    mockGetCheckoutInfo.mockImplementation(
      () => new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )
    const store = usePaymentCheckoutStore()
    const checkout = checkoutFixture({ balance_recharge_available: true })

    const first = store.fetchCheckoutInfo()
    const second = store.fetchCheckoutInfo()
    resolveRequest({ data: checkout })

    await expect(Promise.all([first, second])).resolves.toEqual([checkout, checkout])
    await expect(store.fetchCheckoutInfo()).resolves.toEqual(checkout)
    expect(mockGetCheckoutInfo).toHaveBeenCalledTimes(1)
    expect(store.checkoutInfoLoaded).toBe(true)
  })

  it('derives purchase access from balance recharge or subscription plans', async () => {
    mockGetCheckoutInfo.mockResolvedValueOnce({ data: checkoutFixture() })
    const store = usePaymentCheckoutStore()

    await store.fetchCheckoutInfo()
    expect(store.canAccessPurchase).toBe(false)

    mockGetCheckoutInfo.mockResolvedValueOnce({
      data: checkoutFixture({
        plans: [{ id: 7 } as SubscriptionPlan],
      }),
    })
    await store.fetchCheckoutInfo(true)
    expect(store.canAccessPurchase).toBe(true)

    mockGetCheckoutInfo.mockResolvedValueOnce({
      data: checkoutFixture({
        plans: [],
        balance_recharge_available: true,
      }),
    })
    await store.fetchCheckoutInfo(true)
    expect(store.canAccessPurchase).toBe(true)
  })

  it('supports the legacy balance_disabled fallback', async () => {
    mockGetCheckoutInfo.mockResolvedValue({
      data: checkoutFixture({ balance_disabled: false }),
    })
    const store = usePaymentCheckoutStore()

    await store.fetchCheckoutInfo()

    expect(store.canRechargeBalance).toBe(true)
    expect(store.canAccessPurchase).toBe(true)
  })

  it('synchronizes plans and checkout-backed config fields to the base payment store', async () => {
    const paymentStore = usePaymentStore()
    paymentStore.config = paymentConfigFixture()
    const plans = [{ id: 9 } as SubscriptionPlan]
    mockGetCheckoutInfo.mockResolvedValue({
      data: checkoutFixture({
        plans,
        balance_disabled: false,
        balance_recharge_unlock_threshold: 200,
        balance_recharge_multiplier: 6.9,
        help_text: 'help',
        help_image_url: 'https://example.com/help.png',
        stripe_publishable_key: 'pk_test',
      }),
    })
    const store = usePaymentCheckoutStore()

    await store.fetchCheckoutInfo()

    expect(paymentStore.plans).toEqual(plans)
    expect(paymentStore.config).toMatchObject({
      balance_disabled: false,
      balance_recharge_unlock_threshold: 200,
      balance_recharge_multiplier: 6.9,
      help_text: 'help',
      help_image_url: 'https://example.com/help.png',
      stripe_publishable_key: 'pk_test',
    })
  })

  it('returns null without caching failures', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    mockGetCheckoutInfo.mockRejectedValue(new Error('Network error'))
    const store = usePaymentCheckoutStore()

    await expect(store.fetchCheckoutInfo()).resolves.toBeNull()

    expect(store.checkoutInfoLoaded).toBe(false)
    expect(store.checkoutInfoLoading).toBe(false)
    expect(consoleError).toHaveBeenCalledOnce()
    consoleError.mockRestore()
  })

  it('clear resets checkout state and ignores an older account response', async () => {
    let resolveRequest!: (value: { data: CheckoutInfoResponse }) => void
    mockGetCheckoutInfo.mockImplementation(
      () => new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )
    const paymentStore = usePaymentStore()
    const clearCurrentOrder = vi.spyOn(paymentStore, 'clearCurrentOrder')
    const store = usePaymentCheckoutStore()
    const request = store.fetchCheckoutInfo()

    store.clear()
    resolveRequest({ data: checkoutFixture({ balance_recharge_available: true }) })
    await request

    expect(store.checkoutInfo).toBeNull()
    expect(store.checkoutInfoLoaded).toBe(false)
    expect(store.checkoutInfoLoading).toBe(false)
    expect(clearCurrentOrder).toHaveBeenCalledOnce()
  })
})
