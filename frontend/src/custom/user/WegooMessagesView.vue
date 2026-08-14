<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <UserPageHero
        :kicker="t('announcements.gateway.kicker')"
        :title="t('announcements.title')"
      >
        <template #actions>
          <div class="flex w-full flex-wrap gap-2 sm:w-auto sm:justify-end">
            <button
              v-if="unreadCount > 0"
              type="button"
              class="btn btn-secondary flex-1 justify-center sm:flex-none"
              :disabled="loading"
              @click="markAllRead"
            >
              <Icon name="checkCircle" size="sm" />
              {{ t('announcements.markAllRead') }}
            </button>
            <button
              type="button"
              class="btn btn-secondary flex-1 justify-center sm:flex-none"
              :disabled="loading"
              @click="refreshMessages"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
          </div>
        </template>

        <template #below>
          <div class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-[var(--apple-muted)]">
            <span>{{ t('announcements.messageCount', { count: announcements.length }) }}</span>
            <span class="inline-flex items-center gap-2">
              <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
              {{ t('announcements.unreadCount', { count: unreadCount }) }}
            </span>
          </div>
        </template>
      </UserPageHero>

      <section class="message-workspace overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm">
        <header class="border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-3 sm:p-4">
          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="inline-flex w-full rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-1 md:w-auto">
              <button
                v-for="option in filterOptions"
                :key="option.value"
                type="button"
                class="min-w-0 flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-colors md:flex-none"
                :class="filter === option.value ? 'bg-[var(--apple-blue)] text-white' : 'text-[var(--apple-muted)] hover:text-[var(--apple-text)]'"
                :aria-pressed="filter === option.value"
                @click="filter = option.value"
              >
                {{ option.label }}
              </button>
            </div>

            <label class="relative block w-full md:max-w-sm">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted-2)]" />
              <input
                v-model.trim="searchQuery"
                type="search"
                class="input pl-9"
                :placeholder="t('announcements.searchPlaceholder')"
              />
            </label>
          </div>
        </header>

        <div class="lg:grid lg:grid-cols-[minmax(300px,380px)_minmax(0,1fr)]">
          <div
            class="min-h-[420px] border-[color:var(--apple-border-soft)] lg:max-h-[720px] lg:min-h-[560px] lg:overflow-y-auto lg:border-r"
            :class="mobileDetailOpen ? 'hidden lg:block' : 'block'"
          >
            <div v-if="loading && announcements.length === 0" class="flex min-h-[420px] items-center justify-center">
              <div class="h-8 w-8 animate-spin rounded-full border-2 border-[color:var(--apple-border)] border-t-[var(--apple-blue)]"></div>
            </div>

            <div v-else-if="filteredMessages.length === 0" class="flex min-h-[420px] flex-col items-center justify-center px-6 text-center">
              <Icon name="inbox" size="xl" class="mb-3 text-[var(--apple-muted-2)]" />
              <p class="font-medium text-[var(--apple-text)]">
                {{ searchQuery ? t('announcements.emptySearch') : filter === 'unread' ? t('announcements.emptyUnread') : t('announcements.empty') }}
              </p>
              <p v-if="!searchQuery && filter === 'all'" class="mt-1 max-w-xs text-sm leading-6 text-[var(--apple-muted)]">
                {{ t('announcements.emptyDescription') }}
              </p>
            </div>

            <div v-else>
              <button
                v-for="message in filteredMessages"
                :key="message.id"
                type="button"
                class="message-row relative flex w-full items-start gap-3 border-b border-[color:var(--apple-border-soft)] px-4 py-4 text-left transition-colors hover:bg-[var(--apple-hover)]"
                :class="{ 'message-row-selected': selectedMessage?.id === message.id }"
                @click="selectMessage(message)"
              >
                <span
                  class="mt-1.5 h-2.5 w-2.5 flex-none rounded-full"
                  :class="message.read_at ? 'bg-[var(--apple-border)]' : 'bg-emerald-500 ring-4 ring-emerald-500/10'"
                ></span>
                <span class="min-w-0 flex-1">
                  <span class="flex items-start justify-between gap-3">
                    <span class="line-clamp-2 break-words text-sm font-semibold leading-5 text-[var(--apple-text)]">
                      {{ message.title }}
                    </span>
                    <time class="flex-none text-[11px] text-[var(--apple-muted-2)]">{{ formatMessageDate(message.created_at, true) }}</time>
                  </span>
                  <span class="mt-1 line-clamp-2 break-words text-xs leading-5 text-[var(--apple-muted)]">
                    {{ previewText(message.content) }}
                  </span>
                  <span class="mt-2 inline-flex items-center gap-1.5 text-[11px] font-medium text-[var(--apple-muted-2)]">
                    <Icon name="bell" size="xs" />
                    {{ t('announcements.systemSender') }}
                  </span>
                </span>
              </button>
            </div>
          </div>

          <article
            class="min-h-[420px] lg:max-h-[720px] lg:min-h-[560px] lg:overflow-y-auto"
            :class="mobileDetailOpen ? 'block' : 'hidden lg:block'"
          >
            <template v-if="selectedMessage">
              <div class="sticky top-0 z-10 flex items-center justify-between gap-3 border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] px-4 py-3 lg:hidden">
                <button type="button" class="btn btn-ghost px-2" @click="closeMobileDetail">
                  <Icon name="arrowLeft" size="sm" />
                  {{ t('common.back') }}
                </button>
                <span class="text-xs font-medium text-[var(--apple-muted)]">
                  {{ selectedMessage.read_at ? t('announcements.read') : t('announcements.unread') }}
                </span>
              </div>

              <div class="px-5 py-6 sm:px-8 sm:py-8 lg:px-10">
                <div class="flex flex-wrap items-center gap-2 text-xs font-medium text-[var(--apple-muted)]">
                  <span class="inline-flex items-center gap-1.5">
                    <Icon name="bell" size="xs" />
                    {{ t('announcements.systemSender') }}
                  </span>
                  <span aria-hidden="true">·</span>
                  <time>{{ formatMessageDate(selectedMessage.created_at) }}</time>
                  <span aria-hidden="true">·</span>
                  <span :class="selectedMessage.read_at ? 'text-[var(--apple-muted)]' : 'text-emerald-600'">
                    {{ selectedMessage.read_at ? t('announcements.read') : t('announcements.unread') }}
                  </span>
                </div>

                <h2 class="mt-4 break-words text-xl font-semibold leading-8 text-[var(--apple-text)] sm:text-2xl">
                  {{ selectedMessage.title }}
                </h2>

                <div
                  class="announcement-markdown markdown-body mt-6 max-w-none overflow-hidden"
                  v-html="renderMarkdown(selectedMessage.content)"
                ></div>

                <div v-if="selectedMessage.read_at" class="mt-8 border-t border-[color:var(--apple-border-soft)] pt-4 text-xs text-[var(--apple-muted-2)]">
                  {{ t('announcements.readAt') }}: {{ formatMessageDate(selectedMessage.read_at) }}
                </div>
              </div>
            </template>

            <div v-else class="flex min-h-[420px] flex-col items-center justify-center px-6 text-center lg:min-h-[560px]">
              <Icon name="inbox" size="xl" class="mb-3 text-[var(--apple-muted-2)]" />
              <p class="text-sm font-medium text-[var(--apple-muted)]">{{ t('announcements.selectMessage') }}</p>
            </div>
          </article>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

