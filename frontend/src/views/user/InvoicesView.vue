<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <div class="flex flex-col gap-4 border-b border-gray-200 pb-4 dark:border-dark-700 lg:flex-row lg:items-end lg:justify-between">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold tracking-normal text-gray-950 dark:text-white">订单中心</h1>
          <p class="mt-1 max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">
            查看充值订单，筛选可开票订单并提交发票申请。当前系统按可开票额度锁定金额，订单勾选用于汇总本次申请金额。
          </p>
        </div>
        <button class="btn btn-secondary w-full justify-center sm:w-auto" :disabled="loading" @click="reload">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>

      <div class="rounded-lg border border-blue-100 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-100">
        可开票订单会高亮显示；不可开票订单无法勾选，并显示具体原因。开票起始范围：系统已完成的余额充值订单。
      </div>

      <div class="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">订单总数</p>
          <p class="mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{{ orderPagination.total }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">当前页可开票</p>
          <p class="mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{{ invoiceableOrderRows.length }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">可开票金额</p>
          <p class="mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{{ formatMoney(summary?.available_amount) }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">处理中占用</p>
          <p class="mt-2 text-2xl font-semibold text-amber-600 dark:text-amber-300">{{ formatMoney(lockedInProgress) }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <p class="text-xs text-gray-500 dark:text-gray-400">当前页不可开票</p>
          <p class="mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{{ unavailableOrderRows.length }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_390px]">
        <section class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 p-4 dark:border-dark-700">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-950 dark:text-white">全部订单</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">处理中和已完成的开票申请会占用可申请额度，已驳回或已取消的申请会释放额度。</p>
              </div>
              <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:min-w-[520px] lg:grid-cols-4">
                <div class="lg:col-span-2">
                  <label class="input-label">订单号 / 交易单号</label>
                  <input v-model.trim="orderFilters.keyword" class="input" placeholder="搜索订单号" />
                </div>
                <div>
                  <label class="input-label">支付状态</label>
                  <Select v-model="orderFilters.status" :options="orderStatusOptions" @change="handleOrderServerFilterChange" />
                </div>
                <div>
                  <label class="input-label">开票范围</label>
                  <Select v-model="orderFilters.invoiceability" :options="invoiceabilityOptions" />
                </div>
              </div>
            </div>
            <div class="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-[1fr_auto_1fr] sm:items-end lg:max-w-[520px]">
              <div>
                <label class="input-label">开始时间</label>
                <input v-model="orderFilters.start_date" class="input" type="date" />
              </div>
              <span class="hidden pb-2 text-center text-sm text-gray-400 sm:block">~</span>
              <div>
                <label class="input-label">结束时间</label>
                <input v-model="orderFilters.end_date" class="input" type="date" />
              </div>
            </div>
          </div>

          <div v-if="ordersLoading" class="flex justify-center py-16">
            <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
          </div>

          <div v-else-if="visibleOrderRows.length === 0" class="py-16 text-center">
            <Icon name="inbox" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
            <p class="font-medium text-gray-700 dark:text-gray-200">暂无符合条件的订单</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">调整筛选条件后再查看。</p>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="min-w-[980px] w-full text-left text-sm">
              <thead class="bg-gray-50 text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-900 dark:text-gray-400">
                <tr>
                  <th class="w-10 px-4 py-3">
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="allVisibleInvoiceableSelected"
                      :disabled="visibleInvoiceableRows.length === 0"
                      @change="toggleAllVisibleOrders"
                    />
                  </th>
                  <th class="px-4 py-3">订单号</th>
                  <th class="px-4 py-3">交易单号</th>
                  <th class="px-4 py-3 text-right">开票金额</th>
                  <th class="px-4 py-3 text-right">手续费</th>
                  <th class="px-4 py-3">支付方式</th>
                  <th class="px-4 py-3">支付状态</th>
                  <th class="px-4 py-3">支付时间</th>
                  <th class="px-4 py-3">开票状态</th>
                  <th class="px-4 py-3">不可开票原因</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr
                  v-for="row in visibleOrderRows"
                  :key="row.order.id"
                  :class="[
                    row.invoiceable ? 'bg-emerald-50/40 dark:bg-emerald-500/5' : 'bg-white dark:bg-dark-800',
                    'transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/50'
                  ]"
                >
                  <td class="px-4 py-3 align-top">
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-40"
                      :checked="isOrderSelected(row)"
                      :disabled="!row.invoiceable"
                      @change="toggleOrderSelection(row)"
                    />
                  </td>
                  <td class="px-4 py-3 align-top font-mono text-xs text-gray-700 dark:text-gray-200">#{{ row.order.id }}</td>
                  <td class="max-w-[220px] truncate px-4 py-3 align-top font-mono text-xs text-gray-500 dark:text-gray-400" :title="row.order.out_trade_no">
                    {{ row.order.out_trade_no || '-' }}
                  </td>
                  <td class="px-4 py-3 text-right align-top font-semibold text-gray-950 dark:text-white">{{ formatMoney(row.invoiceAmount) }}</td>
                  <td class="px-4 py-3 text-right align-top text-gray-500 dark:text-gray-400">{{ formatOrderFee(row.order) }}</td>
                  <td class="px-4 py-3 align-top text-gray-700 dark:text-gray-300">{{ paymentTypeLabel(row.order.payment_type) }}</td>
                  <td class="px-4 py-3 align-top">
                    <span :class="['badge', orderStatusBadgeClass(row.order.status)]">{{ orderStatusLabel(row.order.status) }}</span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 align-top text-gray-500 dark:text-gray-400">{{ formatDateTime(row.paidAt) }}</td>
                  <td class="px-4 py-3 align-top">
                    <span :class="['badge', row.invoiceable ? 'badge-success' : 'badge-gray']">{{ row.invoiceable ? '可申请' : '不可开' }}</span>
                  </td>
                  <td class="max-w-[220px] px-4 py-3 align-top text-gray-500 dark:text-gray-400">{{ row.reason || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="border-t border-gray-100 dark:border-dark-700">
            <Pagination
              v-if="orderPagination.total > 0"
              :page="orderPagination.page"
              :total="orderPagination.total"
              :page-size="orderPagination.page_size"
              @update:page="handleOrderPageChange"
              @update:pageSize="handleOrderPageSizeChange"
            />
          </div>
        </section>

        <aside class="space-y-5">
          <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 xl:sticky xl:top-5">
            <div class="mb-4">
              <h2 class="text-base font-semibold text-gray-950 dark:text-white">开票信息</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">已选 {{ selectedOrderRows.length }} 笔订单 · {{ formatMoney(selectedInvoiceAmount) }}</p>
            </div>

            <div class="mb-4 rounded-lg bg-gray-50 p-3 text-xs leading-5 text-gray-600 dark:bg-dark-900 dark:text-gray-300">
              <p>最低开票金额 {{ formatMoney(summary?.min_amount || 500) }}，当前可申请 {{ formatMoney(summary?.available_amount) }}。</p>
              <p>完成开票时扣除税点 {{ formatMoney(taxFeePreview) }}（{{ summary?.tax_rate_percent || 2 }}%）。</p>
            </div>

            <form class="space-y-4" @submit.prevent="submit">
              <div>
                <div class="flex items-center justify-between gap-3">
                  <label class="input-label">选择模板</label>
                  <button type="button" class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openCreateTemplateDialog">
                    保存为模板
                  </button>
                </div>
                <Select v-model="selectedTemplateId" :options="templateOptions" @change="applySelectedTemplate" />
                <div v-if="activeTemplate" class="mt-2 flex flex-wrap items-center gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="openUpdateTemplateDialog">更新模板</button>
                  <button v-if="!activeTemplate.is_default" type="button" class="btn btn-secondary btn-sm" :disabled="templateSaving" @click="setDefaultTemplate">设为默认</button>
                  <button type="button" class="btn btn-danger btn-sm" :disabled="templateSaving" @click="deleteSelectedTemplate">删除</button>
                </div>
              </div>

              <div>
                <label class="input-label">发票类型</label>
                <Select v-model="form.invoice_type" :options="invoiceTypeOptions" />
              </div>

              <div>
                <label class="input-label">发票抬头</label>
                <input v-model.trim="form.title" class="input" maxlength="255" required placeholder="公司全称或个人姓名" />
              </div>

              <div v-if="form.invoice_type !== 'personal'">
                <label class="input-label">税号</label>
                <input v-model.trim="form.tax_id" class="input uppercase" maxlength="100" required placeholder="纳税人识别号" />
              </div>

              <div>
                <label class="input-label">开票项目</label>
                <input v-model.trim="form.item_name" class="input" maxlength="100" required placeholder="信息技术服务费" />
              </div>

              <div>
                <label class="input-label">开票金额</label>
                <input v-model.number="form.amount" class="input" type="number" min="0" step="0.01" required placeholder="500.00" />
              </div>

              <div>
                <label class="input-label">接收邮箱</label>
                <input v-model.trim="form.receiver_email" class="input" type="email" maxlength="255" required placeholder="接收电子发票的邮箱" />
              </div>

              <div>
                <div class="flex items-center justify-between gap-3">
                  <label class="input-label">备注</label>
                  <span class="text-xs text-gray-400">{{ form.note.length }}/1000</span>
                </div>
                <textarea v-model.trim="form.note" class="input" rows="3" maxlength="1000" placeholder="可选"></textarea>
              </div>

              <button class="btn btn-primary w-full py-3" :disabled="!canSubmit || submitting">
                <span v-if="submitting" class="inline-flex items-center gap-2">
                  <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                  提交中
                </span>
                <span v-else>提交开票申请</span>
              </button>
              <p v-if="!summary?.can_apply" class="text-center text-xs text-amber-600 dark:text-amber-300">当前可开票金额未达到起开金额。</p>
            </form>
          </section>
        </aside>
      </div>

      <section class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-100 p-5 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">我的申请记录</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">查看发票申请处理状态。</p>
        </div>

        <div v-if="loading" class="flex justify-center py-16">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
        </div>

        <div v-else-if="invoices.length === 0" class="py-16 text-center">
          <Icon name="document" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
          <p class="font-medium text-gray-700 dark:text-gray-200">暂无申请记录</p>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-[880px] w-full text-left text-sm">
            <thead class="bg-gray-50 text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-900 dark:text-gray-400">
              <tr>
                <th class="px-4 py-3">发票抬头</th>
                <th class="px-4 py-3">发票类型</th>
                <th class="px-4 py-3 text-right">开票金额</th>
                <th class="px-4 py-3">订单数</th>
                <th class="px-4 py-3">状态</th>
                <th class="px-4 py-3">发票号码</th>
                <th class="px-4 py-3">提交时间</th>
                <th class="px-4 py-3">备注</th>
                <th class="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in invoices" :key="item.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                <td class="max-w-[220px] truncate px-4 py-3 font-medium text-gray-950 dark:text-white" :title="item.title">{{ item.title }}</td>
                <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ invoiceTypeLabel(item.invoice_type) }}</td>
                <td class="px-4 py-3 text-right font-semibold text-gray-950 dark:text-white">{{ formatMoney(item.amount) }}</td>
                <td class="px-4 py-3 text-gray-500 dark:text-gray-400">按金额申请</td>
                <td class="px-4 py-3"><span :class="['badge', statusBadgeClass(item.status)]">{{ statusLabel(item.status) }}</span></td>
                <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{{ item.invoice_no || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-500 dark:text-gray-400">{{ formatDateTime(item.created_at) }}</td>
                <td class="max-w-[220px] truncate px-4 py-3 text-gray-500 dark:text-gray-400" :title="item.admin_note || item.note">{{ item.admin_note || item.note || '-' }}</td>
                <td class="px-4 py-3 text-right">
                  <button v-if="item.status === 'pending'" class="text-sm font-medium text-red-600 hover:text-red-700 dark:text-red-400" @click="cancel(item)">取消</button>
                  <span v-else class="text-sm text-gray-400">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="border-t border-gray-100 dark:border-dark-700">
          <Pagination
            v-if="invoicePagination.total > 0"
            :page="invoicePagination.page"
            :total="invoicePagination.total"
            :page-size="invoicePagination.page_size"
            @update:page="handleInvoicePageChange"
            @update:pageSize="handleInvoicePageSizeChange"
          />
        </div>
      </section>
    </div>

    <BaseDialog :show="templateDialog.open" :title="templateDialog.mode === 'update' ? '更新开票模板' : '保存开票模板'" width="narrow" @close="templateDialog.open = false">
      <div class="space-y-4">
        <div>
          <label class="input-label">模板名称</label>
          <input v-model.trim="templateDialog.name" class="input" maxlength="80" placeholder="默认模板" />
        </div>
        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-700">
          <input v-model="templateDialog.is_default" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <span class="text-gray-700 dark:text-gray-300">设为默认模板</span>
        </label>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="templateDialog.open = false">取消</button>
          <button class="btn btn-primary" :disabled="!canSaveTemplate || templateSaving" @click="saveTemplate">
            {{ templateSaving ? '保存中' : '保存' }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import invoicesAPI, { type InvoiceRequest, type InvoiceSummary, type InvoiceTemplate, type InvoiceType } from '@/api/invoices'
import { paymentAPI } from '@/api/payment'
import type { PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { useI18n } from 'vue-i18n'

type InvoiceabilityFilter = 'all' | 'available' | 'unavailable'

interface OrderRow {
  order: PaymentOrder
  invoiceable: boolean
  reason: string
  paidAt: string | null
  invoiceAmount: number
}

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const ordersLoading = ref(false)
const submitting = ref(false)
const templateSaving = ref(false)
const summary = ref<InvoiceSummary | null>(null)
const invoices = ref<InvoiceRequest[]>([])
const templates = ref<InvoiceTemplate[]>([])
const orders = ref<PaymentOrder[]>([])
const selectedOrderIds = ref<Set<number>>(new Set())
const selectedTemplateId = ref<number | ''>('')

const invoicePagination = reactive({ page: 1, page_size: 10, total: 0 })
const orderPagination = reactive({ page: 1, page_size: 20, total: 0 })

const orderFilters = reactive({
  keyword: '',
  status: '',
  invoiceability: 'all' as InvoiceabilityFilter,
  start_date: '',
  end_date: '',
})

const form = reactive({
  invoice_type: 'company_vat_general' as InvoiceType,
  title: '',
  tax_id: '',
  item_name: '信息技术服务费',
  amount: undefined as number | undefined,
  receiver_email: '',
  note: '',
})

const templateDialog = reactive({
  open: false,
  mode: 'create' as 'create' | 'update',
  id: 0,
  name: '',
  is_default: false,
})

const invoiceTypeOptions = [
  { value: 'company_vat_general', label: '普通发票' },
  { value: 'company_vat_special', label: '专用发票' },
  { value: 'personal', label: '个人发票' },
]

const orderStatusOptions = [
  { value: '', label: '全部状态' },
  { value: 'PENDING', label: '待支付' },
  { value: 'COMPLETED', label: '已完成' },
  { value: 'FAILED', label: '失败' },
  { value: 'CANCELLED', label: '已取消' },
  { value: 'REFUNDED', label: '已退款' },
]

const invoiceabilityOptions = [
  { value: 'all', label: '全部订单' },
  { value: 'available', label: '可开票' },
  { value: 'unavailable', label: '不可开票' },
]

const lockedInProgress = computed(() => Math.max((summary.value?.locked_amount || 0) - (summary.value?.invoiced_amount || 0), 0))
const taxFeePreview = computed(() => roundMoney((Number(form.amount) || 0) * (summary.value?.tax_rate || 0.02)))

const orderRows = computed<OrderRow[]>(() => orders.value.map((order) => {
  const invoiceAmount = roundMoney(Number(order.amount) || 0)
  const paidAt = order.completed_at || order.paid_at || order.created_at || null
  const reason = getUnavailableReason(order, invoiceAmount)
  return {
    order,
    invoiceable: reason === '',
    reason,
    paidAt,
    invoiceAmount,
  }
}))

const visibleOrderRows = computed(() => {
  const keyword = orderFilters.keyword.trim().toLowerCase()
  return orderRows.value.filter((row) => {
    if (orderFilters.invoiceability === 'available' && !row.invoiceable) return false
    if (orderFilters.invoiceability === 'unavailable' && row.invoiceable) return false
    if (keyword) {
      const haystack = `${row.order.id} ${row.order.out_trade_no || ''}`.toLowerCase()
      if (!haystack.includes(keyword)) return false
    }
    if (!isWithinDateRange(row.paidAt)) return false
    return true
  })
})

const invoiceableOrderRows = computed(() => orderRows.value.filter((row) => row.invoiceable))
const unavailableOrderRows = computed(() => orderRows.value.filter((row) => !row.invoiceable))
const visibleInvoiceableRows = computed(() => visibleOrderRows.value.filter((row) => row.invoiceable))
const allVisibleInvoiceableSelected = computed(() => {
  if (visibleInvoiceableRows.value.length === 0) return false
  return visibleInvoiceableRows.value.every((row) => selectedOrderIds.value.has(row.order.id))
})
const selectedOrderRows = computed(() => orderRows.value.filter((row) => row.invoiceable && selectedOrderIds.value.has(row.order.id)))
const selectedInvoiceAmount = computed(() => roundMoney(selectedOrderRows.value.reduce((sum, row) => sum + row.invoiceAmount, 0)))

const activeTemplate = computed(() => {
  const id = Number(selectedTemplateId.value)
  if (!id) return null
  return templates.value.find((item) => item.id === id) || null
})

const templateOptions = computed(() => [
  { value: '', label: templates.value.length > 0 ? '不使用模板' : '暂无模板' },
  ...templates.value.map((item) => ({
    value: item.id,
    label: `${item.name}${item.is_default ? '（默认）' : ''}`,
  })),
])

const canSubmit = computed(() => {
  const amount = Number(form.amount) || 0
  if (!summary.value?.can_apply) return false
  if (amount < (summary.value.min_amount || 500)) return false
  if (amount > (summary.value.available_amount || 0)) return false
  if (!form.title.trim() || !form.item_name.trim() || !form.receiver_email.trim()) return false
  if (form.invoice_type !== 'personal' && !form.tax_id.trim()) return false
  return true
})

const canSaveTemplate = computed(() => {
  if (!templateDialog.name.trim()) return false
  if (!form.title.trim() || !form.item_name.trim() || !form.receiver_email.trim()) return false
  if (form.invoice_type !== 'personal' && !form.tax_id.trim()) return false
  return true
})

async function reload() {
  loading.value = true
  try {
    await Promise.all([
      loadSummary(),
      loadInvoices(),
      loadOrders(),
      loadTemplates(),
    ])
  } finally {
    loading.value = false
  }
}

async function loadSummary() {
  try {
    const res = await invoicesAPI.getSummary()
    summary.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '加载开票额度失败'))
  }
}

async function loadInvoices() {
  try {
    const res = await invoicesAPI.list({ page: invoicePagination.page, page_size: invoicePagination.page_size })
    invoices.value = res.data.items || []
    invoicePagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '加载开票申请失败'))
  }
}

