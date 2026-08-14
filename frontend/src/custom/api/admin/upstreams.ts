import { apiClient } from '@/api/client'
import type { PaginatedResponse } from '@/types'
import type { UpstreamSub2APIAccountStatus } from './accounts'

export type UpstreamKind = 'auto' | 'newapi' | 'sub2api'
export type UpstreamStatus = 'unknown' | 'healthy' | 'degraded' | 'error'
export type ManagedUpstreamPlatform = 'anthropic' | 'openai' | 'gemini' | 'grok'

export interface UpstreamCredentialStatus {
  has_api_key: boolean
  has_openai_api_key: boolean
  has_anthropic_api_key: boolean
  has_gemini_api_key: boolean
  has_grok_api_key: boolean
  has_management_access_token: boolean
  has_management_user_id: boolean
  has_username: boolean
  has_password: boolean
  generated_group_key_count: number
}

export interface UpstreamWalletMetadata {
  balance?: number
  unit?: string
}

export interface UpstreamKeyMetadata {
  id?: number
  name?: string
  group_id?: number
  group_name?: string
  unlimited_quota: boolean
  remaining?: number
  unit?: string
}

export interface UpstreamGroupMetadata {
  id?: number
  name: string
  platform?: string
  description?: string
  rate_multiplier?: number
  models?: string[]
}

export interface UpstreamProtocolMetadata {
  platform: ManagedUpstreamPlatform
  status: string
  models: string[]
  message?: string
  fetched_at: string
}

export interface UpstreamModelProbeResult {
  success: boolean
  platform: ManagedUpstreamPlatform
  group_name: string
  model: string
  latency_ms: number
  status_code?: number
  status: string
  message?: string
  verified_at?: string
  expires_at?: string
}

export interface UpstreamModelsProbeResponse {
  success: boolean
  platform: ManagedUpstreamPlatform
  group_name: string
  status: 'ok' | 'partial' | 'error' | string
  message?: string
  source?: string
  latency_ms: number
  available_models: string[]
  results: UpstreamModelProbeResult[]
}

export interface UpstreamProbeMetadata {
  detected_kind?: UpstreamKind
  probe_source?: string
  management_status?: string
  management_hint?: string
  wallet?: UpstreamWalletMetadata
  key?: UpstreamKeyMetadata
  groups?: UpstreamGroupMetadata[]
  protocols?: UpstreamProtocolMetadata[]
  account_billing?: Record<string, UpstreamAccountBillingMetadata>
  refresh?: UpstreamRefreshMetadata
  fetched_at?: string
}

export interface UpstreamRefreshMetadata {
  status: 'ok' | 'partial' | 'failed' | string
  stale: boolean
  last_attempt_at: string
  last_success_at?: string
  next_refresh_at: string
  failure_count?: number
  account_success_count: number
  account_failure_count: number
}

export interface UpstreamAccountBillingMetadata {
  account_id: number
  status: string
  message?: string
  probe_source?: string
  fetched_at: string
  last_success_at?: string
  stale: boolean
  failure_count?: number
  key_remaining?: number
  balance_unit?: string
  usage_mode?: string
  usage_plan_name?: string
  upstream_key_id?: number
  upstream_key_name?: string
  upstream_group_id?: number
  upstream_group_name?: string
  upstream_group_platform?: string
  group_default_rate_multiplier?: number
  group_effective_rate_multiplier?: number
}

export interface UpstreamAccountSummary {
  id: number
  name: string
  platform: string
  type: string
  status: string
  schedulable: boolean
  upstream_id?: number
  group_ids: number[]
  generated: boolean
  upstream_group_id?: number
  upstream_group?: string
  upstream_group_rate_multiplier?: number
  upstream_group_rate_source?: 'effective' | 'default' | 'catalogue' | string
  upstream_group_stale: boolean
  upstream_group_change_supported: boolean
  upstream_group_change_reason?: string
}

export interface UpstreamLocalGroupSummary {
  id: number
  name: string
  platform: string
}

const MANAGED_ACCOUNT_STATUS_BATCH_SIZE = 500
const UPSTREAM_MODEL_PROBE_CONCURRENCY = 3
const UPSTREAM_MODEL_PROBE_TIMEOUT_PER_WAVE_MS = 90_000
const UPSTREAM_MODEL_PROBE_OVERHEAD_MS = 30_000

function upstreamModelProbeTimeout(modelCount: number): number {
  const waves = Math.max(1, Math.ceil(Math.max(0, modelCount) / UPSTREAM_MODEL_PROBE_CONCURRENCY))
  return UPSTREAM_MODEL_PROBE_OVERHEAD_MS + waves * UPSTREAM_MODEL_PROBE_TIMEOUT_PER_WAVE_MS
}

