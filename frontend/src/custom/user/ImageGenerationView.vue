<template>
  <AppLayout>
    <div class="image-workbench mx-auto max-w-7xl space-y-5">
      <UserPageHero
        :kicker="t('imageGeneration.hero.kicker')"
        :title="t('imageGeneration.hero.title')"
      >
        <template #actions>
        <button class="image-soft-button shrink-0" :disabled="loadingPage" @click="loadInitialData">
          <Icon name="refresh" size="md" :class="loadingPage ? 'animate-spin' : ''" />
          {{ t('common.refresh') }}
        </button>
        </template>
      </UserPageHero>

      <div v-if="loadingPage && imageGroups.length === 0" class="flex justify-center py-16">
        <LoadingSpinner />
      </div>

      <div v-else class="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,380px)_minmax(0,1fr)]">
        <aside class="min-w-0 space-y-4 xl:sticky xl:top-5 xl:self-start">
          <section class="image-panel p-5">
            <div class="mb-5 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h2 class="image-section-title">{{ t('imageGeneration.createImageKey') }}</h2>
                <p class="image-section-description">{{ selectedGroup ? limitSummary : t('imageGeneration.selectGroup') }}</p>
              </div>
              <div class="image-icon-surface">
                <Icon name="sparkles" size="md" />
              </div>
            </div>

            <div class="space-y-4">
              <div>
                <label class="image-label">{{ t('imageGeneration.group') }}</label>
                <Select
                  :model-value="selectedGroupId"
                  :options="imageGroupOptions"
                  :placeholder="t('imageGeneration.selectGroup')"
                  :disabled="imageGroupOptions.length === 0"
                  searchable
                  @update:model-value="onSelectGroup"
                />
                <p v-if="imageGroups.length === 0" class="input-hint">
                  {{ t('imageGeneration.noImageGroup') }}
                </p>
                <p v-else-if="autoCreatingKeys" class="input-hint">
                  {{ t('imageGeneration.autoCreatingKeys') }}
                </p>
                <p v-else-if="selectedGroup && !selectedGroupKey" class="input-hint">
                  {{ t('imageGeneration.groupKeyUnavailable') }}
                </p>
              </div>

              <div>
                <label class="image-label">{{ t('imageGeneration.model') }}</label>
                <Select
                  :model-value="selectedImageModel"
                  :options="imageModelOptions"
                  :placeholder="t('imageGeneration.selectModel')"
                  :disabled="imageModelOptions.length === 0"
                  searchable
                  @update:model-value="onSelectModel"
                />
                <p v-if="selectedGroup && imageModelOptions.length === 0" class="input-hint">
                  {{ t('imageGeneration.noImageModel') }}
                </p>
              </div>

              <div>
                <label class="image-label">{{ t('imageGeneration.size') }}</label>
                <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-2">
                  <button
                    v-for="option in visibleSizeOptions"
                    :key="option.value"
                    type="button"
                    :class="[
                      'image-size-option',
                      size === option.value
                        ? 'image-size-option-active'
                        : 'image-size-option-idle'
                    ]"
                    @click="size = option.value"
                  >
                    {{ option.label }}
                  </button>
                </div>
                <p v-if="sizeCapabilityHint" class="input-hint">
                  {{ sizeCapabilityHint }}
                </p>
              </div>

              <div :class="supportsQualitySelection ? 'grid grid-cols-1 gap-3 sm:grid-cols-2' : 'grid grid-cols-1 gap-3'">
                <div v-if="supportsQualitySelection">
                  <label class="image-label">{{ t('imageGeneration.quality') }}</label>
                  <Select v-model="quality" :options="qualityOptions" />
                </div>
                <div>
                  <label class="image-label">{{ t('imageGeneration.count') }}</label>
                  <Select v-model="count" :options="countOptions" />
                </div>
              </div>
            </div>
          </section>

          <section class="image-panel p-5">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <h2 class="image-section-title">
                  {{ t('imageGeneration.pricing.title') }}
                </h2>
                <p class="mt-1 break-words text-xs leading-5 text-[color:var(--image-muted)]">
                  {{ pricingSummary }}
                </p>
              </div>
              <div class="image-icon-surface">
                <Icon name="calculator" size="md" />
              </div>
            </div>

            <div class="image-price-grid mt-4">
              <div class="image-price-cell">
                <div class="image-price-label">{{ t('imageGeneration.pricing.unitCost') }}</div>
                <div class="image-price-value">
                  {{ priceEstimate ? formatCredit(priceEstimate.unitCost) : t('common.notAvailable') }}
                </div>
                <div v-if="priceEstimate" class="image-price-subvalue">
                  {{ formatCny(priceEstimate.unitCost) }}
                </div>
              </div>
              <div class="image-price-cell">
                <div class="image-price-label">{{ t('imageGeneration.pricing.batchCost') }}</div>
                <div class="image-price-value">
                  {{ priceEstimate ? formatCredit(priceEstimate.batchCost) : t('common.notAvailable') }}
                </div>
                <div v-if="priceEstimate" class="image-price-subvalue">
                  {{ formatCny(priceEstimate.batchCost) }}
                </div>
              </div>
              <div class="image-price-cell">
                <div class="image-price-label">{{ t('imageGeneration.pricing.remainingImages') }}</div>
                <div class="image-price-value">
                  {{ priceEstimate ? t('imageGeneration.pricing.imageCountValue', { count: priceEstimate.remainingImages }) : t('common.notAvailable') }}
                </div>
                <div class="image-price-subvalue">
                  {{ limitSummary }}
                </div>
              </div>
              <div class="image-price-cell">
                <div class="image-price-label">{{ t('imageGeneration.pricing.balance') }}</div>
                <div class="image-price-value">
                  {{ formatCredit(userBalance) }}
                </div>
                <div class="image-price-subvalue">
                  {{ formatCny(userBalance) }}
                </div>
              </div>
            </div>
          </section>
        </aside>

        <main class="min-w-0 space-y-4">
          <section class="image-panel p-3 sm:p-4">
            <div
              class="image-prompt-frame"
              :class="isDragging && supportsReferenceImages ? 'image-prompt-frame-dragging' : ''"
              @dragenter="handleDragEnter"
              @dragover="handleDragOver"
              @dragleave="handleDragLeave"
              @drop="handleDrop"
            >
              <textarea
                v-model="prompt"
                rows="7"
                class="image-prompt-input"
                :placeholder="supportsReferenceImages && referenceImages.length > 0 ? t('imageGeneration.editPlaceholder') : t('imageGeneration.promptPlaceholder')"
                @paste="handlePaste"
                @keydown.ctrl.enter.prevent="submit"
                @keydown.meta.enter.prevent="submit"
              />

              <div class="border-t border-[color:var(--image-border-soft)] px-3 py-3 sm:px-4">
                <input v-if="supportsReferenceImages" ref="fileInputRef" type="file" accept="image/*" multiple class="hidden" @change="handleFileInput" />
                <div v-if="supportsReferenceImages && referenceImages.length > 0" class="mb-3 flex flex-wrap gap-2">
                  <div v-for="image in referenceImages" :key="image.id" class="group relative h-16 w-16 overflow-hidden rounded-lg border border-[color:var(--image-border)] bg-[var(--image-surface-muted)]">
                    <button type="button" class="h-full w-full" @click="previewImage = image.previewUrl">
                      <img :src="image.previewUrl" :alt="image.name" class="h-full w-full object-contain" />
                    </button>
                    <button
                      type="button"
                      class="absolute right-1 top-1 rounded-full bg-black/60 p-1 text-white opacity-100 transition sm:opacity-0 sm:group-hover:opacity-100"
                      :title="t('common.remove')"
                      @click="removeReference(image.id)"
                    >
                      <Icon name="x" size="xs" />
                    </button>
                  </div>
                </div>

                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="flex min-w-0 flex-wrap items-center gap-2">
                    <button v-if="supportsReferenceImages" type="button" class="image-soft-button image-soft-button-sm" @click="openFilePicker">
                      <Icon name="upload" size="sm" />
                      {{ referenceImages.length > 0 ? t('imageGeneration.addReference') : t('imageGeneration.uploadReference') }}
                    </button>
                    <span class="image-chip">
                      {{ modeLabel }}
                    </span>
                    <span v-if="selectedGroup" class="image-chip max-w-full truncate sm:max-w-[18rem]">
                      {{ selectedGroup.name }}
                    </span>
                    <span v-if="supportsReferenceImages && isDragging" class="image-chip image-chip-active">
                      {{ t('imageGeneration.dropImages') }}
                    </span>
                  </div>
                  <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center sm:justify-end">
                    <button v-if="generating" type="button" class="image-soft-button image-soft-button-sm justify-center" @click="cancelGeneration">
                      {{ t('common.cancel') }}
                    </button>
                    <button type="button" class="image-primary-button" :disabled="!canSubmit" @click="submit">
                      <Icon v-if="!generating" name="sparkles" size="md" />
                      <Icon v-else name="refresh" size="md" class="animate-spin" />
                      {{ generating ? t('imageGeneration.generating') : submitLabel }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section class="image-note">
            <div class="flex items-start gap-3">
              <div class="image-icon-surface shrink-0">
                <Icon name="infoCircle" size="sm" />
              </div>
              <p class="pt-0.5 text-sm leading-6 text-[color:var(--image-muted)]">
                {{ t('imageGeneration.localCacheWarning') }}
              </p>
            </div>
          </section>

          <section v-if="runs.length === 0" class="image-empty">
            <Icon name="sparkles" size="xl" class="mx-auto text-[color:var(--image-muted-soft)]" />
            <p class="mt-3 text-sm text-[color:var(--image-muted)]">{{ t('imageGeneration.empty') }}</p>
          </section>

          <section v-else class="space-y-4">
            <article v-for="run in runs" :key="run.id" class="image-panel overflow-hidden">
              <div class="border-b border-[color:var(--image-border-soft)] px-4 py-3 sm:px-5">
                <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <p class="line-clamp-2 break-words text-sm font-medium text-[color:var(--image-text)]">{{ run.prompt }}</p>
                    <p class="mt-1 break-words text-xs leading-5 text-[color:var(--image-muted)]">
                      {{ run.groupName }} · {{ run.model }} · {{ run.sizeLabel }} · {{ t('imageGeneration.pricing.imageCountValue', { count: run.requestedCount }) }}
                    </p>
                  </div>
                  <span
                    :class="[
                      'image-status-pill',
                      run.status === 'success' ? 'image-status-success' :
                      run.status === 'error' ? 'image-status-error' :
                      'image-status-loading'
                    ]"
                  >
                    {{ statusLabel(run.status) }}
                  </span>
                </div>
              </div>

              <div class="p-4 sm:p-5">
                <div v-if="run.status === 'loading'" class="image-result-grid">
                  <div v-for="index in run.requestedCount" :key="index" class="skeleton aspect-square rounded-lg" />
                </div>
                <div v-else-if="run.status === 'error'" class="rounded-lg border border-red-200/70 bg-red-50 p-4 text-sm leading-6 text-red-700 dark:border-red-500/25 dark:bg-red-500/10 dark:text-red-200">
                  <p>{{ run.error }}</p>
                  <p class="mt-2 text-xs leading-5 text-red-600 dark:text-red-200/80">
                    {{ t('imageGeneration.errorSupportHint') }}
                  </p>
                </div>
                <div v-else class="image-result-grid">
                  <div v-for="image in run.images" :key="image.id" class="group relative overflow-hidden rounded-lg border border-[color:var(--image-border-soft)] bg-[var(--image-surface-muted)]">
                    <button type="button" class="block aspect-square w-full" @click="previewImage = image.src">
                      <img :src="image.src" :alt="run.prompt" class="h-full w-full object-contain" />
                    </button>
                    <div class="absolute inset-x-2 bottom-2 flex flex-wrap justify-end gap-2 opacity-100 transition sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100">
                      <button v-if="supportsReferenceImages" type="button" class="image-overlay-button" @click="continueEditFromResult(image)">
                        <Icon name="edit" size="xs" />
                        {{ t('imageGeneration.continueEdit') }}
                      </button>
                      <button type="button" class="image-overlay-button" @click="downloadImage(image, run)">
                        <Icon name="download" size="xs" />
                        {{ t('common.download') }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </article>
          </section>
        </main>
      </div>

      <div
        v-if="previewImage"
        class="fixed inset-0 z-[100000030] flex items-center justify-center bg-black/80 p-4"
        @click="previewImage = ''"
      >
        <button class="absolute right-4 top-4 rounded-full bg-white/10 p-2 text-white hover:bg-white/20" :title="t('common.close')" @click="previewImage = ''">
          <Icon name="x" size="lg" />
        </button>
        <img :src="previewImage" alt="" class="max-h-full max-w-full rounded-lg object-contain" @click.stop />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { keysAPI } from '@/custom/api/keys'
import userGroupsAPI from '@/api/groups'
import {
  MAX_IMAGE_GENERATION_COUNT,
  normalizeOpenAIImageResults,
  submitGeminiImageGatewayRequest,
  submitImageGatewayRequest,
  type OpenAIImageResult,
} from '@/custom/api/imageGeneration'
import { useAppStore, useAuthStore } from '@/stores'
import type { ApiKey, Group, PublicSettings } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  DEFAULT_IMAGE_MODEL,
  isImageGenerationGroup,
  resolveSupportedImageModels,
} from '@/custom/utils/imageGenerationGroups'

