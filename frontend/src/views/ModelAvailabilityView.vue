<template>
  <AppLayout>
    <div class="avail-page px-2 md:px-6 py-4 sm:py-6 max-w-[1400px] mx-auto">

      <!-- ═══════════ HERO SECTION ═══════════ -->
      <div class="hero-card">
        <div class="hero-glow"></div>
        <div class="relative z-10 p-5 sm:p-8">
          <!-- Header row -->
          <div class="flex items-start sm:items-center justify-between gap-3">
            <div class="flex items-center gap-3 sm:gap-4">
              <!-- Animated radar icon -->
              <div class="hero-icon-wrap">
                <div class="hero-icon-ring"></div>
                <div class="hero-icon">
                  <svg class="w-5 h-5 sm:w-6 sm:h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9.348 14.652a3.75 3.75 0 010-5.304m5.304 0a3.75 3.75 0 010 5.304m-7.425 2.121a6.75 6.75 0 010-9.546m9.546 0a6.75 6.75 0 010 9.546M5.106 18.894c-3.808-3.807-3.808-9.98 0-13.788m13.788 0c3.808 3.807 3.808 9.98 0 13.788M12 12h.008v.008H12V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
                  </svg>
                </div>
              </div>
              <div>
                <h1 class="text-lg sm:text-xl font-bold text-gray-900 dark:text-white tracking-tight">
                  {{ t('availability.title') }}
                </h1>
                <p class="mt-0.5 text-xs sm:text-sm text-gray-500 dark:text-dark-400">
                  {{ t('availability.subtitle') }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <div v-if="!loading && groups.length > 0" class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200/60 dark:border-emerald-500/20">
                <span class="relative flex h-2 w-2">
                  <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                </span>
                <span class="text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">LIVE</span>
                <span class="text-[11px] text-emerald-500/70 dark:text-emerald-400/60">{{ lastUpdated }}</span>
              </div>
              <button @click="refresh" :disabled="loading"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-white dark:bg-dark-700 border border-gray-200 dark:border-dark-600 text-gray-600 dark:text-dark-300 hover:bg-gray-50 dark:hover:bg-dark-600 hover:border-gray-300 dark:hover:border-dark-500 transition-all duration-200 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm">
                <svg class="h-3.5 w-3.5" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" />
                </svg>
                {{ t('availability.refresh') }}
              </button>
            </div>
          </div>

          <!-- Overview stat cards -->
          <div v-if="!loading && groups.length > 0" class="mt-5 sm:mt-7 grid grid-cols-3 gap-1.5 sm:gap-4">
            <div class="stat-card">
              <div class="stat-value text-primary-600 dark:text-primary-400">{{ groups.length }}</div>
              <div class="stat-label">{{ t('availability.totalGroups') }}</div>
              <div class="stat-bar">
                <div class="stat-bar-fill bg-primary-400 dark:bg-primary-500" style="width:100%"></div>
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-value text-emerald-600 dark:text-emerald-400">{{ availableCount }}</div>
              <div class="stat-label text-emerald-600/70 dark:text-emerald-400/70">{{ t('availability.online') }}</div>
              <div class="stat-bar">
                <div class="stat-bar-fill bg-emerald-400 dark:bg-emerald-500" :style="{ width: uptimePercent + '%' }"></div>
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-value" :class="unavailableCount > 0 ? 'text-rose-600 dark:text-rose-400' : 'text-gray-300 dark:text-dark-600'">{{ unavailableCount }}</div>
              <div class="stat-label" :class="unavailableCount > 0 ? 'text-rose-600/70 dark:text-rose-400/70' : ''">{{ t('availability.offline') }}</div>
              <div class="stat-bar">
                <div class="stat-bar-fill bg-rose-400 dark:bg-rose-500" :style="{ width: groups.length ? ((unavailableCount / groups.length) * 100) + '%' : '0%' }"></div>
              </div>
            </div>
          </div>

          <!-- Uptime bar -->
          <div v-if="!loading && groups.length > 0" class="mt-4 sm:mt-6">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-dark-500">{{ t('availability.available') }}</span>
              <span class="text-sm font-bold tabular-nums" :class="uptimePercent >= 100 ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ uptimePercent }}%</span>
            </div>
            <div class="h-1.5 rounded-full bg-gray-100 dark:bg-dark-700 overflow-hidden">
              <div class="h-full rounded-full transition-all duration-1000 ease-out uptime-shimmer"
                :class="uptimePercent >= 100 ? 'bg-gradient-to-r from-primary-400 to-emerald-400 dark:from-primary-500 dark:to-emerald-500' : 'bg-gradient-to-r from-amber-400 to-amber-500 dark:from-amber-500 dark:to-amber-600'"
                :style="{ width: uptimePercent + '%' }"></div>
            </div>
          </div>

          <!-- Mobile live indicator -->
          <div v-if="!loading && groups.length > 0" class="mt-3 flex sm:hidden items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
            </span>
            <span class="text-[11px] text-gray-500 dark:text-dark-400">{{ lastUpdated }}</span>
          </div>
        </div>
      </div>

      <!-- ═══════════ PLATFORM FILTERS ═══════════ -->
      <div v-if="!loading && groups.length > 0" class="filter-scroll mb-4 sm:mb-5">
        <button @click="selectedPlatform = ''"
          class="filter-chip"
          :class="selectedPlatform === '' ? 'filter-chip-active' : 'filter-chip-idle'">
          {{ t('availability.all') }}
          <span class="chip-count" :class="selectedPlatform === '' ? 'chip-count-active' : ''">{{ groups.length }}</span>
        </button>
        <button v-for="p in platforms" :key="p.name" @click="selectedPlatform = p.name"
          class="filter-chip"
          :class="selectedPlatform === p.name ? 'filter-chip-active' : 'filter-chip-idle'">
          <span class="h-2 w-2 rounded-full flex-shrink-0" :style="{ background: platformColor(p.name) }"></span>
          {{ platformDisplayName(p.name) }}
          <span class="chip-count" :class="selectedPlatform === p.name ? 'chip-count-active' : ''">{{ p.count }}</span>
        </button>
      </div>

      <!-- ═══════════ LOADING STATE ═══════════ -->
      <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        <div v-for="i in 6" :key="i" class="skel-card" :style="{ animationDelay: `${i * 60}ms` }">
          <div class="skel-shimmer"></div>
          <div class="p-4 sm:p-5">
            <div class="flex items-center gap-3 mb-4">
              <div class="h-10 w-10 sm:h-12 sm:w-12 rounded-xl skel-block"></div>
              <div class="flex-1 space-y-2">
                <div class="h-4 w-28 rounded-lg skel-block"></div>
                <div class="h-3 w-20 rounded-lg skel-block"></div>
              </div>
            </div>
            <div class="space-y-2">
              <div class="h-8 w-full rounded-lg skel-block"></div>
              <div class="h-3 w-full rounded-lg skel-block"></div>
            </div>
          </div>
        </div>
      </div>

      <!-- ═══════════ ERROR STATE ═══════════ -->
      <div v-else-if="error" class="flex flex-col items-center justify-center py-16 px-4">
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200/60 dark:border-rose-500/20">
          <svg class="h-7 w-7 text-rose-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
        </div>
        <p class="text-sm text-gray-600 dark:text-dark-400 mb-4 mt-3">{{ error }}</p>
        <button @click="refresh"
          class="px-5 py-2 rounded-xl text-sm font-medium bg-primary-500 hover:bg-primary-600 text-white shadow-sm shadow-primary-500/20 transition-all duration-200">
          {{ t('availability.retry') }}
        </button>
      </div>

      <!-- ═══════════ EMPTY STATE ═══════════ -->
      <div v-else-if="groups.length === 0" class="flex flex-col items-center justify-center py-16 px-4">
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-700 border border-gray-200 dark:border-dark-600">
          <svg class="h-7 w-7 text-gray-400 dark:text-dark-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375" />
          </svg>
        </div>
        <p class="text-sm text-gray-500 dark:text-dark-400 mt-3">{{ t('availability.noGroups') }}</p>
      </div>

      <!-- ═══════════ GROUP CARDS (Relay-Pulse Style) ═══════════ -->
      <div v-if="!loading && !error && filteredGroups.length > 0"
        class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        <div v-for="(group, idx) in filteredGroups" :key="group.group_id"
          class="group avail-card"
          :style="{ animationDelay: `${idx * 40}ms` }">

          <div class="relative flex-1 p-4 sm:p-5">
            <!-- Header: Icon + Name + Status Dot + Latency info -->
            <div class="mb-4 sm:mb-5 grid grid-cols-[1fr_auto] gap-3">
              <!-- Left: Icon + Service info -->
              <div class="flex min-w-0 items-center gap-3">
                <div class="platform-icon-wrap"
                  :style="{ '--p-color': platformColor(group.platform), '--p-color-light': platformColor(group.platform) + '18' }">
                  <div class="scale-75 sm:scale-100">
                    <ModelIcon :model="platformToModel(group.platform)" size="26px" />
                  </div>
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <h3 class="flex-1 truncate text-sm font-semibold leading-tight text-gray-900 dark:text-white sm:text-base">
                      {{ group.group_name }}
                    </h3>
                  </div>
                  <div class="mt-1 flex items-center gap-2 text-xs flex-wrap">
                    <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium"
                      :style="{ background: platformColor(group.platform) + '15', color: platformColor(group.platform), border: '1px solid ' + platformColor(group.platform) + '30' }">
                      {{ platformDisplayName(group.platform) }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Right: Status dot -->
              <div class="flex flex-col items-end gap-1.5">
                <div class="flex items-center gap-1.5">
                  <!-- Status dot with glow -->
                  <span class="status-dot-lg" :class="group.available ? 'dot-available' : 'dot-unavailable'"></span>
                  <span class="status-label" :class="group.available ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'">
                    {{ group.available ? t('availability.online') : t('availability.offline') }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Heatmap bar (simulated from account ratio) -->
            <div class="mb-3">
              <div class="flex gap-[2px] w-full overflow-hidden rounded-sm h-8">
                <div v-for="(block, bIdx) in generateHeatmapBlocks(group)" :key="bIdx"
                  class="heatmap-block"
                  :style="{ width: `${100 / heatmapBlockCount}%`, backgroundColor: block.color }"
                  :title="block.label">
                </div>
              </div>
              <div class="flex justify-between mt-1">
                <span class="text-[10px] text-gray-400 dark:text-dark-500 font-mono">{{ t('availability.recent') }}</span>
                <span class="text-[10px] text-gray-400 dark:text-dark-500 font-mono">{{ t('availability.now') }}</span>
              </div>
            </div>

            <!-- Status indicator bar -->
            <div class="status-bar" :class="group.available ? 'status-bar-ok' : 'status-bar-err'">
              <div class="flex items-center gap-2">
                <svg class="h-3.5 w-3.5 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M16.247 7.761a6 6 0 0 1 0 8.478"/><path d="M19.075 4.933a10 10 0 0 1 0 14.134"/>
                  <path d="M4.925 19.067a10 10 0 0 1 0-14.134"/><path d="M7.753 16.239a6 6 0 0 1 0-8.478"/>
                  <circle cx="12" cy="12" r="2"/>
                </svg>
                <span class="text-[10px] font-semibold uppercase tracking-wider opacity-60">{{ t('availability.status') }}</span>
              </div>
              <div class="flex items-center gap-1.5">
                <span class="inline-flex h-1.5 w-1.5 rounded-full" :class="group.available ? 'bg-emerald-500' : 'bg-rose-500'"></span>
                <span class="text-xs font-bold">{{ group.available ? t('availability.online') : t('availability.offline') }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { getAvailability, type GroupAvailabilityItem } from '@/api/groups'

const { t } = useI18n()

const groups = ref<GroupAvailabilityItem[]>([])
const loading = ref(true)
const error = ref('')
const selectedPlatform = ref('')
const lastUpdated = ref('')
let autoRefreshTimer: ReturnType<typeof setInterval> | null = null

const heatmapBlockCount = 30

const availableCount = computed(() => groups.value.filter(g => g.available).length)
const unavailableCount = computed(() => groups.value.filter(g => !g.available).length)
const uptimePercent = computed(() => {
  if (groups.value.length === 0) return 100
  return Math.round((availableCount.value / groups.value.length) * 100)
})

const platforms = computed(() => {
  const map = new Map<string, number>()
  for (const g of groups.value) {
    map.set(g.platform, (map.get(g.platform) || 0) + 1)
  }
  return Array.from(map.entries())
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
})

const filteredGroups = computed(() => {
  if (!selectedPlatform.value) return groups.value
  return groups.value.filter(g => g.platform === selectedPlatform.value)
})

const PLATFORM_COLORS: Record<string, string> = {
  anthropic: '#d97706', openai: '#6366f1', gemini: '#4285f4',
  antigravity: '#8b5cf6', sora: '#ec4899', qwen: '#615eff',
  deepseek: '#4d6bfe', glm: '#3859ff', kimi: '#a855f7', iflow: '#10b981'
}
const PLATFORM_MODEL_MAP: Record<string, string> = {
  anthropic: 'claude-3', openai: 'gpt-4', gemini: 'gemini-pro',
  antigravity: 'antigravity', sora: 'sora', qwen: 'qwen-max',
  deepseek: 'deepseek-chat', glm: 'glm-4', kimi: 'moonshot-v1', iflow: 'iflow'
}

function platformDisplayName(p: string): string {
  const m: Record<string, string> = {
    anthropic:'Anthropic', openai:'OpenAI', gemini:'Gemini', antigravity:'Antigravity',
    sora:'Sora', qwen:'Qwen', deepseek:'DeepSeek', glm:'GLM', kimi:'Kimi', iflow:'iFlow'
  }
  return m[p] || p
}
function platformColor(p: string): string { return PLATFORM_COLORS[p] || '#6b7280' }
function platformToModel(p: string): string { return PLATFORM_MODEL_MAP[p] || p }

// Generate simulated heatmap blocks based on current account ratio
function generateHeatmapBlocks(group: GroupAvailabilityItem): { color: string; label: string }[] {
  const blocks: { color: string; label: string }[] = []
  const ratio = group.total_accounts > 0 ? group.active_accounts / group.total_accounts : 0

  for (let i = 0; i < heatmapBlockCount; i++) {
    // Last block represents current status, earlier blocks simulate slight variance
    const isRecent = i >= heatmapBlockCount - 3
    const variance = isRecent ? 0 : (Math.random() - 0.5) * 0.15
    const effectiveRatio = Math.max(0, Math.min(1, ratio + variance))

    let color: string
    let label: string

    if (effectiveRatio >= 0.8) {
      // Green - all good
      const greenIntensity = 0.6 + effectiveRatio * 0.4
      color = `rgba(34, 197, 94, ${greenIntensity})`
      label = `${Math.round(effectiveRatio * 100)}% ${t('availability.available')}`
    } else if (effectiveRatio >= 0.4) {
      // Yellow/amber - degraded
      const yellowIntensity = 0.6 + effectiveRatio * 0.4
      color = `rgba(234, 179, 8, ${yellowIntensity})`
      label = `${Math.round(effectiveRatio * 100)}% ${t('availability.available')}`
    } else if (effectiveRatio > 0) {
      // Orange/red - critical
      color = `rgba(239, 68, 68, 0.7)`
      label = `${Math.round(effectiveRatio * 100)}% ${t('availability.available')}`
    } else {
      // Red - unavailable
      color = `rgba(239, 68, 68, 0.85)`
      label = t('availability.offline')
    }

    blocks.push({ color, label })
  }
  return blocks
}

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    groups.value = await getAvailability()
    lastUpdated.value = new Date().toLocaleTimeString()
  } catch (e: any) {
    error.value = e?.message || 'Failed to load availability data'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  // Auto-refresh every 60s
  autoRefreshTimer = setInterval(() => {
    if (!document.hidden) {
      refresh()
    }
  }, 60000)
})

onUnmounted(() => {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
})
</script>

<style scoped>
/* ═══════════ HERO CARD ═══════════ */
.hero-card {
  position: relative;
  overflow: hidden;
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(229, 231, 235, 0.6);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  margin-bottom: 1.25rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.06);
}
:root.dark .hero-card {
  background: rgba(30, 41, 59, 0.5);
  border-color: rgba(51, 65, 85, 0.5);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}
.hero-glow {
  position: absolute;
  top: -50%;
  right: -20%;
  width: 60%;
  height: 200%;
  background: radial-gradient(ellipse, rgba(20, 184, 166, 0.06), transparent 70%);
  pointer-events: none;
}
:root.dark .hero-glow {
  background: radial-gradient(ellipse, rgba(20, 184, 166, 0.08), transparent 70%);
}
@media (min-width: 640px) { .hero-card { margin-bottom: 1.5rem; } }

/* Hero icon */
.hero-icon-wrap {
  position: relative;
  flex-shrink: 0;
}
.hero-icon {
  width: 44px;
  height: 44px;
  border-radius: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(20, 184, 166, 0.1), rgba(16, 185, 129, 0.08));
  color: #14b8a6;
  border: 1px solid rgba(20, 184, 166, 0.2);
  position: relative;
  z-index: 2;
}
:root.dark .hero-icon {
  background: linear-gradient(135deg, rgba(20, 184, 166, 0.15), rgba(16, 185, 129, 0.1));
  border-color: rgba(20, 184, 166, 0.25);
}
@media (min-width: 640px) { .hero-icon { width: 52px; height: 52px; } }
.hero-icon-ring {
  position: absolute;
  inset: -4px;
  border-radius: 1rem;
  border: 1.5px solid rgba(20, 184, 166, 0.15);
  animation: icon-ring-pulse 3s ease-in-out infinite;
}
@keyframes icon-ring-pulse {
  0%, 100% { opacity: 0.3; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(1.05); }
}

