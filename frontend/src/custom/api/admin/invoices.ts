import { apiClient } from '@/api/client'
import type { BasePaginationResponse } from '@/types'
import type { InvoiceRequest, InvoiceStatus } from '@/custom/api/invoices'

export interface AdminInvoiceListParams {
  page?: number
  page_size?: number
  status?: InvoiceStatus | ''
  search?: string
  user_id?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export const adminInvoicesAPI = {
  list(params?: AdminInvoiceListParams) {
    return apiClient.get<BasePaginationResponse<InvoiceRequest>>('/admin/invoices', { params })
  },

  getById(id: number) {
    return apiClient.get<InvoiceRequest>(`/admin/invoices/${id}`)
  },

  approve(id: number, data?: { admin_note?: string }) {
    return apiClient.post<InvoiceRequest>(`/admin/invoices/${id}/approve`, data || {})
  },

  reject(id: number, data: { admin_note: string }) {
    return apiClient.post<InvoiceRequest>(`/admin/invoices/${id}/reject`, data)
  },

  complete(id: number, data: { invoice_no?: string; admin_note?: string }) {
    return apiClient.post<InvoiceRequest>(`/admin/invoices/${id}/complete`, data)
  },
}

export default adminInvoicesAPI
