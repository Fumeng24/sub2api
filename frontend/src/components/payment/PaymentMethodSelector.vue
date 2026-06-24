<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-[var(--apple-text)]">
      {{ t('payment.paymentMethod') }}
    </label>
    <div class="grid grid-cols-2 gap-3 sm:flex">
      <button
        v-for="method in sortedMethods"
        :key="method.type"
        type="button"
        :disabled="!method.available"
        :class="[
          'relative flex min-h-[72px] min-w-0 flex-col items-center justify-center rounded-lg border px-3 transition-colors sm:flex-1',
          !method.available
            ? 'cursor-not-allowed border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] opacity-55'
            : selected === method.type
              ? methodSelectedClass(method.type)
              : 'border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-text)] hover:bg-[var(--apple-hover)]',
        ]"
        @click="method.available && emit('select', method.type)"
      >
        <span class="flex min-w-0 items-center gap-2">
          <img :src="methodIcon(method.type)" :alt="t(`payment.methods.${method.type}`)" class="h-7 w-7 shrink-0 object-contain" />
          <span class="flex min-w-0 flex-col items-start leading-none">
            <span class="max-w-full truncate text-base font-semibold">{{ t(`payment.methods.${method.type}`) }}</span>
            <span
              v-if="method.fee_rate > 0"
              class="max-w-full truncate text-[10px] text-[var(--apple-muted)]"
            >
              {{ t('payment.fee') }} {{ method.fee_rate }}%
            </span>
            <span v-else class="max-w-full truncate text-[10px] text-[var(--apple-muted-2)]">
              {{ t('payment.methodNoFee') }}
            </span>
          </span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER } from './providerConfig'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import usdtIcon from '@/assets/icons/usdt.svg'

export interface PaymentMethodOption {
  type: string
  fee_rate: number
  available: boolean
}

const props = defineProps<{
  methods: PaymentMethodOption[]
  selected: string
}>()

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()

const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
  airwallex: airwallexIcon,
  usdt: usdtIcon,
}

const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function methodIcon(type: string): string {
  if (type.includes('alipay')) return METHOD_ICONS.alipay
  if (type.includes('wxpay')) return METHOD_ICONS.wxpay
  if (type === 'airwallex') return METHOD_ICONS.airwallex
  if (type === 'usdt') return METHOD_ICONS.usdt
  return METHOD_ICONS[type] || alipayIcon
}

function methodSelectedClass(type: string): string {
  void type
  return 'border-[color:var(--apple-blue)] bg-[color-mix(in_srgb,var(--apple-blue)_9%,var(--apple-surface))] text-[var(--apple-text)] shadow-sm'
}
</script>
