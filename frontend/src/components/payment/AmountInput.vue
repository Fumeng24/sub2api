<template>
  <div class="space-y-4">
    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-[repeat(auto-fit,minmax(92px,1fr))] gap-2 sm:grid-cols-3">
        <button
          v-for="amt in displayAmounts"
          :key="amt"
          type="button"
          :disabled="isAmountDisabled(amt)"
          :title="isAmountDisabled(amt) ? disabledReasonText(amt) : undefined"
          :class="[
            'flex h-[72px] min-w-0 flex-col items-center justify-center overflow-hidden rounded-lg border-2 px-1.5 py-2 text-center font-medium transition-colors sm:px-3',
            isAmountDisabled(amt)
              ? 'cursor-not-allowed border-gray-100 bg-gray-50 text-gray-300 dark:border-dark-700 dark:bg-dark-800/50 dark:text-dark-500'
              : modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(amt)"
        >
          <span class="block max-w-full whitespace-nowrap text-[13px] leading-5 sm:text-sm">{{ amountLabel ? amountLabel(amt) : amt }}</span>
          <span
            v-if="amountDescription || isAmountDisabled(amt)"
            class="mt-0.5 block max-w-full whitespace-nowrap text-[11px] leading-4"
            :class="isAmountDisabled(amt) ? 'text-gray-300 dark:text-dark-500' : 'text-gray-500 dark:text-gray-400'"
          >
            {{ isAmountDisabled(amt) ? disabledReasonText(amt) : amountDescriptionText(amt) }}
          </span>
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-500">
          {{ prefix }}
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input w-full py-3 pl-8 pr-4"
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
