<template>
  <div class="table-page-layout" :class="{ 'mobile-mode': isMobile }">
    <div v-if="$slots.actions" class="layout-section-fixed layout-section-panel">
      <slot name="actions" />
    </div>

    <div v-if="$slots.filters" class="layout-section-fixed layout-section-panel">
      <slot name="filters" />
    </div>

    <div class="layout-section-scrollable">
      <div class="card table-scroll-container">
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
  @apply rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-900;
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
  @apply flex h-full min-w-0 flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900;
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
  @apply bg-gray-50/95 dark:bg-dark-800/95;
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply border-b border-gray-200 px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:text-dark-300;
}

.table-scroll-container :deep(td) {
  @apply border-b border-gray-100 px-4 py-3 text-sm text-gray-700 dark:border-dark-800 dark:text-gray-300;
}

.layout-section-pagination {
  @apply overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900;
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