async function loadOrders() {
  ordersLoading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: orderPagination.page,
      page_size: orderPagination.page_size,
      status: orderFilters.status || undefined,
    })
    orders.value = res.data.items || []
    orderPagination.total = res.data.total || 0
    selectedOrderIds.value = new Set([...selectedOrderIds.value].filter((id) => orders.value.some((order) => order.id === id)))
    syncAmountFromSelection()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', '加载订单失败'))
  } finally {
    ordersLoading.value = false
  }
}

async function loadTemplates() {
  try {
    const res = await invoicesAPI.listTemplates()
    templates.value = res.data || []
    if (!selectedTemplateId.value) {
      const defaultTemplate = templates.value.find((item) => item.is_default)
      if (defaultTemplate) {
        selectedTemplateId.value = defaultTemplate.id
        copyTemplateToForm(defaultTemplate)
      }
    }
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '加载开票模板失败'))
  }
}

async function submit() {
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  try {
    await invoicesAPI.create({
      invoice_type: form.invoice_type,
      title: form.title,
      tax_id: form.invoice_type === 'personal' ? '' : form.tax_id,
      item_name: form.item_name,
      amount: Number(form.amount),
      receiver_email: form.receiver_email,
      note: form.note,
    })
    appStore.showSuccess('开票申请已提交')
    selectedOrderIds.value = new Set()
    form.amount = undefined
    form.note = ''
    await Promise.all([loadSummary(), loadInvoices()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '提交开票申请失败'))
  } finally {
    submitting.value = false
  }
}

