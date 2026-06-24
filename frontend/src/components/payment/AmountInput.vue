<template>
  <div class="space-y-4">
    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2 block text-sm font-medium text-[var(--apple-text)]">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-[repeat(auto-fit,minmax(104px,1fr))] gap-2 sm:grid-cols-3">
        <button
          v-for="amt in displayAmounts"
          :key="amt"
          type="button"
          :disabled="isAmountDisabled(amt)"
          :title="isAmountDisabled(amt) ? disabledReasonText(amt) : undefined"
          :class="[
            'flex h-[76px] min-w-0 flex-col items-center justify-center overflow-hidden rounded-lg border px-2 py-2 text-center font-medium transition-colors sm:px-3',
            isAmountDisabled(amt)
              ? 'cursor-not-allowed border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted-2)] opacity-55'
              : modelValue === amt
              ? 'border-[color:var(--apple-blue)] bg-[color-mix(in_srgb,var(--apple-blue)_9%,var(--apple-surface))] text-[var(--apple-blue)] shadow-sm'
              : 'border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-text)] hover:bg-[var(--apple-hover)]',
          ]"
          @click="selectAmount(amt)"
        >
          <span class="block max-w-full whitespace-nowrap text-[13px] leading-5 sm:text-sm">{{ amountLabel ? amountLabel(amt) : amt }}</span>
          <span
            v-if="amountDescription || isAmountDisabled(amt)"
            class="mt-0.5 block max-w-full whitespace-nowrap text-[11px] leading-4"
            :class="isAmountDisabled(amt) ? 'text-[var(--apple-muted-2)]' : 'text-[var(--apple-muted)]'"
          >
            {{ isAmountDisabled(amt) ? disabledReasonText(amt) : amountDescriptionText(amt) }}
          </span>
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-[var(--apple-text)]">
        {{ t('payment.customBalanceCredit') }}
      </label>
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted-2)]">
          {{ prefix }}
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input w-full min-h-[48px] py-3 pl-8 pr-4 text-base sm:text-sm"
          @input="handleInput"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  prefix?: string
  amountLabel?: (amount: number) => string
  amountDescription?: (amount: number) => string
  disabledReason?: (amount: number) => string
}>(), {
  amounts: () => [5, 10, 20, 50, 100, 200],
  min: 0,
  max: 0,
  prefix: '$',
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const displayAmounts = computed(() => props.amounts)

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  if (isAmountDisabled(amt)) return
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function isAmountDisabled(amt: number): boolean {
  if (props.disabledReason?.(amt)) return true
  if (props.min > 0 && amt < props.min) return true
  if (props.max > 0 && amt > props.max) return true
  return false
}

function disabledReasonText(amt: number): string {
  const reason = props.disabledReason?.(amt)
  if (reason) return reason
  if (props.min > 0 && amt < props.min) return t('payment.quickAmountBelowLimit')
  if (props.max > 0 && amt > props.max) return t('payment.quickAmountAboveLimit')
  return t('payment.quickAmountUnavailable')
}

function amountDescriptionText(amt: number): string {
  return props.amountDescription?.(amt) || ''
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  if (!AMOUNT_PATTERN.test(val)) return
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
