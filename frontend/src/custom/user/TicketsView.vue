<template>
  <AppLayout>
    <TablePageLayout flush-actions>
      <template #actions>
        <UserPageHero
          :kicker="t('tickets.gateway.kicker')"
          :title="t('tickets.title')"
          :description="t('tickets.description')"
        >
          <template #body>
              <button class="btn btn-primary mt-4 w-full justify-center sm:w-auto" @click="openCreateDialog">
                <Icon name="plus" size="md" />
                <span>{{ t('tickets.createTicket') }}</span>
              </button>

              <UserSummaryStats class="mt-5" :items="ticketSummaryItems" grid-class="grid-cols-1 sm:grid-cols-3" />
          </template>
        </UserPageHero>
      </template>

      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full sm:w-64">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted-2)]"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('tickets.searchPlaceholder')"
              class="input pl-10"
              @input="handleSearch"
            />
          </div>

          <Select v-model="filters.status" :options="statusFilterOptions" class="w-full sm:w-36" @change="applyFilters" />
          <Select v-model="filters.category" :options="categoryFilterOptions" class="w-full sm:w-40" @change="applyFilters" />
          <Select v-model="filters.priority" :options="priorityFilterOptions" class="w-full sm:w-36" @change="applyFilters" />
          <Select v-model="filters.unread_only" :options="unreadFilterOptions" class="w-full sm:w-36" @change="applyFilters" />

          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="fetchTickets"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
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
          <template #cell-subject="{ value, row }">
            <button class="group max-w-[360px] text-left" @click="openDetail(row)">
              <span class="block truncate font-medium text-[var(--apple-text)] transition-colors group-hover:text-[var(--apple-blue)]">
                {{ value }}
              </span>
              <span class="mt-1 block text-xs text-[var(--apple-muted-2)]">
                {{ row.ticket_no }}
              </span>
            </button>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass(value)]">{{ statusLabel(value) }}</span>
          </template>

          <template #cell-category="{ value }">
            <span class="badge badge-gray">{{ categoryLabel(value) }}</span>
          </template>

          <template #cell-priority="{ value }">
            <span :class="['badge', priorityBadgeClass(value)]">{{ priorityLabel(value) }}</span>
          </template>

          <template #cell-last_message_at="{ value }">
            <span class="text-sm text-[var(--apple-muted)]">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-unread_count="{ value }">
            <span v-if="value > 0" class="badge badge-primary">{{ t('tickets.unreadCount', { count: value }) }}</span>
            <span v-else class="text-sm text-[var(--apple-muted-2)]">{{ t('tickets.noUnread') }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              class="ticket-action-link inline-flex min-h-8 items-center justify-center gap-1 rounded-md px-2 py-1 text-xs font-medium"
              @click="openDetail(row)"
            >
              <Icon name="chatBubble" size="sm" />
              {{ t('tickets.viewDetail') }}
            </button>
          </template>

          <template #empty>
            <EmptyState
              :title="t('tickets.empty')"
              :description="t('tickets.emptyDescription')"
              :action-text="t('tickets.createTicket')"
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
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showCreateDialog"
      :title="t('tickets.createTicket')"
      width="wide"
      @close="closeCreateDialog"
    >
      <form id="ticket-create-form" class="space-y-4" @submit.prevent="handleCreate">
        <div>
          <label class="input-label">{{ t('tickets.form.template') }}</label>
          <Select v-model="createForm.template_key" :options="templateOptions" @change="handleTemplateChange" />
          <p v-if="selectedTemplate?.description" class="input-hint">{{ selectedTemplate.description }}</p>
        </div>

        <div
          v-if="selectedTemplate?.requires_super_admin"
          class="ticket-warning-note rounded-lg p-3 text-sm"
        >
          {{ t('tickets.form.superAdminHint') }}
        </div>

        <div>
          <label class="input-label">{{ t('tickets.form.subject') }}</label>
          <input v-model="createForm.subject" type="text" class="input" maxlength="200" required />
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('tickets.form.category') }}</label>
            <Select v-model="createForm.category" :options="categoryOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('tickets.form.priority') }}</label>
            <Select v-model="createForm.priority" :options="priorityOptions" />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('tickets.form.body') }}</label>
          <textarea
            v-model="createForm.body"
            rows="6"
            class="input"
            required
            :placeholder="t('tickets.form.bodyPlaceholder')"
          ></textarea>
          <p v-if="selectedTemplate?.body_min_length" class="input-hint">
            {{ t('tickets.form.bodyMinLength', { count: selectedTemplate.body_min_length }) }}
          </p>
        </div>

        <div v-if="requestAdviceText" class="ticket-guidance-note flex gap-3 rounded-lg p-3 text-sm">
          <Icon name="infoCircle" size="sm" class="mt-0.5 flex-shrink-0 text-[var(--apple-blue)]" />
          <div class="min-w-0">
            <p class="font-medium text-[var(--apple-text)]">{{ t('tickets.form.requestAdviceTitle') }}</p>
            <p class="mt-1 leading-5 text-[var(--apple-muted)]">{{ requestAdviceText }}</p>
          </div>
        </div>

        <div v-if="templateFields.length > 0" class="ticket-form-section space-y-4 p-4">
          <div v-for="field in templateFields" :key="field.key">
            <label class="input-label">
              {{ field.label }}
              <span v-if="field.required" class="text-[var(--apple-danger)]">*</span>
            </label>

            <Select
              v-if="field.type === 'group_select'"
              :model-value="selectFieldValue(field.key)"
              :options="groupOptions"
              @update:modelValue="setContextField(field.key, $event)"
            />

            <div v-else-if="field.type === 'recent_orders'" class="space-y-2">
              <div
                v-if="recentOrders.length > 0"
                class="grid grid-cols-1 gap-2 md:grid-cols-2"
              >
                <label
                  v-for="order in recentOrders"
                  :key="order.id"
                  class="ticket-order-card flex gap-3 p-3 text-sm"
                >
                  <input
                    type="checkbox"
                    class="ticket-checkbox mt-1 h-4 w-4 flex-shrink-0 rounded"
                    :checked="selectedRecentOrderIds.includes(order.id)"
                    @change="toggleRecentOrder(order.id, eventChecked($event))"
                  />
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center justify-between gap-3">
                      <span class="font-medium text-[var(--apple-text)]">#{{ order.id }}</span>
                      <span class="badge badge-gray">{{ order.status }}</span>
                    </div>
                    <div class="mt-1 text-xs text-[var(--apple-muted-2)]">
                      {{ formatDateTime(order.created_at) }}
                    </div>
                    <div class="mt-2 text-xs text-[var(--apple-muted)]">
                      {{ t('tickets.form.orderAmount', { amount: formatAmount(order.amount), pay: formatAmount(order.pay_amount ?? order.amount, order.currency) }) }}
                    </div>
                  </div>
                </label>
              </div>
              <p v-else class="text-sm text-[var(--apple-warning)]">
                {{ t('tickets.form.noRecentOrders') }}
              </p>
            </div>

            <input
              v-else-if="field.type === 'amount'"
              v-model.number="createContextData[field.key]"
              type="number"
              min="0"
              step="0.01"
              class="input"
              :placeholder="field.placeholder || t('tickets.form.amountPlaceholder')"
            />

            <textarea
              v-else-if="field.type === 'textarea'"
              :value="textFieldValue(field.key)"
              rows="3"
              class="input"
              :placeholder="field.placeholder || field.description"
              @input="setContextField(field.key, eventText($event))"
            ></textarea>

            <Select
              v-else-if="field.type === 'select'"
              :model-value="selectFieldValue(field.key)"
              :options="templateFieldOptions(field)"
              @update:modelValue="setContextField(field.key, $event)"
            />

            <div v-else-if="field.type === 'image'" class="space-y-2">
              <input
                :id="`ticket-image-${field.key}`"
                type="file"
                accept="image/png,image/jpeg,image/webp,image/gif"
                class="sr-only"
                @change="handleImageFieldFile(field.key, $event)"
              />
              <div class="flex flex-col gap-2 sm:flex-row">
                <label
                  :for="`ticket-image-${field.key}`"
                  class="btn btn-secondary cursor-pointer justify-center sm:w-auto"
                >
                  <Icon name="upload" size="sm" class="mr-1" />
                  {{ t('tickets.form.chooseImage') }}
                </label>
                <input
                  :value="imageURLInputValue(field.key)"
                  type="url"
                  class="input flex-1"
                  :placeholder="field.placeholder || t('tickets.form.imagePlaceholder')"
                  @input="setContextField(field.key, eventText($event))"
                />
              </div>
              <div
                v-if="imagePreviewValue(field.key)"
                class="ticket-image-preview flex flex-wrap items-center gap-3 p-3"
              >
                <img
                  :src="imagePreviewValue(field.key)"
                  :alt="field.label"
                  class="h-20 w-20 rounded-md object-cover"
                />
                <div class="min-w-0 flex-1 text-sm">
                  <p class="truncate font-medium text-[var(--apple-text)]">
                    {{ imageFileName(field.key) || t('tickets.form.imageSelected') }}
                  </p>
                  <a
                    :href="imagePreviewValue(field.key)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="ticket-action-link mt-1 inline-flex items-center gap-1"
                  >
                    <Icon name="externalLink" size="xs" />
                    {{ t('tickets.form.viewImage') }}
                  </a>
                </div>
                <button type="button" class="btn btn-secondary btn-icon" :title="t('common.delete')" @click="clearImageField(field.key)">
                  <Icon name="x" size="sm" />
                </button>
              </div>
            </div>

            <input
              v-else
              :value="textFieldValue(field.key)"
              type="text"
              class="input"
              :placeholder="field.placeholder || ''"
              @input="setContextField(field.key, eventText($event))"
            />

            <p v-if="field.description" class="input-hint">{{ field.description }}</p>
          </div>
        </div>

        <TicketAttachmentFields v-model="createForm.attachments" />
        <p v-if="contextAttachmentHint" class="ticket-context-hint px-3 py-2 text-xs leading-5">
          {{ contextAttachmentHint }}
        </p>
        <p class="ticket-trust-note px-3 py-2 text-xs leading-5">
          {{ t('tickets.form.privacyNote') }}
        </p>
      </form>

      <template #footer>
        <div class="grid grid-cols-2 gap-2 sm:flex sm:justify-end sm:gap-3">
          <button type="button" class="btn btn-secondary justify-center" @click="closeCreateDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="ticket-create-form" class="btn btn-primary justify-center" :disabled="creating">
            {{ creating ? t('common.saving') : t('tickets.submitTicket') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showDetailDialog"
      :title="detailTitle"
      width="wide"
      @close="closeDetailDialog"
    >
      <div v-if="selectedTicket" class="space-y-5">
        <div class="ticket-detail-summary p-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span :class="['badge', statusBadgeClass(selectedTicket.status)]">{{ statusLabel(selectedTicket.status) }}</span>
                <span :class="['badge', priorityBadgeClass(selectedTicket.priority)]">{{ priorityLabel(selectedTicket.priority) }}</span>
                <span class="badge badge-gray">{{ categoryLabel(selectedTicket.category) }}</span>
              </div>
              <h3 class="mt-3 break-words text-base font-semibold text-[var(--apple-text)]">
                {{ selectedTicket.subject }}
              </h3>
              <p class="mt-1 text-sm text-[var(--apple-muted)]">
                {{ selectedTicket.ticket_no }} · {{ t('tickets.lastMessageAt') }} {{ formatDateTime(selectedTicket.last_message_at) }}
              </p>
              <div class="mt-2 flex flex-wrap items-center gap-2">
                <TicketContextLink
                  :context-type="selectedTicket.context_type"
                  :context-id="selectedTicket.context_id"
                />
              </div>
              <div v-if="selectedContextEntries.length > 0" class="ticket-context-grid mt-3 grid grid-cols-1 gap-2 text-xs sm:grid-cols-2">
                <div
                  v-for="item in selectedContextEntries"
                  :key="item.key"
                  class="ticket-context-item rounded-md px-3 py-2"
                >
                  <span class="font-medium text-[var(--apple-text)]">{{ contextLabel(item.key) }}</span>
                  <span class="ml-1 break-words font-mono text-[var(--apple-muted)]">{{ item.value }}</span>
                </div>
              </div>
            </div>
            <button
              v-if="selectedTicket.status === 'closed'"
              class="btn btn-secondary"
              :disabled="updatingStatus"
              @click="handleReopenTicket"
            >
              {{ updatingStatus ? t('common.processing') : t('tickets.reopenTicket') }}
            </button>
            <button
              v-else
              class="btn btn-secondary"
              :disabled="updatingStatus"
              @click="handleCloseTicket"
            >
              {{ updatingStatus ? t('common.processing') : t('tickets.closeTicket') }}
            </button>
          </div>
        </div>

        <div class="max-h-[48vh] space-y-3 overflow-y-auto pr-1">
          <div v-if="detailLoading" class="py-10 text-center text-sm text-[var(--apple-muted)]">
            {{ t('common.loading') }}
          </div>
          <template v-else>
            <div
              v-for="message in selectedTicket.messages || []"
              :key="message.id"
              class="flex"
              :class="message.sender_type === 'user' ? 'justify-end' : 'justify-start'"
            >
              <div
                :class="[
                  'max-w-full rounded-lg border px-4 py-3 text-sm shadow-sm sm:max-w-[85%]',
                  message.sender_type === 'user'
                    ? 'ticket-message-user'
                    : 'ticket-message-support'
                ]"
              >
                <div class="mb-2 flex flex-wrap items-center gap-2 text-xs text-[var(--apple-muted-2)]">
                  <span class="font-medium">{{ senderLabel(message) }}</span>
                  <span>{{ formatDateTime(message.created_at) }}</span>
                </div>
                <p class="whitespace-pre-wrap break-words leading-6 [overflow-wrap:anywhere]">{{ message.body }}</p>
                <TicketAttachments :attachments="message.attachments" />
              </div>
            </div>
          </template>
        </div>

        <div v-if="selectedTicket.status === 'closed'" class="ticket-detail-summary p-4 text-sm text-[var(--apple-muted)]">
          {{ t('tickets.closedReplyHint') }}
        </div>
        <form v-else class="space-y-3" @submit.prevent="handleReply">
          <label class="input-label">{{ t('tickets.reply') }}</label>
          <textarea
            v-model="replyBody"
            rows="4"
            class="input"
            :placeholder="t('tickets.replyPlaceholder')"
          ></textarea>
          <TicketAttachmentFields v-model="replyAttachments" :i18n-prefix="'tickets.replyAttachments'" />
          <div class="flex justify-end">
            <button type="submit" class="btn btn-primary" :disabled="replying || !replyBody.trim()">
              {{ replying ? t('common.saving') : t('tickets.sendReply') }}
            </button>
          </div>
        </form>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import ticketsAPI from '@/custom/api/tickets'
import { useAppStore } from '@/stores'
import { useTicketStore } from '@/custom/stores/tickets'
import { getPersistedPageSize } from '@/custom/composables/usePersistedPageSize'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import type {
  Ticket,
  TicketAttachment,
  TicketCategory,
  TicketMessage,
  TicketPrefillData,
  TicketPrefillOrder,
  TicketTemplate,
  TicketTemplateField,
  TicketPriority,
  TicketStatus
} from '@/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import UserSummaryStats from '@/custom/user/UserSummaryStats.vue'
import TicketAttachmentFields from '@/custom/tickets/TicketAttachmentFields.vue'
import TicketAttachments from '@/custom/tickets/TicketAttachments.vue'
import TicketContextLink from '@/custom/tickets/TicketContextLink.vue'
import { formatCreditedBalance } from '@/custom/payment/orderAmounts'
import { formatPaymentAmount } from '@/components/payment/currency'

const MAX_TICKET_IMAGE_BYTES = 2 * 1024 * 1024
const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const ticketStore = useTicketStore()

const tickets = ref<Ticket[]>([])
const templates = ref<TicketTemplate[]>([])
const prefillData = ref<TicketPrefillData>({})
const loading = ref(false)
const detailLoading = ref(false)
const creating = ref(false)
const replying = ref(false)
const updatingStatus = ref(false)
const showCreateDialog = ref(false)
const showDetailDialog = ref(false)
const selectedTicket = ref<Ticket | null>(null)
const replyBody = ref('')
const replyAttachments = ref<TicketAttachment[]>([])
const searchQuery = ref('')
let searchTimer: number | undefined

const filters = reactive({
  status: '',
  category: '',
  priority: '',
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

const createForm = reactive<{
  subject: string
  body: string
  category: string
  priority: string
  template_key: string
  context_type: string
  context_id: string
  context_data: Record<string, unknown>
  attachments: TicketAttachment[]
}>({
  subject: '',
  body: '',
  category: 'general',
  priority: 'normal',
  template_key: '',
  context_type: '',
  context_id: '',
  context_data: {},
  attachments: []
})

const createContextData = createForm.context_data
const imageFieldFiles = reactive<Record<string, string>>({})
type IconName = InstanceType<typeof Icon>['$props']['name']

const ROUTE_CONTEXT_QUERY_KEYS = [
  'request_id',
  'model',
  'api_key_id',
  'group_id',
  'group_name',
  'inbound_endpoint',
  'upstream_endpoint',
  'actual_cost',
  'total_cost',
  'duration_ms',
  'first_token_ms',
  'status_code',
  'category',
  'platform',
  'error_message',
  'api_key_name',
  'created_at'
] as const

const columns = computed<Column[]>(() => [
  { key: 'subject', label: t('tickets.columns.subject'), sortable: true, class: 'min-w-[260px]' },
  { key: 'status', label: t('tickets.columns.status'), sortable: true },
  { key: 'category', label: t('tickets.columns.category'), sortable: true },
  { key: 'priority', label: t('tickets.columns.priority'), sortable: true },
  { key: 'last_message_at', label: t('tickets.columns.lastMessageAt'), sortable: true },
  { key: 'unread_count', label: t('tickets.columns.unread') },
  { key: 'actions', label: t('common.actions') }
])

const detailTitle = computed(() => selectedTicket.value?.ticket_no || t('tickets.detailTitle'))

const selectedContextEntries = computed(() => contextEntriesFromData(selectedTicket.value?.context_data))

const statusOptions = computed(() => [
  { value: 'open', label: t('tickets.status.open') },
  { value: 'pending', label: t('tickets.status.pending') },
  { value: 'resolved', label: t('tickets.status.resolved') },
  { value: 'closed', label: t('tickets.status.closed') }
])

const priorityOptions = computed(() => [
  { value: 'low', label: t('tickets.priority.low') },
  { value: 'normal', label: t('tickets.priority.normal') },
  { value: 'high', label: t('tickets.priority.high') },
  { value: 'urgent', label: t('tickets.priority.urgent') }
])

const categoryOptions = computed(() => [
  { value: 'general', label: t('tickets.category.general') },
  { value: 'billing', label: t('tickets.category.billing') },
  { value: 'usage', label: t('tickets.category.usage') },
  { value: 'technical', label: t('tickets.category.technical') },
  { value: 'account', label: t('tickets.category.account') }
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('tickets.filters.allStatus') },
  ...statusOptions.value
])

const priorityFilterOptions = computed(() => [
  { value: '', label: t('tickets.filters.allPriority') },
  ...priorityOptions.value
])

const categoryFilterOptions = computed(() => [
  { value: '', label: t('tickets.filters.allCategory') },
  ...categoryOptions.value
])

const unreadFilterOptions = computed(() => [
  { value: false, label: t('tickets.filters.allUnread') },
  { value: true, label: t('tickets.filters.onlyUnread') }
])

const ticketSummaryItems = computed<Array<{
  icon: IconName
  iconClass: string
  label: string
  value: string
  meta: string
}>>(() => {
  const openCount = tickets.value.filter((ticket) => ticket.status === 'open').length
  const pendingCount = tickets.value.filter((ticket) => ticket.status === 'pending').length
  const unreadCount = tickets.value.reduce((sum, ticket) => sum + (ticket.unread_count || 0), 0)
  return [
    {
      icon: 'document',
      iconClass: 'text-[var(--apple-blue)]',
      label: t('tickets.gateway.totalTickets'),
      value: pagination.total.toLocaleString(),
      meta: t('tickets.gateway.totalTicketsMeta')
    },
    {
      icon: 'chatBubble',
      iconClass: 'text-amber-300',
      label: t('tickets.gateway.activeTickets'),
      value: String(openCount + pendingCount),
      meta: `${statusLabel('open')} ${openCount} / ${statusLabel('pending')} ${pendingCount}`
    },
    {
      icon: 'bell',
      iconClass: 'text-[var(--apple-success)]',
      label: t('tickets.gateway.unreadMessages'),
      value: String(unreadCount),
      meta: unreadCount > 0 ? t('tickets.gateway.needsAttention') : t('tickets.noUnread')
    }
  ]
})

const selectedTemplate = computed(() => {
  return templates.value.find((item) => item.key === createForm.template_key) || null
})

const templateFields = computed<TicketTemplateField[]>(() => selectedTemplate.value?.fields || [])

const templateOptions = computed(() => templates.value.map((item) => ({
  value: item.key,
  label: item.name
})))

const groupOptions = computed(() => (prefillData.value.groups || []).map((group) => ({
  value: group.id,
  label: group.name
})))

const recentOrders = computed<TicketPrefillOrder[]>(() => prefillData.value.recent_orders || [])
const selectedTemplateHasRecentOrders = computed(() => templateFields.value.some((field) => field.type === 'recent_orders'))
const selectedRecentOrderIds = computed<number[]>(() => {
  const raw = createContextData.recent_order_ids
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item) && item > 0)
})
const requestAdviceKey = computed(() => {
  if (createForm.context_type !== 'request') return ''
  return resolveRequestAdviceKey(
    String(createContextData.category || ''),
    Number(createContextData.status_code || 0),
  )
})
const requestAdviceText = computed(() => {
  if (!requestAdviceKey.value) return ''
  return t(`tickets.form.requestAdvice.${requestAdviceKey.value}`)
})
const contextAttachmentHint = computed(() => {
  if (createForm.context_type === 'request') return t('tickets.form.attachmentHint.request')
  if (createForm.context_type === 'order') return t('tickets.form.attachmentHint.order')
  if (createForm.context_type === 'invoice') return t('tickets.form.attachmentHint.invoice')
  return ''
})