function managedAccountStatusBatches(accountIds: number[]): number[][] {
  const ids = Array.from(new Set(accountIds.filter(id => Number.isSafeInteger(id) && id > 0)))
  const batches: number[][] = []
  for (let offset = 0; offset < ids.length; offset += MANAGED_ACCOUNT_STATUS_BATCH_SIZE) {
    batches.push(ids.slice(offset, offset + MANAGED_ACCOUNT_STATUS_BATCH_SIZE))
  }
  return batches
}

export interface Upstream {
  id: number
  name: string
  base_url: string
  kind: UpstreamKind
  proxy_id?: number | null
  proxy_name?: string
  status: UpstreamStatus
  last_probe_at?: string
  last_probe_error?: string
  metadata: UpstreamProbeMetadata
  credential_status: UpstreamCredentialStatus
  account_count: number
  local_groups: UpstreamLocalGroupSummary[]
  accounts?: UpstreamAccountSummary[]
  duplicate_base_url_count: number
  created_at: string
  updated_at: string
}

export interface UpstreamMutationRequest {
  name: string
  base_url: string
  kind: UpstreamKind
  proxy_id?: number | null
  clear_proxy?: boolean
  credentials?: Record<string, string>
  clear_credentials?: string[]
}

export interface UpstreamAccountGenerationSpec {
  name?: string
  platform: ManagedUpstreamPlatform
  upstream_group_name: string
  upstream_group_id?: number
  models: string[]
  local_group_ids: number[]
  concurrency?: number
  priority?: number
  rate_multiplier?: number
  api_key?: string
}

export interface UpstreamAccountGenerationPreviewItem {
  index: number
  name: string
  platform: ManagedUpstreamPlatform
  upstream_group_name: string
  upstream_group_id?: number
  models: string[]
  local_group_ids: number[]
  concurrency: number
  priority: number
  rate_multiplier?: number
  action: 'create' | 'skip'
  existing_account_id?: number
  key_source?: string
  will_create_upstream_key: boolean
  warnings: string[]
  errors: string[]
}

export interface UpstreamAccountGenerationPreview {
  valid: boolean
  creates: number
  skips: number
  items: UpstreamAccountGenerationPreviewItem[]
}

export interface UpstreamAccountGenerationResult {
  index: number
  success: boolean
  skipped: boolean
  account_id?: number
  existing_account_id?: number
  error?: string
}

export interface UpstreamAccountGroupChangeResponse {
  account: UpstreamAccountSummary
  models: string[]
  warning?: string
}

export interface UpstreamAccountRenameItem {
  account_id: number
  upstream_id: number
  current_name: string
  proposed_name?: string
  action: 'rename' | 'skip' | 'renamed' | 'failed'
  reason?: string
}

export interface UpstreamAccountRenamePreview {
  renames: number
  skips: number
  items: UpstreamAccountRenameItem[]
}

export interface UpstreamAccountRenameApplyResult {
  renamed: number
  skipped: number
  failed: number
  items: UpstreamAccountRenameItem[]
}

export async function listUpstreams(page = 1, pageSize = 20, search = ''): Promise<PaginatedResponse<Upstream>> {
  const { data } = await apiClient.get<PaginatedResponse<Upstream>>('/admin/upstreams', {
    params: { page, page_size: pageSize, search: search || undefined }
  })
  return data
}

export async function getUpstream(id: number): Promise<Upstream> {
  const { data } = await apiClient.get<Upstream>(`/admin/upstreams/${id}`)
  return data
}

export async function createUpstream(payload: UpstreamMutationRequest): Promise<Upstream> {
  const { data } = await apiClient.post<Upstream>('/admin/upstreams', payload)
  return data
}

export async function updateUpstream(id: number, payload: UpstreamMutationRequest): Promise<Upstream> {
  const { data } = await apiClient.put<Upstream>(`/admin/upstreams/${id}`, payload)
  return data
}

export async function deleteUpstream(id: number, force = false): Promise<{ deleted: boolean; unbound_account_count: number }> {
  const { data } = await apiClient.delete<{ deleted: boolean; unbound_account_count: number }>(`/admin/upstreams/${id}`, {
    params: force ? { force: true } : undefined
  })
  return data
}

export async function probeUpstream(id: number): Promise<Upstream> {
  const { data } = await apiClient.post<Upstream>(`/admin/upstreams/${id}/probe`)
  return data
}

