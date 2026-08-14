<template>
  <div
    v-if="groupRateDiscountSummary"
    :class="[
      'rounded-lg border px-4 py-3 text-sm shadow-sm',
      groupRateDiscountSummary.status === 'active'
        ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
        : 'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200'
    ]"
  >
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <Icon name="badge" size="sm" class="shrink-0" />
        <span class="font-semibold">
          {{ groupRateDiscountSummary.discount.name || localText('限时分组折扣', 'Limited-time group discount') }}
        </span>
        <span class="rounded bg-white/70 px-1.5 py-0.5 text-xs font-semibold dark:bg-black/20">
          {{ groupRateDiscountStatusLabel }}
        </span>
        <span class="rounded bg-white/70 px-1.5 py-0.5 text-xs font-bold dark:bg-black/20">
          {{ formatDiscountLabel(groupRateDiscountSummary.discount.discount_multiplier) }}
        </span>
      </div>
      <span v-if="groupRateDiscountSchedule" class="text-xs font-medium">
        {{ groupRateDiscountSchedule }}
      </span>
    </div>
    <p class="mt-2 text-xs font-medium opacity-90">
      {{ groupRateDiscountMessage }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useMinuteNow } from '@/custom/composables/useMinuteNow'
import { useAppStore } from '@/stores/app'
import {
  formatDiscountLabel,
  formatDiscountSchedule,
  formatDiscountStatusLabel,
  resolvePublicGroupRateDiscount,
} from '@/custom/utils/groupRateDiscount'

const { locale } = useI18n()
const appStore = useAppStore()
const now = useMinuteNow()

const groupRateDiscountSummary = computed(() => resolvePublicGroupRateDiscount(
  appStore.cachedPublicSettings?.group_rate_discount ?? null,
  appStore.cachedPublicSettings?.upcoming_group_rate_discount ?? null,
  now.value,
))
const groupRateDiscountSchedule = computed(() =>
  formatDiscountSchedule(
    groupRateDiscountSummary.value?.discount,
    locale.value,
    groupRateDiscountSummary.value?.status,
  )
)
const groupRateDiscountStatusLabel = computed(() =>
  groupRateDiscountSummary.value
    ? formatDiscountStatusLabel(groupRateDiscountSummary.value.status, locale.value)
    : ''
)
const groupRateDiscountMessage = computed(() => {
  if (!groupRateDiscountSummary.value) return ''
  if (groupRateDiscountSummary.value.status === 'active') {
    return localText('当前请求按折后倍率计费。', 'Current requests are billed at the discounted rate.')
  }
  return localText('到点后自动生效，当前请求仍按原倍率计费。', 'It starts automatically at the scheduled time; current requests still use the original rate.')
})

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

onMounted(() => {
  appStore.fetchPublicSettings(true)
})
</script>