function textFieldValue(key: string) {
  const value = createContextData[key]
  if (value === null || value === undefined) return ''
  return typeof value === 'string' || typeof value === 'number' ? value : String(value)
}

function selectFieldValue(key: string) {
  const value = createContextData[key]
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean' || value === null) {
    return value
  }
  return ''
}

function setContextField(key: string, value: unknown) {
  createContextData[key] = value
  if (typeof value !== 'string' || !isImageDataURL(value)) {
    delete imageFieldFiles[key]
  }
}

function eventText(event: Event) {
  return (event.target as HTMLInputElement | HTMLTextAreaElement | null)?.value || ''
}

function imageURLInputValue(key: string) {
  const value = String(textFieldValue(key))
  return isImageDataURL(value) ? '' : value
}

function imagePreviewValue(key: string) {
  const value = String(textFieldValue(key)).trim()
  return isImageReference(value) ? value : ''
}

function imageFileName(key: string) {
  return imageFieldFiles[key] || ''
}

function clearImageField(key: string) {
  delete createContextData[key]
  delete imageFieldFiles[key]
}

async function handleImageFieldFile(key: string, event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0]
  if (input) input.value = ''
  if (!file) return
  if (!isAllowedInlineImageType(file.type)) {
    appStore.showError(t('tickets.errors.imageFileRequired'))
    return
  }
  if (file.size > MAX_TICKET_IMAGE_BYTES) {
    appStore.showError(t('tickets.errors.imageTooLarge', { size: 2 }))
    return
  }
  try {
    const dataURL = await readFileAsDataURL(file)
    createContextData[key] = dataURL
    imageFieldFiles[key] = file.name
  } catch {
    appStore.showError(t('tickets.errors.imageReadFailed'))
  }
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result)
      } else {
        reject(new Error('invalid file reader result'))
      }
    }
    reader.onerror = () => reject(reader.error || new Error('failed to read file'))
    reader.readAsDataURL(file)
  })
}

