<template>
  <div :class="['grid gap-2', gridClass]">
    <div
      v-for="item in items"
      :key="item.label"
      class="card min-w-0 overflow-hidden p-3"
    >
      <div class="flex items-center justify-between gap-3">
        <p class="min-w-0 truncate text-xs font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</p>
        <Icon v-if="item.icon" :name="item.icon" size="xs" :class="item.iconClass" />
      </div>
      <strong :class="['mt-1 block truncate text-xl font-bold text-gray-900 dark:text-white', item.valueClass]">{{ item.value }}</strong>
      <span v-if="item.meta" class="mt-1 block truncate text-xs text-gray-500 dark:text-dark-400" :title="item.meta">{{ item.meta }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']

withDefaults(defineProps<{
  items: Array<{
    label: string
    value: string | number
    valueClass?: string
    meta?: string
    icon?: IconName
    iconClass?: string
  }>
  gridClass?: string
}>(), {
  gridClass: 'grid-cols-2 sm:grid-cols-4',
})
</script>
