import { apiClient } from '../client'

export interface BulkVerifyStartRequest {
  concurrency?: number
  group_ids?: number[]
}

export interface BulkVerifyJob {
  id: string
  total: number
  done: number
  high: number
  medium: number
  low: number
  exhausted: number
  expired: number
  missing: number
  unknown: number
  error: number
  running: boolean
  cancelled: boolean
  started_at: string
  finished_at?: string
}

export const BULK_VERIFY_JOB_ALREADY_RUNNING = 'job_already_running'

export async function startBulkVerify(req: BulkVerifyStartRequest = {}): Promise<BulkVerifyJob> {
  const { data } = await apiClient.post<BulkVerifyJob>('/admin/accounts/bulk-verify', req)
  return data
}

export async function getBulkVerifyJob(jobID: string): Promise<BulkVerifyJob> {
  const { data } = await apiClient.get<BulkVerifyJob>(`/admin/accounts/bulk-verify/${jobID}`)
  return data
}

export async function cancelBulkVerifyJob(jobID: string): Promise<{ cancelled: boolean }> {
  const { data } = await apiClient.delete<{ cancelled: boolean }>(`/admin/accounts/bulk-verify/${jobID}`)
  return data
}

export const bulkVerifyAPI = {
  start: startBulkVerify,
  get: getBulkVerifyJob,
  cancel: cancelBulkVerifyJob
}

export default bulkVerifyAPI
