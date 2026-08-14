<template>
  <div class="public-gateway-shell">
    <PublicGatewayHeader />

    <main class="public-gateway-container pb-16">
      <section class="public-gateway-hero">
        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.45fr)] lg:items-end">
          <div class="min-w-0">
            <p class="public-gateway-kicker">{{ copy.kicker }}</p>
            <h1 class="public-gateway-title">{{ copy.title }}</h1>
            <p class="public-gateway-lead">{{ copy.lead }}</p>
            <div class="mt-5 flex flex-wrap gap-2">
              <span v-for="item in trustSignals" :key="item" class="public-gateway-chip">
                {{ item }}
              </span>
            </div>
          </div>

          <aside class="public-gateway-panel p-4">
            <div class="grid grid-cols-2 gap-2">
              <div v-for="item in summaryItems" :key="item.label" class="public-gateway-stat">
                <p>{{ item.label }}</p>
                <strong>{{ item.value }}</strong>
              </div>
            </div>
            <p class="mt-3 text-xs leading-5 text-[var(--gw-text-3)]">
              {{ copy.factSource }}
            </p>
          </aside>
        </div>
      </section>

      <section class="public-gateway-panel mb-5 p-3 sm:p-4">
        <div class="grid gap-3 xl:grid-cols-[minmax(260px,0.7fr)_minmax(760px,1.3fr)] xl:items-start">
          <div class="relative min-w-0">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--gw-text-3)]" />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="copy.search"
              class="input h-11 pl-10"
            />
          </div>

          <div class="grid min-w-0 gap-2 sm:grid-cols-2 lg:grid-cols-5">
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.platformFilter }}</span>
              <select v-model="platformFilter" class="input h-11">
                <option v-for="option in platformFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.groupFilter }}</span>
              <select v-model="groupFilter" class="input h-11">
                <option v-for="option in groupFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.endpointFilter }}</span>
              <select v-model="endpointFilter" class="input h-11">
                <option v-for="option in endpointFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.pricingFilter }}</span>
              <select v-model="pricingFilter" class="input h-11">
                <option v-for="option in pricingFilterOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
            <label class="min-w-0">
              <span class="mb-1 block text-xs font-semibold text-[var(--gw-text-3)]">{{ copy.sortBy }}</span>
              <select v-model="sortBy" class="input h-11">
                <option v-for="option in sortOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-[color:var(--gw-border)] pt-3">
          <div class="flex flex-wrap gap-2 text-xs font-semibold text-[var(--gw-text-3)]">
            <span class="public-gateway-chip">{{ copy.filteredGroups }} {{ summary.groups }}</span>
            <span class="public-gateway-chip">{{ copy.filteredModels }} {{ summary.models }}</span>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="public-gateway-secondary rounded-lg" type="button" @click="resetFilters">
              {{ copy.reset }}
            </button>
            <button class="public-gateway-secondary rounded-lg" :disabled="loading" type="button" @click="loadChannels">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ copy.refresh }}
            </button>
          </div>
        </div>
      </section>

      <section v-if="errorMessage" class="public-gateway-alert mb-5">
        {{ errorMessage }}
      </section>

      <AvailableChannelsTable
        :rows="filteredChannels"
        :loading="loading"
        :user-group-rates="{}"
        pricing-key-prefix="availableChannels.pricing"
        :no-pricing-label="copy.noPricing"
        :no-models-label="copy.noModels"
        :empty-label="errorMessage || copy.empty"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AvailableChannelsTable from '@/custom/channels/WegooAvailableChannelsTable.vue'
import Icon from '@/components/icons/Icon.vue'
import publicGatewayAPI from '@/custom/api/publicGateway'
import type { UserAvailableChannel, UserAvailableGroup } from '@/api/channels'
import PublicGatewayHeader from './PublicGatewayHeader.vue'
import './publicGateway.css'

const { locale } = useI18n()

const channels = ref<UserAvailableChannel[]>([])
const loading = ref(false)
const errorMessage = ref('')
const searchQuery = ref('')
const platformFilter = ref('all')
const groupFilter = ref('all')
const endpointFilter = ref('all')
const pricingFilter = ref('all')
const sortBy = ref('name')

