<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-5">
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
      </div>
      <template v-else>
        <header class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm sm:p-6">
          <div class="grid gap-5 lg:grid-cols-[1fr_auto] lg:items-start">
            <div class="min-w-0">
              <h1 class="text-2xl font-semibold tracking-normal text-[var(--apple-text)] sm:text-3xl">
                {{ t('nav.buySubscription') }}
              </h1>
              <p class="mt-2 max-w-3xl text-sm leading-6 text-[var(--apple-muted)] sm:text-base">
                {{ t('purchase.description') }}
              </p>
              <div class="mt-4 grid gap-2 sm:flex sm:flex-wrap">
                <span
                  v-for="item in paymentTrustItems"
                  :key="item"
                  class="inline-flex min-w-0 items-center gap-1.5 rounded-md bg-[var(--apple-surface-elevated)] px-2.5 py-1 text-xs font-medium text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]"
                >
                  <Icon name="checkCircle" size="xs" class="text-[var(--apple-success)]" />
                  <span class="min-w-0 truncate">{{ item }}</span>
                </span>
              </div>
              <p class="mt-2 text-xs leading-5 text-[var(--apple-muted-2)]">
                {{ t('payment.trust.caption') }}
              </p>
            </div>
            <div class="border-t border-[color:var(--apple-border-soft)] pt-4 text-sm leading-6 text-[var(--apple-muted)] lg:w-[300px] lg:border-l lg:border-t-0 lg:pl-5 lg:pt-0">
              <p v-if="!checkout.balance_disabled" class="mb-3 text-xs text-[var(--apple-muted-2)]">
                {{ t('payment.currentBalance') }}
                <span class="ml-1 font-semibold text-[var(--apple-text)]">{{ formatCreditedBalance(user?.balance || 0) }}</span>
              </p>
              <p class="font-medium text-[var(--apple-text)]">
                {{ t('payment.support.title') }}
              </p>
              <div class="mt-1 flex flex-wrap items-center gap-2">
                <p>{{ t('payment.support.detail') }}</p>
                <button
                  type="button"
                  class="font-semibold text-[var(--apple-blue)] transition-colors hover:text-[var(--apple-blue-hover)]"
                  @click="router.push('/tickets')"
                >
                  {{ t('payment.support.action') }}
                </button>
              </div>
            </div>
          </div>
        </header>
        <!-- Tab Switcher (hide during payment and subscription confirm) -->
        <div v-if="tabs.length > 1 && paymentPhase === 'select' && !selectedPlan" class="flex space-x-1 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-1">
          <button v-for="tab in tabs" :key="tab.key"
            class="flex-1 rounded-md px-4 py-2.5 text-sm font-medium transition-all"
            :class="activeTab === tab.key ? 'bg-[var(--apple-surface)] text-[var(--apple-text)] shadow-sm' : 'text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'"
            @click="activeTab = tab.key">{{ tab.label }}</button>
        </div>
        <div v-if="errorMessage && paymentPhase === 'select'" class="rounded-lg border border-[color:color-mix(in_srgb,var(--apple-danger)_24%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_8%,var(--apple-surface))] px-4 py-3" role="alert">
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="md" :stroke-width="2" class="mt-0.5 shrink-0 text-[var(--apple-danger)]" />
            <div class="min-w-0 flex-1">
              <p class="text-sm font-semibold text-[var(--apple-danger)]">{{ errorMessage }}</p>
              <p class="mt-1 text-xs leading-relaxed text-[var(--apple-muted)]">{{ orderErrorHint }}</p>
            </div>
          </div>
        </div>
        <!-- Payment in progress (shared by recharge and subscription) -->
        <template v-if="paymentPhase === 'paying'">
          <PaymentStatusPanel
            :order-id="paymentState.orderId"
            :qr-code="paymentState.qrCode"
            :expires-at="paymentState.expiresAt"
            :payment-type="paymentState.paymentType"
            :pay-url="paymentState.payUrl"
            :order-type="paymentState.orderType"
            :currency="paymentState.currency || selectedCurrency"
            @done="onPaymentDone"
            @success="onPaymentSuccess"
            @settled="onPaymentSettled"
          />
        </template>
        <!-- Tab content (select phase) -->
        <template v-else>
          <!-- Top-up Tab -->
          <template v-if="activeTab === 'recharge'">
            <section class="overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm">
              <div class="flex flex-col gap-2 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div class="min-w-0">
                  <p class="text-xs font-medium text-[var(--apple-muted)]">{{ t('payment.rechargeAccount') }}</p>
                  <p class="mt-1 text-base font-semibold text-[var(--apple-text)]">{{ user?.username || '' }}</p>
                </div>
                <p class="text-sm font-medium text-[var(--apple-success)]">
                  {{ t('payment.currentBalance') }}: {{ formatCreditedBalance(user?.balance || 0) }}
                </p>
              </div>
              <div v-if="enabledMethods.length === 0" class="border-t border-[color:var(--apple-border-soft)] py-14 text-center">
                <p class="text-[var(--apple-muted)]">{{ t('payment.notAvailable') }}</p>
              </div>
              <template v-else>
                <div class="border-t border-[color:var(--apple-border-soft)] p-5">
                  <AmountInput
                    v-model="amount"
                    :amounts="quickRechargeAmounts"
                    :amount-label="formatQuickRechargeAmountLabel"
                    :amount-description="formatQuickRechargeAmountDescription"
                    :disabled-reason="quickRechargeDisabledReason"
                    :min="globalMinAmount"
                    :max="globalMaxAmount"
                    :prefix="selectedPaymentInputPrefix"
                  />
                  <p v-if="amountError" class="mt-2 text-xs text-[var(--apple-warning)]">{{ amountError }}</p>
                </div>
                <div v-if="enabledMethods.length >= 1" class="border-t border-[color:var(--apple-border-soft)] p-5">
                  <PaymentMethodSelector
                    :methods="methodOptions"
                    :selected="selectedMethod"
                    @select="selectedMethod = $event"
                  />
                </div>
                <div v-if="validAmount > 0" class="border-t border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-5">
                  <div class="space-y-2 text-sm">
                    <p class="text-xs font-medium text-[var(--apple-muted-2)]">
                      {{ t('payment.orderPreview') }}
                    </p>
                    <div class="flex justify-between gap-4">
                      <span class="text-[var(--apple-muted)]">{{ t('payment.paymentAmount') }}</span>
                      <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(validAmount) }}</span>
                    </div>
                    <div v-if="feeRate > 0" class="flex justify-between gap-4">
                      <span class="text-[var(--apple-muted)]">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                      <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(feeAmount) }}</span>
                    </div>
                    <div v-if="feeRate > 0" class="flex justify-between gap-4 border-t border-[color:var(--apple-border-soft)] pt-2">
                      <span class="font-medium text-[var(--apple-text)]">{{ t('payment.actualPay') }}</span>
                      <span class="text-lg font-semibold text-[var(--apple-blue)]">{{ formatSelectedPaymentAmount(totalAmount) }}</span>
                    </div>
                    <div class="flex justify-between gap-4" :class="{ 'border-t border-[color:var(--apple-border-soft)] pt-2': feeRate <= 0 }">
                      <span class="text-[var(--apple-muted)]">{{ t('payment.creditedBalance') }}</span>
                      <span class="text-[var(--apple-text)]">{{ formatBalanceCreditAmount(creditedAmount) }}</span>
                    </div>
                    <p v-if="showBalanceRechargeRate" class="border-t border-[color:var(--apple-border-soft)] pt-2 text-xs leading-5 text-[var(--apple-muted)]">
                      {{ t('payment.rechargeRatePreview', { cny: balanceRechargeCnyPerCredit.toFixed(2) }) }}
                    </p>
                  </div>
                </div>
                <div class="border-t border-[color:var(--apple-border-soft)] p-5">
                  <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmit || submitting" @click="handleSubmitRecharge">
                    <span v-if="submitting" class="flex items-center justify-center gap-2">
                      <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                      {{ t('common.processing') }}
                    </span>
                    <span v-else>{{ t('payment.confirmPayment') }} {{ formatSelectedPaymentAmount(totalAmount) }}</span>
                  </button>
                </div>
              </template>
            </section>
          </template>
          <!-- Subscribe Tab -->
          <template v-else-if="activeTab === 'subscription'">
            <!-- Subscription confirm (inline, replaces plan list) -->
            <template v-if="selectedPlan">
              <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
                <!-- Header: platform badge + plan name -->
                <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <p class="text-xs font-medium text-[var(--apple-muted-2)]">
                      {{ t('payment.selectedPlan') }}
                    </p>
                    <div class="mt-2 flex flex-wrap items-center gap-2">
                      <span class="rounded-md border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2 py-0.5 text-xs font-medium text-[var(--apple-muted)]">
                        {{ platformLabel(selectedPlan.group_platform || '') }}
                      </span>
                      <h3 class="text-lg font-semibold text-[var(--apple-text)]">{{ selectedPlan.name }}</h3>
                    </div>
                  </div>
                  <button class="btn btn-secondary btn-sm w-full shrink-0 sm:w-auto" @click="selectedPlan = null">{{ t('payment.changePlan') }}</button>
                </div>
                <!-- Price -->
                <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                  <span v-if="selectedPlan.original_price" class="text-sm text-[var(--apple-muted-2)] line-through">
                    {{ formatSelectedPaymentAmount(selectedPlan.original_price) }}
                  </span>
                  <span class="text-3xl font-semibold text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(selectedPlan.price) }}</span>
                  <span class="text-sm text-[var(--apple-muted)]">/ {{ planValiditySuffix }}</span>
                </div>
                <!-- Description -->
                <p v-if="selectedPlan.description" class="mt-2 text-sm leading-relaxed text-[var(--apple-muted)]">
                  {{ selectedPlan.description }}
                </p>
                <!-- Rate + Limits grid -->
                <div class="mt-4 grid grid-cols-1 gap-3 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-3 sm:grid-cols-2">
                  <div class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.rate') }}</span>
                    <div class="flex items-baseline">
                      <span class="text-lg font-semibold text-[var(--apple-text)]">×{{ selectedPlan.rate_multiplier ?? 1 }}</span>
                    </div>
                  </div>
                  <div v-if="selectedPlan.daily_limit_usd != null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.dailyLimit') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ formatSettlementAmount(selectedPlan.daily_limit_usd, 2) }}</div>
                  </div>
                  <div v-if="selectedPlan.weekly_limit_usd != null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.weeklyLimit') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ formatSettlementAmount(selectedPlan.weekly_limit_usd, 2) }}</div>
                  </div>
                  <div v-if="selectedPlan.monthly_limit_usd != null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.monthlyLimit') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ formatSettlementAmount(selectedPlan.monthly_limit_usd, 2) }}</div>
                  </div>
                  <div v-if="selectedPlan.daily_limit_usd == null && selectedPlan.weekly_limit_usd == null && selectedPlan.monthly_limit_usd == null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.quota') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ t('payment.planCard.unlimited') }}</div>
                  </div>
                </div>
              </div>
              <div v-if="enabledMethods.length >= 1" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
                <PaymentMethodSelector
                  :methods="subMethodOptions"
                  :selected="selectedMethod"
                  @select="selectedMethod = $event"
                />
              </div>
              <div v-if="selectedPlan.price > 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
                <div class="space-y-2 text-sm">
                  <p class="text-xs font-medium text-[var(--apple-muted-2)]">
                    {{ t('payment.orderPreview') }}
                  </p>
                  <div class="flex justify-between gap-4">
                    <span class="text-[var(--apple-muted)]">{{ t('payment.amountLabel') }}</span>
                    <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(selectedPlan.price) }}</span>
                  </div>
                  <div v-if="feeRate > 0" class="flex justify-between gap-4">
                    <span class="text-[var(--apple-muted)]">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                    <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(subFeeAmount) }}</span>
                  </div>
                  <div class="flex justify-between gap-4 border-t border-[color:var(--apple-border-soft)] pt-2">
                    <span class="font-medium text-[var(--apple-text)]">{{ t('payment.actualPay') }}</span>
                    <span class="text-lg font-semibold text-[var(--apple-blue)]">{{ formatSelectedPaymentAmount(subTotalAmount) }}</span>
                  </div>
                </div>
              </div>
              <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmitSubscription || submitting" @click="confirmSubscribe">
                <span v-if="submitting" class="flex items-center justify-center gap-2">
                  <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                  {{ t('common.processing') }}
                </span>
                <span v-else>{{ t('payment.confirmPayment') }} {{ formatSelectedPaymentAmount(feeRate > 0 ? subTotalAmount : selectedPlan.price) }}</span>
              </button>
            </template>
            <!-- Plan list -->
            <template v-else>
              <div v-if="checkout.plans.length === 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-6 py-16 text-center shadow-sm">
                <Icon name="gift" size="xl" class="mx-auto mb-3 text-[var(--apple-muted-2)]" />
                <p class="text-[var(--apple-muted)]">{{ t('payment.noPlans') }}</p>
                <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-[var(--apple-muted)]">{{ t('payment.noPlansDesc') }}</p>
              </div>
              <div v-else :class="planGridClass">
                <SubscriptionPlanCard v-for="plan in checkout.plans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlan" />
              </div>
              <!-- Active subscriptions (compact, below plan list) -->
              <div v-if="activeSubscriptions.length > 0">
                <p class="mb-2 text-xs font-medium text-[var(--apple-muted-2)]">{{ t('payment.activeSubscription') }}</p>
                <div class="space-y-2">
                  <div v-for="sub in activeSubscriptions" :key="sub.id"
                    class="flex items-center gap-3 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 py-2 shadow-sm">
                    <div class="h-6 w-1 shrink-0 rounded-full bg-[var(--apple-blue)]" />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <span class="truncate text-xs font-semibold text-[var(--apple-text)]">{{ sub.group?.name || t('payment.groupFallback', { id: sub.group_id }) }}</span>
                        <span class="shrink-0 rounded-full border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-1.5 py-0.5 text-[9px] font-medium text-[var(--apple-muted)]">{{ platformLabel(sub.group?.platform || '') }}</span>
                      </div>
                      <div class="flex flex-wrap gap-x-3 text-[11px] text-[var(--apple-muted-2)]">
                        <span>{{ t('payment.planCard.rate') }}: ×{{ sub.group?.rate_multiplier ?? 1 }}</span>
                        <span v-if="sub.group?.daily_limit_usd == null && sub.group?.weekly_limit_usd == null && sub.group?.monthly_limit_usd == null">{{ t('payment.planCard.quota') }}: {{ t('payment.planCard.unlimited') }}</span>
                        <span v-if="sub.expires_at">{{ t('userSubscriptions.daysRemaining', { days: getDaysRemaining(sub.expires_at) }) }}</span>
                        <span v-else>{{ t('userSubscriptions.noExpiration') }}</span>
                      </div>
                    </div>
                    <span class="badge badge-success shrink-0 text-[10px]">{{ t('userSubscriptions.status.active') }}</span>
                  </div>
                </div>
              </div>
            </template>
          </template>
        </template>
        <div v-if="(checkout.help_text || checkout.help_image_url) && paymentPhase === 'select' && !selectedPlan" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm">
          <div class="flex flex-col items-center gap-3">
            <img v-if="checkout.help_image_url" :src="checkout.help_image_url" alt=""
              class="h-40 max-w-full cursor-pointer rounded-lg object-contain transition-opacity hover:opacity-80"
              @click="previewImage = checkout.help_image_url" />
            <p v-if="checkout.help_text" class="text-center text-sm text-[var(--apple-muted)]">{{ checkout.help_text }}</p>
          </div>
        </div>
      </template>
    </div>
    <!-- Renewal Plan Selection Modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showRenewalModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" @click.self="closeRenewalModal">
          <div class="relative w-full max-w-lg rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6 shadow-sm">
            <!-- Close button -->
            <button class="absolute right-4 top-4 rounded-lg p-1 text-[var(--apple-muted-2)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]" @click="closeRenewalModal">
              <Icon name="x" size="md" :stroke-width="2" />
            </button>
            <h3 class="mb-4 text-lg font-semibold text-[var(--apple-text)]">{{ t('payment.selectPlan') }}</h3>
            <div class="space-y-4">
              <SubscriptionPlanCard v-for="plan in renewalPlans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlanFromModal" />
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
    <!-- Image Preview Overlay -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="previewImage" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/70" @click="previewImage = ''">
          <img :src="previewImage" alt="" class="max-h-[85vh] max-w-[90vw] rounded-lg object-contain" />
        </div>
      </Transition>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePaymentStore } from '@/stores/payment'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAppStore } from '@/stores'
