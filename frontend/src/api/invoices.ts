import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'

export type InvoiceStatus = 'pending' | 'approved' | 'rejected' | 'completed' | 'cancelled'
export type InvoiceType = 'company_vat_general' | 'company_vat_special' | 'personal'

export interface InvoiceRequest {
  id: number
  user_id: number
  user_email: string
  user_name: string
  status: InvoiceStatus
  invoice_type: InvoiceType
  title: string
  tax_id: string
  item_name: string
  amount: number
  tax_rate: number
  tax_fee: number
  receiver_email: string
  note: string
  admin_note: string
  invoice_no: string
  completed_at?: string | null
  rejected_at?: string | null
  approved_at?: string | null
  processed_by?: number | null
  created_at: string
  updated_at: string
}

export interface InvoiceSummary {
  recharge_amount: number
  invoiced_amount: number
  locked_amount: number
  available_amount: number
  min_amount: number
  tax_rate: number
  tax_rate_percent: number
  can_apply: boolean
  current_balance: number
  invoiceable_basis: string
}

export interface CreateInvoiceRequestPayload {
  invoice_type: InvoiceType
  title: string
  tax_id?: string
  item_name: string
  amount: number
  receiver_email: string
  note?: string
}

export const invoicesAPI = {
  getSummary() {
    return apiClient.get<InvoiceSummary>('/invoices/summary')
  },

  list(params?: { page?: number; page_size?: number; sort_by?: string; sort_order?: 'asc' | 'desc' }) {
    return apiClient.get<BasePaginationResponse<InvoiceRequest>>('/invoices', { params })
  },

  getById(id: number) {
    return apiClient.get<InvoiceRequest>(`/invoices/${id}`)
  },

  create(data: CreateInvoiceRequestPayload) {
    return apiClient.post<InvoiceRequest>('/invoices', data)
  },

  cancel(id: number) {
    return apiClient.post<InvoiceRequest>(`/invoices/${id}/cancel`)
  },
}

export default invoicesAPI
