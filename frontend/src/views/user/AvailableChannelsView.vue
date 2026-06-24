<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-5">
      <section class="available-models-hero">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="min-w-0 max-w-2xl">
            <p class="available-models-hero__eyebrow">
              {{ t('availableChannels.eyebrow') }}
            </p>
            <h1 class="page-title">
              {{ t('availableChannels.title') }}
            </h1>
            <p class="page-description mt-2 leading-6">
              {{ t('availableChannels.description') }}
            </p>
            <div class="mt-3 flex flex-wrap gap-2">
              <span
                v-for="signal in modelTrustSignals"
                :key="signal"
                class="min-w-0 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2.5 py-1 text-xs font-medium text-[var(--apple-muted)]"
              >
                {{ signal }}
              </span>
            </div>
            <p class="mt-3 max-w-2xl text-sm leading-6 text-[var(--apple-muted)]">
              {{ t('availableChannels.assurance') }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:min-w-[440px]">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="available-models-stat"
            >
              <p>{{ item.label }}</p>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
        </div>

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
 * - 命中服务名/描述 → 整个服务（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，服务本身被过滤掉
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

const modelTrustSignals = computed(() => [
  t('availableChannels.trustSignals.full'),
  t('availableChannels.trustSignals.stable'),
  t('availableChannels.trustSignals.noRetention'),
  t('availableChannels.trustSignals.privacy'),
  t('availableChannels.trustSignals.transparent'),
  t('availableChannels.trustSignals.billing'),
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
.available-models-hero {
  border: 1px solid var(--apple-border);
  border-radius: var(--apple-radius);
  background: var(--apple-surface);
  padding: 20px;
  box-shadow: var(--apple-shadow-sm);
}

:global(.dark) .available-models-hero {
  background: color-mix(in srgb, var(--apple-surface) 94%, white 6%);
}

.available-models-hero__eyebrow {
  margin: 0 0 6px;
  color: var(--apple-muted);
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 0;
}

.available-models-stat {
  min-width: 0;
  border-left: 1px solid var(--apple-border-soft);
  padding: 6px 0 6px 12px;
  overflow: hidden;
}

.available-models-stat p {
  margin: 0;
  color: var(--apple-muted);
  font-size: 11px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.available-models-stat strong {
  display: block;
  margin-top: 4px;
  color: var(--apple-text);
  font-size: 22px;
  font-weight: 650;
  letter-spacing: 0;
  line-height: 1.1;
}

.available-models-toolbar {
  display: flex;
  gap: 12px;
  margin-top: 18px;
  align-items: center;
}

@media (max-width: 640px) {
  .available-models-hero {
    padding: 16px;
  }

  .available-models-stat {
    border-left: 0;
    border-radius: var(--apple-radius);
    background: var(--apple-surface-elevated);
    padding: 10px 12px;
  }

  .available-models-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
