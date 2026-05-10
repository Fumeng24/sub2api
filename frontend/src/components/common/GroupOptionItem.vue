<template>
  <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
    <!-- Left: name + description -->
    <div
      class="flex min-w-0 flex-1 flex-col items-start"
      :title="description || undefined"
    >
      <!-- Row 1: platform badge (name bold) -->
      <GroupBadge
        :name="name"
        :platform="platform"
        :subscription-type="subscriptionType"
        :group-id="groupId"
        :show-rate="false"
        class="groupOptionItemBadge"
      />
      <!-- Row 2: description with top spacing -->
      <span
        v-if="description"
        class="mt-1.5 w-full text-left text-xs leading-relaxed text-gray-500 dark:text-gray-400 line-clamp-2"
      >
        {{ description }}
      </span>
    </div>

    <!-- Right: rate pill + checkmark (vertically centered to first row) -->
    <div class="flex shrink-0 items-center gap-2 pt-0.5">
      <!-- Rate pill (platform color) -->
      <span v-if="rateMultiplier !== undefined" :class="['inline-flex items-center whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold', ratePillClass]">
        <template v-if="discountDisplay">
          <span class="mr-1 line-through opacity-50">{{ formatRateMultiplier(originalDisplayRate) }}x</span>
          <span class="font-bold">{{ formatRateMultiplier(discountDisplay.discountedRate) }}x</span>
          <span class="ml-1 rounded bg-white/60 px-1 text-[10px] dark:bg-black/20">
            {{ formatDiscountLabel(discountDisplay.multiplier) }}
          </span>
        </template>
        <template v-else-if="hasCustomRate">
          <span class="mr-1 line-through opacity-50">{{ formatRateMultiplier(rateMultiplier) }}x</span>
          <span class="font-bold">{{ formatRateMultiplier(userRateMultiplier) }}x</span>
        </template>
        <template v-else>
          {{ formatRateMultiplier(rateMultiplier) }}x 倍率
        </template>
      </span>
      <!-- Checkmark -->
      <svg
        v-if="showCheckmark && selected"
        class="h-4 w-4 shrink-0 text-primary-600 dark:text-primary-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import GroupBadge from './GroupBadge.vue'
import { useAppStore } from '@/stores/app'
import { useMinuteNow } from '@/composables/useMinuteNow'
import type { SubscriptionType, GroupPlatform } from '@/types'
import {
  formatDiscountLabel,
  formatRateMultiplier,
  resolveGroupRateDiscount,
  resolvePublicGroupRateDiscount,
} from '@/utils/groupRateDiscount'

interface Props {
  name: string
  platform: GroupPlatform
  subscriptionType?: SubscriptionType
  rateMultiplier?: number
  userRateMultiplier?: number | null
  groupId?: number | null
  discountMultiplier?: number | null
  discountedRateMultiplier?: number | null
  discountName?: string | null
  discountStartAt?: string | null
  discountEndAt?: string | null
  description?: string | null
  selected?: boolean
  showCheckmark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  subscriptionType: 'standard',
  selected: false,
  showCheckmark: true,
  userRateMultiplier: null
})

const appStore = useAppStore()
const now = useMinuteNow()
const originalDisplayRate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const publicDiscountSummary = computed(() => resolvePublicGroupRateDiscount(
  appStore.cachedPublicSettings?.group_rate_discount ?? null,
  appStore.cachedPublicSettings?.upcoming_group_rate_discount ?? null,
  now.value,
))
const discountDisplay = computed(() => resolveGroupRateDiscount(
  props.groupId,
  originalDisplayRate.value,
  publicDiscountSummary.value?.discount ?? null,
  {
    multiplier: props.discountMultiplier,
    discountedRate: props.discountedRateMultiplier,
    name: props.discountName,
    startAt: props.discountStartAt,
    endAt: props.discountEndAt
  },
  publicDiscountSummary.value?.status === 'upcoming',
  now.value,
))

// Whether user has a custom rate different from default
const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

// Rate pill color matches platform badge color
const ratePillClass = computed(() => {
  switch (props.platform) {
    case 'anthropic':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
    case 'openai':
      return 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400'
    case 'gemini':
      return 'bg-sky-50 text-sky-700 dark:bg-sky-900/20 dark:text-sky-400'
    default: // antigravity and others
      return 'bg-violet-50 text-violet-700 dark:bg-violet-900/20 dark:text-violet-400'
  }
})
</script>

<style scoped>
/* Bold the group name inside GroupBadge when used in dropdown option */
.groupOptionItemBadge :deep(span.truncate) {
  font-weight: 600;
}
</style>