type RunStatus = 'loading' | 'success' | 'error'

interface ReferenceImage {
  id: string
  file: File
  name: string
  previewUrl: string
}

interface CachedReferenceImage {
  id: string
  name: string
  previewUrl: string
}

interface GenerationRun {
  id: string
  prompt: string
  groupName: string
  model: string
  sizeLabel: string
  requestedCount: number
  status: RunStatus
  images: OpenAIImageResult[]
  error?: string
}

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const STORAGE_PREFIX = 'sub2api:image-generation:'
const IMAGE_CACHE_DB_NAME = 'sub2api-image-generation'
const IMAGE_CACHE_STORE_NAME = 'cache'
const IMAGE_CACHE_RUNS_KEY = 'runs'
const IMAGE_CACHE_REFERENCES_KEY = 'references'
const MAX_CACHED_IMAGES = 5
const MAX_REFERENCE_IMAGES = 4
const MAX_REFERENCE_IMAGE_SIZE = 10 * 1024 * 1024
const DEFAULT_IMAGE_PRICE_1K = 0.134

const availableGroups = ref<Group[]>([])
const apiKeys = ref<ApiKey[]>([])
const userGroupRates = ref<Record<number, number>>({})
const publicSettings = ref<PublicSettings | null>(null)
const loadingPage = ref(false)
const autoCreatingKeys = ref(false)
const autoCreatingGroupId = ref<number | null>(null)
const selectedGroupId = ref<number | null>(readStoredNumber('groupId'))
const selectedModelId = ref(localStorage.getItem(`${STORAGE_PREFIX}model`) || '')
const prompt = ref(localStorage.getItem(`${STORAGE_PREFIX}prompt`) || '')
const size = ref(localStorage.getItem(`${STORAGE_PREFIX}size`) || '1024x1024')
const quality = ref(localStorage.getItem(`${STORAGE_PREFIX}quality`) || 'auto')
const count = ref(Number(localStorage.getItem(`${STORAGE_PREFIX}count`) || '1'))
const referenceImages = ref<ReferenceImage[]>([])
const runs = ref<GenerationRun[]>([])
const generating = ref(false)
const isDragging = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)
const previewImage = ref('')
let activeController: AbortController | null = null
let runCachePersistVersion = 0
let runCachePersistChain: Promise<void> = Promise.resolve()
let referenceCachePersistVersion = 0
let referenceCachePersistChain: Promise<void> = Promise.resolve()

