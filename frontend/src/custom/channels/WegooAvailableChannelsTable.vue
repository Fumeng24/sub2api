<template>
  <div v-if="loading" class="grid gap-4">
    <div class="h-64 animate-pulse rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4">
      <div class="h-4 w-1/4 rounded bg-[var(--apple-surface-elevated)]" />
      <div class="mt-4 grid gap-2 sm:grid-cols-3">
        <div class="h-12 rounded-lg bg-[var(--apple-surface-elevated)]" />
        <div class="h-12 rounded-lg bg-[var(--apple-surface-elevated)]" />
        <div class="h-12 rounded-lg bg-[var(--apple-surface-elevated)]" />
      </div>
      <div class="mt-5 h-32 rounded-lg bg-[var(--apple-surface-elevated)]" />
    </div>
  </div>

  <div v-else-if="availableGroups.length === 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] py-16 text-center">
    <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-[var(--apple-muted)]" />
    <p class="text-sm font-medium text-[var(--apple-muted)]">{{ emptyLabel }}</p>
  </div>

  <section v-else class="available-model-catalog">
    <div class="available-model-catalog__header">
      <div class="min-w-0">
        <p class="text-xs font-semibold text-[var(--apple-muted)]">
          {{ t('availableChannels.modelTable.title') }}
        </p>
        <h2 class="mt-1 text-xl font-semibold text-[var(--apple-text)]">
          {{ selectedGroup?.group.name || '-' }}
        </h2>
        <p class="mt-1 text-sm text-[var(--apple-muted)]">
          {{ t('availableChannels.modelTable.groupSummary', {
            groups: availableGroups.length,
            models: selectedModelRows.length,
          }) }}
        </p>
      </div>

      <div v-if="selectedGroup" class="available-selected-rate">
        <span>{{ t('availableChannels.modelTable.effectiveRate') }}</span>
        <strong>{{ formatRate(effectiveGroupRate(selectedGroup.group)) }}x</strong>
        <em v-if="hasPeakRate(selectedGroup.group)" :title="peakRateTitle(selectedGroup.group)">
          {{ peakRateLabel(selectedGroup.group) }}
        </em>
      </div>
    </div>

    <div class="available-group-tabs-shell">
      <button
        type="button"
        class="available-group-scroll-button"
        :aria-label="t('availableChannels.modelTable.scrollLeft')"
        @click="scrollGroupTabs(-1)"
      >
        <Icon name="chevronLeft" size="sm" />
      </button>
      <div
        ref="groupTabsRef"
        class="available-group-tabs"
        role="tablist"
        :aria-label="t('availableChannels.modelTable.selectGroup')"
      >
        <button
          v-for="entry in availableGroups"
          :key="entry.group.id"
          type="button"
          class="available-group-tab"
          :class="{ 'available-group-tab--active': selectedGroupID === entry.group.id }"
          role="tab"
          :aria-selected="selectedGroupID === entry.group.id"
          @click="selectedGroupID = entry.group.id"
        >
          <span class="available-group-tab__main">
            <span class="available-group-tab__name">{{ entry.group.name }}</span>
            <span class="available-group-tab__meta">
              {{ entry.group.is_exclusive ? t('availableChannels.exclusive') : t('availableChannels.public') }}
              <template v-if="entry.group.subscription_type === 'subscription'">
                · {{ t('availableChannels.subscription') }}
              </template>
            </span>
          </span>
          <span class="available-group-tab__rate">
            {{ formatRate(effectiveGroupRate(entry.group)) }}x
            <small v-if="hasPeakRate(entry.group)" :title="peakRateTitle(entry.group)">
              {{ peakRateLabel(entry.group) }}
            </small>
          </span>
        </button>
      </div>
      <button
        type="button"
        class="available-group-scroll-button"
        :aria-label="t('availableChannels.modelTable.scrollRight')"
        @click="scrollGroupTabs(1)"
      >
        <Icon name="chevronRight" size="sm" />
      </button>
    </div>

    <div v-if="selectedModelRows.length > 0" class="available-model-results">
      <div class="available-model-table-wrap">
        <table class="available-model-table" data-testid="available-channels-model-table">
        <thead>
          <tr>
            <th>{{ t('availableChannels.columns.model') }}</th>
            <th>{{ t('availableChannels.columns.platform') }}</th>
            <th>{{ t('availableChannels.pricing.billingMode') }}</th>
            <th>{{ t('availableChannels.modelTable.inputPrice') }}</th>
            <th>{{ t('availableChannels.modelTable.outputPrice') }}</th>
            <th>{{ t('availableChannels.modelTable.cacheWritePrice') }}</th>
            <th>{{ t('availableChannels.modelTable.cacheReadPrice') }}</th>
            <th>{{ t('availableChannels.columns.endpoints') }}</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="section in platformSections" :key="section.platform">
            <tr class="available-platform-section-row">
              <td :colspan="8">
                <div class="available-platform-section">
                  <span class="available-platform-section__icon">
                    <PlatformIcon :platform="section.platform as GroupPlatform" size="sm" />
                  </span>
                  <div class="available-platform-section__main">
                    <strong>{{ platformLabel(section.platform) }}</strong>
                    <span>{{ t('availableChannels.modelTable.modelCount', { count: section.rows.length }) }}</span>
                  </div>
                  <div class="available-platform-section__endpoints">
                    <span v-for="endpoint in section.endpoints" :key="`${section.platform}-${endpoint}`">
                      {{ endpoint }}
                    </span>
                  </div>
                </div>
              </td>
            </tr>
            <tr v-for="row in section.rows" :key="row.key">
            <td class="available-model-table__model">
              <p class="break-all font-mono text-xs font-semibold text-[var(--apple-text)]">
                {{ row.model.name }}
              </p>
              <p class="mt-1 text-[11px] text-[var(--apple-muted)]">
                {{ row.channelName }}
              </p>
            </td>
            <td>
              <span class="available-platform-pill">
                <PlatformIcon :platform="row.platform as GroupPlatform" size="xs" />
                {{ row.platform }}
              </span>
            </td>
            <td>
              <span class="available-billing-mode">
                {{ billingModeLabel(row.model.pricing?.billing_mode) }}
              </span>
            </td>
            <td class="available-price-cell">
              <strong>{{ groupAdjustedPrice(row.model, 'input') }}</strong>
              <span v-if="priceUnitLabel(row.model, 'input')">{{ priceUnitLabel(row.model, 'input') }}</span>
            </td>
            <td class="available-price-cell">
              <strong>{{ groupAdjustedPrice(row.model, 'output') }}</strong>
              <span v-if="priceUnitLabel(row.model, 'output')">{{ priceUnitLabel(row.model, 'output') }}</span>
            </td>
            <td class="available-price-cell">
              <strong>{{ groupAdjustedPrice(row.model, 'cache_write') }}</strong>
              <span v-if="priceUnitLabel(row.model, 'cache_write')">{{ priceUnitLabel(row.model, 'cache_write') }}</span>
            </td>
            <td class="available-price-cell">
              <strong>{{ groupAdjustedPrice(row.model, 'cache_read') }}</strong>
              <span v-if="priceUnitLabel(row.model, 'cache_read')">{{ priceUnitLabel(row.model, 'cache_read') }}</span>
            </td>
            <td>
              <div class="available-endpoint-list">
                <span v-for="endpoint in row.endpoints" :key="`${row.key}-${endpoint}`">
                  {{ endpoint }}
                </span>
              </div>
            </td>
            </tr>
          </template>
        </tbody>
        </table>
      </div>

      <div class="available-model-mobile" data-testid="available-channels-mobile">
        <section
          v-for="section in platformSections"
          :key="`mobile-${section.platform}`"
          class="available-model-mobile__section"
        >
          <header class="available-model-mobile__platform">
            <span class="available-platform-section__icon">
              <PlatformIcon :platform="section.platform as GroupPlatform" size="sm" />
            </span>
            <div class="available-model-mobile__platform-main">
              <strong>{{ platformLabel(section.platform) }}</strong>
              <span>{{ t('availableChannels.modelTable.modelCount', { count: section.rows.length }) }}</span>
            </div>
            <div class="available-model-mobile__endpoints">
              <span v-for="endpoint in section.endpoints" :key="`mobile-${section.platform}-${endpoint}`">
                {{ endpoint }}
              </span>
            </div>
          </header>

          <article
            v-for="row in section.rows"
            :key="`mobile-${row.key}`"
            class="available-model-mobile__model"
          >
            <header class="available-model-mobile__model-header">
              <div class="min-w-0">
                <p class="break-all font-mono text-xs font-semibold text-[var(--apple-text)]">
                  {{ row.model.name }}
                </p>
                <p class="mt-1 break-words text-[11px] text-[var(--apple-muted)]">
                  {{ row.channelName }}
                </p>
              </div>
              <span class="available-billing-mode">
                {{ billingModeLabel(row.model.pricing?.billing_mode) }}
              </span>
            </header>

            <dl class="available-model-mobile__prices">
              <div>
                <dt>{{ t('availableChannels.modelTable.inputPrice') }}</dt>
                <dd>
                  <strong>{{ groupAdjustedPrice(row.model, 'input') }}</strong>
                  <span v-if="priceUnitLabel(row.model, 'input')">{{ priceUnitLabel(row.model, 'input') }}</span>
                </dd>
              </div>
              <div>
                <dt>{{ t('availableChannels.modelTable.outputPrice') }}</dt>
                <dd>
                  <strong>{{ groupAdjustedPrice(row.model, 'output') }}</strong>
                  <span v-if="priceUnitLabel(row.model, 'output')">{{ priceUnitLabel(row.model, 'output') }}</span>
                </dd>
              </div>
              <div>
                <dt>{{ t('availableChannels.modelTable.cacheWritePrice') }}</dt>
                <dd>
                  <strong>{{ groupAdjustedPrice(row.model, 'cache_write') }}</strong>
                  <span v-if="priceUnitLabel(row.model, 'cache_write')">{{ priceUnitLabel(row.model, 'cache_write') }}</span>
                </dd>
              </div>
              <div>
                <dt>{{ t('availableChannels.modelTable.cacheReadPrice') }}</dt>
                <dd>
                  <strong>{{ groupAdjustedPrice(row.model, 'cache_read') }}</strong>
                  <span v-if="priceUnitLabel(row.model, 'cache_read')">{{ priceUnitLabel(row.model, 'cache_read') }}</span>
                </dd>
              </div>
            </dl>
          </article>
        </section>
      </div>
    </div>

    <div
      v-else
      class="rounded-lg border border-dashed border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 py-8 text-center text-xs font-medium text-[var(--apple-muted)]"
    >
      {{ noModelsLabel }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  type BillingMode,
} from '@/constants/channel'
import { formatScaledCny } from '@/custom/channels/pricing'
import type {
  UserAvailableChannel,
  UserAvailableGroup,
  UserChannelPlatformSection,
  UserSupportedModel,
} from '@/api/channels'
import type { GroupPlatform } from '@/types'
import { useAppStore } from '@/stores/app'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'

