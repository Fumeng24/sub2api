<template>
  <AppLayout>
    <div class="admin-apple-page admin-table-page">
    <TablePageLayout>
      <template #filters>
        <div class="space-y-3">
          <div class="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-8">
            <div
              v-for="item in statsCards"
              :key="item.key"
              class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900"
            >
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</div>
              <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-64">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.tickets.searchPlaceholder')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>

            <Select v-model="filters.status" :options="statusFilterOptions" class="w-full sm:w-36" @change="applyFilters" />
            <Select v-model="filters.priority" :options="priorityFilterOptions" class="w-full sm:w-36" @change="applyFilters" />
            <Select v-model="filters.category" :options="categoryFilterOptions" class="w-full sm:w-40" @change="applyFilters" />
            <Select v-model="filters.assignee_id" :options="assigneeFilterOptions" class="w-full sm:w-44" @change="applyFilters" />
            <Select v-model="filters.queue" :options="queueFilterOptions" class="w-full sm:w-44" @change="applyFilters" />
            <Select v-model="filters.unread_only" :options="unreadFilterOptions" class="w-full sm:w-36" @change="applyFilters" />

            <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
              <button
                class="btn btn-secondary"
                :disabled="loading"
                :title="t('common.refresh')"
                @click="refreshAll"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button v-if="capabilities?.is_super_admin" class="btn btn-secondary" @click="showAutoCloseDialog = true">
                <Icon name="clock" size="md" class="mr-1" />
                {{ t('admin.tickets.autoClose.button') }}
              </button>
            </div>
          </div>

          <div
            v-if="selectedCount > 0 && canBatchUpdate"
            class="flex flex-wrap items-center gap-3 rounded-lg border border-primary-200 bg-primary-50 p-3 dark:border-primary-900/40 dark:bg-primary-900/20"
          >
            <span class="text-sm font-medium text-primary-700 dark:text-primary-200">
              {{ t('admin.tickets.bulk.selected', { count: selectedCount }) }}
            </span>
            <Select v-model="bulkForm.status" :options="bulkStatusOptions" class="w-full sm:w-36" />
            <Select v-if="canUpdatePriority" v-model="bulkForm.priority" :options="bulkPriorityOptions" class="w-full sm:w-36" />
            <Select v-if="canUpdateCategory" v-model="bulkForm.category" :options="bulkCategoryOptions" class="w-full sm:w-40" />
            <Select v-if="canTransfer" v-model="bulkForm.assignee_id" :options="assigneeUpdateOptions" class="w-full sm:w-52" searchable />
            <div class="ml-auto flex items-center gap-2">
              <button class="btn btn-secondary" @click="clearSelection">{{ t('common.cancel') }}</button>
              <button class="btn btn-primary" :disabled="bulkUpdating || !hasBulkChanges" @click="handleBulkUpdate">
                {{ bulkUpdating ? t('common.processing') : t('admin.tickets.bulk.apply') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="tickets"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="last_message_at"
          default-sort-order="desc"
          row-key="id"
          @sort="handleSort"
        >
          <template #header-select>
            <input
              v-if="canBatchUpdate"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allPageSelected"
              :indeterminate="somePageSelected && !allPageSelected"
              @change="togglePageSelection(eventChecked($event))"
            />
          </template>

          <template #cell-select="{ row }">
            <input
              v-if="canBatchUpdate"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isSelected(row.id)"
              @change="toggleSelection(row.id, eventChecked($event))"
            />
          </template>

          <template #cell-subject="{ value, row }">
            <button class="max-w-[360px] text-left" @click="openDetail(row)">
              <span class="block truncate font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400">
                {{ value }}
              </span>
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">
                {{ row.ticket_no }}
              </span>
            </button>
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.user_name || row.user_email }}</div>
              <div class="truncate text-xs text-gray-500 dark:text-dark-400">{{ row.user_email }}</div>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass(value)]">{{ statusLabel(value) }}</span>
          </template>

          <template #cell-priority="{ value }">
            <span :class="['badge', priorityBadgeClass(value)]">{{ priorityLabel(value) }}</span>
          </template>

          <template #cell-category="{ value }">
            <span class="badge badge-gray">{{ categoryLabel(value) }}</span>
          </template>

          <template #cell-assignee_id="{ value }">
            <span v-if="value" class="badge badge-primary">{{ assigneeLabel(value) }}</span>
            <span v-else class="badge badge-gray">{{ t('admin.tickets.unassigned') }}</span>
          </template>

          <template #cell-last_message_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-unread_count="{ value }">
            <span v-if="value > 0" class="badge badge-danger">{{ t('admin.tickets.unreadCount', { count: value }) }}</span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{ t('admin.tickets.noUnread') }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
              @click="openDetail(row)"
            >
              <Icon name="chatBubble" size="sm" />
              {{ t('admin.tickets.viewDetail') }}
            </button>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.tickets.empty')"
              :description="t('admin.tickets.emptyDescription')"
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
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
    </div>

    <BaseDialog
      :show="showDetailDialog"
      :title="detailTitle"
      width="extra-wide"
      @close="closeDetailDialog"
    >
      <div v-if="selectedTicket" class="space-y-5">
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span :class="['badge', statusBadgeClass(selectedTicket.status)]">{{ statusLabel(selectedTicket.status) }}</span>
                <span :class="['badge', priorityBadgeClass(selectedTicket.priority)]">{{ priorityLabel(selectedTicket.priority) }}</span>
                <span class="badge badge-gray">{{ categoryLabel(selectedTicket.category) }}</span>
                <span v-if="selectedTicket.template_key" class="badge badge-gray">{{ selectedTicket.template_key }}</span>
                <span v-if="selectedTicket.escalated_at" class="badge badge-danger">
                  {{ t('admin.tickets.escalated') }}
                </span>
                <span v-if="selectedTicket.unread_count > 0" class="badge badge-danger">
                  {{ t('admin.tickets.unreadCount', { count: selectedTicket.unread_count }) }}
                </span>
              </div>
              <h3 class="mt-3 break-words text-base font-semibold text-gray-900 dark:text-white">
                {{ selectedTicket.subject }}
              </h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                {{ selectedTicket.ticket_no }} · {{ selectedTicket.user_name || selectedTicket.user_email }}
              </p>
              <div class="mt-2 flex flex-wrap items-center gap-2">
                <TicketContextLink
                  :context-type="selectedTicket.context_type"
                  :context-id="selectedTicket.context_id"
                  admin
                />
              </div>
              <div v-if="contextEntries.length > 0" class="mt-3 grid grid-cols-1 gap-2 text-xs md:grid-cols-2">
                <div
                  v-for="item in contextEntries"
                  :key="item.key"
                  class="rounded-md bg-white px-3 py-2 text-gray-600 dark:bg-dark-900 dark:text-gray-300"
                >
                  <span class="font-medium text-gray-900 dark:text-white">{{ item.key }}:</span>
                  <a
                    v-if="item.image"
                    :href="item.value"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="mt-2 block"
                  >
                    <img
                      :src="item.value"
                      :alt="item.key"
                      class="max-h-36 max-w-full rounded-md border border-gray-200 object-contain dark:border-dark-700"
                    />
                    <span class="mt-1 inline-flex items-center gap-1 text-primary-600 dark:text-primary-400">
                      <Icon name="externalLink" size="xs" />
                      {{ t('admin.tickets.viewImage') }}
                    </span>
                  </a>
                  <span v-else class="ml-1 break-words">{{ item.value }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button
            v-if="canClaimSelectedTicket"
            class="btn btn-secondary"
            :disabled="claiming"
            @click="handleClaimTicket"
          >
            {{ claiming ? t('common.processing') : t('admin.tickets.claim') }}
          </button>
          <button
            v-if="canEscalateSelectedTicket"
            class="btn btn-secondary"
            :disabled="escalating"
            @click="handleEscalateTicket"
          >
            {{ escalating ? t('common.processing') : t('admin.tickets.escalate') }}
          </button>
        </div>

        <form v-if="canEditSelectedTicket" class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700" @submit.prevent="handleSaveTicket">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div v-if="canCloseTicket">
              <label class="input-label">{{ t('admin.tickets.form.status') }}</label>
              <Select v-model="ticketForm.status" :options="statusOptions" />
            </div>
            <div v-if="canUpdatePriority">
              <label class="input-label">{{ t('admin.tickets.form.priority') }}</label>
              <Select v-model="ticketForm.priority" :options="priorityOptions" />
            </div>
            <div v-if="canUpdateCategory">
              <label class="input-label">{{ t('admin.tickets.form.category') }}</label>
              <Select v-model="ticketForm.category" :options="categoryOptions" />
            </div>
            <div v-if="canTransfer">
              <label class="input-label">{{ t('admin.tickets.form.assigneeId') }}</label>
              <Select v-model="ticketForm.assignee_id" :options="assigneeUpdateOptions" searchable />
            </div>
          </div>

          <div class="flex justify-end">
            <button type="submit" class="btn btn-primary" :disabled="savingTicket || !hasTicketFormChanges">
              {{ savingTicket ? t('common.saving') : t('admin.tickets.saveChanges') }}
            </button>
          </div>
        </form>

        <form
          v-if="canAdjustBalance"
          class="space-y-3 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900/40 dark:bg-emerald-900/20"
          @submit.prevent="handleBalanceAdjust"
        >
          <div class="grid grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,2fr)_auto]">
            <div>
              <label class="input-label">{{ t('admin.tickets.balance.operation') }}</label>
              <Select
                v-model="balanceForm.operation"
                :options="balanceOperationOptions"
                @change="balanceForm.business_category = defaultBalanceBusinessCategory(balanceForm.operation)"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.tickets.balance.amount') }}</label>
              <input v-model.number="balanceForm.amount" type="number" min="0" step="0.01" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.users.businessCategory') }}</label>
              <Select v-model="balanceForm.business_category" :options="balanceBusinessCategoryOptions" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.tickets.balance.notes') }}</label>
              <input v-model="balanceForm.notes" type="text" class="input" :placeholder="t('admin.tickets.balance.notesPlaceholder')" />
            </div>
            <div class="flex items-end">
              <button type="submit" class="btn btn-primary" :disabled="adjustingBalance || balanceForm.amount <= 0">
                {{ adjustingBalance ? t('common.processing') : t('admin.tickets.balance.apply') }}
              </button>
            </div>
          </div>
        </form>

        <div class="max-h-[44vh] space-y-3 overflow-y-auto pr-1">
          <div v-if="detailLoading" class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('common.loading') }}
          </div>
          <template v-else>
            <div
              v-for="message in selectedTicket.messages || []"
              :key="message.id"
              class="flex"
              :class="message.sender_type === 'admin' ? 'justify-end' : 'justify-start'"
            >
              <div
                :class="[
                  'max-w-[85%] rounded-lg border px-4 py-3 text-sm',
                  message.visibility === 'internal'
                    ? 'border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-100'
                    : message.sender_type === 'admin'
                      ? 'border-primary-200 bg-primary-50 text-primary-950 dark:border-primary-900/40 dark:bg-primary-900/20 dark:text-primary-100'
                      : 'border-gray-200 bg-white text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200'
                ]"
              >
                <div class="mb-2 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                  <span class="font-medium">{{ message.visibility === 'internal' ? t('admin.tickets.internalNote') : senderLabel(message) }}</span>
                  <span>{{ formatDateTime(message.created_at) }}</span>
                </div>
                <p class="whitespace-pre-wrap break-words leading-6">{{ message.body }}</p>
                <TicketAttachments :attachments="message.attachments" />
              </div>
            </div>
          </template>
        </div>

        <form v-if="canReplySelectedTicket" class="space-y-3 rounded-lg bg-gray-50 p-4 dark:bg-dark-800" @submit.prevent="handleReply">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <label class="input-label mb-0">{{ t('admin.tickets.reply') }}</label>
            <label v-if="canInternalNote" class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="replyInternal" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span>{{ t('admin.tickets.internalReply') }}</span>
            </label>
          </div>
          <textarea
            v-model="replyBody"
            rows="4"
            class="input"
            :placeholder="t('admin.tickets.replyPlaceholder')"
          ></textarea>
          <TicketAttachmentFields v-model="replyAttachments" :i18n-prefix="'admin.tickets.replyAttachments'" />
          <div class="flex justify-end">
            <button type="submit" class="btn btn-primary" :disabled="replying || !replyBody.trim()">
              {{ replying ? t('common.saving') : t('admin.tickets.sendReply') }}
            </button>
          </div>
        </form>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="showAutoCloseDialog"
      :title="t('admin.tickets.autoClose.title')"
      width="narrow"
      @close="showAutoCloseDialog = false"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.tickets.autoClose.description') }}</p>
        <div>
          <label class="input-label">{{ t('admin.tickets.autoClose.days') }}</label>
          <input v-model.number="autoCloseDays" type="number" min="1" max="365" class="input" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="showAutoCloseDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="autoClosing" @click="handleAutoClose">
            {{ autoClosing ? t('common.processing') : t('admin.tickets.autoClose.confirm') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/custom/api/admin'
import adminTicketsAPI from '@/custom/api/admin/tickets'
import { useAppStore, useAuthStore } from '@/stores'
import { useTicketStore } from '@/custom/stores/tickets'
import { getPersistedPageSize } from '@/custom/composables/usePersistedPageSize'
import { extractI18nErrorMessage } from '@/utils/apiError'
import {
  balanceBusinessCategoryOptions,
  defaultBalanceBusinessCategory
} from '@/custom/utils/balanceBusinessCategory'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import type { AdminUser, BalanceBusinessCategory, Ticket, TicketAdminCapabilities, TicketAttachment, TicketCategory, TicketMessage, TicketPriority, TicketStats, TicketStatus } from '@/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import TicketAttachmentFields from '@/custom/tickets/TicketAttachmentFields.vue'
import TicketAttachments from '@/custom/tickets/TicketAttachments.vue'
import TicketContextLink from '@/custom/tickets/TicketContextLink.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const ticketStore = useTicketStore()

const tickets = ref<Ticket[]>([])
const adminUsers = ref<AdminUser[]>([])
const ticketStats = ref<TicketStats | null>(null)
const capabilities = ref<TicketAdminCapabilities | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const savingTicket = ref(false)
const bulkUpdating = ref(false)
const replying = ref(false)
const autoClosing = ref(false)
const claiming = ref(false)
const escalating = ref(false)
const adjustingBalance = ref(false)
const showDetailDialog = ref(false)
const showAutoCloseDialog = ref(false)
const selectedTicket = ref<Ticket | null>(null)
const replyBody = ref('')
const replyAttachments = ref<TicketAttachment[]>([])
const replyInternal = ref(false)
const searchQuery = ref('')
const selectedIds = ref<Set<number>>(new Set())
const autoCloseDays = ref(7)
let searchTimer: number | undefined

const filters = reactive({
  status: '',
  priority: '',
  category: '',
  assignee_id: '',
  queue: '',
  unread_only: false
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const sortState = reactive({
  sort_by: 'last_message_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const ticketForm = reactive({
  status: 'open',
  priority: 'normal',
  category: 'general',
  assignee_id: '0'
})

const bulkForm = reactive({
  status: '',
  priority: '',
  category: '',
  assignee_id: ''
})

const balanceForm = reactive({
  operation: 'add' as 'set' | 'add' | 'subtract',
  amount: 0,
  notes: '',
  business_category: 'manual_collection' as BalanceBusinessCategory
})

const currentAdminId = computed(() => authStore.user?.id ?? 0)
const isSuperAdmin = computed(() => authStore.isAdmin)
const isSupportAgent = computed(() => authStore.isSupport)
const canViewAll = computed(() => Boolean(capabilities.value?.can_view_all || capabilities.value?.is_super_admin))
const canViewEscalated = computed(() => Boolean(capabilities.value?.can_view_escalated || capabilities.value?.is_super_admin))
const canInternalNote = computed(() => Boolean(capabilities.value?.can_internal_note || capabilities.value?.is_super_admin))
const canCloseTicket = computed(() => Boolean(capabilities.value?.can_close || capabilities.value?.is_super_admin))
const canTransfer = computed(() => Boolean(capabilities.value?.can_transfer || capabilities.value?.is_super_admin))
const canBatchUpdate = computed(() => Boolean(capabilities.value?.can_batch_update || capabilities.value?.is_super_admin))
const canUpdatePriority = computed(() => Boolean(capabilities.value?.can_update_priority || capabilities.value?.is_super_admin))
const canUpdateCategory = computed(() => Boolean(capabilities.value?.can_update_category || capabilities.value?.is_super_admin))
const canEscalate = computed(() => Boolean(capabilities.value?.can_escalate || capabilities.value?.is_super_admin))
const canAdjustBalance = computed(() => Boolean(capabilities.value?.can_adjust_balance || capabilities.value?.is_super_admin))

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', class: 'w-10' },
  { key: 'subject', label: t('admin.tickets.columns.subject'), sortable: true, class: 'min-w-[260px]' },
  { key: 'user', label: t('admin.tickets.columns.user') },
  { key: 'status', label: t('admin.tickets.columns.status'), sortable: true },
  { key: 'priority', label: t('admin.tickets.columns.priority'), sortable: true },
  { key: 'category', label: t('admin.tickets.columns.category'), sortable: true },
  { key: 'assignee_id', label: t('admin.tickets.columns.assignee'), sortable: true },
  { key: 'last_message_at', label: t('admin.tickets.columns.lastMessageAt'), sortable: true },
  { key: 'unread_count', label: t('admin.tickets.columns.unread') },
  { key: 'actions', label: t('common.actions') }
])

const detailTitle = computed(() => selectedTicket.value?.ticket_no || t('admin.tickets.detailTitle'))

const statusOptions = computed(() => [
  { value: 'open', label: t('admin.tickets.status.open') },
  { value: 'pending', label: t('admin.tickets.status.pending') },
  { value: 'resolved', label: t('admin.tickets.status.resolved') },
  { value: 'closed', label: t('admin.tickets.status.closed') }
])

const priorityOptions = computed(() => [
  { value: 'low', label: t('admin.tickets.priority.low') },
  { value: 'normal', label: t('admin.tickets.priority.normal') },
  { value: 'high', label: t('admin.tickets.priority.high') },
  { value: 'urgent', label: t('admin.tickets.priority.urgent') }
])

const categoryOptions = computed(() => [
  { value: 'general', label: t('admin.tickets.category.general') },
  { value: 'billing', label: t('admin.tickets.category.billing') },
  { value: 'usage', label: t('admin.tickets.category.usage') },
  { value: 'technical', label: t('admin.tickets.category.technical') },
  { value: 'account', label: t('admin.tickets.category.account') }
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('admin.tickets.filters.allStatus') },
  ...statusOptions.value
])

const priorityFilterOptions = computed(() => [
  { value: '', label: t('admin.tickets.filters.allPriority') },
  ...priorityOptions.value
])

const categoryFilterOptions = computed(() => [
  { value: '', label: t('admin.tickets.filters.allCategory') },
  ...categoryOptions.value
])

const unreadFilterOptions = computed(() => [
  { value: false, label: t('admin.tickets.filters.allUnread') },
  { value: true, label: t('admin.tickets.filters.onlyUnread') }
])

const queueFilterOptions = computed(() => {
  if (isSupportAgent.value) {
    const options = [
      { value: '', label: t('admin.tickets.queue.support') },
      { value: 'mine', label: t('admin.tickets.queue.mine') }
    ]
    if (canViewAll.value) {
      options.push({ value: 'all', label: t('admin.tickets.queue.allNormal') })
    }
    if (canViewEscalated.value) {
      options.push({ value: 'super_admin', label: t('admin.tickets.queue.superAdmin') })
    }
    return options
  }
  return [
    { value: '', label: t('admin.tickets.queue.all') },
    { value: 'mine', label: t('admin.tickets.queue.mine') },
    { value: 'support', label: t('admin.tickets.queue.support') },
    { value: 'super_admin', label: t('admin.tickets.queue.superAdmin') }
  ]
})

const balanceOperationOptions = computed(() => [
  { value: 'add', label: t('admin.tickets.balance.add') },
  { value: 'subtract', label: t('admin.tickets.balance.subtract') },
  { value: 'set', label: t('admin.tickets.balance.set') }
])

const bulkStatusOptions = computed(() => [
  { value: '', label: t('admin.tickets.bulk.keepStatus') },
  ...statusOptions.value
])

const bulkPriorityOptions = computed(() => [
  { value: '', label: t('admin.tickets.bulk.keepPriority') },
  ...priorityOptions.value
])

const bulkCategoryOptions = computed(() => [
  { value: '', label: t('admin.tickets.bulk.keepCategory') },
  ...categoryOptions.value
])

const assigneeFilterOptions = computed(() => {
  const options = [
    { value: '', label: t('admin.tickets.filters.allAssignees') },
    { value: 'unassigned', label: t('admin.tickets.filters.unassigned') }
  ]
  if (currentAdminId.value > 0) {
    options.push({ value: String(currentAdminId.value), label: t('admin.tickets.filters.assignedToMe') })
  }
  return options
})

const supportAssigneeOptions = computed(() =>
  adminUsers.value
    .filter((user) => user.role === 'admin' || user.role === 'support')
    .map((user) => ({
      value: String(user.id),
      label: `${user.username || user.email} (#${user.id})`
    }))
)

const assigneeUpdateOptions = computed(() => {
  const options = [
    { value: '0', label: t('admin.tickets.unassigned') },
    ...supportAssigneeOptions.value.map((user) => ({
      value: user.value,
      label: user.label
    }))
  ]
  const selected = selectedTicket.value?.assignee_id
  if (selected && !options.some((item) => item.value === String(selected))) {
    options.push({ value: String(selected), label: `#${selected}` })
  }
  return options
})

const selectedCount = computed(() => selectedIds.value.size)
const allPageSelected = computed(() => tickets.value.length > 0 && tickets.value.every((ticket) => selectedIds.value.has(ticket.id)))
const somePageSelected = computed(() => tickets.value.some((ticket) => selectedIds.value.has(ticket.id)))
const hasBulkChanges = computed(() => !!(bulkForm.status || bulkForm.priority || bulkForm.category || bulkForm.assignee_id))

const canClaimSelectedTicket = computed(() => {
  if (!selectedTicket.value || currentAdminId.value <= 0) return false
  if (selectedTicket.value.assignee_id === currentAdminId.value) return false
  if (selectedTicket.value.escalated_at && !isSuperAdmin.value) return false
  if (isSupportAgent.value && selectedTicket.value.assignee_id && !canTransfer.value) return false
  return true
})

const canReplySelectedTicket = computed(() => {
  const ticket = selectedTicket.value
  if (!ticket) return false
  if (ticket.escalated_at && !isSuperAdmin.value) return false
  if (isSuperAdmin.value) return true
  if (!isSupportAgent.value) return false
  if (ticket.assignee_id === currentAdminId.value) {
    return Boolean(capabilities.value?.can_reply_assigned_to_self)
  }
  if (!ticket.assignee_id) {
    return Boolean(capabilities.value?.can_reply_unassigned || canViewAll.value)
  }
  return canViewAll.value
})

const canEscalateSelectedTicket = computed(() => {
  const ticket = selectedTicket.value
  if (!ticket || ticket.escalated_at) return false
  if (isSuperAdmin.value) return true
  return canEscalate.value && canReplySelectedTicket.value
})

const canEditSelectedTicket = computed(() => {
  if (!selectedTicket.value) return false
  if (selectedTicket.value.escalated_at && !isSuperAdmin.value) return false
  return canCloseTicket.value || canUpdatePriority.value || canUpdateCategory.value || canTransfer.value
})

const hasTicketFormChanges = computed(() => {
  const ticket = selectedTicket.value
  if (!ticket) return false
  if (canCloseTicket.value && ticketForm.status !== ticket.status) return true
  if (canUpdatePriority.value && ticketForm.priority !== ticket.priority) return true
  if (canUpdateCategory.value && ticketForm.category !== ticket.category) return true
  if (canTransfer.value) {
    const assignee = ticket.assignee_id && ticket.assignee_id > 0 ? String(ticket.assignee_id) : '0'
    if (ticketForm.assignee_id !== assignee) return true
  }
  return false
})

const contextEntries = computed(() => {
  const data = selectedTicket.value?.context_data
  if (!data || typeof data !== 'object') return []
  return Object.entries(data)
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .slice(0, 12)
    .map(([key, value]) => ({
      key,
      value: stringifyContextValue(value),
      image: typeof value === 'string' && isImageReference(value)
    }))
})

function stringifyContextValue(value: unknown): string {
  if (Array.isArray(value)) {
    if (value.length > 5) return `${value.length} items`
    return value.map((item) => stringifyContextValue(item)).join(', ')
  }
  if (typeof value === 'object' && value !== null) {
    const record = value as Record<string, unknown>
    if ('id' in record) return `#${String(record.id)}`
    return JSON.stringify(value)
  }
  return String(value)
}

function isImageReference(value: string) {
  return /^data:image\/(?:png|jpe?g|webp|gif);base64,/i.test(value) || /\.(png|jpe?g|webp|gif)(\?|#|$)/i.test(value)
}

const statsCards = computed(() => {
  const stats = ticketStats.value || {
    total: 0,
    open: 0,
    pending: 0,
    resolved: 0,
    closed: 0,
    unassigned: 0,
    assigned_to_me: 0,
    handled_by_me: 0,
    escalated: 0,
    sla_overdue: 0,
    unread: 0
  }
  return [
    { key: 'total', label: t('admin.tickets.stats.total'), value: stats.total },
    { key: 'open', label: t('admin.tickets.status.open'), value: stats.open },
    { key: 'pending', label: t('admin.tickets.status.pending'), value: stats.pending },
    { key: 'resolved', label: t('admin.tickets.status.resolved'), value: stats.resolved },
    { key: 'closed', label: t('admin.tickets.status.closed'), value: stats.closed },
    { key: 'unassigned', label: t('admin.tickets.stats.unassigned'), value: stats.unassigned },
    { key: 'assigned_to_me', label: t('admin.tickets.stats.assignedToMe'), value: stats.assigned_to_me },
    { key: 'handled_by_me', label: t('admin.tickets.stats.handledByMe'), value: stats.handled_by_me },
    { key: 'escalated', label: t('admin.tickets.stats.escalated'), value: stats.escalated },
    { key: 'sla_overdue', label: t('admin.tickets.stats.slaOverdue'), value: stats.sla_overdue },
    { key: 'unread', label: t('admin.tickets.columns.unread'), value: stats.unread }
  ]
})

function buildListFilters() {
  const queue = filters.queue || undefined
  return {
    status: filters.status || undefined,
    priority: filters.priority || undefined,
    category: filters.category || undefined,
    assignee_id: filters.assignee_id || undefined,
    queue,
    escalated_only: queue === 'super_admin' || undefined,
    unread_only: filters.unread_only || undefined,
    search: searchQuery.value.trim() || undefined,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order
  }
}

async function fetchTickets() {
  loading.value = true
  try {
    const res = await adminTicketsAPI.list(pagination.page, pagination.page_size, buildListFilters())
    tickets.value = res.items || []
    pagination.total = res.total || 0
    pagination.pages = res.pages || 0
    pagination.page = res.page || pagination.page
    pagination.page_size = res.page_size || pagination.page_size
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToLoad')))
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  try {
    ticketStats.value = await adminTicketsAPI.getStats()
    await ticketStore.fetchUnreadSummary('admin', true)
  } catch (error: unknown) {
    console.error('Failed to load ticket stats:', error)
  }
}

async function fetchCapabilities() {
  try {
    capabilities.value = await adminTicketsAPI.getCapabilities()
    if (isSupportAgent.value && filters.queue === 'all' && !canViewAll.value) {
      filters.queue = ''
    }
    if (isSupportAgent.value && filters.queue === 'super_admin' && !canViewEscalated.value) {
      filters.queue = ''
    }
  } catch (error: unknown) {
    console.error('Failed to load ticket capabilities:', error)
  }
}

async function fetchAdminUsers() {
  try {
    const [admins, supports] = await Promise.all([
      adminAPI.users.list(1, 100, {
        role: 'admin',
        status: 'active',
        sort_by: 'id',
        sort_order: 'asc'
      }),
      adminAPI.users.list(1, 100, {
        role: 'support',
        status: 'active',
        sort_by: 'id',
        sort_order: 'asc'
      })
    ])
    adminUsers.value = [...(admins.items || []), ...(supports.items || [])]
  } catch (error: unknown) {
    try {
      const res = await adminAPI.users.list(1, 100, {
        status: 'active',
        sort_by: 'id',
        sort_order: 'asc'
      })
      adminUsers.value = (res.items || []).filter((user) => user.role === 'admin' || user.role === 'support')
    } catch (fallbackError: unknown) {
      console.error('Failed to load admin users:', fallbackError)
    }
  }
}

async function refreshAll() {
  await Promise.all([fetchCapabilities(), fetchTickets(), fetchStats()])
}

function handleSearch() {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    applyFilters()
  }, 300)
}

function applyFilters() {
  if (isSupportAgent.value && filters.queue === 'all' && !canViewAll.value) {
    filters.queue = ''
  }
  if (isSupportAgent.value && filters.queue === 'super_admin' && !canViewEscalated.value) {
    filters.queue = ''
  }
  pagination.page = 1
  fetchTickets()
}

function handlePageChange(page: number) {
  pagination.page = page
  fetchTickets()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  fetchTickets()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  fetchTickets()
}

function syncTicketForm(ticket: Ticket) {
  ticketForm.status = ticket.status
  ticketForm.priority = ticket.priority
  ticketForm.category = ticket.category
  ticketForm.assignee_id = ticket.assignee_id && ticket.assignee_id > 0 ? String(ticket.assignee_id) : '0'
}

async function openDetail(ticket: Ticket) {
  selectedTicket.value = { ...ticket, messages: ticket.messages || [] }
  showDetailDialog.value = true
  replyBody.value = ''
  replyAttachments.value = []
  replyInternal.value = false
  syncTicketForm(ticket)
  await loadTicketDetail(ticket.id)
}

async function loadTicketDetail(id: number) {
  detailLoading.value = true
  try {
    const detail = await adminTicketsAPI.getById(id)
    selectedTicket.value = detail
    syncTicketForm(detail)
    tickets.value = tickets.value.map((item) =>
      item.id === detail.id ? { ...item, ...detail, messages: item.messages, unread_count: 0 } : item
    )
    await fetchStats()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToLoadDetail')))
  } finally {
    detailLoading.value = false
  }
}

function closeDetailDialog() {
  showDetailDialog.value = false
  selectedTicket.value = null
  replyBody.value = ''
  replyAttachments.value = []
  replyInternal.value = false
}

async function handleSaveTicket() {
  if (!selectedTicket.value) return
  savingTicket.value = true
  try {
    const payload: Record<string, unknown> = {}
    if (canCloseTicket.value) {
      payload.status = ticketForm.status
    }
    if (canUpdatePriority.value) {
      payload.priority = ticketForm.priority
    }
    if (canUpdateCategory.value) {
      payload.category = ticketForm.category
    }
    if (canTransfer.value) {
      const assigneeValue = Number.parseInt(ticketForm.assignee_id || '0', 10)
      payload.assignee_id = Number.isFinite(assigneeValue) ? assigneeValue : 0
    }
    if (Object.keys(payload).length === 0) return
    const updated = await adminTicketsAPI.update(selectedTicket.value.id, payload)
    selectedTicket.value = updated
    syncTicketForm(updated)
    appStore.showSuccess(t('admin.tickets.updated'))
    await loadTicketDetail(updated.id)
    await fetchTickets()
    await fetchStats()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToUpdate')))
  } finally {
    savingTicket.value = false
  }
}

async function handleClaimTicket() {
  if (!selectedTicket.value) return
  claiming.value = true
  try {
    const updated = await adminTicketsAPI.claim(selectedTicket.value.id)
    selectedTicket.value = updated
    syncTicketForm(updated)
    appStore.showSuccess(t('admin.tickets.claimed'))
    await loadTicketDetail(updated.id)
    await refreshAll()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToUpdate')))
  } finally {
    claiming.value = false
  }
}

async function handleEscalateTicket() {
  if (!selectedTicket.value) return
  const reason = window.prompt(t('admin.tickets.escalateReasonPrompt'), '')
  if (reason === null) return
  escalating.value = true
  try {
    const updated = await adminTicketsAPI.escalate(selectedTicket.value.id, reason.trim())
    selectedTicket.value = updated
    syncTicketForm(updated)
    appStore.showSuccess(t('admin.tickets.escalatedDone'))
    await loadTicketDetail(updated.id)
    await refreshAll()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToUpdate')))
  } finally {
    escalating.value = false
  }
}

async function handleBalanceAdjust() {
  if (!selectedTicket.value || balanceForm.amount <= 0) return
  adjustingBalance.value = true
  try {
    await adminTicketsAPI.adjustBalance(selectedTicket.value.id, {
      operation: balanceForm.operation,
      amount: balanceForm.amount,
      notes: balanceForm.notes.trim() || undefined,
      business_category: balanceForm.business_category
    })
    appStore.showSuccess(t('admin.tickets.balance.done'))
    balanceForm.amount = 0
    balanceForm.notes = ''
    balanceForm.business_category = defaultBalanceBusinessCategory(balanceForm.operation)
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.balance.failed')))
  } finally {
    adjustingBalance.value = false
  }
}

async function handleReply() {
  if (!selectedTicket.value || !replyBody.value.trim()) return
  if (!canReplySelectedTicket.value) return
  replying.value = true
  try {
    await adminTicketsAPI.addMessage(selectedTicket.value.id, {
      body: replyBody.value.trim(),
      internal: canInternalNote.value ? replyInternal.value : false,
      attachments: normalizeAttachments(replyAttachments.value)
    })
    replyBody.value = ''
    replyAttachments.value = []
    replyInternal.value = false
    appStore.showSuccess(t('admin.tickets.replied'))
    await loadTicketDetail(selectedTicket.value.id)
    await fetchTickets()
    await fetchStats()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToReply')))
  } finally {
    replying.value = false
  }
}

function isSelected(id: number) {
  return selectedIds.value.has(id)
}

function eventChecked(event: Event) {
  return (event.target as HTMLInputElement | null)?.checked ?? false
}

function toggleSelection(id: number, checked: boolean) {
  if (!canBatchUpdate.value) return
  const next = new Set(selectedIds.value)
  if (checked) {
    next.add(id)
  } else {
    next.delete(id)
  }
  selectedIds.value = next
}

function togglePageSelection(checked: boolean) {
  if (!canBatchUpdate.value) return
  const next = new Set(selectedIds.value)
  for (const ticket of tickets.value) {
    if (checked) {
      next.add(ticket.id)
    } else {
      next.delete(ticket.id)
    }
  }
  selectedIds.value = next
}

function clearSelection() {
  selectedIds.value = new Set()
  bulkForm.status = ''
  bulkForm.priority = ''
  bulkForm.category = ''
  bulkForm.assignee_id = ''
}

async function handleBulkUpdate() {
  if (selectedIds.value.size === 0 || !hasBulkChanges.value || !canBatchUpdate.value) return
  bulkUpdating.value = true
  try {
    const assigneeID = bulkForm.assignee_id ? Number.parseInt(bulkForm.assignee_id, 10) : undefined
    const payload: {
      ids: number[]
      status?: string
      priority?: string
      category?: string
      assignee_id?: number
    } = {
      ids: Array.from(selectedIds.value),
      status: bulkForm.status || undefined
    }
    if (canUpdatePriority.value) {
      payload.priority = bulkForm.priority || undefined
    }
    if (canUpdateCategory.value) {
      payload.category = bulkForm.category || undefined
    }
    if (canTransfer.value) {
      payload.assignee_id = Number.isFinite(assigneeID) ? assigneeID : undefined
    }
    const res = await adminTicketsAPI.batchUpdate(payload)
    appStore.showSuccess(t('admin.tickets.bulk.updated', { count: res.updated }))
    clearSelection()
    await refreshAll()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToUpdate')))
  } finally {
    bulkUpdating.value = false
  }
}

async function handleAutoClose() {
  autoClosing.value = true
  try {
    const res = await adminTicketsAPI.autoCloseResolved(autoCloseDays.value)
    appStore.showSuccess(t('admin.tickets.autoClose.done', { count: res.closed }))
    showAutoCloseDialog.value = false
    await refreshAll()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.tickets.errors', t('admin.tickets.failedToUpdate')))
  } finally {
    autoClosing.value = false
  }
}

function assigneeLabel(id: number) {
  const user = adminUsers.value.find((item) => item.id === id)
  return user ? `${user.username || user.email}` : `#${id}`
}

function normalizeAttachments(items: TicketAttachment[]) {
  return items
    .map((item) => ({
      name: item.name.trim(),
      url: item.url.trim(),
      content_type: item.content_type?.trim() || undefined,
      size: item.size
    }))
    .filter((item) => item.name && item.url)
}

function statusLabel(status: string) {
  return t(`admin.tickets.status.${status as TicketStatus}`)
}

function priorityLabel(priority: string) {
  return t(`admin.tickets.priority.${priority as TicketPriority}`)
}

function categoryLabel(category: string) {
  return t(`admin.tickets.category.${category as TicketCategory}`)
}

function statusBadgeClass(status: string) {
  if (status === 'open') return 'badge-primary'
  if (status === 'pending') return 'badge-warning'
  if (status === 'resolved') return 'badge-success'
  if (status === 'closed') return 'badge-gray'
  return 'badge-gray'
}

function priorityBadgeClass(priority: string) {
  if (priority === 'urgent') return 'badge-danger'
  if (priority === 'high') return 'badge-warning'
  if (priority === 'low') return 'badge-gray'
  return 'badge-primary'
}

function senderLabel(message: TicketMessage) {
  const senderName = message.sender_name?.trim()
  const fallback = t(`admin.tickets.sender.${message.sender_type}`)
  return senderName ? `${fallback} · ${senderName}` : fallback
}

onMounted(() => {
  fetchCapabilities()
  fetchTickets()
  fetchStats()
  fetchAdminUsers()
})

onUnmounted(() => {
  if (searchTimer) window.clearTimeout(searchTimer)
})
</script>
