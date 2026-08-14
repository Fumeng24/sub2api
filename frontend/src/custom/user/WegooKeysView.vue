<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-3">
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput
              v-model="filterSearch"
              :placeholder="t('keys.searchPlaceholder')"
              class="w-full sm:w-64"
              @search="onFilterChange"
            />
            <Select
              :model-value="filterGroupId"
              class="w-40"
              :options="groupFilterOptions"
              @update:model-value="onGroupFilterChange"
            />
            <Select
              :model-value="filterStatus"
              class="w-40"
              :options="statusFilterOptions"
              @update:model-value="onStatusFilterChange"
            />
          </div>
          <EndpointPopover
            v-if="publicSettings?.api_base_url || (publicSettings?.custom_endpoints?.length ?? 0) > 0"
            :api-base-url="publicSettings?.api_base_url || ''"
            :custom-endpoints="publicSettings?.custom_endpoints || []"
          />
        </div>
      </template>

      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            @click="loadApiKeys"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <div class="relative" ref="columnDropdownRef">
            <button
              @click="showColumnDropdown = !showColumnDropdown"
              class="btn btn-secondary px-2 md:px-3"
              :title="t('keys.columnSettings')"
            >
              <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
              </svg>
              <span class="hidden md:inline">{{ t('keys.columnSettings') }}</span>
            </button>
            <div
              v-if="showColumnDropdown"
              class="absolute right-0 top-full z-50 mt-1 max-h-80 w-48 overflow-y-auto rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] py-1 shadow-[var(--apple-shadow-md)]"
            >
              <button
                v-for="col in toggleableColumns"
                :key="col.key"
                @click="toggleColumn(col.key)"
                class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-[var(--apple-text)] hover:bg-[var(--apple-hover)]"
              >
                <span>{{ col.label }}</span>
                <Icon
                  v-if="isColumnVisible(col.key)"
                  name="check"
                  size="sm"
                  class="text-[var(--apple-blue)]"
                  :stroke-width="2"
                />
              </button>
            </div>
          </div>
          <button @click="showCreateModal = true" class="btn btn-primary" data-tour="keys-create-btn">
            <Icon name="plus" size="md" class="mr-2" />
            {{ t('keys.createKey') }}
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="apiKeys"
          :loading="loading"
          :server-side-sort="true"
          :estimate-row-height="88"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-key="{ value, row }">
            <div class="flex items-center gap-2">
              <code class="code text-xs">
                {{ maskApiKey(value) }}
              </code>
              <button
                @click="copyToClipboard(value, row.id)"
                class="rounded-lg p-1 text-[var(--apple-muted-2)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
                :class="copiedKeyId === row.id ? 'text-[var(--apple-success)]' : ''"
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
              >
                <Icon v-if="copiedKeyId === row.id" name="check" size="sm" :stroke-width="2" />
                <Icon v-else name="clipboard" size="sm" />
              </button>
            </div>
          </template>

          <template #cell-name="{ value, row }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-[var(--apple-text)]">{{ value }}</span>
              <Icon
                v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
                name="shield"
                size="sm"
                class="text-[var(--apple-blue)]"
                :title="t('keys.ipRestrictionEnabled')"
              />
            </div>
          </template>

          <template #cell-group="{ row }">
            <div class="key-row-group group/dropdown relative">
              <button
                :ref="(el) => setGroupButtonRef(row.id, el)"
                @click="openGroupSelector(row)"
                class="-mx-2 -my-1 flex max-w-full min-w-0 cursor-pointer flex-nowrap items-center justify-start gap-x-2 gap-y-1 rounded-lg px-2 py-1 text-left transition-all duration-200 hover:bg-[var(--apple-hover)]"
                :title="t('keys.clickToChangeGroup')"
              >
                <GroupBadge
                  v-if="row.group"
                  class="min-w-0 max-w-full"
                  :name="row.group.name"
                  :platform="row.group.platform"
                  :subscription-type="row.group.subscription_type"
                  :group-id="row.group.id"
                  :rate-multiplier="row.group.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[row.group.id]"
                  :discount-multiplier="row.group.group_rate_discount_multiplier"
                  :discounted-rate-multiplier="row.group.discounted_rate_multiplier"
                  :discount-name="row.group.group_rate_discount_name"
                  :discount-schedule-mode="row.group.group_rate_discount_schedule_mode"
                  :discount-start-at="row.group.group_rate_discount_start_at"
                  :discount-end-at="row.group.group_rate_discount_end_at"
                  :discount-weekdays="row.group.group_rate_discount_weekdays"
                  :discount-daily-start-time="row.group.group_rate_discount_daily_start_time"
                  :discount-daily-end-time="row.group.group_rate_discount_daily_end_time"
                  :discount-timezone="row.group.group_rate_discount_timezone"
                  :peak-rate-enabled="row.group.peak_rate_enabled"
                  :peak-start="row.group.peak_start"
                  :peak-end="row.group.peak_end"
                  :peak-rate-multiplier="row.group.peak_rate_multiplier"
                />
                <span v-else class="shrink-0 whitespace-nowrap text-sm text-[var(--apple-muted-2)]">{{
                  t('keys.noGroup')
                }}</span>
                <svg
                  class="h-3.5 w-3.5 shrink-0 text-[var(--apple-muted-2)] opacity-60 transition-opacity group-hover/dropdown:opacity-100"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
                  />
                </svg>
              </button>
            </div>
          </template>

          <template #cell-current_concurrency="{ value }">
            <span
              :class="[
                'inline-flex min-w-8 items-center justify-center rounded px-2 py-1 text-sm font-semibold tabular-nums',
                (value ?? 0) > 0
                  ? 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-900/25 dark:text-emerald-300 dark:ring-emerald-800'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
              ]"
            >
              {{ value ?? 0 }}
            </span>
          </template>

          <template #cell-usage="{ row }">
            <div class="text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-[var(--apple-muted)]">{{ t('keys.today') }}:</span>
                <span class="font-medium text-[var(--apple-text)]">
                  {{ formatSettlementAmount(usageStats[row.id]?.today_actual_cost ?? 0, 4) }}
                </span>
              </div>
              <div class="mt-0.5 flex items-center gap-1.5">
                <span class="text-[var(--apple-muted)]">{{ t('keys.total') }}:</span>
                <span class="font-medium text-[var(--apple-text)]">
                  {{ formatSettlementAmount(usageStats[row.id]?.total_actual_cost ?? 0, 4) }}
                </span>
              </div>
              <div v-if="row.quota > 0" class="mt-1.5">
                <div class="flex items-center gap-1.5">
                  <span class="text-[var(--apple-muted)]">{{ t('keys.quota') }}:</span>
                  <span :class="[
                    'font-medium',
                    row.quota_used >= row.quota ? 'text-[var(--apple-danger)]' :
                    row.quota_used >= row.quota * 0.8 ? 'text-[var(--apple-warning)]' :
                    'text-[var(--apple-text)]'
                  ]">
                    {{ formatSettlementAmountPair(row.quota_used ?? 0, row.quota ?? 0, 2) }}
                  </span>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.quota_used >= row.quota ? 'bg-[var(--apple-danger)]' :
                      row.quota_used >= row.quota * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-blue)]'
                    ]"
                    :style="{ width: Math.min((row.quota_used / row.quota) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>
          </template>

          <template #cell-rate_limit="{ row }">
            <div v-if="row.rate_limit_5h > 0 || row.rate_limit_1d > 0 || row.rate_limit_7d > 0" class="min-w-[140px] space-y-1.5">
              <div v-if="row.rate_limit_5h > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-[var(--apple-muted)]">5h</span>
                  <span :class="[
                    'font-medium tabular-nums',
                    row.usage_5h >= row.rate_limit_5h ? 'text-[var(--apple-danger)]' :
                    row.usage_5h >= row.rate_limit_5h * 0.8 ? 'text-[var(--apple-warning)]' :
                    'text-[var(--apple-text)]'
                  ]">
                    {{ formatSettlementAmountPair(row.usage_5h ?? 0, row.rate_limit_5h ?? 0, 2) }}
                  </span>
                </div>
                <div class="h-1 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.usage_5h >= row.rate_limit_5h ? 'bg-[var(--apple-danger)]' :
                      row.usage_5h >= row.rate_limit_5h * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-success)]'
                    ]"
                    :style="{ width: Math.min((row.usage_5h / row.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
              <div v-if="row.rate_limit_1d > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-[var(--apple-muted)]">1d</span>
                  <span :class="[
                    'font-medium tabular-nums',
                    row.usage_1d >= row.rate_limit_1d ? 'text-[var(--apple-danger)]' :
                    row.usage_1d >= row.rate_limit_1d * 0.8 ? 'text-[var(--apple-warning)]' :
                    'text-[var(--apple-text)]'
                  ]">
                    {{ formatSettlementAmountPair(row.usage_1d ?? 0, row.rate_limit_1d ?? 0, 2) }}
                  </span>
                </div>
                <div class="h-1 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.usage_1d >= row.rate_limit_1d ? 'bg-[var(--apple-danger)]' :
                      row.usage_1d >= row.rate_limit_1d * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-success)]'
                    ]"
                    :style="{ width: Math.min((row.usage_1d / row.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
              <div v-if="row.rate_limit_7d > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-[var(--apple-muted)]">7d</span>
                  <span :class="[
                    'font-medium tabular-nums',
                    row.usage_7d >= row.rate_limit_7d ? 'text-[var(--apple-danger)]' :
                    row.usage_7d >= row.rate_limit_7d * 0.8 ? 'text-[var(--apple-warning)]' :
                    'text-[var(--apple-text)]'
                  ]">
                    {{ formatSettlementAmountPair(row.usage_7d ?? 0, row.rate_limit_7d ?? 0, 2) }}
                  </span>
                </div>
                <div class="h-1 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.usage_7d >= row.rate_limit_7d ? 'bg-[var(--apple-danger)]' :
                      row.usage_7d >= row.rate_limit_7d * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-success)]'
                    ]"
                    :style="{ width: Math.min((row.usage_7d / row.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
              <button
                v-if="row.usage_5h > 0 || row.usage_1d > 0 || row.usage_7d > 0"
                @click.stop="confirmResetRateLimitFromTable(row)"
                class="mt-0.5 inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-blue)]"
                :title="t('keys.resetRateLimitUsage')"
              >
                <Icon name="refresh" size="xs" />
                {{ t('keys.resetUsage') }}
              </button>
            </div>
            <span v-else class="text-sm text-[var(--apple-muted-2)]">-</span>
          </template>

          <template #cell-expires_at="{ value }">
            <span v-if="value" :class="[
              'text-sm',
              new Date(value) < new Date() ? 'text-[var(--apple-danger)]' : 'text-[var(--apple-muted)]'
            ]">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-[var(--apple-muted-2)]">{{ t('keys.noExpiration') }}</span>
          </template>

          <template #cell-status="{ value }">
            <span :class="[
              'badge',
              value === 'active' ? 'badge-success' :
              value === 'quota_exhausted' ? 'badge-warning' :
              value === 'expired' ? 'badge-danger' :
              'badge-gray'
            ]">
              {{ t('keys.status.' + value) }}
            </span>
          </template>

          <template #cell-last_used_at="{ value }">
            <span v-if="value" class="text-sm text-[var(--apple-muted)]">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-[var(--apple-muted-2)]">-</span>
          </template>

          <template #cell-last_used_ip="{ value }">
            <span v-if="value" class="text-sm text-[var(--apple-muted)]">
              {{ value }}
            </span>
            <span v-else class="text-sm text-[var(--apple-muted-2)]">-</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-[var(--apple-muted)]">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="key-row-actions">
              <button
                v-if="!publicSettings?.hide_ccs_import_button"
                @click="importToCcswitch(row)"
                class="inline-flex min-h-8 shrink-0 items-center justify-center gap-1.5 rounded-lg bg-[color-mix(in_srgb,var(--apple-blue)_10%,var(--apple-surface-elevated))] px-2.5 py-1.5 text-xs font-medium text-[var(--apple-blue)] ring-1 ring-[color-mix(in_srgb,var(--apple-blue)_24%,var(--apple-border-soft))] transition-colors hover:bg-[color-mix(in_srgb,var(--apple-blue)_14%,var(--apple-surface-elevated))]"
                :title="t('keys.importToCcSwitch')"
                :aria-label="t('keys.importToCcSwitch')"
              >
                <Icon name="upload" size="sm" />
                <span class="whitespace-nowrap">{{ t('keys.importToCcSwitch') }}</span>
              </button>
              <button
                @click="openUseKeyModal(row)"
                :data-testid="`use-key-open-${row.id}`"
                class="key-row-icon-action text-[var(--apple-muted)] hover:text-[var(--apple-text)]"
                :title="t('keys.useKey')"
                :aria-label="t('keys.useKey')"
              >
                <Icon name="terminal" size="sm" />
              </button>
              <button
                @click="editKey(row)"
                class="key-row-icon-action text-[var(--apple-muted)] hover:text-[var(--apple-text)]"
                :title="t('common.edit')"
                :aria-label="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                @click="confirmDelete(row)"
                class="key-row-icon-action text-[var(--apple-muted)] hover:bg-[color-mix(in_srgb,var(--apple-danger)_8%,var(--apple-surface))] hover:text-[var(--apple-danger)]"
                :title="t('common.delete')"
                :aria-label="t('common.delete')"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('keys.noKeysYet')"
              :description="t('keys.createFirstKey')"
              :action-text="t('keys.createKey')"
              @action="showCreateModal = true"
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

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
          />
        </div>

        <div>
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            :group-tabs="true"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :group-id="(option as unknown as GroupOption).value"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :discount-multiplier="(option as unknown as GroupOption).discountMultiplier"
                :discounted-rate-multiplier="(option as unknown as GroupOption).discountedRateMultiplier"
                :discount-name="(option as unknown as GroupOption).discountName"
                :discount-schedule-mode="(option as unknown as GroupOption).discountScheduleMode"
                :discount-start-at="(option as unknown as GroupOption).discountStartAt"
                :discount-end-at="(option as unknown as GroupOption).discountEndAt"
                :discount-weekdays="(option as unknown as GroupOption).discountWeekdays"
                :discount-daily-start-time="(option as unknown as GroupOption).discountDailyStartTime"
                :discount-daily-end-time="(option as unknown as GroupOption).discountDailyEndTime"
                :discount-timezone="(option as unknown as GroupOption).discountTimezone"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
              />
              <span v-else class="text-[var(--apple-muted-2)]">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :group-id="(option as unknown as GroupOption).value"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :discount-multiplier="(option as unknown as GroupOption).discountMultiplier"
                :discounted-rate-multiplier="(option as unknown as GroupOption).discountedRateMultiplier"
                :discount-name="(option as unknown as GroupOption).discountName"
                :discount-schedule-mode="(option as unknown as GroupOption).discountScheduleMode"
                :discount-start-at="(option as unknown as GroupOption).discountStartAt"
                :discount-end-at="(option as unknown as GroupOption).discountEndAt"
                :discount-weekdays="(option as unknown as GroupOption).discountWeekdays"
                :discount-daily-start-time="(option as unknown as GroupOption).discountDailyStartTime"
                :discount-daily-end-time="(option as unknown as GroupOption).discountDailyEndTime"
                :discount-timezone="(option as unknown as GroupOption).discountTimezone"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
          <div
            v-if="selectedGroupForForm"
            class="mt-3 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] px-4 py-3 text-sm leading-6 text-[var(--apple-text)]"
          >
            <div class="flex items-start gap-2">
              <Icon name="calculator" size="sm" class="mt-0.5 shrink-0 text-[var(--apple-muted)]" />
              <div>
                <p class="font-semibold">
                  {{ t('keys.groupCostPreview.title', { group: selectedGroupForForm.label }) }}
                </p>
                <p class="mt-1 text-[var(--apple-muted)]">
                  {{ selectedGroupCostPreview }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
            <button
              type="button"
              @click="formData.use_custom_key = !formData.use_custom_key"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.use_custom_key ? 'bg-[var(--apple-blue)]' : 'bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-[var(--apple-surface)] shadow ring-0 transition duration-200 ease-in-out',
                  formData.use_custom_key ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-[color:var(--apple-danger)]': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-[var(--apple-danger)]">{{ customKeyError }}</p>
            <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <div
          v-if="showEditModal && selectedKey"
          class="rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-4 py-3"
        >
          <div class="mb-3 flex items-center gap-2 text-sm font-semibold text-[var(--apple-text)]">
            <Icon name="infoCircle" size="sm" class="text-[var(--apple-muted)]" />
            <span>{{ t('keys.overview') }}</span>
          </div>
          <dl class="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt class="text-xs text-[var(--apple-muted)]">{{ t('common.status') }}</dt>
              <dd class="mt-1">
                <span :class="[
                  'badge',
                  selectedKey.status === 'active' ? 'badge-success' :
                  selectedKey.status === 'quota_exhausted' ? 'badge-warning' :
                  selectedKey.status === 'expired' ? 'badge-danger' :
                  'badge-gray'
                ]">
                  {{ t('keys.status.' + selectedKey.status) }}
                </span>
              </dd>
            </div>
            <div>
              <dt class="text-xs text-[var(--apple-muted)]">{{ t('keys.usage') }}</dt>
              <dd class="mt-1 text-[var(--apple-text)]">
                {{ t('keys.today') }} {{ formatSettlementAmount(usageStats[selectedKey.id]?.today_actual_cost ?? 0, 4) }}
                <span class="mx-1 text-[var(--apple-muted-2)]">/</span>
                {{ t('keys.total') }} {{ formatSettlementAmount(usageStats[selectedKey.id]?.total_actual_cost ?? 0, 4) }}
              </dd>
            </div>
            <div>
              <dt class="text-xs text-[var(--apple-muted)]">{{ t('keys.quota') }}</dt>
              <dd class="mt-1 text-[var(--apple-text)]">
                <span v-if="selectedKey.quota > 0">
                  {{ formatSettlementAmountPair(selectedKey.quota_used ?? 0, selectedKey.quota ?? 0, 2) }}
                </span>
                <span v-else>{{ t('keys.unlimited') }}</span>
              </dd>
            </div>
            <div>
              <dt class="text-xs text-[var(--apple-muted)]">{{ t('keys.lastUsedAt') }}</dt>
              <dd class="mt-1 text-[var(--apple-text)]">
                {{ selectedKey.last_used_at ? formatDateTime(selectedKey.last_used_at) : t('keys.neverUsed') }}
              </dd>
            </div>
          </dl>
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
            <button
              type="button"
              @click="formData.enable_ip_restriction = !formData.enable_ip_restriction"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_ip_restriction ? 'bg-[var(--apple-blue)]' : 'bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-[var(--apple-surface)] shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_ip_restriction ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div class="space-y-3">
          <label class="input-label">{{ t('keys.quotaLimit') }}</label>
          <!-- Switch commented out - always show input, 0 = unlimited
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.quotaLimit') }}</label>
            <button
              type="button"
              @click="formData.enable_quota = !formData.enable_quota"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_quota ? 'bg-[var(--apple-blue)]' : 'bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-[var(--apple-surface)] shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_quota ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          -->

          <div class="space-y-4">
            <div>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted)]">{{ settlementAmountPrefix }}</span>
                <input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
              <label class="input-label">{{ t('keys.quotaUsed') }}</label>
              <div class="flex items-center gap-2">
                <div class="flex-1 rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2 ring-1 ring-[color:var(--apple-border-soft)]">
                  <span class="font-medium text-[var(--apple-text)]">
                    {{ formatSettlementAmount(selectedKey.quota_used ?? 0, 4) }}
                  </span>
                  <span class="mx-2 text-[var(--apple-muted-2)]">/</span>
                  <span class="text-[var(--apple-muted)]">
                    {{ formatSettlementAmount(selectedKey.quota ?? 0, 2) }}
                  </span>
                </div>
                <button
                  type="button"
                  @click="confirmResetQuota"
                  class="btn btn-secondary text-sm"
                  :title="t('keys.resetQuotaUsed')"
                >
                  {{ t('keys.reset') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.rateLimitSection') }}</label>
            <button
              type="button"
              @click="formData.enable_rate_limit = !formData.enable_rate_limit"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_rate_limit ? 'bg-[var(--apple-blue)]' : 'bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-[var(--apple-surface)] shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_rate_limit ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="input-hint -mt-2">{{ t('keys.rateLimitHint') }}</p>
            <!-- 5-Hour Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit5h') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted)]">{{ settlementAmountPrefix }}</span>
                <input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2 text-sm ring-1 ring-[color:var(--apple-border-soft)]">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'text-[var(--apple-danger)]' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'text-[var(--apple-warning)]' :
                      'text-[var(--apple-text)]'
                    ]">
                      {{ formatSettlementAmount(selectedKey.usage_5h ?? 0, 4) }}
                    </span>
                    <span class="mx-2 text-[var(--apple-muted-2)]">/</span>
                    <span class="text-[var(--apple-muted)]">
                      {{ formatSettlementAmount(selectedKey.rate_limit_5h ?? 0, 2) }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'bg-[var(--apple-danger)]' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-success)]'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit1d') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted)]">{{ settlementAmountPrefix }}</span>
                <input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2 text-sm ring-1 ring-[color:var(--apple-border-soft)]">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'text-[var(--apple-danger)]' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'text-[var(--apple-warning)]' :
                      'text-[var(--apple-text)]'
                    ]">
                      {{ formatSettlementAmount(selectedKey.usage_1d ?? 0, 4) }}
                    </span>
                    <span class="mx-2 text-[var(--apple-muted-2)]">/</span>
                    <span class="text-[var(--apple-muted)]">
                      {{ formatSettlementAmount(selectedKey.rate_limit_1d ?? 0, 2) }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'bg-[var(--apple-danger)]' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-success)]'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit7d') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted)]">{{ settlementAmountPrefix }}</span>
                <input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2 text-sm ring-1 ring-[color:var(--apple-border-soft)]">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'text-[var(--apple-danger)]' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'text-[var(--apple-warning)]' :
                      'text-[var(--apple-text)]'
                    ]">
                      {{ formatSettlementAmount(selectedKey.usage_7d ?? 0, 4) }}
                    </span>
                    <span class="mx-2 text-[var(--apple-muted-2)]">/</span>
                    <span class="text-[var(--apple-muted)]">
                      {{ formatSettlementAmount(selectedKey.rate_limit_7d ?? 0, 2) }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'bg-[var(--apple-danger)]' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'bg-[var(--apple-warning)]' :
                      'bg-[var(--apple-success)]'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
              <button
                type="button"
                @click="confirmResetRateLimit"
                class="btn btn-secondary text-sm"
              >
                {{ t('keys.resetRateLimitUsage') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.expiration') }}</label>
            <button
              type="button"
              @click="formData.enable_expiration = !formData.enable_expiration"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_expiration ? 'bg-[var(--apple-blue)]' : 'bg-[var(--apple-surface-elevated)] ring-1 ring-[color:var(--apple-border-soft)]'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-[var(--apple-surface)] shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_expiration ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-[color-mix(in_srgb,var(--apple-blue)_12%,var(--apple-surface))] text-[var(--apple-blue)]'
                    : 'bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'
                ]"
              >
                {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
              </button>
              <button
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-[color-mix(in_srgb,var(--apple-blue)_12%,var(--apple-surface))] text-[var(--apple-blue)]'
                    : 'bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'
                ]"
              >
                {{ t('keys.customDate') }}
              </button>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div>
              <label class="input-label">{{ t('keys.expirationDate') }}</label>
              <input
                v-model="formData.expiration_date"
                type="datetime-local"
                class="input"
              />
              <p class="input-hint">{{ t('keys.expirationDateHint') }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
              <span class="text-[var(--apple-muted)]">{{ t('keys.currentExpiration') }}: </span>
              <span class="font-medium text-[var(--apple-text)]">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
          >
            <svg
              v-if="submitting"
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
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey ? formatSettlementAmount(selectedKey.quota_used ?? 0, 4) : '-' })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      :group-name="selectedKey?.group?.name || ''"
      :custom-endpoints="publicSettings?.custom_endpoints || []"
      @close="closeUseKeyModal"
    />

    <!-- CC Switch Import Dialog -->
    <BaseDialog
      :show="showCcsImportDialog"
      :title="t('keys.ccSwitchDialog.title')"
      width="wide"
      @close="closeCcsImportDialog"
    >
      <div class="space-y-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div class="space-y-2">
            <p class="text-sm text-[var(--apple-muted)]">
              {{ t('keys.ccSwitchDialog.description') }}
            </p>
            <div
              v-if="selectedCcsRow?.group"
              class="flex flex-wrap items-center gap-2 text-xs text-[var(--apple-muted)]"
            >
              <span>{{ t('keys.ccSwitchDialog.currentGroup') }}</span>
              <GroupBadge
                :name="selectedCcsRow.group.name"
                :platform="selectedCcsRow.group.platform"
                :subscription-type="selectedCcsRow.group.subscription_type"
                :group-id="selectedCcsRow.group.id"
                :rate-multiplier="selectedCcsRow.group.rate_multiplier"
                :show-rate="false"
              />
            </div>
          </div>
          <a
            :href="ccSwitchOfficialUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex shrink-0 items-center gap-1.5 text-sm font-medium text-[var(--apple-blue)] transition-colors hover:text-[var(--apple-blue-hover)]"
          >
            <Icon name="externalLink" size="sm" />
            {{ t('keys.ccSwitchDialog.openOfficial') }}
          </a>
        </div>

        <div
          v-if="ccsImportTargets.length === 0"
          class="rounded-lg border border-[color:color-mix(in_srgb,var(--apple-warning)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_10%,var(--apple-surface))] p-4 text-sm text-[var(--apple-warning)]"
        >
          {{ t('keys.ccSwitchDialog.noGroup') }}
        </div>

        <div v-else class="grid gap-3 md:grid-cols-2">
          <div
            v-for="target in ccsImportTargets"
            :key="target.targetId"
            class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h4 class="text-sm font-semibold text-[var(--apple-text)]">
                  <span class="inline-flex items-center gap-1.5">
                    <Icon :name="ccSwitchTargetIcon[target.targetId]" size="sm" class="text-[var(--apple-muted)]" />
                    {{ getCcSwitchTargetName(target.targetId) }}
                  </span>
                </h4>
                <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
                  {{ getCcSwitchTargetDescription(target.targetId) }}
                </p>
              </div>
              <span class="rounded-md bg-[var(--apple-surface-elevated)] px-2 py-1 text-[11px] font-medium text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]">
                {{ target.app }}
              </span>
            </div>

            <dl class="mt-4 space-y-2 text-xs">
              <div class="flex gap-2">
                <dt class="w-16 shrink-0 text-[var(--apple-muted-2)]">
                  {{ t('keys.ccSwitchDialog.endpoint') }}
                </dt>
                <dd class="min-w-0 flex-1 break-all font-mono text-[var(--apple-text)]">
                  {{ target.endpoint }}
                </dd>
              </div>
              <div v-if="target.model" class="flex gap-2">
                <dt class="w-16 shrink-0 text-[var(--apple-muted-2)]">
                  {{ t('keys.ccSwitchDialog.model') }}
                </dt>
                <dd class="min-w-0 flex-1 break-all font-mono text-[var(--apple-text)]">
                  {{ target.model }}
                </dd>
              </div>
              <div class="flex gap-2">
                <dt class="w-16 shrink-0 text-[var(--apple-muted-2)]">
                  {{ t('keys.ccSwitchDialog.protocol') }}
                </dt>
                <dd class="min-w-0 flex-1">
                  <span
                    :class="[
                      'inline-flex rounded-md px-2 py-0.5 font-medium',
                      target.protocol.support === 'native'
                        ? 'bg-[color-mix(in_srgb,var(--apple-success)_10%,var(--apple-surface))] text-[var(--apple-success)]'
                        : 'bg-[color-mix(in_srgb,var(--apple-warning)_10%,var(--apple-surface))] text-[var(--apple-warning)]'
                    ]"
                  >
                    {{ getCcSwitchProtocolLabel(target.protocol.mode) }}
                  </span>
                </dd>
              </div>
              <div v-if="target.reasoningEffort" class="flex gap-2">
                <dt class="w-16 shrink-0 text-[var(--apple-muted-2)]">
                  {{ t('keys.ccSwitchDialog.reasoning') }}
                </dt>
                <dd class="min-w-0 flex-1 font-mono text-[var(--apple-text)]">
                  {{ target.reasoningEffort }}
                </dd>
              </div>
            </dl>

            <div class="mt-4 flex flex-wrap gap-2">
              <button
                class="btn btn-primary btn-sm inline-flex items-center gap-1.5"
                @click="openCcSwitchTarget(target.targetId)"
              >
                <Icon name="upload" size="sm" />
                {{ t('keys.ccSwitchDialog.import') }}
              </button>
              <button
                class="btn btn-secondary btn-sm inline-flex items-center gap-1.5"
                @click="copyCcSwitchTargetLink(target.targetId)"
              >
                <Icon :name="copiedCcsTargetId === target.targetId ? 'check' : 'copy'" size="sm" />
                {{ copiedCcsTargetId === target.targetId ? t('keys.ccSwitchDialog.linkCopied') : t('keys.ccSwitchDialog.copyLink') }}
              </button>
            </div>
          </div>
        </div>

        <p class="text-xs text-[var(--apple-muted)]">
          {{ t('keys.ccSwitchDialog.installHint') }}
        </p>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsImportDialog" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-[var(--apple-shadow-md)] ring-1 ring-[color:var(--apple-border-soft)] duration-200"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top + 'px',
          left: dropdownPosition.left + 'px',
          width: dropdownPosition.width + 'px'
        }"
      >
        <!-- Search box -->
        <div class="border-b border-[color:var(--apple-border-soft)] p-2">
          <div class="relative">
            <svg class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--apple-muted-2)]" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="w-full rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] py-1.5 pl-8 pr-3 text-sm text-[var(--apple-text)] placeholder-[var(--apple-muted-2)] outline-none focus:border-[color:var(--apple-blue)] focus:ring-1 focus:ring-[color:var(--apple-focus-ring)]"
              :placeholder="t('keys.searchGroup')"
              @click.stop
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="grid grid-cols-3 gap-2.5 border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-2.5">
          <button
            v-for="tab in groupPlatformTabs"
            :key="tab.value"
            type="button"
            @click.stop="activeGroupPlatformTab = tab.value"
            :class="groupPlatformTabClasses(tab.value)"
          >
            {{ tab.label }}
          </button>
        </div>
        <div class="overflow-y-auto p-1.5" :style="{ maxHeight: dropdownPosition.listMaxHeight + 'px' }">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'border-b border-[color:var(--apple-border-soft)] last:border-0',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'bg-[color-mix(in_srgb,var(--apple-blue)_10%,var(--apple-surface))]'
                : 'hover:bg-[var(--apple-hover)]'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :group-id="option.value"
              :rate-multiplier="option.rate"
              :user-rate-multiplier="option.userRate"
              :discount-multiplier="option.discountMultiplier"
              :discounted-rate-multiplier="option.discountedRateMultiplier"
              :discount-name="option.discountName"
              :discount-schedule-mode="option.discountScheduleMode"
              :discount-start-at="option.discountStartAt"
              :discount-end-at="option.discountEndAt"
              :discount-weekdays="option.discountWeekdays"
              :discount-daily-start-time="option.discountDailyStartTime"
              :discount-daily-end-time="option.discountDailyEndTime"
              :discount-timezone="option.discountTimezone"
              :peak-rate-enabled="option.peakRateEnabled"
              :peak-start="option.peakStart"
              :peak-end="option.peakEnd"
              :peak-rate-multiplier="option.peakRateMultiplier"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="py-4 text-center text-sm text-[var(--apple-muted-2)]">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/custom/composables/usePersistedPageSize'
