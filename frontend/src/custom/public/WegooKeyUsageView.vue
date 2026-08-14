<template>
  <div class="gateway-key-usage relative flex min-h-screen flex-col bg-[var(--apple-bg)] text-[var(--apple-text)]">
    <!-- Header (same pattern as HomeView) -->
    <header class="relative z-20 border-b border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-4 py-4 sm:px-6">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <router-link to="/home" class="flex items-center gap-3">
          <div class="h-9 w-9 overflow-hidden rounded-lg bg-[var(--apple-surface)] ring-1 ring-[color:var(--apple-border)]">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="truncate text-base font-semibold tracking-normal text-[var(--apple-text)]">{{ siteName }}</span>
        </router-link>
        <div class="flex items-center gap-3">
          <LocaleSwitcher />
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="rounded-md p-2 text-[var(--apple-muted)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]"
            :title="t('home.viewDocs')"
            @click="handleDocsLinkClick"
          >
            <Icon name="book" size="md" />
          </a>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="mx-auto w-full max-w-6xl flex-1 px-4 py-10 sm:px-6 sm:py-14">
      <section class="mb-12 grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(360px,440px)] lg:items-center">
        <div class="min-w-0 text-center lg:text-left">
          <p class="inline-flex rounded-full border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] px-3 py-1 text-xs font-semibold text-[var(--apple-muted)]">
            {{ t('keyUsage.heroKicker') }}
          </p>
          <h1 class="mx-auto mt-5 max-w-3xl text-4xl font-semibold leading-tight tracking-normal text-[var(--apple-text)] sm:text-5xl lg:mx-0 lg:text-6xl">
            {{ t('keyUsage.title') }}
          </h1>
          <p class="mx-auto mt-4 max-w-2xl text-base leading-7 text-[var(--apple-muted)] sm:text-lg lg:mx-0">
            {{ t('keyUsage.subtitle') }}
          </p>

          <div class="mt-7 grid gap-2 text-left sm:grid-cols-3">
            <div
              v-for="item in keyUsageTrustNotes"
              :key="item.title"
              class="rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] px-3 py-2.5"
            >
              <div class="flex items-center gap-2 text-xs font-semibold text-[var(--apple-text)]">
                <Icon :name="item.icon" size="xs" class="text-[var(--apple-blue)]" />
                <span>{{ item.title }}</span>
              </div>
              <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
                {{ item.description }}
              </p>
            </div>
          </div>
        </div>

        <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm sm:p-5">
          <div class="mb-4">
            <h2 class="text-base font-semibold text-[var(--apple-text)]">
              {{ t('keyUsage.queryPanel.title') }}
            </h2>
            <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
              {{ t('keyUsage.queryPanel.description') }}
            </p>
          </div>

          <div class="flex flex-col gap-3">
            <div class="relative min-w-0">
              <div class="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--apple-muted-2)]">
                <Icon name="lock" size="md" />
              </div>
              <input
                v-model="apiKey"
                :type="keyVisible ? 'text' : 'password'"
                :placeholder="t('keyUsage.placeholder')"
                class="input h-12 pl-12 pr-12"
                @keydown.enter="queryKey"
              />
              <button
                @click="keyVisible = !keyVisible"
                class="absolute right-4 top-1/2 -translate-y-1/2 text-[var(--apple-muted-2)] transition-colors hover:text-[var(--apple-text)]"
                :aria-label="keyVisible ? t('auth.hidePassword') : t('auth.showPassword')"
                type="button"
              >
                <Icon v-if="keyVisible" name="eyeOff" size="md" />
                <Icon v-else name="eye" size="md" />
              </button>
            </div>
            <button
              @click="queryKey"
              :disabled="isQuerying"
              class="btn btn-primary h-12 w-full justify-center whitespace-nowrap px-6"
            >
              <svg v-if="isQuerying" class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" opacity="0.25"/>
                <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
              </svg>
              <Icon v-else name="search" size="sm" :stroke-width="2.5" />
              {{ isQuerying ? t('keyUsage.querying') : t('keyUsage.query') }}
            </button>
          </div>

          <p class="mt-3 text-xs leading-5 text-[var(--apple-muted-2)]">
            {{ t('keyUsage.privacyNote') }}
          </p>

          <div class="mt-4 w-full sm:w-44">
            <Select
              :model-value="settlementCurrency"
              :options="settlementCurrencyOptions"
              :placeholder="t('settlementCurrency.label')"
              @update:model-value="setSettlementCurrency"
            />
          </div>

          <div v-if="showDatePicker" class="mt-4 border-t border-[color:var(--apple-border-soft)] pt-4">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-xs font-medium text-[var(--apple-muted)]">{{ t('keyUsage.dateRange') }}</span>
              <button
                v-for="range in dateRanges"
                :key="range.key"
                @click="setDateRange(range.key)"
                class="rounded-md border px-3 py-1.5 text-xs transition-all"
                :class="currentRange === range.key
                  ? 'border-[color:var(--apple-blue)] bg-[var(--apple-blue)] text-white'
                  : 'border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-text)] hover:bg-[var(--apple-hover)]'"
              >{{ range.label }}</button>
              <div v-if="currentRange === 'custom'" class="flex w-full flex-col items-stretch gap-2 sm:flex-row sm:items-center">
                <input
                  v-model="customStartDate"
                  type="date"
                  class="input min-h-9 px-2 py-1.5 text-xs"
                />
                <span class="hidden text-xs text-[var(--apple-muted-2)] sm:inline">-</span>
                <input
                  v-model="customEndDate"
                  type="date"
                  class="input min-h-9 px-2 py-1.5 text-xs"
                />
                <button
                  @click="queryKey"
                  class="btn btn-primary min-h-9 px-3 py-1.5 text-xs"
                >{{ t('keyUsage.apply') }}</button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Results Container -->
      <div v-if="showResults">
        <!-- Loading Skeleton -->
        <div v-if="showLoading" class="space-y-6">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6">
              <div class="skeleton h-5 w-24 mb-6"></div>
              <div class="flex justify-center"><div class="skeleton h-40 w-40 rounded-full sm:h-44 sm:w-44"></div></div>
            </div>
            <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6">
              <div class="skeleton h-5 w-24 mb-6"></div>
              <div class="flex justify-center"><div class="skeleton h-40 w-40 rounded-full sm:h-44 sm:w-44"></div></div>
            </div>
          </div>
          <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6">
            <div class="skeleton h-5 w-32 mb-6"></div>
            <div class="space-y-4">
              <div class="skeleton h-4 w-full"></div>
              <div class="skeleton h-4 w-3/4"></div>
              <div class="skeleton h-4 w-5/6"></div>
              <div class="skeleton h-4 w-2/3"></div>
            </div>
          </div>
        </div>

        <!-- Result Content -->
        <div v-else-if="resultData" class="space-y-6">
          <!-- Status Badge -->
          <div v-if="statusInfo" class="fade-up flex items-center justify-center mb-2">
            <div class="inline-flex max-w-full items-center gap-2 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-4 py-2.5">
              <span
                class="w-2.5 h-2.5 rounded-full pulse-dot"
                :class="statusInfo.isActive ? 'bg-emerald-500' : 'bg-rose-500'"
              ></span>
              <span class="min-w-0 truncate text-sm font-medium text-[var(--apple-text)]">{{ statusInfo.label }}</span>
              <span class="text-xs text-[var(--apple-muted-2)]">|</span>
              <span class="min-w-0 truncate text-xs text-[var(--apple-muted)]">{{ statusInfo.statusText }}</span>
            </div>
          </div>

          <!-- Ring Cards Grid -->
          <div v-if="ringItems.length > 0" :class="ringGridClass">
            <div
              v-for="(ring, i) in ringItems"
              :key="i"
              class="fade-up rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 transition-colors hover:border-[color:var(--apple-blue)] sm:p-6"
              :class="`fade-up-delay-${Math.min(i + 1, 4)}`"
            >
              <div class="flex items-center justify-between mb-6">
                <h3 class="text-sm font-semibold uppercase text-[var(--apple-muted)]">
                  {{ ring.title }}
                </h3>
                <Icon
                  :name="ring.iconType"
                  size="md"
                  class="text-[var(--apple-muted-2)]"
                  :stroke-width="2"
                />
              </div>
              <div class="flex justify-center">
                <div class="relative">
                  <svg class="h-40 w-40 sm:h-44 sm:w-44" viewBox="0 0 160 160">
                    <circle cx="80" cy="80" r="68" fill="none" :stroke="ringTrackColor" stroke-width="10"/>
                    <circle
                      class="progress-ring"
                      cx="80" cy="80" r="68" fill="none"
                      :stroke="ringColor(i)"
                      stroke-width="10" stroke-linecap="round"
                      :stroke-dasharray="CIRCUMFERENCE.toFixed(2)"
                      :stroke-dashoffset="getRingOffset(ring)"
                    />
                  </svg>
                  <div class="absolute inset-0 flex flex-col items-center justify-center">
                    <template v-if="ring.isBalance">
                      <span class="max-w-32 truncate text-center text-xl font-semibold tabular-nums sm:max-w-36 sm:text-2xl" :style="{ color: ringColor(i) }">
                        {{ ring.amount }}
                      </span>
                    </template>
                    <template v-else>
                      <span class="text-3xl font-semibold tabular-nums text-[var(--apple-text)]">
                        {{ displayPcts[i] ?? 0 }}%
                      </span>
                      <span class="text-xs text-[var(--apple-muted)] mt-0.5">{{ t('keyUsage.used') }}</span>
                      <span
                        class="mt-1 max-w-32 truncate text-center text-sm font-semibold tabular-nums sm:max-w-36"
                        :style="{ color: ringColor(i) }"
                      >{{ ring.amount }}</span>
                      <p v-if="ring.resetAt && formatResetTime(ring.resetAt)" class="text-xs text-[var(--apple-muted-2)] mt-0.5 tabular-nums">
                        ⟳ {{ formatResetTime(ring.resetAt) }}
                      </p>
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Detail Card -->
          <div
            v-if="detailRows.length > 0"
            class="fade-up fade-up-delay-3 overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)]"
          >
            <div class="border-b border-[color:var(--apple-border-soft)] px-5 py-4 sm:px-6">
              <h3 class="text-sm font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.detailInfo') }}</h3>
            </div>
            <div class="divide-y divide-[color:var(--apple-border-soft)]">
              <div
                v-for="(row, i) in detailRows"
                :key="i"
                class="flex items-start justify-between gap-4 px-5 py-4 sm:items-center sm:px-6"
              >
                <div class="flex min-w-0 items-center gap-3">
                  <div class="w-8 h-8 rounded-lg flex items-center justify-center" :class="row.iconBg">
                    <svg
                      class="w-4 h-4"
                      :class="row.iconColor"
                      viewBox="0 0 24 24" fill="none" stroke="currentColor"
                      stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                      v-html="row.iconSvg"
                    ></svg>
                  </div>
                  <span class="break-words text-sm text-[var(--apple-muted)]">{{ row.label }}</span>
                </div>
                <span class="min-w-0 max-w-[55%] break-words text-right text-sm font-semibold tabular-nums" :class="row.valueClass || 'text-[var(--apple-text)]'">
                  {{ row.value }}
                </span>
              </div>
            </div>
          </div>

          <!-- Usage Stats Card -->
          <div
            v-if="usageStatCells.length > 0"
            class="fade-up fade-up-delay-3 overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)]"
          >
            <div class="border-b border-[color:var(--apple-border-soft)] px-5 py-4 sm:px-6">
              <h3 class="text-sm font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.tokenStats') }}</h3>
            </div>
            <div class="grid grid-cols-2 gap-px bg-[var(--apple-border-soft)] md:grid-cols-4">
              <div
                v-for="(cell, i) in usageStatCells"
                :key="i"
                class="bg-[var(--apple-surface)] px-4 py-4 sm:px-6"
              >
                <div class="text-xs text-[var(--apple-muted)] mb-1">{{ cell.label }}</div>
                <div class="break-words text-sm font-semibold tabular-nums text-[var(--apple-text)]">{{ cell.value }}</div>
              </div>
            </div>
          </div>

          <!-- Daily Usage Table -->
          <div
            v-if="showDailyUsage"
            class="fade-up fade-up-delay-4 overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)]"
          >
            <div class="flex flex-col gap-3 border-b border-[color:var(--apple-border-soft)] px-5 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6">
              <h3 class="text-sm font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.dailyDetail') }}</h3>
              <div class="inline-flex rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-0.5">
                <button
                  v-for="option in dailyUsageOptions"
                  :key="option.value"
                  @click="setDailyUsageDays(option.value)"
                  class="min-w-12 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
                  :class="dailyUsageDays === option.value
                    ? 'bg-[var(--apple-blue)] text-white'
                    : 'text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'"
                >
                  {{ option.label }}
                </button>
              </div>
            </div>
            <div v-if="dailyUsageRows.length > 0" class="overflow-x-auto">
              <table class="w-full">
                <thead>
                  <tr class="border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)]">
                    <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.date') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.requests') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.inputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.outputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.cacheReadTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.cacheWriteTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.cost') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="row in dailyUsageRows"
                    :key="row.date"
                    class="border-b border-[color:var(--apple-border-soft)] last:border-b-0"
                  >
                    <td class="whitespace-nowrap px-4 py-3 text-sm font-medium text-[var(--apple-text)]">{{ row.date }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(row.requests) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(row.input_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(row.output_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(row.cache_read_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(row.cache_write_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm font-medium tabular-nums text-[var(--apple-text)]">{{ formatMoney(row.actual_cost != null ? row.actual_cost : row.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="px-5 py-8 text-center text-sm text-[var(--apple-muted)] sm:px-6">
              {{ t('keyUsage.noDailyUsage') }}
            </div>
          </div>

          <!-- Model Stats Table -->
          <div
            v-if="modelStats.length > 0"
            class="fade-up fade-up-delay-4 overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)]"
          >
            <div class="border-b border-[color:var(--apple-border-soft)] px-5 py-4 sm:px-6">
              <h3 class="text-sm font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.modelStats') }}</h3>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full">
                <thead>
                  <tr class="border-b border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)]">
                    <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.model') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.requests') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.inputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.outputTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.cacheCreationTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.cacheReadTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.totalTokens') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-[var(--apple-muted)]">{{ t('keyUsage.cost') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(m, i) in modelStats"
                    :key="i"
                    class="border-b border-[color:var(--apple-border-soft)] last:border-b-0"
                  >
                    <td class="whitespace-nowrap px-4 py-3 text-sm font-medium text-[var(--apple-text)]">{{ m.model || '-' }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(m.requests) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(m.input_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(m.output_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(m.cache_creation_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(m.cache_read_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm tabular-nums text-[var(--apple-muted)]">{{ fmtNum(m.total_tokens) }}</td>
                    <td class="px-4 py-3 text-right text-sm font-medium tabular-nums text-[var(--apple-text)]">{{ formatMoney(m.actual_cost != null ? m.actual_cost : m.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer (same pattern as HomeView) -->
    <footer class="relative z-10 border-t border-[color:var(--apple-border-soft)] px-6 py-8">
      <div class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left">
        <p class="text-sm text-[var(--apple-muted)]">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div v-if="docsLink" class="flex items-center gap-4">
          <a
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="text-sm text-[var(--apple-muted)] transition-colors hover:text-[var(--apple-text)]"
            @click="handleDocsLinkClick"
          >{{ t('home.docs') }}</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Select from '@/custom/common/WegooSelect.vue'
import Icon from '@/components/icons/Icon.vue'
import { setSettlementCnyPerCredit, useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/custom/utils/docsLink'
import { initTheme } from '@/custom/utils/theme'
import { sanitizeUrl } from '@/utils/url'
import { DEFAULT_SITE_NAME } from '@/utils/branding'
import { formatDateLocalInput } from '@/utils/format'

const { t, locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const {
  settlementCurrency,
  settlementCurrencyOptions,
  setSettlementCurrency,
  formatSettlementAmount,
  formatSettlementAmountPair,
} = useSettlementCurrency()

// ==================== Site Settings (same as HomeView) ====================

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || DEFAULT_SITE_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const rawDocUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const sanitizedDocUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const docsLink = computed(() => resolveDocsLink(sanitizedDocUrl.value || rawDocUrl.value, appStore.cachedPublicSettings?.custom_menu_items ?? []))
const publicBalanceCnyPerCredit = computed(() => appStore.cachedPublicSettings?.payment_balance_recharge_multiplier)

watch(
  publicBalanceCnyPerCredit,
  (value) => {
    setSettlementCnyPerCredit(value)
  },
  { immediate: true }
)

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

const currentYear = computed(() => new Date().getFullYear())
const keyUsageTrustNotes = computed<Array<{ icon: 'lock' | 'document' | 'shield'; title: string; description: string }>>(() => [
  {
    icon: 'lock',
    title: t('keyUsage.trust.sessionQuery'),
    description: t('keyUsage.trust.sessionQueryDesc')
  },
  {
    icon: 'document',
    title: t('keyUsage.trust.auditableRecords'),
    description: t('keyUsage.trust.auditableRecordsDesc')
  },
  {
    icon: 'shield',
    title: t('keyUsage.trust.privacyBoundary'),
    description: t('keyUsage.trust.privacyBoundaryDesc')
  }
])

// ==================== Key Query State ====================

const apiKey = ref('')
const keyVisible = ref(false)
const isQuerying = ref(false)
const showResults = ref(false)
const showLoading = ref(false)
const showDatePicker = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const resultData = ref<any>(null)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null

// ==================== Date Range State ====================

type DateRangeKey = 'today' | '7d' | '30d' | 'custom'
const currentRange = ref<DateRangeKey>('today')
const customStartDate = ref('')
const customEndDate = ref('')
const dailyUsageDays = ref<7 | 30 | 90>(30)

const dateRanges = computed(() => [
  { key: 'today' as const, label: t('keyUsage.dateRangeToday') },
  { key: '7d' as const, label: t('keyUsage.dateRange7d') },
  { key: '30d' as const, label: t('keyUsage.dateRange30d') },
  { key: 'custom' as const, label: t('keyUsage.dateRangeCustom') },
])

const dailyUsageOptions = computed(() => [
  { value: 7 as const, label: t('keyUsage.dateRange7d') },
  { value: 30 as const, label: t('keyUsage.dateRange30d') },
  { value: 90 as const, label: t('keyUsage.dateRange90d') },
])

function setDateRange(key: DateRangeKey) {
  currentRange.value = key
  if (key !== 'custom') {
    queryKey()
  }
}

function getDateParams(): string {
  const now = new Date()
  const params = new URLSearchParams()

  if (currentRange.value === 'custom') {
    if (customStartDate.value && customEndDate.value) {
      params.set('start_date', customStartDate.value)
      params.set('end_date', customEndDate.value)
    }
  } else {
    const end = formatDateLocalInput(now)
    let start: string
    switch (currentRange.value) {
      case 'today': start = end; break
      case '7d': start = formatDateLocalInput(new Date(now.getTime() - 7 * 86400000)); break
      case '30d': start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000)); break
      default: start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000))
    }
    params.set('start_date', start)
    params.set('end_date', end)
  }
  params.set('days', String(dailyUsageDays.value))
  params.set('timezone', getBrowserTimezone())
  return params.toString()
}

function setDailyUsageDays(days: 7 | 30 | 90) {
  if (dailyUsageDays.value === days) return
  dailyUsageDays.value = days
  if (resultData.value && apiKey.value.trim()) {
    queryKey()
  }
}

// ==================== Ring Animation ====================

const CIRCUMFERENCE = 2 * Math.PI * 68
const RING_COLORS = ['#0071E3', '#30D158', '#FF9F0A', '#64D2FF']

const ringAnimated = ref(false)
const displayPcts = ref<number[]>([])
let ringStartRAF: number | null = null
let ringTickRAF: number | null = null
let ringDelayTimer: ReturnType<typeof setTimeout> | null = null

const ringTrackColor = computed(() => '#2c2c2e')

function ringColor(index: number): string {
  return RING_COLORS[index % RING_COLORS.length]
}

interface RingItem {
  title: string
  pct: number
  amount: string
  isBalance?: boolean
  iconType: 'clock' | 'calendar' | 'dollar'
  resetAt?: string | null
}

function getRingOffset(ring: RingItem): number {
  if (!ringAnimated.value) return CIRCUMFERENCE
  if (ring.isBalance) return 0
  return CIRCUMFERENCE - (Math.min(ring.pct, 100) / 100) * CIRCUMFERENCE
}

function triggerRingAnimation(items: RingItem[]) {
  cancelRingAnimation()
  ringAnimated.value = false
  displayPcts.value = items.map(() => 0)

  nextTick(() => {
    ringStartRAF = requestAnimationFrame(() => {
      ringStartRAF = null
      ringDelayTimer = setTimeout(() => {
        ringDelayTimer = null
        ringAnimated.value = true

        // Animate percentage numbers
        const duration = 1000
        const startTime = performance.now()
        const targets = items.map(item => item.isBalance ? 0 : item.pct)

        function tick() {
          const elapsed = performance.now() - startTime
          const p = Math.min(elapsed / duration, 1)
          const ease = 1 - Math.pow(1 - p, 3)
          displayPcts.value = targets.map(target => Math.round(ease * target))
          if (p < 1) {
            ringTickRAF = requestAnimationFrame(tick)
          } else {
            ringTickRAF = null
          }
        }
        ringTickRAF = requestAnimationFrame(tick)
      }, 50)
    })
  })
}

function cancelRingAnimation() {
  if (ringStartRAF != null) {
    cancelAnimationFrame(ringStartRAF)
    ringStartRAF = null
  }
  if (ringTickRAF != null) {
    cancelAnimationFrame(ringTickRAF)
    ringTickRAF = null
  }
  if (ringDelayTimer != null) {
    clearTimeout(ringDelayTimer)
    ringDelayTimer = null
  }
}

// ==================== Computed Data ====================

const statusInfo = computed(() => {
  const data = resultData.value
  if (!data) return null

  if (data.mode === 'quota_limited') {
    const isValid = data.isValid !== false
    const statusMap: Record<string, string> = {
      active: 'Active',
      quota_exhausted: 'Quota Exhausted',
      expired: 'Expired',
    }
    return {
      label: t('keyUsage.quotaMode'),
      statusText: statusMap[data.status] || data.status || 'Unknown',
      isActive: isValid && data.status === 'active',
    }
  }

  return {
    label: data.planName || t('keyUsage.walletBalance'),
    statusText: 'Active',
    isActive: true,
  }
})

const ringItems = computed<RingItem[]>(() => {
  const data = resultData.value
  if (!data) return []

  const items: RingItem[] = []

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const pct = data.quota.limit > 0 ? Math.min(Math.round((data.quota.used / data.quota.limit) * 100), 100) : 0
      items.push({ title: t('keyUsage.totalQuota'), pct, amount: formatSettlementAmountPair(data.quota.used, data.quota.limit, 2), iconType: 'dollar' })
    }
    if (data.rate_limits) {
      const windowLabels: Record<string, string> = { '5h': t('keyUsage.limit5h'), '1d': t('keyUsage.limitDaily'), '7d': t('keyUsage.limit7d') }
      const windowIcons: Record<string, 'clock' | 'calendar'> = { '5h': 'clock', '1d': 'calendar', '7d': 'calendar' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? Math.min(Math.round((rl.used / rl.limit) * 100), 100) : 0
        items.push({
          title: windowLabels[rl.window] || rl.window,
          pct,
          amount: formatSettlementAmountPair(rl.used, rl.limit, 2),
          iconType: windowIcons[rl.window] || 'clock',
          resetAt: rl.reset_at,
        })
      }
    }
  } else {
    if (data.subscription) {
      const sub = data.subscription
      const limits = [
        { label: t('keyUsage.limitDaily'), usage: sub.daily_usage_usd, limit: sub.daily_limit_usd },
        { label: t('keyUsage.limitWeekly'), usage: sub.weekly_usage_usd, limit: sub.weekly_limit_usd },
        { label: t('keyUsage.limitMonthly'), usage: sub.monthly_usage_usd, limit: sub.monthly_limit_usd },
      ]
      for (const l of limits) {
        if (l.limit != null && l.limit > 0) {
          const pct = Math.min(Math.round((l.usage / l.limit) * 100), 100)
          items.push({ title: l.label, pct, amount: formatSettlementAmountPair(l.usage, l.limit, 2), iconType: 'calendar' })
        }
      }
    }
    if (!data.subscription && data.balance != null) {
      items.push({ title: t('keyUsage.walletBalance'), pct: 0, amount: formatMoney(data.balance), isBalance: true, iconType: 'dollar' })
    }
  }

  return items
})

const ringGridClass = computed(() => {
  const len = ringItems.value.length
  if (len === 1) return 'grid grid-cols-1 max-w-md mx-auto gap-6'
  if (len === 2) return 'grid grid-cols-1 md:grid-cols-2 gap-6'
  return 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6'
})

interface DetailRow {
  iconBg: string
  iconColor: string
  iconSvg: string
  label: string
  value: string
  valueClass: string
}

function getUsageColor(pct: number): string {
  if (pct > 90) return 'text-rose-500'
  if (pct > 70) return 'text-amber-500'
  return 'text-emerald-500'
}

const detailRows = computed<DetailRow[]>(() => {
  const data = resultData.value
  if (!data) return []

  const rows: DetailRow[] = []
  const ICON_SHIELD = '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'
  const ICON_CALENDAR = '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>'
  const ICON_DOLLAR = '<line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>'
  const ICON_CHECK = '<polyline points="20 6 9 17 4 12"/>'

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const remainColor = data.quota.remaining <= 0 ? 'text-rose-500'
        : data.quota.remaining < data.quota.limit * 0.1 ? 'text-amber-500'
        : 'text-emerald-500'
      rows.push({
        iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
        label: t('keyUsage.remainingQuota'), value: formatMoney(data.quota.remaining), valueClass: remainColor,
      })
    }
    if (data.expires_at) {
      const daysLeft = data.days_until_expiry
      let expiryStr = formatDate(data.expires_at)
      if (daysLeft != null) {
        expiryStr += daysLeft > 0 ? ` ${t('keyUsage.daysLeft', { days: daysLeft })}` : daysLeft === 0 ? ` ${t('keyUsage.todayExpires')}` : ''
      }
      rows.push({
        iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
        label: t('keyUsage.expiresAt'), value: expiryStr, valueClass: '',
      })
    }
    if (data.rate_limits) {
      const windowMap: Record<string, string> = { '5h': '5H', '1d': locale.value === 'zh' ? '日' : 'D', '7d': '7D' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? (rl.used / rl.limit) * 100 : 0
        let valueStr = formatSettlementAmountPair(rl.used, rl.limit, 2)
        const resetStr = formatResetTime(rl.reset_at)
        if (resetStr) {
          valueStr += ` (⟳ ${resetStr})`
        }
        rows.push({
          iconBg: 'bg-primary-500/10', iconColor: 'text-primary-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${windowMap[rl.window] || rl.window})`,
          value: valueStr,
          valueClass: getUsageColor(pct),
        })
      }
    }
  } else {
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_CHECK,
      label: t('keyUsage.subscriptionType'), value: data.planName || t('keyUsage.walletBalance'), valueClass: '',
    })

    if (data.subscription) {
      const sub = data.subscription
      if (sub.daily_limit_usd > 0) {
        const pct = (sub.daily_usage_usd / sub.daily_limit_usd) * 100
        rows.push({
          iconBg: 'bg-primary-500/10', iconColor: 'text-primary-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '日' : 'D'})`, value: formatSettlementAmountPair(sub.daily_usage_usd, sub.daily_limit_usd, 2), valueClass: getUsageColor(pct),
        })
      }
      if (sub.weekly_limit_usd > 0) {
        const pct = (sub.weekly_usage_usd / sub.weekly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-indigo-500/10', iconColor: 'text-indigo-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '周' : 'W'})`, value: formatSettlementAmountPair(sub.weekly_usage_usd, sub.weekly_limit_usd, 2), valueClass: getUsageColor(pct),
        })
      }
      if (sub.monthly_limit_usd > 0) {
        const pct = (sub.monthly_usage_usd / sub.monthly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '月' : 'M'})`, value: formatSettlementAmountPair(sub.monthly_usage_usd, sub.monthly_limit_usd, 2), valueClass: getUsageColor(pct),
        })
      }
      if (sub.expires_at) {
        rows.push({
          iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
          label: t('keyUsage.subscriptionExpires'), value: formatDate(sub.expires_at), valueClass: '',
        })
      }
    }

    const remainColor = data.remaining != null
      ? (data.remaining <= 0 ? 'text-rose-500' : data.remaining < 10 ? 'text-amber-500' : 'text-emerald-500')
      : ''
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
      label: t('keyUsage.remainingQuota'), value: data.remaining != null ? formatMoney(data.remaining) : '-', valueClass: remainColor,
    })
  }

  return rows
})