async function cancel(item: InvoiceRequest) {
  try {
    await invoicesAPI.cancel(item.id)
    appStore.showSuccess('已取消开票申请')
    await Promise.all([loadSummary(), loadInvoices()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '取消失败'))
  }
}

function handleInvoicePageChange(page: number) {
  invoicePagination.page = page
  loadInvoices()
}

function handleInvoicePageSizeChange(size: number) {
  invoicePagination.page_size = size
  invoicePagination.page = 1
  loadInvoices()
}

function handleOrderPageChange(page: number) {
  orderPagination.page = page
  loadOrders()
}

function handleOrderPageSizeChange(size: number) {
  orderPagination.page_size = size
  orderPagination.page = 1
  loadOrders()
}

function handleOrderServerFilterChange() {
  orderPagination.page = 1
  loadOrders()
}

function isOrderSelected(row: OrderRow) {
  return selectedOrderIds.value.has(row.order.id)
}

function toggleOrderSelection(row: OrderRow) {
  if (!row.invoiceable) return
  const next = new Set(selectedOrderIds.value)
  if (next.has(row.order.id)) {
    next.delete(row.order.id)
  } else {
    next.add(row.order.id)
  }
  selectedOrderIds.value = next
  syncAmountFromSelection()
}

function toggleAllVisibleOrders() {
  const next = new Set(selectedOrderIds.value)
  if (allVisibleInvoiceableSelected.value) {
    visibleInvoiceableRows.value.forEach((row) => next.delete(row.order.id))
  } else {
    visibleInvoiceableRows.value.forEach((row) => next.add(row.order.id))
  }
  selectedOrderIds.value = next
  syncAmountFromSelection()
}

