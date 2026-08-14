/**
 * User Support Ticket API endpoints
 */

import { apiClient } from '@/api/client'
import type {
  AddTicketMessageRequest,
  BasePaginationResponse,
  CreateTicketRequest,
  FetchOptions,
  Ticket,
  TicketPrefillData,
  TicketTemplate,
  TicketUnreadSummary,
  TicketMessage
} from '@/types'

export interface TicketListFilters {
  status?: string
  priority?: string
  category?: string
  search?: string
  unread_only?: boolean
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: TicketListFilters,
  options?: FetchOptions
): Promise<BasePaginationResponse<Ticket>> {
  const { data } = await apiClient.get<BasePaginationResponse<Ticket>>('/tickets', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<Ticket> {
  const { data } = await apiClient.get<Ticket>(`/tickets/${id}`)
  return data
}

export async function create(request: CreateTicketRequest): Promise<Ticket> {
  const { data } = await apiClient.post<Ticket>('/tickets', request)
  return data
}

export async function templates(): Promise<TicketTemplate[]> {
  const { data } = await apiClient.get<TicketTemplate[]>('/tickets/templates')
  return data
}

export async function prefill(): Promise<TicketPrefillData> {
  const { data } = await apiClient.get<TicketPrefillData>('/tickets/prefill')
  return data
}

export async function addMessage(id: number, request: AddTicketMessageRequest): Promise<TicketMessage> {
  const { data } = await apiClient.post<TicketMessage>(`/tickets/${id}/messages`, request)
  return data
}

export async function close(id: number): Promise<Ticket> {
  const { data } = await apiClient.post<Ticket>(`/tickets/${id}/close`)
  return data
}

export async function reopen(id: number): Promise<Ticket> {
  const { data } = await apiClient.post<Ticket>(`/tickets/${id}/reopen`)
  return data
}

export async function getUnreadSummary(): Promise<TicketUnreadSummary> {
  const { data } = await apiClient.get<TicketUnreadSummary>('/tickets/unread-summary')
  return data
}

const ticketsAPI = {
  list,
  getById,
  create,
  templates,
  prefill,
  addMessage,
  close,
  reopen,
  getUnreadSummary
}

export default ticketsAPI
