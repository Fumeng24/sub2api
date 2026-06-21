<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <section class="space-y-4">
        <div class="flex flex-col gap-4 border-b border-gray-200 pb-4 dark:border-dark-700 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-2xl">
            <h1 class="text-2xl font-semibold tracking-normal text-gray-950 dark:text-white">
              {{ t('availableChannels.title') }}
            </h1>
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t('availableChannels.description') }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:min-w-[420px]">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="rounded-lg border border-gray-200 bg-white px-3 py-2 shadow-sm dark:border-dark-700 dark:bg-dark-800"
            >
              <p class="text-[11px] font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
              <p class="mt-1 text-xl font-semibold text-gray-950 dark:text-white">{{ item.value }}</p>
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:flex-row sm:items-center">
          <div class="relative min-w-0 flex-1">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('availableChannels.searchPlaceholder')"
              class="input h-11 pl-10"
            />
          </div>
          <button
            @click="loadChannels"
            :disabled="loading"
            class="btn btn-secondary h-11"
            :title="t('common.refresh', 'Refresh')"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh', 'Refresh') }}</span>
          </button>
        </div>
      </section>

      <AvailableChannelsTable
        :rows="filteredChannels"
        :loading="loading"
        :user-group-rates="userGroupRates"
        pricing-key-prefix="availableChannels.pricing"
        :no-pricing-label="t('availableChannels.noPricing')"
        :no-models-label="t('availableChannels.noModels')"
        :empty-label="t('availableChannels.empty')"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { UserAvailableGroup } from '@/api/channels'

const { t } = useI18n()
const appStore = useAppStore()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return channels.value
  return channels.value
    .map((ch) => {
      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      if (nameHit || descHit) return ch
      const matchingSections = ch.platforms.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.groups.some((g) =>
            groupModels(g).some((m) => m.name.toLowerCase().includes(q)),
          ),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

const summary = computed(() => {
  const list = filteredChannels.value
  const platformCount = list.reduce((sum, channel) => sum + channel.platforms.length, 0)
  const groupCount = list.reduce(
    (sum, channel) => sum + channel.platforms.reduce((n, section) => n + section.groups.length, 0),
    0,
  )
  const modelCount = list.reduce(
    (sum, channel) => sum + channelModelCount(channel),
    0,
  )
  return {
    channels: list.length,
    platforms: platformCount,
    groups: groupCount,
    models: modelCount,
  }
})

const summaryItems = computed(() => [
  { label: t('availableChannels.stats.channels'), value: summary.value.channels },
  { label: t('availableChannels.stats.platforms'), value: summary.value.platforms },
  { label: t('availableChannels.stats.groups'), value: summary.value.groups },
  { label: t('availableChannels.stats.models'), value: summary.value.models },
])

function groupModels(group: UserAvailableGroup) {
  return group.supported_models || []
}

function channelModelCount(channel: UserAvailableChannel): number {
  const names = new Set<string>()
  for (const section of channel.platforms) {
    for (const group of section.groups) {
      for (const model of groupModels(group)) {
        names.add(`${model.platform || section.platform}:${model.name.toLowerCase()}`)
      }
    }
  }
  return names.size
}

async function loadChannels() {
  loading.value = true
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
      appStore.fetchPublicSettings(true),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>