type ImageSizeTier = '1K' | '2K' | '4K'

interface ImageSizeOption {
  value: string
  label: string
  tier: ImageSizeTier
}

const openAIImageSizeOptions: readonly ImageSizeOption[] = [
  { value: '1024x1024', label: '1:1 · 1K', tier: '1K' },
  { value: '1024x1536', label: '2:3 · 1K', tier: '1K' },
  { value: '1536x1024', label: '3:2 · 1K', tier: '1K' },
  { value: '1024x1365', label: '3:4 · 1K', tier: '1K' },
  { value: '1365x1024', label: '4:3 · 1K', tier: '1K' },
  { value: '1088x1920', label: '9:16 · 1K', tier: '1K' },
  { value: '1920x1088', label: '16:9 · 1K', tier: '1K' },
  { value: 'auto', label: 'Auto · 1K', tier: '1K' },
] as const

const geminiImageSizeOptions: readonly ImageSizeOption[] = [
  ...openAIImageSizeOptions,
  { value: '2048x2048', label: '1:1 · 2K', tier: '2K' },
  { value: '2048x3072', label: '2:3 · 2K', tier: '2K' },
  { value: '3072x2048', label: '3:2 · 2K', tier: '2K' },
  { value: '2048x2730', label: '3:4 · 2K', tier: '2K' },
  { value: '2730x2048', label: '4:3 · 2K', tier: '2K' },
  { value: '2176x3840', label: '9:16 · 2K', tier: '2K' },
  { value: '3840x2176', label: '16:9 · 2K', tier: '2K' },
  { value: '4096x4096', label: '1:1 · 4K', tier: '4K' },
  { value: '4096x6144', label: '2:3 · 4K', tier: '4K' },
  { value: '6144x4096', label: '3:2 · 4K', tier: '4K' },
  { value: '4096x5460', label: '3:4 · 4K', tier: '4K' },
  { value: '5460x4096', label: '4:3 · 4K', tier: '4K' },
  { value: '4352x7680', label: '9:16 · 4K', tier: '4K' },
  { value: '7680x4352', label: '16:9 · 4K', tier: '4K' },
] as const

const grokImageSizeOptions: readonly ImageSizeOption[] = [
  { value: '1k', label: '1K', tier: '1K' },
]

const qualityOptions = computed(() => [
  { value: 'auto', label: t('imageGeneration.qualityOptions.auto') },
  { value: 'low', label: t('imageGeneration.qualityOptions.low') },
  { value: 'medium', label: t('imageGeneration.qualityOptions.medium') },
  { value: 'high', label: t('imageGeneration.qualityOptions.high') },
])

const countOptions = computed(() => Array.from({ length: MAX_IMAGE_GENERATION_COUNT }, (_, index) => {
  const value = index + 1
  return { value, label: t('imageGeneration.pricing.imageCountValue', { count: value }) }
}))

const imageGroups = computed(() => availableGroups.value.filter(isImageGenerationGroup))

const imageGroupOptions = computed(() => imageGroups.value.map((group) => ({
  value: group.id,
  label: group.name,
})))

const selectedGroup = computed(() => imageGroups.value.find((group) => group.id === selectedGroupId.value) || null)

const isGrokImageGroup = computed(() => selectedGroup.value?.platform.trim().toLowerCase() === 'grok')
const supportsReferenceImages = computed(() => !isGrokImageGroup.value)
const supportsQualitySelection = computed(() => !isGrokImageGroup.value)

const visibleSizeOptions = computed<readonly ImageSizeOption[]>(() => (
  isGrokImageGroup.value
    ? grokImageSizeOptions
    : selectedGroup.value?.platform === 'gemini'
      ? geminiImageSizeOptions.filter((option) => supportsGeminiImageSizeTier(selectedImageModel.value, option.tier))
      : openAIImageSizeOptions
))

const sizeCapabilityHint = computed(() => {
  if (selectedGroup.value?.platform !== 'gemini') return ''
  const tiers = new Set(visibleSizeOptions.value.map((option) => option.tier))
  if (!tiers.has('2K') && !tiers.has('4K')) {
    return t('imageGeneration.geminiFlashSizeHint')
  }
  return ''
})

const imageModelOptions = computed(() => {
  const group = selectedGroup.value
  if (!group) return []
  return resolveSupportedImageModels(group).map((model) => ({
    value: model,
    label: model,
  }))
})

const selectedImageModel = computed(() => {
  const current = selectedModelId.value.trim()
  if (current && imageModelOptions.value.some((option) => option.value === current)) {
    return current
  }
  return String(imageModelOptions.value[0]?.value || '')
})

const activeImageKeys = computed(() => apiKeys.value.filter((key) => key.status === 'active' && key.group_id !== null))

const selectedGroupKey = computed(() => activeImageKeys.value.find((key) => key.group_id === selectedGroupId.value) || null)

const selectedSize = computed(() => visibleSizeOptions.value.find((option) => option.value === size.value) || visibleSizeOptions.value[0])
const userBalance = computed(() => Number(authStore.user?.balance || 0))
const gatewayBaseUrl = computed(() => publicSettings.value?.api_base_url || window.location.origin)
const cnyPerCredit = computed(() => {
  const value = Number(publicSettings.value?.payment_balance_recharge_multiplier)
  return Number.isFinite(value) && value > 0 ? value : 6.8
})

const selectedGroupKeyQuotaRemaining = computed(() => {
  const key = selectedGroupKey.value
  if (!key || !Number.isFinite(Number(key.quota)) || Number(key.quota) <= 0) {
    return null
  }
  return Math.max(0, Number(key.quota) - Number(key.quota_used || 0))
})