const props = defineProps<{
  rows: UserAvailableChannel[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  /** 用户专属倍率（group_id → multiplier）。 */
  userGroupRates: Record<number, number>
}>()

const { t } = useI18n()
const selectedGroupID = ref<number | null>(null)
const groupTabsRef = ref<HTMLElement | null>(null)

type GroupEntry = {
  group: UserAvailableGroup
  channels: Set<string>
  platforms: Set<string>
  endpoints: Set<string>
  models: ModelRow[]
}

type ModelRow = {
  key: string
  channelName: string
  platform: string
  endpoints: string[]
  model: UserSupportedModel
}

type PriceKind = 'input' | 'output' | 'cache_write' | 'cache_read'

type PlatformSection = {
  platform: string
  rows: ModelRow[]
  endpoints: string[]
}

const availableGroups = computed<GroupEntry[]>(() => {
  const byID = new Map<number, GroupEntry>()

  for (const channel of props.rows) {
    for (const section of channel.platforms) {
      const endpoints = sectionEndpoints(section)
      for (const group of section.groups) {
        let entry = byID.get(group.id)
        if (!entry) {
          entry = {
            group,
            channels: new Set<string>(),
            platforms: new Set<string>(),
            endpoints: new Set<string>(),
            models: [],
          }
          byID.set(group.id, entry)
        }

        entry.channels.add(channel.name)
        entry.platforms.add(section.platform)
        for (const endpoint of endpoints) entry.endpoints.add(endpoint)

        for (const model of groupModels(group)) {
          entry.models.push({
            key: `${group.id}:${channel.name}:${section.platform}:${model.platform}:${model.name}`,
            channelName: channel.name,
            platform: model.platform || section.platform,
            endpoints,
            model,
          })
        }
      }
    }
  }

  return [...byID.values()]
    .map((entry) => ({
      ...entry,
      models: dedupeModelRows(entry.models),
    }))
    .filter((entry) => !isImageGroup(entry) && entry.models.length > 0)
    .sort(compareGroupEntries)
})

const selectedGroup = computed<GroupEntry | null>(() => {
  if (availableGroups.value.length === 0) return null
  return availableGroups.value.find((entry) => entry.group.id === selectedGroupID.value) ?? availableGroups.value[0]
})

const selectedModelRows = computed<ModelRow[]>(() => {
  return selectedGroup.value?.models ?? []
})

const platformSections = computed<PlatformSection[]>(() => {
  const byPlatform = new Map<string, { platform: string, rows: ModelRow[], endpoints: Set<string> }>()
  for (const row of selectedModelRows.value) {
    const platform = normalizePlatform(row.platform)
    let section = byPlatform.get(platform)
    if (!section) {
      section = {
        platform,
        rows: [],
        endpoints: new Set<string>(),
      }
      byPlatform.set(platform, section)
    }
    section.rows.push(row)
    for (const endpoint of row.endpoints) section.endpoints.add(endpoint)
  }

  return [...byPlatform.values()]
    .map((section) => ({
      platform: section.platform,
      rows: section.rows.sort((a, b) => compareSupportedModelsByName(a.model, b.model)),
      endpoints: [...section.endpoints].sort((a, b) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })),
    }))
    .sort((a, b) => platformRank(a.platform) - platformRank(b.platform)
      || platformLabel(a.platform).localeCompare(platformLabel(b.platform), undefined, { numeric: true, sensitivity: 'base' }))
})