import {
  DEFAULT_CNY_PER_CREDIT,
  convertSettlementAmount,
  setSettlementCnyPerCredit,
  useSettlementCurrency,
  type SettlementCurrency,
} from '@/custom/composables/useSettlementCurrency'
import { authAPI, userGroupsAPI } from '@/api'
import { keysAPI } from '@/custom/api/keys'
import { usageAPI } from '@/custom/api/usage'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import TablePageLayout from '@/custom/layout/WegooTablePageLayout.vue'
import DataTable from '@/custom/common/WegooDataTable.vue'
import Pagination from '@/custom/common/WegooPagination.vue'
import BaseDialog from '@/custom/common/WegooBaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/custom/common/WegooEmptyState.vue'
import Select from '@/custom/common/WegooSelect.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import UseKeyModal from '@/custom/keys/WegooUseKeyModal.vue'
import EndpointPopover from '@/custom/keys/WegooEndpointPopover.vue'
import GroupBadge from '@/custom/common/WegooGroupBadge.vue'
import GroupOptionItem from '@/custom/common/WegooGroupOptionItem.vue'
import type { ApiKey, Group, PublicSettings, SubscriptionType, GroupPlatform, UpdateApiKeyRequest } from '@/types'
import type { SelectOption } from '@/custom/common/WegooSelect.vue'
import type { Column } from '@/components/common/types'
import type { BatchApiKeyUsageStats } from '@/custom/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'
import { formatRateMultiplier, resolveGroupRateDiscount } from '@/custom/utils/groupRateDiscount'
import {
  buildCcSwitchImportDeeplink,
  listCcSwitchImportTargets,
  type CcSwitchProtocolMode,
  type CcSwitchTargetId
} from '@/custom/keys/ccswitchImport'