function syncAmountFromSelection() {
  if (selectedInvoiceAmount.value <= 0) return
  const available = summary.value?.available_amount || selectedInvoiceAmount.value
  form.amount = roundMoney(Math.min(selectedInvoiceAmount.value, available))
}

function applySelectedTemplate() {
  if (activeTemplate.value) {
    copyTemplateToForm(activeTemplate.value)
  }
}

function copyTemplateToForm(template: InvoiceTemplate) {
  form.invoice_type = template.invoice_type
  form.title = template.title
  form.tax_id = template.tax_id || ''
  form.item_name = template.item_name || '信息技术服务费'
  form.receiver_email = template.receiver_email || ''
  form.note = template.note || ''
}

function openCreateTemplateDialog() {
  templateDialog.mode = 'create'
  templateDialog.id = 0
  templateDialog.name = form.title.trim() || '默认模板'
  templateDialog.is_default = templates.value.length === 0
  templateDialog.open = true
}

function openUpdateTemplateDialog() {
  if (!activeTemplate.value) return
  templateDialog.mode = 'update'
  templateDialog.id = activeTemplate.value.id
  templateDialog.name = activeTemplate.value.name
  templateDialog.is_default = activeTemplate.value.is_default
  templateDialog.open = true
}

async function saveTemplate() {
  if (!canSaveTemplate.value || templateSaving.value) return
  templateSaving.value = true
  try {
    const payload = {
      name: templateDialog.name,
      invoice_type: form.invoice_type,
      title: form.title,
      tax_id: form.invoice_type === 'personal' ? '' : form.tax_id,
      item_name: form.item_name,
      receiver_email: form.receiver_email,
      note: form.note,
      is_default: templateDialog.is_default,
    }
    const res = templateDialog.mode === 'update' && templateDialog.id > 0
      ? await invoicesAPI.updateTemplate(templateDialog.id, payload)
      : await invoicesAPI.createTemplate(payload)
    templateDialog.open = false
    appStore.showSuccess(templateDialog.mode === 'update' ? '开票模板已更新' : '开票模板已保存')
    await loadTemplates()
    selectedTemplateId.value = res.data.id
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '保存开票模板失败'))
  } finally {
    templateSaving.value = false
  }
}

