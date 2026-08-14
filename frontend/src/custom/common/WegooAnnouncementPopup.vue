<template>
  <Teleport to="body">
    <Transition name="popup-fade">
      <div
        v-if="displayedAnnouncement"
        class="announcement-popup-shell fixed inset-0 z-[120] bg-black/45 backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
      >
        <div
          data-testid="announcement-popup-dialog"
          class="announcement-popup-card flex w-full max-w-[560px] flex-col overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm"
          @click.stop
        >
          <div class="relative shrink-0 overflow-hidden border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-5 py-6 sm:px-8 sm:py-7">

            <div class="relative z-10">
              <!-- Icon and badge -->
              <div class="mb-4 flex items-center gap-2">
                <div class="flex h-10 w-10 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-blue)]">
                  <Icon name="bell" size="md" />
                </div>
                <span class="announcement-popup-soft-badge inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-medium">
                  {{ t('announcements.unread') }}
                </span>
              </div>

              <!-- Title -->
              <h2 class="mb-2 max-w-[30rem] break-words text-lg font-semibold leading-tight text-[var(--apple-text)] sm:text-xl">
                {{ displayedAnnouncement.title }}
              </h2>

              <!-- Time -->
              <div class="flex items-center gap-1.5 text-sm text-[var(--apple-muted)]">
                <Icon name="clock" size="sm" />
                <time>{{ formatRelativeWithDateTime(displayedAnnouncement.created_at) }}</time>
              </div>
            </div>
          </div>

          <!-- Body -->
          <div class="announcement-popup-body announcement-scroll min-h-0 flex-1 overflow-y-auto bg-[var(--apple-surface)] px-5 py-6 sm:px-8 sm:py-8">
            <div class="max-w-[64ch]">
              <div
                class="announcement-popup-markdown announcement-markdown markdown-body prose prose-sm max-w-none dark:prose-invert"
                v-html="renderedContent"
              ></div>
            </div>
          </div>

          <!-- Footer -->
          <div class="shrink-0 border-t border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-4 py-4 sm:px-8 sm:py-5">
            <div class="flex items-center justify-center">
              <button
                data-testid="announcement-popup-dismiss"
                @click="handleDismiss"
                class="w-full rounded-lg bg-[var(--apple-blue)] px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--apple-blue-hover)] sm:w-auto"
              >
                <span class="flex items-center justify-center gap-2">
                  <Icon :name="preview ? 'x' : 'check'" size="sm" />
                  {{ preview ? t('common.close') : t('announcements.markRead') }}
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAnnouncementStore } from '@/custom/stores/announcements'
import { formatRelativeWithDateTime } from '@/utils/format'
import { releaseBodyModalLock, setBodyModalLock } from '@/custom/utils/modalLock'
import type { Announcement, UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import '@/styles/announcement-markdown.css'

defineOptions({ name: 'AnnouncementPopup' })

type PreviewAnnouncement = Pick<Announcement | UserAnnouncement, 'title' | 'content' | 'created_at'>

const props = withDefaults(defineProps<{
  announcement?: PreviewAnnouncement | null
  preview?: boolean
}>(), {
  announcement: null,
  preview: false,
})

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const modalLockToken = Symbol('announcement-popup')
const displayedAnnouncement = computed(() => (
  props.preview ? props.announcement : announcementStore.currentPopup
))

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = displayedAnnouncement.value?.content
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

function handleDismiss() {
  if (props.preview) {
    emit('close')
    return
  }
  announcementStore.dismissPopup()
}

watch(
  displayedAnnouncement,
  (announcement) => {
    setBodyModalLock(modalLockToken, Boolean(announcement))
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  releaseBodyModalLock(modalLockToken)
})
</script>

<style scoped>
.announcement-popup-shell {
  --announcement-popup-inset-top: max(1rem, env(safe-area-inset-top, 0px));
  --announcement-popup-inset-right: max(1rem, env(safe-area-inset-right, 0px));
  --announcement-popup-inset-bottom: max(1rem, env(safe-area-inset-bottom, 0px));
  --announcement-popup-inset-left: max(1rem, env(safe-area-inset-left, 0px));
  --announcement-popup-inline-margin: max(var(--announcement-popup-inset-left), var(--announcement-popup-inset-right));
  --announcement-popup-block-margin: max(var(--announcement-popup-inset-top), var(--announcement-popup-inset-bottom));
  --announcement-popup-scrollbar-offset: calc((100vw - 100%) / 2);

  height: 100vh;
  height: 100svh;
  height: 100dvh;
  overflow: hidden;
  box-sizing: border-box;
  overscroll-behavior: contain;
}

.announcement-popup-card {
  position: absolute;
  left: calc(50% + var(--announcement-popup-scrollbar-offset));
  top: max(8vh, var(--announcement-popup-inset-top));
  width: min(560px, calc(100% - var(--announcement-popup-inline-margin) - var(--announcement-popup-inline-margin)));
  max-height: calc(100% - var(--announcement-popup-block-margin) - var(--announcement-popup-block-margin));
  max-height: min(760px, calc(100dvh - max(8vh, var(--announcement-popup-inset-top)) - var(--announcement-popup-block-margin)));
  margin: 0;
  box-shadow: var(--apple-shadow-md);
  transform-origin: top center;
  transform: translate3d(-50%, 0, 0) scale(1);
}

.announcement-popup-markdown {
  color: var(--apple-text);
  text-align: left;
}

.announcement-popup-soft-badge {
  background: color-mix(in srgb, var(--apple-blue) 12%, var(--apple-surface));
  color: var(--apple-blue);
}

.announcement-popup-markdown :deep(:first-child) {
  margin-top: 0;
}

.announcement-popup-markdown :deep(:last-child) {
  margin-bottom: 0;
}

.announcement-popup-markdown :deep(:is(ul, ol)) {
  max-width: 100%;
}

.announcement-popup-body {
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

.popup-fade-enter-active {
  transition: opacity 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.popup-fade-leave-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.popup-fade-enter-from,
.popup-fade-leave-to {
  opacity: 0;
}

.popup-fade-enter-active .announcement-popup-card,
.popup-fade-leave-active .announcement-popup-card {
  transition: opacity 0.2s cubic-bezier(0.16, 1, 0.3, 1), transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.popup-fade-enter-from .announcement-popup-card {
  transform: translate3d(-50%, -10px, 0) scale(0.98);
  opacity: 0;
}

.popup-fade-leave-to .announcement-popup-card {
  transform: translate3d(-50%, -6px, 0) scale(0.99);
  opacity: 0;
}

.popup-fade-enter-to .announcement-popup-card,
.popup-fade-leave-from .announcement-popup-card {
  transform: translate3d(-50%, 0, 0) scale(1);
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .popup-fade-enter-active,
  .popup-fade-leave-active,
  .popup-fade-enter-active .announcement-popup-card,
  .popup-fade-leave-active .announcement-popup-card {
    transition-duration: 1ms;
  }

  .popup-fade-enter-from .announcement-popup-card,
  .popup-fade-leave-to .announcement-popup-card {
    transform: translate3d(-50%, 0, 0) scale(1);
  }
}

</style>
