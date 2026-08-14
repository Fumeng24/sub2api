<template>
  <section class="card p-5 md:p-6">
    <div
      v-if="hasAside"
      class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(340px,0.58fr)] lg:items-stretch"
    >
      <div class="min-w-0">
        <p v-if="kicker" class="text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">{{ kicker }}</p>
        <div class="mt-2 flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h1 class="min-w-0 text-2xl font-bold leading-tight text-gray-900 dark:text-white sm:text-3xl">
            {{ title }}
          </h1>
          <slot name="titleSuffix" />
        </div>
        <p v-if="description" class="mt-2 max-w-3xl text-sm leading-6 text-gray-500 dark:text-dark-400">
          {{ description }}
        </p>
        <slot name="body" />
      </div>

      <div class="min-w-0">
        <slot name="aside" />
      </div>
    </div>

    <div v-else class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div class="min-w-0 flex-1">
        <p v-if="kicker" class="text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">{{ kicker }}</p>
        <div class="mt-2 flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h1 class="min-w-0 text-2xl font-bold leading-tight text-gray-900 dark:text-white sm:text-3xl">
            {{ title }}
          </h1>
          <slot name="titleSuffix" />
        </div>
        <p v-if="description" class="mt-2 max-w-3xl text-sm leading-6 text-gray-500 dark:text-dark-400">
          {{ description }}
        </p>
        <slot name="body" />
      </div>

      <slot name="actions" />
    </div>

    <slot name="below" />
  </section>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue'

defineProps<{
  kicker?: string
  title: string
  description?: string
}>()

const slots = useSlots()
const hasAside = computed(() => Boolean(slots.aside))
</script>
