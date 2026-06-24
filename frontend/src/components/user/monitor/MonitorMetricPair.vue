<template>
  <div class="mt-5 grid grid-cols-2 gap-2">
    <div
      class="rounded-lg bg-[var(--apple-surface-elevated)] p-3"
    >
      <div
        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wide text-[var(--apple-muted)]"
      >
        <Icon :name="primaryIcon" size="xs" />
        <span>{{ primaryLabel }}</span>
      </div>
      <div class="mt-1.5 font-mono text-lg font-semibold tabular-nums text-[var(--apple-text)]">
        {{ primaryValue }}<span class="ml-0.5 text-xs font-normal text-[var(--apple-muted)]">{{ primaryUnit }}</span>
      </div>
    </div>
    <div
      class="rounded-lg bg-[var(--apple-surface-elevated)] p-3"
    >
      <div
        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wide text-[var(--apple-muted)]"
      >
        <Icon :name="secondaryIcon" size="xs" />
        <span>{{ secondaryLabel }}</span>
      </div>
      <div class="mt-1.5 font-mono text-lg font-semibold tabular-nums text-[var(--apple-text)]">
        {{ secondaryValue }}<span class="ml-0.5 text-xs font-normal text-[var(--apple-muted)]">{{ secondaryUnit }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  primaryLabel: string
  primaryValue: string
  primaryUnit: string
  primaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
  secondaryLabel: string
  secondaryValue: string
  secondaryUnit: string
  secondaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
}>()
</script>
