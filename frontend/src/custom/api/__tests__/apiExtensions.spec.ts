import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
  },
}))

import { keysAPI } from '@/custom/api/keys'
import { paymentAPI } from '@/custom/api/payment'
import { usageAPI } from '@/custom/api/usage'
import { channelMonitorAPI } from '@/custom/api/admin/channelMonitor'

describe('custom API extensions', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
    put.mockReset().mockResolvedValue({ data: {} })
  })

  it('creates keys from the object payload used by the custom key editor', async () => {
    const payload = { name: 'image-key', group_id: 8, category: 'other' as const }

    await keysAPI.create(payload)

    expect(post).toHaveBeenCalledWith('/keys', payload)
  })

  it('requests custom usage group and endpoint breakdowns', async () => {
    await usageAPI.getDashboardGroups({ start_date: '2026-07-01' })
    await usageAPI.getDashboardEndpoints({ end_date: '2026-07-12' })

    expect(get).toHaveBeenNthCalledWith(1, '/usage/dashboard/groups', {
      params: { start_date: '2026-07-01' },
    })
    expect(get).toHaveBeenNthCalledWith(2, '/usage/dashboard/endpoints', {
      params: { end_date: '2026-07-12' },
    })
  })

  it('passes invoice and order filters through the custom payment API', async () => {
    const params = { page: 2, order_type: 'balance', invoiceable: true }

    await paymentAPI.getMyOrders(params)

    expect(get).toHaveBeenCalledWith('/payment/orders/my', { params })
  })

  it('updates channel monitor sort order through the custom endpoint', async () => {
    const updates = [{ id: 3, sort_order: 10 }]

    await channelMonitorAPI.updateSortOrder(updates)

    expect(put).toHaveBeenCalledWith('/admin/channel-monitors/sort-order', { updates })
  })
})
