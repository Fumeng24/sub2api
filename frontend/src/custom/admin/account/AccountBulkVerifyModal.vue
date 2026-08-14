<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.bulkVerify.title')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.bulkVerify.description') }}
      </p>

      <!-- Start form -->
      <div v-if="!job" class="space-y-3 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700">
        <div class="flex items-center gap-3">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-200 min-w-24">
            {{ t('admin.accounts.bulkVerify.concurrency') }}
          </label>
          <input
            v-model.number="concurrency"
            type="number"
            min="1"
            max="128"
            class="w-32 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800 dark:text-white"
          />
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.bulkVerify.concurrencyHint') }}
          </span>
        </div>
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.bulkVerify.scopeHint') }}
        </div>
        <div class="text-xs text-amber-600 dark:text-amber-400">
          {{ t('admin.accounts.bulkVerify.multiInstanceWarning') }}
        </div>
      </div>

      <!-- Progress panel -->
      <div v-else class="space-y-3">
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-2 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Icon v-if="job.running" name="refresh" size="sm" class="animate-spin text-primary-500" />
              <Icon v-else-if="job.cancelled" name="x" size="sm" class="text-gray-500" />
              <Icon v-else name="check" size="sm" class="text-emerald-500" />
              <span class="text-sm font-medium text-gray-900 dark:text-white">
                {{ statusLabel }}
              </span>
            </div>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ job.done }} / {{ job.total }}
            </span>
          </div>

          <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
            <div
              class="h-full bg-primary-500 transition-all duration-300"
              :style="{ width: progressPct + '%' }"
            ></div>
          </div>

          <div class="mt-3 grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
            <div class="flex items-center justify-between rounded bg-emerald-50 px-2 py-1 dark:bg-emerald-900/20">
              <span class="text-emerald-700 dark:text-emerald-300">{{ t('admin.accounts.bulkVerify.high') }}</span>
              <span class="font-medium text-emerald-800 dark:text-emerald-200">{{ job.high }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-yellow-50 px-2 py-1 dark:bg-yellow-900/20">
              <span class="text-yellow-700 dark:text-yellow-300">{{ t('admin.accounts.bulkVerify.medium') }}</span>
              <span class="font-medium text-yellow-800 dark:text-yellow-200">{{ job.medium }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-orange-50 px-2 py-1 dark:bg-orange-900/20">
              <span class="text-orange-700 dark:text-orange-300">{{ t('admin.accounts.bulkVerify.low') }}</span>
              <span class="font-medium text-orange-800 dark:text-orange-200">{{ job.low }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-red-50 px-2 py-1 dark:bg-red-900/20">
              <span class="text-red-700 dark:text-red-300">{{ t('admin.accounts.bulkVerify.exhausted') }}</span>
              <span class="font-medium text-red-800 dark:text-red-200">{{ job.exhausted }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-gray-100 px-2 py-1 dark:bg-dark-600">
              <span class="text-gray-600 dark:text-gray-400">{{ t('admin.accounts.bulkVerify.expired') }}</span>
              <span class="font-medium text-gray-800 dark:text-gray-200">{{ job.expired }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-gray-100 px-2 py-1 dark:bg-dark-600">
              <span class="text-gray-600 dark:text-gray-400">{{ t('admin.accounts.bulkVerify.missing') }}</span>
              <span class="font-medium text-gray-800 dark:text-gray-200">{{ job.missing }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-gray-100 px-2 py-1 dark:bg-dark-600">
              <span class="text-gray-600 dark:text-gray-400">{{ t('admin.accounts.bulkVerify.unknown') }}</span>
              <span class="font-medium text-gray-800 dark:text-gray-200">{{ job.unknown }}</span>
            </div>
            <div class="flex items-center justify-between rounded bg-red-50 px-2 py-1 dark:bg-red-900/20">
              <span class="text-red-700 dark:text-red-300">{{ t('admin.accounts.bulkVerify.error') }}</span>
              <span class="font-medium text-red-800 dark:text-red-200">{{ job.error }}</span>
            </div>
          </div>

          <div class="mt-3 text-xs text-gray-500 dark:text-gray-400">
            <div>{{ t('admin.accounts.bulkVerify.startedAt') }}: {{ formatTs(job.started_at) }}</div>
            <div v-if="job.finished_at">{{ t('admin.accounts.bulkVerify.finishedAt') }}: {{ formatTs(job.finished_at) }}</div>
          </div>
        </div>

        <!-- Apply panel: only shown after the job finishes -->
        <div
          v-if="!job.running && (job.refreshable_expired > 0 || job.markable_exhausted > 0)"
          class="space-y-3 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700"
        >
          <div class="text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.accounts.bulkVerify.applyTitle') }}
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.bulkVerify.applyDescription') }}
          </p>

          <label
            v-if="job.refreshable_expired > 0"
            class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200"
          >
            <input
              v-model="applyRefreshExpired"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>
              {{ t('admin.accounts.bulkVerify.applyRefresh', { n: job.refreshable_expired }) }}
            </span>
          </label>

          <label
            v-if="job.markable_exhausted > 0"
            class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200"
          >
            <input
              v-model="applyMarkExhausted"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>
              {{ t('admin.accounts.bulkVerify.applyMarkExhausted', { n: job.markable_exhausted }) }}
            </span>
          </label>

          <label class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <input
              v-model="applyDryRun"
              type="checkbox"
              class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>{{ t('admin.accounts.bulkVerify.applyDryRun') }}</span>
          </label>

          <button
            @click="handleApply"
            :disabled="applying || (!applyRefreshExpired && !applyMarkExhausted)"
            class="btn btn-primary btn-sm"
          >
            <Icon v-if="applying" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ applyDryRun ? t('admin.accounts.bulkVerify.applyDryRunBtn') : t('admin.accounts.bulkVerify.applyBtn') }}
          </button>
        </div>

        <!-- Apply result -->
        <div
          v-if="applyResult"
          class="space-y-2 rounded-lg border border-gray-200 p-4 text-sm dark:border-dark-600"
        >
          <div class="font-medium text-gray-900 dark:text-white">
            {{ applyResult.dry_run ? t('admin.accounts.bulkVerify.applyResultDryRun') : t('admin.accounts.bulkVerify.applyResult') }}
          </div>
          <div class="grid grid-cols-2 gap-2 text-xs sm:grid-cols-3">
            <div class="flex justify-between rounded bg-emerald-50 px-2 py-1 dark:bg-emerald-900/20">
              <span class="text-emerald-700 dark:text-emerald-300">{{ t('admin.accounts.bulkVerify.applyRefreshed') }}</span>
              <span class="font-medium text-emerald-800 dark:text-emerald-200">{{ applyResult.refresh_succeeded }}</span>
            </div>
            <div class="flex justify-between rounded bg-red-50 px-2 py-1 dark:bg-red-900/20">
              <span class="text-red-700 dark:text-red-300">{{ t('admin.accounts.bulkVerify.applyRefreshFailed') }}</span>
              <span class="font-medium text-red-800 dark:text-red-200">{{ applyResult.refresh_failed }}</span>
            </div>
            <div class="flex justify-between rounded bg-emerald-50 px-2 py-1 dark:bg-emerald-900/20">
              <span class="text-emerald-700 dark:text-emerald-300">{{ t('admin.accounts.bulkVerify.applyMarked') }}</span>
              <span class="font-medium text-emerald-800 dark:text-emerald-200">{{ applyResult.marked_rate_limited }}</span>
            </div>
            <div class="flex justify-between rounded bg-red-50 px-2 py-1 dark:bg-red-900/20">
              <span class="text-red-700 dark:text-red-300">{{ t('admin.accounts.bulkVerify.applyMarkFailed') }}</span>
              <span class="font-medium text-red-800 dark:text-red-200">{{ applyResult.mark_rate_limit_failed }}</span>
            </div>
            <div v-if="applyResult.skipped_no_reset_at > 0" class="flex justify-between rounded bg-gray-100 px-2 py-1 dark:bg-dark-600">
              <span class="text-gray-600 dark:text-gray-400">{{ t('admin.accounts.bulkVerify.applySkippedNoReset') }}</span>
              <span class="font-medium text-gray-800 dark:text-gray-200">{{ applyResult.skipped_no_reset_at }}</span>
            </div>
          </div>
          <details v-if="applyResult.failures && applyResult.failures.length" class="text-xs text-gray-600 dark:text-gray-400">
            <summary class="cursor-pointer">{{ t('admin.accounts.bulkVerify.applyFailuresTitle', { n: applyResult.failures.length }) }}</summary>
            <ul class="mt-2 max-h-48 space-y-1 overflow-auto pl-4">
              <li v-for="f in applyResult.failures" :key="`${f.account_id}-${f.action}`" class="list-disc">
                #{{ f.account_id }} · {{ f.action }} · {{ f.error }}
              </li>
            </ul>
          </details>
        </div>
      </div>

      <div v-if="errorMsg" class="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMsg }}
      </div>
    </div>

    <template #footer>
      <button v-if="!job" @click="handleStart" :disabled="starting" class="btn btn-primary">
        <Icon v-if="starting" name="refresh" size="sm" class="mr-1 animate-spin" />
        {{ t('admin.accounts.bulkVerify.start') }}
      </button>
      <button v-else-if="job.running" @click="handleCancel" :disabled="cancelling" class="btn btn-danger">
        {{ t('admin.accounts.bulkVerify.cancel') }}
      </button>
      <button v-else @click="handleReset" class="btn btn-secondary">
        {{ t('admin.accounts.bulkVerify.reset') }}
      </button>
      <button @click="handleClose" class="btn btn-secondary">
        {{ t('common.close') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import bulkVerifyAPI, {
  BULK_VERIFY_JOB_ALREADY_RUNNING,
  type BulkVerifyJob,
  type BulkVerifyApplyResult
} from '@/custom/api/admin/bulkVerify'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'finished', job: BulkVerifyJob): void }>()

const { t } = useI18n()

const concurrency = ref(32)
const job = ref<BulkVerifyJob | null>(null)
const starting = ref(false)
const cancelling = ref(false)
const errorMsg = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const applyRefreshExpired = ref(true)
const applyMarkExhausted = ref(true)
const applyDryRun = ref(false)
const applying = ref(false)
const applyResult = ref<BulkVerifyApplyResult | null>(null)

const progressPct = computed(() => {
  if (!job.value || job.value.total === 0) return 0
  return Math.min(100, Math.round((job.value.done / job.value.total) * 100))
})

const statusLabel = computed(() => {
  if (!job.value) return ''
  if (job.value.cancelled) return t('admin.accounts.bulkVerify.statusCancelled')
  if (job.value.running) return t('admin.accounts.bulkVerify.statusRunning')
  return t('admin.accounts.bulkVerify.statusFinished')
})

function stopPolling() {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling(jobID: string) {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const latest = await bulkVerifyAPI.get(jobID)
      job.value = latest
      if (!latest.running) {
        stopPolling()
        emit('finished', latest)
      }
    } catch (err) {
      errorMsg.value = (err as Error).message || 'failed to poll job'
      stopPolling()
    }
  }, 1500)
}

async function handleStart() {
  errorMsg.value = ''
  starting.value = true
  try {
    const created = await bulkVerifyAPI.start({ concurrency: concurrency.value })
    job.value = created
    if (created.running) {
      startPolling(created.id)
    }
  } catch (err) {
    const e = err as { code?: string; data?: BulkVerifyJob; message?: string }
    if (e.code === BULK_VERIFY_JOB_ALREADY_RUNNING && e.data && e.data.id) {
      // Another job is already running — attach to it instead of starting a new one.
      job.value = e.data
      errorMsg.value = t('admin.accounts.bulkVerify.alreadyRunning')
      if (e.data.running) {
        startPolling(e.data.id)
      }
    } else {
      errorMsg.value = e.message || 'failed to start job'
    }
  } finally {
    starting.value = false
  }
}

async function handleCancel() {
  if (!job.value) return
  cancelling.value = true
  try {
    await bulkVerifyAPI.cancel(job.value.id)
  } catch (err) {
    errorMsg.value = (err as Error).message || 'failed to cancel'
  } finally {
    cancelling.value = false
  }
}

function handleReset() {
  stopPolling()
  job.value = null
  errorMsg.value = ''
  applyResult.value = null
}

async function handleApply() {
  if (!job.value) return
  applying.value = true
  errorMsg.value = ''
  applyResult.value = null
  try {
    const result = await bulkVerifyAPI.apply(job.value.id, {
      refresh_expired: applyRefreshExpired.value,
      mark_exhausted: applyMarkExhausted.value,
      dry_run: applyDryRun.value
    })
    applyResult.value = result
    if (!result.dry_run) {
      // Actions committed — tell the parent to refresh the accounts list.
      emit('finished', job.value)
    }
  } catch (err) {
    const e = err as { message?: string }
    errorMsg.value = e.message || 'failed to apply actions'
  } finally {
    applying.value = false
  }
}

function handleClose() {
  stopPolling()
  emit('close')
}

function formatTs(ts?: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  return Number.isNaN(d.getTime()) ? ts : d.toLocaleString()
}

watch(() => props.show, (v) => {
  if (!v) {
    stopPolling()
  }
})

onUnmounted(() => stopPolling())
</script>
