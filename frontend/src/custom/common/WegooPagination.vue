<template>
  <div class="pagination-root flex min-w-0 items-center justify-between gap-3 px-3 py-3 sm:px-4">
    <div class="flex min-w-0 flex-1 items-center justify-between gap-3 sm:hidden">
      <button
        @click="goToPage(page - 1)"
        :disabled="page === 1"
        class="btn btn-secondary btn-sm shrink-0"
      >
        {{ t('pagination.previous') }}
      </button>
      <span class="min-w-0 truncate text-sm text-gray-600 dark:text-gray-300">
        {{ t('pagination.pageOf', { page, total: totalPages }) }}
      </span>
      <button
        @click="goToPage(page + 1)"
        :disabled="page === totalPages"
        class="btn btn-secondary btn-sm shrink-0"
      >
        {{ t('pagination.next') }}
      </button>
    </div>

    <div class="hidden min-w-0 flex-1 flex-wrap items-center justify-between gap-3 sm:flex">
      <div class="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-2">
        <p class="min-w-0 text-sm text-gray-600 dark:text-gray-300">
          {{ t('pagination.showing') }}
          <span class="font-medium">{{ fromItem }}</span>
          {{ t('pagination.to') }}
          <span class="font-medium">{{ toItem }}</span>
          {{ t('pagination.of') }}
          <span class="font-medium">{{ total }}</span>
          {{ t('pagination.results') }}
        </p>

        <!-- Page size selector -->
        <div v-if="showPageSizeSelector" class="flex items-center gap-2">
          <span class="text-sm text-gray-600 dark:text-gray-300"
            >{{ t('pagination.perPage') }}:</span
          >
          <div class="page-size-select w-20">
            <Select
              :model-value="pageSize"
              :options="pageSizeSelectOptions"
              @update:model-value="handlePageSizeChange"
            />
          </div>
        </div>

        <div v-if="showJump" class="flex items-center gap-2">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('pagination.jumpTo') }}</span>
          <input
            v-model="jumpPage"
            type="number"
            min="1"
            :max="totalPages"
            class="input w-20 text-sm"
            :placeholder="t('pagination.jumpPlaceholder')"
            @keyup.enter="submitJump"
          />
          <button type="button" class="btn btn-ghost btn-sm" @click="submitJump">
            {{ t('pagination.jumpAction') }}
          </button>
        </div>
      </div>

      <nav
        class="pagination-nav relative z-0 inline-flex max-w-full overflow-hidden rounded-lg border"
        aria-label="Pagination"
      >
        <button
          @click="goToPage(page - 1)"
          :disabled="page === 1"
          class="pagination-nav-button relative inline-flex items-center border-r px-2.5 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
          :aria-label="t('pagination.previous')"
        >
          <Icon name="chevronLeft" size="md" />
        </button>

        <button
          v-for="(pageNum, index) in visiblePages"
          :key="`${pageNum}-${index}`"
          @click="typeof pageNum === 'number' && goToPage(pageNum)"
          :disabled="typeof pageNum !== 'number'"
          :class="[
            'pagination-nav-button relative inline-flex min-w-10 items-center justify-center border-r px-3 py-2 text-sm font-medium last:border-r-0',
            pageNum === page
              ? 'pagination-nav-button-active z-10'
              : '',
            typeof pageNum !== 'number' && 'cursor-default'
          ]"
          :aria-label="
            typeof pageNum === 'number' ? t('pagination.goToPage', { page: pageNum }) : undefined
          "
          :aria-current="pageNum === page ? 'page' : undefined"
        >
          {{ pageNum }}
        </button>

        <button
          @click="goToPage(page + 1)"
          :disabled="page === totalPages"
          class="pagination-nav-button relative inline-flex items-center px-2.5 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
          :aria-label="t('pagination.next')"
        >
          <Icon name="chevronRight" size="md" />
        </button>
      </nav>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from './WegooSelect.vue'
import { getConfiguredTablePageSizeOptions } from '@/utils/tablePreferences'
import { setPersistedPageSize } from '@/custom/composables/usePersistedPageSize'