const { t } = useI18n()

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption extends SelectOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
  groupKey: KeyGroupPlatformTab
  discountMultiplier?: number | null
  discountedRateMultiplier?: number | null
  discountName?: string | null
  discountScheduleMode?: string | null
  discountStartAt?: string | null
  discountEndAt?: string | null
  discountWeekdays?: number[] | null
  discountDailyStartTime?: string | null
  discountDailyEndTime?: string | null
  discountTimezone?: string | null
}

interface GroupTabOption extends SelectOption {
  value: KeyGroupPlatformTab
  label: string
  kind: 'group'
  disabled: true
  groupKey: KeyGroupPlatformTab
}

type KeyGroupPlatformTab = 'openai' | 'anthropic' | 'other'

type KeyGroupSelectOption = GroupOption | GroupTabOption

const appStore = useAppStore()
const { copyToClipboard: clipboardCopy } = useClipboard()
const {
  settlementCurrency,
  settlementAmountPrefix,
  cnyPerCredit,
  formatSettlementAmount,
  formatSettlementAmountPair,
  toBalanceCreditAmount,
} = useSettlementCurrency()

const allColumns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'current_concurrency', label: t('keys.currentConcurrency'), sortable: true },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'rate_limit', label: t('keys.rateLimitColumn'), sortable: false },
  { key: 'expires_at', label: t('keys.expiresAt'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'last_used_ip', label: t('keys.lastUsedIP'), sortable: false },
  { key: 'created_at', label: t('keys.created'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const ALWAYS_VISIBLE_COLUMNS = new Set(['name', 'actions'])
const DEFAULT_HIDDEN_COLUMNS = ['current_concurrency', 'rate_limit', 'last_used_at', 'last_used_ip']
const HIDDEN_COLUMNS_KEY = 'api-key-hidden-columns'
const COLUMN_SETTINGS_VERSION_KEY = 'api-key-column-settings-version'
const COLUMN_SETTINGS_VERSION = 3
const VERSION_NEW_HIDDEN_COLUMNS: Record<number, string[]> = {
  2: ['last_used_ip'],
  3: ['current_concurrency']
}
const hiddenColumns = reactive<Set<string>>(new Set())

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => !ALWAYS_VISIBLE_COLUMNS.has(col.key))
)

const saveColumnSettings = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
  } catch (error) {
    console.error('Failed to save API key table columns:', error)
  }
}