function eventChecked(event: Event) {
  return (event.target as HTMLInputElement | null)?.checked ?? false
}

function toggleRecentOrder(orderID: number, checked: boolean) {
  const next = new Set(selectedRecentOrderIds.value)
  if (checked) {
    next.add(orderID)
  } else {
    next.delete(orderID)
  }
  createContextData.recent_order_ids = Array.from(next)
}

function templateFieldOptions(field: TicketTemplateField) {
  return (field.options || []).map((option) => ({ value: option.value, label: option.label }))
}

function shouldSeedRecentOrdersContext() {
  return selectedTemplateHasRecentOrders.value
}

function buildListFilters() {
  return {
    status: filters.status || undefined,
    category: filters.category || undefined,
    priority: filters.priority || undefined,
    unread_only: filters.unread_only || undefined,
    search: searchQuery.value.trim() || undefined,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order
  }
}

async function fetchTickets() {
  loading.value = true
  try {
    const res = await ticketsAPI.list(pagination.page, pagination.page_size, buildListFilters())
    tickets.value = res.items || []
    pagination.total = res.total || 0
    pagination.pages = res.pages || 0
    pagination.page = res.page || pagination.page
    pagination.page_size = res.page_size || pagination.page_size
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'tickets.errors', t('tickets.failedToLoad')))
  } finally {
    loading.value = false
  }
}