const priceEstimate = computed(() => {
  const group = selectedGroup.value
  if (!group) return null
  const tier = selectedSize.value.tier
  const basePrice = resolveImageBasePrice(group, tier)
  const multiplier = resolveImageMultiplier(group)
  const unitCost = Math.max(0, basePrice.price * multiplier)
  const requestedCount = normalizeCount(count.value)
  const keyQuotaRemaining = selectedGroupKeyQuotaRemaining.value
  const spendableBalance = Math.max(0, Math.min(userBalance.value, keyQuotaRemaining ?? userBalance.value))
  return {
    tier,
    basePrice: basePrice.price,
    priceSource: basePrice.source,
    multiplier,
    unitCost,
    batchCost: unitCost * requestedCount,
    remainingImages: unitCost > 0 ? Math.floor(spendableBalance / unitCost) : 0,
  }
})

const pricingSummary = computed(() => {
  if (!priceEstimate.value) {
    return t('imageGeneration.pricing.noEstimate')
  }
  const sourceLabel = priceEstimate.value.priceSource === 'group'
    ? t('imageGeneration.pricing.groupPrice')
    : t('imageGeneration.pricing.defaultPrice')
  return t('imageGeneration.pricing.summary', {
    source: sourceLabel,
    base: formatCredit(priceEstimate.value.basePrice),
    multiplier: formatMultiplier(priceEstimate.value.multiplier),
    tier: priceEstimate.value.tier,
  })
})

const limitSummary = computed(() => {
  if (autoCreatingKeys.value) {
    return t('imageGeneration.pricing.autoCreatingKey')
  }
  if (!selectedGroupKey.value) {
    return t('imageGeneration.pricing.noKey')
  }
  if (selectedGroupKeyQuotaRemaining.value !== null && selectedGroupKeyQuotaRemaining.value < userBalance.value) {
    return t('imageGeneration.pricing.keyQuotaLimited')
  }
  return t('imageGeneration.pricing.balanceLimited')
})

const canSubmit = computed(() => Boolean(
  !generating.value &&
  selectedGroup.value &&
  selectedGroupKey.value &&
  selectedImageModel.value &&
  prompt.value.trim() &&
  normalizeCount(count.value) > 0,
))

const modeLabel = computed(() => supportsReferenceImages.value && referenceImages.value.length > 0
  ? t('imageGeneration.editMode')
  : t('imageGeneration.generateMode'))

const submitLabel = computed(() => supportsReferenceImages.value && referenceImages.value.length > 0
  ? t('imageGeneration.submitEdit')
  : t('imageGeneration.submitGenerate'))

