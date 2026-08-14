<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <UserPageHero
        :kicker="t('availableChannels.gateway.kicker')"
        :title="t('availableChannels.title')"
        :description="t('availableChannels.gateway.description')"
      >
        <template #body>
          <UserSummaryStats class="mt-5" :items="summaryItems" />
        </template>

        <template #below>
          <div class="available-models-toolbar">
            <div class="relative min-w-0 flex-1">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--apple-muted)]"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="input h-11 pl-10"
              />
            </div>
            <div class="flex min-w-0 flex-wrap gap-2">
              <button
                v-for="option in platformFilterOptions"
                :key="option.value"
                type="button"
                :class="[
                  'rounded-[var(--apple-radius)] border px-3 py-2 text-sm font-medium transition-colors',
                  platformFilter === option.value
                    ? 'border-[color:var(--apple-blue)] bg-[color-mix(in_srgb,var(--apple-blue)_12%,var(--apple-surface))] text-[var(--apple-blue)]'
                    : 'border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)] hover:border-[color:var(--apple-border)] hover:text-[var(--apple-text)]'
                ]"
                @click="platformFilter = option.value"
              >
                {{ option.label }}
              </button>
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
        </template>
      </UserPageHero>

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
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import UserSummaryStats from '@/custom/user/UserSummaryStats.vue'
import AvailableChannelsTable from '@/custom/channels/WegooAvailableChannelsTable.vue'
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
const platformFilter = ref('all')

/**
 * 搜索过滤：
 * - 命中服务名/描述 → 整个服务（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，服务本身被过滤掉
 */
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const base = platformFilteredChannels.value
  if (!q) return base
  return base
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

const platformFilteredChannels = computed(() => {
  if (platformFilter.value === 'all') return channels.value
  return channels.value
    .map((ch) => {
      const matchingSections = ch.platforms.filter((p) => p.platform === platformFilter.value)
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

const platformFilterOptions = computed(() => [
  { value: 'all', label: t('availableChannels.filters.allPlatforms') },
  ...Array.from(new Set(channels.value.flatMap((channel) => channel.platforms.map((section) => section.platform))))
    .sort()
    .map((platform) => ({
      value: platform,
      label: platformLabel(platform),
    })),
])

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

function platformLabel(platform: string): string {
  switch (platform) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    default:
      return platform
  }
}

async function loadChannels() {
  loading.value = true
  try {
    // 服务列表和用户专属倍率并发拉取。专属倍率失败不阻塞服务展示——
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

<style scoped>
.available-models-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 12px;
  margin-top: 18px;
  align-items: center;
}

@media (max-width: 1024px) {
  .available-models-toolbar {
    grid-template-columns: 1fr;
  }
}

</style>
