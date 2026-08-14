import baseSubscriptionsAPI from '@/api/subscriptions'
import { apiClient } from '@/api/client'
import type { UserSubscription } from '@/types'

export async function resetSubscription(subscriptionId: number): Promise<UserSubscription> {
  const response = await apiClient.post<UserSubscription>(`/subscriptions/${subscriptionId}/reset`)
  return response.data
}

export async function setAutoResetDaily(
  subscriptionId: number,
  enabled: boolean
): Promise<UserSubscription> {
  const response = await apiClient.patch<UserSubscription>(
    `/subscriptions/${subscriptionId}/auto-reset`,
    { enabled }
  )
  return response.data
}

const subscriptionsAPI = {
  ...baseSubscriptionsAPI,
  resetSubscription,
  setAutoResetDaily
}

export default subscriptionsAPI
