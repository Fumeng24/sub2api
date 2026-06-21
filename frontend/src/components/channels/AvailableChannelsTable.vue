<template>
  <div v-if="loading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
    <div
      v-for="idx in 6"
      :key="idx"
      class="h-64 animate-pulse rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <div class="h-4 w-1/3 rounded bg-gray-200 dark:bg-dark-700" />
      <div class="mt-4 h-8 w-2/3 rounded bg-gray-200 dark:bg-dark-700" />
      <div class="mt-6 grid gap-3">
        <div class="h-16 rounded-lg bg-gray-100 dark:bg-dark-700/70" />
        <div class="h-16 rounded-lg bg-gray-100 dark:bg-dark-700/70" />
      </div>
    </div>
  </div>

  <div v-else-if="rows.length === 0" class="rounded-lg border border-gray-200 bg-white py-16 text-center shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
    <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
  </div>

  <div v-else class="space-y-5">
    <article
      v-for="(channel, chIdx) in rows"
      :key="`${channel.name}-${chIdx}`"
      class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <div class="border-b border-gray-100 bg-white p-4 dark:border-dark-700 dark:bg-dark-800 sm:p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="min-w-0">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center gap-1.5 rounded-md bg-gray-50 px-2.5 py-1 text-[11px] font-semibold text-gray-600 ring-1 ring-gray-200 dark:bg-dark-900 dark:text-gray-300 dark:ring-dark-700">
                <Icon name="server" size="xs" />
                {{ t('availableChannels.channel') }}
              </span>
              <span class="rounded-md bg-emerald-50 px-2.5 py-1 text-[11px] font-semibold text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-300 dark:ring-emerald-800/60">
                {{ t('availableChannels.visible') }}
              </span>
            </div>
            <h2 class="break-words text-xl font-semibold tracking-normal text-gray-950 dark:text-white">
              {{ channel.name }}
            </h2>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              <template v-if="channel.description">{{ channel.description }}</template>
              <span v-else>-</span>
            </p>
          </div>

          <div class="grid grid-cols-3 gap-2 lg:min-w-[300px]">
            <div
              v-for="item in channelStats(channel)"
              :key="item.label"
              class="rounded-lg bg-gray-50 px-3 py-2 ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700"
            >
              <p class="text-[10px] font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
              <p class="mt-1 text-lg font-semibold text-gray-950 dark:text-white">{{ item.value }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-4 p-3 sm:p-4">
        <section
          v-for="section in channel.platforms"
          :key="sectionKey(channel, section)"
          class="overflow-hidden rounded-lg border bg-white dark:bg-dark-900/40"
          :class="platformBorderClass(section.platform)"
        >
          <div class="flex flex-col gap-3 border-b border-slate-100 p-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex min-w-0 items-center gap-3">
              <span
                :class="[
                  'inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border',
                  platformBadgeLightClass(section.platform),
                  platformBorderClass(section.platform),
                ]"
              >
                <PlatformIcon :platform="section.platform as GroupPlatform" size="sm" />
              </span>
              <div class="min-w-0">
                <p class="text-[11px] font-medium text-gray-500 dark:text-gray-400">
                  {{ section.platform }}
                </p>
                <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('availableChannels.platformSectionTitle') }}
                </h3>
              </div>
            </div>

            <div class="flex flex-wrap gap-2 text-xs font-medium text-gray-500 dark:text-gray-400">
              <span class="rounded-md bg-white px-3 py-1 ring-1 ring-gray-100 dark:bg-dark-800 dark:ring-dark-700">
                {{ section.groups.length }} {{ t('availableChannels.stats.groups') }}
              </span>
              <span class="rounded-md bg-white px-3 py-1 ring-1 ring-gray-100 dark:bg-dark-800 dark:ring-dark-700">
                {{ sectionModelCount(section) }} {{ t('availableChannels.stats.models') }}
              </span>
            </div>
          </div>

          <div class="grid gap-3 p-3 sm:p-4 lg:grid-cols-2 2xl:grid-cols-3">
            <div
              v-for="group in section.groups"
              :key="group.id"
              class="group relative overflow-hidden rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition-colors hover:border-blue-200 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-900/50 dark:hover:border-blue-900/50"
            >
              <div
                class="absolute inset-x-0 top-0 h-1"
                :class="group.is_exclusive ? 'bg-gradient-to-r from-amber-400 to-orange-500' : platformAccentBarClass(section.platform)"
              />

              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="mb-2 flex flex-wrap items-center gap-1.5">
                    <span
                      :class="[
                        'inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-semibold ring-1',
                        group.is_exclusive
                          ? 'bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/30 dark:text-amber-300 dark:ring-amber-800/60'
                          : 'bg-slate-50 text-slate-600 ring-slate-200 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-700',
                      ]"
                      :title="group.is_exclusive ? t('availableChannels.exclusiveTooltip') : t('availableChannels.publicTooltip')"
                    >
                      <Icon :name="group.is_exclusive ? 'shield' : 'globe'" size="xs" />
                      {{ group.is_exclusive ? t('availableChannels.exclusive') : t('availableChannels.public') }}
                    </span>
                    <span
                      v-if="group.subscription_type === 'subscription'"
                      class="rounded-md bg-sky-50 px-2 py-0.5 text-[10px] font-semibold text-sky-700 ring-1 ring-sky-200 dark:bg-sky-950/30 dark:text-sky-300 dark:ring-sky-800/60"
                    >
                      {{ t('availableChannels.subscription') }}
                    </span>
                  </div>

                  <h4 class="break-words text-base font-semibold leading-snug text-gray-950 dark:text-white">
                    {{ group.name }}
                  </h4>
                </div>

                <GroupBadge
                  :name="group.name"
                  :platform="group.platform as GroupPlatform"
                  :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                  :group-id="group.id"
                  :rate-multiplier="group.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[group.id] ?? null"
                  :discount-multiplier="group.group_rate_discount_multiplier"
                  :discounted-rate-multiplier="group.discounted_rate_multiplier"
                  :discount-name="group.group_rate_discount_name"
                  :discount-schedule-mode="group.group_rate_discount_schedule_mode"
                  :discount-start-at="group.group_rate_discount_start_at"
                  :discount-end-at="group.group_rate_discount_end_at"
                  :discount-weekdays="group.group_rate_discount_weekdays"
                  :discount-daily-start-time="group.group_rate_discount_daily_start_time"
                  :discount-daily-end-time="group.group_rate_discount_daily_end_time"
                  :discount-timezone="group.group_rate_discount_timezone"
                  always-show-rate
                />
              </div>

              <div class="mt-4">
                <div class="mb-2 flex items-center justify-between gap-2">
                  <p class="text-xs font-semibold text-gray-800 dark:text-gray-100">
                    {{ t('availableChannels.groupModelsTitle') }}
                  </p>
                  <span class="rounded-md bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-500 dark:bg-dark-800 dark:text-gray-400">
                    {{ groupModels(group).length }}
                  </span>
                </div>

                <div
                  v-if="groupModels(group).length > 0"
                  class="max-h-48 overflow-y-auto rounded-lg border border-slate-100 bg-slate-50/80 p-2 dark:border-dark-700 dark:bg-dark-800/50"
                >
                  <div class="flex flex-wrap gap-1.5">
                    <SupportedModelChip
                      v-for="m in groupModels(group)"
                      :key="`${group.id}-${m.platform}-${m.name}`"
                      :model="m"
                      :pricing-key-prefix="pricingKeyPrefix"
                      :no-pricing-label="noPricingLabel"
                      :show-platform="false"
                      :platform-hint="section.platform"
                    />
                  </div>
                </div>

                <div
                  v-else
                  class="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-3 py-4 text-center text-xs font-medium text-gray-400 dark:border-dark-700 dark:bg-dark-800/40"
                >
                  {{ noModelsLabel }}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from './SupportedModelChip.vue'