async function fetchTicketFormMetadata() {
  try {
    const [templateItems, prefill] = await Promise.all([
      ticketsAPI.templates(),
      ticketsAPI.prefill()
    ])
    templates.value = templateItems || []
    prefillData.value = prefill || {}
    if (!createForm.template_key && templates.value.length > 0) {
      createForm.template_key = templates.value[0].key
      applySelectedTemplateDefaults()
    } else if (shouldSeedRecentOrdersContext()) {
      seedRecentOrdersContext()
    }
    applyCreateContextFromQuery()
  } catch (error: unknown) {
    console.error('Failed to load ticket form metadata:', error)
  }
}

function handleSearch() {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    applyFilters()
  }, 300)
}

function applyFilters() {
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

function openCreateDialog() {
  resetCreateForm()
  if (templates.value.length === 0) {
    fetchTicketFormMetadata()
  }
  showCreateDialog.value = true
}

function closeCreateDialog() {
  showCreateDialog.value = false
}

function applyCreateContextFromQuery() {
  const contextType = firstQueryString(route.query.context_type)
  const contextId = firstQueryString(route.query.context_id)
  const subject = firstQueryString(route.query.subject)
  if (contextType) createForm.context_type = contextType
  if (contextId) createForm.context_id = contextId
  if (subject) createForm.subject = subject
  for (const key of ROUTE_CONTEXT_QUERY_KEYS) {
    const value = firstQueryString(route.query[key])
    if (value) createContextData[key] = value
  }
  if (contextType === 'usage' && !createForm.body.trim()) {
    createForm.body = t('tickets.form.usageBodyPrefill', {
      usageId: contextId || '-',
      requestId: String(createContextData.request_id || '-'),
      model: String(createContextData.model || '-')
    })
  } else if (contextType === 'request' && !createForm.body.trim()) {
    const action = t(`tickets.form.requestAdvice.${resolveRequestAdviceKey(
      String(createContextData.category || ''),
      Number(createContextData.status_code || 0),
    )}`)
    createForm.body = t('tickets.form.requestBodyPrefill', {
      requestId: String(createContextData.request_id || contextId || '-'),
      model: String(createContextData.model || '-'),
      status: String(createContextData.status_code || '-'),
      category: String(createContextData.category || '-'),
      action,
    })
  }
}

function resetCreateForm() {
  createForm.subject = ''
  createForm.body = ''
  createForm.category = 'general'
  createForm.priority = 'normal'
  createForm.template_key = templates.value[0]?.key || ''
  createForm.context_type = ''
  createForm.context_id = ''
  clearCreateContextData()
  createForm.attachments = []
  applySelectedTemplateDefaults()
  applyCreateContextFromQuery()
}

function clearCreateContextData() {
  for (const key of Object.keys(createContextData)) {
    delete createContextData[key]
  }
  for (const key of Object.keys(imageFieldFiles)) {
    delete imageFieldFiles[key]
  }
}

function handleTemplateChange() {
  clearCreateContextData()
  createForm.attachments = []
  applySelectedTemplateDefaults()
}

function applySelectedTemplateDefaults() {
  const tpl = selectedTemplate.value
  if (!tpl) return
  createForm.subject = tpl.subject_template || tpl.name
  createForm.category = tpl.category || 'general'
  createForm.priority = tpl.priority || 'normal'
  createForm.context_type = tpl.context_type || ''
  if (shouldSeedRecentOrdersContext()) {
    seedRecentOrdersContext()
  } else {
    createForm.context_id = ''
  }
}

function seedRecentOrdersContext() {
  const orderIDs = recentOrders.value.map((order) => order.id)
  createContextData.recent_order_ids = orderIDs
  if (orderIDs.length > 0 && !createForm.context_id) {
    createForm.context_id = String(orderIDs[0])
  }
}

async function handleCreate() {
  if (!createForm.subject.trim() || !createForm.body.trim()) {
    appStore.showError(t('tickets.errors.requiredFields'))
    return
  }
  if (!validateTemplateSubmission()) {
    return
  }
  creating.value = true
  try {
    const created = await ticketsAPI.create({
      subject: createForm.subject.trim(),
      body: createForm.body.trim(),
      category: createForm.category,
      priority: createForm.priority,
      template_key: createForm.template_key || undefined,
      context_type: createForm.context_type.trim() || undefined,
      context_id: createForm.context_id.trim() || undefined,
      context_data: buildContextData(),
      attachments: normalizeAttachments(createForm.attachments)
    })
    appStore.showSuccess(t('tickets.created'))
    showCreateDialog.value = false
    resetCreateForm()
    await fetchTickets()
    await openDetail(created)
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'tickets.errors', t('tickets.failedToCreate')))
  } finally {
    creating.value = false
  }
}

