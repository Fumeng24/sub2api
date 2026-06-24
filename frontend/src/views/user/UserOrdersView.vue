<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold tracking-normal text-[var(--apple-text)]">
            {{ t('nav.myOrders') }}
          </h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-[var(--apple-muted)]">
            {{ t('userOrders.description') }}
          </p>
        </div>
        <button class="btn btn-primary w-full justify-center sm:w-auto" @click="router.push('/purchase')">
          {{ t('userOrders.newOrder') }}
        </button>
      </header>

      <section class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-4 py-3 shadow-sm">
        <div class="grid gap-2 sm:flex sm:flex-wrap">
          <span
            v-for="item in orderTrustItems"
            :key="item"
            class="inline-flex min-w-0 items-center gap-1.5 rounded-md bg-[var(--apple-surface-elevated)] px-2.5 py-1 text-xs font-medium text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]"
          >
            <Icon name="checkCircle" size="xs" class="text-[var(--apple-success)]" />
            <span class="min-w-0 truncate">{{ item }}</span>
          </span>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-3 md:grid-cols-3" :aria-label="t('userOrders.summary.label')">
        <article
          v-for="item in orderSummaryItems"
          :key="item.key"
          class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm"
        >
          <p class="text-xs font-medium text-[var(--apple-muted)]">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-semibold tracking-normal text-[var(--apple-text)]">{{ item.value }}</p>
          <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">{{ item.hint }}</p>
        </article>
      </section>

      <!-- Filters -->
      <section class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div class="w-full sm:max-w-xs">
            <label class="input-label">{{ t('userOrders.statusFilter') }}</label>
            <Select v-model="currentFilter" :options="statusFilters" @change="fetchOrders" />
          </div>
          <div class="flex items-center justify-end">
            <button
              class="btn btn-secondary btn-icon"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="fetchOrders"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </section>

      <!-- Table -->
      <OrderTable :orders="orders" :loading="loading">
        <template #actions="{ row }">
          <div class="flex flex-wrap items-center gap-2">
            <button
              v-if="row.status === 'PENDING'"
              class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
              @click="handleCancel(row.id)"
            >
              <Icon name="x" size="sm" />
              <span>{{ t('payment.orders.cancel') }}</span>
            </button>
            <button
              v-if="canRequestRefund(row)"
              class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
              @click="openRefundDialog(row)"
            >
              <Icon name="dollar" size="sm" />
              <span>{{ t('payment.orders.requestRefund') }}</span>
            </button>
            <button
              class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium text-[var(--apple-blue)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-blue-hover)]"
              @click="openOrderTicket(row)"
            >
              <Icon name="chatBubble" size="sm" />
              <span>{{ t('tickets.createTicket') }}</span>
            </button>
          </div>
        </template>
      </OrderTable>

      <!-- Pagination -->
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <!-- Cancel Confirm Dialog -->
    <BaseDialog :show="!!cancelTargetId" :title="t('payment.orders.cancel')" width="narrow" @close="cancelTargetId = null">
      <p class="text-sm leading-6 text-[var(--apple-muted)]">{{ t('userOrders.cancelDescription') }}</p>
      <template #footer>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button class="btn btn-secondary w-full sm:w-auto" @click="cancelTargetId = null">{{ t('common.cancel') }}</button>
          <button
            class="btn w-full bg-[var(--apple-danger)] text-white hover:opacity-90 sm:w-auto"
            :disabled="actionLoading"
            @click="confirmCancel"
          >
            {{ actionLoading ? t('common.processing') : t('payment.orders.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Refund Dialog -->
    <BaseDialog :show="!!refundTarget" :title="t('payment.orders.requestRefund')" @close="refundTarget = null">
      <div v-if="refundTarget" class="space-y-4">
        <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4">
          <div class="flex justify-between text-sm">
            <span class="text-[var(--apple-muted)]">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono text-[var(--apple-text)]">#{{ refundTarget.id }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-[var(--apple-muted)]">{{ refundTarget.order_type === 'balance' ? t('payment.orders.creditedBalance') : t('payment.orders.amount') }}</span>
            <span class="text-[var(--apple-text)]">{{ formatRefundTargetAmount(refundTarget) }}</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('userOrders.refundNote') }}</label>
          <textarea v-model="refundReason" rows="3" class="input mt-1 w-full" :placeholder="t('userOrders.refundNotePlaceholder')" />
        </div>
      </div>
      <template #footer>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button class="btn btn-secondary w-full sm:w-auto" @click="refundTarget = null">{{ t('common.cancel') }}</button>
          <button
            class="btn btn-primary w-full sm:w-auto"
            :disabled="actionLoading || !refundReason.trim()"
            @click="confirmRefund"
          >
            {{ actionLoading ? t('common.processing') : t('payment.orders.requestRefund') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import { formatBalanceCreditAmount, formatOrderPaymentAmount, shouldShowCreditedBalance } from '@/components/payment/orderAmounts'

const i18n = useI18n()
const { t } = i18n
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const orderTrustItems = computed(() => [
  t('userOrders.trust.amount'),
  t('userOrders.trust.status'),
  t('userOrders.trust.privacy'),
  t('userOrders.trust.support'),
])
const orderSummaryItems = computed(() => [
  {
    key: 'total',
    label: t('userOrders.summary.total'),
    value: pagination.total,
    hint: t('userOrders.summary.totalHint'),
  },
  {
    key: 'currentPage',
    label: t('userOrders.summary.currentPage'),
    value: orders.value.length,
    hint: t('userOrders.summary.currentPageHint'),
  },
  {
    key: 'completed',
    label: t('userOrders.summary.completedOnPage'),
    value: orders.value.filter((order) => order.status === 'COMPLETED').length,
    hint: t('userOrders.summary.completedOnPageHint'),
  },
])

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

async function fetchOrders() {
  loading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentFilter.value || undefined,
    })
    orders.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) { pagination.page = page; fetchOrders() }
function handlePageSizeChange(size: number) { pagination.page_size = size; pagination.page = 1; fetchOrders() }

function handleCancel(orderId: number) { cancelTargetId.value = orderId }

async function confirmCancel() {
  if (!cancelTargetId.value) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess(t('common.success'))
    cancelTargetId.value = null
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) { refundTarget.value = order; refundReason.value = '' }

function openOrderTicket(order: PaymentOrder) {
  router.push({
    path: '/tickets',
    query: {
      new: '1',
      context_type: 'order',
      context_id: String(order.id),
      subject: `${t('payment.orders.orderId')} #${order.id}`
    }
  })
}

async function confirmRefund() {
  if (!refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess(t('common.success'))
    refundTarget.value = null
    refundReason.value = ''
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder): boolean {
  if (order.order_type !== 'balance') return false
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

function localeCode(): string | undefined {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
}

function formatRefundTargetAmount(order: PaymentOrder): string {
  return shouldShowCreditedBalance(order)
    ? formatBalanceCreditAmount(order.amount)
    : formatOrderPaymentAmount(order, order.amount, localeCode())
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => { fetchOrders(); loadRefundEligibility() })
</script>
