<template>
  <div class="table-page-layout" :class="{ 'mobile-mode': isMobile && !disableMobileMode }">
    <div
      v-if="$slots.actions"
      class="layout-section-fixed"
      :class="{ 'layout-section-panel': !flushActions }"
    >
      <slot name="actions" />
    </div>

    <div v-if="$slots.filters" class="layout-section-fixed layout-section-panel">
      <slot name="filters" />
    </div>

    <div v-if="$slots.belowFilters" class="layout-section-fixed">
      <slot name="belowFilters" />
    </div>

    <div class="layout-section-scrollable">
      <div class="table-scroll-container">
        <slot name="table" />
      </div>
    </div>

    <div v-if="$slots.pagination" class="layout-section-fixed layout-section-pagination">
      <slot name="pagination" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{
  flushActions?: boolean
  disableMobileMode?: boolean
}>()

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
.table-page-layout {
  @apply flex min-w-0 flex-col gap-4;
  height: calc(100vh - 64px - 3rem);
}

.layout-section-fixed {
  @apply min-w-0 flex-shrink-0;
}

.layout-section-panel {
  @apply rounded-lg border p-3;
  background: var(--apple-surface);
  border-color: var(--apple-border);
  box-shadow: var(--apple-shadow-sm);
}

.layout-section-panel :deep(> *) {
  @apply min-w-0;
}

.layout-section-panel :deep(.btn) {
  @apply flex-shrink-0;
}

.layout-section-scrollable {
  @apply min-w-0 flex-1 min-h-0 flex flex-col;
}

.table-scroll-container {
  @apply flex h-full min-w-0 flex-col overflow-hidden rounded-lg border;
  background: var(--apple-surface);
  border-color: var(--apple-border);
  border-radius: var(--apple-radius);
  box-shadow: var(--apple-shadow-sm);
}

:global(.dark) .layout-section-panel,
:global(.dark) .table-scroll-container {
  background: color-mix(in srgb, var(--apple-surface) 94%, white 6%);
}

.table-scroll-container :deep(.table-wrapper) {
  @apply flex-1 overflow-x-auto overflow-y-auto;
  /* 确保横向滚动条显示在最底部 */
  scrollbar-gutter: stable;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  background: var(--apple-surface-elevated);
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply border-b px-4 py-3 text-left text-xs font-semibold uppercase text-gray-500 dark:text-dark-300;
  border-color: var(--apple-border);
  color: var(--apple-muted);
  letter-spacing: 0;
}

.table-scroll-container :deep(td) {
  @apply border-b px-4 py-3 text-sm text-gray-700 dark:text-gray-300;
  border-color: var(--apple-border-soft);
  color: var(--apple-text);
}

.layout-section-pagination {
  @apply overflow-hidden rounded-lg border;
  background: var(--apple-surface);
  border-color: var(--apple-border);
  box-shadow: none;
}

.table-page-layout.mobile-mode {
  @apply gap-3;
  height: auto;
}

.table-page-layout.mobile-mode .table-scroll-container {
  @apply h-auto overflow-hidden border-none bg-transparent shadow-none;
}

.table-page-layout.mobile-mode .layout-section-scrollable {
  @apply flex-none min-h-fit min-w-0;
}

.table-page-layout.mobile-mode .layout-section-panel {
  @apply p-3;
  border-color: var(--apple-border-soft);
}

.table-page-layout.mobile-mode .layout-section-pagination {
  @apply border-none bg-transparent shadow-none;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(.table-wrapper) {
  @apply overflow-x-auto overflow-y-hidden;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-x: contain;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none min-w-max;
  display: table;
}
</style>