function validateTemplateSubmission() {
  const tpl = selectedTemplate.value
  if (!tpl) return true
  const bodyMinLength = tpl.body_min_length || 0
  if (bodyMinLength > 0 && createForm.body.trim().length < bodyMinLength) {
    appStore.showError(t('tickets.errors.bodyTooShort', { count: bodyMinLength }))
    return false
  }
  for (const field of templateFields.value) {
    const value = createContextData[field.key]
    if (field.required && isEmptyTemplateValue(value)) {
      appStore.showError(t('tickets.errors.templateFieldRequired', { field: field.label }))
      return false
    }
    if ((field.type === 'image') && typeof value === 'string' && value.trim() && !isImageReference(value)) {
      appStore.showError(t('tickets.errors.imageURLRequired', { field: field.label }))
      return false
    }
  }
  return true
}

function buildContextData() {
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(createContextData)) {
    if (!isEmptyTemplateValue(value)) {
      out[key] = value
    }
  }
  if (selectedTemplateHasRecentOrders.value) {
    const selectedIDs = selectedRecentOrderIds.value
    out.recent_order_ids = selectedIDs
    out.recent_orders = recentOrders.value.filter((order) => selectedIDs.includes(order.id))
  }
  if (requestAdviceText.value && !out.recommended_action) {
    out.recommended_action = requestAdviceText.value
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function isEmptyTemplateValue(value: unknown) {
  if (value === null || value === undefined) return true
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'string') return value.trim() === ''
  return false
}