watch(
  availableGroups,
  (groups) => {
    if (groups.length === 0) {
      selectedGroupID.value = null
      return
    }
    if (!groups.some((entry) => entry.group.id === selectedGroupID.value)) {
      selectedGroupID.value = groups[0].group.id
    }
  },
  { immediate: true },
)

function groupModels(group: UserAvailableGroup): UserSupportedModel[] {
  return [...(group.supported_models || [])]
    .filter((model) => !isImageModel(model))
    .sort(compareSupportedModelsByName)
}

function compareSupportedModelsByName(a: UserSupportedModel, b: UserSupportedModel): number {
  const imageDiff = Number(isImageModel(a)) - Number(isImageModel(b))
  if (imageDiff !== 0) return imageDiff
  return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
    || (a.platform || '').localeCompare(b.platform || '', undefined, { numeric: true, sensitivity: 'base' })
}

function dedupeModelRows(rows: ModelRow[]): ModelRow[] {
  const seen = new Set<string>()
  const result: ModelRow[] = []
  for (const row of rows) {
    const key = `${row.platform}:${row.model.name}`
    if (seen.has(key)) continue
    seen.add(key)
    result.push(row)
  }
  return result.sort((a, b) => compareSupportedModelsByName(a.model, b.model))
}

function normalizePlatform(platform: string): string {
  return (platform || 'compatible').trim().toLowerCase()
}

