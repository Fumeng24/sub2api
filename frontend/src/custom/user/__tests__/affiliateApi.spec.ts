import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { post },
}))

import {
  bindAffiliateInviter,
  claimAffiliateBindBonus,
} from '@/custom/user/affiliateApi'

describe('affiliate site API', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('binds an inviter with the backend contract payload', async () => {
    const detail = { aff_code: 'MY-CODE' }
    post.mockResolvedValue({ data: detail })

    await expect(bindAffiliateInviter('INVITER-1')).resolves.toBe(detail)
    expect(post).toHaveBeenCalledWith('/user/aff/bind', { code: 'INVITER-1' })
  })

  it('claims the binding bonus without an invented request body', async () => {
    const result = { balance: 12, detail: { aff_code: 'MY-CODE' } }
    post.mockResolvedValue({ data: result })

    await expect(claimAffiliateBindBonus()).resolves.toBe(result)
    expect(post).toHaveBeenCalledWith('/user/aff/bind-bonus/claim')
  })
})
