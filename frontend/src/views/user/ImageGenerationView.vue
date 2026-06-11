<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-normal text-gray-900 dark:text-white">
            {{ t('imageGeneration.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
            {{ t('imageGeneration.description') }}
          </p>
        </div>
        <button class="btn btn-secondary" :disabled="loadingPage" @click="loadInitialData">
          <Icon name="refresh" size="md" :class="loadingPage ? 'animate-spin' : ''" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="loadingPage && imageGroups.length === 0" class="flex justify-center py-16">
        <LoadingSpinner />
      </div>

      <div v-else class="grid gap-6 xl:grid-cols-[390px_minmax(0,1fr)]">
        <aside class="space-y-4">
          <section class="card p-5">
            <div class="space-y-4">
              <div>
                <label class="input-label">{{ t('imageGeneration.group') }}</label>
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
                <label class="input-label">{{ t('imageGeneration.model') }}</label>
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
                <label class="input-label">{{ t('imageGeneration.size') }}</label>
                <div class="grid grid-cols-2 gap-2">
                  <button
                    v-for="option in sizeOptions"
                    :key="option.value"
                    type="button"
                    :class="[
                      'h-11 rounded-xl border px-3 text-sm font-medium transition',
                      size === option.value
                        ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-950/30 dark:text-primary-200'
                        : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700'
                    ]"
                    @click="size = option.value"
                  >
                    {{ option.label }}
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="input-label">{{ t('imageGeneration.quality') }}</label>
                  <Select v-model="quality" :options="qualityOptions" />
                </div>
                <div>
                  <label class="input-label">{{ t('imageGeneration.count') }}</label>
                  <Select v-model="count" :options="countOptions" />
                </div>
              </div>
            </div>
          </section>

          <section class="card p-5">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('imageGeneration.pricing.title') }}
                </h2>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ pricingSummary }}
                </p>
              </div>
              <div class="rounded-xl bg-primary-50 p-2 text-primary-600 dark:bg-primary-950/40 dark:text-primary-300">
                <Icon name="calculator" size="md" />
              </div>
            </div>

            <div class="mt-4 grid grid-cols-2 gap-3">
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/60">
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageGeneration.pricing.unitCost') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ priceEstimate ? formatCredit(priceEstimate.unitCost) : t('common.notAvailable') }}
                </div>
                <div v-if="priceEstimate" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ formatCny(priceEstimate.unitCost) }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/60">
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageGeneration.pricing.batchCost') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ priceEstimate ? formatCredit(priceEstimate.batchCost) : t('common.notAvailable') }}
                </div>
                <div v-if="priceEstimate" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ formatCny(priceEstimate.batchCost) }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/60">
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageGeneration.pricing.remainingImages') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ priceEstimate ? t('imageGeneration.pricing.imageCountValue', { count: priceEstimate.remainingImages }) : t('common.notAvailable') }}
                </div>
                <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ limitSummary }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/60">
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('imageGeneration.pricing.balance') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ formatCredit(userBalance) }}
                </div>
                <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ formatCny(userBalance) }}
                </div>
              </div>
            </div>
          </section>
        </aside>

        <main class="min-w-0 space-y-4">
          <section class="card p-4 sm:p-5">
            <div
              class="rounded-2xl border border-gray-200 bg-white transition dark:border-dark-600 dark:bg-dark-900/60"
              :class="isDragging ? 'border-primary-400 bg-primary-50/50 dark:bg-primary-950/20' : ''"
              @dragenter="handleDragEnter"
              @dragover="handleDragOver"
              @dragleave="handleDragLeave"
              @drop="handleDrop"
            >
              <textarea
                v-model="prompt"
                rows="7"
                class="min-h-[180px] w-full resize-none rounded-t-2xl border-0 bg-transparent px-4 py-4 text-sm leading-6 text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-0 dark:text-white dark:placeholder:text-dark-400 sm:px-5"
                :placeholder="referenceImages.length > 0 ? t('imageGeneration.editPlaceholder') : t('imageGeneration.promptPlaceholder')"
                @paste="handlePaste"
                @keydown.ctrl.enter.prevent="submit"
                @keydown.meta.enter.prevent="submit"
              />

              <div class="border-t border-gray-100 px-4 py-3 dark:border-dark-700 sm:px-5">
                <input ref="fileInputRef" type="file" accept="image/*" multiple class="hidden" @change="handleFileInput" />
                <div v-if="referenceImages.length > 0" class="mb-3 flex flex-wrap gap-2">
                  <div v-for="image in referenceImages" :key="image.id" class="group relative h-16 w-16 overflow-hidden rounded-xl border border-gray-200 dark:border-dark-600">
                    <button type="button" class="h-full w-full" @click="previewImage = image.previewUrl">
                      <img :src="image.previewUrl" :alt="image.name" class="h-full w-full bg-gray-50 object-contain dark:bg-dark-900" />
                    </button>
                    <button
                      type="button"
                      class="absolute right-1 top-1 rounded-full bg-black/60 p-1 text-white opacity-0 transition group-hover:opacity-100"
                      :title="t('common.remove')"
                      @click="removeReference(image.id)"
                    >
                      <Icon name="x" size="xs" />
                    </button>
                  </div>
                </div>

                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="flex flex-wrap items-center gap-2">
                    <button type="button" class="btn btn-secondary btn-sm" @click="openFilePicker">
                      <Icon name="upload" size="sm" />
                      {{ referenceImages.length > 0 ? t('imageGeneration.addReference') : t('imageGeneration.uploadReference') }}
                    </button>
                    <span class="rounded-lg bg-gray-100 px-2.5 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
                      {{ modeLabel }}
                    </span>
                    <span v-if="selectedGroup" class="rounded-lg bg-gray-100 px-2.5 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
                      {{ selectedGroup.name }}
                    </span>
                    <span v-if="isDragging" class="rounded-lg bg-primary-100 px-2.5 py-1 text-xs text-primary-700 dark:bg-primary-950/40 dark:text-primary-200">
                      {{ t('imageGeneration.dropImages') }}
                    </span>
                  </div>
                  <div class="flex items-center justify-end gap-2">
                    <button v-if="generating" type="button" class="btn btn-secondary btn-sm" @click="cancelGeneration">
                      {{ t('common.cancel') }}
                    </button>
                    <button type="button" class="btn btn-primary" :disabled="!canSubmit" @click="submit">
                      <Icon v-if="!generating" name="sparkles" size="md" />
                      <Icon v-else name="refresh" size="md" class="animate-spin" />
                      {{ generating ? t('imageGeneration.generating') : submitLabel }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-900/40 dark:bg-amber-950/20">
            <div class="flex items-start gap-3">
              <div class="rounded-lg bg-amber-100 p-2 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200">
                <Icon name="infoCircle" size="sm" />
              </div>
              <p class="pt-0.5 text-sm text-amber-800 dark:text-amber-100">
                {{ t('imageGeneration.localCacheWarning') }}
              </p>
            </div>
          </section>

          <section v-if="runs.length === 0" class="rounded-2xl border border-dashed border-gray-200 p-10 text-center dark:border-dark-700">
            <Icon name="sparkles" size="xl" class="mx-auto text-gray-300 dark:text-dark-500" />
            <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('imageGeneration.empty') }}</p>
          </section>

          <section v-else class="space-y-4">
            <article v-for="run in runs" :key="run.id" class="card overflow-hidden">
              <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:px-5">
                <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <p class="line-clamp-2 text-sm font-medium text-gray-900 dark:text-white">{{ run.prompt }}</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                      {{ run.groupName }} · {{ run.model }} · {{ run.sizeLabel }} · {{ t('imageGeneration.pricing.imageCountValue', { count: run.requestedCount }) }}
                    </p>
                  </div>
                  <span
                    :class="[
                      'inline-flex w-fit items-center rounded-full px-2.5 py-1 text-xs font-medium',
                      run.status === 'success' ? 'bg-green-100 text-green-700 dark:bg-green-950/40 dark:text-green-300' :
                      run.status === 'error' ? 'bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-300' :
                      'bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
                    ]"
                  >
                    {{ statusLabel(run.status) }}
                  </span>
                </div>
              </div>

              <div class="p-4 sm:p-5">
                <div v-if="run.status === 'loading'" class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  <div v-for="index in run.requestedCount" :key="index" class="aspect-square animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-700" />
                </div>
                <div v-else-if="run.status === 'error'" class="rounded-xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-200">
                  {{ run.error }}
                </div>
                <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  <div v-for="image in run.images" :key="image.id" class="group relative overflow-hidden rounded-2xl border border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-900">
                    <button type="button" class="block aspect-square w-full" @click="previewImage = image.src">
                      <img :src="image.src" :alt="run.prompt" class="h-full w-full object-contain" />
                    </button>
                    <div class="absolute inset-x-2 bottom-2 flex justify-end gap-2 opacity-0 transition group-hover:opacity-100">
                      <button type="button" class="rounded-lg bg-black/70 px-3 py-1.5 text-xs font-medium text-white" @click="continueEditFromResult(image)">
                        {{ t('imageGeneration.continueEdit') }}
                      </button>
                      <button type="button" class="rounded-lg bg-black/70 px-3 py-1.5 text-xs font-medium text-white" @click="downloadImage(image, run)">
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
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { keysAPI } from '@/api/keys'
import userGroupsAPI from '@/api/groups'
import {
  MAX_IMAGE_GENERATION_COUNT,
  normalizeOpenAIImageResults,
  submitGeminiImageGatewayRequest,
  submitImageGatewayRequest,
  type OpenAIImageResult,
} from '@/api/imageGeneration'
import { useAppStore, useAuthStore } from '@/stores'
import type { ApiKey, Group, PublicSettings } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  DEFAULT_IMAGE_MODEL,
  isImageGenerationGroup,
  resolveSupportedImageModels,
} from '@/utils/imageGenerationGroups'

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

