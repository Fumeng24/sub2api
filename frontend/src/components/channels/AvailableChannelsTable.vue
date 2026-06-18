<template>
  <div v-if="loading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
    <div
      v-for="idx in 6"
      :key="idx"
      class="h-64 animate-pulse rounded-[1.75rem] border border-gray-100 bg-white p-5 shadow-card dark:border-dark-700 dark:bg-dark-800/60"
    >
      <div class="h-4 w-1/3 rounded bg-gray-200 dark:bg-dark-700" />
      <div class="mt-4 h-8 w-2/3 rounded bg-gray-200 dark:bg-dark-700" />
      <div class="mt-6 grid gap-3">
        <div class="h-16 rounded-2xl bg-gray-100 dark:bg-dark-700/70" />
        <div class="h-16 rounded-2xl bg-gray-100 dark:bg-dark-700/70" />
      </div>
    </div>
  </div>

  <div v-else-if="rows.length === 0" class="card py-16 text-center">
    <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
    <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
  </div>

  <div v-else class="space-y-5">
    <article
      v-for="(channel, chIdx) in rows"
      :key="`${channel.name}-${chIdx}`"
      class="overflow-hidden rounded-[1.75rem] border border-slate-200/80 bg-white shadow-card dark:border-dark-700/70 dark:bg-dark-800/60"
    >
      <div class="relative overflow-hidden border-b border-slate-100 bg-slate-950 p-4 text-white dark:border-dark-700 sm:p-5">
        <div class="pointer-events-none absolute -right-10 -top-14 h-36 w-36 rounded-full bg-emerald-400/20 blur-3xl" />
        <div class="pointer-events-none absolute bottom-0 left-1/3 h-24 w-24 rounded-full bg-cyan-400/10 blur-2xl" />

        <div class="relative flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="min-w-0">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center gap-1.5 rounded-full bg-white/10 px-2.5 py-1 text-[11px] font-bold text-emerald-100 ring-1 ring-white/15">
                <Icon name="server" size="xs" />
                {{ t('availableChannels.channel') }}
              </span>
              <span class="rounded-full bg-emerald-400/15 px-2.5 py-1 text-[11px] font-bold text-emerald-100 ring-1 ring-emerald-300/20">
                {{ t('availableChannels.visible') }}
              </span>
            </div>
            <h2 class="break-words text-xl font-black tracking-tight sm:text-2xl">
              {{ channel.name }}
            </h2>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-slate-300">
              <template v-if="channel.description">{{ channel.description }}</template>
              <span v-else>-</span>
            </p>
          </div>

          <div class="grid grid-cols-3 gap-2 lg:min-w-[300px]">
            <div
              v-for="item in channelStats(channel)"
              :key="item.label"
              class="rounded-2xl bg-white/10 px-3 py-2 ring-1 ring-white/15 backdrop-blur"
            >
              <p class="text-[10px] font-bold uppercase tracking-wide text-slate-400">{{ item.label }}</p>
              <p class="mt-1 text-lg font-black text-white">{{ item.value }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-4 p-3 sm:p-4">
        <section
          v-for="section in channel.platforms"
          :key="sectionKey(channel, section)"
          class="overflow-hidden rounded-[1.4rem] border bg-gradient-to-br from-white to-slate-50/70 dark:from-dark-900/40 dark:to-dark-800/60"
          :class="platformBorderClass(section.platform)"
        >
          <div class="flex flex-col gap-3 border-b border-slate-100 p-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex min-w-0 items-center gap-3">
              <span
                :class="[
                  'inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-2xl border',
                  platformBadgeLightClass(section.platform),
                  platformBorderClass(section.platform),
                ]"
              >
                <PlatformIcon :platform="section.platform as GroupPlatform" size="sm" />
              </span>
              <div class="min-w-0">
                <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-gray-400">
                  {{ section.platform }}
                </p>
                <h3 class="truncate text-base font-black text-gray-900 dark:text-white">
                  {{ t('availableChannels.platformSectionTitle') }}
                </h3>
              </div>
            </div>

            <div class="flex flex-wrap gap-2 text-xs font-bold text-gray-500 dark:text-gray-400">
              <span class="rounded-full bg-white px-3 py-1 ring-1 ring-gray-100 dark:bg-dark-800 dark:ring-dark-700">
                {{ section.groups.length }} {{ t('availableChannels.stats.groups') }}
              </span>
              <span class="rounded-full bg-white px-3 py-1 ring-1 ring-gray-100 dark:bg-dark-800 dark:ring-dark-700">
                {{ sectionModelCount(section) }} {{ t('availableChannels.stats.models') }}
              </span>
            </div>
          </div>

          <div class="grid gap-3 p-3 sm:p-4 lg:grid-cols-2 2xl:grid-cols-3">
            <div
              v-for="group in section.groups"
              :key="group.id"
              class="group relative overflow-hidden rounded-[1.25rem] border border-slate-200 bg-white p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-card dark:border-dark-700 dark:bg-dark-900/50"
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
                        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-black uppercase ring-1',
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
                      class="rounded-full bg-sky-50 px-2 py-0.5 text-[10px] font-black uppercase text-sky-700 ring-1 ring-sky-200 dark:bg-sky-950/30 dark:text-sky-300 dark:ring-sky-800/60"
                    >
                      {{ t('availableChannels.subscription') }}
                    </span>
                  </div>

                  <h4 class="break-words text-base font-black leading-snug text-gray-950 dark:text-white">
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
                  <p class="text-xs font-black text-gray-800 dark:text-gray-100">
                    {{ t('availableChannels.groupModelsTitle') }}
                  </p>
                  <span class="rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-bold text-slate-500 dark:bg-dark-800 dark:text-gray-400">
                    {{ groupModels(group).length }}
                  </span>
                </div>

                <div
                  v-if="groupModels(group).length > 0"
                  class="max-h-48 overflow-y-auto rounded-2xl border border-slate-100 bg-slate-50/80 p-2 dark:border-dark-700 dark:bg-dark-800/50"
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
                  class="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-3 py-4 text-center text-xs font-medium text-gray-400 dark:border-dark-700 dark:bg-dark-800/40"
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