const copy = computed(() => locale.value.startsWith('zh')
  ? {
    kicker: 'Model Pricing · Public Catalog',
    title: '公开模型与价格',
    lead: '这里展示公开可用的模型分组、价格倍率和模型能力。登录后可查看账户可用的完整分组和专属价格。',
    factSource: '价格和模型信息会随服务配置更新，实际可用范围以登录后的控制台展示为准。',
    search: '搜索模型或分组',
    refresh: '刷新',
    noPricing: '暂无公开价格',
    noModels: '该分组暂未公开模型清单',
    empty: '暂无公开模型数据',
    loadFailed: '公开模型接口暂时不可用，请稍后刷新；登录控制台可查看完整授权分组。',
    channels: '渠道',
    platforms: '平台',
    groups: '公开分组',
    models: '模型',
    allPlatforms: '全部平台',
    platformFilter: '平台',
    groupFilter: '分组',
    endpointFilter: '端点',
    pricingFilter: '价格能力',
    sortBy: '排序',
    reset: '重置筛选',
    filteredGroups: '当前分组',
    filteredModels: '当前模型',
    allGroups: '全部分组',
    allEndpoints: '全部端点',
    allPricing: '全部价格',
    tokenPricing: 'Token 计费',
    cachePricing: 'Cache 计费',
    requestPricing: '按次 / 生图',
    sortName: '名称',
    sortRate: '分组倍率',
    sortModelCount: '模型数',
    sortInput: '输入价',
    sortOutput: '输出价',
    sortCacheRead: 'Cache Read',
    sortCacheWrite: 'Cache Write',
  }
  : {
    kicker: 'Model Pricing · Public Catalog',
    title: 'Public Models and Pricing',
    lead: 'This page only shows public standard groups. Authorized groups, private groups, and user-specific rates remain available in the console.',
    factSource: 'Pricing and model information update with service configuration. Sign in to see the full catalog available to your account.',
    search: 'Search models or groups',
    refresh: 'Refresh',
    noPricing: 'No public price',
    noModels: 'No public model list yet',
    empty: 'No public model data yet',
    loadFailed: 'The public model endpoint is temporarily unavailable. Refresh later or sign in for the full authorized catalog.',
    channels: 'Channels',
    platforms: 'Platforms',
    groups: 'Public Groups',
    models: 'Models',
    allPlatforms: 'All Platforms',
    platformFilter: 'Provider',
    groupFilter: 'Group',
    endpointFilter: 'Endpoint',
    pricingFilter: 'Pricing',
    sortBy: 'Sort',
    reset: 'Reset Filters',
    filteredGroups: 'Groups',
    filteredModels: 'Models',
    allGroups: 'All Groups',
    allEndpoints: 'All Endpoints',
    allPricing: 'All Pricing',
    tokenPricing: 'Token Pricing',
    cachePricing: 'Cache Pricing',
    requestPricing: 'Per-request / Image',
    sortName: 'Name',
    sortRate: 'Group Rate',
    sortModelCount: 'Model Count',
    sortInput: 'Input Price',
    sortOutput: 'Output Price',
    sortCacheRead: 'Cache Read',
    sortCacheWrite: 'Cache Write',
  })

const trustSignals = computed(() => locale.value.startsWith('zh')
  ? ['公开价格', '公开分组', '模型能力', '缓存价格', '登录查看更多']
  : ['Public pricing', 'Public groups', 'Model capabilities', 'Cache pricing', 'Sign in for more'])

const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return sortChannels(channels.value
    .map((channel) => {
      const channelMatches = q
        ? channel.name.toLowerCase().includes(q) || (channel.description || '').toLowerCase().includes(q)
        : false
      const platforms = channel.platforms
        .filter((section) => platformFilter.value === 'all' || section.platform === platformFilter.value)
        .filter((section) => endpointFilter.value === 'all' || sectionEndpointValues(section).includes(endpointFilter.value))
        .map((section) => {
          const sectionMatches = q ? section.platform.toLowerCase().includes(q) : false
          const groups = section.groups
            .map((group) => filterGroup(group, section.platform, q, channelMatches || sectionMatches))
            .filter((group): group is UserAvailableGroup => group !== null)
          return groups.length > 0 ? { ...section, groups, supported_models: sectionSupportedModels(groups, section.platform) } : null
        })
        .filter((section): section is UserAvailableChannel['platforms'][number] => section !== null)
      return platforms.length > 0 ? { ...channel, platforms } : null
    })
    .filter((channel): channel is UserAvailableChannel => channel !== null))
})

