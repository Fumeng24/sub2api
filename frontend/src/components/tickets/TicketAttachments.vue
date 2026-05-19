<template>
  <div v-if="items.length > 0" class="mt-3 flex flex-wrap gap-2">
    <a
      v-for="(attachment, index) in items"
      :key="`${attachment.url}-${index}`"
      :href="attachment.url"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex max-w-full items-center gap-1.5 rounded-md border border-gray-200 bg-white px-2 py-1 text-xs font-medium text-gray-700 hover:border-primary-300 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200 dark:hover:border-primary-800 dark:hover:text-primary-300"
    >
      <Icon name="link" size="xs" />
      <span class="truncate">{{ attachment.name }}</span>
    </a>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { TicketAttachment } from '@/types'

const props = defineProps<{
  attachments?: TicketAttachment[]
}>()

const items = computed(() => (props.attachments || []).filter((item) => item.name && item.url))
</script>
