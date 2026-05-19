<template>
  <div v-if="items.length > 0" class="mt-3 flex flex-wrap gap-2">
    <div v-for="(attachment, index) in items" :key="`${attachment.url}-${index}`" class="max-w-full">
      <a
        :href="attachment.url"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex max-w-full items-center gap-1.5 rounded-md border border-gray-200 bg-white px-2 py-1 text-xs font-medium text-gray-700 hover:border-primary-300 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200 dark:hover:border-primary-800 dark:hover:text-primary-300"
      >
        <Icon :name="isImageURL(attachment.url) ? 'externalLink' : 'link'" size="xs" />
        <span class="truncate">{{ attachment.name }}</span>
      </a>
      <a
        v-if="isImageURL(attachment.url)"
        :href="attachment.url"
        target="_blank"
        rel="noopener noreferrer"
        class="mt-2 block"
      >
        <img
          :src="attachment.url"
          :alt="attachment.name"
          class="max-h-36 max-w-full rounded-md border border-gray-200 object-contain dark:border-dark-700"
        />
      </a>
    </div>
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

function isImageURL(value?: string) {
  return Boolean(value && (/^data:image\/(?:png|jpe?g|webp|gif);base64,/i.test(value) || /\.(png|jpe?g|webp|gif)(\?|#|$)/i.test(value)))
}
</script>
