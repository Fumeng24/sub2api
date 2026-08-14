import { beforeEach, describe, expect, it, vi } from 'vitest'

const { patch, post } = vi.hoisted(() => ({
  patch: vi.fn(),
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { patch, post }
}))

import subscriptionsAPI, {
  resetSubscription,
  setAutoResetDaily
} from '@/custom/api/subscriptions'

describe('custom subscriptions API', () => {
  beforeEach(() => {
    patch.mockReset()
    post.mockReset()
  })

  it('resets the selected subscription with the backend route contract', async () => {
    const subscription = { id: 42 }
    post.mockResolvedValue({ data: subscription })

    await expect(resetSubscription(42)).resolves.toBe(subscription)
    expect(post).toHaveBeenCalledWith('/subscriptions/42/reset')
  })

  it('updates auto-reset with the expected PATCH payload', async () => {
    const subscription = { id: 42, auto_reset_daily: true }
    patch.mockResolvedValue({ data: subscription })

    await expect(setAutoResetDaily(42, true)).resolves.toBe(subscription)
    expect(patch).toHaveBeenCalledWith('/subscriptions/42/auto-reset', { enabled: true })
  })

  it('preserves the official read-only subscription API', () => {
    expect(subscriptionsAPI.getMySubscriptions).toBeTypeOf('function')
    expect(subscriptionsAPI.getActiveSubscriptions).toBeTypeOf('function')
  })
})
