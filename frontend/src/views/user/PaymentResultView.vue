<template>
  <div class="flex min-h-screen items-center justify-center bg-[var(--apple-bg)] px-4">
    <div class="w-full max-w-lg space-y-5">
      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
      </div>
      <template v-else>
        <!-- Status Icon -->
        <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6 text-center shadow-sm">
          <div v-if="isSuccess"
            class="mx-auto flex h-16 w-16 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)]">
            <Icon name="check" size="xl" :stroke-width="2" class="text-[var(--apple-success)]" />
          </div>
          <div v-else-if="isPending"
            class="mx-auto flex h-16 w-16 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)]">
            <div class="h-10 w-10 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-warning)]"></div>
          </div>
          <div v-else
            class="mx-auto flex h-16 w-16 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)]">
            <Icon name="x" size="xl" :stroke-width="2" class="text-[var(--apple-danger)]" />
          </div>
          <h2 class="mt-4 text-2xl font-semibold text-[var(--apple-text)]">
            {{ statusTitle }}
          </h2>
          <p class="mx-auto mt-2 max-w-sm text-sm leading-6 text-[var(--apple-muted)]">
            {{ statusHint }}
          </p>
          <p v-if="order?.id" class="mt-3 text-xs text-[var(--apple-muted-2)]">
            {{ t('payment.orders.orderId') }} #{{ order.id }}
          </p>
        </div>
        <section class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm">
          <p class="text-sm font-semibold text-[var(--apple-text)]">
            {{ t('payment.result.assuranceTitle') }}
          </p>
          <div class="mt-3 space-y-2">
            <div
              v-for="item in resultAssuranceItems"
              :key="item"
              class="flex items-start gap-2 text-sm leading-6 text-[var(--apple-muted)]"
            >
              <Icon name="checkCircle" size="sm" class="mt-0.5 shrink-0 text-[var(--apple-success)]" />
              <span>{{ item }}</span>
            </div>
          </div>
        </section>
        <!-- Order Info -->
        <div v-if="order" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
          <p class="mb-3 text-xs font-medium text-[var(--apple-muted-2)]">
            {{ t('payment.result.orderSnapshot') }}
          </p>
          <div class="space-y-3 text-sm">
            <div class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.orderId') }}</span>
              <span class="font-medium text-[var(--apple-text)]">#{{ order.id }}</span>
            </div>
            <div v-if="order.out_trade_no" class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.orderNo') }}</span>
              <span class="max-w-[220px] truncate font-medium text-[var(--apple-text)]" :title="order.out_trade_no">{{ order.out_trade_no }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.baseAmount') }}</span>
              <span class="font-medium text-[var(--apple-text)]">{{ formatGatewayAmount(baseAmount) }}</span>
            </div>
            <div v-if="order.fee_rate > 0" class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.fee') }} ({{ order.fee_rate }}%)</span>
              <span class="font-medium text-[var(--apple-text)]">{{ formatGatewayAmount(feeAmount) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.payAmount') }}</span>
              <span class="font-semibold text-[var(--apple-blue)]">{{ formatGatewayAmount(order.pay_amount) }}</span>
            </div>
            <div v-if="shouldShowCreditedBalance(order)" class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.creditedBalance') }}</span>
              <span class="font-medium text-[var(--apple-text)]">{{ formatBalanceCreditAmount(order.amount) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.paymentMethod') }}</span>
              <span class="font-medium text-[var(--apple-text)]">{{ t(paymentMethodI18nKey(order.payment_type), normalizedOrderPaymentType(order.payment_type)) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.status') }}</span>
              <OrderStatusBadge :status="order.status" />
            </div>
          </div>
        </div>
        <!-- EasyPay return info (when no order loaded) -->
        <div v-else-if="returnInfo" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
          <p class="mb-3 text-xs font-medium text-[var(--apple-muted-2)]">
            {{ t('payment.result.returnSnapshot') }}
          </p>
          <div class="space-y-3 text-sm">
            <div v-if="returnInfo.outTradeNo" class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.orderId') }}</span>
              <span class="max-w-[220px] truncate font-medium text-[var(--apple-text)]" :title="returnInfo.outTradeNo">{{ returnInfo.outTradeNo }}</span>
            </div>
            <div v-if="returnInfo.money" class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.payAmount') }}</span>
              <span class="font-medium text-[var(--apple-text)]">{{ formatGatewayAmount(Number(returnInfo.money) || 0) }}</span>
            </div>
            <div v-if="returnInfo.type" class="flex justify-between">
              <span class="text-[var(--apple-muted)]">{{ t('payment.orders.paymentMethod') }}</span>
              <span class="font-medium text-[var(--apple-text)]">{{ t(paymentMethodI18nKey(returnInfo.type), normalizedOrderPaymentType(returnInfo.type)) }}</span>
            </div>
          </div>
        </div>
        <!-- Actions -->
        <div class="flex gap-3">
          <button class="btn btn-secondary flex-1" @click="router.push('/purchase')">{{ t('payment.result.backToRecharge') }}</button>
          <button class="btn btn-primary flex-1" @click="router.push('/orders')">{{ t('payment.result.viewOrders') }}</button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  clearPaymentRecoverySnapshot,
  readPaymentRecoverySnapshot,
} from '@/components/payment/paymentFlow'
import { usePaymentStore } from '@/stores/payment'
import { paymentAPI } from '@/api/payment'
import { authAPI } from '@/api/auth'
import type { PaymentOrder } from '@/types/payment'
import { formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import { formatBalanceCreditAmount, shouldShowCreditedBalance } from '@/components/payment/orderAmounts'
import { setSettlementCnyPerCredit } from '@/composables/useSettlementCurrency'
import { normalizePaymentMethodForDisplay, paymentMethodI18nKey } from './paymentUx'
import Icon from '@/components/icons/Icon.vue'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const paymentStore = usePaymentStore()

const order = ref<PaymentOrder | null>(null)
const loading = ref(true)
const currency = ref('CNY')

interface ReturnInfo {
  outTradeNo: string
  money: string
  type: string
  tradeStatus: string
}
const returnInfo = ref<ReturnInfo | null>(null)
const resultAssuranceItems = computed(() => [
  t('payment.result.officialModels'),
  t('payment.result.privacy'),
  t('payment.result.refundProtection'),
])

const SUCCESS_STATUSES = new Set(['COMPLETED', 'PAID', 'RECHARGING'])
const PENDING_STATUSES = new Set(['PENDING', 'CREATED', 'WAITING', 'PROCESSING'])
const STATUS_REFRESH_INTERVAL_MS = 2000
const STATUS_REFRESH_MAX_ATTEMPTS = 15

let statusRefreshTimer: ReturnType<typeof setTimeout> | null = null
const refreshAttempts = ref(0)

/** 充值金额 = pay_amount / (1 + fee_rate/100)，fee_rate=0 时等于 pay_amount */
const baseAmount = computed(() => {
  if (!order.value) return 0
  const feeRate = Number(order.value.fee_rate) || 0
  if (feeRate <= 0) return order.value.pay_amount ?? 0
  return Math.round((order.value.pay_amount / (1 + feeRate / 100)) * 100) / 100
})

/** 手续费 = pay_amount - baseAmount */
const feeAmount = computed(() => {
  if (!order.value) return 0
  const feeRate = Number(order.value.fee_rate) || 0
  if (feeRate <= 0) return 0
  return Math.round((order.value.pay_amount - baseAmount.value) * 100) / 100
})

const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

const isSuccess = computed(() => {
  return isSuccessStatus(order.value?.status)
})

const isPending = computed(() => {
  return isPendingStatus(order.value?.status)
})

const statusTitle = computed(() => {
  if (isSuccess.value) {
    return t('payment.result.success')
  }
  if (isPending.value) {
    return t('payment.result.processing')
  }
  return t('payment.result.failed')
})

const statusHint = computed(() => {
  if (isSuccess.value) {
    return t('payment.result.successHint')
  }
  if (isPending.value) {
    return t('payment.result.processingHint')
  }
  return t('payment.result.failedHint')
})

function normalizedOrderPaymentType(paymentType: string): string {
  return normalizePaymentMethodForDisplay(paymentType) || paymentType
}

function formatGatewayAmount(value: number): string {
  return formatPaymentAmount(value, currency.value, localeCode.value)
}

function setResolvedOrder(nextOrder: PaymentOrder | null): void {
  order.value = nextOrder
  if (nextOrder?.currency) {
    currency.value = normalizePaymentCurrency(nextOrder.currency)
  }
}

function normalizeOrderStatus(status: string | null | undefined): string {
  return String(status || '').trim().toUpperCase()
}

function isSuccessStatus(status: string | null | undefined): boolean {
  return SUCCESS_STATUSES.has(normalizeOrderStatus(status))
}

function isPendingStatus(status: string | null | undefined): boolean {
  return PENDING_STATUSES.has(normalizeOrderStatus(status))
}

function readRouteQueryString(key: string): string {
  const value = route.query[key]
  if (Array.isArray(value)) {
    return typeof value[0] === 'string' ? value[0] : ''
  }
  return typeof value === 'string' ? value : ''
}

function restoreRecoverySnapshot(context: {
  resumeToken: string
  routeOrderId: number
  routeOutTradeNo: string
}) {
  if (typeof window === 'undefined') {
    return null
  }

  const rawSnapshot = window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)
  if (!rawSnapshot) {
    return null
  }

  if (context.resumeToken) {
    return readPaymentRecoverySnapshot(rawSnapshot, {
      resumeToken: context.resumeToken,
    })
  }

  if (!context.routeOrderId && !context.routeOutTradeNo) {
    return null
  }

  const restored = readPaymentRecoverySnapshot(rawSnapshot)
  if (!restored) {
    return null
  }

  if (context.routeOrderId > 0 && restored.orderId !== context.routeOrderId) {
    return null
  }

  if (context.routeOutTradeNo && restored.outTradeNo !== context.routeOutTradeNo) {
    return null
  }

  return restored
}

async function resolveOrderFromResumeToken(resumeToken: string): Promise<PaymentOrder | null> {
  try {
    const result = await paymentAPI.resolveOrderPublicByResumeToken(resumeToken)
    return result.data
  } catch (_err: unknown) {
    return null
  }
}

async function resolveOrderFromOutTradeNo(outTradeNo: string): Promise<PaymentOrder | null> {
  try {
    const result = await paymentAPI.verifyOrder(outTradeNo)
    return result.data
  } catch (_err: unknown) {
    try {
      const result = await paymentAPI.verifyOrderPublic(outTradeNo)
      return result.data
    } catch (_innerErr: unknown) {
      return null
    }
  }
}

function clearStatusRefreshTimer(): void {
  if (statusRefreshTimer !== null) {
    clearTimeout(statusRefreshTimer)
    statusRefreshTimer = null
  }
}

function clearRecoverySnapshot(): void {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function clearRecoverySnapshotForTerminalStatus(status: string | null | undefined): void {
  if (!status) return
  if (!isPendingStatus(status)) {
    clearRecoverySnapshot()
  }
}

function scheduleStatusRefresh(refreshOrder: (() => Promise<PaymentOrder | null>) | null): void {
  clearStatusRefreshTimer()
  if (!refreshOrder || !isPending.value || refreshAttempts.value >= STATUS_REFRESH_MAX_ATTEMPTS) {
    return
  }

  statusRefreshTimer = setTimeout(async () => {
    refreshAttempts.value += 1
    const refreshedOrder = await refreshOrder()
    if (refreshedOrder) {
      setResolvedOrder(refreshedOrder)
      clearRecoverySnapshotForTerminalStatus(refreshedOrder.status)
    }

    if (isPendingStatus(order.value?.status)) {
      scheduleStatusRefresh(refreshOrder)
    }
  }, STATUS_REFRESH_INTERVAL_MS)
}

onMounted(async () => {
  authAPI.getPublicSettings()
    .then((settings) => {
      setSettlementCnyPerCredit(settings?.payment_balance_recharge_multiplier)
    })
    .catch(() => {})

  const resumeToken = readRouteQueryString('resume_token')
  const routeOrderId = Number(readRouteQueryString('order_id')) || 0
  let outTradeNo = readRouteQueryString('out_trade_no')
  let orderId = 0
  let resumeTokenLookupFailed = false

  const restored = restoreRecoverySnapshot({
    resumeToken,
    routeOrderId,
    routeOutTradeNo: outTradeNo,
  })
  if (restored?.orderId) {
    orderId = restored.orderId
  }
  if (restored?.currency) {
    currency.value = normalizePaymentCurrency(restored.currency)
  }
  if (!outTradeNo && restored?.outTradeNo) {
    outTradeNo = restored.outTradeNo
  }

  if (resumeToken) {
    const resolvedOrder = await resolveOrderFromResumeToken(resumeToken)
    if (resolvedOrder) {
      setResolvedOrder(resolvedOrder)
      if (!orderId) {
        orderId = resolvedOrder.id
      }
    } else if (routeOrderId > 0) {
      resumeTokenLookupFailed = true
      orderId = routeOrderId
    } else {
      resumeTokenLookupFailed = true
    }
  } else if (routeOrderId > 0) {
    orderId = routeOrderId
  }

  const hasLegacyFallbackContext = readRouteQueryString('trade_status').trim() !== ''
  const shouldUsePublicOutTradeNo = outTradeNo !== '' && (hasLegacyFallbackContext || routeOrderId > 0 || orderId > 0)

  if (!order.value && orderId && (!resumeToken || routeOrderId > 0)) {
    try {
      setResolvedOrder(await paymentStore.pollOrderStatus(orderId))
    } catch (_err: unknown) {
      // Order lookup failed, will try legacy fallback below when possible.
    }
  }

  if (!order.value && shouldUsePublicOutTradeNo && (!resumeToken || resumeTokenLookupFailed)) {
    const legacyOrder = await resolveOrderFromOutTradeNo(outTradeNo)
    if (legacyOrder) {
      setResolvedOrder(legacyOrder)
      if (!orderId) {
        orderId = legacyOrder.id
      }
    }
  }

  if (!order.value && !orderId && outTradeNo && hasLegacyFallbackContext) {
    returnInfo.value = {
      outTradeNo,
      money: String(route.query.money || ''),
      type: String(route.query.type || ''),
      tradeStatus: String(route.query.trade_status || ''),
    }
  }

  const refreshOrder = async (): Promise<PaymentOrder | null> => {
    if (resumeToken) {
      const resolvedOrder = await resolveOrderFromResumeToken(resumeToken)
      if (resolvedOrder) {
        return resolvedOrder
      }
    }

    if (orderId) {
      try {
        return await paymentStore.pollOrderStatus(orderId)
      } catch (_err: unknown) {
        // Fall through to legacy public verification when order polling is unavailable.
      }
    }

    if (shouldUsePublicOutTradeNo) {
      return await resolveOrderFromOutTradeNo(outTradeNo)
    }

    return null
  }

  if (isPendingStatus(order.value?.status)) {
    scheduleStatusRefresh(refreshOrder)
  } else if (order.value) {
    clearRecoverySnapshotForTerminalStatus(order.value.status)
  } else if (returnInfo.value) {
    clearRecoverySnapshot()
  }
  loading.value = false
})

onBeforeUnmount(() => {
  clearStatusRefreshTimer()
})
</script>
