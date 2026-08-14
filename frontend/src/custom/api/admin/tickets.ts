/**
 * Admin Support Ticket API endpoints
 */

import { apiClient } from '@/api/client'
import type {
  AddTicketMessageRequest,
  BasePaginationResponse,
  BalanceBusinessCategory,
  FetchOptions,
  Ticket,
  TicketAdminCapabilities,
  TicketMessage,
  TicketStats,
  TicketUnreadSummary,
  UpdateTicketRequest
} from '@/types'

export interface AdminTicketListFilters {
  status?: string
  priority?: string
  category?: string
  search?: string
  assignee_id?: string | number
  template_key?: string
  queue?: string
  escalated_only?: boolean
  unread_only?: boolean
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: AdminTicketListFilters,
  options?: FetchOptions
): Promise<BasePaginationResponse<Ticket>> {
  const { data } = await apiClient.get<BasePaginationResponse<Ticket>>('/admin/tickets', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<Ticket> {
  const { data } = await apiClient.get<Ticket>(`/admin/tickets/${id}`)
  return data
}

export async function update(id: number, request: UpdateTicketRequest): Promise<Ticket> {
  const { data } = await apiClient.put<Ticket>(`/admin/tickets/${id}`, request)
  return data
}

export async function addMessage(id: number, request: AddTicketMessageRequest): Promise<TicketMessage> {
  const { data } = await apiClient.post<TicketMessage>(`/admin/tickets/${id}/messages`, request)
  return data
}

export async function claim(id: number): Promise<Ticket> {
  const { data } = await apiClient.post<Ticket>(`/admin/tickets/${id}/claim`)
  return data
}

export async function escalate(id: number, reason: string): Promise<Ticket> {
  const { data } = await apiClient.post<Ticket>(`/admin/tickets/${id}/escalate`, { reason })
  return data
}

export async function adjustBalance(id: number, request: {
  amount: number
  operation: 'set' | 'add' | 'subtract'
  notes?: string
  business_category?: BalanceBusinessCategory
}): Promise<unknown> {
  const { data } = await apiClient.post(`/admin/tickets/${id}/balance-adjust`, request)
  return data
}

export async function getUnreadSummary(): Promise<TicketUnreadSummary> {
  const { data } = await apiClient.get<TicketUnreadSummary>('/admin/tickets/unread-summary')
  return data
}

export async function getStats(): Promise<TicketStats> {
  const { data } = await apiClient.get<TicketStats>('/admin/tickets/stats')
  return data
}

export async function getCapabilities(): Promise<TicketAdminCapabilities> {
  const { data } = await apiClient.get<TicketAdminCapabilities>('/admin/tickets/capabilities')
  return data
}

export async function batchUpdate(request: {
  ids: number[]
  status?: string
  priority?: string
  category?: string
  assignee_id?: number
}): Promise<{ updated: number }> {
  const { data } = await apiClient.post<{ updated: number }>('/admin/tickets/batch-update', request)
  return data
}

export async function autoCloseResolved(days: number): Promise<{ closed: number }> {
  const { data } = await apiClient.post<{ closed: number }>('/admin/tickets/auto-close-resolved', { days })
  return data
}

const ticketsAPI = {
  list,
  getById,
  update,
  addMessage,
  claim,
  escalate,
  adjustBalance,
  getUnreadSummary,
  getStats,
  getCapabilities,
  batchUpdate,
  autoCloseResolved
}

export default ticketsAPI
