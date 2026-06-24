<template>
  <div class="pricing-row flex justify-between gap-3">
    <span class="pricing-row-label">{{ label }}</span>
    <span class="pricing-row-value font-mono tabular-nums">{{ display }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatScaled } from '@/utils/pricing'

const props = withDefaults(
  defineProps<{
    label: string
    value: number | null
    unit: string
    scale: number
  }>(),
  { value: null }
)

const display = computed(() =>
  props.value == null ? '-' : `${formatScaled(props.value, props.scale)} ${props.unit}`
)
</script>

<style scoped>
.pricing-row-label {
  color: var(--apple-muted);
}

.pricing-row-value {
  color: var(--apple-text);
  font-weight: 600;
  text-align: right;
}
</style>
