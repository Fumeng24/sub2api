import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post, put }
}))

import {
  getAccountScheduling,
  getAccountSchedulingHistory,
  previewRateChangeNotification,
  sendRateChangeNotification,
  updateAccountScheduling
} from '@/custom/api/admin/groups'

describe('custom admin groups API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
  })

  it('loads and updates account scheduling through group-scoped routes', async () => {
    const config = { accounts: [] }
    const payload = { accounts: [] }
    get.mockResolvedValueOnce({ data: config })
    put.mockResolvedValueOnce({ data: config })

    await expect(getAccountScheduling(12)).resolves.toBe(config)
    expect(get).toHaveBeenCalledWith('/admin/groups/12/account-scheduling')
    await expect(updateAccountScheduling(12, payload)).resolves.toBe(config)
    expect(put).toHaveBeenCalledWith('/admin/groups/12/account-scheduling', payload)
  })

  it('loads scheduler history with the requested limit', async () => {
    const items = [{ id: 1, event_type: 'updated', created_at: '2026-07-11T00:00:00Z' }]
    get.mockResolvedValue({ data: { items } })

    await expect(getAccountSchedulingHistory(12, 50)).resolves.toBe(items)
    expect(get).toHaveBeenCalledWith('/admin/groups/12/account-scheduling/history', {
      params: { limit: 50 }
    })
  })

  it('previews and sends rate-change notifications', async () => {
    const payload = { new_rate_multiplier: 0.8 }
    const preview = { group_id: 12, user_count: 3 }
    const result = { group_id: 12, sent: 3 }
    post.mockResolvedValueOnce({ data: preview }).mockResolvedValueOnce({ data: result })

    await expect(previewRateChangeNotification(12, payload)).resolves.toBe(preview)
    expect(post).toHaveBeenNthCalledWith(
      1,
      '/admin/groups/12/rate-change-notification/preview',
      payload
    )
    await expect(sendRateChangeNotification(12, payload)).resolves.toBe(result)
    expect(post).toHaveBeenNthCalledWith(
      2,
      '/admin/groups/12/rate-change-notification/send',
      payload
    )
  })
})