export async function getManagedAccountStatuses(accountIds: number[]): Promise<UpstreamSub2APIAccountStatus[]> {
  const result: UpstreamSub2APIAccountStatus[] = []
  for (const batch of managedAccountStatusBatches(accountIds)) {
    const { data } = await apiClient.get<UpstreamSub2APIAccountStatus[]>('/admin/upstreams/account-status', {
      params: { account_ids: batch.join(',') }
    })
    result.push(...data)
  }
  return result
}

export async function refreshManagedAccountStatuses(accountIds: number[]): Promise<UpstreamSub2APIAccountStatus[]> {
  const result: UpstreamSub2APIAccountStatus[] = []
  for (const batch of managedAccountStatusBatches(accountIds)) {
    const { data } = await apiClient.post<UpstreamSub2APIAccountStatus[]>('/admin/upstreams/account-status/refresh', {
      account_ids: batch
    })
    result.push(...data)
  }
  return result
}

export async function testUpstreamModel(
  id: number,
  payload: { platform: ManagedUpstreamPlatform; group_name: string; model: string }
): Promise<UpstreamModelProbeResult> {
  const { data } = await apiClient.post<UpstreamModelProbeResult>(`/admin/upstreams/${id}/model-test`, payload, {
    timeout: upstreamModelProbeTimeout(1)
  })
  return data
}

export async function probeUpstreamModels(
  id: number,
  payload: {
    platform: ManagedUpstreamPlatform
    group_name: string
    models: string[]
    api_key?: string
  }
): Promise<UpstreamModelsProbeResponse> {
  const { data } = await apiClient.post<UpstreamModelsProbeResponse>(`/admin/upstreams/${id}/models/probe`, payload, {
    timeout: upstreamModelProbeTimeout(payload.models.length)
  })
  return data
}

export async function listBindCandidates(id: number, page = 1, pageSize = 100, search = ''): Promise<PaginatedResponse<UpstreamAccountSummary>> {
  const { data } = await apiClient.get<PaginatedResponse<UpstreamAccountSummary>>(`/admin/upstreams/${id}/bind-candidates`, {
    params: { page, page_size: pageSize, search: search || undefined }
  })
  return data
}

export async function bindAccounts(id: number, accountIds: number[], allowRebind = false): Promise<void> {
  await apiClient.post(`/admin/upstreams/${id}/bind`, {
    account_ids: accountIds,
    allow_rebind: allowRebind
  })
}

export async function unbindAccounts(id: number, accountIds: number[], deleteAccounts = true): Promise<void> {
  await apiClient.post(`/admin/upstreams/${id}/unbind`, {
    account_ids: accountIds,
    delete_accounts: deleteAccounts,
  })
}

export async function previewGeneratedAccounts(id: number, accounts: UpstreamAccountGenerationSpec[]): Promise<UpstreamAccountGenerationPreview> {
  const { data } = await apiClient.post<UpstreamAccountGenerationPreview>(`/admin/upstreams/${id}/accounts/preview`, { accounts })
  return data
}

export async function generateAccounts(id: number, accounts: UpstreamAccountGenerationSpec[]): Promise<UpstreamAccountGenerationResult[]> {
  const { data } = await apiClient.post<{ results: UpstreamAccountGenerationResult[] }>(`/admin/upstreams/${id}/accounts/generate`, { accounts })
  return data.results
}

export async function changeAccountUpstreamGroup(
  id: number,
  accountId: number,
  payload: { group_name: string; group_id?: number }
): Promise<UpstreamAccountGroupChangeResponse> {
  const { data } = await apiClient.put<UpstreamAccountGroupChangeResponse>(
    `/admin/upstreams/${id}/accounts/${accountId}/upstream-group`,
    payload
  )
  return data
}

export async function previewAccountRenames(): Promise<UpstreamAccountRenamePreview> {
  const { data } = await apiClient.post<UpstreamAccountRenamePreview>('/admin/upstreams/accounts/rename-preview')
  return data
}

export async function applyAccountRenames(): Promise<UpstreamAccountRenameApplyResult> {
  const { data } = await apiClient.post<UpstreamAccountRenameApplyResult>('/admin/upstreams/accounts/rename-apply')
  return data
}

export default {
  list: listUpstreams,
  get: getUpstream,
  create: createUpstream,
  update: updateUpstream,
  delete: deleteUpstream,
  probe: probeUpstream,
  getManagedAccountStatuses,
  refreshManagedAccountStatuses,
  testModel: testUpstreamModel,
  probeModels: probeUpstreamModels,
  listBindCandidates,
  bindAccounts,
  unbindAccounts,
  previewGeneratedAccounts,
  generateAccounts,
  changeAccountUpstreamGroup,
  previewAccountRenames,
  applyAccountRenames
}
