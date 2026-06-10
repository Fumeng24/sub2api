<template>
  <AppLayout>
    <div class="space-y-5">
      <section class="relative overflow-hidden rounded-[2rem] border border-slate-200/80 bg-gradient-to-br from-slate-950 via-slate-900 to-emerald-950 p-5 text-white shadow-card dark:border-dark-700 sm:p-6">
        <div class="pointer-events-none absolute -right-16 -top-16 h-44 w-44 rounded-full bg-emerald-400/20 blur-3xl" />
        <div class="pointer-events-none absolute bottom-0 left-1/4 h-32 w-32 rounded-full bg-cyan-400/10 blur-3xl" />

        <div class="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-2xl">
            <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-white/10 px-3 py-1 text-xs font-semibold text-emerald-100 ring-1 ring-white/15">
              <span class="h-1.5 w-1.5 rounded-full bg-emerald-300" />
              {{ t('availableChannels.title') }}
            </div>
            <h1 class="text-2xl font-black tracking-tight sm:text-3xl">
              {{ t('availableChannels.title') }}
            </h1>
            <p class="mt-2 text-sm leading-6 text-slate-300">
              {{ t('availableChannels.description') }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:min-w-[420px]">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="rounded-2xl bg-white/10 px-3 py-2 ring-1 ring-white/15 backdrop-blur"
            >
              <p class="text-[11px] font-semibold text-slate-300">{{ item.label }}</p>
              <p class="mt-1 text-xl font-black text-white">{{ item.value }}</p>
            </div>
          </div>
        </div>

        <div class="relative mt-5 flex flex-col gap-3 sm:flex-row sm:items-center">
          <div class="relative min-w-0 flex-1">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('availableChannels.searchPlaceholder')"
              class="h-11 w-full rounded-2xl border border-white/15 bg-white/10 pl-10 pr-4 text-sm text-white placeholder:text-slate-400 outline-none backdrop-blur transition focus:border-emerald-300 focus:bg-white/15"
            />
          </div>
          <button
            @click="loadChannels"
            :disabled="loading"
            class="inline-flex h-11 items-center justify-center gap-2 rounded-2xl bg-emerald-400 px-4 text-sm font-bold text-slate-950 shadow-lg shadow-emerald-500/20 transition hover:bg-emerald-300 disabled:cursor-not-allowed disabled:opacity-60"
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
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
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
    (sum, channel) => sum + channel.platforms.reduce((n, section) => n + section.supported_models.length, 0),
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