import { setSettlementCnyPerCredit, useSettlementCurrency } from '@/composables/useSettlementCurrency'
import { paymentAPI } from '@/api/payment'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'
import { isMobileDevice } from '@/utils/device'
import type { SubscriptionPlan, CheckoutInfoResponse, CreateOrderResult, OrderType } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import AmountInput from '@/components/payment/AmountInput.vue'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'
import { METHOD_ORDER, getPaymentPopupFeatures } from '@/components/payment/providerConfig'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  buildCreateOrderPayload,
  clearPaymentRecoverySnapshot,
  decidePaymentLaunch,
  getVisibleMethods,
  normalizeVisibleMethod,
  readPaymentRecoverySnapshot,
  type PaymentRecoverySnapshot,
  writePaymentRecoverySnapshot,
} from '@/components/payment/paymentFlow'
import { platformLabel } from '@/utils/platformColors'
import SubscriptionPlanCard from '@/components/payment/SubscriptionPlanCard.vue'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { DEFAULT_PAYMENT_CURRENCY, formatPaymentAmount, normalizePaymentCurrency, paymentAmountPrefix } from '@/components/payment/currency'
import type { PaymentMethodOption } from '@/components/payment/PaymentMethodSelector.vue'
import { buildPaymentErrorToastMessage, describePaymentScenarioError } from './paymentUx'
import { formatBalanceCreditAmount, formatCreditedBalance } from '@/components/payment/orderAmounts'
import { hasWechatResumeQuery, parseWechatResumeRoute, stripWechatResumeQuery } from './paymentWechatResume'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const paymentStore = usePaymentStore()
const subscriptionStore = useSubscriptionStore()
const appStore = useAppStore()
const { formatSettlementAmount } = useSettlementCurrency()

