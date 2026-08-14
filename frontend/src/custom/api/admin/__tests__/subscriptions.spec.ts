import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

import { resetSubscriptionWithCost } from '@/custom/api/admin/subscriptions'

describe('custom admin subscriptions API', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('resets the selected subscription through the admin route', async () => {
    const subscription = { id: 73 }
    post.mockResolvedValue({ data: subscription })

    await expect(resetSubscriptionWithCost(73)).resolves.toBe(subscription)
    expect(post).toHaveBeenCalledWith('/admin/subscriptions/73/reset-with-cost')
  })
})
