<template>
  <AppLayout>
    <div class="admin-apple-page admin-table-page">
      <TablePageLayout>
      <template #filters>
        <MonitorFiltersBar
          v-model:search="searchQuery"
          v-model:provider="providerFilter"
          v-model:enabled="enabledFilter"
          :loading="loading"
          @reload="reload"
          @create="openCreateDialog"
          @manage-sort="openSortModal"
          @manage-templates="showTemplateManager = true"
          @search-input="handleSearch"
        />
      </template>

      <template #table>
        <DataTable :columns="columns" :data="monitors" :loading="loading">
          <template #cell-sort_order="{ value }">
            <span class="inline-flex min-w-10 justify-center rounded-md bg-gray-100 px-2 py-1 text-xs font-semibold text-gray-700 dark:bg-dark-800 dark:text-gray-200">
              {{ value ?? 0 }}
            </span>
          </template>

          <template #cell-name="{ row, value }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <HelpTooltip v-if="row.api_key_decrypt_failed" :content="t('admin.channelMonitor.apiKeyDecryptFailed')">
                <Icon name="exclamationTriangle" size="sm" class="text-red-500" />
              </HelpTooltip>
            </div>
          </template>

          <template #cell-provider="{ row }">
            <span class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="providerBadgeClass(row.provider)">
              {{ providerLabel(row.provider) }}
            </span>
          </template>

          <template #cell-primary_model="{ row }">
            <MonitorPrimaryModelCell :row="row" />
          </template>

          <template #cell-availability_7d="{ row }">
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ formatAvailability(row) }}</span>
          </template>

          <template #cell-latency="{ row }">
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ formatLatency(row.primary_latency_ms) }}</span>
          </template>

          <template #cell-enabled="{ row }">
            <Toggle :modelValue="row.enabled" @update:modelValue="toggleEnabled(row)" />
          </template>

          <template #cell-actions="{ row }">
            <MonitorActionsCell
              :row="row"
              :running="runningId === row.id"
              :duplicating="false"
              @run="handleRunNow"
              @edit="openEditDialog"
              @delete="handleDelete"
            />
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.channelMonitor.noMonitorsYet')"
              :description="t('admin.channelMonitor.createFirstMonitor')"
              :action-text="t('admin.channelMonitor.createButton')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <MonitorFormDialog
      :show="showDialog"
      :monitor="editing"
      @close="closeDialog"
      @saved="reload"
    />

    <MonitorTemplateManagerDialog
      :show="showTemplateManager"
      @close="showTemplateManager = false"
      @updated="reload"
    />

    <MonitorRunResultDialog
      :show="showRunResult"
      :results="runResults"
      @close="showRunResult = false"
    />

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <BaseDialog
      :show="showSortModal"
      :title="t('admin.channelMonitor.sortOrder')"
      width="normal"
      @close="closeSortModal"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.channelMonitor.sortOrderHint') }}
        </p>
        <VueDraggable
          v-model="sortableMonitors"
          :animation="200"
          class="max-h-[60vh] space-y-2 overflow-y-auto pr-1"
        >
          <div
            v-for="monitor in sortableMonitors"
            :key="monitor.id"
            class="flex cursor-grab items-center gap-3 rounded-lg border border-gray-200 bg-white p-3 transition-shadow hover:shadow-md active:cursor-grabbing dark:border-dark-600 dark:bg-dark-700"
          >
            <div class="text-gray-400">
              <Icon name="menu" size="md" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium text-gray-900 dark:text-white">
                {{ monitor.name }}
              </div>
              <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span
                  class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
                  :class="providerBadgeClass(monitor.provider)"
                >
                  {{ providerLabel(monitor.provider) }}
                </span>
                <span class="truncate font-mono">{{ monitor.primary_model }}</span>
                <span v-if="monitor.group_name" class="truncate">{{ monitor.group_name }}</span>
              </div>
            </div>
            <div class="text-sm text-gray-400">#{{ monitor.id }}</div>
          </div>
        </VueDraggable>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button @click="closeSortModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            @click="saveSortOrder"
            :disabled="sortSubmitting"
            class="btn btn-primary"
          >
            <svg
              v-if="sortSubmitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ sortSubmitting ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type {
  ChannelMonitor,
  CheckResult,
  ListParams,
  Provider,
} from '@/custom/api/admin/channelMonitor'
import { channelMonitorAPI } from '@/custom/api/admin/channelMonitor'
import type { Column } from '@/components/common/types'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import TablePageLayout from '@/custom/layout/WegooTablePageLayout.vue'
import DataTable from '@/custom/common/WegooDataTable.vue'
import Pagination from '@/custom/common/WegooPagination.vue'
import BaseDialog from '@/custom/common/WegooBaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/custom/common/WegooEmptyState.vue'
import HelpTooltip from '@/custom/common/WegooHelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import MonitorFiltersBar from '@/custom/admin/monitor/WegooMonitorFiltersBar.vue'
import MonitorFormDialog from '@/custom/admin/monitor/WegooMonitorFormDialog.vue'
import MonitorTemplateManagerDialog from '@/components/admin/monitor/MonitorTemplateManagerDialog.vue'
import MonitorRunResultDialog from '@/components/admin/monitor/MonitorRunResultDialog.vue'
import MonitorPrimaryModelCell from '@/components/admin/monitor/MonitorPrimaryModelCell.vue'
import MonitorActionsCell from '@/components/admin/monitor/MonitorActionsCell.vue'
import { getPersistedPageSize } from '@/custom/composables/usePersistedPageSize'
import { useChannelMonitorFormat } from '@/custom/composables/useChannelMonitorFormat'
import '@/custom/admin/adminApple.css'