const user = computed(() => authStore.user)
const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)

function getDaysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

const loading = ref(true)
const submitting = ref(false)
const errorMessage = ref('')
const errorHintMessage = ref('')
const orderErrorHint = computed(() => errorHintMessage.value || t('payment.errors.createOrderHint'))
const activeTab = ref<'recharge' | 'subscription'>('recharge')
const amount = ref<number | null>(null)
const selectedMethod = ref('')
const selectedPlan = ref<SubscriptionPlan | null>(null)
const previewImage = ref('')
const paymentTrustItems = computed(() => [
  t('payment.trust.recharge'),
  t('payment.trust.privacy'),
  t('payment.trust.subscription'),
  t('payment.trust.stability'),
  t('payment.trust.orders'),
])

const paymentPhase = ref<'select' | 'paying'>('select')

interface CreateOrderOptions {
  openid?: string
  wechatResumeToken?: string
  paymentType?: string
  isResume?: boolean
  mobileQrFallbackAttempted?: boolean
}

interface WeixinJSBridgeLike {
  invoke(
    action: string,
    payload: Record<string, unknown>,
    callback: (result: Record<string, unknown>) => void,
  ): void
}

function emptyPaymentState(): PaymentRecoverySnapshot {
  return {
    orderId: 0,
    amount: 0,
    qrCode: '',
    expiresAt: '',
    paymentType: '',
    payUrl: '',
    outTradeNo: '',
    clientSecret: '',
    intentId: '',
    currency: '',
    countryCode: '',
    paymentEnv: '',
    payAmount: 0,
    orderType: '',
    paymentMode: '',
    resumeToken: '',
    createdAt: 0,
  }
}

