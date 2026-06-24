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
        class="grid grid-cols-1 gap-2 rounded-lg border border-gray-200 p-3 dark:border-dark-700 md:grid-cols-[minmax(0,0.8fr)_minmax(0,1.4fr)_auto_auto]"
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
          type="text"
          class="input"
          :maxlength="maxURLLength(attachment.url)"
          :placeholder="t(`${i18nPrefix}.urlPlaceholder`)"
          @input="updateAttachment(index, 'url', eventValue($event))"
        />
        <input
          :id="`ticket-attachment-file-${index}`"
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          class="sr-only"
          @change="handleAttachmentFile(index, $event)"
        />
        <label
          :for="`ticket-attachment-file-${index}`"
          class="btn btn-secondary btn-icon cursor-pointer"
          :title="t(`${i18nPrefix}.chooseImage`)"
        >
          <Icon name="upload" size="sm" />
        </label>
        <button
          type="button"
          class="btn btn-secondary btn-icon"
          :title="t('common.delete')"
          @click="removeAttachment(index)"
        >
          <Icon name="trash" size="sm" />
        </button>
        <div
          v-if="isImageURL(attachment.url)"
          class="flex items-center gap-2 md:col-span-4"
        >
          <img
            :src="attachment.url"
            :alt="attachment.name"
            class="h-14 w-14 rounded-md border border-gray-200 object-cover dark:border-dark-700"
          />
          <span class="min-w-0 flex-1 truncate text-xs text-gray-500 dark:text-dark-400">
            {{ attachment.name || t(`${i18nPrefix}.imageSelected`) }}
          </span>
        </div>
      </div>
    </div>

    <p v-if="attachmentError" class="text-xs leading-5 text-[var(--apple-danger)]">
      {{ attachmentError }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
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
const maxInlineImageBytes = 2 * 1024 * 1024
const attachments = computed(() => props.modelValue || [])
const attachmentError = ref('')

function addAttachment() {
  if (attachments.value.length >= props.maxItems) return
  emit('update:modelValue', [...attachments.value, { name: '', url: '' }])
}

function updateAttachment(index: number, key: keyof TicketAttachment, value: string) {
  const next = attachments.value.map((item, i) => (i === index ? { ...item, [key]: value } : item))
  emit('update:modelValue', next)
}

async function handleAttachmentFile(index: number, event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0]
  if (input) input.value = ''
  attachmentError.value = ''
  if (!file) return
  if (!isAllowedInlineImageType(file.type)) {
    attachmentError.value = t(`${props.i18nPrefix}.invalidImage`)
    return
  }
  if (file.size > maxInlineImageBytes) {
    attachmentError.value = t(`${props.i18nPrefix}.imageTooLarge`, { size: 2 })
    return
  }
  try {
    const dataURL = await readFileAsDataURL(file)
    const next = attachments.value.map((item, i) => (
      i === index
        ? { ...item, name: item.name || file.name, url: dataURL, content_type: file.type, size: file.size }
        : item
    ))
    emit('update:modelValue', next)
  } catch {
    attachmentError.value = t(`${props.i18nPrefix}.readFailed`)
  }
}

function removeAttachment(index: number) {
  emit('update:modelValue', attachments.value.filter((_, i) => i !== index))
}

function eventValue(event: Event) {
  return (event.target as HTMLInputElement | null)?.value || ''
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === 'string') resolve(reader.result)
      else reject(new Error('invalid file reader result'))
    }
    reader.onerror = () => reject(reader.error || new Error('failed to read file'))
    reader.readAsDataURL(file)
  })
}

function isImageURL(value?: string) {
  return Boolean(value && (/^data:image\/(?:png|jpe?g|webp|gif);base64,/i.test(value) || /\.(png|jpe?g|webp|gif)(\?|#|$)/i.test(value)))
}

function maxURLLength(value?: string) {
  return value && /^data:image\/(?:png|jpe?g|webp|gif);base64,/i.test(value) ? 3 * 1024 * 1024 : 1000
}

function isAllowedInlineImageType(value: string) {
  return ['image/png', 'image/jpeg', 'image/webp', 'image/gif'].includes(value.toLowerCase())
}
</script>