function readStoredNumber(key: string) {
  const raw = localStorage.getItem(`${STORAGE_PREFIX}${key}`)
  const parsed = raw ? Number(raw) : NaN
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

function createId(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function isDataUrl(value: string) {
  return /^data:image\/.+;base64,/i.test(value)
}

function normalizeCount(value: unknown) {
  const parsed = Math.floor(Number(value) || 1)
  return Math.min(MAX_IMAGE_GENERATION_COUNT, Math.max(1, parsed))
}

function resolvePreferredGroupId(groups: Group[], keys: ApiKey[]) {
  if (groups.length === 0) {
    return null
  }
  const activeGroupIDs = new Set(keys.filter((key) => key.status === 'active' && key.group_id !== null).map((key) => key.group_id))
  if (selectedGroupId.value && groups.some((group) => group.id === selectedGroupId.value)) {
    return selectedGroupId.value
  }
  const firstWithKey = groups.find((group) => activeGroupIDs.has(group.id))
  return firstWithKey?.id || groups[0].id
}

function buildAutoImageKeyName(group: Group) {
  return t('imageGeneration.autoAccessName', { group: group.name })
}

function hasActiveKeyForGroup(keys: ApiKey[], groupId: number) {
  return keys.some((key) => key.status === 'active' && key.group_id === groupId)
}

function normalizeImageModelName(model: string) {
  return model.trim().toLowerCase().replace(/^models\//, '')
}

function supportsGeminiImageSizeTier(model: string, tier: string) {
  if (tier === '1K') return true
  const normalized = normalizeImageModelName(model)
  return normalized === 'gemini-3.1-flash-image'
    || normalized === 'gemini-3-pro-image'
    || normalized.startsWith('gemini-3-pro-image-')
}

function resolveImageBasePrice(group: Group, tier: string) {
  const value = tier === '2K'
    ? group.image_price_2k
    : tier === '4K'
      ? group.image_price_4k
      : group.image_price_1k
  if (typeof value === 'number' && Number.isFinite(value) && value >= 0) {
    return { price: value, source: 'group' as const }
  }
  const multiplier = tier === '2K' ? 1.5 : tier === '4K' ? 2 : 1
  return { price: DEFAULT_IMAGE_PRICE_1K * multiplier, source: 'default' as const }
}

function resolveGroupMultiplier(group: Group) {
  const userRate = userGroupRates.value[group.id]
  const base = Number.isFinite(Number(userRate)) && Number(userRate) >= 0
    ? Number(userRate)
    : Number(group.rate_multiplier || 1)
  const discount = resolveDiscountMultiplier(group)
  if (Number.isFinite(discount) && discount > 0) {
    return base * discount
  }
  const discounted = Number(group.discounted_rate_multiplier)
  if (Number.isFinite(discounted) && discounted >= 0 && !(Number.isFinite(Number(userRate)) && Number(userRate) >= 0)) {
    return discounted
  }
  return base
}

function resolveDiscountMultiplier(group: Group) {
  const direct = Number(group.group_rate_discount_multiplier)
  if (Number.isFinite(direct) && direct > 0) {
    return direct
  }
  const discounted = Number(group.discounted_rate_multiplier)
  const base = Number(group.rate_multiplier)
  if (Number.isFinite(discounted) && discounted >= 0 && Number.isFinite(base) && base > 0) {
    return discounted / base
  }
  return 1
}

function resolveImageMultiplier(group: Group) {
  if (group.image_rate_independent) {
    const imageMultiplier = Number(group.image_rate_multiplier)
    const base = Number.isFinite(imageMultiplier) && imageMultiplier >= 0 ? imageMultiplier : 1
    return base * resolveDiscountMultiplier(group)
  }
  return resolveGroupMultiplier(group)
}

function formatCredit(value: number) {
  if (!Number.isFinite(value)) return '-'
  return `$${value.toFixed(6).replace(/0+$/, '').replace(/\.$/, '')}`
}

function formatCny(value: number) {
  if (!Number.isFinite(value)) return ''
  return `≈ ¥${(value * cnyPerCredit.value).toFixed(4).replace(/0+$/, '').replace(/\.$/, '')}`
}

function formatMultiplier(value: number) {
  if (!Number.isFinite(value)) return '1'
  return value.toFixed(6).replace(/0+$/, '').replace(/\.$/, '')
}

function onSelectGroup(value: string | number | boolean | null) {
  selectedGroupId.value = typeof value === 'number' ? value : value ? Number(value) : null
}

function onSelectModel(value: string | number | boolean | null) {
  selectedModelId.value = typeof value === 'string' ? value : value ? String(value) : ''
}

function syncSelectedModel() {
  selectedModelId.value = selectedImageModel.value
}

function syncSelectedSize() {
  if (!visibleSizeOptions.value.some((option) => option.value === size.value)) {
    size.value = visibleSizeOptions.value[0]?.value || '1024x1024'
  }
}

function hasBrowserImageCache() {
  return typeof window !== 'undefined' && 'indexedDB' in window
}

function idbRequestToPromise<T>(request: IDBRequest<T>) {
  return new Promise<T>((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

function idbTransactionToPromise(transaction: IDBTransaction) {
  return new Promise<void>((resolve, reject) => {
    transaction.oncomplete = () => resolve()
    transaction.onabort = () => reject(transaction.error)
    transaction.onerror = () => reject(transaction.error)
  })
}

function openImageCacheDB() {
  return new Promise<IDBDatabase>((resolve, reject) => {
    const request = window.indexedDB.open(IMAGE_CACHE_DB_NAME, 1)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(IMAGE_CACHE_STORE_NAME)) {
        db.createObjectStore(IMAGE_CACHE_STORE_NAME)
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

function cloneRun(run: GenerationRun): GenerationRun {
  return {
    ...run,
    images: run.images.map((image) => ({
      id: image.id,
      src: image.src,
      output_format: image.output_format,
      quality: image.quality,
      size: image.size,
      revised_prompt: image.revised_prompt,
    })),
  }
}

function buildCachedRuns(source: GenerationRun[]) {
  const cachedRuns: GenerationRun[] = []
  let remainingImages = MAX_CACHED_IMAGES

  for (const run of source) {
    if (run.status !== 'success' || run.images.length === 0 || remainingImages <= 0) {
      continue
    }

    const keptImages = run.images.slice(0, remainingImages)
    if (keptImages.length === 0) {
      continue
    }

    cachedRuns.push(cloneRun({
      ...run,
      requestedCount: keptImages.length,
      images: keptImages,
    }))
    remainingImages -= keptImages.length
  }

  return cachedRuns
}

function buildCachedReferenceImages(source: ReferenceImage[]) {
  return source
    .filter((image) => isDataUrl(image.previewUrl))
    .slice(0, MAX_REFERENCE_IMAGES)
    .map((image) => ({
      id: image.id,
      name: image.name,
      previewUrl: image.previewUrl,
    }))
}

function normalizeCachedRuns(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as GenerationRun[]
  }
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') {
      return []
    }
    const run = item as Partial<GenerationRun>
    if (run.status !== 'success' || !Array.isArray(run.images) || typeof run.prompt !== 'string') {
      return []
    }
    return [{
      id: typeof run.id === 'string' ? run.id : createId('cached-run'),
      prompt: run.prompt,
      groupName: typeof run.groupName === 'string' ? run.groupName : '',
      model: typeof run.model === 'string' ? run.model : DEFAULT_IMAGE_MODEL,
      sizeLabel: typeof run.sizeLabel === 'string' ? run.sizeLabel : '',
      requestedCount: normalizeCount(run.requestedCount),
      status: 'success' as const,
      images: run.images
        .filter((image) => image && typeof image === 'object' && typeof image.src === 'string')
        .map((image, index) => ({
          ...(image as OpenAIImageResult),
          id: typeof image.id === 'string' ? image.id : `cached-image-${index + 1}`,
        })),
    }]
  })
}

function normalizeCachedReferenceImages(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as CachedReferenceImage[]
  }
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') {
      return []
    }
    const image = item as Partial<CachedReferenceImage>
    if (typeof image.previewUrl !== 'string' || !isDataUrl(image.previewUrl)) {
      return []
    }
    return [{
      id: typeof image.id === 'string' ? image.id : createId('cached-ref'),
      name: typeof image.name === 'string' && image.name.trim() ? image.name : 'reference.png',
      previewUrl: image.previewUrl,
    }]
  })
}

function runCacheKey(runId: string) {
  return `run:${runId}`
}

async function readCacheEntry(key: string) {
  if (!hasBrowserImageCache()) {
    return null
  }
  const db = await openImageCacheDB()
  try {
    const transaction = db.transaction(IMAGE_CACHE_STORE_NAME, 'readonly')
    const store = transaction.objectStore(IMAGE_CACHE_STORE_NAME)
    const value = await idbRequestToPromise(store.get(key))
    await idbTransactionToPromise(transaction)
    return value
  } finally {
    db.close()
  }
}

async function writeCacheEntry(key: string, value: unknown) {
  if (!hasBrowserImageCache()) {
    return
  }
  const db = await openImageCacheDB()
  try {
    const transaction = db.transaction(IMAGE_CACHE_STORE_NAME, 'readwrite')
    const store = transaction.objectStore(IMAGE_CACHE_STORE_NAME)
    store.put(value, key)
    await idbTransactionToPromise(transaction)
  } finally {
    db.close()
  }
}

async function deleteCacheEntries(keys: string[]) {
  if (!hasBrowserImageCache() || keys.length === 0) {
    return
  }
  const db = await openImageCacheDB()
  try {
    const transaction = db.transaction(IMAGE_CACHE_STORE_NAME, 'readwrite')
    const store = transaction.objectStore(IMAGE_CACHE_STORE_NAME)
    keys.forEach((key) => store.delete(key))
    await idbTransactionToPromise(transaction)
  } finally {
    db.close()
  }
}

async function readCachedRuns() {
  try {
    const cached = await readCacheEntry(IMAGE_CACHE_RUNS_KEY)
    if (Array.isArray(cached) && cached.every((item) => typeof item === 'string')) {
      const runs = await Promise.all(cached.map((id) => readCacheEntry(runCacheKey(id))))
      return normalizeCachedRuns(runs)
    }
    return normalizeCachedRuns(cached)
  } catch {
    return [] as GenerationRun[]
  }
}

async function readCachedReferenceImages() {
  try {
    return normalizeCachedReferenceImages(await readCacheEntry(IMAGE_CACHE_REFERENCES_KEY))
  } catch {
    return [] as CachedReferenceImage[]
  }
}

async function persistRunsSnapshotToBrowserCache(candidates: GenerationRun[]) {
  if (candidates.length === 0) {
    try {
      const previousIndex = await readCacheEntry(IMAGE_CACHE_RUNS_KEY)
      const previousIds = Array.isArray(previousIndex) ? previousIndex.filter((item): item is string => typeof item === 'string') : []
      await deleteCacheEntries([IMAGE_CACHE_RUNS_KEY, ...previousIds.map(runCacheKey)])
    } catch {
      // Ignore browser cache cleanup failures.
    }
    return
  }

  const previousIndex = await readCacheEntry(IMAGE_CACHE_RUNS_KEY)
  const previousIds = Array.isArray(previousIndex) && previousIndex.every((item) => typeof item === 'string')
    ? previousIndex.filter((item): item is string => typeof item === 'string')
    : []
  const candidateIds = candidates.map((run) => run.id)

  try {
    await deleteCacheEntries([
      IMAGE_CACHE_RUNS_KEY,
      ...previousIds.filter((id) => !candidateIds.includes(id)).map(runCacheKey),
    ])
  } catch {
    // Ignore pre-clean failures and continue with best effort writes.
  }

  const savedIds: string[] = []
  for (const [index, run] of candidates.entries()) {
    try {
      await writeCacheEntry(runCacheKey(run.id), run)
      savedIds.push(run.id)
    } catch {
      if (index === 0) {
        try {
          const reclaimIds = Array.from(new Set([
            ...previousIds.filter((id) => id !== run.id),
            ...candidates.slice(1).map((item) => item.id),
          ]))
          await deleteCacheEntries(reclaimIds.map(runCacheKey))
          await writeCacheEntry(runCacheKey(run.id), run)
          savedIds.push(run.id)
          continue
        } catch {
          break
        }
      }
      break
    }
  }

  try {
    const unsavedIds = candidateIds.filter((id) => !savedIds.includes(id))
    await deleteCacheEntries(unsavedIds.map(runCacheKey))
    if (savedIds.length === 0) {
      await deleteCacheEntries([IMAGE_CACHE_RUNS_KEY])
    } else {
      await writeCacheEntry(IMAGE_CACHE_RUNS_KEY, savedIds)
    }
  } catch {
    // Ignore manifest write failures.
  }
}

function scheduleRunsPersistence() {
  const version = ++runCachePersistVersion
  const snapshot = buildCachedRuns(runs.value)
  runCachePersistChain = runCachePersistChain
    .catch(() => undefined)
    .then(async () => {
      if (version !== runCachePersistVersion) {
        return
      }
      await persistRunsSnapshotToBrowserCache(snapshot)
    })
}

async function persistReferenceImagesSnapshotToBrowserCache(candidates: CachedReferenceImage[]) {
  try {
    if (candidates.length === 0) {
      await deleteCacheEntries([IMAGE_CACHE_REFERENCES_KEY])
      return
    }
    await writeCacheEntry(IMAGE_CACHE_REFERENCES_KEY, candidates)
  } catch {
    // Ignore browser cache write failures.
  }
}

function scheduleReferenceImagesPersistence() {
  const version = ++referenceCachePersistVersion
  const snapshot = buildCachedReferenceImages(referenceImages.value)
  referenceCachePersistChain = referenceCachePersistChain
    .catch(() => undefined)
    .then(async () => {
      if (version !== referenceCachePersistVersion) {
        return
      }
      await persistReferenceImagesSnapshotToBrowserCache(snapshot)
    })
}

async function restoreRunsFromBrowserCache() {
  try {
    const cachedRuns = await readCachedRuns()
    if (cachedRuns.length > 0) {
      runs.value = cachedRuns
    }
  } catch {
    // Ignore browser cache restore failures.
  }
}

async function restoreReferenceImagesFromBrowserCache() {
  try {
    const cachedImages = await readCachedReferenceImages()
    if (cachedImages.length === 0) {
      return
    }
    referenceImages.value = cachedImages.map((image) => ({
      id: image.id,
      name: image.name,
      previewUrl: image.previewUrl,
      file: dataUrlToFile(image.previewUrl, image.name),
    }))
  } catch {
    // Ignore browser cache restore failures.
  }
}

async function loadActiveKeys() {
  const keyPage = await keysAPI.list(1, 200, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })
  return keyPage.items || []
}