/* ═══════════ STAT CARDS ═══════════ */
.stat-card {
  padding: 10px 6px;
  border-radius: 0.75rem;
  background: rgba(249, 250, 251, 0.8);
  border: 1px solid rgba(229, 231, 235, 0.5);
  text-align: center;
  transition: all 0.2s;
  min-width: 0;
}
:root.dark .stat-card {
  background: rgba(30, 41, 59, 0.4);
  border-color: rgba(51, 65, 85, 0.4);
}
.stat-card:hover {
  border-color: rgba(20, 184, 166, 0.3);
  box-shadow: 0 0 20px rgba(20, 184, 166, 0.06);
}
@media (min-width: 640px) { .stat-card { padding: 16px 20px; } }
.stat-value {
  font-size: 1.25rem;
  font-weight: 800;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
@media (min-width: 640px) { .stat-value { font-size: 2.25rem; } }
.stat-label {
  font-size: 10px;
  font-weight: 600;
  color: rgba(107, 114, 128, 0.7);
  margin-top: 4px;
  line-height: 1.3;
}
:root.dark .stat-label { color: rgba(148, 163, 184, 0.6); }
@media (min-width: 640px) { .stat-label { font-size: 11px; margin-top: 6px; } }
.stat-bar {
  height: 3px;
  background: rgba(229, 231, 235, 0.5);
  margin-top: 10px;
  border-radius: 2px;
  overflow: hidden;
}
:root.dark .stat-bar { background: rgba(51, 65, 85, 0.4); }
.stat-bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 1.2s cubic-bezier(0.4, 0, 0.2, 1);
}