function isImageReference(value: string) {
  const trimmed = value.trim()
  return isHTTPURL(trimmed) || isImageDataURL(trimmed)
}

function isHTTPURL(value: string) {
  try {
    const parsed = new URL(value.trim())
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

function isImageDataURL(value: string) {
  return /^data:image\/(?:png|jpe?g|webp|gif);base64,/.test(value.trim())
}

function isAllowedInlineImageType(value: string) {
  return ['image/png', 'image/jpeg', 'image/webp', 'image/gif'].includes(value.toLowerCase())
}

function resolveRequestAdviceKey(category: string, statusCode: number) {
  const normalizedCategory = category.trim().toLowerCase()
  if ([
    'auth',
    'rate_limit',
    'quota',
    'invalid_request',
    'service_unavailable',
    'upstream',
    'internal',
    'cyber',
  ].includes(normalizedCategory)) {
    return normalizedCategory
  }
  if (statusCode === 401 || statusCode === 403) return 'auth'
  if (statusCode === 429) return 'rate_limit'
  if (statusCode === 400 || statusCode === 422) return 'invalid_request'
  if (statusCode >= 500) return 'service_unavailable'
  return 'default'
}

function formatAmount(value: number, currency?: string) {
  if (currency) {
    return formatPaymentAmount(Number(value || 0), currency)
  }
  return formatCreditedBalance(Number(value || 0))
}

async function openDetail(ticket: Ticket) {
  selectedTicket.value = { ...ticket, messages: ticket.messages || [] }
  showDetailDialog.value = true
  replyBody.value = ''
  replyAttachments.value = []
  await loadTicketDetail(ticket.id)
}

async function loadTicketDetail(id: number) {
  detailLoading.value = true
  try {
    const detail = await ticketsAPI.getById(id)
    selectedTicket.value = detail
    tickets.value = tickets.value.map((item) =>
      item.id === detail.id ? { ...item, ...detail, messages: item.messages, unread_count: 0 } : item
    )
    await ticketStore.fetchUnreadSummary('user', true)
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'tickets.errors', t('tickets.failedToLoadDetail')))
  } finally {
    detailLoading.value = false
  }
}

