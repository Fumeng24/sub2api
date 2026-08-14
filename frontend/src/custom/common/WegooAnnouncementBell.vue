<template>
  <div>
    <!-- 铃铛按钮 -->
    <button
      data-testid="announcement-bell-open"
      @click="openModal"
      class="relative flex h-9 w-9 items-center justify-center rounded-lg text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
      :class="{ 'text-[var(--apple-blue)]': unreadCount > 0 }"
      :aria-label="t('announcements.title')"
    >
      <Icon name="bell" size="md" />
      <!-- 未读红点 -->
      <span
        v-if="unreadCount > 0"
        class="absolute right-1 top-1 h-2 w-2 rounded-full bg-[var(--apple-blue)] ring-2 ring-[var(--apple-surface)]"
      >
      </span>
    </button>

    <!-- 公告列表 Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="isModalOpen"
          class="announcement-modal-shell fixed inset-0 z-[100] bg-black/45 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          @click="closeModal"
        >
          <div
            data-testid="announcement-list-dialog"
            class="announcement-modal-card flex w-full max-w-[620px] flex-col overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm"
            @click.stop
          >
            <div class="relative shrink-0 overflow-hidden border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-4 py-5 sm:px-6">
              <div class="relative z-10 flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-blue)]">
                      <Icon name="bell" size="sm" />
                    </div>
                    <h2 class="min-w-0 break-words text-lg font-semibold leading-tight text-[var(--apple-text)]">
                      {{ t('announcements.title') }}
                    </h2>
                  </div>
                  <p v-if="unreadCount > 0" class="mt-2 text-sm text-[var(--apple-muted)]">
                    <span class="font-medium text-[var(--apple-blue)]">{{ unreadCount }}</span>
                    {{ t('announcements.unread') }}
                  </p>
                </div>
                <div class="flex flex-shrink-0 items-center gap-2">
                  <button
                    v-if="unreadCount > 0"
                    @click="markAllAsRead"
                    :disabled="loading"
                    class="inline-flex h-9 max-w-[8.5rem] items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 text-xs font-medium text-[var(--apple-text)] transition-colors hover:bg-[var(--apple-hover)] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <span class="truncate">{{ t('announcements.markAllRead') }}</span>
                  </button>
                  <button
                    @click="closeModal"
                    class="flex h-9 w-9 items-center justify-center rounded-lg text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
                    :aria-label="t('common.close')"
                  >
                    <Icon name="x" size="sm" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Body -->
            <div class="min-h-0 flex-1 overflow-y-auto">
              <!-- Loading -->
              <div v-if="loading" class="flex items-center justify-center py-16">
                <div class="h-8 w-8 animate-spin rounded-full border-2 border-[color:var(--apple-border)] border-t-[var(--apple-blue)]"></div>
              </div>

              <!-- Announcements List -->
              <div v-else-if="recentAnnouncements.length > 0">
                <div
                  v-for="item in recentAnnouncements"
                  :key="item.id"
                  :data-testid="`announcement-row-${item.id}`"
                  class="group relative flex cursor-pointer items-start gap-3 border-b border-[color:var(--apple-border-soft)] px-4 py-4 text-left transition-colors hover:bg-[var(--apple-surface-elevated)] sm:gap-4 sm:px-6"
                  :class="{ 'announcement-row-unread': !item.read_at }"
                  style="min-height: 72px"
                  role="button"
                  tabindex="0"
                  @click="openDetail(item)"
                  @keydown.enter.prevent="openDetail(item)"
                  @keydown.space.prevent="openDetail(item)"
                >
                  <!-- Status Indicator -->
                  <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center">
                    <div
                      v-if="!item.read_at"
                      class="relative flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--apple-blue)] text-white"
                    >
                      <Icon name="infoCircle" size="md" :stroke-width="2" />
                    </div>
                    <div
                      v-else
                      class="flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--apple-surface-elevated)] text-[var(--apple-muted-2)]"
                    >
                      <Icon name="checkCircle" size="md" />
                    </div>
                  </div>

                  <!-- Content -->
                  <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
                    <div class="min-w-0 flex-1">
                      <h3 class="line-clamp-2 break-words text-sm font-medium leading-5 text-[var(--apple-text)]">
                        {{ item.title }}
                      </h3>
                      <div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
                        <time class="text-xs text-[var(--apple-muted)]">
                          {{ formatRelativeTime(item.created_at) }}
                        </time>
                        <span
                          v-if="!item.read_at"
                          class="announcement-soft-badge inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-medium"
                        >
                          {{ t('announcements.unread') }}
                        </span>
                      </div>
                    </div>

                    <!-- Arrow -->
                    <div class="mt-1 flex-shrink-0">
                      <Icon name="chevronRight" size="md" class="text-[var(--apple-muted-2)] transition-transform group-hover:translate-x-0.5" />
                    </div>
                  </div>

                  <!-- Unread indicator bar -->
                  <div
                    v-if="!item.read_at"
                    class="absolute left-0 top-0 h-full w-1 bg-[var(--apple-blue)]"
                  ></div>
                </div>
              </div>

              <!-- Empty State -->
              <div v-else class="flex flex-col items-center justify-center py-16">
                <div class="mb-4">
                  <div class="flex h-20 w-20 items-center justify-center rounded-lg bg-[var(--apple-surface-elevated)]">
                    <Icon name="inbox" size="xl" class="text-[var(--apple-muted-2)]" />
                  </div>
                </div>
                <p class="text-sm font-medium text-[var(--apple-text)]">{{ t('announcements.empty') }}</p>
                <p class="mt-1 text-xs text-[var(--apple-muted)]">{{ t('announcements.emptyDescription') }}</p>
              </div>
            </div>

            <div class="shrink-0 border-t border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-3 sm:px-6">
              <button
                type="button"
                class="btn btn-secondary w-full justify-center"
                @click="openMessageCenter"
              >
                <Icon name="inbox" size="sm" />
                {{ t('announcements.viewAll') }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 公告详情 Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="detailModalOpen && selectedAnnouncement"
          class="announcement-modal-shell fixed inset-0 z-[110] bg-black/45 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          @click="closeDetail"
        >
          <div
            data-testid="announcement-detail-dialog"
            class="announcement-modal-card flex w-full max-w-[780px] flex-col overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm"
            @click.stop
          >
            <div class="relative shrink-0 overflow-hidden border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-5 py-6 sm:px-8 sm:py-7">
              <button
                @click="closeDetail"
                class="absolute right-4 top-4 z-20 flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
                :aria-label="t('common.close')"
              >
                <Icon name="x" size="md" />
              </button>

              <div class="relative z-10 max-w-[38rem] pr-12">
                <!-- Icon and Category -->
                <div class="mb-4 flex items-center gap-2">
                  <div class="flex h-10 w-10 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-blue)]">
                    <Icon name="bell" size="md" />
                  </div>
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <span class="announcement-soft-badge rounded-lg px-2.5 py-1 text-xs font-medium">
                      {{ t('announcements.title') }}
                    </span>
                    <span
                      v-if="!selectedAnnouncement.read_at"
                      class="announcement-soft-badge inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-medium"
                    >
                      {{ t('announcements.unread') }}
                    </span>
                  </div>
                </div>

                <!-- Title -->
                <h2 class="mb-3 max-w-[36rem] break-words text-lg font-semibold leading-tight text-[var(--apple-text)] sm:text-2xl">
                  {{ selectedAnnouncement.title }}
                </h2>

                <!-- Meta Info -->
                <div class="flex flex-col gap-2 text-sm text-[var(--apple-muted)] sm:flex-row sm:flex-wrap sm:gap-4">
                  <div class="flex items-center gap-1.5">
                    <Icon name="clock" size="sm" />
                    <time>{{ formatRelativeWithDateTime(selectedAnnouncement.created_at) }}</time>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <Icon :name="selectedAnnouncement.read_at ? 'checkCircle' : 'infoCircle'" size="sm" />
                    <span>{{ selectedAnnouncement.read_at ? t('announcements.read') : t('announcements.unread') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Body with Enhanced Markdown -->
            <div class="announcement-detail-body announcement-scroll min-h-0 flex-1 overflow-y-auto bg-[var(--apple-surface)] px-5 py-6 sm:px-8 sm:py-8">
              <div class="max-w-[68ch]">
                <div
                  class="announcement-detail-markdown announcement-markdown markdown-body prose prose-sm max-w-none dark:prose-invert"
                  v-html="renderMarkdown(selectedAnnouncement.content)"
                ></div>
              </div>
            </div>

            <!-- Footer with Actions -->
            <div class="shrink-0 border-t border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-4 py-4 sm:px-8 sm:py-5">
              <div class="flex flex-col items-stretch gap-4 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex min-w-0 items-center gap-2 text-xs text-[var(--apple-muted)]">
                  <Icon name="infoCircle" size="sm" class="flex-shrink-0" />
                  <span class="min-w-0 break-words">{{ selectedAnnouncement.read_at ? t('announcements.readStatus') : t('announcements.markReadHint') }}</span>
                </div>
                <div class="flex w-full items-center gap-3 sm:w-auto">
                  <button
                    @click="closeDetail"
                    class="flex-1 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-5 py-2.5 text-sm font-medium text-[var(--apple-text)] transition-colors hover:bg-[var(--apple-surface-elevated)] sm:flex-none"
                  >
                    {{ t('common.close') }}
                  </button>
                  <button
                    v-if="!selectedAnnouncement.read_at"
                    @click="markAsReadAndClose(selectedAnnouncement.id)"
                    class="flex-1 rounded-lg bg-[var(--apple-blue)] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--apple-blue-hover)] sm:flex-none"
                  >
                    <span class="flex items-center justify-center gap-2">
                      <Icon name="check" size="sm" />
                      {{ t('announcements.markRead') }}
                    </span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/custom/stores/announcements'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import { releaseBodyModalLock, setBodyModalLock } from '@/custom/utils/modalLock'
import type { UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import '@/styles/announcement-markdown.css'

defineOptions({ name: 'AnnouncementBell' })

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()
const modalLockToken = Symbol('announcement-bell')

// Configure marked
marked.setOptions({
  breaks: true,
  gfm: true,
})

// Use store state (storeToRefs for reactivity)
const { announcements, loading } = storeToRefs(announcementStore)
const unreadCount = computed(() => announcementStore.unreadCount)
const recentAnnouncements = computed(() => announcements.value.slice(0, 20))

// Local modal state
const isModalOpen = ref(false)
const detailModalOpen = ref(false)
const selectedAnnouncement = ref<UserAnnouncement | null>(null)

// Methods
function renderMarkdown(content: string): string {
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
}

function openModal() {
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
}

function openMessageCenter() {
  closeModal()
  router.push('/messages')
}

function openDetail(announcement: UserAnnouncement) {
  selectedAnnouncement.value = announcement
  detailModalOpen.value = true
  if (!announcement.read_at) {
    markAsRead(announcement.id)
  }
}

function closeDetail() {
  detailModalOpen.value = false
  selectedAnnouncement.value = null
}

async function markAsRead(id: number) {
  try {
    await announcementStore.markAsRead(id)
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

async function markAsReadAndClose(id: number) {
  await markAsRead(id)
  appStore.showSuccess(t('announcements.markedAsRead'))
  closeDetail()
}

async function markAllAsRead() {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (detailModalOpen.value) {
      closeDetail()
    } else if (isModalOpen.value) {
      closeModal()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
  releaseBodyModalLock(modalLockToken)
})

watch(
  [isModalOpen, detailModalOpen],
  ([modal, detail]) => {
    setBodyModalLock(modalLockToken, modal || detail)
  },
  { immediate: true }
)
</script>

<style scoped>
.announcement-modal-shell {
  --announcement-modal-inset-top: max(1rem, env(safe-area-inset-top, 0px));
  --announcement-modal-inset-right: max(1rem, env(safe-area-inset-right, 0px));
  --announcement-modal-inset-bottom: max(1rem, env(safe-area-inset-bottom, 0px));
  --announcement-modal-inset-left: max(1rem, env(safe-area-inset-left, 0px));
  --announcement-modal-inline-margin: max(var(--announcement-modal-inset-left), var(--announcement-modal-inset-right));
  --announcement-modal-block-margin: max(var(--announcement-modal-inset-top), var(--announcement-modal-inset-bottom));
  --announcement-modal-scrollbar-offset: calc((100vw - 100%) / 2);

  height: 100vh;
  height: 100svh;
  height: 100dvh;
  overflow: hidden;
  box-sizing: border-box;
  overscroll-behavior: contain;
}

.announcement-modal-card {
  position: absolute;
  left: calc(50% + var(--announcement-modal-scrollbar-offset));
  top: max(6vh, var(--announcement-modal-inset-top));
  width: min(var(--announcement-modal-width, 620px), calc(100% - var(--announcement-modal-inline-margin) - var(--announcement-modal-inline-margin)));
  max-height: calc(100% - var(--announcement-modal-block-margin) - var(--announcement-modal-block-margin));
  max-height: min(820px, calc(100dvh - max(6vh, var(--announcement-modal-inset-top)) - var(--announcement-modal-block-margin)));
  margin: 0;
  box-shadow: var(--apple-shadow-md);
  transform-origin: top center;
  transform: translate3d(-50%, 0, 0) scale(1);
}

.announcement-modal-card.max-w-\[780px\] {
  --announcement-modal-width: 780px;
}

.announcement-row-unread {
  background: color-mix(in srgb, var(--apple-blue) 6%, var(--apple-surface));
}

.announcement-soft-badge {
  background: color-mix(in srgb, var(--apple-blue) 12%, var(--apple-surface));
  color: var(--apple-blue);
}

.announcement-detail-markdown {
  color: var(--apple-text);
  text-align: left;
}

.announcement-detail-markdown :deep(:first-child) {
  margin-top: 0;
}

.announcement-detail-markdown :deep(:last-child) {
  margin-bottom: 0;
}

.announcement-detail-markdown :deep(:is(ul, ol)) {
  max-width: 100%;
}

.announcement-detail-body {
  scrollbar-gutter: stable both-edges;
}

.announcement-scroll::-webkit-scrollbar {
  width: 6px;
}

.announcement-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.announcement-scroll::-webkit-scrollbar-thumb {
  background: var(--apple-border);
  border-radius: 999px;
}

.announcement-markdown {
  overflow-wrap: anywhere;
}

.announcement-markdown :deep(pre),
.announcement-markdown :deep(table) {
  max-width: 100%;
  overflow-x: auto;
}

.announcement-markdown :deep(table) {
  display: block;
  border-collapse: collapse;
}

.announcement-markdown :deep(img) {
  height: auto;
  max-width: 100%;
}

/* Modal Animations */
.modal-fade-enter-active {
  transition: opacity 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-fade-leave-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-from .announcement-modal-card {
  transform: translate3d(-50%, -10px, 0) scale(0.98);
  opacity: 0;
}

.modal-fade-leave-to .announcement-modal-card {
  transform: translate3d(-50%, -6px, 0) scale(0.99);
  opacity: 0;
}

.modal-fade-enter-active .announcement-modal-card,
.modal-fade-leave-active .announcement-modal-card {
  transition: opacity 0.2s cubic-bezier(0.16, 1, 0.3, 1), transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-fade-enter-to .announcement-modal-card,
.modal-fade-leave-from .announcement-modal-card {
  transform: translate3d(-50%, 0, 0) scale(1);
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .modal-fade-enter-active,
  .modal-fade-leave-active,
  .modal-fade-enter-active .announcement-modal-card,
  .modal-fade-leave-active .announcement-modal-card {
    transition-duration: 1ms;
  }

  .modal-fade-enter-from .announcement-modal-card,
  .modal-fade-leave-to .announcement-modal-card {
    transform: translate3d(-50%, 0, 0) scale(1);
  }
}

</style>

<style>
.markdown-body {
  color: var(--apple-muted);
  font-size: 15px;
  line-height: 1.75;
}

.markdown-body h1 {
  margin: 2rem 0 1.5rem;
  border-bottom: 1px solid var(--apple-border-soft);
  padding-bottom: 0.75rem;
  color: var(--apple-text);
  font-size: 1.5rem;
  font-weight: 650;
  line-height: 1.2;
}

.markdown-body h2 {
  margin: 1.75rem 0 1rem;
  border-bottom: 1px solid var(--apple-border-soft);
  padding-bottom: 0.5rem;
  color: var(--apple-text);
  font-size: 1.25rem;
  font-weight: 650;
  line-height: 1.25;
}

.markdown-body h3 {
  margin: 1.5rem 0 0.75rem;
  color: var(--apple-text);
  font-size: 1.125rem;
  font-weight: 650;
  line-height: 1.3;
}

.markdown-body h4 {
  margin: 1.25rem 0 0.5rem;
  color: var(--apple-text);
  font-size: 1rem;
  font-weight: 650;
  line-height: 1.35;
}

.markdown-body p {
  margin-bottom: 1rem;
}

.markdown-body a {
  color: var(--apple-blue);
  font-weight: 550;
  text-decoration-line: underline;
  text-decoration-color: color-mix(in srgb, var(--apple-blue) 32%, transparent);
  text-decoration-thickness: 1.5px;
  text-underline-offset: 3px;
  transition: text-decoration-color 150ms ease;
}

.markdown-body a:hover {
  text-decoration-color: var(--apple-blue);
}

.markdown-body ul,
.markdown-body ol {
  margin: 0 0 1rem 1.5rem;
}

.markdown-body ul {
  list-style-type: disc;
}

.markdown-body ol {
  list-style-type: decimal;
}

.markdown-body li {
  padding-left: 0.5rem;
  line-height: 1.7;
}

.markdown-body li + li {
  margin-top: 0.5rem;
}

.markdown-body li::marker {
  color: var(--apple-blue);
}

.markdown-body blockquote {
  position: relative;
  margin: 1.25rem 0;
  border-left: 3px solid var(--apple-blue);
  background: color-mix(in srgb, var(--apple-blue) 7%, var(--apple-surface));
  padding: 0.75rem 1rem 0.75rem 1.25rem;
  color: var(--apple-muted);
  font-style: italic;
}

.markdown-body blockquote::before {
  content: '"';
  position: absolute;
  left: -0.25rem;
  top: -0.15rem;
  color: color-mix(in srgb, var(--apple-blue) 22%, transparent);
  font-family: ui-serif, Georgia, Cambria, "Times New Roman", Times, serif;
  font-size: 3rem;
  line-height: 1;
}

.markdown-body code {
  border-radius: 6px;
  background: var(--apple-surface-elevated);
  padding: 0.125rem 0.375rem;
  color: var(--apple-text);
  font-family: ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, monospace;
  font-size: 13px;
}

.markdown-body pre {
  margin: 1.25rem 0;
  overflow-x: auto;
  border: 1px solid var(--apple-border);
  border-radius: 8px;
  background: var(--apple-surface-elevated);
  padding: 1rem;
}

.markdown-body pre code {
  background: transparent;
  padding: 0;
  color: var(--apple-text);
  font-size: 13px;
}

.markdown-body hr {
  margin: 2rem 0;
  border: 0;
  border-top: 1px solid var(--apple-border-soft);
}

.markdown-body table {
  margin-bottom: 1.25rem;
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--apple-border);
  border-radius: 8px;
}

.markdown-body th,
.markdown-body td {
  border-right: 1px solid var(--apple-border-soft);
  border-bottom: 1px solid var(--apple-border-soft);
  padding: 0.75rem 1rem;
  text-align: left;
}

.markdown-body th:last-child,
.markdown-body td:last-child {
  border-right: 0;
}

.markdown-body tr:last-child td {
  border-bottom: 0;
}

.markdown-body th {
  background: var(--apple-surface-elevated);
  color: var(--apple-text);
  font-weight: 650;
}

.markdown-body tbody tr {
  transition: background-color 150ms ease;
}

.markdown-body tbody tr:hover {
  background: var(--apple-hover);
}

.markdown-body img {
  margin: 1.25rem 0;
  max-width: 100%;
  border-radius: 8px;
  border: 1px solid var(--apple-border);
  border-color: var(--apple-border);
}

.markdown-body strong {
  color: var(--apple-text);
  font-weight: 650;
}

.markdown-body em {
  color: var(--apple-muted);
  font-style: italic;
}
</style>