/* Uptime shimmer */
.uptime-shimmer {
  position: relative;
}
.uptime-shimmer::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent 30%, rgba(255, 255, 255, 0.4) 50%, transparent 70%);
  animation: shimmer-move 2.5s ease-in-out infinite;
}
@keyframes shimmer-move {
  0% { transform: translateX(-150%); }
  100% { transform: translateX(250%); }
}

/* ═══════════ FILTER CHIPS ═══════════ */
.filter-scroll {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  padding-bottom: 2px;
}
.filter-scroll::-webkit-scrollbar { display: none; }
@media (min-width: 640px) { .filter-scroll { flex-wrap: wrap; overflow-x: visible; } }

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
  white-space: nowrap;
  flex-shrink: 0;
}
.filter-chip-active {
  background: rgba(20, 184, 166, 0.1);
  color: #0d9488;
  border-color: rgba(20, 184, 166, 0.3);
  font-weight: 600;
}
:root.dark .filter-chip-active {
  background: rgba(20, 184, 166, 0.15);
  color: #5eead4;
  border-color: rgba(20, 184, 166, 0.3);
}
.filter-chip-idle {
  background: rgba(255, 255, 255, 0.6);
  color: rgba(107, 114, 128, 0.8);
  border-color: rgba(229, 231, 235, 0.5);
}
:root.dark .filter-chip-idle {
  background: rgba(30, 41, 59, 0.4);
  color: rgba(148, 163, 184, 0.7);
  border-color: rgba(51, 65, 85, 0.4);
}
.filter-chip-idle:hover {
  background: rgba(20, 184, 166, 0.05);
  color: #0d9488;
  border-color: rgba(20, 184, 166, 0.2);
}
:root.dark .filter-chip-idle:hover {
  background: rgba(20, 184, 166, 0.08);
  color: #5eead4;
  border-color: rgba(20, 184, 166, 0.2);
}
.chip-count {
  font-size: 0.65rem;
  padding: 0 6px;
  border-radius: 9999px;
  line-height: 1.6;
  background: rgba(0, 0, 0, 0.04);
  font-weight: 600;
}
:root.dark .chip-count { background: rgba(255, 255, 255, 0.06); }
.chip-count-active {
  background: rgba(20, 184, 166, 0.15);
  color: #0d9488;
}
:root.dark .chip-count-active {
  background: rgba(20, 184, 166, 0.2);
  color: #5eead4;
}

