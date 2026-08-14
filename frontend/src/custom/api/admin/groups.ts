import baseGroupsAPI from '@/api/admin/groups'
import { apiClient } from '@/api/client'
import type {
  AccountSchedulingConfig,
  GroupRateChangeNotificationPreview,
  GroupRateChangeNotificationRequest,
  GroupRateChangeNotificationSendResult,
  UpdateAccountSchedulingRequest
} from '@/types'

export interface GroupSchedulerHistoryItem {
  id: number
  event_type: string
  account_id?: number | null
  group_id?: number | null
  payload?: Record<string, unknown> | null
  created_at: string
}

export interface GroupSchedulerHistoryResponse {
  items: GroupSchedulerHistoryItem[]
}

export interface GroupRelativeRateMultiplierEntry {
  user_id: number
  user_name: string
  user_email: string
  user_notes: string
  user_status: string
  relative_rate_multiplier?: number | null
  fixed_rate_multiplier?: number | null
}

export interface GroupRelativeRateMultiplierInput {
  user_id: number
  multiplier: number
}

export async function getAccountScheduling(id: number): Promise<AccountSchedulingConfig> {
  const { data } = await apiClient.get<AccountSchedulingConfig>(
    `/admin/groups/${id}/account-scheduling`
  )
  return data
}

export async function updateAccountScheduling(
  id: number,
  payload: UpdateAccountSchedulingRequest
): Promise<AccountSchedulingConfig> {
  const { data } = await apiClient.put<AccountSchedulingConfig>(
    `/admin/groups/${id}/account-scheduling`,
    payload
  )
  return data
}

export async function getAccountSchedulingHistory(
  id: number,
  limit: number = 30
): Promise<GroupSchedulerHistoryItem[]> {
  const { data } = await apiClient.get<GroupSchedulerHistoryResponse>(
    `/admin/groups/${id}/account-scheduling/history`,
    { params: { limit } }
  )
  return data.items || []
}

export async function previewRateChangeNotification(
  id: number,
  payload: GroupRateChangeNotificationRequest
): Promise<GroupRateChangeNotificationPreview> {
  const { data } = await apiClient.post<GroupRateChangeNotificationPreview>(
    `/admin/groups/${id}/rate-change-notification/preview`,
    payload
  )
  return data
}

export async function sendRateChangeNotification(
  id: number,
  payload: GroupRateChangeNotificationRequest
): Promise<GroupRateChangeNotificationSendResult> {
  const { data } = await apiClient.post<GroupRateChangeNotificationSendResult>(
    `/admin/groups/${id}/rate-change-notification/send`,
    payload
  )
  return data
}

export async function getGroupRelativeRateMultipliers(
  id: number
): Promise<GroupRelativeRateMultiplierEntry[]> {
  const { data } = await apiClient.get<GroupRelativeRateMultiplierEntry[]>(
    `/admin/groups/${id}/relative-rate-multipliers`
  )
  return data || []
}

export async function setGroupRelativeRateMultipliers(
  id: number,
  entries: GroupRelativeRateMultiplierInput[]
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(
    `/admin/groups/${id}/relative-rate-multipliers`,
    { entries }
  )
  return data
}

export async function clearGroupRelativeRateMultipliers(
  id: number
): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/groups/${id}/relative-rate-multipliers`
  )
  return data
}

const groupsAPI = {
  ...baseGroupsAPI,
  getAccountScheduling,
  updateAccountScheduling,
  getAccountSchedulingHistory,
  previewRateChangeNotification,
  sendRateChangeNotification,
  getGroupRelativeRateMultipliers,
  setGroupRelativeRateMultipliers,
  clearGroupRelativeRateMultipliers,
}

export default groupsAPI
