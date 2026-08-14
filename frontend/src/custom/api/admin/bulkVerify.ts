import { apiClient } from '@/api/client'

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
  refreshable_expired: number
  markable_exhausted: number
}

export const BULK_VERIFY_JOB_ALREADY_RUNNING = 'job_already_running'

export interface BulkVerifyApplyRequest {
  refresh_expired?: boolean
  mark_exhausted?: boolean
  dry_run?: boolean
}

export interface BulkVerifyApplyFailure {
  account_id: number
  action: string
  error: string
}

export interface BulkVerifyApplyResult {
  refresh_succeeded: number
  refresh_failed: number
  marked_rate_limited: number
  mark_rate_limit_failed: number
  skipped_no_reset_at: number
  dry_run: boolean
  failures?: BulkVerifyApplyFailure[]
}

export async function applyBulkVerifyJob(
  jobID: string,
  req: BulkVerifyApplyRequest
): Promise<BulkVerifyApplyResult> {
  const { data } = await apiClient.post<BulkVerifyApplyResult>(
    `/admin/accounts/bulk-verify/${jobID}/apply`,
    req
  )
  return data
}

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
  cancel: cancelBulkVerifyJob,
  apply: applyBulkVerifyJob
}

export default bulkVerifyAPI