const { t } = useI18n()
const appStore = useAppStore()
const {
  providerLabel,
  providerBadgeClass,
  formatLatency,
  formatAvailability,
} = useChannelMonitorFormat()

const monitors = ref<ChannelMonitor[]>([])
const loading = ref(false)
const runningId = ref<number | null>(null)
const searchQuery = ref('')
const providerFilter = ref<Provider | ''>('')
const enabledFilter = ref<'' | 'true' | 'false'>('')
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const showDialog = ref(false)
const showTemplateManager = ref(false)
const editing = ref<ChannelMonitor | null>(null)
const showDeleteDialog = ref(false)
const deleting = ref<ChannelMonitor | null>(null)
const showRunResult = ref(false)
const runResults = ref<CheckResult[]>([])
const showSortModal = ref(false)
const sortSubmitting = ref(false)
const sortableMonitors = ref<ChannelMonitor[]>([])

let abortController: AbortController | null = null
let searchTimeout: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => [
  { key: 'sort_order', label: t('admin.channelMonitor.columns.sortOrder'), sortable: false },
  { key: 'name', label: t('admin.channelMonitor.columns.name'), sortable: false },
  { key: 'provider', label: t('admin.channelMonitor.columns.provider'), sortable: false },
  { key: 'primary_model', label: t('admin.channelMonitor.columns.primaryModel'), sortable: false },
  { key: 'availability_7d', label: t('admin.channelMonitor.columns.availability7d'), sortable: false },
  { key: 'latency', label: t('admin.channelMonitor.columns.latency'), sortable: false },
  { key: 'enabled', label: t('admin.channelMonitor.columns.enabled'), sortable: false },
  { key: 'actions', label: t('admin.channelMonitor.columns.actions'), sortable: false },
])

const deleteConfirmMessage = computed(() => {
  const name = deleting.value?.name || ''
  return t('admin.channelMonitor.deleteConfirm', { name })
})

async function reload() {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  loading.value = true
  try {
    const params: ListParams = {
      page: pagination.page,
      page_size: pagination.page_size,
    }
    if (providerFilter.value) params.provider = providerFilter.value
    if (enabledFilter.value === 'true') params.enabled = true
    if (enabledFilter.value === 'false') params.enabled = false
    if (searchQuery.value.trim()) params.search = searchQuery.value.trim()

    const res = await channelMonitorAPI.list(params, { signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    monitors.value = res.items || []
    pagination.total = res.total
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.loadError')))
  } finally {
    if (abortController === ctrl) {
      loading.value = false
      abortController = null
    }
  }
}

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    reload()
  }, 300)
}

function onPageChange(page: number) {
  pagination.page = page
  reload()
}

function onPageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function openCreateDialog() {
  editing.value = null
  showDialog.value = true
}

function openEditDialog(row: ChannelMonitor) {
  editing.value = row
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editing.value = null
}

async function toggleEnabled(row: ChannelMonitor) {
  const next = !row.enabled
  try {
    await channelMonitorAPI.update(row.id, { enabled: next })
    row.enabled = next
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

async function openSortModal() {
  try {
    const allMonitors = await channelMonitorAPI.listAll()
    sortableMonitors.value = [...allMonitors].sort(
      (a, b) => (a.sort_order || 0) - (b.sort_order || 0) || b.id - a.id,
    )
    showSortModal.value = true
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.failedToLoadSortOrder')))
  }
}

function closeSortModal() {
  showSortModal.value = false
  sortableMonitors.value = []
}

async function saveSortOrder() {
  sortSubmitting.value = true
  try {
    const updates = sortableMonitors.value.map((monitor, index) => ({
      id: monitor.id,
      sort_order: index * 10,
    }))
    await channelMonitorAPI.updateSortOrder(updates)
    appStore.showSuccess(t('admin.channelMonitor.sortOrderUpdated'))
    closeSortModal()
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.failedToUpdateSortOrder')))
  } finally {
    sortSubmitting.value = false
  }
}

async function handleRunNow(row: ChannelMonitor) {
  if (runningId.value != null) return
  runningId.value = row.id
  try {
    const res = await channelMonitorAPI.runNow(row.id)
    runResults.value = res.results || []
    showRunResult.value = true
    appStore.showSuccess(t('admin.channelMonitor.runSuccess'))
    // Refresh row to get latest status from backend
    void reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.runFailed')))
  } finally {
    runningId.value = null
  }
}

function handleDelete(row: ChannelMonitor) {
  deleting.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  try {
    await channelMonitorAPI.del(deleting.value.id)
    appStore.showSuccess(t('admin.channelMonitor.deleteSuccess'))
    showDeleteDialog.value = false
    deleting.value = null
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

onMounted(reload)
onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
  abortController?.abort()
})
</script>
