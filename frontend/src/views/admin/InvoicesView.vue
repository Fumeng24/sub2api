<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div class="flex-1">
            <h1 class="text-xl font-bold text-gray-950 dark:text-white">发票管理</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">审核用户开票申请，完成开票时会从用户余额扣除 2% 税点并锁定已开票金额。</p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <input v-model.trim="filters.search" class="input w-full sm:w-64" placeholder="搜索用户、抬头、税号、发票号" @input="debounceLoad" />
            <Select v-model="filters.status" :options="statusOptions" class="w-full sm:w-36" @change="load" />
            <button class="btn btn-secondary" :disabled="loading" @click="load">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              刷新
            </button>
          </div>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div v-if="loading" class="flex justify-center py-20">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
        </div>

        <div v-else-if="items.length === 0" class="py-20 text-center">
          <Icon name="document" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
          <p class="font-medium text-gray-700 dark:text-gray-200">暂无发票申请</p>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">申请</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">用户</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">开票信息</th>
                <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">金额</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">状态</th>
                <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="item in items" :key="item.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/70">
                <td class="whitespace-nowrap px-4 py-4">
                  <p class="font-mono text-sm font-semibold text-gray-900 dark:text-white">#{{ item.id }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(item.created_at) }}</p>
                </td>
                <td class="px-4 py-4">
                  <p class="max-w-[220px] truncate text-sm font-medium text-gray-900 dark:text-white">{{ item.user_email }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">UID {{ item.user_id }} · {{ item.user_name || '-' }}</p>
                </td>
                <td class="px-4 py-4">
                  <p class="max-w-[260px] truncate text-sm font-semibold text-gray-900 dark:text-white">{{ item.title }}</p>
                  <p class="mt-1 max-w-[260px] truncate text-xs text-gray-500 dark:text-gray-400">{{ invoiceTypeLabel(item.invoice_type) }} · {{ item.item_name }}</p>
                  <p v-if="item.tax_id" class="mt-1 max-w-[260px] truncate font-mono text-xs text-gray-400">{{ item.tax_id }}</p>
                </td>
                <td class="whitespace-nowrap px-4 py-4 text-right">
                  <p class="text-base font-black text-gray-950 dark:text-white">{{ formatMoney(item.amount) }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">税点 {{ formatMoney(item.tax_fee || item.amount * item.tax_rate) }}</p>
                </td>
                <td class="whitespace-nowrap px-4 py-4">
                  <span :class="['badge', statusBadgeClass(item.status)]">{{ statusLabel(item.status) }}</span>
                </td>
                <td class="px-4 py-4">
                  <div class="flex flex-wrap justify-end gap-2">
                    <button class="rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700" @click="openDetail(item)">详情</button>
                    <button v-if="item.status === 'pending'" class="rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-300 dark:hover:bg-primary-900/20" @click="openApprove(item)">确认</button>
                    <button v-if="item.status === 'pending' || item.status === 'approved'" class="rounded-md px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-900/20" @click="openReject(item)">驳回</button>
                    <button v-if="item.status === 'pending' || item.status === 'approved'" class="rounded-md px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 dark:text-emerald-300 dark:hover:bg-emerald-900/20" @click="openComplete(item)">完成</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="border-t border-gray-100 p-4 dark:border-dark-700">
          <Pagination
            v-if="pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </div>
    </div>

    <BaseDialog :show="showDetail" title="发票申请详情" width="wide" @close="showDetail = false">
      <div v-if="selected" class="space-y-4">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <InfoItem label="申请编号" :value="`#${selected.id}`" />
          <InfoItem label="状态" :value="statusLabel(selected.status)" />
          <InfoItem label="用户" :value="`${selected.user_email} / UID ${selected.user_id}`" />
          <InfoItem label="接收邮箱" :value="selected.receiver_email" />
          <InfoItem label="发票类型" :value="invoiceTypeLabel(selected.invoice_type)" />
          <InfoItem label="开票项目" :value="selected.item_name" />
          <InfoItem label="发票抬头" :value="selected.title" />
          <InfoItem label="税号" :value="selected.tax_id || '-'" />
          <InfoItem label="开票金额" :value="formatMoney(selected.amount)" />
          <InfoItem label="税点扣费" :value="formatMoney(selected.tax_fee || selected.amount * selected.tax_rate)" />
          <InfoItem label="发票号" :value="selected.invoice_no || '-'" />
          <InfoItem label="创建时间" :value="formatDateTime(selected.created_at)" />
        </div>
        <div v-if="selected.note" class="rounded-xl bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p class="mb-1 font-medium text-gray-900 dark:text-white">用户备注</p>
          <p class="whitespace-pre-wrap">{{ selected.note }}</p>
        </div>
        <div v-if="selected.admin_note" class="rounded-xl bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p class="mb-1 font-medium text-gray-900 dark:text-white">后台备注</p>
          <p class="whitespace-pre-wrap">{{ selected.admin_note }}</p>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :show="actionDialog.show" :title="actionDialogTitle" @close="closeAction">
      <form class="space-y-4" @submit.prevent="submitAction">
        <div v-if="selected" class="rounded-xl bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p>申请 #{{ selected.id }} · {{ selected.user_email }}</p>
          <p class="mt-1">开票金额 {{ formatMoney(selected.amount) }}，税点 {{ formatMoney(selected.amount * selected.tax_rate) }}</p>
        </div>
        <div v-if="actionDialog.type === 'complete'">
          <label class="input-label">发票号</label>
          <input v-model.trim="actionForm.invoice_no" class="input" maxlength="128" placeholder="可选，电子票号/流水号" />
        </div>
        <div>
          <label class="input-label">{{ actionDialog.type === 'reject' ? '驳回原因' : '后台备注' }}</label>
          <textarea v-model.trim="actionForm.admin_note" class="input" rows="4" maxlength="2000" :required="actionDialog.type === 'reject'" placeholder="会展示给用户，请写清楚处理结果"></textarea>
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closeAction">取消</button>
          <button class="btn btn-primary" :disabled="actionSubmitting">
            <span v-if="actionSubmitting">处理中...</span>
            <span v-else>{{ actionSubmitText }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import adminInvoicesAPI from '@/api/admin/invoices'
import type { InvoiceRequest, InvoiceStatus, InvoiceType } from '@/api/invoices'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { useI18n } from 'vue-i18n'

const InfoItem = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', { class: 'rounded-xl border border-gray-100 p-3 dark:border-dark-700' }, [
      h('p', { class: 'text-xs text-gray-500 dark:text-gray-400' }, props.label),
      h('p', { class: 'mt-1 break-words text-sm font-medium text-gray-900 dark:text-white' }, props.value),
    ])
  },
})

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const actionSubmitting = ref(false)
const items = ref<InvoiceRequest[]>([])
const selected = ref<InvoiceRequest | null>(null)
const showDetail = ref(false)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const filters = reactive({ status: '' as InvoiceStatus | '', search: '' })
const actionDialog = reactive({ show: false, type: '' as 'approve' | 'reject' | 'complete' | '' })
const actionForm = reactive({ invoice_no: '', admin_note: '' })
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待确认' },
  { value: 'approved', label: '已确认' },
  { value: 'rejected', label: '已驳回' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' },
]

