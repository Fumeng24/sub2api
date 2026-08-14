<template>
  <DataTable :columns="columns" :data="orders" :loading="loading">
    <template #cell-id="{ value }">
      <span class="font-mono text-sm">#{{ value }}</span>
    </template>
    <template #cell-out_trade_no="{ value }">
      <span class="text-sm text-[var(--apple-text)]">{{ value || t('common.notAvailable') }}</span>
    </template>
    <template v-if="showUser" #cell-user_email="{ value, row }">
      <div class="text-sm">
        <span class="text-[var(--apple-text)]">{{ value || row.user_name || '#' + row.user_id }}</span>
        <span v-if="row.user_notes" class="ml-1 text-xs text-[var(--apple-muted-2)]">({{ row.user_notes }})</span>
      </div>
    </template>
    <template v-if="showSource" #cell-record_source="{ value, row }">
      <span class="inline-flex items-center rounded-full border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2 py-0.5 text-xs font-medium text-[var(--apple-muted)]">
        {{ sourceLabel(row, value) }}
      </span>
    </template>
    <template v-if="showBusinessCategory" #cell-business_category="{ row }">
      <span class="inline-flex items-center rounded-full border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2 py-0.5 text-xs font-medium text-[var(--apple-muted)]">
        {{ businessCategoryLabel(row) }}
      </span>
    </template>
    <template v-if="showInvoiceability" #cell-invoiceability="{ row }">
      <span
        :class="[
          'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium',
          isOrderInvoiceable(row)
            ? 'border-[color:color-mix(in_srgb,var(--apple-success)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_9%,var(--apple-surface))] text-[var(--apple-success)]'
            : 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]'
        ]"
        :title="invoiceabilityTitle(row)"
      >
        {{ isOrderInvoiceable(row) ? t('invoice.page.orders.invoiceability.available') : t('invoice.page.orders.invoiceability.unavailable') }}
      </span>
    </template>
    <template #cell-pay_amount="{ value, row }">
      <div class="text-sm">
        <span class="font-medium text-[var(--apple-text)]">{{ formatOrderAmount(row, value) }}</span>
        <span v-if="row.fee_rate > 0" class="ml-1 text-xs text-[var(--apple-muted-2)]" :title="t('payment.orders.fee') + ': ' + row.fee_rate + '%'">
          ({{ t('payment.orders.fee') }} {{ row.fee_rate }}%)
        </span>
        <div v-if="shouldShowCreditedBalance(row)" class="text-xs text-[var(--apple-muted)]">
          {{ t('payment.orders.creditedBalance') }}: {{ formatBalanceCreditAmount(row.amount, localeCode()) }}
        </div>
      </div>
    </template>
    <template #cell-payment_type="{ value }">
      <span class="text-sm text-[var(--apple-muted)]">{{ t('payment.methods.' + value, value) }}</span>
    </template>
    <template #cell-status="{ value }">
      <OrderStatusBadge :status="value" />
    </template>
    <template #cell-created_at="{ value }">
      <span class="text-xs text-[var(--apple-muted)]">{{ formatDate(value) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <slot name="actions" :row="row" />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaymentOrder } from '@/types/payment'
import type { Column } from '@/components/common/types'
import DataTable from '@/custom/common/WegooDataTable.vue'
import OrderStatusBadge from '@/custom/payment/WegooOrderStatusBadge.vue'
import {
  formatBalanceCreditAmount,
  formatOrderPaymentAmount,
  shouldShowCreditedBalance,
} from '@/custom/payment/orderAmounts'
import {
  businessCategoryI18nKey,
  invoiceUnavailableReasonKey,
  isOrderInvoiceable,
  recordSourceI18nKey,
} from '@/custom/utils/paymentRecordSemantics'

const i18n = useI18n()
const { t } = i18n

const props = defineProps<{
  orders: PaymentOrder[]
  loading: boolean
  showUser?: boolean
  showSource?: boolean
  showBusinessCategory?: boolean
  showInvoiceability?: boolean
}>()

function formatDate(dateStr: string) { return new Date(dateStr).toLocaleString() }

function localeCode(): string | undefined {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
}

function formatOrderAmount(order: PaymentOrder, amount: number): string {
  if (order.record_source && order.record_source !== 'payment_order') {
    return formatBalanceCreditAmount(order.amount, localeCode())
  }
  return formatOrderPaymentAmount(order, amount, localeCode())
}

function sourceLabel(order: PaymentOrder, source: string): string {
  if (order.record_source_label) return order.record_source_label
  return t(recordSourceI18nKey({ record_source: source as PaymentOrder['record_source'] }), source)
}

function businessCategoryLabel(order: PaymentOrder): string {
  const key = businessCategoryI18nKey(order)
  const label = t(key)
  return label === key ? (order.business_category || t('payment.businessCategories.uncategorized')) : label
}

function invoiceabilityTitle(order: PaymentOrder): string {
  const reason = invoiceUnavailableReasonKey(order)
  return reason ? t(`invoice.page.reasons.${reason}`) : t('invoice.page.orders.invoiceability.available')
}

const columns = computed((): Column[] => {
  const cols: Column[] = [
    { key: 'id', label: t('payment.orders.orderId') },
    { key: 'out_trade_no', label: t('payment.orders.orderNo') },
  ]
  if (props.showUser) {
    cols.push({ key: 'user_email', label: t('payment.admin.colUser') })
  }
  if (props.showSource) {
    cols.push({ key: 'record_source', label: t('payment.admin.recordSource') })
  }
  if (props.showBusinessCategory) {
    cols.push({ key: 'business_category', label: t('payment.businessCategory') })
  }
  if (props.showInvoiceability) {
    cols.push({ key: 'invoiceability', label: t('invoice.page.orders.columns.invoiceability') })
  }
  cols.push(
    { key: 'pay_amount', label: t('payment.orders.payAmount') },
    { key: 'payment_type', label: t('payment.orders.paymentMethod') },
    { key: 'status', label: t('payment.orders.status') },
    { key: 'created_at', label: t('payment.orders.createdAt') },
    { key: 'actions', label: t('common.actions') },
  )
  return cols
})
</script>
