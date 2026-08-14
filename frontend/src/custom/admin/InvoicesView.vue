<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex min-w-0 flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div class="min-w-0 flex-1">
            <h1 class="text-xl font-bold text-gray-950 dark:text-white">{{ localText('发票管理', 'Invoice Management') }}</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ localText('审核用户开票申请。服务费在用户提交时冻结，完成开票时确认扣除，取消或驳回时释放。', 'Review user invoice requests. Service fees are reserved on submission, charged on completion, and released after rejection or cancellation.') }}</p>
          </div>
          <div class="flex w-full min-w-0 flex-wrap items-center justify-end gap-2 lg:w-auto">
            <input v-model.trim="filters.search" class="input w-full min-w-0 sm:w-64" :placeholder="localText('搜索用户、抬头、税号、发票号', 'Search user, title, tax ID, or invoice number')" @input="debounceLoad" />
            <Select v-model="filters.status" :options="statusOptions" class="w-full min-w-0 sm:w-36" @change="load" />
            <button class="btn btn-secondary min-w-0 flex-1 sm:flex-none" :disabled="loading" @click="load">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
          </div>
        </div>
      </div>

      <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div v-if="loading" class="flex justify-center py-20">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
        </div>

        <div v-else-if="items.length === 0" class="py-20 text-center">
          <Icon name="document" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
          <p class="font-medium text-gray-700 dark:text-gray-200">{{ localText('暂无发票申请', 'No invoice requests') }}</p>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ localText('申请', 'Request') }}</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ localText('用户', 'User') }}</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ localText('开票信息', 'Invoice details') }}</th>
                <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ localText('金额', 'Amount') }}</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ localText('状态', 'Status') }}</th>
                <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ localText('操作', 'Actions') }}</th>
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
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ localText('服务费', 'Service fee') }} {{ formatMoney(invoiceTaxFee(item)) }}</p>
                </td>
                <td class="whitespace-nowrap px-4 py-4">
                  <span :class="['badge', statusBadgeClass(item.status)]">{{ statusLabel(item.status) }}</span>
                </td>
                <td class="px-4 py-4">
                  <div class="flex flex-wrap justify-end gap-2">
                    <button class="rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700" @click="openDetail(item)">{{ localText('详情', 'Details') }}</button>
                    <button v-if="item.status === 'pending'" class="rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-300 dark:hover:bg-primary-900/20" @click="openApprove(item)">{{ localText('确认', 'Approve') }}</button>
                    <button v-if="item.status === 'pending' || item.status === 'approved'" class="rounded-md px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-900/20" @click="openReject(item)">{{ localText('驳回', 'Reject') }}</button>
                    <button v-if="item.status === 'pending' || item.status === 'approved'" class="rounded-md px-2 py-1 text-xs font-medium text-emerald-600 hover:bg-emerald-50 dark:text-emerald-300 dark:hover:bg-emerald-900/20" @click="openComplete(item)">{{ localText('完成', 'Complete') }}</button>
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

    <BaseDialog :show="showDetail" :title="localText('发票申请详情', 'Invoice Request Details')" width="wide" @close="showDetail = false">
      <div v-if="selected" class="space-y-4">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <InfoItem :label="localText('申请编号', 'Request ID')" :value="`#${selected.id}`" />
          <InfoItem :label="localText('状态', 'Status')" :value="statusLabel(selected.status)" />
          <InfoItem :label="localText('用户', 'User')" :value="`${selected.user_email} / UID ${selected.user_id}`" />
          <InfoItem :label="localText('接收邮箱', 'Receiver email')" :value="selected.receiver_email" />
          <InfoItem :label="localText('发票类型', 'Invoice type')" :value="invoiceTypeLabel(selected.invoice_type)" />
          <InfoItem :label="localText('开票项目', 'Invoice item')" :value="selected.item_name" />
          <InfoItem :label="localText('发票抬头', 'Invoice title')" :value="selected.title" />
          <InfoItem :label="localText('税号', 'Tax ID')" :value="selected.tax_id || '-'" />
          <InfoItem :label="localText('开票金额', 'Invoice amount')" :value="formatMoney(selected.amount)" />
          <InfoItem :label="localText('服务费', 'Service fee')" :value="formatMoney(invoiceTaxFee(selected))" />
          <InfoItem :label="localText('来源订单数', 'Source orders')" :value="String(selected.source_order_count ?? 0)" />
          <InfoItem :label="localText('发票号', 'Invoice number')" :value="selected.invoice_no || '-'" />
          <InfoItem :label="localText('创建时间', 'Created at')" :value="formatDateTime(selected.created_at)" />
        </div>
        <div v-if="selected.note" class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p class="mb-1 font-medium text-gray-900 dark:text-white">{{ localText('用户备注', 'User note') }}</p>
          <p class="whitespace-pre-wrap">{{ selected.note }}</p>
        </div>
        <div v-if="selected.admin_note" class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p class="mb-1 font-medium text-gray-900 dark:text-white">{{ localText('后台备注', 'Admin note') }}</p>
          <p class="whitespace-pre-wrap">{{ selected.admin_note }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p class="mb-2 font-medium text-gray-900 dark:text-white">{{ localText('来源订单快照', 'Source order snapshot') }}</p>
          <div v-if="sourceOrders.length" class="space-y-2">
            <div v-for="order in sourceOrders" :key="order.id" class="rounded-md border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ order.id }} · {{ order.out_trade_no || '-' }}</p>
                <span :class="['badge', order.invoiceable ? 'badge-success' : 'badge-gray']">{{ order.invoiceable ? localText('可开票', 'Eligible') : localText('不可开票', 'Not eligible') }}</span>
              </div>
              <div class="mt-2 grid grid-cols-1 gap-2 md:grid-cols-2">
                <InfoItem :label="localText('来源类型', 'Source type')" :value="sourceTypeLabel(order.record_source)" />
                <InfoItem :label="localText('业务分类', 'Business category')" :value="order.business_category || '-'" />
                <InfoItem :label="localText('付款方式', 'Payment method')" :value="order.payment_type || '-'" />
                <InfoItem :label="localText('订单金额', 'Order amount')" :value="formatMoney(order.amount)" />
                <InfoItem :label="localText('退款抵扣', 'Refund offset')" :value="formatMoney(order.refund_amount)" />
                <InfoItem :label="localText('可开票', 'Eligible')" :value="order.invoiceable ? localText('是', 'Yes') : localText('否', 'No')" />
              </div>
            </div>
          </div>
          <p v-else class="text-xs text-gray-400">{{ localText('暂无来源订单快照', 'No source order snapshot') }}</p>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :show="actionDialog.show" :title="actionDialogTitle" @close="closeAction">
      <form class="space-y-4" @submit.prevent="submitAction">
        <div v-if="selected" class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <p>{{ localText('申请', 'Request') }} #{{ selected.id }} · {{ selected.user_email }}</p>
          <p class="mt-1">{{ localText('开票金额', 'Invoice amount') }} {{ formatMoney(selected.amount) }} · {{ localText('服务费', 'service fee') }} {{ formatMoney(invoiceTaxFee(selected)) }}</p>
        </div>
        <div v-if="actionDialog.type === 'complete'">
          <label class="input-label">{{ localText('发票号', 'Invoice number') }}</label>
          <input v-model.trim="actionForm.invoice_no" class="input" maxlength="128" :placeholder="localText('可选，电子票号/流水号', 'Optional e-invoice number or reference')" />
        </div>
        <div>
          <label class="input-label">{{ actionDialog.type === 'reject' ? localText('驳回原因', 'Rejection reason') : localText('后台备注', 'Admin note') }}</label>
          <textarea v-model.trim="actionForm.admin_note" class="input" rows="4" maxlength="2000" :required="actionDialog.type === 'reject'" :placeholder="localText('会展示给用户，请写清楚处理结果', 'Shown to the user; describe the outcome clearly')"></textarea>
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closeAction">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="actionSubmitting">
            <span v-if="actionSubmitting">{{ localText('处理中...', 'Processing...') }}</span>
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
import adminInvoicesAPI from '@/custom/api/admin/invoices'
import type { InvoiceRequest, InvoiceSourceOrder, InvoiceStatus, InvoiceType } from '@/custom/api/invoices'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { useI18n } from 'vue-i18n'

const InfoItem = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', { class: 'rounded-lg border border-gray-100 p-3 dark:border-dark-700' }, [
      h('p', { class: 'text-xs text-gray-500 dark:text-gray-400' }, props.label),
      h('p', { class: 'mt-1 break-words text-sm font-medium text-gray-900 dark:text-white' }, props.value),
    ])
  },
})