function getWeixinJSBridge(): WeixinJSBridgeLike | undefined {
  return (window as Window & { WeixinJSBridge?: WeixinJSBridgeLike }).WeixinJSBridge
}

function waitForWeixinJSBridge(timeoutMs = 4000): Promise<WeixinJSBridgeLike | null> {
  const existing = getWeixinJSBridge()
  if (existing) return Promise.resolve(existing)

  return new Promise((resolve) => {
    let settled = false
    const finish = (bridge: WeixinJSBridgeLike | null) => {
      if (settled) return
      settled = true
      document.removeEventListener('WeixinJSBridgeReady', handleReady)
      document.removeEventListener('onWeixinJSBridgeReady', handleReady)
      window.clearTimeout(timer)
      resolve(bridge)
    }
    const handleReady = () => finish(getWeixinJSBridge() ?? null)
    const timer = window.setTimeout(() => finish(getWeixinJSBridge() ?? null), timeoutMs)
    document.addEventListener('WeixinJSBridgeReady', handleReady, false)
    document.addEventListener('onWeixinJSBridgeReady', handleReady, false)
  })
}

async function invokeWechatJsapiPayment(payload: Record<string, unknown>): Promise<Record<string, unknown>> {
  const bridge = await waitForWeixinJSBridge()
  if (!bridge) {
    throw new Error('WECHAT_JSAPI_UNAVAILABLE')
  }
  return new Promise((resolve) => {
    bridge.invoke('getBrandWCPayRequest', payload, (result) => resolve(result || {}))
  })
}

const paymentState = ref<PaymentRecoverySnapshot>(emptyPaymentState())

function persistRecoverySnapshot(snapshot: PaymentRecoverySnapshot) {
  if (typeof window === 'undefined' || !snapshot.orderId) return
  writePaymentRecoverySnapshot(window.localStorage, snapshot, PAYMENT_RECOVERY_STORAGE_KEY)
}

