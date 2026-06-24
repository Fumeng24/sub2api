<template>
  <div v-if="loading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
    <div
      v-for="idx in 6"
      :key="idx"
      class="h-56 animate-pulse rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5"
    >
      <div class="h-4 w-1/3 rounded bg-[var(--apple-surface-elevated)]" />
      <div class="mt-4 h-8 w-2/3 rounded bg-[var(--apple-surface-elevated)]" />
      <div class="mt-6 grid gap-3">
        <div class="h-16 rounded-lg bg-[var(--apple-surface-elevated)]" />
        <div class="h-16 rounded-lg bg-[var(--apple-surface-elevated)]" />
      </div>
    </div>
  </div>

  <div v-else-if="rows.length === 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] py-16 text-center">
    <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-[var(--apple-muted)]" />
    <p class="text-sm font-medium text-[var(--apple-muted)]">{{ emptyLabel }}</p>
  </div>

  <div v-else class="space-y-4">
    <article
      v-for="(channel, chIdx) in rows"
      :key="`${channel.name}-${chIdx}`"
      class="space-y-3 border-t border-[color:var(--apple-border-soft)] pt-5 first:border-t-0 first:pt-0"
    >
      <div>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="min-w-0">
            <p class="mb-2 text-xs font-semibold text-[var(--apple-muted)]">
              {{ t('availableChannels.channel') }}
            </p>
            <h2 class="break-words text-xl font-semibold tracking-normal text-[var(--apple-text)]">
              {{ channel.name }}
            </h2>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-[var(--apple-muted)]">
              <template v-if="channel.description">{{ channel.description }}</template>
              <span v-else>{{ t('availableChannels.defaultServiceDescription') }}</span>
            </p>
          </div>

          <div class="grid grid-cols-3 gap-2 lg:min-w-[300px]" :aria-label="t('availableChannels.serviceSummary')">
            <div
              v-for="item in channelStats(channel)"
              :key="item.label"
              class="min-w-0 rounded-lg bg-[var(--apple-surface-elevated)] px-3 py-2 ring-1 ring-[color:var(--apple-border-soft)]"
            >
              <p class="truncate text-[10px] font-medium text-[var(--apple-muted)]">{{ item.label }}</p>
              <p class="mt-1 text-lg font-semibold text-[var(--apple-text)]">{{ item.value }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-4">
        <section
          v-for="section in channel.platforms"
          :key="sectionKey(channel, section)"
          class="space-y-3 rounded-lg bg-[var(--apple-surface-elevated)] p-3 ring-1 ring-[color:var(--apple-border-soft)] sm:p-4"
        >
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex min-w-0 items-center gap-3">
              <span
                :class="[
                  'inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg',
                  platformTileClass(section.platform),
                ]"
              >
                <PlatformIcon :platform="section.platform as GroupPlatform" size="sm" />
              </span>
              <div class="min-w-0">
                <p class="text-[11px] font-medium text-[var(--apple-muted)]">
                  {{ section.platform }}
                </p>
                <h3 class="truncate text-base font-semibold text-[var(--apple-text)]">
                  {{ t('availableChannels.platformSectionTitle') }}
                </h3>
              </div>
            </div>

            <div class="flex flex-wrap gap-2 text-xs font-medium text-[var(--apple-muted)]">
              <span class="rounded-md bg-[var(--apple-surface)] px-3 py-1 ring-1 ring-[color:var(--apple-border-soft)]">
                {{ section.groups.length }} {{ t('availableChannels.stats.groups') }}
              </span>
              <span class="rounded-md bg-[var(--apple-surface)] px-3 py-1 ring-1 ring-[color:var(--apple-border-soft)]">
                {{ sectionModelCount(section) }} {{ t('availableChannels.stats.models') }}
              </span>
            </div>
          </div>

          <div class="grid gap-3 lg:grid-cols-2 2xl:grid-cols-3">
            <div
              v-for="group in section.groups"
              :key="group.id"
              class="available-channel-group-card group relative overflow-hidden rounded-lg p-4 transition-colors"
            >
              <div
                class="available-channel-group-accent absolute inset-y-4 left-0 w-1 rounded-r-full"
                :class="group.is_exclusive ? 'available-channel-group-accent--exclusive' : 'available-channel-group-accent--standard'"
              />

              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 pl-2">
                  <div class="mb-2 flex flex-wrap items-center gap-1.5">
                    <span
                      :class="[
                        'available-channel-chip inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-semibold',
                        group.is_exclusive ? 'available-channel-chip--exclusive' : 'available-channel-chip--public',
                      ]"
                      :title="group.is_exclusive ? t('availableChannels.exclusiveTooltip') : t('availableChannels.publicTooltip')"
                    >
                      <Icon :name="group.is_exclusive ? 'shield' : 'globe'" size="xs" />
                      {{ group.is_exclusive ? t('availableChannels.exclusive') : t('availableChannels.public') }}
                    </span>
                    <span
                      v-if="group.subscription_type === 'subscription'"
                      class="available-channel-chip available-channel-chip--subscription rounded-md px-2 py-0.5 text-[10px] font-semibold"
                    >
                      {{ t('availableChannels.subscription') }}
                    </span>
                  </div>

                  <h4 class="break-words text-base font-semibold leading-snug text-[var(--apple-text)]">
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
                  <p class="text-xs font-semibold text-[var(--apple-text)]">
                    {{ t('availableChannels.groupModelsTitle') }}
                  </p>
                  <span class="rounded-md border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] px-2 py-0.5 text-[11px] font-medium text-[var(--apple-muted)]">
                    {{ groupModels(group).length }}
                  </span>
                </div>

                <div
                  v-if="groupModels(group).length > 0"
                  class="max-h-48 overflow-y-auto rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] p-2"
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
                  class="rounded-lg border border-dashed border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 py-4 text-center text-xs font-medium text-[var(--apple-muted)]"
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

