<template>
  <div class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.quickActions') }}</h2>
      <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">
        {{ t('dashboard.quickActionsHint') }}
      </p>
    </div>
    <div class="space-y-3 p-4">
      <div class="rounded-lg border border-blue-100 bg-blue-50/60 p-4 dark:border-blue-900/40 dark:bg-blue-950/20">
        <div class="flex items-start gap-3">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-600 text-white shadow-sm shadow-blue-500/20">
            <Icon name="bolt" size="md" />
          </div>
          <div>
            <p class="text-sm font-semibold text-blue-950 dark:text-blue-100">
              {{ t('dashboard.firstRun.title') }}
            </p>
            <p class="mt-1 text-xs leading-5 text-blue-800 dark:text-blue-200">
              {{ t('dashboard.firstRun.description') }}
            </p>
          </div>
        </div>
        <div class="mt-3 grid gap-2">
          <div
            v-for="(step, index) in firstRunSteps"
            :key="step"
            class="flex items-center gap-2 rounded-md bg-white px-3 py-2 text-xs font-medium text-gray-700 ring-1 ring-blue-100/70 dark:bg-dark-900/45 dark:text-dark-200 dark:ring-blue-900/30"
          >
            <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-blue-100 text-[10px] font-bold text-blue-700 dark:bg-blue-900/50 dark:text-blue-200">
              {{ index + 1 }}
            </span>
            <span>{{ step }}</span>
          </div>
        </div>
      </div>

      <button @click="router.push('/keys')" class="group flex w-full items-center gap-3 rounded-lg border border-gray-100 bg-gray-50 p-3 text-left transition-colors hover:border-blue-200 hover:bg-blue-50/50 dark:border-dark-700 dark:bg-dark-900/40 dark:hover:border-blue-900/50 dark:hover:bg-blue-950/20">
        <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30">
          <Icon name="key" size="lg" class="text-primary-600 dark:text-primary-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('dashboard.createApiKey') }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.generateNewKey') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500"
        />
      </button>

      <button @click="router.push('/usage')" class="group flex w-full items-center gap-3 rounded-lg border border-gray-100 bg-gray-50 p-3 text-left transition-colors hover:border-emerald-200 hover:bg-emerald-50/50 dark:border-dark-700 dark:bg-dark-900/40 dark:hover:border-emerald-900/50 dark:hover:bg-emerald-950/20">
        <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
          <Icon name="chart" size="lg" class="text-emerald-600 dark:text-emerald-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('dashboard.viewUsage') }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.checkDetailedLogs') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-emerald-500 dark:text-dark-500"
        />
      </button>

      <button @click="router.push('/monitor')" class="group flex w-full items-center gap-3 rounded-lg border border-gray-100 bg-gray-50 p-3 text-left transition-colors hover:border-sky-200 hover:bg-sky-50/50 dark:border-dark-700 dark:bg-dark-900/40 dark:hover:border-sky-900/50 dark:hover:bg-sky-950/20">
        <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-sky-100 dark:bg-sky-900/30">
          <Icon name="shield" size="lg" class="text-sky-600 dark:text-sky-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('dashboard.serviceStatus') }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.checkGroupHealth') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-sky-500 dark:text-dark-500"
        />
      </button>

      <button @click="router.push('/redeem')" class="group flex w-full items-center gap-3 rounded-lg border border-gray-100 bg-gray-50 p-3 text-left transition-colors hover:border-amber-200 hover:bg-amber-50/50 dark:border-dark-700 dark:bg-dark-900/40 dark:hover:border-amber-900/50 dark:hover:bg-amber-950/20">
        <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-amber-100 dark:bg-amber-900/30">
          <Icon name="gift" size="lg" class="text-amber-600 dark:text-amber-400" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('dashboard.redeemCode') }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.addBalanceWithCode') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-amber-500 dark:text-dark-500"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
const router = useRouter()
const { t } = useI18n()
const firstRunSteps = [
  t('dashboard.firstRun.steps.createKey'),
  t('dashboard.firstRun.steps.selectGroup'),
  t('dashboard.firstRun.steps.copyConfig')
]
</script>
