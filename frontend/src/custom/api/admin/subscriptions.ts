import { apiClient } from '@/api/client'
import type { UserSubscription } from '@/types'

export async function resetSubscriptionWithCost(id: number): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/subscriptions/${id}/reset-with-cost`
  )
  return data
}
