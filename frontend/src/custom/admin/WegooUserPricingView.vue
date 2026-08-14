<template>
  <AppLayout>
    <main class="mx-auto w-full min-w-0 max-w-7xl space-y-5">
      <header class="flex flex-col gap-4 border-b border-gray-200 pb-5 dark:border-dark-700 lg:flex-row lg:items-end lg:justify-between">
        <div class="min-w-0">
          <p class="text-xs font-semibold text-primary-600 dark:text-primary-400">{{ t('admin.userPricing.kicker') }}</p>
          <h1 class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.userPricing.title') }}</h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">
            {{ t('admin.userPricing.description') }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2">
          <button
            type="button"
            class="btn btn-secondary px-3"
            :disabled="loadingGroups || loadingConfiguredUsers || loadingUser || saving"
            :title="t('common.refresh')"
            @click="refresh"
          >
            <Icon name="refresh" size="sm" :class="(loadingGroups || loadingConfiguredUsers || loadingUser) && 'animate-spin'" />
            <span>{{ t('common.refresh') }}</span>
          </button>
          <button
            type="button"
            class="btn btn-primary px-4"
            :disabled="!selectedUser || loadingUser || saving || groups.length === 0"
            @click="save"
          >
            <Icon name="check" size="sm" />
            <span>{{ saving ? t('common.saving') : t('common.save') }}</span>
          </button>
        </div>
      </header>

      <section class="min-w-0 border-b border-gray-200 pb-5 dark:border-dark-700">
        <label class="block max-w-3xl">
          <span class="mb-2 block text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.userPricing.searchUser') }}</span>
          <div class="flex gap-2">
            <div class="relative min-w-0 flex-1">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                v-model.trim="userSearch"
                type="search"
                class="input h-10 w-full pl-9"
                :placeholder="t('admin.userPricing.findUserPlaceholder')"
                @keydown.enter.prevent="searchUsers"
              />
            </div>
            <button type="button" class="btn btn-secondary h-10 px-3" :disabled="searchingUsers || !userSearch" @click="searchUsers">
              <Icon name="search" size="sm" :class="searchingUsers && 'animate-pulse'" />
              <span>{{ t('common.search') }}</span>
            </button>
          </div>
        </label>

        <div v-if="searchResults.length > 0" class="mt-2 max-w-3xl overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <button
            v-for="user in searchResults"
            :key="user.id"
            type="button"
            class="flex w-full items-center justify-between gap-3 border-b border-gray-100 px-3 py-2.5 text-left last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-800"
            @click="selectUser(user)"
          >
            <span class="min-w-0">
              <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
              <span class="block truncate text-xs text-gray-500 dark:text-gray-400">
                #{{ user.id }}{{ user.username ? ` · ${user.username}` : '' }}
              </span>
            </span>
            <span class="shrink-0 text-xs font-medium text-primary-600 dark:text-primary-400">{{ t('admin.userPricing.select') }}</span>
          </button>
        </div>

        <div class="mt-6 space-y-3">
          <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.userPricing.configuredUsersTitle') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.userPricing.configuredUsersDescription') }}</p>
            </div>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.userPricing.configuredUsersCount', { count: configuredUsersPagination.total }) }}
            </span>
          </div>

          <div class="max-w-full overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div v-if="loadingConfiguredUsers" class="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('common.loading') }}
            </div>
            <div v-else-if="configuredUsers.length === 0" class="px-4 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.userPricing.noConfiguredUsers') }}
            </div>
            <div v-else class="max-w-full overflow-x-auto">
              <table class="w-full min-w-[820px] border-collapse text-sm">
                <thead class="bg-gray-50 text-left text-xs font-semibold text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                  <tr>
                    <th class="w-64 px-4 py-3">{{ t('admin.userPricing.user') }}</th>
                    <th class="px-4 py-3">{{ t('admin.userPricing.configuredGroups') }}</th>
                    <th class="w-28 px-4 py-3">{{ t('admin.userPricing.configuredCountLabel') }}</th>
                    <th class="w-24 px-4 py-3">{{ t('admin.userPricing.status') }}</th>
                    <th class="w-24 px-4 py-3 text-right">{{ t('admin.userPricing.actions') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="user in configuredUsers"
                    :key="user.id"
                    class="border-t border-gray-100 dark:border-dark-700"
                    :class="selectedUser?.id === user.id && 'bg-primary-50 dark:bg-primary-500/10'"
                  >
                    <td class="px-4 py-3">
                      <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
                      <p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
                        #{{ user.id }}{{ user.username ? ` · ${user.username}` : '' }}
                      </p>
                    </td>
                    <td class="px-4 py-3">
                      <div class="flex max-w-3xl flex-wrap gap-1.5">
                        <span
                          v-for="entry in configuredEntries(user)"
                          :key="`${user.id}-${entry.groupID}-${entry.kind}`"
                          class="inline-flex items-center gap-1 border px-2 py-1 text-xs"
                          :class="entry.kind === 'fixed'
                            ? 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-300'
                            : 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-900/20 dark:text-emerald-300'"
                        >
                          <span>{{ groupName(entry.groupID) }}</span>
                          <span class="font-semibold tabular-nums">{{ formatMultiplier(entry.value) }}x</span>
                          <span class="text-[10px] opacity-75">{{ entry.kind === 'fixed' ? t('admin.userPricing.fixedShort') : t('admin.userPricing.coefficientShort') }}</span>
                        </span>
                      </div>
                    </td>
                    <td class="px-4 py-3 tabular-nums text-gray-900 dark:text-white">{{ configuredEntries(user).length }}</td>
                    <td class="px-4 py-3">
                      <span :class="user.status === 'active' ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                        {{ user.status === 'active' ? t('admin.userPricing.active') : t('admin.userPricing.inactive') }}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-right">
                      <button type="button" class="btn btn-secondary px-2.5" @click="selectUser(user)">
                        <Icon name="edit" size="sm" />
                        <span>{{ t('admin.userPricing.edit') }}</span>
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <Pagination
              v-if="configuredUsersPagination.total > 0"
              :page="configuredUsersPagination.page"
              :total="configuredUsersPagination.total"
              :page-size="configuredUsersPagination.page_size"
              @update:page="handleConfiguredUsersPageChange"
              @update:pageSize="handleConfiguredUsersPageSizeChange"
            />
          </div>
        </div>

        <div v-if="selectedUser" class="mt-4 flex flex-col gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ selectedUser.email }}</p>
            <p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
              #{{ selectedUser.id }}{{ selectedUser.username ? ` · ${selectedUser.username}` : '' }}
            </p>
          </div>
          <div class="flex shrink-0 flex-wrap items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('admin.userPricing.groupCount', { count: groups.length }) }}</span>
            <span>{{ t('admin.userPricing.configuredCount', { count: configuredCount }) }}</span>
            <span :class="selectedUser.status === 'active' ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
              {{ selectedUser.status === 'active' ? t('admin.userPricing.active') : t('admin.userPricing.inactive') }}
            </span>
          </div>
        </div>
      </section>

      <section v-if="selectedUser" class="min-w-0 space-y-4">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.userPricing.groupTableTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.userPricing.groupTableDescription') }}</p>
          </div>
          <button
            type="button"
            class="btn btn-secondary text-red-600 dark:text-red-400"
            :disabled="saving || loadingUser || editableDiscountCount === 0"
            @click="showClearDialog = true"
          >
            <Icon name="trash" size="sm" />
            <span>{{ t('admin.userPricing.clearUser') }}</span>
          </button>
        </div>

        <div class="max-w-full overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="max-w-full overflow-x-auto">
            <table class="w-full min-w-[860px] border-collapse text-sm">
              <thead class="bg-gray-50 text-left text-xs font-semibold text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                <tr>
                  <th class="px-4 py-3">{{ t('admin.userPricing.group') }}</th>
                  <th class="w-32 px-4 py-3">{{ t('admin.userPricing.groupRate') }}</th>
                  <th class="w-36 px-4 py-3">{{ t('admin.userPricing.fixedRate') }}</th>
                  <th class="w-40 px-4 py-3">{{ t('admin.userPricing.coefficient') }}</th>
                  <th class="w-36 px-4 py-3">{{ t('admin.userPricing.finalRate') }}</th>
                  <th class="w-28 px-4 py-3">{{ t('admin.userPricing.status') }}</th>
                  <th class="w-16 px-4 py-3 text-right">{{ t('admin.userPricing.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="loadingUser">
                  <td colspan="7" class="px-4 py-12 text-center text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</td>
                </tr>
                <tr v-else-if="groupRows.length === 0" class="border-t border-gray-100 dark:border-dark-700">
                  <td colspan="7" class="px-4 py-12 text-center text-gray-500 dark:text-gray-400">{{ t('admin.userPricing.emptyGroups') }}</td>
                </tr>
                <tr v-for="row in groupRows" :key="row.group.id" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="px-4 py-3">
                    <p class="font-medium text-gray-900 dark:text-white">{{ row.group.name }}</p>
                    <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ row.group.platform.toUpperCase() }} · {{ row.group.status === 'active' ? t('admin.userPricing.active') : t('admin.userPricing.inactive') }}
                    </p>
                  </td>
                  <td class="px-4 py-3 tabular-nums text-gray-500 dark:text-gray-400">{{ formatMultiplier(row.group.rate_multiplier) }}x</td>
                  <td class="px-4 py-3 tabular-nums">
                    <span v-if="row.fixedRate !== null" class="font-semibold text-amber-600 dark:text-amber-400">{{ formatMultiplier(row.fixedRate) }}x</span>
                    <span v-else class="text-gray-500 dark:text-gray-400">-</span>
                  </td>
                  <td class="px-4 py-3">
                    <div v-if="row.fixedRate === null" class="relative w-32">
                      <input
                        v-model.number="row.coefficient"
                        type="number"
                        min="0.0001"
                        max="100"
                        step="0.0001"
                        class="input h-9 w-full pr-7 tabular-nums"
                        placeholder="1"
                      />
                      <span class="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-xs text-gray-500 dark:text-gray-400">x</span>
                    </div>
                    <span v-else class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.userPricing.fixedRateLocked') }}</span>
                  </td>
                  <td class="px-4 py-3 tabular-nums font-semibold text-gray-900 dark:text-white">{{ formatMultiplier(finalRate(row)) }}x</td>
                  <td class="px-4 py-3">
                    <span
                      class="inline-flex items-center border px-2 py-1 text-xs"
                      :class="rowStatusClass(row)"
                    >
                      {{ rowStatusLabel(row) }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      v-if="row.fixedRate === null && row.coefficient !== null"
                      type="button"
                      class="btn btn-ghost px-2 text-red-600 dark:text-red-400"
                      :title="t('admin.userPricing.clearRow')"
                      @click="row.coefficient = null"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                    <span v-else class="text-gray-500 dark:text-gray-400">-</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

    </main>

    <ConfirmDialog
      :show="showClearDialog"
      :title="t('admin.userPricing.clearUser')"
      :message="t('admin.userPricing.clearConfirm', { user: selectedUser?.email || '' })"
      :confirm-text="t('common.clear')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="clearUserDiscounts"
      @cancel="showClearDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminAPI } from '@/custom/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, AdminUser } from '@/types'

interface UserPricingRow {
  group: AdminGroup
  fixedRate: number | null
  coefficient: number | null
}

interface ConfiguredPricingEntry {
  groupID: number
  value: number
  kind: 'fixed' | 'discount'
}

const { t } = useI18n()
const appStore = useAppStore()

const groups = ref<AdminGroup[]>([])
const configuredUsers = ref<AdminUser[]>([])
const selectedUser = ref<AdminUser | null>(null)
const groupRows = ref<UserPricingRow[]>([])
const userSearch = ref('')
const searchResults = ref<AdminUser[]>([])
const loadingGroups = ref(false)
const loadingConfiguredUsers = ref(false)
const loadingUser = ref(false)
const searchingUsers = ref(false)
const saving = ref(false)
const showClearDialog = ref(false)

const configuredUsersPagination = ref({
  page: 1,
  page_size: 20,
  total: 0,
})

const groupsByID = computed(() => new Map(groups.value.map((group) => [group.id, group])))
const configuredCount = computed(() => groupRows.value.filter(hasEffectivePricingChange).length)
const editableDiscountCount = computed(() => groupRows.value.filter((row) => (
  row.fixedRate === null && row.coefficient !== null && !sameMultiplier(row.coefficient, 1)
)).length)

function formatMultiplier(value: number | null | undefined): string {
  const numberValue = Number(value)
  if (!Number.isFinite(numberValue)) return '-'
  return numberValue.toFixed(6).replace(/0+$/, '').replace(/\.$/, '') || '0'
}

function buildRows(user: AdminUser): void {
  const fixedRates = user.group_rates || {}
  const discounts = user.group_discounts || {}
  groupRows.value = groups.value.map((group) => {
    const fixedValue = Number(fixedRates[group.id])
    const discountValue = Number(discounts[group.id])
    return {
      group,
      fixedRate: Number.isFinite(fixedValue) && fixedValue > 0 ? fixedValue : null,
      coefficient: Number.isFinite(discountValue) && discountValue > 0 ? discountValue : null,
    }
  })
}

function sameMultiplier(left: number, right: number): boolean {
  return Math.abs(left - right) < 1e-9
}

function hasEffectivePricingChange(row: UserPricingRow): boolean {
  if (row.fixedRate !== null) return !sameMultiplier(row.fixedRate, row.group.rate_multiplier)
  return row.coefficient !== null && !sameMultiplier(row.coefficient, 1)
}

function groupName(groupID: number): string {
  return groupsByID.value.get(groupID)?.name || `#${groupID}`
}

function configuredEntries(user: AdminUser): ConfiguredPricingEntry[] {
  const entries: ConfiguredPricingEntry[] = []
  const fixedRates = user.group_rates || {}
  const discounts = user.group_discounts || {}
  for (const [groupID, value] of Object.entries(user.group_rates || {})) {
    const id = Number(groupID)
    const numberValue = Number(value)
    const group = groupsByID.value.get(id)
    if (Number.isFinite(numberValue) && numberValue > 0 && (!group || !sameMultiplier(numberValue, group.rate_multiplier))) {
      entries.push({ groupID: id, value: numberValue, kind: 'fixed' })
    }
  }
  for (const [groupID, value] of Object.entries(discounts)) {
    const id = Number(groupID)
    const numberValue = Number(value)
    const fixedValue = Number(fixedRates[id])
    if (
      (!Number.isFinite(fixedValue) || fixedValue <= 0)
      && Number.isFinite(numberValue)
      && numberValue > 0
      && !sameMultiplier(numberValue, 1)
    ) {
      entries.push({ groupID: id, value: numberValue, kind: 'discount' })
    }
  }
  return entries.sort((a, b) => {
    const groupOrder = (groupsByID.value.get(a.groupID)?.sort_order ?? Number.MAX_SAFE_INTEGER)
      - (groupsByID.value.get(b.groupID)?.sort_order ?? Number.MAX_SAFE_INTEGER)
    return groupOrder || a.groupID - b.groupID || a.kind.localeCompare(b.kind)
  })
}

function finalRate(row: UserPricingRow): number {
  if (row.fixedRate !== null) return row.fixedRate
  const coefficient = row.coefficient === null ? 1 : Number(row.coefficient)
  if (!Number.isFinite(coefficient)) return 0
  return row.group.rate_multiplier * coefficient
}

function rowStatusLabel(row: UserPricingRow): string {
  if (row.fixedRate !== null) return t('admin.userPricing.fixedRate')
  if (row.coefficient === null || sameMultiplier(row.coefficient, 1)) return t('admin.userPricing.default')
  return row.coefficient < 1 ? t('admin.userPricing.discount') : t('admin.userPricing.markup')
}

function rowStatusClass(row: UserPricingRow): string {
  if (row.fixedRate !== null) {
    return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-300'
  }
  if (row.coefficient === null || sameMultiplier(row.coefficient, 1)) {
    return 'border-gray-200 text-gray-500 dark:border-dark-600 dark:text-gray-400'
  }
  return row.coefficient < 1
    ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-900/20 dark:text-emerald-300'
    : 'border-primary-200 bg-primary-50 text-primary-700 dark:border-primary-800/50 dark:bg-primary-900/20 dark:text-primary-300'
}

async function loadGroups(): Promise<void> {
  loadingGroups.value = true
  try {
    const result = await adminAPI.groups.getAllIncludingInactive()
    groups.value = result
      .filter((group) => group.subscription_type === 'standard')
      .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    if (selectedUser.value) buildRows(selectedUser.value)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loadingGroups.value = false
  }
}

async function loadConfiguredUsers(): Promise<void> {
  loadingConfiguredUsers.value = true
  try {
    const result = await adminAPI.users.list(configuredUsersPagination.value.page, configuredUsersPagination.value.page_size, {
      has_group_rate_config: true,
      include_subscriptions: false,
      sort_by: 'email',
      sort_order: 'asc',
    })
    configuredUsers.value = result.items
    configuredUsersPagination.value.total = result.total
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loadingConfiguredUsers.value = false
  }
}

function handleConfiguredUsersPageChange(page: number): void {
  configuredUsersPagination.value.page = page
  void loadConfiguredUsers()
}

function handleConfiguredUsersPageSizeChange(pageSize: number): void {
  configuredUsersPagination.value.page_size = pageSize
  configuredUsersPagination.value.page = 1
  void loadConfiguredUsers()
}

async function searchUsers(): Promise<void> {
  if (!userSearch.value) {
    searchResults.value = []
    return
  }
  searchingUsers.value = true
  try {
    const result = await adminAPI.users.list(1, 20, {
      search: userSearch.value,
      include_subscriptions: false,
      sort_by: 'email',
      sort_order: 'asc',
    })
    searchResults.value = result.items
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    searchingUsers.value = false
  }
}

async function selectUser(user: AdminUser): Promise<void> {
  selectedUser.value = user
  searchResults.value = []
  userSearch.value = ''
  buildRows(user)
  await loadSelectedUser(user.id)
}

async function loadSelectedUser(userID: number): Promise<void> {
  loadingUser.value = true
  try {
    const user = await adminAPI.users.getById(userID)
    if (selectedUser.value?.id !== userID) return
    selectedUser.value = user
    buildRows(user)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loadingUser.value = false
  }
}

async function refresh(): Promise<void> {
  await Promise.all([loadGroups(), loadConfiguredUsers()])
  if (selectedUser.value) await loadSelectedUser(selectedUser.value.id)
}

function isValidMultiplier(value: number): boolean {
  return Number.isFinite(value) && value >= 0.0001 && value <= 100
}

async function save(): Promise<void> {
  if (!selectedUser.value) return
  if (groupRows.value.length === 0) {
    appStore.showError(t('admin.userPricing.emptyGroups'))
    return
  }

  const discounts: Record<number, number | null> = {}
  for (const row of groupRows.value) {
    // Fixed final rates have priority. Clearing a stale relative value here
    // keeps the stored configuration from carrying an ineffective duplicate.
    if (row.fixedRate !== null || row.coefficient === null) {
      discounts[row.group.id] = null
      continue
    }
    const coefficient = Number(row.coefficient)
    if (!isValidMultiplier(coefficient)) {
      appStore.showError(t('admin.userPricing.invalidCoefficient'))
      return
    }
    discounts[row.group.id] = coefficient
  }

  saving.value = true
  try {
    const user = await adminAPI.users.update(selectedUser.value.id, { group_discounts: discounts })
    selectedUser.value = user
    buildRows(user)
    await loadConfiguredUsers()
    showClearDialog.value = false
    appStore.showSuccess(t('admin.userPricing.saved'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function clearUserDiscounts(): Promise<void> {
  for (const row of groupRows.value) row.coefficient = null
  await save()
  showClearDialog.value = false
}

onMounted(async () => {
  await Promise.all([loadGroups(), loadConfiguredUsers()])
})
</script>
