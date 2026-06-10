<template>
  <div v-if="loading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
    <div
      v-for="idx in 6"
      :key="idx"
      class="h-56 animate-pulse rounded-[1.5rem] border border-gray-100 bg-white p-5 shadow-card dark:border-dark-700 dark:bg-dark-800/60"
    >
      <div class="h-4 w-1/3 rounded bg-gray-200 dark:bg-dark-700" />
      <div class="mt-4 h-8 w-2/3 rounded bg-gray-200 dark:bg-dark-700" />
      <div class="mt-6 space-y-3">
        <div class="h-4 rounded bg-gray-100 dark:bg-dark-700/70" />
        <div class="h-4 w-5/6 rounded bg-gray-100 dark:bg-dark-700/70" />
        <div class="h-4 w-3/4 rounded bg-gray-100 dark:bg-dark-700/70" />
      </div>
    </div>
  </div>

  <div v-else-if="rows.length === 0" class="card py-16 text-center">
    <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
    <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
  </div>

  <div v-else class="space-y-4">
    <article
      v-for="(channel, chIdx) in rows"
      :key="`${channel.name}-${chIdx}`"
      class="overflow-hidden rounded-[1.5rem] border border-gray-100 bg-white shadow-card dark:border-dark-700/70 dark:bg-dark-800/60"
    >
      <div class="border-b border-gray-100 bg-gradient-to-r from-gray-50 to-white p-4 dark:border-dark-700 dark:from-dark-800 dark:to-dark-800/50 sm:p-5">
        <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div class="min-w-0">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <span class="rounded-full bg-slate-900 px-2.5 py-1 text-[11px] font-bold text-white dark:bg-white dark:text-slate-900">
                {{ t('availableChannels.channel') }}
              </span>
              <span class="text-xs font-semibold text-gray-400">
                {{ channel.platforms.length }} {{ t('availableChannels.stats.platforms') }}
              </span>
            </div>
            <h2 class="break-words text-xl font-black tracking-tight text-gray-900 dark:text-white">
              {{ channel.name }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
              <template v-if="channel.description">{{ channel.description }}</template>
              <span v-else>-</span>
            </p>
          </div>

          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 md:min-w-[280px]">
            <div class="rounded-2xl bg-gray-100 px-3 py-2 dark:bg-dark-700/70">
              <p class="text-[10px] font-bold uppercase tracking-wide text-gray-400">{{ t('availableChannels.stats.groups') }}</p>
              <p class="mt-1 text-lg font-black text-gray-900 dark:text-white">{{ channelGroupCount(channel) }}</p>
            </div>
            <div class="rounded-2xl bg-gray-100 px-3 py-2 dark:bg-dark-700/70">
              <p class="text-[10px] font-bold uppercase tracking-wide text-gray-400">{{ t('availableChannels.stats.models') }}</p>
              <p class="mt-1 text-lg font-black text-gray-900 dark:text-white">{{ channelModelCount(channel) }}</p>
            </div>
            <div class="col-span-2 rounded-2xl bg-emerald-50 px-3 py-2 dark:bg-emerald-950/30 sm:col-span-1">
              <p class="text-[10px] font-bold uppercase tracking-wide text-emerald-700/70 dark:text-emerald-300/70">{{ t('availableChannels.visible') }}</p>
              <p class="mt-1 text-lg font-black text-emerald-700 dark:text-emerald-300">{{ t('common.available') }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="grid gap-3 p-3 sm:p-4 xl:grid-cols-2 2xl:grid-cols-3">
        <section
          v-for="section in channel.platforms"
          :key="sectionKey(channel, section)"
          class="relative overflow-hidden rounded-2xl border bg-white p-4 dark:bg-dark-900/40"
          :class="platformBorderClass(section.platform)"
        >
          <div class="absolute inset-x-0 top-0 h-1" :class="platformAccentBarClass(section.platform)" />

          <div class="flex flex-wrap items-start justify-between gap-3">
            <span
              :class="[
                'inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-bold uppercase',
                platformBadgeClass(section.platform),
              ]"
            >
              <PlatformIcon :platform="section.platform as GroupPlatform" size="xs" />
              {{ section.platform }}
            </span>
            <div class="flex gap-2 text-[11px] font-semibold text-gray-500 dark:text-gray-400">
              <span>{{ section.groups.length }} {{ t('availableChannels.stats.groups') }}</span>
              <span>{{ section.supported_models.length }} {{ t('availableChannels.stats.models') }}</span>
            </div>
          </div>

          <div class="mt-4 space-y-4">
            <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-800/70">
              <div class="mb-2 flex items-center justify-between">
                <p class="text-xs font-bold text-gray-700 dark:text-gray-200">{{ t('availableChannels.groupsTitle') }}</p>
                <span class="text-[11px] text-gray-400">{{ section.groups.length }}</span>
              </div>

              <div v-if="section.groups.length > 0" class="space-y-2">
                <div
                  v-if="exclusiveGroups(section).length > 0"
                  class="space-y-1.5"
                >
                  <span
                    class="inline-flex items-center gap-0.5 text-[10px] font-bold uppercase text-purple-600 dark:text-purple-400"
                    :title="t('availableChannels.exclusiveTooltip')"
                  >
                    <Icon name="shield" size="xs" class="h-3 w-3" />
                    {{ t('availableChannels.exclusive') }}
                  </span>
                  <div class="flex flex-wrap gap-1.5">
                    <GroupBadge
                      v-for="g in exclusiveGroups(section)"
                      :key="`ex-${g.id}`"
                      :name="g.name"
                      :platform="g.platform as GroupPlatform"
                      :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                      :group-id="g.id"
                      :rate-multiplier="g.rate_multiplier"
                      :user-rate-multiplier="userGroupRates[g.id] ?? null"
                      :discount-multiplier="g.group_rate_discount_multiplier"
                      :discounted-rate-multiplier="g.discounted_rate_multiplier"
                      :discount-name="g.group_rate_discount_name"
                      :discount-schedule-mode="g.group_rate_discount_schedule_mode"
                      :discount-start-at="g.group_rate_discount_start_at"
                      :discount-end-at="g.group_rate_discount_end_at"
                      :discount-weekdays="g.group_rate_discount_weekdays"
                      :discount-daily-start-time="g.group_rate_discount_daily_start_time"
                      :discount-daily-end-time="g.group_rate_discount_daily_end_time"
                      :discount-timezone="g.group_rate_discount_timezone"
                      always-show-rate
                    />
                  </div>
                </div>

                <div
                  v-if="publicGroups(section).length > 0"
                  class="space-y-1.5"
                >
                  <span
                    class="inline-flex items-center gap-0.5 text-[10px] font-bold uppercase text-gray-500 dark:text-gray-400"
                    :title="t('availableChannels.publicTooltip')"
                  >
                    <Icon name="globe" size="xs" class="h-3 w-3" />
                    {{ t('availableChannels.public') }}
                  </span>
                  <div class="flex flex-wrap gap-1.5">
                    <GroupBadge
                      v-for="g in publicGroups(section)"
                      :key="`pub-${g.id}`"
                      :name="g.name"
                      :platform="g.platform as GroupPlatform"
                      :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                      :group-id="g.id"
                      :rate-multiplier="g.rate_multiplier"
                      :user-rate-multiplier="userGroupRates[g.id] ?? null"
                      :discount-multiplier="g.group_rate_discount_multiplier"
                      :discounted-rate-multiplier="g.discounted_rate_multiplier"
                      :discount-name="g.group_rate_discount_name"
                      :discount-schedule-mode="g.group_rate_discount_schedule_mode"
                      :discount-start-at="g.group_rate_discount_start_at"
                      :discount-end-at="g.group_rate_discount_end_at"
                      :discount-weekdays="g.group_rate_discount_weekdays"
                      :discount-daily-start-time="g.group_rate_discount_daily_start_time"
                      :discount-daily-end-time="g.group_rate_discount_daily_end_time"
                      :discount-timezone="g.group_rate_discount_timezone"
                      always-show-rate
                    />
                  </div>
                </div>
              </div>

              <span v-else class="text-xs text-gray-400">-</span>
            </div>

            <div>
              <div class="mb-2 flex items-center justify-between">
                <p class="text-xs font-bold text-gray-700 dark:text-gray-200">{{ t('availableChannels.modelsTitle') }}</p>
                <span class="text-[11px] text-gray-400">{{ section.supported_models.length }}</span>
              </div>

              <div
                v-if="section.supported_models.length > 0"
                class="max-h-44 overflow-y-auto rounded-2xl border border-gray-100 bg-white/70 p-2 dark:border-dark-700 dark:bg-dark-800/40"
              >
                <div class="flex flex-wrap gap-1.5">
                  <SupportedModelChip
                    v-for="m in section.supported_models"
                    :key="`${section.platform}-${m.name}`"
                    :model="m"
                    :pricing-key-prefix="pricingKeyPrefix"
                    :no-pricing-label="noPricingLabel"
                    :show-platform="false"
                    :platform-hint="section.platform"
                  />
                </div>
              </div>

              <span v-else class="text-xs text-gray-400">
                {{ noModelsLabel }}
              </span>
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
import type { UserAvailableChannel, UserAvailableGroup, UserChannelPlatformSection } from '@/api/channels'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformAccentBarClass, platformBadgeClass, platformBorderClass } from '@/utils/platformColors'

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

function channelModelCount(channel: UserAvailableChannel): number {
  return channel.platforms.reduce((sum, section) => sum + section.supported_models.length, 0)
}

function exclusiveGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => g.is_exclusive)
}

function publicGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => !g.is_exclusive)
}
</script>
