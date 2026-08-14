<template>
  <div class="card">
    <div class="card-header">
      <h2 class="text-lg font-semibold text-[var(--apple-text)]">
        {{ t('profile.totp.title') }}
      </h2>
      <p class="mt-1 text-sm text-[var(--apple-muted)]">
        {{ t('profile.totp.description') }}
      </p>
    </div>
    <div class="px-6 py-6">
      <div v-if="loading" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="lg" class="animate-spin text-[var(--apple-blue)]" />
      </div>

      <div v-else-if="status && !status.feature_enabled" class="flex items-center gap-4 py-4">
        <div class="flex-shrink-0 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-3 text-[var(--apple-muted)]">
          <Icon name="exclamationTriangle" size="lg" />
        </div>
        <div class="min-w-0">
          <p class="font-medium text-[var(--apple-text)]">
            {{ t('profile.totp.featureDisabled') }}
          </p>
          <p class="text-sm text-[var(--apple-muted)]">
            {{ t('profile.totp.featureDisabledHint') }}
          </p>
        </div>
      </div>

      <div v-else-if="status?.enabled" class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex min-w-0 items-center gap-4">
          <div class="flex-shrink-0 rounded-lg border border-[color:color-mix(in_srgb,var(--apple-success)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_10%,var(--apple-surface))] p-3 text-[var(--apple-success)]">
            <Icon name="shield" size="lg" />
          </div>
          <div class="min-w-0">
            <p class="font-medium text-[var(--apple-text)]">
              {{ t('profile.totp.enabled') }}
            </p>
            <p v-if="status.enabled_at" class="text-sm text-[var(--apple-muted)]">
              {{ t('profile.totp.enabledAt') }}: {{ formatDate(status.enabled_at) }}
            </p>
          </div>
        </div>
        <button
          type="button"
          class="btn btn-secondary w-full text-[var(--apple-danger)] hover:text-[var(--apple-danger)] sm:w-auto"
          @click="showDisableDialog = true"
        >
          {{ t('profile.totp.disable') }}
        </button>
      </div>

      <div v-else class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex min-w-0 items-center gap-4">
          <div class="flex-shrink-0 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-3 text-[var(--apple-muted)]">
            <Icon name="shield" size="lg" />
          </div>
          <div class="min-w-0">
            <p class="font-medium text-[var(--apple-text)]">
              {{ t('profile.totp.notEnabled') }}
            </p>
            <p class="text-sm text-[var(--apple-muted)]">
              {{ t('profile.totp.notEnabledHint') }}
            </p>
          </div>
        </div>
        <button
          type="button"
          class="btn btn-primary w-full sm:w-auto"
          @click="showSetupModal = true"
        >
          {{ t('profile.totp.enable') }}
        </button>
      </div>
    </div>

    <!-- Setup Modal -->
    <TotpSetupModal
      v-if="showSetupModal"
      @close="showSetupModal = false"
      @success="handleSetupSuccess"
    />

    <!-- Disable Dialog -->
    <TotpDisableDialog
      v-if="showDisableDialog"
      @close="showDisableDialog = false"
      @success="handleDisableSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { totpAPI } from '@/api'
import type { TotpStatus } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import TotpSetupModal from './WegooTotpSetupModal.vue'
import TotpDisableDialog from './WegooTotpDisableDialog.vue'

defineOptions({ name: 'ProfileTotpCard' })

const { t } = useI18n()

const loading = ref(true)
const status = ref<TotpStatus | null>(null)
const showSetupModal = ref(false)
const showDisableDialog = ref(false)

const loadStatus = async () => {
  loading.value = true
  try {
    status.value = await totpAPI.getStatus()
  } catch (error) {
    console.error('Failed to load TOTP status:', error)
  } finally {
    loading.value = false
  }
}

const handleSetupSuccess = () => {
  showSetupModal.value = false
  loadStatus()
}

const handleDisableSuccess = () => {
  showDisableDialog.value = false
  loadStatus()
}

const formatDate = (timestamp: number) => {
  // Backend returns Unix timestamp in seconds, convert to milliseconds
  const date = new Date(timestamp * 1000)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  loadStatus()
})
</script>
