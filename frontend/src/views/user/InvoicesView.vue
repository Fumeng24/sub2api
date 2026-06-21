<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="overflow-hidden rounded-3xl border border-emerald-200 bg-gradient-to-br from-emerald-50 via-white to-sky-50 p-6 shadow-sm dark:border-emerald-900/40 dark:from-emerald-950/30 dark:via-dark-900 dark:to-sky-950/20">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p class="text-sm font-semibold uppercase tracking-[0.18em] text-emerald-700 dark:text-emerald-300">Invoice</p>
            <h1 class="mt-2 text-2xl font-bold text-gray-950 dark:text-white">开票申请</h1>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
              可开票金额按已完成余额充值统计，已完成和处理中申请都会占用额度。500 元起开，完成开票时从账户余额扣除 2% 税点。
            </p>
          </div>
          <button class="btn btn-secondary" :disabled="loading" @click="reload">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            刷新
          </button>
        </div>

        <div class="mt-6 grid grid-cols-1 gap-3 md:grid-cols-4">
          <div class="rounded-2xl border border-white/70 bg-white/80 p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800/70">
            <p class="text-xs text-gray-500 dark:text-gray-400">累计可开票充值</p>
            <p class="mt-2 text-2xl font-bold text-gray-950 dark:text-white">{{ formatMoney(summary?.recharge_amount) }}</p>
          </div>
          <div class="rounded-2xl border border-white/70 bg-white/80 p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800/70">
            <p class="text-xs text-gray-500 dark:text-gray-400">已开票金额</p>
            <p class="mt-2 text-2xl font-bold text-gray-950 dark:text-white">{{ formatMoney(summary?.invoiced_amount) }}</p>
          </div>
          <div class="rounded-2xl border border-white/70 bg-white/80 p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800/70">
            <p class="text-xs text-gray-500 dark:text-gray-400">处理中占用</p>
            <p class="mt-2 text-2xl font-bold text-amber-600 dark:text-amber-300">{{ formatMoney(lockedInProgress) }}</p>
          </div>
          <div class="rounded-2xl border border-emerald-200 bg-emerald-600 p-4 text-white shadow-sm dark:border-emerald-500/40">
            <p class="text-xs text-emerald-100">当前可申请</p>
            <p class="mt-2 text-3xl font-black">{{ formatMoney(summary?.available_amount) }}</p>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-[0.95fr_1.35fr]">
        <div class="card p-5">
          <div class="mb-4">
            <h2 class="text-lg font-bold text-gray-950 dark:text-white">提交开票信息</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">请确认开票金额不超过可申请额度。专票信息请填写完整，避免反复确认。</p>
          </div>

          <form class="space-y-4" @submit.prevent="submit">
            <div>
              <label class="input-label">发票类型</label>
              <Select v-model="form.invoice_type" :options="invoiceTypeOptions" />
            </div>

            <div>
              <label class="input-label">发票抬头</label>
              <input v-model.trim="form.title" class="input" maxlength="255" required placeholder="公司名称或个人姓名" />
            </div>

            <div v-if="form.invoice_type !== 'personal'">
              <label class="input-label">税号</label>
              <input v-model.trim="form.tax_id" class="input uppercase" maxlength="100" required placeholder="统一社会信用代码 / 纳税人识别号" />
            </div>

            <div>
              <label class="input-label">开票项目</label>
              <input v-model.trim="form.item_name" class="input" maxlength="100" required placeholder="例如：信息技术服务费" />
            </div>

            <div>
              <label class="input-label">开票金额</label>
              <input v-model.number="form.amount" class="input" type="number" min="0" step="0.01" required placeholder="500.00" />
              <div class="mt-2 rounded-xl bg-gray-50 p-3 text-xs leading-5 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                <p>最低开票 {{ formatMoney(summary?.min_amount || 500) }}，当前可开 {{ formatMoney(summary?.available_amount) }}。</p>
                <p>完成开票时扣除税点：{{ formatMoney(taxFeePreview) }}（{{ summary?.tax_rate_percent || 2 }}%），当前余额 {{ formatMoney(summary?.current_balance) }}。</p>
              </div>
            </div>

            <div>
              <label class="input-label">接收邮箱</label>
              <input v-model.trim="form.receiver_email" class="input" type="email" maxlength="255" required placeholder="用于接收电子发票" />
            </div>

            <div>
              <label class="input-label">备注</label>
              <textarea v-model.trim="form.note" class="input" rows="3" maxlength="2000" placeholder="需要开票前沟通的内容可以写在这里"></textarea>
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
        </div>

        <div class="card overflow-hidden">
          <div class="flex items-center justify-between border-b border-gray-100 p-5 dark:border-dark-700">
            <div>
              <h2 class="text-lg font-bold text-gray-950 dark:text-white">申请记录</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">后台确认后会进入开票处理，完成后锁定该金额。</p>
            </div>
          </div>

          <div v-if="loading" class="flex justify-center py-16">
            <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
          </div>

          <div v-else-if="invoices.length === 0" class="py-16 text-center">
            <Icon name="document" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
            <p class="font-medium text-gray-700 dark:text-gray-200">还没有开票申请</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">充值累计达到起开金额后即可在左侧提交。</p>
          </div>

          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="item in invoices" :key="item.id" class="p-5">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span :class="['badge', statusBadgeClass(item.status)]">{{ statusLabel(item.status) }}</span>
                    <span class="text-sm font-semibold text-gray-950 dark:text-white">#{{ item.id }}</span>
                    <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(item.created_at) }}</span>
                  </div>
                  <p class="mt-2 truncate text-base font-bold text-gray-950 dark:text-white">{{ item.title }}</p>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ item.item_name }} · {{ item.receiver_email }}</p>
                </div>
                <div class="text-left sm:text-right">
                  <p class="text-xl font-black text-gray-950 dark:text-white">{{ formatMoney(item.amount) }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">税点 {{ formatMoney(item.tax_fee || item.amount * item.tax_rate) }}</p>
                </div>
              </div>

              <div v-if="item.admin_note || item.invoice_no" class="mt-3 rounded-xl bg-gray-50 p-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                <p v-if="item.invoice_no">发票号：{{ item.invoice_no }}</p>
                <p v-if="item.admin_note">后台备注：{{ item.admin_note }}</p>
              </div>

              <div v-if="item.status === 'pending'" class="mt-3">
                <button class="text-sm font-medium text-red-600 hover:text-red-700 dark:text-red-400" @click="cancel(item)">取消申请</button>
              </div>
            </div>
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
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import invoicesAPI, { type InvoiceRequest, type InvoiceSummary, type InvoiceType } from '@/api/invoices'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const summary = ref<InvoiceSummary | null>(null)
const invoices = ref<InvoiceRequest[]>([])
const pagination = reactive({ page: 1, page_size: 10, total: 0 })

const form = reactive({
  invoice_type: 'company_vat_general' as InvoiceType,
  title: '',
  tax_id: '',
  item_name: '信息技术服务费',
  amount: undefined as number | undefined,
  receiver_email: '',
  note: '',
})

const invoiceTypeOptions = [
  { value: 'company_vat_general', label: '企业普票' },
  { value: 'company_vat_special', label: '企业专票' },
  { value: 'personal', label: '个人发票' },
]

const lockedInProgress = computed(() => Math.max((summary.value?.locked_amount || 0) - (summary.value?.invoiced_amount || 0), 0))
const taxFeePreview = computed(() => roundMoney((Number(form.amount) || 0) * (summary.value?.tax_rate || 0.02)))
const canSubmit = computed(() => {
  const amount = Number(form.amount) || 0
  if (!summary.value?.can_apply) return false
  if (amount < (summary.value.min_amount || 500)) return false
  if (amount > (summary.value.available_amount || 0)) return false
  if (!form.title.trim() || !form.item_name.trim() || !form.receiver_email.trim()) return false
  if (form.invoice_type !== 'personal' && !form.tax_id.trim()) return false
  return true
})

async function reload() {
  loading.value = true
  try {
    const [summaryRes, listRes] = await Promise.all([
      invoicesAPI.getSummary(),
      invoicesAPI.list({ page: pagination.page, page_size: pagination.page_size }),
    ])
    summary.value = summaryRes.data
    invoices.value = listRes.data.items || []
    pagination.total = listRes.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '加载开票信息失败'))
  } finally {
    loading.value = false
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
    form.amount = undefined
    form.note = ''
    await reload()
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
    await reload()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', '取消失败'))
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  reload()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function formatMoney(value?: number | null) {
  return `¥${(Number(value) || 0).toFixed(2)}`
}

function roundMoney(value: number) {
  return Math.round(value * 100) / 100
}

function formatDateTime(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
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

onMounted(reload)
</script>