function removeRecoverySnapshot() {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function resetPayment() {
  paymentPhase.value = 'select'
  paymentState.value = emptyPaymentState()
  removeRecoverySnapshot()
}

async function redirectToPaymentResult(state: PaymentRecoverySnapshot): Promise<void> {
  const query: Record<string, string | undefined> = {}
  if (state.orderId > 0) {
    query.order_id = String(state.orderId)
  }
  if (state.outTradeNo) {
    query.out_trade_no = state.outTradeNo
  }
  if (state.resumeToken) {
    query.resume_token = state.resumeToken
  }
  await router.push({
    path: '/payment/result',
    query,
  })
}

function buildWechatOAuthAuthorizeUrl(
  authorizeUrl: string,
  context: { paymentType: string; orderType: OrderType; planId?: number; orderAmount: number },
): string {
  const normalizedUrl = authorizeUrl.trim()
  if (!normalizedUrl || typeof window === 'undefined') {
    return normalizedUrl
  }

  try {
    const targetUrl = new URL(normalizedUrl, window.location.origin)
    const redirectPath = targetUrl.searchParams.get('redirect') || '/purchase'
    const redirectUrl = new URL(redirectPath, window.location.origin)
    const paymentType = normalizeVisibleMethod(context.paymentType) || context.paymentType.trim() || 'wxpay'

    redirectUrl.searchParams.set('payment_type', paymentType)
    redirectUrl.searchParams.set('order_type', context.orderType)

    if (context.planId) {
      redirectUrl.searchParams.set('plan_id', String(context.planId))
    } else {
      redirectUrl.searchParams.delete('plan_id')
    }

    if (context.orderAmount > 0) {
      redirectUrl.searchParams.set('amount', String(context.orderAmount))
    } else {
      redirectUrl.searchParams.delete('amount')
    }

    targetUrl.searchParams.set('redirect', `${redirectUrl.pathname}${redirectUrl.search}`)
    return targetUrl.toString()
  } catch {
    return normalizedUrl
  }
}

function onPaymentDone() {
  const wasSubscription = paymentState.value.orderType === 'subscription'
  resetPayment()
  selectedPlan.value = null
  if (wasSubscription) {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function onPaymentSuccess() {
  removeRecoverySnapshot()
  authStore.refreshUser()
  if (paymentState.value.orderType === 'subscription') {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function onPaymentSettled() {
  removeRecoverySnapshot()
}

// All checkout data from single API call
const checkout = ref<CheckoutInfoResponse>({
  methods: {}, global_min: 0, global_max: 0,
  plans: [], balance_disabled: false, balance_recharge_multiplier: 6.8, recharge_fee_rate: 0, help_text: '', help_image_url: '', stripe_publishable_key: '',
})

const tabs = computed(() => {
  const result: { key: 'recharge' | 'subscription'; label: string }[] = []
  if (!checkout.value.balance_disabled) result.push({ key: 'recharge', label: t('payment.tabTopUp') })
  result.push({ key: 'subscription', label: t('payment.tabSubscribe') })
  return result
})

const visibleMethods = computed(() => getVisibleMethods(checkout.value.methods))
const enabledMethods = computed(() => Object.keys(visibleMethods.value))
const validAmount = computed(() => amount.value ?? 0)
const quickRechargeBalanceCredits = [5, 10, 20, 50, 100, 200] as const
const defaultQuickRechargeBalanceCredit = 20
const balanceRechargeCnyPerCredit = computed(() => {
  const multiplier = checkout.value.balance_recharge_multiplier
  return multiplier > 0 ? multiplier : 6.8
})

// Adaptive grid: center single card, 2-col for 2 plans, 3-col for 3+
const planGridClass = computed(() => {
  const n = checkout.value.plans.length
  if (n <= 2) return 'grid grid-cols-1 gap-5 sm:grid-cols-2'
  return 'grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3'
})

// Check if an amount fits a method's [min, max]. 0 = no limit.
function amountFitsMethod(amt: number, methodType: string): boolean {
  if (amt <= 0) return true
  const ml = visibleMethods.value[methodType]
  if (!ml) return false
  if (ml.single_min > 0 && amt < ml.single_min) return false
  if (ml.single_max > 0 && amt > ml.single_max) return false
  return true
}

// Visible methods decide the amount range shown to users.
const globalMinAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_min <= 0)) return 0
  return Math.min(...limits.map(limit => limit.single_min))
})
const globalMaxAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_max <= 0)) return 0
  return Math.max(...limits.map(limit => limit.single_max))
})

// Selected method's limits (for validation and error messages)
const selectedLimit = computed(() => visibleMethods.value[selectedMethod.value])
const selectedCurrency = computed(() => normalizePaymentCurrency(selectedLimit.value?.currency))
const appliesBalanceRechargeRate = computed(() => selectedCurrency.value === DEFAULT_PAYMENT_CURRENCY)
const quickRechargeAmounts = computed(() =>
  quickRechargeBalanceCredits.map((value) => value)
)
const showBalanceRechargeRate = computed(() => appliesBalanceRechargeRate.value && balanceRechargeCnyPerCredit.value !== 1)
const creditedAmount = computed(() => {
  return Math.round(validAmount.value * 100) / 100
})
const selectedPaymentInputPrefix = computed(() => paymentAmountPrefix(selectedCurrency.value))
const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