async function setDefaultTemplate() {
  if (!activeTemplate.value || templateSaving.value) return
  templateSaving.value = true
  try {
    const res = await invoicesAPI.setDefaultTemplate(activeTemplate.value.id)
    appStore.showSuccess('默认模板已更新')
    await loadTemplates()
    selectedTemplateId.value = res.data.id
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '设置默认模板失败'))
  } finally {
    templateSaving.value = false
  }
}

async function deleteSelectedTemplate() {
  if (!activeTemplate.value || templateSaving.value) return
  if (!window.confirm('删除这个开票模板？')) return
  templateSaving.value = true
  try {
    await invoicesAPI.deleteTemplate(activeTemplate.value.id)
    appStore.showSuccess('开票模板已删除')
    selectedTemplateId.value = ''
    await loadTemplates()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '删除开票模板失败'))
  } finally {
    templateSaving.value = false
  }
}

function getUnavailableReason(order: PaymentOrder, amount: number) {
  if (order.order_type !== 'balance') return '非余额充值订单'
  if (order.status !== 'COMPLETED') return '订单未完成'
  if (amount <= 0) return '订单金额为 0'
  return ''
}

function isWithinDateRange(value?: string | null) {
  if (!value) return true
  const time = new Date(value).getTime()
  if (Number.isNaN(time)) return true
  if (orderFilters.start_date) {
    const start = new Date(`${orderFilters.start_date}T00:00:00`).getTime()
    if (time < start) return false
  }
  if (orderFilters.end_date) {
    const end = new Date(`${orderFilters.end_date}T23:59:59`).getTime()
    if (time > end) return false
  }
  return true
}