/* ═══════════ SKELETON LOADING ═══════════ */
.skel-card {
  border-radius: 1rem;
  border: 1px solid rgba(229, 231, 235, 0.5);
  background: rgba(255, 255, 255, 0.5);
  position: relative;
  overflow: hidden;
  animation: card-appear 0.4s ease-out both;
}
:root.dark .skel-card {
  border-color: rgba(51, 65, 85, 0.4);
  background: rgba(30, 41, 59, 0.3);
}
.skel-shimmer {
  position: absolute;
  top: 0;
  left: -100%;
  width: 50%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(20, 184, 166, 0.04), transparent);
  animation: skel-sweep 2s ease-in-out infinite;
}
@keyframes skel-sweep {
  0% { left: -50%; }
  100% { left: 150%; }
}
.skel-block {
  background: rgba(229, 231, 235, 0.5);
}
:root.dark .skel-block { background: rgba(51, 65, 85, 0.4); }

/* ═══════════ GROUP CARDS (Relay-Pulse Inspired) ═══════════ */
.avail-card {
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(229, 231, 235, 0.5);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  animation: card-appear 0.4s ease-out both;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}
:root.dark .avail-card {
  background: rgba(30, 41, 59, 0.5);
  border-color: rgba(51, 65, 85, 0.5);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}