function closeDetailDialog() {
  showDetailDialog.value = false
  selectedTicket.value = null
  replyBody.value = ''
  replyAttachments.value = []
}

async function handleReply() {
  if (!selectedTicket.value || !replyBody.value.trim()) return
  replying.value = true
  try {
    await ticketsAPI.addMessage(selectedTicket.value.id, {
      body: replyBody.value.trim(),
      attachments: normalizeAttachments(replyAttachments.value)
    })
    replyBody.value = ''
    replyAttachments.value = []
    appStore.showSuccess(t('tickets.replySent'))
    await loadTicketDetail(selectedTicket.value.id)
    await fetchTickets()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'tickets.errors', t('tickets.failedToReply')))
  } finally {
    replying.value = false
  }
}

async function handleCloseTicket() {
  if (!selectedTicket.value) return
  updatingStatus.value = true
  try {
    selectedTicket.value = await ticketsAPI.close(selectedTicket.value.id)
    appStore.showSuccess(t('tickets.closed'))
    await loadTicketDetail(selectedTicket.value.id)
    await fetchTickets()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'tickets.errors', t('tickets.failedToUpdate')))
  } finally {
    updatingStatus.value = false
  }
}

async function handleReopenTicket() {
  if (!selectedTicket.value) return
  updatingStatus.value = true
  try {
    selectedTicket.value = await ticketsAPI.reopen(selectedTicket.value.id)
    appStore.showSuccess(t('tickets.reopened'))
    await loadTicketDetail(selectedTicket.value.id)
    await fetchTickets()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'tickets.errors', t('tickets.failedToUpdate')))
  } finally {
    updatingStatus.value = false
  }
}

