<template>
  <router-link
    v-if="contextType || contextId"
    :to="targetPath"
    class="inline-flex max-w-full items-center gap-1.5 rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 hover:bg-primary-50 hover:text-primary-700 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
  >
    <Icon name="link" size="xs" />
    <span>{{ label }}</span>
    <span v-if="contextId" class="truncate font-mono">#{{ contextId }}</span>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  contextType?: string
  contextId?: string
  admin?: boolean
}>()

const { t } = useI18n()

const label = computed(() => {
  const key = props.contextType || 'general'
  return t(`tickets.context.${key}`, key)
})

const targetPath = computed(() => {
  const id = props.contextId || ''
  switch (props.contextType) {
    case 'usage':
      return props.admin ? `/admin/usage?search=${encodeURIComponent(id)}` : `/usage?search=${encodeURIComponent(id)}`
    case 'request':
    case 'request_id':
      return props.admin ? `/admin/usage?search=${encodeURIComponent(id)}` : `/usage?search=${encodeURIComponent(id)}`
    case 'order':
      return props.admin ? `/admin/orders?search=${encodeURIComponent(id)}` : `/orders?search=${encodeURIComponent(id)}`
    case 'invoice':
      return props.admin ? `/admin/invoices?search=${encodeURIComponent(id)}` : `/invoices?search=${encodeURIComponent(id)}`
    case 'api_key':
      return props.admin ? `/admin/users?search=${encodeURIComponent(id)}` : `/keys?search=${encodeURIComponent(id)}`
    default:
      return props.admin ? '/admin/tickets' : '/tickets'
  }
})
</script>