import type {
  UserAvailableChannel,
  UserAvailableGroup,
  UserChannelPlatformSection,
  UserSupportedModel,
} from '@/api/channels'
import type { GroupPlatform, SubscriptionType } from '@/types'
import {
  platformAccentBarClass,
  platformBadgeLightClass,
  platformBorderClass,
} from '@/utils/platformColors'

defineProps<{
  rows: UserAvailableChannel[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  /** 用户专属倍率（group_id → multiplier）；无专属时由 GroupBadge 仅显示默认倍率。 */
  userGroupRates: Record<number, number>
}>()

const { t } = useI18n()

function sectionKey(channel: UserAvailableChannel, section: UserChannelPlatformSection): string {
  return `${channel.name}-${section.platform}`
}

function channelGroupCount(channel: UserAvailableChannel): number {
  return channel.platforms.reduce((sum, section) => sum + section.groups.length, 0)
}

function groupModels(
  group: UserAvailableGroup,
): UserSupportedModel[] {
  return group.supported_models || []
}

function sectionModelCount(section: UserChannelPlatformSection): number {
  const names = new Set<string>()
  for (const group of section.groups) {
    for (const model of groupModels(group)) {
      names.add(`${model.platform || section.platform}:${model.name.toLowerCase()}`)
    }
  }
  return names.size
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

function channelStats(channel: UserAvailableChannel) {
  return [
    { label: t('availableChannels.stats.platforms'), value: channel.platforms.length },
    { label: t('availableChannels.stats.groups'), value: channelGroupCount(channel) },
    { label: t('availableChannels.stats.models'), value: channelModelCount(channel) },
  ]
}
</script>