async function ensureKeyForGroup(group: Group | null) {
  if (!group) {
    return
  }
  if (hasActiveKeyForGroup(apiKeys.value, group.id)) {
    return
  }
  if (autoCreatingGroupId.value === group.id) {
    return
  }
  autoCreatingKeys.value = true
  autoCreatingGroupId.value = group.id
  try {
    const created = await keysAPI.create({
      name: buildAutoImageKeyName(group),
      group_id: group.id
    })
    apiKeys.value = [created, ...apiKeys.value.filter((item) => item.id !== created.id)]
    appStore.showSuccess(t('imageGeneration.autoCreatedKey', { group: group.name }))
  } catch {
    try {
      apiKeys.value = await loadActiveKeys()
    } catch {
      // Ignore refresh failures and keep the current state.
    }
    if (!hasActiveKeyForGroup(apiKeys.value, group.id)) {
      appStore.showWarning(t('imageGeneration.autoCreateKeyFailed', { group: group.name }))
    }
  } finally {
    autoCreatingGroupId.value = null
    autoCreatingKeys.value = false
  }
}

async function loadInitialData() {
  loadingPage.value = true
  try {
    const [groups, keyPage, rates, settings] = await Promise.all([
      userGroupsAPI.getAvailable(),
      keysAPI.list(1, 200, { status: 'active', sort_by: 'created_at', sort_order: 'desc' }),
      userGroupsAPI.getUserGroupRates().catch(() => ({} as Record<number, number>)),
      appStore.fetchPublicSettings(false),
      authStore.refreshUser().catch(() => null),
    ])

    availableGroups.value = groups || []
    userGroupRates.value = rates || {}
    publicSettings.value = settings || appStore.cachedPublicSettings

    const targetGroups = availableGroups.value.filter(isImageGenerationGroup)
    apiKeys.value = keyPage.items || []
    selectedGroupId.value = resolvePreferredGroupId(targetGroups, apiKeys.value)
    syncSelectedModel()
    syncSelectedSize()
    await ensureKeyForGroup(selectedGroup.value)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('imageGeneration.loadKeysFailed')))
  } finally {
    autoCreatingKeys.value = false
    autoCreatingGroupId.value = null
    loadingPage.value = false
  }
}

function isImageFile(file: File) {
  return file.type.startsWith('image/') || /\.(avif|bmp|gif|heic|heif|ico|jpe?g|png|svg|tiff?|webp)$/i.test(file.name)
}