function formatMoney(value?: number | null) {
  return `¥${(Number(value) || 0).toFixed(2)}`
}

function formatOrderFee(order: PaymentOrder) {
  const payAmount = Number(order.pay_amount) || 0
  const amount = Number(order.amount) || 0
  if (payAmount > amount) return formatMoney(roundMoney(payAmount - amount))
  if (Number(order.fee_rate) > 0) return `${order.fee_rate}%`
  return '-'
}

function roundMoney(value: number) {
  return Math.round(value * 100) / 100
}

function formatDateTime(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function invoiceTypeLabel(type: InvoiceType | string) {
  const map: Record<string, string> = {
    company_vat_general: '普通发票',
    company_vat_special: '专用发票',
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

function orderStatusLabel(status: string) {
  const map: Record<string, string> = {
    PENDING: '待支付',
    PAID: '已支付',
    RECHARGING: '充值中',
    COMPLETED: '已完成',
    EXPIRED: '已过期',
    CANCELLED: '已取消',
    FAILED: '失败',
    REFUND_REQUESTED: '退款申请中',
    REFUNDING: '退款中',
    PARTIALLY_REFUNDED: '部分退款',
    REFUNDED: '已退款',
    REFUND_FAILED: '退款失败',
  }
  return map[status] || status
}

function orderStatusBadgeClass(status: string) {
  const map: Record<string, string> = {
    PENDING: 'badge-warning',
    PAID: 'badge-primary',
    RECHARGING: 'badge-warning',
    COMPLETED: 'badge-success',
    EXPIRED: 'badge-gray',
    CANCELLED: 'badge-gray',
    FAILED: 'badge-danger',
    REFUND_REQUESTED: 'badge-warning',
    REFUNDING: 'badge-warning',
    PARTIALLY_REFUNDED: 'badge-warning',
    REFUNDED: 'badge-gray',
    REFUND_FAILED: 'badge-danger',
  }
  return map[status] || 'badge-gray'
}

function paymentTypeLabel(type: string) {
  const map: Record<string, string> = {
    alipay: '支付宝',
    alipay_direct: '支付宝',
    wxpay: '微信支付',
    wxpay_direct: '微信支付',
    stripe: '银行卡',
    easypay: '易支付',
    airwallex: 'Airwallex',
    gmpay: 'GMPay',
    usdt: 'USDT',
  }
  return map[type] || type || '-'
}

onMounted(reload)
</script>
