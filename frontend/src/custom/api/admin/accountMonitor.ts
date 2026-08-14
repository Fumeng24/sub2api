// 账号监控 API（与渠道监控完全独立的一套，纯管理员）。
// 后端路由：/admin/account-monitors（无任何用户路由）。
import { apiClient } from '@/api/client'

export type AccountMonitorProvider = 'openai' | 'anthropic' | 'gemini'
export type AccountMonitorStatusValue = 'operational' | 'degraded' | 'failed' | 'error' | ''

export interface AccountMonitor {
  id: number
  account_id: number
  provider: AccountMonitorProvider
  model: string
  enabled: boolean
  interval_seconds: number
  jitter_seconds: number
  last_checked_at: string | null
  created_at: string
  updated_at: string
}

export interface AccountMonitorTimelinePoint {
  status: AccountMonitorStatusValue
  latency_ms: number | null
  checked_at: string
}

export interface AccountMonitorStatus {
  monitor_id: number
  account_id: number
  model: string
  enabled: boolean
  latest_status: AccountMonitorStatusValue
  latest_latency_ms: number | null
  ping_latency_ms: number | null
  availability_1h: number
  avg_latency_1h: number | null
  last_checked_at: string | null
  // 最近 N 条探测（newest-first），用于渲染彩虹状态条。
  timeline: AccountMonitorTimelinePoint[]
}

export interface CreateParams {
  account_id: number
  provider?: AccountMonitorProvider
  model?: string
  enabled?: boolean
  interval_seconds?: number
  jitter_seconds?: number
}

export interface UpdateParams {
  provider?: AccountMonitorProvider
  model?: string
  enabled?: boolean
  interval_seconds?: number
  jitter_seconds?: number
}

export interface RunResponse {
  status: AccountMonitorStatusValue
  latency_ms: number | null
  message: string
  checked_at: string
}

interface ListResponse {
  items: AccountMonitor[]
}

interface StatusResponse {
  // 后端按 account_id（字符串键）索引返回。
  statuses: Record<string, AccountMonitorStatus>
}

export async function list(): Promise<AccountMonitor[]> {
  const { data } = await apiClient.get<ListResponse>('/admin/account-monitors')
  return data.items || []
}

// status 返回 account_id -> 聚合状态 的 Map，供 SchedulerView 按行匹配。
export async function status(): Promise<Map<number, AccountMonitorStatus>> {
  const { data } = await apiClient.get<StatusResponse>('/admin/account-monitors/status')
  const map = new Map<number, AccountMonitorStatus>()
  for (const [key, st] of Object.entries(data.statuses || {})) {
    map.set(Number(key), st)
  }
  return map
}

export async function create(params: CreateParams): Promise<AccountMonitor> {
  const { data } = await apiClient.post<AccountMonitor>('/admin/account-monitors', params)
  return data
}

export async function update(id: number, params: UpdateParams): Promise<AccountMonitor> {
  const { data } = await apiClient.put<AccountMonitor>(`/admin/account-monitors/${id}`, params)
  return data
}

export async function del(id: number): Promise<void> {
  await apiClient.delete(`/admin/account-monitors/${id}`)
}

export async function runNow(id: number): Promise<RunResponse> {
  const { data } = await apiClient.post<RunResponse>(`/admin/account-monitors/${id}/run`)
  return data
}

export const accountMonitorAPI = {
  list,
  status,
  create,
  update,
  del,
  runNow,
}

export default accountMonitorAPI
