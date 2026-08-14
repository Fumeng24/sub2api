import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get }
}))

import userGroupsAPI, { getSubscriptionCapability } from '@/custom/api/groups'

describe('custom groups API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads subscription capability from the dedicated endpoint', async () => {
    const capability = { has_subscription_groups: true }
    get.mockResolvedValue({ data: capability })

    await expect(getSubscriptionCapability()).resolves.toBe(capability)
    expect(get).toHaveBeenCalledWith('/groups/subscription-capability')
  })

  it('preserves the official user groups API', () => {
    expect(userGroupsAPI.getAvailable).toBeTypeOf('function')
    expect(userGroupsAPI.getUserGroupRates).toBeTypeOf('function')
  })
})
