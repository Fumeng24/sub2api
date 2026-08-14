<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <UserPageHero
        :kicker="t('userOrders.gateway.kicker')"
        :title="t('nav.myOrders')"
      >
        <template #actions>
          <button v-if="canAccessPurchase" class="btn btn-primary w-full justify-center sm:w-auto" @click="router.push('/purchase')">
            <Icon name="plus" size="sm" />
            {{ t('userOrders.newOrder') }}
          </button>
        </template>

        <template #below>
          <UserSummaryStats
            class="mt-5"
            :items="orderSummaryItems"
            grid-class="grid-cols-1 sm:grid-cols-3"
            :aria-label="t('userOrders.summary.label')"
          />
        </template>
      </UserPageHero>

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
      <OrderTable
        :orders="orders"
        :loading="loading"
        show-source
        show-business-category
        show-invoiceability
      >
        <template #actions="{ row }">
          <div class="flex flex-wrap items-center gap-2">
            <button
              v-if="isPaymentOrder(row) && row.status === 'PENDING'"
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
import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { paymentAPI } from '@/custom/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Pagination from '@/custom/common/WegooPagination.vue'
import BaseDialog from '@/custom/common/WegooBaseDialog.vue'
import Select from '@/custom/common/WegooSelect.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/custom/payment/WegooOrderTable.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import UserSummaryStats from '@/custom/user/UserSummaryStats.vue'
import { formatBalanceCreditAmount, formatOrderPaymentAmount, shouldShowCreditedBalance } from '@/custom/payment/orderAmounts'

const i18n = useI18n()
const { t } = i18n
const router = useRouter()
const appStore = useAppStore()
const paymentCheckoutStore = usePaymentCheckoutStore()
const canAccessPurchase = computed(() => paymentCheckoutStore.canAccessPurchase)

const loading = ref(false)
const actionLoading = ref(false)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const orderSummaryItems = computed(() => [
  {
    label: t('userOrders.summary.total'),
    value: pagination.total,
  },
  {
    label: t('userOrders.summary.currentPage'),
    value: orders.value.length,
  },
  {
    label: t('userOrders.summary.completedOnPage'),
    value: orders.value.filter((order) => order.status === 'COMPLETED').length,
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
  if (!isPaymentOrder(order)) return false
  if (order.order_type !== 'balance') return false
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

function isPaymentOrder(order: PaymentOrder): boolean {
  return !order.record_source || order.record_source === 'payment_order'
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
    ? formatBalanceCreditAmount(order.amount, localeCode())
    : formatOrderPaymentAmount(order, order.amount, localeCode())
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => {
  paymentCheckoutStore.fetchCheckoutInfo().catch(() => {})
  fetchOrders()
  loadRefundEligibility()
})
</script>