const groupFilterOptions = computed(() => [
  { value: 'all', label: copy.value.allGroups },
  ...Array.from(new Set(channels.value.flatMap((channel) =>
    channel.platforms.flatMap((section) => section.groups.map((group) => group.name)),
  )))
    .sort((a, b) => a.localeCompare(b))
    .map((name) => ({ value: name, label: name })),
])

const pricingFilterOptions = computed(() => [
  { value: 'all', label: copy.value.allPricing },
  { value: 'token', label: copy.value.tokenPricing },
  { value: 'cache', label: copy.value.cachePricing },
  { value: 'request', label: copy.value.requestPricing },
])

const endpointFilterOptions = computed(() => [
  { value: 'all', label: copy.value.allEndpoints },
  ...Array.from(new Set(channels.value.flatMap((channel) =>
    channel.platforms.flatMap((section) => sectionEndpointValues(section)),
  )))
    .sort((a, b) => a.localeCompare(b))
    .map((endpoint) => ({ value: endpoint, label: endpointLabel(endpoint) })),
])

const sortOptions = computed(() => [
  { value: 'name', label: copy.value.sortName },
  { value: 'rate', label: copy.value.sortRate },
  { value: 'modelCount', label: copy.value.sortModelCount },
  { value: 'input', label: copy.value.sortInput },
  { value: 'output', label: copy.value.sortOutput },
  { value: 'cacheRead', label: copy.value.sortCacheRead },
  { value: 'cacheWrite', label: copy.value.sortCacheWrite },
])

function filterGroup(
  group: UserAvailableGroup,
  platform: string,
  query: string,
  parentMatches: boolean,
): UserAvailableGroup | null {
  if (groupFilter.value !== 'all' && group.name !== groupFilter.value) return null
  const groupMatches = query ? group.name.toLowerCase().includes(query) : false
  const models = sortModels(groupModels(group).filter((model) => {
    if (!matchesPricingFilter(model)) return false
    if (!query || parentMatches || groupMatches) return true
    return model.name.toLowerCase().includes(query) || (model.platform || platform).toLowerCase().includes(query)
  }))
  if (models.length === 0) return null
  return { ...group, supported_models: models }
}

function matchesPricingFilter(model: ReturnType<typeof groupModels>[number]): boolean {
  const pricing = model.pricing
  if (pricingFilter.value === 'all') return true
  if (!pricing) return false
  if (pricingFilter.value === 'cache') {
    return pricing.cache_read_price !== null || pricing.cache_write_price !== null ||
      pricing.intervals.some((interval) => interval.cache_read_price !== null || interval.cache_write_price !== null)
  }
  if (pricingFilter.value === 'token') {
    return pricing.input_price !== null || pricing.output_price !== null ||
      pricing.intervals.some((interval) => interval.input_price !== null || interval.output_price !== null)
  }
  if (pricingFilter.value === 'request') {
    return pricing.per_request_price !== null || pricing.image_output_price !== null ||
      pricing.intervals.some((interval) => interval.per_request_price !== null)
  }
  return true
}