import AppLayout from '@/components/layout/AppLayout.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAppStore } from '@/stores/app'
import type { UserAnnouncement } from '@/types'
import '@/styles/announcement-markdown.css'

defineOptions({ name: 'WegooMessagesView' })

type MessageFilter = 'all' | 'unread'

const { t, locale } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()
const { announcements, loading } = storeToRefs(announcementStore)

const filter = ref<MessageFilter>('all')
const searchQuery = ref('')
const selectedMessageID = ref<number | null>(null)
const mobileDetailOpen = ref(false)
const unreadCount = computed(() => announcementStore.unreadCount)

const filterOptions = computed<Array<{ value: MessageFilter; label: string }>>(() => [
  { value: 'all', label: t('announcements.all') },
  { value: 'unread', label: t('announcements.unreadOnly') },
])

const filteredMessages = computed(() => {
  const query = searchQuery.value.toLocaleLowerCase().trim()
  return announcements.value.filter((message) => {
    if (filter.value === 'unread' && message.read_at) return false
    if (!query) return true
    return `${message.title}\n${message.content}`.toLocaleLowerCase().includes(query)
  })
})

const selectedMessage = computed(() =>
  announcements.value.find((message) => message.id === selectedMessageID.value) ?? null,
)

marked.setOptions({ breaks: true, gfm: true })

function renderMarkdown(content: string): string {
  return DOMPurify.sanitize(marked.parse(content || '') as string)
}

function previewText(content: string): string {
  return content
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[`*_>#~-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function formatMessageDate(value: string, compact = false): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const localeCode = String(locale.value).startsWith('zh') ? 'zh-CN' : 'en-US'
  return new Intl.DateTimeFormat(localeCode, compact
    ? { month: 'short', day: 'numeric' }
    : { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }
  ).format(date)
}

async function selectMessage(message: UserAnnouncement) {
  selectedMessageID.value = message.id
  mobileDetailOpen.value = true
  if (!message.read_at) {
    await announcementStore.markAsRead(message.id)
  }
}

function closeMobileDetail() {
  mobileDetailOpen.value = false
}

async function refreshMessages() {
  const loaded = await announcementStore.fetchAnnouncements(true)
  if (!loaded) {
    appStore.showError(t('announcements.loadFailed'))
  }
}

async function markAllRead() {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (error: any) {
    appStore.showError(error?.message || t('common.unknownError'))
  }
}

watch(
  filteredMessages,
  (messages) => {
    if (messages.length === 0) {
      selectedMessageID.value = null
      mobileDetailOpen.value = false
      return
    }
    if (!messages.some((message) => message.id === selectedMessageID.value)) {
      selectedMessageID.value = messages[0].id
    }
  },
  { immediate: true },
)

onMounted(() => {
  refreshMessages()
})
</script>

<style scoped>
.message-row-selected {
  background: color-mix(in srgb, var(--apple-blue) 8%, var(--apple-surface));
}

.announcement-markdown :deep(:first-child) {
  margin-top: 0;
}

.announcement-markdown :deep(:last-child) {
  margin-bottom: 0;
}

.announcement-markdown :deep(:is(pre, table)) {
  max-width: 100%;
  overflow-x: auto;
}

.announcement-markdown :deep(img) {
  height: auto;
  max-width: 100%;
}
</style>
