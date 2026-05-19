<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-3">
      <label class="input-label mb-0">{{ t(`${i18nPrefix}.title`) }}</label>
      <button
        v-if="attachments.length < maxItems"
        type="button"
        class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
        @click="addAttachment"
      >
        <Icon name="plus" size="sm" />
        {{ t(`${i18nPrefix}.add`) }}
      </button>
    </div>

    <div v-if="attachments.length > 0" class="space-y-2">
      <div
        v-for="(attachment, index) in attachments"
        :key="index"
        class="grid grid-cols-1 gap-2 rounded-lg border border-gray-200 p-3 dark:border-dark-700 md:grid-cols-[minmax(0,0.8fr)_minmax(0,1.4fr)_auto]"
      >
        <input
          :value="attachment.name"
          type="text"
          class="input"
          maxlength="120"
          :placeholder="t(`${i18nPrefix}.namePlaceholder`)"
          @input="updateAttachment(index, 'name', eventValue($event))"
        />
        <input
          :value="attachment.url"
          type="url"
          class="input"
          maxlength="1000"
          :placeholder="t(`${i18nPrefix}.urlPlaceholder`)"
          @input="updateAttachment(index, 'url', eventValue($event))"
        />
        <button
          type="button"
          class="btn btn-secondary btn-icon"
          :title="t('common.delete')"
          @click="removeAttachment(index)"
        >
          <Icon name="trash" size="sm" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { TicketAttachment } from '@/types'

const props = withDefaults(defineProps<{
  modelValue?: TicketAttachment[]
  i18nPrefix?: string
  maxItems?: number
}>(), {
  i18nPrefix: 'tickets.attachments',
  maxItems: 5
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: TicketAttachment[]): void
}>()

const { t } = useI18n()
const attachments = computed(() => props.modelValue || [])

function addAttachment() {
  if (attachments.value.length >= props.maxItems) return
  emit('update:modelValue', [...attachments.value, { name: '', url: '' }])
}

function updateAttachment(index: number, key: keyof TicketAttachment, value: string) {
  const next = attachments.value.map((item, i) => (i === index ? { ...item, [key]: value } : item))
  emit('update:modelValue', next)
}

function removeAttachment(index: number) {
  emit('update:modelValue', attachments.value.filter((_, i) => i !== index))
}

function eventValue(event: Event) {
  return (event.target as HTMLInputElement | null)?.value || ''
}
</script>