function statusLabel(status: string) {
  return t(`tickets.status.${status as TicketStatus}`)
}

function priorityLabel(priority: string) {
  return t(`tickets.priority.${priority as TicketPriority}`)
}

function categoryLabel(category: string) {
  return t(`tickets.category.${category as TicketCategory}`)
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
  const fallback = t(`tickets.sender.${message.sender_type}`)
  return senderName ? `${fallback} · ${senderName}` : fallback
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

function firstQueryString(value: unknown): string {
  if (Array.isArray(value)) return String(value[0] || '').trim()
  return String(value || '').trim()
}

function contextEntriesFromData(data: Record<string, unknown> | undefined) {
  if (!data || typeof data !== 'object') return []
  return Object.entries(data)
    .filter(([, value]) => !isEmptyTemplateValue(value))
    .slice(0, 10)
    .map(([key, value]) => ({
      key,
      value: stringifyContextValue(value)
    }))
}

function contextLabel(key: string) {
  const i18nKey = `tickets.contextFields.${key}`
  const label = t(i18nKey)
  return label === i18nKey ? key : label
}

function stringifyContextValue(value: unknown): string {
  if (Array.isArray(value)) {
    if (value.length > 5) return t('tickets.contextFields.itemCount', { count: value.length })
    return value.map((item) => stringifyContextValue(item)).join(', ')
  }
  if (typeof value === 'object' && value !== null) {
    const record = value as Record<string, unknown>
    if ('id' in record) return `#${String(record.id)}`
    return JSON.stringify(value)
  }
  return String(value)
}

function maybeSeedContext() {
  const contextType = firstQueryString(route.query.context_type)
  const contextId = firstQueryString(route.query.context_id)
  const subject = firstQueryString(route.query.subject)
  if (!contextType && !contextId && !subject && String(route.query.new || '') !== '1') return
  if (!showCreateDialog.value) {
    resetCreateForm()
  }
  if (templates.value.length === 0) {
    fetchTicketFormMetadata()
  }
  if (contextType) createForm.context_type = contextType
  if (contextId) createForm.context_id = contextId
  if (subject) createForm.subject = subject
  showCreateDialog.value = true
}

onMounted(() => {
  fetchTickets()
  fetchTicketFormMetadata()
  ticketStore.fetchUnreadSummary('user')
  maybeSeedContext()
})

watch(
  () => route.query,
  () => maybeSeedContext()
)

onUnmounted(() => {
  if (searchTimer) window.clearTimeout(searchTimer)
})
</script>

<style scoped>
.ticket-trust-note,
.ticket-context-hint,
.ticket-form-section,
.ticket-detail-summary,
.ticket-context-item {
  border: 1px solid var(--apple-border);
  border-radius: var(--apple-radius);
  background: var(--apple-surface-elevated);
  color: var(--apple-muted);
  box-shadow: var(--apple-shadow-sm);
}

.ticket-form-section,
.ticket-detail-summary,
.ticket-context-item {
  color: var(--apple-text);
}

.ticket-warning-note {
  border: 1px solid color-mix(in srgb, var(--apple-warning) 26%, var(--apple-border));
  background: color-mix(in srgb, var(--apple-warning) 10%, var(--apple-surface));
  color: var(--apple-warning);
}

.ticket-guidance-note {
  border: 1px solid color-mix(in srgb, var(--apple-blue) 24%, var(--apple-border));
  background: color-mix(in srgb, var(--apple-blue) 8%, var(--apple-surface));
}

.ticket-action-link {
  color: var(--apple-blue);
  transition:
    background-color 0.16s ease,
    color 0.16s ease;
}

.ticket-action-link:hover {
  background: var(--apple-hover);
  color: var(--apple-blue-hover);
}

.ticket-action-link:focus-visible {
  outline: 2px solid var(--apple-focus-ring);
  outline-offset: 2px;
}

.ticket-order-card,
.ticket-image-preview {
  border: 1px solid var(--apple-border-soft);
  border-radius: var(--apple-radius);
  background: var(--apple-surface);
  box-shadow: var(--apple-shadow-sm);
}

.ticket-order-card {
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease;
}

.ticket-order-card:hover {
  border-color: color-mix(in srgb, var(--apple-blue) 32%, var(--apple-border));
  background: color-mix(in srgb, var(--apple-blue) 5%, var(--apple-surface));
}

.ticket-checkbox {
  border: 1px solid var(--apple-border);
  accent-color: var(--apple-blue);
}

.ticket-checkbox:focus-visible {
  outline: 2px solid var(--apple-focus-ring);
  outline-offset: 2px;
}

.ticket-message-user {
  border-color: color-mix(in srgb, var(--apple-blue) 34%, var(--apple-border));
  background: color-mix(in srgb, var(--apple-blue) 11%, var(--apple-surface));
  color: var(--apple-text);
}

.ticket-message-support {
  border-color: var(--apple-border);
  background: var(--apple-surface);
  color: var(--apple-text);
}
</style>