const sizeOptions = [
  { value: '1024x1024', label: '1:1 · 1K', tier: '1K' },
  { value: '1024x1536', label: '2:3 · 1K', tier: '1K' },
  { value: '1536x1024', label: '3:2 · 1K', tier: '1K' },
  { value: '1024x1365', label: '3:4 · 1K', tier: '1K' },
  { value: '1365x1024', label: '4:3 · 1K', tier: '1K' },
  { value: '1088x1920', label: '9:16 · 1K', tier: '1K' },
  { value: '1920x1088', label: '16:9 · 1K', tier: '1K' },
  { value: 'auto', label: 'Auto · 1K', tier: '1K' },
] as const

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

const selectedSize = computed(() => sizeOptions.find((option) => option.value === size.value) || sizeOptions[0])
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

const modeLabel = computed(() => referenceImages.value.length > 0
  ? t('imageGeneration.editMode')
  : t('imageGeneration.generateMode'))

const submitLabel = computed(() => referenceImages.value.length > 0
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
  return `自动生图 · ${group.name}`
}

function hasActiveKeyForGroup(keys: ApiKey[], groupId: number) {
  return keys.some((key) => key.status === 'active' && key.group_id === groupId)
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
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  isDragging.value = true
}

function handleDragOver(event: DragEvent) {
  if (!hasDraggedImages(event.dataTransfer)) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  isDragging.value = true
}

function handleDragLeave(event: DragEvent) {
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
      prompt: run.prompt,
      model,
      count: requestedCount,
      size: selectedSize.value.value,
      quality: quality.value,
      referenceImages: referenceImages.value.map((image) => image.file),
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
  scheduleReferenceImagesPersistence()
}, { deep: true })

watch(selectedGroup, (group) => {
  if (loadingPage.value || !group) {
    return
  }
  syncSelectedModel()
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