function platformRank(platform: string): number {
  switch (normalizePlatform(platform)) {
    case 'anthropic':
      return 10
    case 'openai':
      return 20
    case 'gemini':
    case 'antigravity':
      return 30
    default:
      return 100
  }
}

function compareGroupEntries(a: GroupEntry, b: GroupEntry): number {
  const imageDiff = Number(isImageGroup(a)) - Number(isImageGroup(b))
  if (imageDiff !== 0) return imageDiff

  const platformDiff = platformRank(a.group.platform) - platformRank(b.group.platform)
  if (platformDiff !== 0) return platformDiff

  const rateDiff = effectiveGroupRate(a.group) - effectiveGroupRate(b.group)
  if (Math.abs(rateDiff) > Number.EPSILON) return rateDiff

  return a.group.name.localeCompare(b.group.name, undefined, { numeric: true, sensitivity: 'base' })
}

function isImageGroup(entry: GroupEntry): boolean {
  return isImageName(entry.group.name)
}

function isImageModel(model: UserSupportedModel): boolean {
  return model.pricing?.billing_mode === BILLING_MODE_IMAGE || isImageName(model.name)
}

function isImageName(value: string): boolean {
  return /生图|图片|image|img/i.test(value || '')
}

function platformLabel(platform: string): string {
  switch (normalizePlatform(platform)) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic / Claude'
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    case 'compatible':
      return 'OpenAI Compatible'
    default:
      return platform
  }
}

