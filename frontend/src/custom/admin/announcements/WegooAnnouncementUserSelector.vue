<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-3">
      <span class="text-xs text-gray-500 dark:text-dark-400">
        {{ t('admin.announcements.form.selectedUsers', { count: modelValue.length, max: MAX_SELECTED_USERS }) }}
      </span>
      <span
        v-if="modelValue.length >= MAX_SELECTED_USERS"
        class="text-xs font-medium text-amber-600 dark:text-amber-400"
      >
        {{ t('admin.announcements.form.userSelectionLimit', { max: MAX_SELECTED_USERS }) }}
      </span>
    </div>

    <div
      v-if="selectedUsers.length > 0"
      class="flex max-h-32 flex-wrap gap-2 overflow-y-auto rounded-lg border border-gray-200 bg-white p-2 dark:border-dark-600 dark:bg-dark-800"
    >
      <span
        v-for="user in selectedUsers"
        :key="user.id"
        class="inline-flex max-w-full items-center gap-1.5 rounded-md border border-gray-200 bg-gray-50 px-2 py-1 text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200"
      >
        <span class="max-w-56 truncate" :title="user.email || user.username || `#${user.id}`">
          {{ user.email || user.username || `#${user.id}` }}
        </span>
        <span v-if="user.email || user.username" class="flex-none text-gray-400">#{{ user.id }}</span>
        <button
          type="button"
          class="flex h-5 w-5 flex-none items-center justify-center rounded text-gray-400 hover:bg-gray-200 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-white"
          :title="t('admin.announcements.form.removeUser')"
          :aria-label="t('admin.announcements.form.removeUser')"
          @click="removeUser(user.id)"
        >
          <Icon name="x" size="xs" />
        </button>
      </span>
    </div>

    <div class="relative">
      <Icon
        name="search"
        size="sm"
        class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
      />
      <input
        v-model="searchQuery"
        type="search"
        class="input pl-9 pr-9"
        :placeholder="t('admin.announcements.form.userSearchPlaceholder')"
        autocomplete="off"
      />
      <span
        v-if="searching"
        class="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin rounded-full border-2 border-gray-300 border-t-primary-500"
      ></span>
    </div>

    <div
      v-if="trimmedSearchQuery"
      class="max-h-52 overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800"
    >
      <button
        v-for="user in searchResults"
        :key="user.id"
        type="button"
        class="flex w-full items-center justify-between gap-3 border-b border-gray-100 px-3 py-2 text-left last:border-b-0 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-55 dark:border-dark-700 dark:hover:bg-dark-700"
        :disabled="isSelected(user.id) || modelValue.length >= MAX_SELECTED_USERS"
        @click="addUser(user)"
      >
        <span class="min-w-0">
          <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">
            {{ user.email || user.username || `#${user.id}` }}
          </span>
          <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-dark-400">
            {{ user.username || '-' }} · #{{ user.id }}
          </span>
        </span>
        <Icon
          :name="isSelected(user.id) ? 'check' : 'plus'"
          size="sm"
          class="flex-none text-gray-400"
        />
      </button>

      <div
        v-if="!searching && searchResults.length === 0"
        class="px-3 py-4 text-center text-sm text-gray-500 dark:text-dark-400"
      >
        {{ searchError || t('admin.announcements.form.userSearchEmpty') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { adminAPI } from '@/custom/api/admin'
import Icon from '@/components/icons/Icon.vue'
import type { AdminUser } from '@/types'

defineOptions({ name: 'WegooAnnouncementUserSelector' })

const MAX_SELECTED_USERS = 100

type UserSummary = Pick<AdminUser, 'id' | 'email' | 'username'>

const props = defineProps<{
  modelValue: number[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const { t } = useI18n()
const searchQuery = ref('')
const searchResults = ref<UserSummary[]>([])
const searching = ref(false)
const searchError = ref('')
const knownUsers = ref<Record<number, UserSummary>>({})

let searchTimer: ReturnType<typeof setTimeout> | null = null
let searchSequence = 0
let resolveSequence = 0

const trimmedSearchQuery = computed(() => searchQuery.value.trim())
const selectedUsers = computed<UserSummary[]>(() =>
  props.modelValue.map((id) => knownUsers.value[id] ?? { id, email: '', username: '' }),
)

function rememberUsers(users: UserSummary[]) {
  if (users.length === 0) return
  const next = { ...knownUsers.value }
  for (const user of users) {
    next[user.id] = user
  }
  knownUsers.value = next
}

function isSelected(userID: number) {
  return props.modelValue.includes(userID)
}

function addUser(user: UserSummary) {
  if (isSelected(user.id) || props.modelValue.length >= MAX_SELECTED_USERS) return
  rememberUsers([user])
  emit('update:modelValue', [...props.modelValue, user.id])
  searchQuery.value = ''
  searchResults.value = []
}

function removeUser(userID: number) {
  emit('update:modelValue', props.modelValue.filter((id) => id !== userID))
}

async function searchUsers(query: string) {
  const sequence = ++searchSequence
  searching.value = true
  searchError.value = ''

  try {
    const response = await adminAPI.users.list(1, 20, {
      search: query,
      sort_by: 'id',
      sort_order: 'asc',
    })
    if (sequence !== searchSequence) return
    const users = response.items.map(({ id, email, username }) => ({ id, email, username }))
    rememberUsers(users)
    searchResults.value = users
  } catch (error) {
    if (sequence !== searchSequence) return
    console.error('Failed to search announcement recipients:', error)
    searchResults.value = []
    searchError.value = t('admin.announcements.form.userSearchFailed')
  } finally {
    if (sequence === searchSequence) {
      searching.value = false
    }
  }
}

async function resolveSelectedUsers(ids: number[]) {
  const missing = [...new Set(ids)].filter((id) => id > 0 && !knownUsers.value[id])
  if (missing.length === 0) return

  const sequence = ++resolveSequence
  let cursor = 0
  const resolved: UserSummary[] = []
  const workerCount = Math.min(4, missing.length)

  await Promise.all(Array.from({ length: workerCount }, async () => {
    while (cursor < missing.length && sequence === resolveSequence) {
      const userID = missing[cursor]
      cursor += 1
      try {
        const { id, email, username } = await adminAPI.users.getById(userID)
        resolved.push({ id, email, username })
      } catch {
        // Deleted or inaccessible users remain visible by immutable ID.
      }
    }
  }))

  if (sequence === resolveSequence) {
    rememberUsers(resolved)
  }
}

watch(trimmedSearchQuery, (query) => {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
  searchSequence += 1
  searchError.value = ''

  if (!query) {
    searching.value = false
    searchResults.value = []
    return
  }

  searching.value = true
  searchTimer = setTimeout(() => {
    searchTimer = null
    void searchUsers(query)
  }, 250)
})

watch(
  () => props.modelValue.slice(),
  (ids) => {
    void resolveSelectedUsers(ids)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
  searchSequence += 1
  resolveSequence += 1
})
</script>
