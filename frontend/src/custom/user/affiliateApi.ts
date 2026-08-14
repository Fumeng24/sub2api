import { apiClient } from '@/api/client'
import type { UserAffiliateDetail } from '@/types'

export async function bindAffiliateInviter(code: string): Promise<UserAffiliateDetail> {
  const { data } = await apiClient.post<UserAffiliateDetail>('/user/aff/bind', { code })
  return data
}

export async function claimAffiliateBindBonus(): Promise<{
  balance: number
  detail: UserAffiliateDetail
}> {
  const { data } = await apiClient.post<{
    balance: number
    detail: UserAffiliateDetail
  }>('/user/aff/bind-bonus/claim')
  return data
}
