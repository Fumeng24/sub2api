import { apiClient } from '@/api/client'
import type { UserAvailableChannel } from '@/api/channels'
import type { UserMonitorDetail, UserMonitorListResponse } from '@/api/channelMonitor'

export interface PublicUserMonitorListResponse extends UserMonitorListResponse {
  last_updated_at?: string | null
  trend_period?: string
}

export async function getPublicAvailableChannels(options?: { signal?: AbortSignal }): Promise<UserAvailableChannel[]> {
  const { data } = await apiClient.get<UserAvailableChannel[]>('/public/model-pricing', {
    signal: options?.signal,
  })
  return data
}

export async function getPublicChannelMonitors(options?: { signal?: AbortSignal }): Promise<PublicUserMonitorListResponse> {
  const { data } = await apiClient.get<PublicUserMonitorListResponse>('/public/channel-monitors', {
    signal: options?.signal,
  })
  return data
}

export async function getPublicChannelMonitorStatus(id: number): Promise<UserMonitorDetail> {
  const { data } = await apiClient.get<UserMonitorDetail>(`/public/channel-monitors/${id}/status`)
  return data
}

export const publicGatewayAPI = {
  getPublicAvailableChannels,
  getPublicChannelMonitors,
  getPublicChannelMonitorStatus,
}

export default publicGatewayAPI