const { t, locale } = useI18n()
const appStore = useAppStore()
const isChinese = computed(() => String(locale.value).toLowerCase().startsWith('zh'))

function localText(zh: string, en: string): string {
  return isChinese.value ? zh : en
}

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

const sourceOrders = computed<InvoiceSourceOrder[]>(() => selected.value?.source_orders_json || [])

const statusOptions = computed(() => [
  { value: '', label: localText('全部状态', 'All statuses') },
  { value: 'pending', label: localText('待确认', 'Pending') },
  { value: 'approved', label: localText('已确认', 'Approved') },
  { value: 'rejected', label: localText('已驳回', 'Rejected') },
  { value: 'completed', label: localText('已完成', 'Completed') },
  { value: 'cancelled', label: localText('已取消', 'Cancelled') },
])

const actionDialogTitle = computed(() => {
  if (actionDialog.type === 'approve') return localText('确认开票申请', 'Approve Invoice Request')
  if (actionDialog.type === 'reject') return localText('驳回开票申请', 'Reject Invoice Request')
  if (actionDialog.type === 'complete') return localText('完成开票', 'Complete Invoice')
  return localText('处理开票申请', 'Process Invoice Request')
})

const actionSubmitText = computed(() => {
  if (actionDialog.type === 'approve') return localText('确认', 'Approve')
  if (actionDialog.type === 'reject') return localText('驳回', 'Reject')
  if (actionDialog.type === 'complete') return localText('完成', 'Complete')
  return localText('提交', 'Submit')
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
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', localText('加载发票申请失败', 'Failed to load invoice requests')))
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
      appStore.showSuccess(localText('已确认开票申请', 'Invoice request approved'))
    } else if (actionDialog.type === 'reject') {
      await adminInvoicesAPI.reject(selected.value.id, { admin_note: actionForm.admin_note })
      appStore.showSuccess(localText('已驳回开票申请', 'Invoice request rejected'))
    } else if (actionDialog.type === 'complete') {
      await adminInvoicesAPI.complete(selected.value.id, { invoice_no: actionForm.invoice_no, admin_note: actionForm.admin_note })
      appStore.showSuccess(localText('已完成开票，服务费已确认扣除', 'Invoice completed and service fee charged'))
    }
    closeAction()
    await load()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', localText('处理失败', 'Failed to process invoice request')))
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

function invoiceTaxFee(item: InvoiceRequest) {
  const reserved = Number(item.tax_fee) || 0
  if (reserved > 0) return reserved
  const amount = Number(item.amount) || 0
  if (amount <= 0) return 0
  return Math.max(amount * (Number(item.tax_rate) || 0.02), 50)
}

function formatDateTime(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString(isChinese.value ? 'zh-CN' : 'en-US')
}

function invoiceTypeLabel(type: InvoiceType | string) {
  const map: Record<string, string> = {
    company_vat_general: localText('企业普票', 'General VAT invoice'),
    company_vat_special: localText('企业专票', 'Special VAT invoice'),
    personal: localText('个人发票', 'Personal invoice'),
  }
  return map[type] || type
}

function sourceTypeLabel(source: string) {
  const map: Record<string, string> = {
    payment_order: localText('支付订单', 'Payment order'),
    redeem_code: localText('卡密兑换', 'Card-code redemption'),
    admin_adjustment: localText('管理员调整', 'Admin adjustment'),
    affiliate: localText('邀请返佣', 'Affiliate rebate'),
    invoice: localText('发票服务费', 'Invoice service fee'),
  }
  return map[source] || source || '-'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    pending: localText('待确认', 'Pending'),
    approved: localText('已确认', 'Approved'),
    rejected: localText('已驳回', 'Rejected'),
    completed: localText('已完成', 'Completed'),
    cancelled: localText('已取消', 'Cancelled'),
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