function formatSelectedPaymentAmount(value: number): string {
  return formatPaymentAmount(value, selectedCurrency.value, localeCode.value)
}

function formatQuickRechargeAmountLabel(value: number): string {
  return formatBalanceCreditAmount(value)
}

function formatQuickRechargeAmountDescription(value: number): string {
  return t('payment.quickAmountPayDescription', {
    amount: formatSelectedPaymentAmount(value),
  })
}

function quickRechargeDisabledReason(value: number): string {
  if (!selectedMethod.value || amountFitsMethod(value, selectedMethod.value)) return ''
  const limit = selectedLimit.value
  if (limit?.single_min && limit.single_min > 0 && value < limit.single_min) {
    return t('payment.quickAmountBelowLimit')
  }
  if (limit?.single_max && limit.single_max > 0 && value > limit.single_max) {
    return t('payment.quickAmountAboveLimit')
  }
  return t('payment.quickAmountUnavailable')
}

function paymentAmountForBalanceCredit(credit: number): number {
  return Math.round(credit * 100) / 100
}

function applyDefaultRechargeAmount() {
  if (amount.value !== null || checkout.value.balance_disabled || activeTab.value !== 'recharge' || paymentPhase.value !== 'select') {
    return
  }
  const preferredAmount = paymentAmountForBalanceCredit(defaultQuickRechargeBalanceCredit)
  if (amountFitsMethod(preferredAmount, selectedMethod.value)) {
    amount.value = preferredAmount
    return
  }
  const fallbackAmount = quickRechargeAmounts.value.find((value) => amountFitsMethod(value, selectedMethod.value))
  if (fallbackAmount) {
    amount.value = fallbackAmount
  }
}

const methodOptions = computed<PaymentMethodOption[]>(() =>
  enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(validAmount.value, type),
    }
  })
)

const feeRate = computed(() => checkout.value?.recharge_fee_rate ?? 0)
const feeAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.ceil(((validAmount.value * feeRate.value) / 100) * 100) / 100
    : 0
)
const totalAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.round((validAmount.value + feeAmount.value) * 100) / 100
    : validAmount.value
)

const amountError = computed(() => {
  if (validAmount.value <= 0) return ''
  // No method can handle this amount
  if (!enabledMethods.value.some((m) => amountFitsMethod(validAmount.value, m))) {
    return t('payment.amountNoMethod')
  }
  // Selected method can't handle this amount (but others can)
  const ml = selectedLimit.value
  if (ml) {
    if (ml.single_min > 0 && validAmount.value < ml.single_min) return t('payment.amountTooLow', { min: formatSelectedPaymentAmount(ml.single_min) })
    if (ml.single_max > 0 && validAmount.value > ml.single_max) return t('payment.amountTooHigh', { max: formatSelectedPaymentAmount(ml.single_max) })
  }
  return ''
})

const canSubmit = computed(() =>
  validAmount.value > 0
    && amountFitsMethod(validAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

// Subscription-specific: method options based on plan price
const subMethodOptions = computed<PaymentMethodOption[]>(() => {
  const planPrice = selectedPlan.value?.price ?? 0
  return enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(planPrice, type),
    }
  })
})

const subFeeAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  if (feeRate.value <= 0 || price <= 0) return 0
  return Math.ceil(((price * feeRate.value) / 100) * 100) / 100
})

const subTotalAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  if (feeRate.value <= 0 || price <= 0) return price
  return Math.round((price + subFeeAmount.value) * 100) / 100
})