function fileToDataUrl(file: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

async function addReferenceFiles(files: File[], previewUrls?: string[]) {
  if (!supportsReferenceImages.value) {
    return
  }
  const next = [...referenceImages.value]
  for (const [index, file] of files.entries()) {
    if (!isImageFile(file)) {
      appStore.showWarning(t('imageGeneration.invalidReference'))
      continue
    }
    if (file.size > MAX_REFERENCE_IMAGE_SIZE) {
      appStore.showWarning(t('imageGeneration.referenceTooLarge'))
      continue
    }
    if (next.length >= MAX_REFERENCE_IMAGES) {
      appStore.showWarning(t('imageGeneration.referenceLimit', { count: MAX_REFERENCE_IMAGES }))
      break
    }
    const previewUrl = previewUrls?.[index] || await fileToDataUrl(file)
    next.push({
      id: createId('ref'),
      file,
      name: file.name || 'reference.png',
      previewUrl,
    })
  }
  referenceImages.value = next
}

function removeReference(id: string) {
  referenceImages.value = referenceImages.value.filter((image) => image.id !== id)
}

function openFilePicker() {
  if (!supportsReferenceImages.value) {
    return
  }
  fileInputRef.value?.click()
}

function handleFileInput(event: Event) {
  const input = event.target as HTMLInputElement
  void addReferenceFiles(Array.from(input.files || []))
  input.value = ''
}

function handlePaste(event: ClipboardEvent) {
  const files = Array.from(event.clipboardData?.files || []).filter(isImageFile)
  if (files.length === 0) return
  event.preventDefault()
  if (!supportsReferenceImages.value) return
  void addReferenceFiles(files)
}

function hasDraggedImages(dataTransfer: DataTransfer | null) {
  if (!dataTransfer) return false
  const items = Array.from(dataTransfer.items || [])
  if (items.length > 0) {
    return items.some((item) => item.kind === 'file' && (item.type.startsWith('image/') || !item.type))
  }
  return Array.from(dataTransfer.files || []).some(isImageFile)
}

function getDraggedImageFiles(dataTransfer: DataTransfer | null) {
  if (!dataTransfer) return []
  return Array.from(dataTransfer.files || []).filter(isImageFile)
}

function handleDragEnter(event: DragEvent) {
  if (!hasDraggedImages(event.dataTransfer)) return
  event.preventDefault()
  if (!supportsReferenceImages.value) return
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  isDragging.value = true
}

function handleDragOver(event: DragEvent) {
  if (!hasDraggedImages(event.dataTransfer)) return
  event.preventDefault()
  if (!supportsReferenceImages.value) return
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  isDragging.value = true
}

function handleDragLeave(event: DragEvent) {
  if (!supportsReferenceImages.value) return
  const current = event.currentTarget as HTMLElement | null
  if (event.relatedTarget instanceof Node && current?.contains(event.relatedTarget)) {
    return
  }
  isDragging.value = false
}

function handleDrop(event: DragEvent) {
  if (!hasDraggedImages(event.dataTransfer)) return
  event.preventDefault()
  isDragging.value = false
  if (!supportsReferenceImages.value) return
  void addReferenceFiles(getDraggedImageFiles(event.dataTransfer))
}

function updateRun(runId: string, patch: Partial<GenerationRun>) {
  runs.value = runs.value.map((item) => item.id === runId ? { ...item, ...patch } : item)
}

function isFetchNetworkError(error: unknown) {
  if (!error) return false
  const name = error instanceof Error ? error.name.toLowerCase() : ''
  const message = error instanceof Error ? error.message : String(error)
  const normalized = message.toLowerCase()
  const browserNetworkMessages = [
    'failed to fetch',
    'load failed',
    'networkerror',
    'network request failed',
    'fetch failed',
    'connection reset',
    'broken pipe',
  ]
  return name === 'typeerror' && browserNetworkMessages.some((item) => normalized.includes(item))
}

function resolveImageGenerationErrorMessage(error: unknown) {
  if (isFetchNetworkError(error)) {
    return t('imageGeneration.networkDisconnected')
  }
  return extractApiErrorMessage(error, t('imageGeneration.generateFailed'))
}

async function submit() {
  if (!canSubmit.value || !selectedGroup.value || !selectedGroupKey.value) return
  const model = selectedImageModel.value
  if (!model) return
  const controller = new AbortController()
  activeController = controller
  generating.value = true
  const requestedCount = normalizeCount(count.value)
  const run: GenerationRun = {
    id: createId('run'),
    prompt: prompt.value.trim(),
    groupName: selectedGroup.value.name,
    model,
    sizeLabel: selectedSize.value.label,
    requestedCount,
    status: 'loading',
    images: [],
  }
  runs.value = [run, ...runs.value]

  try {
    const request = {
      apiKey: selectedGroupKey.value.key,
      baseUrl: gatewayBaseUrl.value,
      platform: selectedGroup.value.platform,
      prompt: run.prompt,
      model,
      count: requestedCount,
      size: isGrokImageGroup.value ? undefined : selectedSize.value.value,
      quality: supportsQualitySelection.value ? quality.value : undefined,
      referenceImages: supportsReferenceImages.value ? referenceImages.value.map((image) => image.file) : [],
      signal: controller.signal,
    }
    const payload = selectedGroup.value.platform === 'gemini'
      ? await submitGeminiImageGatewayRequest(request)
      : await submitImageGatewayRequest(request)
    const images = normalizeOpenAIImageResults(payload)
    if (images.length === 0) {
      throw new Error(t('imageGeneration.noImagesReturned'))
    }
    updateRun(run.id, {
      status: 'success',
      images,
    })
    appStore.showSuccess(t('imageGeneration.generatedSuccess'))
    authStore.refreshUser().catch(() => null)
  } catch (error) {
    if (controller.signal.aborted) {
      updateRun(run.id, {
        status: 'error',
        error: t('imageGeneration.cancelled'),
      })
    } else {
      const message = resolveImageGenerationErrorMessage(error)
      updateRun(run.id, {
        status: 'error',
        error: message,
      })
      appStore.showError(message)
    }
  } finally {
    generating.value = false
    activeController = null
  }
}

function cancelGeneration() {
  activeController?.abort()
}

function statusLabel(status: RunStatus) {
  if (status === 'success') return t('common.success')
  if (status === 'error') return t('common.error')
  return t('imageGeneration.generating')
}

function dataUrlToFile(dataUrl: string, fileName: string) {
  const [header, content] = dataUrl.split(',', 2)
  const mimeType = header.match(/^data:(.*?);base64/i)?.[1] || 'image/png'
  const binary = atob(content || '')
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new File([bytes], fileName, { type: mimeType })
}

function buildGeneratedImageFileName(image: OpenAIImageResult) {
  const extension = image.output_format === 'jpeg' ? 'jpg' : image.output_format || 'png'
  return `generated-${Date.now()}.${extension}`
}

async function imageResultToDataUrl(image: OpenAIImageResult) {
  if (isDataUrl(image.src)) {
    return image.src
  }
  const response = await fetch(image.url || image.src)
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }
  return fileToDataUrl(await response.blob())
}

async function continueEditFromResult(image: OpenAIImageResult) {
  if (!supportsReferenceImages.value) {
    return
  }
  try {
    const previewUrl = await imageResultToDataUrl(image)
    const file = dataUrlToFile(previewUrl, buildGeneratedImageFileName(image))
    await addReferenceFiles([file], [previewUrl])
    appStore.showSuccess(t('imageGeneration.referenceAdded'))
  } catch {
    appStore.showError(t('imageGeneration.referenceFromResultFailed'))
  }
}

function downloadImage(image: OpenAIImageResult, run: GenerationRun) {
  const link = document.createElement('a')
  link.href = image.src
  const format = image.output_format || (image.src.includes('image/webp') ? 'webp' : 'png')
  link.download = `${run.model}-${Date.now()}.${format === 'jpeg' ? 'jpg' : format}`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

watch([selectedGroupId, selectedModelId, size, quality, count, prompt], () => {
  const normalizedCount = normalizeCount(count.value)
  if (count.value !== normalizedCount) {
    count.value = normalizedCount
    return
  }
  if (selectedGroupId.value) {
    localStorage.setItem(`${STORAGE_PREFIX}groupId`, String(selectedGroupId.value))
  }
  if (selectedImageModel.value) {
    localStorage.setItem(`${STORAGE_PREFIX}model`, selectedImageModel.value)
  }
  localStorage.setItem(`${STORAGE_PREFIX}size`, size.value)
  localStorage.setItem(`${STORAGE_PREFIX}quality`, quality.value)
  localStorage.setItem(`${STORAGE_PREFIX}count`, String(count.value))
  localStorage.setItem(`${STORAGE_PREFIX}prompt`, prompt.value)
})

watch(runs, () => {
  scheduleRunsPersistence()
}, { deep: true })

watch(referenceImages, () => {
  if (!supportsReferenceImages.value && referenceImages.value.length > 0) {
    referenceImages.value = []
    return
  }
  scheduleReferenceImagesPersistence()
}, { deep: true })

watch(selectedGroup, (group) => {
  if (!group) {
    return
  }
  if (!supportsReferenceImages.value && referenceImages.value.length > 0) {
    referenceImages.value = []
  }
  if (loadingPage.value) {
    return
  }
  syncSelectedModel()
  syncSelectedSize()
  void ensureKeyForGroup(group)
})

onMounted(() => {
  void restoreReferenceImagesFromBrowserCache()
  void restoreRunsFromBrowserCache()
  void loadInitialData()
})

onUnmounted(() => {
  activeController?.abort()
})
</script>

<style scoped>
.image-workbench {
  --image-text: var(--apple-text);
  --image-muted: var(--apple-muted);
  --image-muted-soft: var(--apple-muted-2);
  --image-surface: color-mix(in srgb, var(--apple-surface) 94%, transparent);
  --image-surface-elevated: var(--apple-surface-elevated);
  --image-surface-muted: color-mix(in srgb, var(--apple-surface-elevated) 90%, var(--apple-bg));
  --image-border: var(--apple-border);
  --image-border-soft: var(--apple-border-soft);
  --image-accent: var(--apple-blue);
  --image-accent-soft: color-mix(in srgb, var(--apple-blue) 10%, transparent);
  --image-shadow: var(--apple-shadow-sm);
  color: var(--image-text);
}

:global(.dark) .image-workbench {
  --image-surface: color-mix(in srgb, var(--apple-surface) 92%, transparent);
  --image-surface-muted: color-mix(in srgb, var(--apple-surface-elevated) 72%, var(--apple-bg));
}

.image-panel {
  min-width: 0;
  border: 1px solid var(--image-border);
  border-radius: 8px;
  background: var(--image-surface);
  box-shadow: var(--image-shadow);
}

.image-section-title {
  color: var(--image-text);
  font-size: 0.875rem;
  font-weight: 650;
  line-height: 1.35;
}

.image-section-description {
  margin-top: 0.25rem;
  color: var(--image-muted);
  font-size: 0.75rem;
  line-height: 1.55;
}

.image-label {
  margin-bottom: 0.375rem;
  display: block;
  color: var(--image-muted);
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.4;
}

.image-icon-surface {
  display: inline-flex;
  height: 2.25rem;
  width: 2.25rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--image-border-soft);
  border-radius: 8px;
  background: var(--image-surface-elevated);
  color: var(--image-muted);
}

