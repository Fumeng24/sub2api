<template>
  <div
    :class="[
      'group relative flex flex-col overflow-hidden rounded-lg border transition-colors',
      'border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm',
      'hover:border-[color:var(--apple-border)] hover:bg-[var(--apple-surface-elevated)]',
    ]"
  >
    <div class="flex flex-1 flex-col p-4">
      <div class="mb-3 flex items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <h3
            :title="plan.name"
            class="h-12 min-w-0 break-words text-base font-semibold leading-6 text-[var(--apple-text)] line-clamp-2 [overflow-wrap:anywhere]"
          >
            {{ plan.name }}
          </h3>
          <p v-if="plan.description" class="mt-1 line-clamp-2 text-xs leading-5 text-[var(--apple-muted)]">
            {{ plan.description }}
          </p>
        </div>
        <div class="shrink-0 text-right">
          <div class="flex items-baseline gap-1">
            <span class="text-2xl font-semibold tracking-normal text-[var(--apple-text)]">{{ formatPaymentAmount(plan.price, priceCurrency) }}</span>
          </div>
          <div class="flex items-center justify-end gap-1">
            <span class="shrink-0 rounded-md border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2 py-0.5 text-[11px] font-medium text-[var(--apple-muted)]">
              {{ pLabel }}
            </span>
            <span class="text-[11px] text-[var(--apple-muted-2)]">/ {{ validitySuffix }}</span>
          </div>
          <div v-if="plan.original_price" class="mt-0.5 flex items-center justify-end gap-1.5">
            <span class="text-xs text-[var(--apple-muted-2)] line-through">{{ formatPaymentAmount(plan.original_price, priceCurrency) }}</span>
            <span class="rounded border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-1 py-0.5 text-[10px] font-semibold text-[var(--apple-muted)]">{{ discountText }}</span>
          </div>
        </div>
      </div>

      <div class="mb-3 grid grid-cols-2 gap-x-3 gap-y-1 border-y border-[color:var(--apple-border-soft)] py-3 text-xs">
        <div class="flex items-center justify-between">
          <span class="text-[var(--apple-muted-2)]">{{ t('payment.planCard.rate') }}</span>
          <span class="font-medium text-[var(--apple-text)]">{{ rateDisplay }}</span>
        </div>
        <div v-if="hasPeakRate" class="col-span-2 flex items-center justify-between gap-2">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.peakRate') }}</span>
          <span class="text-right font-medium text-amber-700 dark:text-amber-300">{{ peakRateDisplay }}</span>
        </div>
        <div v-if="plan.daily_limit_usd != null" class="flex items-center justify-between">
          <span class="text-[var(--apple-muted-2)]">{{ t('payment.planCard.dailyLimit') }}</span>
          <span class="font-medium text-[var(--apple-text)]">{{ formatSettlementAmount(plan.daily_limit_usd, 2) }}</span>
        </div>
        <div v-if="plan.weekly_limit_usd != null" class="flex items-center justify-between">
          <span class="text-[var(--apple-muted-2)]">{{ t('payment.planCard.weeklyLimit') }}</span>
          <span class="font-medium text-[var(--apple-text)]">{{ formatSettlementAmount(plan.weekly_limit_usd, 2) }}</span>
        </div>
        <div v-if="plan.monthly_limit_usd != null" class="flex items-center justify-between">
          <span class="text-[var(--apple-muted-2)]">{{ t('payment.planCard.monthlyLimit') }}</span>
          <span class="font-medium text-[var(--apple-text)]">{{ formatSettlementAmount(plan.monthly_limit_usd, 2) }}</span>
        </div>
        <div v-if="plan.daily_limit_usd == null && plan.weekly_limit_usd == null && plan.monthly_limit_usd == null" class="flex items-center justify-between">
          <span class="text-[var(--apple-muted-2)]">{{ t('payment.planCard.quota') }}</span>
          <span class="font-medium text-[var(--apple-text)]">{{ t('payment.planCard.unlimited') }}</span>
        </div>
        <div v-if="modelScopeLabels.length > 0" class="col-span-2 flex items-center justify-between">
          <span class="text-[var(--apple-muted-2)]">{{ t('payment.planCard.models') }}</span>
          <div class="flex flex-wrap justify-end gap-1">
            <span v-for="scope in modelScopeLabels" :key="scope"
              class="rounded bg-[var(--apple-surface-elevated)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]">
              {{ scope }}
            </span>
          </div>
        </div>
      </div>

      <div v-if="plan.features.length > 0" class="mb-3 space-y-1">
        <div v-for="feature in plan.features" :key="feature" class="flex items-start gap-1.5">
          <Icon name="check" size="xs" :stroke-width="2.5" class="mt-0.5 flex-shrink-0 text-[var(--apple-success)]" />
          <span class="text-xs leading-5 text-[var(--apple-muted)]">{{ feature }}</span>
        </div>
      </div>

      <div class="flex-1" />

      <button
        type="button"
        class="btn btn-primary w-full py-2.5 text-sm font-semibold"
        @click="emit('select', plan)"
      >
        {{ isRenewal ? t('payment.renewNow') : t('payment.subscribeNow') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import { DEFAULT_PAYMENT_CURRENCY, formatPaymentAmount } from '@/components/payment/currency'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { useAppStore } from '@/stores/app'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { platformLabel } from '@/utils/platformColors'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ plan: SubscriptionPlan; activeSubscriptions?: UserSubscription[] }>()
const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()
const { formatSettlementAmount } = useSettlementCurrency()
const priceCurrency = DEFAULT_PAYMENT_CURRENCY

const platform = computed(() => props.plan.group_platform || '')
const isRenewal = computed(() =>
  props.activeSubscriptions?.some(s => s.group_id === props.plan.group_id && s.status === 'active') ?? false
)

// Derived color classes from central config
const pLabel = computed(() => platformLabel(platform.value))

const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= 0) return ''
  const pct = Math.round((1 - props.plan.price / props.plan.original_price) * 100)
  return pct > 0 ? `-${pct}%` : ''
})

const rateDisplay = computed(() => {
  const rate = props.plan.rate_multiplier ?? 1
  return `×${Number(rate.toPrecision(10))}`
})

const appStore = useAppStore()

const hasPeakRate = computed(() => groupHasPeakRate(props.plan))

const peakRateDisplay = computed(() => {
  return formatPeakRateWindow(props.plan, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
})

const MODEL_SCOPE_LABELS: Record<string, string> = {
  claude: 'Claude',
  gemini_text: 'Gemini',
  gemini_image: 'Imagen',
}

const modelScopeLabels = computed(() => {
  if (platform.value !== 'antigravity') return []
  const scopes = props.plan.supported_model_scopes
  if (!scopes || scopes.length === 0) return []
  return scopes.map(s => MODEL_SCOPE_LABELS[s] || s)
})

const validitySuffix = computed(() => {
  const u = props.plan.validity_unit || 'day'
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${props.plan.validity_days}${t('payment.days')}`
})
</script>
