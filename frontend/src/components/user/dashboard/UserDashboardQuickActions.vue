<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.quickActions') }}</h2>
      <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">
        {{ t('dashboard.quickActionsHint') }}
      </p>
    </div>
    <div class="space-y-3 p-4">
      <div class="rounded-2xl border border-primary-100 bg-primary-50/70 p-4 dark:border-primary-800/50 dark:bg-primary-950/25">
        <div class="flex items-start gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-500 text-white shadow-sm shadow-primary-500/25">
            <Icon name="bolt" size="md" />
          </div>
          <div>
            <p class="text-sm font-bold text-primary-950 dark:text-primary-100">
              {{ t('dashboard.firstRun.title') }}
            </p>
            <p class="mt-1 text-xs leading-5 text-primary-800 dark:text-primary-200">
              {{ t('dashboard.firstRun.description') }}
            </p>
          </div>
        </div>
        <div class="mt-3 grid gap-2">
          <div
            v-for="(step, index) in firstRunSteps"
            :key="step"
            class="flex items-center gap-2 rounded-xl bg-white/75 px-3 py-2 text-xs font-medium text-gray-700 dark:bg-dark-900/45 dark:text-dark-200"
          >
            <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary-100 text-[10px] font-black text-primary-700 dark:bg-primary-900/50 dark:text-primary-200">
              {{ index + 1 }}
            </span>
            <span>{{ step }}</span>
          </div>
        </div>
      </div>

      <button @click="router.push('/keys')" class="group flex w-full items-center gap-4 rounded-xl bg-gray-50 p-4 text-left transition-all duration-200 hover:bg-gray-100 dark:bg-dark-800/50 dark:hover:bg-dark-800">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-primary-100 transition-transform group-hover:scale-105 dark:bg-primary-900/30">
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

      <button @click="router.push('/usage')" class="group flex w-full items-center gap-4 rounded-xl bg-gray-50 p-4 text-left transition-all duration-200 hover:bg-gray-100 dark:bg-dark-800/50 dark:hover:bg-dark-800">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 transition-transform group-hover:scale-105 dark:bg-emerald-900/30">
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

      <button @click="router.push('/monitor')" class="group flex w-full items-center gap-4 rounded-xl bg-gray-50 p-4 text-left transition-all duration-200 hover:bg-gray-100 dark:bg-dark-800/50 dark:hover:bg-dark-800">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-sky-100 transition-transform group-hover:scale-105 dark:bg-sky-900/30">
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

      <button @click="router.push('/redeem')" class="group flex w-full items-center gap-4 rounded-xl bg-gray-50 p-4 text-left transition-all duration-200 hover:bg-gray-100 dark:bg-dark-800/50 dark:hover:bg-dark-800">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-amber-100 transition-transform group-hover:scale-105 dark:bg-amber-900/30">
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
