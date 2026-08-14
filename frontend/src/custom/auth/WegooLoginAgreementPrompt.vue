<template>
  <div
    v-if="mode === 'checkbox' && documents.length > 0"
    class="px-0.5"
  >
    <div class="flex items-start gap-2">
      <input
        id="login-agreement-consent"
        data-testid="login-agreement-checkbox"
        type="checkbox"
        :checked="accepted"
        class="mt-[2px] h-4 w-4 flex-shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-900"
        @change="handleCheckboxChange"
      />
      <div class="min-w-0 flex-1">
        <p class="text-[13px] leading-5 text-gray-600 dark:text-dark-300">
          <label
            for="login-agreement-consent"
            class="cursor-pointer text-gray-700 dark:text-dark-200"
          >
            {{ t('auth.agreementPrompt.checkboxLabel') }}
          </label>
          <template v-for="(doc, index) in documents" :key="doc.id || doc.title">
            <RouterLink
              :to="documentRoute(doc)"
              target="_blank"
              rel="noopener noreferrer"
              class="font-medium text-primary-600 underline-offset-4 transition hover:text-primary-700 hover:underline dark:text-primary-300 dark:hover:text-primary-200"
            >
              {{ doc.title }}
            </RouterLink>
            <span v-if="index < documents.length - 1">{{ t('auth.agreementPrompt.documentSeparator') }}</span>
          </template>
        </p>
      </div>
    </div>
  </div>

  <div
    v-else-if="!accepted && documents.length > 0"
    class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300"
  >
    <div class="flex items-start gap-3">
      <Icon name="shield" size="sm" class="mt-0.5 flex-shrink-0 text-primary-600 dark:text-primary-300" />
      <div class="min-w-0 flex-1">
        <p class="font-medium">{{ t('auth.agreementPrompt.bannerTitle') }}</p>
        <p class="mt-1 text-gray-500 dark:text-gray-400">
          {{ t('auth.agreementPrompt.bannerDescription') }}
        </p>
      </div>
      <button
        type="button"
        data-testid="login-agreement-open"
        class="btn btn-secondary h-8 flex-shrink-0 px-3 text-xs"
        @click="emit('open')"
      >
        {{ t('auth.agreementPrompt.openButton') }}
      </button>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="agreement-fade">
      <div
        v-if="dialogVisible"
        data-testid="login-agreement-modal"
        class="login-agreement-shell dark fixed inset-0 z-[140] bg-black/[0.74] backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
      >
        <div class="login-agreement-card flex w-full max-w-[600px] flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#0d1219] shadow-[0_24px_80px_rgb(0_0_0_/_55%)]">
          <div class="flex-shrink-0 border-b border-gray-100 bg-white px-6 py-6 dark:border-white/10 dark:bg-[#161617] max-sm:px-5 max-sm:py-4">
            <div class="flex items-start gap-4">
              <span class="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] text-[var(--apple-blue)] max-sm:h-10 max-sm:w-10">
                <Icon name="shield" size="md" />
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="text-xl font-semibold tracking-normal text-gray-950 dark:text-white max-sm:text-lg">
                    {{ t('auth.agreementPrompt.modalTitle') }}
                  </h2>
                  <span
                    v-if="updatedAt"
                    class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-white/[0.08] dark:text-dark-200"
                  >
                    {{ updatedAt }}
                  </span>
                </div>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300 max-sm:leading-5">
                  {{ t('auth.agreementPrompt.modalDescription', { date: updatedAtLabel }) }}
                </p>
              </div>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-6 py-5 max-sm:px-5 max-sm:py-4">
            <div class="mb-5 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4">
              <p class="mb-3 text-sm font-semibold text-[var(--apple-text)]">
                {{ t('auth.agreementPrompt.guaranteesLabel') }}
              </p>
              <div class="grid gap-2">
                <div
                  v-for="item in guaranteeItems"
                  :key="item"
                  class="flex min-w-0 items-start gap-2 text-sm leading-6 text-[var(--apple-muted)]"
                >
                  <Icon name="checkCircle" size="sm" class="mt-1 flex-shrink-0 text-[var(--apple-blue)]" />
                  <span class="min-w-0 break-words">{{ item }}</span>
                </div>
              </div>
            </div>
            <div class="mb-3 flex items-center justify-between gap-3">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('auth.agreementPrompt.documentsLabel') }}
              </p>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <RouterLink
                v-for="(doc, index) in documents"
                :key="doc.id || doc.title"
                :to="documentRoute(doc)"
                target="_blank"
                rel="noopener noreferrer"
                class="group flex min-h-[72px] w-full items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-left transition-colors hover:border-primary-200 hover:bg-white dark:border-white/10 dark:bg-white/[0.035] dark:hover:border-primary-400/30 dark:hover:bg-white/[0.06] max-sm:min-h-[64px]"
              >
                <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-white text-gray-700 ring-1 ring-gray-200 transition group-hover:bg-primary-50 group-hover:text-primary-700 group-hover:ring-primary-100 dark:bg-black/20 dark:text-dark-200 dark:ring-white/10 dark:group-hover:bg-primary-400/10 dark:group-hover:text-primary-200 dark:group-hover:ring-primary-400/20">
                  <Icon :name="documentIcon(index, doc.title)" size="sm" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-semibold text-gray-950 dark:text-white">{{ doc.title }}</span>
                </span>
                <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full text-gray-400 transition group-hover:bg-primary-50 group-hover:text-primary-600 dark:group-hover:bg-primary-500/10 dark:group-hover:text-primary-300">
                  <Icon name="externalLink" size="sm" />
                </span>
              </RouterLink>
            </div>
          </div>

          <div class="flex-shrink-0 border-t border-gray-100 bg-gray-50/80 px-6 py-4 dark:border-white/10 dark:bg-black/[0.18] max-sm:px-4 max-sm:pb-[calc(0.75rem_+_env(safe-area-inset-bottom))] max-sm:pt-3">
            <div class="grid grid-cols-2 gap-3">
              <button
                type="button"
                data-testid="login-agreement-reject"
                class="btn btn-secondary min-h-11 min-w-0 whitespace-nowrap px-3 text-sm"
                @click="emit('reject')"
              >
                {{ t('auth.agreementPrompt.reject') }}
              </button>
              <button
                type="button"
                data-testid="login-agreement-accept"
                class="btn btn-primary min-h-11 min-w-0 whitespace-nowrap px-3 text-sm"
                @click="emit('accept')"
              >
                {{ t('auth.agreementPrompt.accept') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { LoginAgreementDocument } from '@/types'

const props = withDefaults(defineProps<{
  accepted: boolean
  documents: LoginAgreementDocument[]
  mode: 'modal' | 'checkbox' | string
  updatedAt?: string
  visible: boolean
}>(), {
  updatedAt: ''
})

const { t } = useI18n()

const emit = defineEmits<{
  accept: []
  reject: []
  open: []
}>()

const dialogVisible = computed(() => props.visible && documents.value.length > 0)
const documents = computed(() => props.documents.filter((doc) => doc.title.trim()))
const updatedAt = computed(() => props.updatedAt || '')
const updatedAtLabel = computed(() => updatedAt.value || t('auth.agreementPrompt.recent'))
const accepted = computed(() => props.accepted)
const mode = computed(() => props.mode === 'checkbox' ? 'checkbox' : 'modal')
const guaranteeItems = computed(() => [
  t('auth.agreementPrompt.guarantees.official'),
  t('auth.agreementPrompt.guarantees.privacy'),
  t('auth.agreementPrompt.guarantees.billing')
])

function documentRoute(doc: LoginAgreementDocument) {
  return {
    name: 'LegalDocument',
    params: {
      documentId: doc.id || doc.title,
    },
  }
}

function handleCheckboxChange(event: Event): void {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    emit('accept')
  } else {
    emit('reject')
  }
}

function documentIcon(index: number, title: string): 'document' | 'shield' | 'globe' | 'cog' {
  const normalized = title.toLowerCase()
  if (title.includes('政策') || title.includes('隐私') || normalized.includes('policy') || normalized.includes('privacy')) {
    return 'shield'
  }
  if (
    title.includes('国家') ||
    title.includes('地区') ||
    normalized.includes('country') ||
    normalized.includes('region')
  ) {
    return 'globe'
  }
  if (index === 3 || normalized.includes('technical') || normalized.includes('ops') || normalized.includes('config')) {
    return 'cog'
  }
  return 'document'
}
</script>

<style scoped>
.login-agreement-shell {
  --login-agreement-inset-top: max(1rem, env(safe-area-inset-top, 0px));
  --login-agreement-inset-right: max(1rem, env(safe-area-inset-right, 0px));
  --login-agreement-inset-bottom: max(1rem, env(safe-area-inset-bottom, 0px));
  --login-agreement-inset-left: max(1rem, env(safe-area-inset-left, 0px));
  --login-agreement-inline-margin: max(var(--login-agreement-inset-left), var(--login-agreement-inset-right));
  --login-agreement-block-margin: max(var(--login-agreement-inset-top), var(--login-agreement-inset-bottom));
  --login-agreement-scrollbar-offset: calc((100vw - 100%) / 2);

  height: 100vh;
  height: 100svh;
  height: 100dvh;
  overflow: hidden;
  box-sizing: border-box;
  overscroll-behavior: contain;
}

.login-agreement-card {
  position: absolute;
  left: calc(50% + var(--login-agreement-scrollbar-offset));
  top: 50%;
  width: min(600px, calc(100% - var(--login-agreement-inline-margin) - var(--login-agreement-inline-margin)));
  max-height: calc(100% - var(--login-agreement-block-margin) - var(--login-agreement-block-margin));
  max-height: min(760px, calc(100dvh - var(--login-agreement-block-margin) - var(--login-agreement-block-margin)));
  transform: translate3d(-50%, -50%, 0) scale(1);
  transform-origin: center center;
}

.login-agreement-shell :deep(.btn) {
  border-radius: 0.625rem;
}

.login-agreement-shell :deep(.btn-primary) {
  border-color: rgba(77, 212, 230, 0.34);
  background: linear-gradient(135deg, #4dd4e6, #78e6f4);
  color: #071014;
  box-shadow: 0 14px 34px rgba(77, 212, 230, 0.16);
}

.login-agreement-shell :deep(.btn-secondary) {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.045);
  color: rgba(255, 255, 255, 0.82);
}

.login-agreement-shell :deep(.btn-secondary:hover) {
  border-color: rgba(255, 255, 255, 0.20);
  background: rgba(255, 255, 255, 0.075);
}

.agreement-fade-enter-active,
.agreement-fade-leave-active {
  transition: opacity 0.18s ease;
}

.agreement-fade-enter-from,
.agreement-fade-leave-to {
  opacity: 0;
}

.agreement-fade-enter-active > div,
.agreement-fade-leave-active > div {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.agreement-fade-enter-from > div,
.agreement-fade-leave-to > div {
  opacity: 0;
  transform: translate3d(-50%, calc(-50% + 8px), 0) scale(0.98);
}

.agreement-fade-enter-to > div,
.agreement-fade-leave-from > div {
  opacity: 1;
  transform: translate3d(-50%, -50%, 0) scale(1);
}

@media (prefers-reduced-motion: reduce) {
  .agreement-fade-enter-active,
  .agreement-fade-leave-active,
  .agreement-fade-enter-active > div,
  .agreement-fade-leave-active > div {
    transition-duration: 1ms;
  }

  .agreement-fade-enter-from > div,
  .agreement-fade-leave-to > div {
    transform: translate3d(-50%, -50%, 0) scale(1);
  }
}
</style>
