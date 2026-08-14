import { apiClient } from '@/api/client'
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
  source_order_count?: number
  source_orders_json?: InvoiceSourceOrder[]
  completed_at?: string | null
  rejected_at?: string | null
  approved_at?: string | null
  processed_by?: number | null
  created_at: string
  updated_at: string
}

export interface InvoiceSourceOrder {
  id: number
  record_source: string
  business_category: string
  payment_type: string
  out_trade_no: string
  amount: number
  refund_amount: number
  invoiceable: boolean
}

export interface InvoiceTemplate {
  id: number
  user_id: number
  name: string
  invoice_type: InvoiceType
  title: string
  tax_id: string
  item_name: string
  receiver_email: string
  note: string
  is_default: boolean
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
  min_tax_fee: number
  tax_fee_threshold: number
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
  source_order_ids?: number[]
}

export interface SaveInvoiceTemplatePayload {
  name: string
  invoice_type: InvoiceType
  title: string
  tax_id?: string
  item_name: string
  receiver_email: string
  note?: string
  is_default?: boolean
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

  listTemplates() {
    return apiClient.get<InvoiceTemplate[]>('/invoices/templates')
  },

  createTemplate(data: SaveInvoiceTemplatePayload) {
    return apiClient.post<InvoiceTemplate>('/invoices/templates', data)
  },

  updateTemplate(id: number, data: SaveInvoiceTemplatePayload) {
    return apiClient.put<InvoiceTemplate>(`/invoices/templates/${id}`, data)
  },

  deleteTemplate(id: number) {
    return apiClient.delete<{ deleted: boolean }>(`/invoices/templates/${id}`)
  },

  setDefaultTemplate(id: number) {
    return apiClient.post<InvoiceTemplate>(`/invoices/templates/${id}/default`)
  },
}

export default invoicesAPI