const loadColumnSettings = () => {
  hiddenColumns.clear()
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      const validColumnKeys = new Set(allColumns.value.map((col) => col.key))
      parsed
        .filter((key) =>
          typeof key === 'string' &&
          validColumnKeys.has(key) &&
          !ALWAYS_VISIBLE_COLUMNS.has(key)
        )
        .forEach((key) => hiddenColumns.add(key))
      const storedVersion = Number(localStorage.getItem(COLUMN_SETTINGS_VERSION_KEY) ?? '1')
      if (storedVersion < COLUMN_SETTINGS_VERSION) {
        for (let v = storedVersion + 1; v <= COLUMN_SETTINGS_VERSION; v++) {
          for (const key of VERSION_NEW_HIDDEN_COLUMNS[v] ?? []) {
            if (validColumnKeys.has(key) && !ALWAYS_VISIBLE_COLUMNS.has(key)) {
              hiddenColumns.add(key)
            }
          }
        }
        saveColumnSettings()
      } else {
        localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
      localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
    }
  } catch (error) {
    console.error('Failed to load API key table columns:', error)
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

const toggleColumn = (key: string) => {
  if (ALWAYS_VISIBLE_COLUMNS.has(key)) return
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnSettings()
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const columns = computed<Column[]>(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE_COLUMNS.has(col.key) || !hiddenColumns.has(col.key))
)

const ccSwitchTargetIcon: Record<CcSwitchTargetId, 'terminal' | 'sparkles' | 'cpu' | 'cube'> = {
  'claude-code': 'terminal',
  'claude-desktop': 'terminal',
  codex: 'cpu',
  'gemini-cli': 'sparkles',
  opencode: 'terminal',
  openclaw: 'cube',
  hermes: 'sparkles'
}

const ccSwitchTargetI18nKey: Record<CcSwitchTargetId, string> = {
  'claude-code': 'claudeCode',
  'claude-desktop': 'claudeDesktop',
  codex: 'codex',
  'gemini-cli': 'geminiCli',
  opencode: 'opencode',
  openclaw: 'openclaw',
  hermes: 'hermes'
}

const ccSwitchProtocolI18nKey: Record<CcSwitchProtocolMode, string> = {
  'anthropic-messages': 'anthropicMessages',
  'openai-responses': 'openaiResponses',
  'gemini-native': 'geminiNative',
  'openai-compatible': 'openaiCompatible',
  'openai-completions': 'openaiCompletions',
  'chat-completions': 'chatCompletions'
}

const ccSwitchOfficialUrl = 'https://ccswitch.io'

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const userGroupRates = ref<Record<number, number>>({})

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = ref({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

// Filter state
const filterSearch = ref('')
const filterStatus = ref('')
const filterGroupId = ref<string | number>('')

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsImportDialog = ref(false)
const showColumnDropdown = ref(false)
const selectedCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const copiedKeyId = ref<number | null>(null)
const copiedCcsTargetId = ref<CcSwitchTargetId | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const activeGroupPlatformTab = ref<KeyGroupPlatformTab>('openai')
const publicSettings = ref<PublicSettings | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const columnDropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top: number; left: number; width: number; listMaxHeight: number } | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())
let abortController: AbortController | null = null

const primaryBaseUrl = computed(() => normalizePrimaryBaseUrl(publicSettings.value?.api_base_url || appStore.apiBaseUrl))

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const ccsImportTargets = computed(() => {
  const row = selectedCcsRow.value
  if (!row?.group?.platform) return []
  return listCcSwitchImportTargets({
    baseUrl: primaryBaseUrl.value,
    platform: row.group.platform
  })
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
})

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('keys.statusToggle.on') },
  { value: 'inactive', label: t('keys.statusToggle.off') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

const groupPlatformTabs = computed<GroupTabOption[]>(() => [
  { value: 'openai', label: t('keys.categories.openai'), kind: 'group', disabled: true, groupKey: 'openai' },
  { value: 'anthropic', label: t('keys.categories.anthropic'), kind: 'group', disabled: true, groupKey: 'anthropic' },
  { value: 'other', label: t('keys.categories.other'), kind: 'group', disabled: true, groupKey: 'other' }
])

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: '', label: t('keys.allGroups') },
  { value: 0, label: t('keys.noGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  pagination.value.page = 1
  loadApiKeys()
}

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number
  onFilterChange()
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

const keyGroupPlatformTab = (platform?: GroupPlatform | string | null): KeyGroupPlatformTab => {
  if (platform === 'openai') return 'openai'
  if (platform === 'anthropic') return 'anthropic'
  return 'other'
}

const groupPlatformTabClasses = (tab: KeyGroupPlatformTab) => {
  const active = activeGroupPlatformTab.value === tab
  const base = [
    'relative inline-flex min-h-10 items-center justify-center rounded-lg border px-3 py-2 text-center text-sm font-medium transition-colors duration-150',
    'before:mr-1.5 before:inline-block before:h-2 before:w-2 before:rounded-full before:align-middle before:content-[\'\']'
  ]

  if (tab === 'openai') {
    return [
      ...base,
      active
        ? 'border-[color:color-mix(in_srgb,var(--apple-success)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_10%,var(--apple-surface))] text-[var(--apple-success)] before:bg-[var(--apple-success)]'
        : 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] text-[var(--apple-muted)] before:bg-[var(--apple-success)] hover:bg-[color-mix(in_srgb,var(--apple-success)_7%,var(--apple-surface))] hover:text-[var(--apple-success)]'
    ]
  }

  if (tab === 'anthropic') {
    return [
      ...base,
      active
        ? 'border-[color:color-mix(in_srgb,var(--apple-warning)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_10%,var(--apple-surface))] text-[var(--apple-warning)] before:bg-[var(--apple-warning)]'
        : 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] text-[var(--apple-muted)] before:bg-[var(--apple-warning)] hover:bg-[color-mix(in_srgb,var(--apple-warning)_7%,var(--apple-surface))] hover:text-[var(--apple-warning)]'
    ]
  }

  return [
    ...base,
    active
      ? 'border-[color:color-mix(in_srgb,var(--apple-blue)_34%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-blue)_10%,var(--apple-surface))] text-[var(--apple-blue)] before:bg-[var(--apple-blue)]'
      : 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] text-[var(--apple-muted)] before:bg-[var(--apple-blue)] hover:bg-[color-mix(in_srgb,var(--apple-blue)_7%,var(--apple-surface))] hover:text-[var(--apple-blue)]'
  ]
}

