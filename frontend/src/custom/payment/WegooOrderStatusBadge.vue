<template>
  <span
    class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium"
    :class="statusClass"
  >
    {{ statusLabel }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OrderStatus } from '@/types/payment'

const props = defineProps<{
  status: OrderStatus
}>()

const { t } = useI18n()

const statusMap: Record<OrderStatus, { key: string; class: string }> = {
  PENDING: { key: 'payment.status.pending', class: 'border-[color:color-mix(in_srgb,var(--apple-warning)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_9%,var(--apple-surface))] text-[var(--apple-warning)]' },
  PAID: { key: 'payment.status.paid', class: 'border-[color:color-mix(in_srgb,var(--apple-blue)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-blue)_9%,var(--apple-surface))] text-[var(--apple-blue)]' },
  RECHARGING: { key: 'payment.status.recharging', class: 'border-[color:color-mix(in_srgb,var(--apple-blue)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-blue)_9%,var(--apple-surface))] text-[var(--apple-blue)]' },
  COMPLETED: { key: 'payment.status.completed', class: 'border-[color:color-mix(in_srgb,var(--apple-success)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_9%,var(--apple-surface))] text-[var(--apple-success)]' },
  EXPIRED: { key: 'payment.status.expired', class: 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]' },
  CANCELLED: { key: 'payment.status.cancelled', class: 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]' },
  FAILED: { key: 'payment.status.failed', class: 'border-[color:color-mix(in_srgb,var(--apple-danger)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_9%,var(--apple-surface))] text-[var(--apple-danger)]' },
  REFUND_REQUESTED: { key: 'payment.status.refund_requested', class: 'border-[color:color-mix(in_srgb,var(--apple-warning)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_9%,var(--apple-surface))] text-[var(--apple-warning)]' },
  REFUNDING: { key: 'payment.status.refunding', class: 'border-[color:color-mix(in_srgb,var(--apple-warning)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_9%,var(--apple-surface))] text-[var(--apple-warning)]' },
  REFUND_PENDING: { key: 'payment.status.refund_pending', class: 'border-[color:color-mix(in_srgb,var(--apple-warning)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_9%,var(--apple-surface))] text-[var(--apple-warning)]' },
  REFUNDED: { key: 'payment.status.refunded', class: 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]' },
  PARTIALLY_REFUNDED: { key: 'payment.status.partially_refunded', class: 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]' },
  REFUND_FAILED: { key: 'payment.status.refund_failed', class: 'border-[color:color-mix(in_srgb,var(--apple-danger)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_9%,var(--apple-surface))] text-[var(--apple-danger)]' },
}

const statusLabel = computed(() => {
  const entry = statusMap[props.status]
  return entry ? t(entry.key) : props.status
})

const statusClass = computed(() => {
  const entry = statusMap[props.status]
  return entry?.class ?? 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]'
})
</script>