const actionDialogTitle = computed(() => {
  if (actionDialog.type === 'approve') return '确认开票申请'
  if (actionDialog.type === 'reject') return '驳回开票申请'
  if (actionDialog.type === 'complete') return '完成开票'
  return '处理开票申请'
})

const actionSubmitText = computed(() => {
  if (actionDialog.type === 'approve') return '确认'
  if (actionDialog.type === 'reject') return '驳回'
  if (actionDialog.type === 'complete') return '完成并扣税点'
  return '提交'
})

async function load() {
  loading.value = true
  try {
    const res = await adminInvoicesAPI.list({
      page: pagination.page,
      page_size: pagination.page_size,
      status: filters.status || undefined,
      search: filters.search || undefined,
    })
    items.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '加载发票申请失败'))
  } finally {
    loading.value = false
  }
}

function debounceLoad() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    pagination.page = 1
    load()
  }, 300)
}

function openDetail(item: InvoiceRequest) {
  selected.value = item
  showDetail.value = true
}

function openApprove(item: InvoiceRequest) {
  selected.value = item
  actionDialog.type = 'approve'
  actionDialog.show = true
  actionForm.invoice_no = ''
  actionForm.admin_note = ''
}

function openReject(item: InvoiceRequest) {
  selected.value = item
  actionDialog.type = 'reject'
  actionDialog.show = true
  actionForm.invoice_no = ''
  actionForm.admin_note = ''
}

function openComplete(item: InvoiceRequest) {
  selected.value = item
  actionDialog.type = 'complete'
  actionDialog.show = true
  actionForm.invoice_no = item.invoice_no || ''
  actionForm.admin_note = item.admin_note || ''
}

function closeAction() {
  actionDialog.show = false
  actionDialog.type = ''
  selected.value = null
}

async function submitAction() {
  if (!selected.value || !actionDialog.type || actionSubmitting.value) return
  actionSubmitting.value = true
  try {
    if (actionDialog.type === 'approve') {
      await adminInvoicesAPI.approve(selected.value.id, { admin_note: actionForm.admin_note })
      appStore.showSuccess('已确认开票申请')
    } else if (actionDialog.type === 'reject') {
      await adminInvoicesAPI.reject(selected.value.id, { admin_note: actionForm.admin_note })
      appStore.showSuccess('已驳回开票申请')
    } else if (actionDialog.type === 'complete') {
      await adminInvoicesAPI.complete(selected.value.id, { invoice_no: actionForm.invoice_no, admin_note: actionForm.admin_note })
      appStore.showSuccess('已完成开票并扣除税点')
    }
    closeAction()
    await load()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '处理失败'))
  } finally {
    actionSubmitting.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  load()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  load()
}

function formatMoney(value?: number | null) {
  return `¥${(Number(value) || 0).toFixed(2)}`
}

function formatDateTime(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function invoiceTypeLabel(type: InvoiceType | string) {
  const map: Record<string, string> = {
    company_vat_general: '企业普票',
    company_vat_special: '企业专票',
    personal: '个人发票',
  }
  return map[type] || type
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    pending: '待确认',
    approved: '已确认',
    rejected: '已驳回',
    completed: '已完成',
    cancelled: '已取消',
  }
  return map[status] || status
}

function statusBadgeClass(status: string) {
  const map: Record<string, string> = {
    pending: 'badge-warning',
    approved: 'badge-primary',
    rejected: 'badge-danger',
    completed: 'badge-success',
    cancelled: 'badge-gray',
  }
  return map[status] || 'badge-gray'
}

onMounted(load)
</script>