function scrollGroupTabs(direction: -1 | 1): void {
  const el = groupTabsRef.value
  if (!el) return
  el.scrollBy({
    left: direction * Math.max(260, Math.floor(el.clientWidth * 0.72)),
    behavior: 'smooth',
  })
}

function effectiveGroupRate(group: UserAvailableGroup): number {
  const discounted = group.discounted_rate_multiplier
  if (typeof discounted === 'number' && Number.isFinite(discounted) && discounted >= 0) return discounted
  const userRate = props.userGroupRates[group.id]
  if (Number.isFinite(userRate) && userRate >= 0) return userRate
  return group.rate_multiplier
}

function formatRate(value: number): string {
  if (!Number.isFinite(value)) return '-'
  return Number(value.toFixed(6)).toString()
}

function billingModeLabel(mode?: BillingMode): string {
  switch (mode) {
    case BILLING_MODE_TOKEN:
      return t(`${props.pricingKeyPrefix}.billingModeToken`)
    case BILLING_MODE_PER_REQUEST:
      return t(`${props.pricingKeyPrefix}.billingModePerRequest`)
    case BILLING_MODE_IMAGE:
      return t(`${props.pricingKeyPrefix}.billingModeImage`)
    default:
      return '-'
  }
}

function scaledPrice(value: number | null, scale: number, rate = 1): string {
  if (value == null) return '-'
  return formatScaledCny(value * rate, scale)
}

function selectedRate(): number {
  if (!selectedGroup.value) return 1
  return effectiveGroupRate(selectedGroup.value.group)
}

function groupAdjustedPrice(model: UserSupportedModel, kind: PriceKind): string {
  const pricing = model.pricing
  if (!pricing) return '-'

  const rate = selectedRate()
  switch (kind) {
    case 'input':
      return pricing.billing_mode === BILLING_MODE_TOKEN
        ? scaledPrice(pricing.input_price, 1_000_000, rate)
        : '-'
    case 'output':
      return pricing.billing_mode === BILLING_MODE_TOKEN
        ? scaledPrice(pricing.output_price, 1_000_000, rate)
        : '-'
    case 'cache_write':
      return pricing.billing_mode === BILLING_MODE_TOKEN
        ? scaledPrice(pricing.cache_write_price, 1_000_000, rate)
        : '-'
    case 'cache_read':
      return pricing.billing_mode === BILLING_MODE_TOKEN
        ? scaledPrice(pricing.cache_read_price, 1_000_000, rate)
        : '-'
    default:
      return '-'
  }
}

function priceUnitLabel(model: UserSupportedModel, kind: PriceKind): string {
  const pricing = model.pricing
  if (!pricing) return ''
  switch (kind) {
    case 'input':
      return pricing.billing_mode === BILLING_MODE_TOKEN && pricing.input_price != null
        ? t('availableChannels.pricing.unitPerMillion')
        : ''
    case 'output':
      return pricing.billing_mode === BILLING_MODE_TOKEN && pricing.output_price != null
        ? t('availableChannels.pricing.unitPerMillion')
        : ''
    case 'cache_write':
      return pricing.billing_mode === BILLING_MODE_TOKEN && pricing.cache_write_price != null
        ? t('availableChannels.pricing.unitPerMillion')
        : ''
    case 'cache_read':
      return pricing.billing_mode === BILLING_MODE_TOKEN && pricing.cache_read_price != null
        ? t('availableChannels.pricing.unitPerMillion')
        : ''
    default:
      return ''
  }
}

