<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.bulkModelTest.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <div class="space-y-5">
      <div class="rounded-lg border border-blue-100 bg-blue-50/80 p-4 dark:border-blue-900/40 dark:bg-blue-950/30">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div class="text-sm font-semibold text-blue-900 dark:text-blue-100">
              {{ t('admin.accounts.bulkModelTest.selectedAccounts', { count: accounts.length }) }}
            </div>
            <div class="mt-1 max-h-16 overflow-y-auto text-xs text-blue-700 dark:text-blue-200">
              {{ accountSummary }}
            </div>
          </div>
          <button
            class="btn btn-secondary btn-sm"
            :disabled="loadingModels || accounts.length === 0"
            @click="loadModels"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loadingModels }" />
            {{ t('admin.accounts.bulkModelTest.reloadModels') }}
          </button>
        </div>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]">
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="text-sm font-semibold text-gray-800 dark:text-gray-100">
              {{ t('admin.accounts.bulkModelTest.modelChecklist') }}
            </label>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.bulkModelTest.modelCount', { count: modelOptions.length }) }}
            </span>
          </div>

          <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
            <div class="mb-3 flex gap-2">
              <input
                v-model="modelSearch"
                type="search"
                class="input h-9 text-sm"
                :placeholder="t('admin.accounts.bulkModelTest.searchModels')"
              />
              <button class="btn btn-secondary btn-sm shrink-0" @click="selectVisibleModels">
                {{ t('admin.accounts.bulkModelTest.selectVisible') }}
              </button>
              <button class="btn btn-secondary btn-sm shrink-0" @click="selectedModelIds = []">
                {{ t('admin.accounts.bulkModelTest.clearModels') }}
              </button>
            </div>

            <div
              v-if="loadingModels"
              class="flex min-h-32 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
            >
              <Icon name="refresh" size="sm" class="mr-2 animate-spin" />
              {{ t('admin.accounts.bulkModelTest.loadingModels') }}
            </div>
            <div
              v-else-if="filteredModelOptions.length === 0"
              class="flex min-h-32 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
            >
              {{ t('admin.accounts.bulkModelTest.noModels') }}
            </div>
            <div v-else class="grid max-h-60 gap-2 overflow-y-auto pr-1 sm:grid-cols-2 xl:grid-cols-3">
              <label
                v-for="model in filteredModelOptions"
                :key="model.id"
                class="flex cursor-pointer items-start gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm transition hover:border-primary-300 hover:bg-primary-50 dark:border-dark-600 dark:bg-dark-700/70 dark:hover:border-primary-500/60 dark:hover:bg-primary-900/20"
              >
                <input
                  type="checkbox"
                  class="mt-0.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                  :checked="selectedModelIds.includes(model.id)"
                  @change="toggleModel(model.id)"
                />
                <span class="min-w-0">
                  <span class="block truncate font-medium text-gray-900 dark:text-gray-100" :title="model.id">
                    {{ model.displayName }}
                  </span>
                  <span class="block truncate font-mono text-xs text-gray-500 dark:text-gray-400" :title="model.id">
                    {{ model.id }}
                  </span>
                </span>
              </label>
            </div>
          </div>
        </div>

        <div class="space-y-4">
          <TextArea
            v-model="manualModels"
            :label="t('admin.accounts.bulkModelTest.manualModels')"
            :placeholder="t('admin.accounts.bulkModelTest.manualModelsPlaceholder')"
            :hint="t('admin.accounts.bulkModelTest.manualModelsHint')"
            rows="5"
            :disabled="testing"
          />

          <TextArea
            v-model="prompt"
            :label="t('admin.accounts.bulkModelTest.prompt')"
            :placeholder="t('admin.accounts.bulkModelTest.promptPlaceholder')"
            :hint="t('admin.accounts.bulkModelTest.promptHint')"
            rows="3"
            :disabled="testing"
          />

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="space-y-1.5">
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.accounts.bulkModelTest.mode') }}
              </span>
              <select v-model="mode" class="input" :disabled="testing">
                <option value="default">{{ t('admin.accounts.bulkModelTest.modeDefault') }}</option>
                <option value="compact">{{ t('admin.accounts.bulkModelTest.modeCompact') }}</option>
              </select>
            </label>
            <label class="space-y-1.5">
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.accounts.bulkModelTest.concurrency') }}
              </span>
              <input
                v-model.number="concurrency"
                type="number"
                min="1"
                max="20"
                class="input"
                :disabled="testing"
              />
            </label>
          </div>
        </div>
      </div>

      <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800/80">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ t('admin.accounts.bulkModelTest.resultSummary', { total: summary.total, success: summary.success, failed: summary.failed }) }}
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.bulkModelTest.taskPreview', { accounts: accounts.length, models: selectedModels.length, total: taskTotal }) }}
          </div>
        </div>

        <div v-if="results.length > 0" class="mt-3 max-h-80 overflow-auto rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-900">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
            <thead class="sticky top-0 bg-gray-100 text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-700 dark:text-gray-300">
              <tr>
                <th class="px-3 py-2 text-left">{{ t('admin.accounts.bulkModelTest.accountColumn') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.accounts.bulkModelTest.modelColumn') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.accounts.bulkModelTest.statusColumn') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.accounts.bulkModelTest.latencyColumn') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.accounts.bulkModelTest.messageColumn') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="(result, index) in enrichedResults" :key="`${result.account_id}-${result.model_id}-${index}`">
                <td class="whitespace-nowrap px-3 py-2 font-medium text-gray-900 dark:text-gray-100">
                  {{ result.accountName }}
                </td>
                <td class="whitespace-nowrap px-3 py-2 font-mono text-xs text-gray-600 dark:text-gray-300">
                  {{ result.model_id }}
                </td>
                <td class="whitespace-nowrap px-3 py-2">
                  <span
                    :class="[
                      'rounded-full px-2 py-0.5 text-xs font-semibold',
                      result.success
                        ? 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-300'
                        : 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300'
                    ]"
                  >
                    {{ result.success ? t('admin.accounts.bulkModelTest.success') : t('admin.accounts.bulkModelTest.failed') }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-3 py-2 text-gray-600 dark:text-gray-300">
                  {{ formatLatency(result.latency_ms) }}
                </td>
                <td class="max-w-[28rem] px-3 py-2 text-gray-600 dark:text-gray-300">
                  <div class="line-clamp-3 break-words" :title="result.message">
                    {{ result.message || '-' }}
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex flex-wrap justify-end gap-3">
        <button class="btn btn-secondary" :disabled="testing" @click="handleClose">
          {{ t('common.close') }}
        </button>
        <button class="btn btn-primary" :disabled="testing || !canStart" @click="startBulkTest">
          <Icon v-if="testing" name="refresh" size="sm" class="animate-spin" />
          {{ testing ? t('admin.accounts.bulkModelTest.testing') : t('admin.accounts.bulkModelTest.start') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/custom/api/admin'
import { useAppStore } from '@/stores/app'
import type { Account, ClaudeModel } from '@/types'
import type { BulkTestModelResult } from '@/custom/api/admin/accounts'

interface ModelOption {
  id: string
  displayName: string
}

const props = defineProps<{
  show: boolean
  accounts: Account[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loadingModels = ref(false)
const testing = ref(false)
const modelOptions = ref<ModelOption[]>([])
const selectedModelIds = ref<string[]>([])
const modelSearch = ref('')
const manualModels = ref('')
const prompt = ref('')
const mode = ref<'default' | 'compact'>('default')
const concurrency = ref(6)
const results = ref<BulkTestModelResult[]>([])
const summary = ref({ total: 0, success: 0, failed: 0 })
const fallbackTestModelsByPlatform: Record<string, string[]> = {
  openai: ['codex-auto-review', 'gpt-5.3-codex-spark', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5'],
  anthropic: ['claude-fable-5', 'claude-sonnet-4-6', 'claude-opus-4-8', 'claude-opus-4-7', 'claude-opus-4-6', 'claude-haiku-4-5-20251001'],
  gemini: ['gemini-3.1-flash-image', 'gemini-3.5-flash', 'gemini-2.5-flash', 'gemini-2.5-pro', 'gemini-2.0-flash'],
  antigravity: ['claude-fable-5', 'claude-sonnet-4-6', 'gemini-3.1-flash-image', 'gemini-3.1-pro-high', 'gemini-3-flash']
}

const accountNameByID = computed(() => new Map(props.accounts.map(account => [account.id, account.name])))
const accountSummary = computed(() => props.accounts.map(account => `#${account.id} ${account.name}`).join('，'))

const manualModelIds = computed(() => {
  const seen = new Set<string>()
  const parsed: string[] = []
  for (const raw of manualModels.value.split(/[\n,，\s]+/)) {
    const modelID = raw.trim()
    if (!modelID || seen.has(modelID)) continue
    seen.add(modelID)
    parsed.push(modelID)
  }
  return parsed
})

const selectedModels = computed(() => {
  const seen = new Set<string>()
  const merged: string[] = []
  for (const modelID of [...selectedModelIds.value, ...manualModelIds.value]) {
    if (!modelID || seen.has(modelID)) continue
    seen.add(modelID)
    merged.push(modelID)
  }
  return merged
})

const filteredModelOptions = computed(() => {
  const keyword = modelSearch.value.trim().toLowerCase()
  if (!keyword) return modelOptions.value
  return modelOptions.value.filter(model => model.id.toLowerCase().includes(keyword) || model.displayName.toLowerCase().includes(keyword))
})

const taskTotal = computed(() => props.accounts.length * selectedModels.value.length)
const canStart = computed(() => props.accounts.length > 0 && selectedModels.value.length > 0 && taskTotal.value <= 500)

const enrichedResults = computed(() => results.value.map(result => ({
  ...result,
  accountName: accountNameByID.value.get(result.account_id) || `#${result.account_id}`
})))

watch(
  () => props.show,
  (show) => {
    if (!show) return
    resetForm()
    void loadModels()
  }
)

const resetForm = () => {
  modelOptions.value = []
  selectedModelIds.value = []
  modelSearch.value = ''
  manualModels.value = ''
  prompt.value = ''
  mode.value = 'default'
  concurrency.value = 6
  results.value = []
  summary.value = { total: 0, success: 0, failed: 0 }
}

const fallbackModelOptions = (platform: string): ModelOption[] => {
  const models = fallbackTestModelsByPlatform[platform] || fallbackTestModelsByPlatform.anthropic
  return models.map(id => ({ id, displayName: id }))
}

const normalizeModelOptions = (models: ClaudeModel[], platform: string) => {
  const seen = new Set<string>()
  const options: ModelOption[] = []
  for (const model of models) {
    const id = model.id?.trim()
    if (!id || seen.has(id)) continue
    seen.add(id)
    options.push({
      id,
      displayName: model.display_name || id
    })
  }
  return options.length > 0 ? options : fallbackModelOptions(platform)
}

const loadModels = async () => {
  if (props.accounts.length === 0) return
  loadingModels.value = true
  try {
    const collected = new Map<string, ModelOption>()
    const queue = [...props.accounts]
    const workers = Array.from({ length: Math.min(6, queue.length) }, async () => {
      while (queue.length > 0) {
        const account = queue.shift()
        if (!account) continue
        try {
          const models = normalizeModelOptions(await adminAPI.accounts.getAvailableModels(account.id), account.platform)
          for (const model of models) {
            if (!collected.has(model.id)) {
              collected.set(model.id, model)
            }
          }
        } catch (error) {
          console.warn('Failed to load account models:', account.id, error)
          for (const model of fallbackModelOptions(account.platform)) {
            if (!collected.has(model.id)) {
              collected.set(model.id, model)
            }
          }
        }
      }
    })
    await Promise.all(workers)
    modelOptions.value = Array.from(collected.values()).sort((a, b) => a.id.localeCompare(b.id))
  } finally {
    loadingModels.value = false
  }
}

const toggleModel = (modelID: string) => {
  if (selectedModelIds.value.includes(modelID)) {
    selectedModelIds.value = selectedModelIds.value.filter(id => id !== modelID)
  } else {
    selectedModelIds.value = [...selectedModelIds.value, modelID]
  }
}

const selectVisibleModels = () => {
  const seen = new Set(selectedModelIds.value)
  const next = [...selectedModelIds.value]
  for (const model of filteredModelOptions.value) {
    if (seen.has(model.id)) continue
    seen.add(model.id)
    next.push(model.id)
  }
  selectedModelIds.value = next
}

const startBulkTest = async () => {
  if (!canStart.value || testing.value) return
  testing.value = true
  results.value = []
  summary.value = { total: taskTotal.value, success: 0, failed: 0 }
  try {
    const response = await adminAPI.accounts.bulkTestModels({
      account_ids: props.accounts.map(account => account.id),
      model_ids: selectedModels.value,
      prompt: prompt.value.trim(),
      mode: mode.value,
      concurrency: concurrency.value
    })
    results.value = response.results
    summary.value = {
      total: response.total,
      success: response.success,
      failed: response.failed
    }
    if (response.failed > 0) {
      appStore.showError(t('admin.accounts.bulkModelTest.partialDone', { success: response.success, failed: response.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkModelTest.allDone', { count: response.success }))
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.bulkModelTest.failedToRun'))
  } finally {
    testing.value = false
  }
}

const formatLatency = (latencyMs: number) => {
  if (!latencyMs || latencyMs < 0) return '-'
  if (latencyMs >= 1000) return `${(latencyMs / 1000).toFixed(1)}s`
  return `${latencyMs}ms`
}

const handleClose = () => {
  if (testing.value) return
  emit('close')
}
</script>