const { t } = useI18n()

interface Props {
  total: number
  page: number
  pageSize: number
  pageSizeOptions?: number[]
  showPageSizeSelector?: boolean
  showJump?: boolean
}

interface Emits {
  (e: 'update:page', page: number): void
  (e: 'update:pageSize', pageSize: number): void
}

const props = withDefaults(defineProps<Props>(), {
  pageSizeOptions: () => getConfiguredTablePageSizeOptions(),
  showPageSizeSelector: true,
  showJump: false
})

const emit = defineEmits<Emits>()

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))

const fromItem = computed(() => {
  if (props.total === 0) return 0
  return (props.page - 1) * props.pageSize + 1
})

const toItem = computed(() => {
  const to = props.page * props.pageSize
  return to > props.total ? props.total : to
})

const normalizePageSizeOptions = (values: number[]) => {
  const options = values
    .map((value) => Number(value))
    .filter((value) => Number.isInteger(value) && value > 0)
  const unique = Array.from(new Set(options)).sort((a, b) => a - b)
  return unique.length > 0 ? unique : getConfiguredTablePageSizeOptions()
}

const availablePageSizeOptions = computed(() => normalizePageSizeOptions(props.pageSizeOptions))

const normalizePageSizeToAvailableOptions = (value: unknown) => {
  const size = Number(value)
  const options = availablePageSizeOptions.value
  if (!Number.isFinite(size)) return options[0]
  for (const option of options) {
    if (option >= size) return option
  }
  return options[options.length - 1]
}

const pageSizeSelectOptions = computed(() => {
  const options = availablePageSizeOptions.value

  return options.map((size) => ({
    value: size,
    label: String(size)
  }))
})

const jumpPage = ref('')

const visiblePages = computed(() => {
  const pages: (number | string)[] = []
  const maxVisible = 7
  const total = totalPages.value

  if (total <= maxVisible) {
    // Show all pages if total is small
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    // Always show first page
    pages.push(1)

    const start = Math.max(2, props.page - 2)
    const end = Math.min(total - 1, props.page + 2)

    // Add ellipsis before if needed
    if (start > 2) {
      pages.push('...')
    }

    // Add middle pages
    for (let i = start; i <= end; i++) {
      pages.push(i)
    }

    // Add ellipsis after if needed
    if (end < total - 1) {
      pages.push('...')
    }

    // Always show last page
    pages.push(total)
  }

  return pages
})

const goToPage = (newPage: number) => {
  if (newPage >= 1 && newPage <= totalPages.value && newPage !== props.page) {
    emit('update:page', newPage)
  }
}

const handlePageSizeChange = (value: string | number | boolean | null) => {
  if (value === null || typeof value === 'boolean') return
  const newPageSize = normalizePageSizeToAvailableOptions(typeof value === 'string' ? parseInt(value, 10) : value)
  setPersistedPageSize(newPageSize)
  emit('update:pageSize', newPageSize)
}

const submitJump = () => {
  const value = jumpPage.value.trim()
  if (!value) return
  const pageNum = Number.parseInt(value, 10)
  if (Number.isNaN(pageNum)) return
  const nextPage = Math.min(Math.max(pageNum, 1), totalPages.value)
  jumpPage.value = ''
  goToPage(nextPage)
}
</script>

<style scoped>
.pagination-root {
  background: var(--apple-surface);
  color: var(--apple-text);
}

.pagination-nav {
  background: var(--apple-surface);
  border-color: var(--apple-border);
  border-radius: var(--apple-radius);
}

.pagination-nav-button {
  border-color: var(--apple-border-soft);
  color: var(--apple-muted);
  transition: background-color 0.15s ease, color 0.15s ease;
}

.pagination-nav-button:hover:not(:disabled) {
  background: var(--apple-hover);
  color: var(--apple-text);
}

.pagination-nav-button-active,
.pagination-nav-button-active:hover:not(:disabled) {
  background: var(--apple-blue);
  color: #fff;
}

.page-size-select :deep(.select-trigger) {
  @apply px-3 py-1.5 text-sm;
}
</style>