function platformEndpoints(platform: string): string[] {
  switch (platform) {
    case 'openai':
      return ['OpenAI Chat', 'Responses', 'Images']
    case 'anthropic':
      return ['Anthropic Messages']
    case 'gemini':
    case 'antigravity':
      return ['Gemini API']
    default:
      return ['OpenAI Compatible']
  }
}

function sectionEndpoints(section: UserChannelPlatformSection): string[] {
  if (section.endpoints && section.endpoints.length > 0) {
    return section.endpoints
  }
  if (section.supported_endpoint_types && section.supported_endpoint_types.length > 0) {
    return section.supported_endpoint_types
  }
  return platformEndpoints(section.platform)
}

const appStore = useAppStore()

function hasPeakRate(group: UserAvailableGroup): boolean {
  return groupHasPeakRate(group)
}

function peakRateLabel(group: UserAvailableGroup): string {
  return formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

function peakRateTitle(group: UserAvailableGroup): string {
  return t('common.peakRateTooltip', { window: peakRateLabel(group) }) + t('common.peakRateImageNote')
}
</script>

<style scoped>
.available-model-catalog {
  border: 1px solid var(--apple-border);
  border-radius: var(--apple-radius);
  background: var(--apple-surface);
  padding: 1.125rem;
  box-shadow: var(--apple-shadow-sm);
}

.available-model-catalog__header {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.available-selected-rate {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 0.4rem;
  border: 1px solid color-mix(in srgb, var(--apple-blue) 24%, var(--apple-border-soft));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--apple-blue) 7%, var(--apple-surface));
  padding: 0.45rem 0.65rem;
  color: var(--apple-muted);
  font-size: 0.78rem;
}

.available-selected-rate strong {
  color: var(--apple-blue);
  font-weight: 700;
}

.available-selected-rate em {
  border-radius: 999px;
  background: rgba(245, 158, 11, 0.12);
  padding: 0.12rem 0.45rem;
  color: #b45309;
  font-size: 0.68rem;
  font-style: normal;
  font-weight: 700;
}

.available-group-tabs-shell {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.4rem;
  margin-top: 1rem;
}

.available-group-tabs-shell::before,
.available-group-tabs-shell::after {
  position: absolute;
  z-index: 2;
  top: 0;
  bottom: 0.15rem;
  width: 2rem;
  pointer-events: none;
  content: "";
}

.available-group-tabs-shell::before {
  left: 2.35rem;
  background: linear-gradient(90deg, var(--apple-surface), transparent);
}

.available-group-tabs-shell::after {
  right: 2.35rem;
  background: linear-gradient(270deg, var(--apple-surface), transparent);
}

.available-group-scroll-button {
  position: relative;
  z-index: 3;
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--apple-border-soft);
  border-radius: 999px;
  background: var(--apple-surface-elevated);
  color: var(--apple-muted);
  transition: border-color 150ms ease, background-color 150ms ease, color 150ms ease;
}

.available-group-scroll-button:hover {
  border-color: var(--apple-border);
  background: color-mix(in srgb, var(--apple-blue) 8%, var(--apple-surface-elevated));
  color: var(--apple-text);
}

.available-group-tabs {
  display: flex;
  min-width: 0;
  gap: 0.375rem;
  overflow-x: auto;
  padding: 0.1rem 0.25rem 0.3rem;
  scrollbar-width: thin;
  scroll-behavior: smooth;
}

.available-group-tab {
  display: inline-flex;
  min-width: 10rem;
  max-width: 16rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  border: 1px solid var(--apple-border-soft);
  border-radius: 0.45rem;
  background: var(--apple-surface);
  padding: 0.5rem 0.6rem;
  text-align: left;
  transition: border-color 150ms ease, background-color 150ms ease, box-shadow 150ms ease;
}

.available-group-tab:hover {
  border-color: var(--apple-border);
  background: var(--apple-surface-elevated);
}

.available-group-tab--active {
  border-color: color-mix(in srgb, var(--apple-blue) 45%, var(--apple-border));
  background: color-mix(in srgb, var(--apple-blue) 8%, var(--apple-surface));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--apple-blue) 18%, transparent);
}

.available-group-tab__main {
  min-width: 0;
}

.available-group-tab__name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--apple-text);
  font-size: 0.82rem;
  font-weight: 650;
}

.available-group-tab__meta {
  margin-top: 0.2rem;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--apple-muted);
  font-size: 0.68rem;
}