function sectionSupportedModels(groups: UserAvailableGroup[], platform: string) {
  const seen = new Set<string>()
  const models = groups.flatMap((group) => groupModels(group))
  return sortModels(models.filter((model) => {
    const key = `${model.platform || platform}:${model.name.toLowerCase()}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  }))
}

function sortChannels(rows: UserAvailableChannel[]) {
  return rows
    .map((channel) => ({
      ...channel,
      platforms: channel.platforms.map((section) => ({
        ...section,
        groups: sortGroups(section.groups),
      })),
    }))
    .sort((a, b) => {
      if (sortBy.value === 'name') return compareNames(a.name, b.name)
      return minChannelSortValue(a) - minChannelSortValue(b)
    })
}

function sortGroups(groups: UserAvailableGroup[]) {
  return [...groups].sort((a, b) => {
    if (sortBy.value === 'name') return compareNames(a.name, b.name)
    if (sortBy.value === 'rate') return effectiveRate(a) - effectiveRate(b)
    if (sortBy.value === 'modelCount') return groupModels(b).length - groupModels(a).length
    return minGroupSortValue(a) - minGroupSortValue(b)
  })
}

function sortModels(models: ReturnType<typeof groupModels>) {
  return [...models].sort((a, b) => {
    if (sortBy.value === 'name' || sortBy.value === 'rate' || sortBy.value === 'modelCount') {
      return compareNames(a.name, b.name)
    }
    return modelPriceSortValue(a) - modelPriceSortValue(b) || compareNames(a.name, b.name)
  })
}

function compareNames(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
}

function minChannelSortValue(channel: UserAvailableChannel): number {
  const values = channel.platforms.flatMap((section) => section.groups.map(minGroupSortValue))
  return values.length > 0 ? Math.min(...values) : Number.MAX_SAFE_INTEGER
}

function minGroupSortValue(group: UserAvailableGroup): number {
  if (sortBy.value === 'rate') return effectiveRate(group)
  if (sortBy.value === 'modelCount') return -groupModels(group).length
  const values = groupModels(group).map(modelPriceSortValue)
  return values.length > 0 ? Math.min(...values) : Number.MAX_SAFE_INTEGER
}

function effectiveRate(group: UserAvailableGroup): number {
  return group.discounted_rate_multiplier ?? group.rate_multiplier ?? Number.MAX_SAFE_INTEGER
}

function modelPriceSortValue(model: ReturnType<typeof groupModels>[number]): number {
  const pricing = model.pricing
  if (!pricing) return Number.MAX_SAFE_INTEGER
  const direct = (() => {
    switch (sortBy.value) {
      case 'input':
        return pricing.input_price
      case 'output':
        return pricing.output_price
      case 'cacheRead':
        return pricing.cache_read_price
      case 'cacheWrite':
        return pricing.cache_write_price
      default:
        return null
      }
  })()
  if (direct !== null && direct !== undefined) return direct
  const intervalValues = pricing.intervals
    .map((interval) => {
      switch (sortBy.value) {
        case 'input':
          return interval.input_price
        case 'output':
          return interval.output_price
        case 'cacheRead':
          return interval.cache_read_price
        case 'cacheWrite':
          return interval.cache_write_price
        default:
          return null
      }
    })
    .filter((value): value is number => typeof value === 'number')
  return intervalValues.length > 0 ? Math.min(...intervalValues) : Number.MAX_SAFE_INTEGER
}

const platformFilterOptions = computed(() => [
  { value: 'all', label: copy.value.allPlatforms },
  ...Array.from(new Set(channels.value.flatMap((channel) => channel.platforms.map((section) => section.platform))))
    .sort()
    .map((platform) => ({ value: platform, label: platformLabel(platform) })),
])

const summary = computed(() => {
  const rows = filteredChannels.value
  const groupCount = rows.reduce(
    (sum, channel) => sum + channel.platforms.reduce((n, section) => n + section.groups.length, 0),
    0,
  )
  const modelNames = new Set<string>()
  for (const channel of rows) {
    for (const section of channel.platforms) {
      for (const group of section.groups) {
        for (const model of groupModels(group)) {
          modelNames.add(`${model.platform || section.platform}:${model.name.toLowerCase()}`)
        }
      }
    }
  }
  return {
    channels: rows.length,
    platforms: rows.reduce((sum, channel) => sum + channel.platforms.length, 0),
    groups: groupCount,
    models: modelNames.size,
  }
})

const summaryItems = computed(() => [
  { label: copy.value.channels, value: summary.value.channels },
  { label: copy.value.platforms, value: summary.value.platforms },
  { label: copy.value.groups, value: summary.value.groups },
  { label: copy.value.models, value: summary.value.models },
])

function groupModels(group: UserAvailableGroup) {
  return group.supported_models || []
}

function platformLabel(platform: string): string {
  switch (platform) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    default:
      return platform
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

function sectionEndpointValues(section: UserAvailableChannel['platforms'][number]): string[] {
  if (section.endpoints && section.endpoints.length > 0) {
    return section.endpoints
  }
  if (section.supported_endpoint_types && section.supported_endpoint_types.length > 0) {
    return section.supported_endpoint_types
  }
  return platformEndpoints(section.platform)
}

function endpointLabel(endpoint: string): string {
  return endpoint
}

function resetFilters() {
  searchQuery.value = ''
  platformFilter.value = 'all'
  groupFilter.value = 'all'
  endpointFilter.value = 'all'
  pricingFilter.value = 'all'
  sortBy.value = 'name'
}

async function loadChannels() {
  loading.value = true
  errorMessage.value = ''
  try {
    channels.value = await publicGatewayAPI.getPublicAvailableChannels()
  } catch {
    errorMessage.value = copy.value.loadFailed
    channels.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadChannels()
})
</script>
