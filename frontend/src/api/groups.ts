/**
 * User Groups API endpoints (non-admin)
 * Handles group-related operations for regular users
 */

import { apiClient } from './client'
import type { Group } from '@/types'

/**
 * Get available groups that the current user can bind to API keys
 * This returns groups based on user's permissions:
 * - Standard groups: public (non-exclusive) or explicitly allowed
 * - Subscription groups: user has active subscription
 * @returns List of available groups
 */
export async function getAvailable(): Promise<Group[]> {
  const { data } = await apiClient.get<Group[]>('/groups/available')
  return data
}

/**
 * Get current user's custom group rate multipliers
 * @returns Map of group_id to custom rate_multiplier
 */
export async function getUserGroupRates(): Promise<Record<number, number>> {
  const { data } = await apiClient.get<Record<number, number> | null>('/groups/rates')
  return data || {}
}

export interface GroupAvailabilityItem {
  group_id: number
  group_name: string
  platform: string
  available: boolean
  total_accounts: number
  active_accounts: number
}

/**
 * Get availability status for all groups the current user can access
 * @returns List of group availability info
 */
export async function getAvailability(): Promise<GroupAvailabilityItem[]> {
  const { data } = await apiClient.get<GroupAvailabilityItem[]>('/groups/availability')
  return data
}

export const userGroupsAPI = {
  getAvailable,
  getUserGroupRates,
  getAvailability
}

export default userGroupsAPI