.available-group-tab__rate {
  flex-shrink: 0;
  color: var(--apple-blue);
  font-size: 0.78rem;
  font-weight: 700;
}

.available-group-tab__rate small {
  display: block;
  margin-top: 0.08rem;
  color: #b45309;
  font-size: 0.62rem;
  font-weight: 700;
}

.available-model-table-wrap {
  margin-top: 1rem;
  overflow-x: auto;
  border: 1px solid var(--apple-border-soft);
  border-radius: 0.5rem;
  background: var(--apple-surface);
}

.available-model-mobile {
  display: none;
}

.available-platform-overview {
  display: flex;
  min-width: 980px;
  gap: 0.5rem;
  border-bottom: 1px solid var(--apple-border-soft);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--apple-surface-elevated) 72%, transparent), transparent),
    var(--apple-surface);
  padding: 0.65rem;
}

.available-platform-overview__item {
  display: inline-flex;
  min-width: 11rem;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid var(--apple-border-soft);
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--apple-surface-elevated) 72%, var(--apple-surface));
  padding: 0.55rem 0.65rem;
}

.available-platform-overview__icon,
.available-platform-section__icon {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--apple-border-soft);
  border-radius: 0.45rem;
  background: var(--apple-surface);
  color: var(--apple-text);
}

.available-platform-overview__body {
  display: grid;
  min-width: 0;
  gap: 0.15rem;
}

.available-platform-overview__body strong {
  overflow: hidden;
  color: var(--apple-text);
  font-size: 0.82rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.available-platform-overview__body span {
  color: var(--apple-muted);
  font-size: 0.68rem;
  font-weight: 600;
}

.available-model-table {
  width: 100%;
  min-width: 980px;
  border-collapse: collapse;
}

.available-model-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 0.7rem 0.75rem;
  border-bottom: 1px solid var(--apple-border-soft);
  background: color-mix(in srgb, var(--apple-surface-elevated) 86%, var(--apple-surface));
  color: var(--apple-muted);
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0;
  text-align: left;
  white-space: nowrap;
}

.available-model-table td {
  vertical-align: middle;
  padding: 0.72rem 0.75rem;
  border-bottom: 1px solid var(--apple-border-soft);
  color: var(--apple-text);
}

.available-model-table tbody tr {
  transition: background-color 150ms ease;
}

.available-model-table tbody tr:hover {
  background: color-mix(in srgb, var(--apple-blue) 4%, transparent);
}

.available-model-table tbody tr:last-child td {
  border-bottom: 0;
}

.available-platform-section-row td {
  padding: 0;
  border-bottom: 1px solid var(--apple-border-soft);
  background: color-mix(in srgb, var(--apple-blue) 5%, var(--apple-surface));
}

.available-platform-section {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
  padding: 0.68rem 0.75rem;
}

.available-platform-section__main {
  display: grid;
  min-width: 10rem;
  gap: 0.12rem;
}

.available-platform-section__main strong {
  color: var(--apple-text);
  font-size: 0.86rem;
  font-weight: 750;
}

.available-platform-section__main span {
  color: var(--apple-muted);
  font-size: 0.68rem;
  font-weight: 600;
}

.available-platform-section__endpoints {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-wrap: wrap;
  gap: 0.3rem;
  justify-content: flex-end;
}

.available-platform-section__endpoints span {
  border: 1px solid var(--apple-border-soft);
  border-radius: 999px;
  background: var(--apple-surface);
  padding: 0.18rem 0.45rem;
  color: var(--apple-muted);
  font-size: 0.68rem;
  font-weight: 600;
  white-space: nowrap;
}

.available-model-table__model {
  min-width: 14rem;
}

.available-platform-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 0.45rem;
  background: var(--apple-surface-elevated);
  padding: 0.22rem 0.5rem;
  color: var(--apple-muted);
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}

.available-endpoint-list {
  display: flex;
  max-width: 16rem;
  flex-wrap: wrap;
  gap: 0.3rem;
}

.available-endpoint-list span {
  border: 1px solid var(--apple-border-soft);
  border-radius: 0.4rem;
  padding: 0.15rem 0.35rem;
  color: var(--apple-muted);
  font-size: 0.68rem;
  white-space: nowrap;
}