// Convert groups to Select options format with rate multiplier and subscription type
const rawGroupOptions = computed<GroupOption[]>(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    peakRateEnabled: group.peak_rate_enabled,
    peakStart: group.peak_start,
    peakEnd: group.peak_end,
    peakRateMultiplier: group.peak_rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform,
    groupKey: keyGroupPlatformTab(group.platform),
    discountMultiplier: group.group_rate_discount_multiplier,
    discountedRateMultiplier: group.discounted_rate_multiplier,
    discountName: group.group_rate_discount_name,
    discountScheduleMode: group.group_rate_discount_schedule_mode,
    discountStartAt: group.group_rate_discount_start_at,
    discountEndAt: group.group_rate_discount_end_at,
    discountWeekdays: group.group_rate_discount_weekdays,
    discountDailyStartTime: group.group_rate_discount_daily_start_time,
    discountDailyEndTime: group.group_rate_discount_daily_end_time,
    discountTimezone: group.group_rate_discount_timezone
  }))
)

const groupOptions = computed<KeyGroupSelectOption[]>(() => [
  ...groupPlatformTabs.value,
  ...rawGroupOptions.value
])

const selectedGroupForForm = computed<GroupOption | null>(() => {
  if (formData.value.group_id === null) return null
  return rawGroupOptions.value.find((option) => option.value === formData.value.group_id) ?? null
})