function platformTileClass(platform: string): string {
  switch (platform) {
    case 'openai':
      return 'available-platform-tile--success'
    case 'anthropic':
      return 'available-platform-tile--warning'
    case 'gemini':
    case 'antigravity':
      return 'available-platform-tile--blue'
    default:
      return 'available-platform-tile--neutral'
  }
}
</script>

<style scoped>
.available-platform-tile--success {
  background: color-mix(in srgb, var(--apple-success) 10%, var(--apple-surface));
  color: var(--apple-success);
}

.available-platform-tile--warning {
  background: color-mix(in srgb, var(--apple-warning) 11%, var(--apple-surface));
  color: var(--apple-warning);
}

.available-platform-tile--blue {
  background: color-mix(in srgb, var(--apple-blue) 10%, var(--apple-surface));
  color: var(--apple-blue);
}

.available-platform-tile--neutral {
  background: var(--apple-surface);
  color: var(--apple-muted);
}

.available-channel-group-card {
  border: 1px solid var(--apple-border-soft);
  background: var(--apple-surface);
}

.available-channel-group-card:hover {
  border-color: var(--apple-border);
  background: color-mix(in srgb, var(--apple-surface) 95%, var(--apple-hover));
}

.available-channel-group-accent--exclusive {
  background: var(--apple-warning);
}

.available-channel-group-accent--standard {
  background: color-mix(in srgb, var(--apple-muted) 34%, transparent);
}

.available-channel-chip {
  border: 1px solid transparent;
}

.available-channel-chip--exclusive {
  background: color-mix(in srgb, var(--apple-warning) 11%, transparent);
  border-color: color-mix(in srgb, var(--apple-warning) 22%, transparent);
  color: var(--apple-warning);
}

.available-channel-chip--public {
  background: var(--apple-surface-elevated);
  border-color: var(--apple-border-soft);
  color: var(--apple-muted);
}

.available-channel-chip--subscription {
  background: color-mix(in srgb, var(--apple-blue) 10%, transparent);
  border-color: color-mix(in srgb, var(--apple-blue) 20%, transparent);
  color: var(--apple-blue);
}
</style>