interface StatCell {
  label: string
  value: string
}

const usageStatCells = computed<StatCell[]>(() => {
  const usage = resultData.value?.usage
  if (!usage) return []

  const today = usage.today || {}
  const total = usage.total || {}

  return [
    { label: t('keyUsage.todayRequests'), value: fmtNum(today.requests) },
    { label: t('keyUsage.todayInputTokens'), value: fmtNum(today.input_tokens) },
    { label: t('keyUsage.todayOutputTokens'), value: fmtNum(today.output_tokens) },
    { label: t('keyUsage.todayTokens'), value: fmtNum(today.total_tokens) },
    { label: t('keyUsage.todayCacheCreation'), value: fmtNum(today.cache_creation_tokens) },
    { label: t('keyUsage.todayCacheRead'), value: fmtNum(today.cache_read_tokens) },
    { label: t('keyUsage.todayCost'), value: formatMoney(today.actual_cost) },
    { label: t('keyUsage.rpmTpm'), value: `${usage.rpm || 0} / ${usage.tpm || 0}` },
    { label: t('keyUsage.totalRequests'), value: fmtNum(total.requests) },
    { label: t('keyUsage.totalInputTokens'), value: fmtNum(total.input_tokens) },
    { label: t('keyUsage.totalOutputTokens'), value: fmtNum(total.output_tokens) },
    { label: t('keyUsage.totalTokensLabel'), value: fmtNum(total.total_tokens) },
    { label: t('keyUsage.totalCacheCreation'), value: fmtNum(total.cache_creation_tokens) },
    { label: t('keyUsage.totalCacheRead'), value: fmtNum(total.cache_read_tokens) },
    { label: t('keyUsage.totalCost'), value: formatMoney(total.actual_cost) },
    { label: t('keyUsage.avgDuration'), value: usage.average_duration_ms ? `${Math.round(usage.average_duration_ms)} ms` : '-' },
  ]
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const modelStats = computed<any[]>(() => resultData.value?.model_stats || [])

interface DailyUsageRow {
  date: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  cost: number
  actual_cost?: number
}

const dailyUsageRows = computed<DailyUsageRow[]>(() => {
  const rows = resultData.value?.daily_usage
  return Array.isArray(rows) ? rows : []
})

const showDailyUsage = computed(() => Boolean(resultData.value && Array.isArray(resultData.value.daily_usage)))

// ==================== Utility Functions ====================

function formatMoney(value: number | null | undefined): string {
  if (value == null || value < 0) return '-'
  return formatSettlementAmount(value, 2)
}

function fmtNum(val: number | null | undefined): string {
  if (val == null) return '-'
  return val.toLocaleString()
}

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const loc = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return d.toLocaleDateString(loc, { year: 'numeric', month: 'long', day: 'numeric' })
}

function getBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

// ==================== API Query ====================

async function fetchUsage(key: string) {
  const dateParams = getDateParams()
  const url = '/v1/usage' + (dateParams ? '?' + dateParams : '')
  const res = await fetch(url, {
    headers: { 'Authorization': 'Bearer ' + key },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const msg = body?.error?.message || body?.message || `${t('keyUsage.queryFailed')} (${res.status})`
    throw new Error(msg)
  }
  return await res.json()
}

async function queryKey() {
  if (isQuerying.value) return
  const key = apiKey.value.trim()
  if (!key) {
    appStore.showInfo(t('keyUsage.enterApiKey'))
    return
  }

  isQuerying.value = true
  showResults.value = true
  showLoading.value = true
  resultData.value = null

  try {
    const data = await fetchUsage(key)
    resultData.value = data
    showLoading.value = false
    showDatePicker.value = true

    // Trigger ring animations after DOM update
    nextTick(() => {
      triggerRingAnimation(ringItems.value)
    })

    appStore.showSuccess(t('keyUsage.querySuccess'))
  } catch (err) {
    showResults.value = false
    showLoading.value = false
    appStore.showError((err as Error).message || t('keyUsage.queryFailedRetry'))
  } finally {
    isQuerying.value = false
  }
}

// ==================== Lifecycle ====================

function formatResetTime(resetAt: string | null | undefined): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keyUsage.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

onMounted(() => {
  initTheme()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  if (resetTimer) clearInterval(resetTimer)
  cancelRingAnimation()
})
</script>

<style scoped>
/* Input focus ring */
.input-ring {
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}
.input-ring:focus {
  border-color: var(--apple-blue);
  box-shadow: 0 0 0 3px var(--apple-focus-ring);
  outline: none;
}

/* Ring animation */
.progress-ring {
  transition: stroke-dashoffset 1.2s cubic-bezier(0.4, 0, 0.2, 1);
  transform: rotate(-90deg);
  transform-origin: 50% 50%;
}

/* Skeleton loading */
@keyframes skeleton-pulse-kv {
  0%, 100% { opacity: 0.68; }
  50% { opacity: 1; }
}
.skeleton {
  background: #e5e5ea;
  animation: skeleton-pulse-kv 1.4s ease-in-out infinite;
  border-radius: 8px;
}
:global(.dark) .skeleton {
  background: #2c2c2e;
}

/* Fade up animation */
@keyframes fade-up-kv {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}
.fade-up {
  animation: fade-up-kv 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
}
.fade-up-delay-1 { animation-delay: 0.1s; opacity: 0; }
.fade-up-delay-2 { animation-delay: 0.2s; opacity: 0; }
.fade-up-delay-3 { animation-delay: 0.3s; opacity: 0; }
.fade-up-delay-4 { animation-delay: 0.4s; opacity: 0; }

/* Pulse dot */
@keyframes pulse-dot-kv {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 currentColor; }
  50% { opacity: 0.6; box-shadow: 0 0 8px 2px currentColor; }
}
.pulse-dot {
  animation: pulse-dot-kv 2s ease-in-out infinite;
}

/* Tabular nums */
.tabular-nums {
  font-variant-numeric: tabular-nums;
}
</style>