const selectedGroupEffectiveRate = computed<number | null>(() => {
  const group = selectedGroupForForm.value
  if (!group) return null
  const baseRate = group.userRate ?? group.rate
  const discount = resolveGroupRateDiscount(
    group.value,
    baseRate,
    appStore.cachedPublicSettings?.group_rate_discount ?? null,
    {
      multiplier: group.discountMultiplier,
      discountedRate: group.discountedRateMultiplier,
      name: group.discountName,
      scheduleMode: group.discountScheduleMode,
      startAt: group.discountStartAt,
      endAt: group.discountEndAt,
      weekdays: group.discountWeekdays,
      dailyStartTime: group.discountDailyStartTime,
      dailyEndTime: group.discountDailyEndTime,
      timezone: group.discountTimezone
    },
    false,
    now.value.getTime()
  )
  const effectiveRate = Number(discount?.discountedRate ?? baseRate)
  return Number.isFinite(effectiveRate) && effectiveRate > 0 ? effectiveRate : null
})

const resolveOfficialUsdCnyRate = (value: unknown): number => {
  const numeric = Number(value)
  if (Number.isFinite(numeric) && numeric >= 2) return numeric
  return DEFAULT_CNY_PER_CREDIT
}

const selectedGroupCostPreview = computed(() => {
  const rate = selectedGroupEffectiveRate.value
  if (!rate) return t('keys.groupCostPreview.unavailable')
  const officialUsdCnyRate = resolveOfficialUsdCnyRate(publicSettings.value?.payment_balance_recharge_multiplier)
  const officialCnyMultiple = rate / officialUsdCnyRate
  return t('keys.groupCostPreview.description', {
    rate: `${formatRateMultiplier(rate)}x`,
    cny: formatRateMultiplier(officialUsdCnyRate),
    multiple: `${Number(officialCnyMultiple.toFixed(2))}`,
  })
})

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  const options = rawGroupOptions.value.filter((opt) => opt.groupKey === activeGroupPlatformTab.value)
  if (!query) return options
  return options.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    // Build filters
    const filters: {
      search?: string
      status?: string
      group_id?: number | string
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    } = {}
    if (filterSearch.value) filters.search = filterSearch.value
    if (filterStatus.value) filters.status = filterStatus.value
    if (filterGroupId.value !== '') filters.group_id = filterGroupId.value
    filters.sort_by = sortState.value.sort_by
    filters.sort_order = sortState.value.sort_order

    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, filters, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    pagination.value.total = response.total
    pagination.value.pages = response.pages

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!isAbortError(e)) {
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates()
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await appStore.fetchPublicSettings(true) || await authAPI.getPublicSettings()
    setSettlementCnyPerCredit(publicSettings.value?.payment_balance_recharge_multiplier)
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const toSettlementInputAmount = (
  amount: number | null | undefined,
  currency: SettlementCurrency = settlementCurrency.value,
): number | null => {
  const value = Number(amount)
  if (!Number.isFinite(value) || value <= 0) return null
  return Number(convertSettlementAmount(value, currency, cnyPerCredit.value).toFixed(4))
}

const toStoredBalanceAmount = (amount: number | null | undefined): number => {
  const value = Number(amount)
  if (!Number.isFinite(value) || value <= 0) return 0
  return Number(toBalanceCreditAmount(value).toFixed(4))
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.value.sort_by = key
  sortState.value.sort_order = order
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: toSettlementInputAmount(key.quota),
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: toSettlementInputAmount(key.rate_limit_5h),
    rate_limit_1d: toSettlementInputAmount(key.rate_limit_1d),
    rate_limit_7d: toSettlementInputAmount(key.rate_limit_7d),
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : ''
  }
  showEditModal.value = true
}