@keyframes card-appear {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}
.avail-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.08);
  border-color: rgba(20, 184, 166, 0.25);
}
:root.dark .avail-card:hover {
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
  border-color: rgba(20, 184, 166, 0.3);
  background: rgba(30, 41, 59, 0.7);
}

/* Platform icon */
.platform-icon-wrap {
  width: 40px;
  height: 40px;
  border-radius: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--p-color-light, rgba(249, 250, 251, 1));
  border: 1px solid rgba(229, 231, 235, 0.5);
  flex-shrink: 0;
  transition: all 0.3s;
}
:root.dark .platform-icon-wrap {
  background: rgba(51, 65, 85, 0.4);
  border-color: rgba(71, 85, 105, 0.4);
}
@media (min-width: 640px) { .platform-icon-wrap { width: 48px; height: 48px; } }
.group:hover .platform-icon-wrap {
  border-color: var(--p-color, rgba(20, 184, 166, 0.3));
  box-shadow: 0 0 12px var(--p-color-light, rgba(20, 184, 166, 0.1));
  transform: scale(1.05);
}

/* Status dot (large, with glow) */
.status-dot-lg {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.dot-available {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
  animation: dot-pulse 2s ease-in-out infinite;
}
.dot-unavailable {
  background: #f43f5e;
  box-shadow: 0 0 8px rgba(244, 63, 94, 0.5);
  animation: dot-blink 1.5s ease-in-out infinite;
}
@keyframes dot-pulse {
  0%, 100% { box-shadow: 0 0 4px rgba(16, 185, 129, 0.4); }
  50% { box-shadow: 0 0 12px rgba(16, 185, 129, 0.6); }
}
@keyframes dot-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.status-label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ═══════════ HEATMAP BLOCKS ═══════════ */
.heatmap-block {
  border-radius: 2px;
  transition: all 0.2s;
  cursor: default;
}
.heatmap-block:hover {
  transform: scaleY(1.15);
  opacity: 1;
  z-index: 10;
  box-shadow: 0 0 6px rgba(0,0,0,0.15);
}

/* Status bar */
.status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 0.625rem;
  transition: all 0.2s;
}
.status-bar-ok {
  background: rgba(16, 185, 129, 0.05);
  color: #059669;
  border: 1px solid rgba(16, 185, 129, 0.1);
}
:root.dark .status-bar-ok {
  background: rgba(16, 185, 129, 0.06);
  color: #34d399;
  border-color: rgba(16, 185, 129, 0.1);
}
.status-bar-err {
  background: rgba(244, 63, 94, 0.05);
  color: #e11d48;
  border: 1px solid rgba(244, 63, 94, 0.1);
}
:root.dark .status-bar-err {
  background: rgba(244, 63, 94, 0.06);
  color: #fb7185;
  border-color: rgba(244, 63, 94, 0.1);
}
.group:hover .status-bar-ok {
  background: rgba(16, 185, 129, 0.08);
  border-color: rgba(16, 185, 129, 0.15);
}
.group:hover .status-bar-err {
  background: rgba(244, 63, 94, 0.08);
  border-color: rgba(244, 63, 94, 0.15);
}
</style>
