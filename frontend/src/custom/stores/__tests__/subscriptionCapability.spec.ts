import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'

const mockGetSubscriptionCapability = vi.fn()

vi.mock('@/custom/api/groups', () => ({
  default: {
    getSubscriptionCapability: (...args: unknown[]) => mockGetSubscriptionCapability(...args),
  },
}))

describe('useSubscriptionCapabilityStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads and caches an available subscription capability', async () => {
    mockGetSubscriptionCapability.mockResolvedValue({ has_subscription_groups: true })
    const store = useSubscriptionCapabilityStore()

    await expect(store.fetchSubscriptionCapability()).resolves.toBe(true)
    await expect(store.fetchSubscriptionCapability()).resolves.toBe(true)

    expect(store.hasSubscriptionGroups).toBe(true)
    expect(store.capabilityLoaded).toBe(true)
    expect(mockGetSubscriptionCapability).toHaveBeenCalledTimes(1)
  })

  it('records an unavailable subscription capability', async () => {
    mockGetSubscriptionCapability.mockResolvedValue({ has_subscription_groups: false })
    const store = useSubscriptionCapabilityStore()

    await expect(store.fetchSubscriptionCapability()).resolves.toBe(false)

    expect(store.hasSubscriptionGroups).toBe(false)
    expect(store.capabilityLoaded).toBe(true)
  })

  it('deduplicates concurrent non-forced requests', async () => {
    let resolveRequest!: (value: { has_subscription_groups: boolean }) => void
    mockGetSubscriptionCapability.mockImplementation(
      () => new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )
    const store = useSubscriptionCapabilityStore()

    const first = store.fetchSubscriptionCapability()
    const second = store.fetchSubscriptionCapability()
    resolveRequest({ has_subscription_groups: true })

    await expect(Promise.all([first, second])).resolves.toEqual([true, true])
    expect(mockGetSubscriptionCapability).toHaveBeenCalledTimes(1)
  })

  it('force refreshes an already loaded capability', async () => {
    mockGetSubscriptionCapability
      .mockResolvedValueOnce({ has_subscription_groups: true })
      .mockResolvedValueOnce({ has_subscription_groups: false })
    const store = useSubscriptionCapabilityStore()

    await store.fetchSubscriptionCapability()
    await expect(store.fetchSubscriptionCapability(true)).resolves.toBe(false)

    expect(store.hasSubscriptionGroups).toBe(false)
    expect(mockGetSubscriptionCapability).toHaveBeenCalledTimes(2)
  })

  it('fails closed and exposes the loaded state when the request fails', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    mockGetSubscriptionCapability.mockRejectedValue(new Error('Network error'))
    const store = useSubscriptionCapabilityStore()

    await expect(store.fetchSubscriptionCapability()).rejects.toThrow('Network error')

    expect(store.hasSubscriptionGroups).toBe(false)
    expect(store.capabilityLoaded).toBe(true)
    expect(consoleError).toHaveBeenCalledOnce()
    consoleError.mockRestore()
  })

  it('polls once per interval and does not create duplicate timers', async () => {
    mockGetSubscriptionCapability.mockResolvedValue({ has_subscription_groups: true })
    const store = useSubscriptionCapabilityStore()

    store.startPolling()
    store.startPolling()
    await vi.advanceTimersByTimeAsync(5 * 60 * 1000)

    expect(mockGetSubscriptionCapability).toHaveBeenCalledTimes(1)
    store.stopPolling()
  })

  it('clear resets state, stops polling, and ignores an older response', async () => {
    let resolveRequest!: (value: { has_subscription_groups: boolean }) => void
    mockGetSubscriptionCapability.mockImplementation(
      () => new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )
    const store = useSubscriptionCapabilityStore()
    const request = store.fetchSubscriptionCapability()

    store.startPolling()
    store.clear()
    resolveRequest({ has_subscription_groups: true })
    await request
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000)

    expect(store.hasSubscriptionGroups).toBe(false)
    expect(store.capabilityLoaded).toBe(false)
    expect(mockGetSubscriptionCapability).toHaveBeenCalledTimes(1)
  })
})