const getClampedGroupDropdownPosition = (rect: DOMRect) => {
  const viewportPadding = 12
  const preferredWidth = 380
  const availableWidth = Math.max(window.innerWidth - viewportPadding * 2, 240)
  const width = Math.min(preferredWidth, availableWidth)
  const left = Math.min(Math.max(rect.left, viewportPadding), window.innerWidth - viewportPadding - width)
  const minListHeight = 160
  const maxListHeight = 320
  const fixedDropdownChrome = 120
  const dropdownGap = 4
  const spaceBelow = window.innerHeight - rect.bottom - viewportPadding - dropdownGap
  const spaceAbove = rect.top - viewportPadding - dropdownGap
  const opensUp = spaceBelow < minListHeight && spaceAbove > spaceBelow
  const availableHeight = Math.max(opensUp ? spaceAbove : spaceBelow, minListHeight)
  const listMaxHeight = Math.min(maxListHeight, Math.max(minListHeight, availableHeight - fixedDropdownChrome))
  const dropdownHeight = fixedDropdownChrome + listMaxHeight
  const top = opensUp
    ? Math.max(viewportPadding, rect.top - dropdownHeight - dropdownGap)
    : Math.min(rect.bottom + dropdownGap, window.innerHeight - viewportPadding - dropdownHeight)

  return { top, left, width, listMaxHeight }
}

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      dropdownPosition.value = getClampedGroupDropdownPosition(rect)
    }
    groupSelectorKeyId.value = key.id
    groupSearchQuery.value = ''
    activeGroupPlatformTab.value = keyGroupPlatformTab(key.group?.platform)
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  try {
    await keysAPI.update(key.id, { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = toStoredBalanceAmount(formData.value.quota)

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: toStoredBalanceAmount(formData.value.rate_limit_5h),
    rate_limit_1d: toStoredBalanceAmount(formData.value.rate_limit_1d),
    rate_limit_7d: toStoredBalanceAmount(formData.value.rate_limit_7d),
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      const updates: UpdateApiKeyRequest = {
        name: formData.value.name,
        group_id: formData.value.group_id,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      }
      if (shouldSubmitEditStatus(selectedKey.value, formData.value.status)) {
        updates.status = formData.value.status
      }
      await keysAPI.update(selectedKey.value.id, updates)
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      const createdKey = await keysAPI.create({
        name: formData.value.name,
        group_id: formData.value.group_id,
        custom_key: customKey,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota,
        expires_in_days: expiresInDays,
        ...rateLimitData
      })
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      selectedKey.value = createdKey
      showUseKeyModal.value = true
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除访问凭证的操作
 * 优化：问题处理改进，优先显示平台返回的具体提示（如权限不足等），
 * 若平台未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用平台返回的问题提示，提供更具体的信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  if (!showUseKeyModal.value) {
    selectedKey.value = null
  }
  formData.value = {
    name: '',
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: '',
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: '30',
    expiration_date: ''
  }
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return
  showResetQuotaDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetQuota')
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Show reset rate limit confirmation dialog (from table row)
const confirmResetRateLimitFromTable = (row: ApiKey) => {
  selectedKey.value = row
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetRateLimit')
    appStore.showError(errorMsg)
  }
}

const importToCcswitch = (row: ApiKey) => {
  selectedCcsRow.value = row
  copiedCcsTargetId.value = null
  showCcsImportDialog.value = true
}

const buildCcSwitchUsageScript = () => `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`

const getCcSwitchProviderName = (targetId: CcSwitchTargetId): string => {
  const siteName = (publicSettings.value?.site_name || 'sub2api').trim() || 'sub2api'
  return `${siteName} ${getCcSwitchTargetName(targetId)}`
}

const buildCcSwitchTargetLink = (targetId: CcSwitchTargetId): string => {
  const row = selectedCcsRow.value
  if (!row?.group?.platform) return ''

  return buildCcSwitchImportDeeplink({
    baseUrl: primaryBaseUrl.value,
    platform: row.group.platform,
    targetId,
    providerName: getCcSwitchProviderName(targetId),
    apiKey: row.key,
    usageScript: buildCcSwitchUsageScript()
  })
}

const getCcSwitchTargetName = (targetId: CcSwitchTargetId): string =>
  t(`keys.ccSwitchDialog.targets.${ccSwitchTargetI18nKey[targetId]}.name`)

const getCcSwitchTargetDescription = (targetId: CcSwitchTargetId): string =>
  t(`keys.ccSwitchDialog.targets.${ccSwitchTargetI18nKey[targetId]}.description`)

const getCcSwitchProtocolLabel = (protocol: CcSwitchProtocolMode): string =>
  t(`keys.ccSwitchDialog.protocols.${ccSwitchProtocolI18nKey[protocol]}`)

const openCcSwitchTarget = (targetId: CcSwitchTargetId) => {
  const deeplink = buildCcSwitchTargetLink(targetId)
  if (!deeplink) {
    appStore.showError(t('keys.ccSwitchDialog.noGroup'))
    return
  }

  try {
    window.open(deeplink, '_self')

    setTimeout(() => {
      if (document.hasFocus()) {
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 500)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const copyCcSwitchTargetLink = async (targetId: CcSwitchTargetId) => {
  const deeplink = buildCcSwitchTargetLink(targetId)
  if (!deeplink) {
    appStore.showError(t('keys.ccSwitchDialog.noGroup'))
    return
  }

  const success = await clipboardCopy(deeplink, t('keys.ccSwitchDialog.linkCopied'))
  if (success) {
    copiedCcsTargetId.value = targetId
    setTimeout(() => {
      if (copiedCcsTargetId.value === targetId) {
        copiedCcsTargetId.value = null
      }
    }, 1200)
  }
}

const closeCcsImportDialog = () => {
  showCcsImportDialog.value = false
  selectedCcsRow.value = null
  copiedCcsTargetId.value = null
}

function normalizePrimaryBaseUrl(value: string | undefined): string {
  const trimmed = value?.trim().replace(/\/+$/, '')
  if (trimmed) return trimmed
  if (typeof window !== 'undefined' && window.location.origin) {
    return `${window.location.origin}/v1`
  }
  return '/v1'
}

watch(cnyPerCredit, () => {
  if (showEditModal.value && selectedKey.value) {
    formData.value.quota = toSettlementInputAmount(selectedKey.value.quota)
    formData.value.rate_limit_5h = toSettlementInputAmount(selectedKey.value.rate_limit_5h)
    formData.value.rate_limit_1d = toSettlementInputAmount(selectedKey.value.rate_limit_1d)
    formData.value.rate_limit_7d = toSettlementInputAmount(selectedKey.value.rate_limit_7d)
  }
})

onMounted(() => {
  loadColumnSettings()
  loadApiKeys()
  loadGroups()
  loadUserGroupRates()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  document.removeEventListener('click', closeGroupSelector)
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
.key-row-group {
  display: inline-flex;
  width: max-content;
  max-width: 100%;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem 0.75rem;
}

.key-row-actions {
  display: flex;
  min-width: 0;
  width: max-content;
  max-width: 100%;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.375rem;
}

.key-row-icon-action {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  transition: color 150ms ease, background-color 150ms ease;
}

.key-row-icon-action:hover {
  background: var(--apple-hover);
}
</style>
