<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="space-y-4">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
            <div class="min-w-0 max-w-3xl">
              <h1 class="usage-page-title text-2xl font-semibold tracking-normal">
                {{ t('usage.title') }}
              </h1>
              <p class="usage-page-description mt-1 max-w-2xl text-sm leading-6">
                {{ t('usage.description') }}
              </p>
              <div class="mt-4 grid gap-2 text-sm sm:grid-cols-3">
                <div
                  v-for="item in usageTrustNotes"
                  :key="item.title"
                  class="usage-trust-card rounded-lg border px-3 py-2.5"
                >
                  <div class="usage-trust-title flex items-center gap-2 font-semibold">
                    <Icon :name="item.icon" size="sm" class="usage-trust-icon" />
                    <span>{{ item.title }}</span>
                  </div>
                  <p class="usage-trust-description mt-1 text-xs leading-5">
                    {{ item.description }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div class="usage-stats-grid grid gap-px overflow-hidden rounded-lg border sm:grid-cols-2 xl:grid-cols-4">
            <div class="usage-stat-card min-w-0 p-4">
              <div class="flex items-start gap-3">
                <div class="rounded-lg bg-blue-50 p-2 ring-1 ring-blue-100 dark:bg-blue-500/10 dark:ring-blue-500/20">
                  <Icon name="document" size="md" class="text-blue-600 dark:text-blue-400" />
                </div>
                <div class="min-w-0">
                  <p class="usage-stat-label text-xs font-medium">
                    {{ t('usage.totalRequests') }}
                  </p>
                  <p class="usage-stat-value text-xl font-semibold">
                    {{ usageStats?.total_requests?.toLocaleString() || '0' }}
                  </p>
                  <p class="usage-stat-muted text-xs">
                    {{ t('usage.inSelectedRange') }}
                  </p>
                </div>
              </div>
            </div>

            <div class="usage-stat-card min-w-0 p-4">
              <div class="flex items-start gap-3">
                <div class="rounded-lg bg-amber-50 p-2 ring-1 ring-amber-100 dark:bg-amber-500/10 dark:ring-amber-500/20">
                  <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" />
                </div>
                <div class="min-w-0 flex-1">
                  <p class="usage-stat-label text-xs font-medium">
                    {{ t('usage.totalTokens') }}
                  </p>
                  <p class="usage-stat-value text-xl font-semibold">
                    {{ formatTokens(usageStats?.total_tokens || 0) }}
                  </p>
                  <div class="mt-1 space-y-1 text-xs">
                    <p class="usage-stat-muted flex flex-wrap gap-x-2 gap-y-0.5">
                      <span>{{ t('usage.in') }} {{ formatTokens(usageStats?.total_input_tokens || 0) }}</span>
                      <span>{{ t('usage.out') }} {{ formatTokens(usageStats?.total_output_tokens || 0) }}</span>
                    </p>
                    <p class="flex flex-wrap gap-x-2 gap-y-0.5">
                      <span class="text-sky-600 dark:text-sky-400">{{ t('usage.cacheHit') }} {{ formatTokens(usageStats?.total_cache_read_tokens || 0) }}</span>
                      <span class="cursor-help text-amber-600 dark:text-amber-400" :title="t('usage.openaiCacheCreateNote')">{{ t('usage.cacheCreate') }} {{ formatTokens(usageStats?.total_cache_creation_tokens || 0) }}</span>
                    </p>
                  </div>
                  <p class="mt-1 text-xs font-medium text-sky-600 dark:text-sky-400">
                    {{ t('usage.cacheHitRate') }}:
                    <template v-if="cacheStats.totalInput > 0">
                      <span>{{ cacheStats.ratePercent }}</span>
                      <span class="usage-stat-muted ml-1">
                        ({{ formatTokens(cacheStats.cacheRead) }}/{{ formatTokens(cacheStats.totalInput) }})
                      </span>
                    </template>
                    <template v-else>-</template>
                  </p>
                </div>
              </div>
            </div>

            <div class="usage-stat-card min-w-0 p-4">
              <div class="flex items-start gap-3">
                <div class="rounded-lg bg-emerald-50 p-2 ring-1 ring-emerald-100 dark:bg-emerald-500/10 dark:ring-emerald-500/20">
                  <Icon name="dollar" size="md" class="text-green-600 dark:text-green-400" />
                </div>
                <div class="min-w-0 flex-1">
                  <p class="usage-stat-label text-xs font-medium">
                    {{ t('usage.totalCost') }}
                  </p>
                  <p class="text-xl font-semibold text-emerald-600 dark:text-emerald-400">
                    {{ formatSettlementAmount(usageStats?.total_actual_cost || 0, 4) }}
                  </p>
                  <p class="usage-stat-muted text-xs">
                    {{ t('usage.actualCost') }} /
                    <span class="line-through">{{ formatSettlementAmount(usageStats?.total_cost || 0, 4) }}</span>
                    {{ t('usage.standardCost') }}
                  </p>
                </div>
              </div>
            </div>

            <div class="usage-stat-card min-w-0 p-4">
              <div class="flex items-start gap-3">
                <div class="rounded-lg bg-slate-50 p-2 ring-1 ring-slate-100 dark:bg-slate-500/10 dark:ring-slate-500/20">
                  <Icon name="clock" size="md" class="text-slate-600 dark:text-slate-300" />
                </div>
                <div class="min-w-0">
                  <p class="usage-stat-label text-xs font-medium">
                    {{ t('usage.avgDuration') }}
                  </p>
                  <p class="usage-stat-value text-xl font-semibold">
                    {{ formatDuration(usageStats?.average_duration_ms || 0) }}
                  </p>
                  <p class="usage-stat-muted text-xs">{{ t('usage.perRequest') }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="grid gap-4 lg:grid-cols-[minmax(0,220px)_minmax(0,280px)_minmax(0,1fr)] lg:items-end">
          <!-- Access credential filter -->
          <div class="min-w-0">
            <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
            <Select
              v-model="filters.api_key_id"
              :options="apiKeyOptions"
              :placeholder="t('usage.allApiKeys')"
              @change="applyFilters"
            />
          </div>

          <!-- Date Range Filter -->
          <div class="min-w-0">
            <label class="input-label">{{ t('usage.timeRange') }}</label>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="onDateRangeChange"
            />
          </div>

          <!-- Actions -->
          <div class="grid min-w-0 grid-cols-2 gap-2 sm:flex sm:flex-wrap sm:items-center lg:justify-end">
            <button
              @click="applyFilters"
              :disabled="loading"
              class="btn btn-secondary min-w-0 justify-center sm:flex-none"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
            <button
              @click="resetFilters"
              class="btn btn-secondary min-w-0 justify-center sm:flex-none"
            >
              {{ t('common.reset') }}
            </button>
            <button
              @click="exportToCSV"
              :disabled="exporting"
              class="btn btn-primary col-span-2 min-w-0 justify-center sm:col-span-1 sm:flex-none"
            >
              <Icon v-if="!exporting" name="download" size="sm" />
              <svg
                v-else
                class="-ml-1 h-4 w-4 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
              {{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <!-- Tab 切换栏 -->
        <div v-if="errorViewEnabled" class="px-3 pt-3">
          <div class="usage-tabs tabs w-full sm:w-fit">
            <button class="tab flex-1 sm:flex-none" :class="{ 'tab-active': activeTab === 'usage' }" @click="activeTab = 'usage'">
              {{ t('usage.tabs.usage') }}
            </button>
            <button class="tab flex-1 sm:flex-none" :class="{ 'tab-active': activeTab === 'errors' }" @click="switchToErrors">
              {{ t('usage.tabs.errors') }}
            </button>
          </div>
        </div>

        <!-- 用量明细表 -->
        <!-- flex 链让 DataTable 根 .table-wrapper(flex:1)拿到有界高度以启用内部滚动。
             虚拟化器测高 race 导致的概率空白,已在 DataTable 内用「就绪门控 + initialRect 兜底」根治。 -->
        <div v-show="activeTab === 'usage'" class="usage-table-panel flex min-h-0 flex-1 flex-col">
          <DataTable
            :columns="columns"
            :data="usageLogs"
            :loading="loading"
            :server-side-sort="true"
            :estimate-row-height="88"
            :overscan="12"
            default-sort-key="created_at"
            default-sort-order="desc"
            @sort="handleSort"
          >
          <template #cell-api_key="{ row }">
            <span class="usage-cell-strong text-sm">{{
              row.api_key?.name || '-'
            }}</span>
          </template>

          <template #cell-model="{ value, row }">
            <div class="min-w-0">
              <span class="usage-cell-strong block max-w-[14rem] truncate font-medium" :title="value">{{ value }}</span>
              <span class="usage-cell-faint mt-0.5 block text-xs">
                {{ t('usage.reasoningEffort') }}: {{ formatReasoningEffort(row.reasoning_effort) }}
              </span>
            </div>
          </template>

          <template #cell-reasoning_effort="{ row }">
            <span class="usage-cell-strong text-sm">
              {{ formatReasoningEffort(row.reasoning_effort) }}
            </span>
          </template>

          <template #cell-endpoint="{ row }">
            <span class="usage-cell-muted block max-w-[320px] whitespace-normal break-all text-sm">
              {{ formatUsageEndpoints(row) }}
            </span>
          </template>

          <template #cell-stream="{ row }">
            <div class="flex flex-wrap gap-1.5">
              <span
                class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
                :class="getRequestTypeBadgeClass(row)"
              >
                {{ getRequestTypeLabel(row) }}
              </span>
              <span class="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium"
                    :class="getUsageBillingModeBadgeClass(getDisplayBillingMode(row))">
                {{ getBillingModeLabel(getDisplayBillingMode(row), t, 'usage') }}
              </span>
            </div>
          </template>

          <template #cell-billing_mode="{ row }">
            <span class="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium"
                  :class="getUsageBillingModeBadgeClass(getDisplayBillingMode(row))">
              {{ getBillingModeLabel(getDisplayBillingMode(row), t, 'usage') }}
            </span>
          </template>

          <template #cell-tokens="{ row }">
            <!-- 图片生成请求 -->
            <div v-if="isImageUsage(row)" class="flex items-center gap-1.5">
              <svg
                class="h-4 w-4 text-indigo-500"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>
              <span class="usage-cell-strong font-medium">{{ row.image_count }}{{ t('usage.imageUnit') }}</span>
              <span class="usage-cell-faint">({{ formatImageBillingSize(row, t) }})</span>
            </div>
            <!-- Token 请求 -->
            <div v-else class="flex items-center gap-1.5">
              <div class="space-y-1.5 text-sm">
                <!-- Input / Output Tokens -->
                <div class="flex items-center gap-2">
                  <!-- Input -->
                  <div class="inline-flex items-center gap-1">
                    <Icon name="arrowDown" size="sm" class="text-emerald-500" />
                    <span class="usage-cell-strong font-medium">{{
                      (row.input_tokens ?? 0).toLocaleString()
                    }}</span>
                  </div>
                  <!-- Output -->
                  <div class="inline-flex items-center gap-1">
                    <Icon name="arrowUp" size="sm" class="text-violet-500" />
                    <span class="usage-cell-strong font-medium">{{
                      (row.output_tokens ?? 0).toLocaleString()
                    }}</span>
                  </div>
                </div>
                <!-- Cache Tokens (Read + Write) -->
                <div
                  v-if="row.cache_read_tokens > 0 || row.cache_creation_tokens > 0"
                  class="flex items-center gap-2"
                >
                  <!-- Cache Read -->
                  <div v-if="row.cache_read_tokens > 0" class="inline-flex items-center gap-1">
                    <Icon name="inbox" size="sm" class="text-sky-500" />
                    <span class="font-medium text-sky-600 dark:text-sky-400">{{
                      formatCacheTokens(row.cache_read_tokens)
                    }}</span>
                  </div>
                  <!-- Cache Write -->
                  <div v-if="row.cache_creation_tokens > 0" class="inline-flex items-center gap-1">
                    <Icon name="edit" size="sm" class="text-amber-500" />
                    <span class="font-medium text-amber-600 dark:text-amber-400">{{
                      formatCacheTokens(row.cache_creation_tokens)
                    }}</span>
                    <span v-if="row.cache_creation_1h_tokens > 0" class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-100 text-orange-600 ring-1 ring-inset ring-orange-200 dark:bg-orange-500/20 dark:text-orange-400 dark:ring-orange-500/30">1h</span>
                    <span v-if="row.cache_ttl_overridden" :title="t('usage.cacheTtlOverriddenHint')" class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30 cursor-help">R</span>
                  </div>
                </div>
                <div v-if="hasImageOutputTokens(row)" class="flex items-center gap-2">
                  <div class="inline-flex items-center gap-1">
                    <svg class="h-3.5 w-3.5 text-pink-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
                    <span class="font-medium text-pink-600 dark:text-pink-400">{{ row.image_output_tokens.toLocaleString() }}</span>
                  </div>
                </div>
              </div>
              <!-- Token Detail Tooltip -->
              <button
                type="button"
                class="group relative inline-flex"
                :aria-label="t('usage.tokenDetails')"
                @click.stop="showTokenTooltip($event, row)"
                @focus="showTokenTooltip($event, row)"
                @blur="hideTokenTooltip"
                @mouseenter="showTokenTooltip($event, row)"
                @mouseleave="hideTokenTooltip"
              >
                <div
                  class="usage-info-dot flex h-4 w-4 cursor-help items-center justify-center rounded-full transition-colors"
                >
                  <Icon
                    name="infoCircle"
                    size="xs"
                    class="usage-info-icon"
                  />
                </div>
              </button>
            </div>
          </template>

          <template #cell-cost="{ row }">
            <div class="flex items-center gap-1.5 text-sm">
              <span class="font-medium text-green-600 dark:text-green-400">
                {{ formatSettlementAmount(row.actual_cost, 6) }}
              </span>
              <!-- Cost Detail Tooltip -->
              <button
                type="button"
                class="group relative inline-flex"
                :aria-label="t('usage.costDetails')"
                @click.stop="showTooltip($event, row)"
                @focus="showTooltip($event, row)"
                @blur="hideTooltip"
                @mouseenter="showTooltip($event, row)"
                @mouseleave="hideTooltip"
              >
                <div
                  class="usage-info-dot flex h-4 w-4 cursor-help items-center justify-center rounded-full transition-colors"
                >
                  <Icon
                    name="infoCircle"
                    size="xs"
                    class="usage-info-icon"
                  />
                </div>
              </button>
            </div>
          </template>

          <template #cell-first_token="{ row }">
            <span
              v-if="row.first_token_ms != null"
              class="usage-cell-muted text-sm"
            >
              {{ formatDuration(row.first_token_ms) }}
            </span>
            <span v-else class="usage-cell-faint text-sm">-</span>
          </template>

          <template #cell-duration="{ row }">
            <div class="text-sm">
              <span class="usage-cell-muted block">{{ formatDuration(row.duration_ms) }}</span>
              <span v-if="row.first_token_ms != null" class="usage-cell-faint mt-0.5 block text-xs">
                {{ t('usage.firstToken') }} {{ formatDuration(row.first_token_ms) }}
              </span>
            </div>
          </template>

          <template #cell-created_at="{ value }">
            <span class="usage-cell-muted text-sm">{{
              formatDateTime(value)
            }}</span>
          </template>

          <template #cell-user_agent="{ row }">
            <span v-if="row.user_agent" class="usage-cell-muted block max-w-[320px] whitespace-normal break-all text-sm" :title="row.user_agent">{{ formatUserAgent(row.user_agent) }}</span>
            <span v-else class="usage-cell-faint text-sm">-</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              class="usage-row-action inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium"
              @click="openUsageTicket(row)"
            >
              <Icon name="chatBubble" size="sm" />
              {{ t('tickets.createTicket') }}
            </button>
          </template>

          <template #empty>
            <EmptyState :title="t('usage.noRecords')" />
          </template>
        </DataTable>
        </div>

        <!-- 问题记录表 -->
        <div v-if="errorViewEnabled" v-show="activeTab === 'errors'" class="flex min-h-0 flex-1 flex-col">
          <UserErrorRequestsTable
            :rows="errorRows"
            :total="errorTotal"
            :loading="errorLoading"
            :page="errorPage"
            :page-size="errorPageSize"
            :api-keys="apiKeys"
            @filter="onErrorFilter"
            @update:page="onErrorPage"
            @update:pageSize="onErrorPageSize"
          />
        </div>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0 && activeTab === 'usage'"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>

  <!-- Token Tooltip Portal -->
  <Teleport to="body">
    <div
      v-if="tokenTooltipVisible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{
        left: tokenTooltipPosition.x + 'px',
        top: tokenTooltipPosition.y + 'px'
      }"
    >
      <div
        class="usage-tooltip max-w-[calc(100vw-1rem)] whitespace-normal rounded-lg border px-3 py-2.5 text-xs shadow-md sm:whitespace-nowrap"
      >
        <div class="space-y-1.5">
          <!-- Token Breakdown -->
          <div>
            <div class="usage-tooltip-title mb-1 text-xs font-semibold">{{ t('usage.tokenDetails') }}</div>
            <div v-if="tokenTooltipData && tokenTooltipData.input_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.inputTokens') }}</span>
              <span class="font-medium text-white">{{ tokenTooltipData.input_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.output_tokens > 0 && !hasImageOutputTokens(tokenTooltipData)" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.outputTokens') }}</span>
              <span class="font-medium text-white">{{ tokenTooltipData.output_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="tokenTooltipData && hasImageOutputTokens(tokenTooltipData) && textOutputTokens(tokenTooltipData) > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.outputTokens') }}</span>
              <span class="font-medium text-white">{{ textOutputTokens(tokenTooltipData).toLocaleString() }}</span>
            </div>
            <div v-if="tokenTooltipData && hasImageOutputTokens(tokenTooltipData)" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.imageOutputTokens') }}</span>
              <span class="font-medium text-pink-300">{{ tokenTooltipData.image_output_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.cache_creation_tokens > 0">
              <!-- 有 5m/1h 明细时，展开显示 -->
              <template v-if="tokenTooltipData.cache_creation_5m_tokens > 0 || tokenTooltipData.cache_creation_1h_tokens > 0">
                <div v-if="tokenTooltipData.cache_creation_5m_tokens > 0" class="flex items-center justify-between gap-4">
                  <span class="usage-tooltip-label flex items-center gap-1.5">
                    {{ t('usage.cacheCreation5mTokens') }}
                    <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-amber-500/20 text-amber-400 ring-1 ring-inset ring-amber-500/30">5m</span>
                  </span>
                  <span class="font-medium text-white">{{ tokenTooltipData.cache_creation_5m_tokens.toLocaleString() }}</span>
                </div>
                <div v-if="tokenTooltipData.cache_creation_1h_tokens > 0" class="flex items-center justify-between gap-4">
                  <span class="usage-tooltip-label flex items-center gap-1.5">
                    {{ t('usage.cacheCreation1hTokens') }}
                    <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-500/20 text-orange-400 ring-1 ring-inset ring-orange-500/30">1h</span>
                  </span>
                  <span class="font-medium text-white">{{ tokenTooltipData.cache_creation_1h_tokens.toLocaleString() }}</span>
                </div>
              </template>
              <!-- 无明细时，只显示聚合值 -->
              <div v-else class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.cacheCreationTokens') }}</span>
                <span class="font-medium text-white">{{ tokenTooltipData.cache_creation_tokens.toLocaleString() }}</span>
              </div>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.cache_ttl_overridden" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label flex items-center gap-1.5">
                {{ t('usage.cacheTtlOverriddenLabel') }}
                <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-500/20 text-rose-400 ring-1 ring-inset ring-rose-500/30">R-{{ tokenTooltipData.cache_creation_1h_tokens > 0 ? '5m' : '1H' }}</span>
              </span>
              <span class="font-medium text-rose-400">{{ tokenTooltipData.cache_creation_1h_tokens > 0 ? t('usage.cacheTtlOverridden1h') : t('usage.cacheTtlOverridden5m') }}</span>
            </div>
            <div v-if="tokenTooltipData && tokenTooltipData.cache_read_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.cacheReadTokens') }}</span>
              <span class="font-medium text-white">{{ tokenTooltipData.cache_read_tokens.toLocaleString() }}</span>
            </div>
            <div
              v-if="tokenTooltipData && isOpenAICacheReadOnlyUsage(tokenTooltipData)"
              class="mt-1 max-w-[18rem] whitespace-normal rounded border border-sky-500/20 bg-sky-500/10 px-2 py-1 text-[11px] leading-4 text-sky-100"
            >
              {{ t('usage.openaiCacheCreateNote') }}
            </div>
          </div>
          <!-- Total -->
          <div class="usage-tooltip-divider flex items-center justify-between gap-6 border-t pt-1.5">
            <span class="usage-tooltip-label">{{ t('usage.totalTokens') }}</span>
            <span class="font-semibold text-blue-400">{{ ((tokenTooltipData?.input_tokens || 0) + (tokenTooltipData?.output_tokens || 0) + (tokenTooltipData?.cache_creation_tokens || 0) + (tokenTooltipData?.cache_read_tokens || 0)).toLocaleString() }}</span>
          </div>
        </div>
        <!-- Tooltip Arrow (left side) -->
        <div
          class="usage-tooltip-arrow absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-t-transparent"
        ></div>
      </div>
    </div>
  </Teleport>

  <!-- Tooltip Portal -->
  <Teleport to="body">
    <div
      v-if="tooltipVisible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{
        left: tooltipPosition.x + 'px',
        top: tooltipPosition.y + 'px'
      }"
    >
      <div
        class="usage-tooltip max-w-[calc(100vw-1rem)] whitespace-normal rounded-lg border px-3 py-2.5 text-xs shadow-md sm:whitespace-nowrap"
      >
        <div class="space-y-1.5">
          <!-- Cost Breakdown -->
          <div class="usage-tooltip-section mb-2 border-b pb-1.5">
            <div class="usage-tooltip-title mb-1 text-xs font-semibold">{{ t('usage.costDetails') }}</div>
            <div v-if="tooltipData && tooltipData.input_cost > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.inputCost') }}</span>
              <span class="font-medium text-white">{{ formatSettlementAmount(tooltipData.input_cost, 6) }}</span>
            </div>
            <div v-if="tooltipData && tooltipData.output_cost > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.outputCost') }}</span>
              <span class="font-medium text-white">{{ formatSettlementAmount(tooltipData.output_cost, 6) }}</span>
            </div>
            <div v-if="tooltipData && hasImageOutputCost(tooltipData)" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.imageOutputCost') }}</span>
              <span class="font-medium text-pink-300">{{ formatSettlementAmount(tooltipData.image_output_cost, 6) }}</span>
            </div>
            <!-- Token billing: show unit prices per 1M tokens -->
            <template v-if="getDisplayBillingMode(tooltipData) !== BILLING_MODE_IMAGE">
              <div v-if="tooltipData && tooltipData.input_tokens > 0" class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.inputTokenPrice') }}</span>
                <span class="font-medium text-sky-300">{{ formatSettlementTokenPricePerMillion(tooltipData.input_cost, tooltipData.input_tokens) }} {{ t('usage.perMillionTokens') }}</span>
              </div>
              <div v-if="tooltipData && tooltipData.output_cost > 0 && textOutputTokens(tooltipData) > 0" class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.outputTokenPrice') }}</span>
                <span class="font-medium text-violet-300">{{ formatSettlementTokenPricePerMillion(tooltipData.output_cost, textOutputTokens(tooltipData)) }} {{ t('usage.perMillionTokens') }}</span>
              </div>
              <div v-if="tooltipData && hasImageOutputTokens(tooltipData)" class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageOutputTokenPrice') }}</span>
                <span class="font-medium text-pink-300">{{ formatSettlementTokenPricePerMillion(tooltipData.image_output_cost ?? 0, tooltipData.image_output_tokens) }} {{ t('usage.perMillionTokens') }}</span>
              </div>
            </template>
            <!-- Per-image billing: show image metadata and unit price -->
            <template v-else-if="tooltipData && isImageUsage(tooltipData)">
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageCount') }}</span>
                <span class="font-medium text-white">{{ tooltipData.image_count }}{{ t('usage.imageUnit') }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageBillingSize') }}</span>
                <span class="font-medium text-white">{{ formatImageBillingSize(tooltipData, t) }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageSizeSource') }}</span>
                <span class="font-medium text-white">{{ formatImageSizeSource(tooltipData, t) }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageInputSize') }}</span>
                <span class="font-medium text-white">{{ formatImageInputSize(tooltipData, t) }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageOutputSize') }}</span>
                <span class="font-medium text-white">{{ formatImageOutputSize(tooltipData, t) }}</span>
              </div>
              <div v-if="formatImageSizeBreakdown(tooltipData)" class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageSizeBreakdown') }}</span>
                <span class="font-medium text-white">{{ formatImageSizeBreakdown(tooltipData) }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageUnitPrice') }}</span>
                <span class="font-medium text-sky-300">{{ formatSettlementAmount(imageUnitPrice(tooltipData), 6) }}</span>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span class="usage-tooltip-label">{{ t('usage.imageTotalPrice') }}</span>
                <span class="font-medium text-white">{{ formatSettlementAmount(tooltipData.total_cost || 0, 6) }}</span>
              </div>
            </template>
            <div v-else class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.unitPrice') }}</span>
              <span class="font-medium text-sky-300">{{ formatSettlementAmount(tooltipData?.total_cost || 0, 6) }}</span>
            </div>
            <div v-if="tooltipData && tooltipData.cache_creation_cost > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.cacheCreationCost') }}</span>
              <span class="font-medium text-white">{{ formatSettlementAmount(tooltipData.cache_creation_cost, 6) }}</span>
            </div>
            <div v-if="tooltipData && tooltipData.cache_read_cost > 0" class="flex items-center justify-between gap-4">
              <span class="usage-tooltip-label">{{ t('usage.cacheReadCost') }}</span>
              <span class="font-medium text-white">{{ formatSettlementAmount(tooltipData.cache_read_cost, 6) }}</span>
            </div>
          </div>
          <!-- Rate and Summary -->
          <div class="flex items-center justify-between gap-6">
            <span class="usage-tooltip-label">{{ t('usage.serviceTier') }}</span>
            <span class="font-semibold text-cyan-300">{{ getUsageServiceTierLabel(tooltipData?.service_tier, t) }}</span>
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="usage-tooltip-label">{{ t('usage.rate') }}</span>
            <span class="font-semibold text-blue-400"
              >{{ formatMultiplier(tooltipData?.rate_multiplier || 1) }}x</span
            >
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="usage-tooltip-label">{{ t('usage.original') }}</span>
            <span class="font-medium text-white">{{ formatSettlementAmount(tooltipData?.total_cost || 0, 6) }}</span>
          </div>
          <div class="usage-tooltip-divider flex items-center justify-between gap-6 border-t pt-1.5">
            <span class="usage-tooltip-label">{{ t('usage.billed') }}</span>
            <span class="font-semibold text-green-400"
              >{{ formatSettlementAmount(tooltipData?.actual_cost || 0, 6) }}</span
            >
          </div>
        </div>
        <!-- Tooltip Arrow (left side) -->
        <div
          class="usage-tooltip-arrow absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-t-transparent"
        ></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { usageAPI, keysAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import type { UsageLog, ApiKey, UsageQueryParams, UsageStatsResponse, UserErrorRequest } from '@/types'
import type { Column } from '@/components/common/types'
import { formatDateTime, formatReasoningEffort } from '@/utils/format'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useSettlementCurrency } from '@/composables/useSettlementCurrency'
import { formatCacheTokens, formatMultiplier } from '@/utils/formatters'
import { isOpenAICacheReadOnlyUsage } from '@/utils/cacheUsage'
import { calculateTokenPricePerMillion } from '@/utils/usagePricing'
import { getUsageServiceTierLabel } from '@/utils/usageServiceTier'
import { resolveUsageRequestType } from '@/utils/usageRequestType'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  getBillingModeLabel,
  isImageUsage,
  getDisplayBillingMode,
  imageUnitPrice,
} from '@/utils/billingMode'
import {
  formatImageBillingSize,
  formatImageInputSize,
  formatImageOutputSize,
  formatImageSizeBreakdown,
  formatImageSizeSource,
  hasImageOutputTokens,
  textOutputTokens,
  hasImageOutputCost,
} from '@/utils/imageUsage'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const {
  settlementCurrency,
  formatSettlementAmount,
} = useSettlementCurrency()

let abortController: AbortController | null = null

// Tooltip state
const tooltipVisible = ref(false)
const tooltipPosition = ref({ x: 0, y: 0 })
const tooltipData = ref<UsageLog | null>(null)

// Token tooltip state
const tokenTooltipVisible = ref(false)
const tokenTooltipPosition = ref({ x: 0, y: 0 })
const tokenTooltipData = ref<UsageLog | null>(null)

// Usage stats from API
const usageStats = ref<UsageStatsResponse | null>(null)

const usageTrustNotes = computed<Array<{ icon: 'document' | 'calculator' | 'shield'; title: string; description: string }>>(() => [
  {
    icon: 'document',
    title: t('usage.trust.transparentUsage'),
    description: t('usage.trust.transparentUsageDesc')
  },
  {
    icon: 'calculator',
    title: t('usage.trust.auditableBilling'),
    description: t('usage.trust.auditableBillingDesc')
  },
  {
    icon: 'shield',
    title: t('usage.trust.recoverableIssues'),
    description: t('usage.trust.recoverableIssuesDesc')
  }
])

// 缓存命中率 = cache_read / (input + cache_creation + cache_read)
// 分母为 0（无任何输入）时显示 '-'
const cacheStats = computed(() => {
  // 总输入 token = 普通输入 + 缓存写入 + 缓存读取（命中）
  // 缓存命中率 = 缓存读取 / 总输入；总输入为 0 时返回零值，模板按 '-' 渲染。
  const cacheRead = usageStats.value?.total_cache_read_tokens || 0
  const cacheCreate = usageStats.value?.total_cache_creation_tokens || 0
  const input = usageStats.value?.total_input_tokens || 0
  const totalInput = input + cacheCreate + cacheRead
  const ratePercent = totalInput > 0 ? `${((cacheRead / totalInput) * 100).toFixed(1)}%` : '-'
  return { cacheRead, totalInput, ratePercent }
})

const columns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'model', label: t('usage.model'), sortable: true },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'tokens', label: t('usage.tokens'), sortable: false },
  { key: 'cost', label: t('usage.cost'), sortable: false },
  { key: 'duration', label: t('usage.duration'), sortable: false },
  { key: 'created_at', label: t('usage.time'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const usageLogs = ref<UsageLog[]>([])
const apiKeys = ref<ApiKey[]>([])
const loading = ref(false)
const exporting = ref(false)

const apiKeyOptions = computed(() => {
  return [
    { value: null, label: t('usage.allApiKeys') },
    ...apiKeys.value.map((key) => ({
      value: key.id,
      label: key.name
    }))
  ]
})

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

// Initialize date range immediately
const now = new Date()
const weekAgo = new Date(now)
weekAgo.setDate(weekAgo.getDate() - 6)

// Date range state
const startDate = ref(formatLocalDate(weekAgo))
const endDate = ref(formatLocalDate(now))

const filters = ref<UsageQueryParams>({
  api_key_id: undefined,
  start_date: undefined,
  end_date: undefined
})

// Initialize filters with date range
filters.value.start_date = startDate.value
filters.value.end_date = endDate.value

// Handle date range change from DateRangePicker
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  filters.value.start_date = range.startDate
  filters.value.end_date = range.endDate
  applyFilters()
  errorPage.value = 1
  if (activeTab.value === 'errors') {
    loadErrors()
  } else {
    errorRows.value = []  // 失效，下次切到 errors tab 时按新日期重新加载
  }
}

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const formatDuration = (ms: number | null | undefined): string => {
  if (ms == null) return '-'
  if (ms < 1000) return `${ms.toFixed(0)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

const formatSettlementTokenPricePerMillion = (
  cost: number | null | undefined,
  tokens: number | null | undefined,
): string => {
  const price = calculateTokenPricePerMillion(cost, tokens)
  return price == null ? '-' : formatSettlementAmount(price, 4)
}

const formatUserAgent = (ua: string): string => {
  return ua
}

const getRequestTypeLabel = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return t('usage.cyber')
  if (requestType === 'ws_v2') return t('usage.ws')
  if (requestType === 'stream') return t('usage.stream')
  if (requestType === 'sync') return t('usage.sync')
  return t('usage.unknown')
}

function openUsageTicket(row: UsageLog) {
  router.push({
    path: '/tickets',
    query: {
      new: '1',
      context_type: 'usage',
      context_id: String(row.id),
      subject: `${t('usage.title')} #${row.id}`
    }
  })
}

const getRequestTypeBadgeClass = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'usage-badge usage-badge-danger'
  if (requestType === 'ws_v2') return 'usage-badge usage-badge-violet'
  if (requestType === 'stream') return 'usage-badge usage-badge-blue'
  if (requestType === 'sync') return 'usage-badge usage-badge-neutral'
  return 'usage-badge usage-badge-warning'
}

const getUsageBillingModeBadgeClass = (mode: string | null | undefined): string => {
  if (mode === BILLING_MODE_PER_REQUEST) return 'usage-badge usage-badge-violet'
  if (mode === BILLING_MODE_IMAGE) return 'usage-badge usage-badge-pink'
  return 'usage-badge usage-badge-blue'
}


const getRequestTypeExportText = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'Cyber'
  if (requestType === 'ws_v2') return 'WS'
  if (requestType === 'stream') return 'Stream'
  if (requestType === 'sync') return 'Sync'
  return 'Unknown'
}

const formatUsageEndpoints = (log: UsageLog): string => {
  const inbound = log.inbound_endpoint?.trim()
  return inbound || '-'
}

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

type UsageTableQueryParams = UsageQueryParams & {
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

const buildUsageQueryParams = (page: number, pageSize: number): UsageTableQueryParams => ({
  page,
  page_size: pageSize,
  ...filters.value,
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order
})

const loadUsageLogs = async () => {
  if (abortController) {
    abortController.abort()
  }
  const currentAbortController = new AbortController()
  abortController = currentAbortController
  const { signal } = currentAbortController
  loading.value = true
  try {
    const response = await usageAPI.query(
      buildUsageQueryParams(pagination.page, pagination.page_size),
      { signal }
    )
    if (signal.aborted) {
      return
    }
    usageLogs.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
  } catch (error) {
    if (signal.aborted) {
      return
    }
    const abortError = error as { name?: string; code?: string }
    if (abortError?.name === 'AbortError' || abortError?.code === 'ERR_CANCELED') {
      return
    }
    appStore.showError(t('usage.failedToLoad'))
  } finally {
    if (abortController === currentAbortController) {
      loading.value = false
    }
  }
}

const loadApiKeys = async () => {
  try {
    const response = await keysAPI.list(1, 100)
    apiKeys.value = response.items
  } catch (error) {
    console.error('Failed to load API keys:', error)
  }
}

const loadUsageStats = async () => {
  try {
    const apiKeyId = filters.value.api_key_id ? Number(filters.value.api_key_id) : undefined
    const stats = await usageAPI.getStatsByDateRange(
      filters.value.start_date || startDate.value,
      filters.value.end_date || endDate.value,
      apiKeyId
    )
    usageStats.value = stats
  } catch (error) {
    console.error('Failed to load usage stats:', error)
  }
}

const applyFilters = () => {
  pagination.page = 1
  loadUsageLogs()
  loadUsageStats()
}

const resetFilters = () => {
  filters.value = {
    api_key_id: undefined,
    start_date: undefined,
    end_date: undefined
  }
  // Reset date range to default (last 7 days)
  const now = new Date()
  const weekAgo = new Date(now)
  weekAgo.setDate(weekAgo.getDate() - 6)
  startDate.value = formatLocalDate(weekAgo)
  endDate.value = formatLocalDate(now)
  filters.value.start_date = startDate.value
  filters.value.end_date = endDate.value
  pagination.page = 1
  loadUsageLogs()
  loadUsageStats()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadUsageLogs()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadUsageLogs()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadUsageLogs()
}

/**
 * Escape CSV value to prevent injection and handle special characters
 */
const escapeCSVValue = (value: unknown): string => {
  if (value == null) return ''

  const str = String(value)
  const escaped = str.replace(/"/g, '""')

  // Prevent formula injection by prefixing dangerous characters with single quote
  if (/^[=+\-@\t\r]/.test(str)) {
    return `"\'${escaped}"`
  }

  // Escape values containing comma, quote, or newline
  if (/[,"\n\r]/.test(str)) {
    return `"${escaped}"`
  }

  return str
}

const exportToCSV = async () => {
  if (pagination.total === 0) {
    appStore.showWarning(t('usage.noDataToExport'))
    return
  }

  exporting.value = true
  appStore.showInfo(t('usage.preparingExport'))

  try {
    const allLogs: UsageLog[] = []
    const pageSize = 100 // Use a larger page size for export to reduce requests
    const totalRequests = Math.ceil(pagination.total / pageSize)

    for (let page = 1; page <= totalRequests; page++) {
      const response = await usageAPI.query(buildUsageQueryParams(page, pageSize))
      allLogs.push(...response.items)
    }

    if (allLogs.length === 0) {
      appStore.showWarning(t('usage.noDataToExport'))
      return
    }

    const costCurrency = settlementCurrency.value
    const headers = [
      t('usage.exportHeaders.time'),
      t('usage.exportHeaders.credentialName'),
      t('usage.exportHeaders.model'),
      t('usage.exportHeaders.reasoningEffort'),
      t('usage.exportHeaders.inboundEndpoint'),
      t('usage.exportHeaders.type'),
      t('usage.exportHeaders.billingMode'),
      t('usage.exportHeaders.inputTokens'),
      t('usage.exportHeaders.outputTokens'),
      t('usage.exportHeaders.cacheReadTokens'),
      t('usage.exportHeaders.cacheCreationTokens'),
      t('usage.exportHeaders.rateMultiplier'),
      t('usage.exportHeaders.billedCost', { currency: costCurrency }),
      t('usage.exportHeaders.originalCost', { currency: costCurrency }),
      t('usage.exportHeaders.firstTokenMs'),
      t('usage.exportHeaders.durationMs')
    ]
    const rows = allLogs.map((log) =>
      [
        log.created_at,
        log.api_key?.name || '',
        log.model,
        formatReasoningEffort(log.reasoning_effort),
        log.inbound_endpoint || '',
        getRequestTypeExportText(log),
        getBillingModeLabel(getDisplayBillingMode(log), t, 'usage'),
        log.input_tokens,
        log.output_tokens,
        log.cache_read_tokens,
        log.cache_creation_tokens,
        log.rate_multiplier,
        formatSettlementAmount(log.actual_cost, 8),
        formatSettlementAmount(log.total_cost, 8),
        log.first_token_ms ?? '',
        log.duration_ms
      ].map(escapeCSVValue)
    )

    const csvContent = [
      headers.map(escapeCSVValue).join(','),
      ...rows.map((row) => row.join(','))
    ].join('\n')

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage_${filters.value.start_date}_to_${filters.value.end_date}.csv`
    link.click()
    window.URL.revokeObjectURL(url)

    appStore.showSuccess(t('usage.exportSuccess'))
  } catch (error) {
    appStore.showError(t('usage.exportFailed'))
    console.error('CSV Export failed:', error)
  } finally {
    exporting.value = false
  }
}

// Tooltip functions
const tooltipX = (rect: DOMRect) => {
  const maxWidth = Math.min(320, window.innerWidth - 16)
  return Math.max(8, Math.min(rect.right + 8, window.innerWidth - maxWidth - 8))
}

const showTooltip = (event: MouseEvent | FocusEvent, row: UsageLog) => {
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()

  tooltipData.value = row
  // Position to the right of the icon, vertically centered
  tooltipPosition.value.x = tooltipX(rect)
  tooltipPosition.value.y = rect.top + rect.height / 2
  tooltipVisible.value = true
}

const hideTooltip = () => {
  tooltipVisible.value = false
  tooltipData.value = null
}

// Token tooltip functions
const showTokenTooltip = (event: MouseEvent | FocusEvent, row: UsageLog) => {
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()

  tokenTooltipData.value = row
  tokenTooltipPosition.value.x = tooltipX(rect)
  tokenTooltipPosition.value.y = rect.top + rect.height / 2
  tokenTooltipVisible.value = true
}

const hideTokenTooltip = () => {
  tokenTooltipVisible.value = false
  tokenTooltipData.value = null
}

// ── Issue Records Tab ───────────────────────────────────────────────────────
const activeTab = ref<'usage' | 'errors'>('usage')
const errorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false)

const errorRows = ref<UserErrorRequest[]>([])
const errorLoading = ref(false)
const errorPage = ref(1)
const errorPageSize = ref(20)
const errorTotal = ref(0)
const errorFilter = ref<{ model: string; category: string; api_key_id: number | null }>({ model: '', category: '', api_key_id: null })

const loadErrors = async () => {
  errorLoading.value = true
  try {
    const resp = await usageAPI.listMyErrorRequests({
      page: errorPage.value,
      page_size: errorPageSize.value,
      start_date: startDate.value,
      end_date: endDate.value,
      model: errorFilter.value.model || undefined,
      category: errorFilter.value.category || undefined,
      api_key_id: errorFilter.value.api_key_id ?? undefined,
    })
    errorRows.value = resp.items
    errorTotal.value = resp.total
  } catch (error) {
    console.error('[UsageView] loadErrors failed:', error)
    appStore.showError(t('usage.errors.failedToLoad'))
  } finally {
    errorLoading.value = false
  }
}

const onErrorFilter = (f: { model: string; category: string; api_key_id: number | null }) => {
  errorFilter.value = f
  errorPage.value = 1
  loadErrors()
}
const onErrorPage = (p: number) => { errorPage.value = p; loadErrors() }
const onErrorPageSize = (s: number) => { errorPageSize.value = s; errorPage.value = 1; loadErrors() }

const switchToErrors = () => {
  activeTab.value = 'errors'
  if (errorRows.value.length === 0) loadErrors()
}

onMounted(() => {
  loadApiKeys()
  loadUsageLogs()
  loadUsageStats()
})
</script>

<style scoped>
.usage-page-title,
.usage-stat-value,
.usage-cell-strong,
.usage-trust-title {
  color: var(--apple-text);
}

.usage-page-description,
.usage-stat-label,
.usage-stat-muted,
.usage-cell-muted,
.usage-trust-description {
  color: var(--apple-muted);
}

.usage-cell-faint {
  color: var(--apple-muted-2);
}

.usage-trust-card {
  background: color-mix(in srgb, var(--apple-surface-elevated) 72%, var(--apple-surface));
  border-color: var(--apple-border-soft);
  box-shadow: var(--apple-shadow-sm);
}

.usage-trust-icon {
  color: var(--apple-muted-2);
}

.usage-stats-grid {
  background: var(--apple-border-soft);
  border-color: var(--apple-border);
  box-shadow: var(--apple-shadow-sm);
}

.usage-stat-card {
  background: var(--apple-surface);
}

.usage-tabs {
  background: var(--apple-surface-elevated);
  border: 1px solid var(--apple-border-soft);
}

.usage-tabs :deep(.tab) {
  color: var(--apple-muted);
  letter-spacing: 0;
}

.usage-tabs :deep(.tab:hover) {
  color: var(--apple-text);
  background: var(--apple-hover);
}

.usage-tabs :deep(.tab-active) {
  color: var(--apple-text);
  background: var(--apple-surface);
  box-shadow: var(--apple-shadow-sm);
}

.usage-table-panel :deep(.table-header) {
  background: var(--apple-surface-elevated);
}

.usage-table-panel :deep(.sticky-header-cell) {
  color: var(--apple-muted);
  border-color: var(--apple-border);
}

.usage-table-panel :deep(.sticky-header-cell:hover) {
  background: var(--apple-hover);
}

.usage-table-panel :deep(.table-body) {
  border-color: var(--apple-border-soft);
}

.usage-table-panel :deep(.data-table-row) {
  background: var(--apple-surface);
}

.usage-table-panel :deep(.data-table-row:hover) {
  background: color-mix(in srgb, var(--apple-blue) 5%, var(--apple-surface));
}

.usage-table-panel :deep(td) {
  border-color: var(--apple-border-soft);
}

.usage-table-panel :deep(.data-table-mobile-card) {
  background: var(--apple-surface);
  border-color: var(--apple-border);
  box-shadow: var(--apple-shadow-sm);
}

.usage-table-panel :deep(.data-table-mobile-card > div) {
  min-width: 0;
}

.usage-table-panel :deep(.data-table-mobile-actions) {
  border-color: var(--apple-border-soft);
}

.usage-info-dot {
  background: var(--apple-surface-elevated);
  box-shadow: inset 0 0 0 1px var(--apple-border-soft);
}

.group:hover .usage-info-dot,
.group:focus-visible .usage-info-dot {
  background: color-mix(in srgb, var(--apple-blue) 12%, var(--apple-surface));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--apple-blue) 26%, var(--apple-border));
}

.usage-info-icon {
  color: var(--apple-muted-2);
}

.group:hover .usage-info-icon,
.group:focus-visible .usage-info-icon {
  color: var(--apple-blue);
}

.usage-row-action {
  color: var(--apple-blue);
}

.usage-row-action:hover {
  background: color-mix(in srgb, var(--apple-blue) 9%, var(--apple-surface));
}

.usage-badge {
  border: 1px solid color-mix(in srgb, currentColor 20%, transparent);
}

.usage-badge-blue {
  color: #0066cc;
  background: color-mix(in srgb, #0071e3 10%, var(--apple-surface));
}

.usage-badge-violet {
  color: #6e38b1;
  background: color-mix(in srgb, #8e44ad 10%, var(--apple-surface));
}

.usage-badge-pink {
  color: #c42a65;
  background: color-mix(in srgb, #ff2d55 10%, var(--apple-surface));
}

.usage-badge-danger {
  color: var(--apple-danger);
  background: color-mix(in srgb, var(--apple-danger) 10%, var(--apple-surface));
}

.usage-badge-warning {
  color: var(--apple-warning);
  background: color-mix(in srgb, var(--apple-warning) 11%, var(--apple-surface));
}

.usage-badge-neutral {
  color: var(--apple-muted);
  background: var(--apple-surface-elevated);
}

.usage-tooltip {
  --usage-tooltip-bg: color-mix(in srgb, var(--apple-surface) 94%, var(--apple-text) 6%);
  --usage-tooltip-border: var(--apple-border);
  --usage-tooltip-muted: var(--apple-muted);
  --usage-tooltip-strong: var(--apple-text);

  color: var(--usage-tooltip-strong);
  background: var(--usage-tooltip-bg);
  border-color: var(--usage-tooltip-border);
  box-shadow: var(--apple-shadow-md);
}

.usage-tooltip-title {
  color: var(--usage-tooltip-strong);
}

.usage-tooltip-label {
  color: var(--usage-tooltip-muted);
}

.usage-tooltip-section,
.usage-tooltip-divider {
  border-color: var(--apple-border-soft);
}

.usage-tooltip :is(.text-white, .text-blue-400, .text-green-400) {
  color: var(--usage-tooltip-strong);
}

.usage-tooltip-arrow {
  border-right-color: var(--usage-tooltip-bg);
}

:global(.dark) .usage-trust-card,
:global(.dark) .usage-stat-card,
:global(.dark) .usage-table-panel :deep(.data-table-row),
:global(.dark) .usage-table-panel :deep(.data-table-mobile-card) {
  background: color-mix(in srgb, var(--apple-surface) 94%, white 6%);
}

:global(.dark) .usage-badge-blue {
  color: #8ecbff;
  background: color-mix(in srgb, var(--apple-blue) 18%, var(--apple-surface));
}

:global(.dark) .usage-badge-violet {
  color: #d8b4fe;
  background: color-mix(in srgb, #a855f7 18%, var(--apple-surface));
}

:global(.dark) .usage-badge-pink {
  color: #ffb3c7;
  background: color-mix(in srgb, #ff2d55 18%, var(--apple-surface));
}

:global(.dark) .usage-badge-danger {
  background: color-mix(in srgb, var(--apple-danger) 18%, var(--apple-surface));
}

:global(.dark) .usage-badge-warning {
  background: color-mix(in srgb, var(--apple-warning) 18%, var(--apple-surface));
}

:global(.dark) .usage-tooltip {
  --usage-tooltip-bg: color-mix(in srgb, var(--apple-surface) 88%, white 12%);
  --usage-tooltip-border: var(--apple-border);
  --usage-tooltip-muted: var(--apple-muted);
  --usage-tooltip-strong: var(--apple-text);
}

@media (max-width: 639px) {
  .usage-stats-grid {
    gap: 0.5rem;
    background: transparent;
    border: 0;
    box-shadow: none;
  }

  .usage-stat-card {
    border: 1px solid var(--apple-border-soft);
    border-radius: var(--apple-radius);
    box-shadow: var(--apple-shadow-sm);
  }
}
</style>
