import baseUserGroupsAPI from '@/api/groups'
import { apiClient } from '@/api/client'

export interface SubscriptionCapability {
  has_subscription_groups: boolean
}

export async function getSubscriptionCapability(): Promise<SubscriptionCapability> {
  const { data } = await apiClient.get<SubscriptionCapability>('/groups/subscription-capability')
  return data
}

const userGroupsAPI = {
  ...baseUserGroupsAPI,
  getSubscriptionCapability
}

export default userGroupsAPI