.image-soft-button,
.image-primary-button {
  display: inline-flex;
  min-width: 0;
  max-width: 100%;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 8px;
  padding: 0.5rem 0.875rem;
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.25rem;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease, opacity 150ms ease;
}

.image-soft-button {
  border: 1px solid var(--image-border);
  background: var(--image-surface-elevated);
  color: var(--image-text);
}

.image-soft-button:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--image-border) 72%, var(--image-accent));
  background: var(--image-surface-muted);
}

.image-soft-button-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.image-primary-button {
  width: 100%;
  border: 1px solid transparent;
  background: var(--apple-blue);
  color: #fff;
  box-shadow: none;
}

:global(.dark) .image-primary-button {
  background: var(--apple-blue);
  color: #fff;
}

.image-primary-button:hover:not(:disabled) {
  background: var(--apple-blue-hover);
}

.image-soft-button:disabled,
.image-primary-button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

@media (min-width: 640px) {
  .image-primary-button {
    width: auto;
  }
}

.image-chip {
  display: inline-flex;
  min-width: 0;
  max-width: 100%;
  align-items: center;
  border: 1px solid var(--image-border-soft);
  border-radius: 8px;
  background: var(--image-surface-elevated);
  padding: 0.25rem 0.625rem;
  color: var(--image-muted);
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.2rem;
}

.image-chip-active {
  border-color: color-mix(in srgb, var(--image-accent) 36%, transparent);
  background: var(--image-accent-soft);
  color: var(--image-accent);
}

.image-status-pill {
  display: inline-flex;
  width: fit-content;
  flex-shrink: 0;
  align-items: center;
  border-radius: 9999px;
  padding: 0.25rem 0.625rem;
  font-size: 0.75rem;
  font-weight: 650;
  line-height: 1rem;
}

.image-status-success {
  background: rgba(16, 185, 129, 0.12);
  color: #047857;
}

.image-status-error {
  background: rgba(239, 68, 68, 0.12);
  color: #b91c1c;
}

.image-status-loading {
  background: var(--image-accent-soft);
  color: var(--image-accent);
}

:global(.dark) .image-status-success {
  color: #6ee7b7;
}

:global(.dark) .image-status-error {
  color: #fca5a5;
}

.image-size-option {
  min-width: 0;
  height: 2.75rem;
  border-radius: 8px;
  border: 1px solid;
  padding: 0 0.625rem;
  color: var(--image-text);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.1rem;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.image-size-option-active {
  border-color: color-mix(in srgb, var(--image-accent) 54%, transparent);
  background: var(--image-accent-soft);
  color: var(--image-accent);
}

.image-size-option-idle {
  border-color: var(--image-border-soft);
  background: var(--image-surface-elevated);
}

.image-size-option-idle:hover {
  border-color: var(--image-border);
  background: var(--image-surface-muted);
}

.image-price-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--image-border-soft);
  border-radius: 8px;
}

.image-price-cell {
  min-width: 0;
  padding: 0.875rem;
  border-bottom: 1px solid var(--image-border-soft);
}

.image-price-cell:last-child {
  border-bottom: 0;
}

.image-price-label,
.image-price-subvalue {
  color: var(--image-muted);
  font-size: 0.75rem;
  line-height: 1.35;
}

.image-price-value {
  margin-top: 0.25rem;
  overflow-wrap: anywhere;
  color: var(--image-text);
  font-size: 1.0625rem;
  font-weight: 650;
  line-height: 1.35;
}

.image-price-subvalue {
  margin-top: 0.125rem;
}

@media (min-width: 640px) {
  .image-price-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .image-price-cell:nth-child(odd) {
    border-right: 1px solid var(--image-border-soft);
  }

  .image-price-cell:nth-last-child(-n + 2) {
    border-bottom: 0;
  }
}

.image-prompt-frame {
  overflow: hidden;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--image-surface-elevated);
  transition: background-color 150ms ease, border-color 150ms ease;
}

.image-prompt-frame-dragging {
  border-color: color-mix(in srgb, var(--image-accent) 52%, transparent);
  background: var(--image-accent-soft);
}

.image-prompt-input {
  min-height: 13rem;
  width: 100%;
  resize: none;
  border: 0;
  border-radius: 8px 8px 0 0;
  background: transparent;
  padding: 1rem;
  color: var(--image-text);
  font-size: 0.9375rem;
  line-height: 1.75;
}

.image-prompt-input::placeholder {
  color: var(--image-muted-soft);
}

.image-prompt-input:focus {
  outline: none;
  box-shadow: none;
}

@media (min-width: 640px) {
  .image-prompt-input {
    min-height: 15rem;
    padding: 1.125rem;
  }
}

.image-note {
  border: 1px solid var(--image-border-soft);
  border-radius: 8px;
  background: var(--image-surface);
  padding: 0.875rem 1rem;
}

.image-empty {
  border: 1px dashed var(--image-border);
  border-radius: 8px;
  background: var(--image-surface);
  padding: 2.5rem 1rem;
  text-align: center;
}

.image-result-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0.75rem;
}

@media (min-width: 640px) {
  .image-result-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .image-result-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

.image-overlay-button {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.375rem;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.72);
  padding: 0.375rem 0.625rem;
  color: #fff;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.1rem;
  backdrop-filter: blur(10px);
}

@supports not (color: color-mix(in srgb, white, black)) {
  .image-soft-button:hover:not(:disabled) {
    border-color: var(--image-accent);
  }

  .image-chip-active,
  .image-size-option-active,
  .image-prompt-frame-dragging {
    border-color: var(--image-accent);
  }
}
</style>