const canSubmitSubscription = computed(() =>
  selectedPlan.value !== null
    && amountFitsMethod(selectedPlan.value.price, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

// Auto-switch to first available method when current selection can't handle the amount
watch(() => [validAmount.value, selectedMethod.value] as const, ([amt, method]) => {
  if (amt <= 0 || amountFitsMethod(amt, method)) return
  const available = enabledMethods.value.find((m) => amountFitsMethod(amt, m))
  if (available) selectedMethod.value = available
})

// Payment buttons stay on the product primary color; provider identity is shown by icons.
const paymentButtonClass = computed(() => {
  return 'btn-primary'
})

// Renewal modal state
const showRenewalModal = ref(false)
const renewGroupId = ref<number | null>(null)
const renewalPlans = computed(() => {
  if (renewGroupId.value == null) return []
  return checkout.value.plans.filter(p => p.group_id === renewGroupId.value)
})

const planValiditySuffix = computed(() => {
  if (!selectedPlan.value) return ''
  const u = selectedPlan.value.validity_unit || 'day'
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${selectedPlan.value.validity_days}${t('payment.days')}`
})

function selectPlan(plan: SubscriptionPlan) {
  selectedPlan.value = plan
  errorMessage.value = ''
}

function selectPlanFromModal(plan: SubscriptionPlan) {
  showRenewalModal.value = false
  renewGroupId.value = null
  selectedPlan.value = plan
  errorMessage.value = ''
}

function closeRenewalModal() {
  showRenewalModal.value = false
  renewGroupId.value = null
}

async function handleSubmitRecharge() {
  if (!canSubmit.value || submitting.value) return
  await createOrder(validAmount.value, 'balance')
}

async function confirmSubscribe() {
  if (!selectedPlan.value || submitting.value) return
  await createOrder(selectedPlan.value.price, 'subscription', selectedPlan.value.id)
}

async function createOrder(orderAmount: number, orderType: OrderType, planId?: number, options: CreateOrderOptions = {}) {
  submitting.value = true
  errorMessage.value = ''
  errorHintMessage.value = ''
  const requestType = normalizeVisibleMethod(options.paymentType || selectedMethod.value) || options.paymentType || selectedMethod.value
  try {
    const payload = buildCreateOrderPayload({
      amount: orderAmount,
      paymentType: requestType,
      orderType,
      planId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && normalizeVisibleMethod(requestType) === 'alipay'),
    })
    if (options.openid) {
      payload.openid = options.openid
    }
    if (options.wechatResumeToken) {
      payload.wechat_resume_token = options.wechatResumeToken
    }

    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const openWindow = (url: string) => {
      const win = window.open(url, 'paymentPopup', getPaymentPopupFeatures())
      if (!win || win.closed) {
        window.location.href = url
      }
    }
    const visibleMethod = normalizeVisibleMethod(requestType) || requestType
    // When user clicks the dedicated Stripe button, leave method blank so the
    // landing page renders Stripe's full Payment Element (card/link/alipay/wxpay).
    const stripeMethod = visibleMethod === 'stripe'
      ? ''
      : visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret && visibleMethod !== 'airwallex'
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const airwallexRouteUrl = result.client_secret && result.intent_id
      ? router.resolve({
        path: '/payment/airwallex',
        query: {
          order_id: String(result.order_id),
          out_trade_no: result.out_trade_no || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType,
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && visibleMethod === 'alipay'),
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
      airwallexRouteUrl,
    })

    if (decision.kind === 'wechat_oauth' && decision.oauth?.authorize_url) {
      window.location.href = buildWechatOAuthAuthorizeUrl(decision.oauth.authorize_url, {
        paymentType: visibleMethod,
        orderType,
        planId,
        orderAmount,
      })
      return
    }

    if (decision.kind === 'unhandled') {
      applyScenarioError({ reason: 'UNHANDLED_PAYMENT_SCENARIO' }, visibleMethod)
      return
    }

    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)

    if (decision.kind === 'stripe_popup') {
      openWindow(decision.paymentState.payUrl)
      return
    }
    if (decision.kind === 'stripe_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'airwallex_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'wechat_jsapi' && decision.jsapi) {
      try {
        const jsapiResult = await invokeWechatJsapiPayment(decision.jsapi as Record<string, unknown>)
        const errMsg = String(jsapiResult.err_msg || '').toLowerCase()
        if (errMsg.includes('cancel')) {
          appStore.showInfo(t('payment.qr.cancelled'))
          resetPayment()
        } else if (errMsg && !errMsg.includes('ok')) {
          resetPayment()
          const fallbackApplied = await attemptMobileQrFallback(
            { reason: 'WECHAT_JSAPI_FAILED', message: errMsg },
            {
              orderAmount,
              orderType,
              planId,
              paymentType: visibleMethod,
              attempted: options.mobileQrFallbackAttempted === true,
            },
          )
          if (!fallbackApplied) {
            applyScenarioError({ reason: 'WECHAT_JSAPI_FAILED', message: errMsg }, visibleMethod)
          }
        } else {
          const resultState = { ...decision.paymentState }
          resetPayment()
          await redirectToPaymentResult(resultState)
        }
      } catch (err: unknown) {
        resetPayment()
        const fallbackApplied = await attemptMobileQrFallback(err, {
          orderAmount,
          orderType,
          planId,
          paymentType: visibleMethod,
          attempted: options.mobileQrFallbackAttempted === true,
        })
        if (!fallbackApplied) {
          throw err
        }
      }
      return
    }
    if (decision.kind === 'redirect_waiting' && decision.paymentState.payUrl) {
      if (isMobileDevice()) {
        window.location.href = decision.paymentState.payUrl
        return
      }
      openWindow(decision.paymentState.payUrl)
    }
  } catch (err: unknown) {
    const apiErr = err as Record<string, unknown>
    if (apiErr.reason === 'TOO_MANY_PENDING') {
      const metadata = apiErr.metadata as Record<string, unknown> | undefined
      errorMessage.value = t('payment.errors.tooManyPending', { max: metadata?.max || '' })
      errorHintMessage.value = ''
    } else if (apiErr.reason === 'CANCEL_RATE_LIMITED') {
      errorMessage.value = t('payment.errors.cancelRateLimited')
      errorHintMessage.value = ''
    } else if (await attemptMobileQrFallback(err, {
      orderAmount,
      orderType,
      planId,
      paymentType: requestType,
      attempted: options.mobileQrFallbackAttempted === true,
    })) {
      return
    } else {
      const handled = applyScenarioError(
        err,
        normalizeVisibleMethod(options.paymentType || selectedMethod.value) || selectedMethod.value,
      )
      if (!handled) {
        errorMessage.value = extractI18nErrorMessage(err, t, 'payment.errors', extractApiErrorMessage(err, t('payment.result.failed')))
        errorHintMessage.value = ''
      }
      if (handled) {
        return
      }
    }
    appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  } finally {
    submitting.value = false
  }
}

interface MobileQrFallbackContext {
  orderAmount: number
  orderType: OrderType
  planId?: number
  paymentType: string
  attempted: boolean
}

function shouldFallbackToDesktopQr(err: unknown, paymentMethod: string, attempted: boolean): boolean {
  if (attempted || !isMobileDevice()) {
    return false
  }

  const normalizedMethod = normalizeVisibleMethod(paymentMethod) || paymentMethod
  const reason = typeof err === 'object' && err && 'reason' in err && typeof err.reason === 'string'
    ? err.reason
    : ''
  const message = err instanceof Error
    ? err.message
    : (typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
      ? err.message
      : '')
  const normalizedMessage = message.toLowerCase()

  if (normalizedMethod === 'wxpay') {
    return reason === 'WECHAT_H5_NOT_AUTHORIZED'
      || reason === 'WECHAT_PAYMENT_MP_NOT_CONFIGURED'
      || reason === 'WECHAT_JSAPI_FAILED'
      || reason === 'PAYMENT_GATEWAY_ERROR'
      || reason === 'UNHANDLED_PAYMENT_SCENARIO'
      || normalizedMessage.includes('weixinjsbridge is unavailable')
      || normalizedMessage.includes('wechat_jsapi_unavailable')
  }

  if (normalizedMethod === 'alipay') {
    return reason === 'PAYMENT_GATEWAY_ERROR' || reason === 'UNHANDLED_PAYMENT_SCENARIO'
  }

  return false
}

async function attemptMobileQrFallback(err: unknown, context: MobileQrFallbackContext): Promise<boolean> {
  if (!shouldFallbackToDesktopQr(err, context.paymentType, context.attempted)) {
    return false
  }

  try {
    const visibleMethod = normalizeVisibleMethod(context.paymentType) || context.paymentType
    const payload = buildCreateOrderPayload({
      amount: context.orderAmount,
      paymentType: visibleMethod,
      orderType: context.orderType,
      planId: context.planId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: false,
      isWechatBrowser: false,
    })
    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const stripeMethod = visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType: context.orderType,
      isMobile: false,
      isWechatBrowser: false,
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
    })

    if (decision.kind !== 'qr_waiting' || !decision.paymentState.qrCode) {
      return false
    }

    errorMessage.value = ''
    errorHintMessage.value = ''
    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)
    appStore.showWarning(t('payment.errors.mobilePaymentFallbackToQr'))
    return true
  } catch {
    return false
  }
}

function applyScenarioError(err: unknown, paymentMethod: string): boolean {
  const descriptor = describePaymentScenarioError(err, {
    paymentMethod,
    isMobile: isMobileDevice(),
    isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
  })
  if (!descriptor) {
    errorMessage.value = ''
    errorHintMessage.value = ''
    return false
  }
  errorMessage.value = t(descriptor.messageKey)
  errorHintMessage.value = descriptor.hintKey ? t(descriptor.hintKey) : ''
  appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  return true
}

async function resumeWechatPaymentFromQuery() {
  const resume = parseWechatResumeRoute(route.query, checkout.value.plans, validAmount.value)
  if (!resume) {
    return
  }

  selectedMethod.value = resume.paymentType
  if (resume.orderType === 'balance' && resume.orderAmount > 0) {
    amount.value = resume.orderAmount
  }
  if (resume.orderType === 'subscription' && resume.planId) {
    selectedPlan.value = checkout.value.plans.find(plan => plan.id === resume.planId) ?? null
  }

  await router.replace({ path: route.path, query: stripWechatResumeQuery(route.query) })

  if (resume.wechatResumeToken) {
    await createOrder(0, resume.orderType, resume.planId, {
      wechatResumeToken: resume.wechatResumeToken,
      paymentType: resume.paymentType,
      isResume: true,
    })
    return
  }

  if (resume.orderAmount > 0 && resume.openid) {
    await createOrder(resume.orderAmount, resume.orderType, resume.planId, {
      openid: resume.openid,
      paymentType: resume.paymentType,
      isResume: true,
    })
  }
}

onMounted(async () => {
  try {
    const res = await paymentAPI.getCheckoutInfo()
    checkout.value = res.data
    setSettlementCnyPerCredit(checkout.value.balance_recharge_multiplier)
    if (enabledMethods.value.length) {
      const order: readonly string[] = METHOD_ORDER
      const sorted = [...enabledMethods.value].sort((a, b) => {
        const ai = order.indexOf(a)
        const bi = order.indexOf(b)
        return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
      })
      selectedMethod.value = sorted[0]
    }
    if (typeof window !== 'undefined') {
      if (hasWechatResumeQuery(route.query)) {
        removeRecoverySnapshot()
      }
      const routeResumeToken = typeof route.query.resume_token === 'string'
        ? route.query.resume_token
        : typeof route.query.wechat_resume_token === 'string'
          ? route.query.wechat_resume_token
          : undefined
      const restored = readPaymentRecoverySnapshot(
        window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY),
        { resumeToken: routeResumeToken },
      )
      if (restored) {
        paymentState.value = restored
        paymentPhase.value = 'paying'
        const restoredMethod = normalizeVisibleMethod(restored.paymentType)
        if (restoredMethod) {
          selectedMethod.value = restoredMethod
        }
      } else {
        removeRecoverySnapshot()
      }
    }
    await resumeWechatPaymentFromQuery()
    if (checkout.value.balance_disabled) {
      activeTab.value = 'subscription'
    }
    // Handle renewal navigation: ?tab=subscription&group=123
    if (route.query.tab === 'subscription') {
      activeTab.value = 'subscription'
      if (route.query.group) {
        const groupId = Number(route.query.group)
        const groupPlans = checkout.value.plans.filter(p => p.group_id === groupId)
        if (groupPlans.length === 1) {
          selectedPlan.value = groupPlans[0]
        } else if (groupPlans.length > 1) {
          renewGroupId.value = groupId
          showRenewalModal.value = true
        }
      }
    }
    applyDefaultRechargeAmount()
  } catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { loading.value = false }
  // Fetch active subscriptions (uses cache, non-blocking)
  subscriptionStore.fetchActiveSubscriptions().catch(() => {})
})
</script>