.available-billing-mode {
  display: inline-flex;
  align-items: center;
  border-radius: 0.45rem;
  background: var(--apple-surface-elevated);
  padding: 0.2rem 0.45rem;
  color: var(--apple-muted);
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}

.available-price-cell {
  min-width: 7rem;
}

.available-price-cell strong {
  display: block;
  color: var(--apple-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.82rem;
  font-weight: 700;
  white-space: nowrap;
}

.available-price-cell span {
  display: block;
  margin-top: 0.18rem;
  color: var(--apple-muted);
  font-size: 0.66rem;
  white-space: nowrap;
}

.available-price-note {
  margin-top: 0.24rem !important;
  font-size: 0.7rem;
  color: var(--apple-muted-2);
}

@media (max-width: 640px) {
  .available-model-catalog {
    padding: 0.8rem;
  }

  .available-model-catalog__header {
    flex-direction: column;
  }

  .available-selected-rate {
    width: 100%;
    justify-content: space-between;
  }

  .available-group-tabs-shell {
    gap: 0.3rem;
  }

  .available-group-tabs-shell::before {
    left: 2.15rem;
  }

  .available-group-tabs-shell::after {
    right: 2.15rem;
  }

  .available-group-scroll-button {
    width: 1.85rem;
    height: 1.85rem;
  }

  .available-group-tab {
    min-width: 9.5rem;
  }

  .available-model-table-wrap {
    display: none;
  }

  .available-model-mobile {
    display: block;
    margin-top: 1rem;
    border-top: 1px solid var(--apple-border-soft);
  }

  .available-model-mobile__section {
    min-width: 0;
    border-bottom: 1px solid var(--apple-border-soft);
  }

  .available-model-mobile__section:last-child {
    border-bottom: 0;
  }

  .available-model-mobile__platform {
    display: flex;
    min-width: 0;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.65rem;
    padding: 0.75rem 0;
  }

  .available-model-mobile__platform-main {
    display: grid;
    min-width: 0;
    flex: 1;
    gap: 0.12rem;
  }

  .available-model-mobile__platform-main strong {
    overflow-wrap: anywhere;
    color: var(--apple-text);
    font-size: 0.84rem;
    font-weight: 750;
  }

  .available-model-mobile__platform-main span {
    color: var(--apple-muted);
    font-size: 0.68rem;
    font-weight: 600;
  }

  .available-model-mobile__endpoints {
    display: flex;
    width: 100%;
    min-width: 0;
    flex-wrap: wrap;
    gap: 0.3rem;
    padding-left: 2.65rem;
  }

  .available-model-mobile__endpoints span {
    max-width: 100%;
    overflow-wrap: anywhere;
    border: 1px solid var(--apple-border-soft);
    border-radius: 0.4rem;
    padding: 0.18rem 0.4rem;
    color: var(--apple-muted);
    font-size: 0.66rem;
    font-weight: 600;
  }

  .available-model-mobile__model {
    min-width: 0;
    padding: 0.8rem 0;
    border-top: 1px solid var(--apple-border-soft);
  }

  .available-model-mobile__model-header {
    display: flex;
    min-width: 0;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.75rem;
  }

  .available-model-mobile__prices {
    display: grid;
    min-width: 0;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.45rem;
    margin-top: 0.7rem;
  }

  .available-model-mobile__prices > div {
    min-width: 0;
    border-radius: 0.4rem;
    background: var(--apple-surface-elevated);
    padding: 0.5rem 0.55rem;
  }

  .available-model-mobile__prices dt {
    overflow-wrap: anywhere;
    color: var(--apple-muted);
    font-size: 0.66rem;
    font-weight: 600;
  }

  .available-model-mobile__prices dd {
    min-width: 0;
    margin-top: 0.22rem;
  }

  .available-model-mobile__prices dd strong {
    display: block;
    overflow-wrap: anywhere;
    color: var(--apple-text);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    font-size: 0.8rem;
    font-weight: 700;
  }

  .available-model-mobile__prices dd span {
    display: block;
    margin-top: 0.12rem;
    overflow-wrap: anywhere;
    color: var(--apple-muted);
    font-size: 0.62rem;
  }
}
</style>
