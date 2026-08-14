<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl">
      <div class="space-y-4">
        <!-- 顶部：分组选择 -->
        <div
          class="card flex flex-col items-stretch justify-between gap-3 px-3 py-3 sm:flex-row flex-wrap sm:items-end sm:px-4"
        >
          <div class="flex w-full flex-wrap items-end gap-3 sm:w-auto">
            <div class="min-w-0 flex-1 sm:flex-none">
              <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.scheduler.group') }}
              </label>
              <Select
                v-model="selectedGroupId"
                :options="groupOptions"
                class="w-full sm:w-64"
                :placeholder="t('admin.scheduler.selectGroup')"
                @update:model-value="onGroupChange"
              />
            </div>
          </div>
          <div class="grid w-full grid-cols-2 gap-2 sm:flex sm:min-w-0 sm:flex-1 flex-wrap sm:items-center sm:justify-end">
            <button
              type="button"
              data-testid="scheduler-bulk-monitor-model"
              class="btn btn-secondary btn-sm min-w-0 justify-center"
              :disabled="!selectedGroupId || entries.length === 0 || bulkMonitorModelSaving"
              :title="t('admin.scheduler.bulkMonitorModel')"
              @click="showBulkMonitorModelDialog = true"
            >
              <Icon name="cog" size="xs" />
              {{ t('admin.scheduler.bulkMonitorModel') }}
            </button>
            <!-- 分组级持续自动排序：后端按稳定性与模型成功率统一计算。 -->
            <div
              class="col-span-2 inline-flex min-w-0 items-center justify-between gap-2 rounded-xl border border-gray-200 bg-white px-2.5 py-1.5 text-xs dark:border-dark-600 dark:bg-dark-800 sm:col-auto sm:justify-start"
              :title="t('admin.scheduler.autoSortHint')"
            >
              <Toggle
                data-testid="scheduler-auto-sort-toggle"
                :model-value="autoSortEnabled"
                :disabled="!selectedGroupId || autoSortSaving"
                :aria-label="t('admin.scheduler.autoSort')"
                @update:model-value="onToggleAutoSort"
              />
              <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">
                {{ t('admin.scheduler.autoSort') }}
              </span>
              <span
                data-testid="scheduler-auto-sort-policy"
                class="whitespace-nowrap rounded-md bg-primary-50 px-2 py-1 text-[11px] font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
              >
                {{ t('admin.scheduler.autoSortPolicy') }}
              </span>
            </div>
            <button
              type="button"
              data-testid="scheduler-refresh-order"
              class="btn btn-secondary btn-sm min-w-0 justify-center"
              :disabled="!selectedGroupId || loading || saving || orderDirty"
              :title="t('admin.scheduler.refreshOrderHint')"
              @click="refreshSchedulerOrder"
            >
              <Icon name="refresh" size="xs" :class="loading ? 'animate-spin' : ''" />
              {{ t('admin.scheduler.refreshOrder') }}
            </button>
            <button
              type="button"
              class="btn btn-secondary btn-sm min-w-0 justify-center"
              :disabled="!selectedGroupId || upstreamLoading"
              @click="refreshUpstreamStatuses(true)"
            >
              <Icon name="refresh" size="xs" :class="upstreamLoading ? 'animate-spin' : ''" />
              {{ t('admin.scheduler.refreshUpstream') }}
            </button>
            <button
              type="button"
              data-testid="scheduler-save-order"
              class="btn btn-primary btn-sm justify-center"
              :disabled="!selectedGroupId || saving || loading || entries.length === 0"
              @click="saveOrder"
            >
              <Icon v-if="saving" name="refresh" size="xs" class="mr-1 animate-spin" />
              {{ saving ? t('common.saving') : t('admin.scheduler.saveOrder') }}
            </button>
          </div>
        </div>

        <p class="px-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduler.orderHint') }}
        </p>

        <div
          v-if="selectedGroupId && entries.length > 0"
          class="card space-y-3 px-3 py-3 sm:px-4"
        >
          <div class="flex flex-col items-stretch justify-between gap-3 sm:flex-row flex-wrap sm:items-center">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <button
                v-for="item in schedulerStateFilters"
                :key="item.value"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors"
                :class="stateFilter === item.value ? 'border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-700/60 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700'"
                @click="stateFilter = item.value"
              >
                <span :class="item.dotClass" class="h-2 w-2 rounded-full"></span>
                <span>{{ item.label }}</span>
                <span class="tabular-nums text-[10px] opacity-70">{{ item.count }}</span>
              </button>
            </div>
            <div class="flex w-full flex-wrap items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400 sm:w-auto sm:shrink-0 sm:justify-start">
              <span
                v-if="insufficientBalanceCount > 0"
                class="rounded-md border border-red-200 bg-red-50 px-2 py-1 font-medium text-red-700 dark:border-red-900/60 dark:bg-red-950/40 dark:text-red-200"
                :title="t('admin.scheduler.insufficientBalanceSummaryTitle')"
              >
                {{ t('admin.scheduler.insufficientBalanceSummary', { count: insufficientBalanceCount }) }}
              </span>
              <span
                class="rounded-md border border-indigo-100 bg-indigo-50 px-2 py-1 font-medium text-indigo-700 dark:border-indigo-900/50 dark:bg-indigo-950/40 dark:text-indigo-200"
                :title="recentGroupFirstTokenTitle"
              >
                {{ recentGroupFirstTokenText }}
              </span>
              <span>{{ t('admin.scheduler.visibleAccounts', { shown: displayedEntries.length, total: entries.length }) }}</span>
            </div>
          </div>

          <div class="flex flex-col items-stretch justify-between gap-3 border-t border-gray-200 pt-3 dark:border-dark-600 sm:flex-row flex-wrap sm:items-center">
            <label class="inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
              <input
                type="checkbox"
                class="h-3.5 w-3.5"
                :checked="allVisibleSelected"
                :disabled="displayedEntries.length === 0"
                @change="toggleSelectAllVisible"
              />
              {{ t('admin.scheduler.selectVisible') }}
            </label>
            <div class="grid grid-cols-2 gap-2 sm:flex flex-wrap sm:items-center">
              <span class="col-span-2 text-xs text-gray-500 dark:text-gray-400 sm:col-auto">
                {{ t('admin.scheduler.selectedCount', { count: selectedEntries.length }) }}
              </span>
              <button
                type="button"
                class="btn btn-secondary btn-sm min-w-0"
                :disabled="selectedAccountCooldownEntries.length === 0 || bulkActionBusy"
                @click="bulkClearTempUnsched"
              >
                <Icon name="xCircle" size="xs" />
                {{ t('admin.scheduler.bulkClearTempUnsched') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary btn-sm min-w-0"
                :disabled="selectedMonitorableEntries.length === 0 || bulkActionBusy"
                @click="bulkEnableMonitor"
              >
                <Icon name="chart" size="xs" />
                {{ t('admin.scheduler.bulkEnableMonitor') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary btn-sm min-w-0"
                :disabled="selectedEntries.length === 0 || bulkActionBusy || upstreamLoading"
                @click="bulkRefreshUpstream"
              >
                <Icon name="refresh" size="xs" :class="bulkActionBusy || upstreamLoading ? 'animate-spin' : ''" />
                {{ t('admin.scheduler.bulkRefreshUpstream') }}
              </button>
            </div>
          </div>
        </div>

        <!-- 主体 -->
        <div
          v-if="!selectedGroupId"
          class="rounded-lg border border-dashed border-gray-300 py-16 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
        >
          {{ t('admin.scheduler.pickGroupFirst') }}
        </div>
        <div v-else-if="loading" class="py-16 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduler.loading') }}
        </div>
        <div
          v-else-if="entries.length === 0"
          class="rounded-lg border border-dashed border-gray-300 py-16 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
        >
          {{ t('admin.scheduler.empty') }}
        </div>
        <section
          v-else
          class="card overflow-hidden"
        >
          <div
            v-if="displayedEntries.length === 0"
            class="p-8 text-center text-sm text-gray-500 dark:text-gray-400"
          >
            {{ t('admin.scheduler.noFilteredAccounts') }}
          </div>
          <VueDraggable
            v-else
            v-model="displayedEntries"
            :animation="180"
            :disabled="stateFilter !== 'all'"
            item-key="account_id"
            handle=".drag-handle"
            class="min-h-40 space-y-2 p-2 sm:p-3"
            @end="orderDirty = true"
          >
            <div
              v-for="(entry, index) in displayedEntries"
              :key="entry.account_id"
              :data-testid="`scheduler-account-row-${entry.account_id}`"
              :class="[
                'flex items-stretch gap-1.5 rounded-md border px-2 py-2 text-xs transition-colors dark:border-dark-600 sm:gap-2 sm:py-1.5',
                selectedAccountIds.has(entry.account_id)
                  ? 'border-primary-400 bg-primary-50 dark:border-primary-500/70 dark:bg-primary-900/20'
                  : 'border-gray-200 bg-gray-50 dark:bg-dark-700/70',
              ]"
            >
              <!-- 拖拽柄 + 序号 -->
              <div class="flex shrink-0 flex-col items-center justify-center gap-1 sm:flex-row">
                <input
                  type="checkbox"
                  class="h-3.5 w-3.5"
                  :checked="selectedAccountIds.has(entry.account_id)"
                  :title="t('admin.scheduler.selectAccount')"
                  @change="toggleEntrySelected(entry)"
                />
                <div
                  :data-testid="`scheduler-drag-handle-${entry.account_id}`"
                  class="drag-handle text-gray-400 active:cursor-grabbing"
                  :class="stateFilter === 'all' ? 'cursor-grab' : 'cursor-not-allowed opacity-40'"
                  :title="stateFilter === 'all' ? '' : t('admin.scheduler.dragDisabledWhenFiltered')"
                >
                  <Icon name="menu" size="sm" />
                </div>
                <span class="w-6 text-center text-gray-400 sm:text-right">{{ index + 1 }}</span>
              </div>

              <!-- 两行主体 -->
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <!-- 第 1 行：账号信息 + 状态 + 余额 + 倍率 + 调度开关 + 配置 -->
                <div class="flex min-w-0 flex-wrap items-center gap-1.5" :title="entryTooltip(entry)">
                  <span :class="headDotClass(entry)" class="h-2 w-2 shrink-0 rounded-full"></span>
                  <span class="min-w-24 flex-1 truncate font-medium text-gray-900 dark:text-white sm:min-w-0 sm:flex-none">{{ accountName(entry) }}</span>
                  <span v-if="schedulerStateLabel(entry)" class="badge badge-gray shrink-0 text-[10px]">
                    {{ schedulerStateLabel(entry) }}
                  </span>
                  <span
                    v-if="hasInsufficientBalance(entry)"
                    class="badge shrink-0 border border-red-200 bg-red-50 text-[10px] text-red-700 dark:border-red-900/60 dark:bg-red-950/40 dark:text-red-200"
                    :title="insufficientBalanceTitle(entry)"
                  >
                    {{ t('admin.scheduler.insufficientBalance') }}
                  </span>
                  <span
                    class="shrink-0 text-cyan-600 dark:text-cyan-300"
                    :title="t('admin.scheduler.currentConcurrency')"
                  >{{ concurrencyUsageText(entry) }}</span>
                  <span
                    class="shrink-0 text-indigo-600 dark:text-indigo-300"
                    :title="recentFirstTokenTitle(entry)"
                  >{{ recentFirstTokenText(entry) }}</span>
                  <span
                    v-if="rateMultiplierText(entry)"
                    class="shrink-0"
                    :class="rateMultiplierClass(entry)"
                    :title="rateMultiplierTitle(entry)"
                  >{{ rateMultiplierText(entry) }}</span>
                  <span
                    v-if="balanceText(entry)"
                    class="shrink-0"
                    :class="balanceClass(entry)"
                    :title="balanceTitle(entry)"
                  >
                    {{ balanceText(entry) }}
                  </span>
                  <span v-else-if="upstreamLoading" class="shrink-0 text-gray-300 dark:text-gray-600">…</span>
                  <span v-if="tempUnschedActive(entry)" class="shrink-0 text-orange-500" :title="t('admin.scheduler.tempUnsched')">●</span>
                  <span
                    v-if="activeModelCooldowns(entry).length > 0"
                    class="badge badge-warning shrink-0 text-[10px]"
                    :title="modelCooldownTooltip(entry)"
                  >
                    {{ modelCooldownBadge(entry) }}
                  </span>
                  <span
                    v-if="accountTempUnschedActive(entry)"
                    class="badge badge-warning shrink-0 text-[10px]"
                    :title="accountCooldownTooltip(entry)"
                  >
                    {{ t('admin.scheduler.accountCooldown') }}
                  </span>
                  <span
                    v-if="groupReserveActive(entry)"
                    class="badge shrink-0 border border-sky-200 bg-sky-50 text-[10px] text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-200"
                    :title="groupReserveTooltip(entry)"
                  >
                    {{ t('admin.scheduler.groupReserve') }}
                  </span>

                  <span class="ml-auto flex shrink-0 items-center gap-1.5">
                    <Toggle
                      :title="t('admin.scheduler.schedulable')"
                      :aria-label="t('admin.scheduler.schedulable')"
                      :model-value="isSchedulable(entry)"
                      :disabled="schedulableBusyIds.has(entry.account_id)"
                      :data-testid="`scheduler-schedulable-toggle-${entry.account_id}`"
                      @update:model-value="(value: boolean) => toggleSchedulable(entry, value)"
                    />
                    <button
                      type="button"
                      :data-testid="`scheduler-account-detail-${entry.account_id}`"
                      class="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600"
                      :title="t('admin.scheduler.viewDetail')"
                      @click="openSchedulerDetail(entry)"
                    >
                      <Icon name="eye" size="xs" />
                    </button>
                    <button
                      type="button"
                      :data-testid="`scheduler-account-config-${entry.account_id}`"
                      class="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600"
                      :title="t('admin.scheduler.accountConfig')"
                      @click="openSchedulerConfig(entry)"
                    >
                      <Icon name="cog" size="xs" />
                    </button>
                  </span>
                </div>

                <!-- 第 2 行：监控开关 + 彩虹状态条（仅当监控开启且有探测历史） -->
                <div class="flex min-w-0 items-center gap-2">
                  <Toggle
                    v-if="canMonitor(entry)"
                    :title="t('admin.scheduler.monitor')"
                    :aria-label="t('admin.scheduler.monitor')"
                    :model-value="monitorEnabled(entry.account_id)"
                    :disabled="monitorBusyIds.has(entry.account_id)"
                    :data-testid="`scheduler-monitor-toggle-${entry.account_id}`"
                    @update:model-value="(value: boolean) => toggleMonitor(entry, value)"
                  />
                  <span v-else class="w-11 shrink-0"></span>
                  <div
                    v-if="canMonitor(entry) && monitorEnabled(entry.account_id) && hasMonitorTimeline(entry.account_id)"
                    class="grid h-3 flex-1 items-end gap-px overflow-hidden"
                    :style="{ gridTemplateColumns: `repeat(${miniBars(entry.account_id).length}, minmax(0, 1fr))` }"
                  >
                    <div
                      v-for="(bar, bi) in miniBars(entry.account_id)"
                      :key="bi"
                      class="min-w-0 rounded-sm"
                      :class="bar.cls"
                      :style="{ height: bar.h + '%' }"
                      :title="bar.title"
                    ></div>
                  </div>
                  <span v-else-if="canMonitor(entry) && monitorEnabled(entry.account_id)" class="text-[10px] text-gray-300 dark:text-gray-600">{{ t('admin.scheduler.monitorNoData') }}</span>
                  <span v-else-if="canMonitor(entry)" class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.monitorOff') }}</span>
                </div>
              </div>

              <div v-if="accountTempUnschedActive(entry)" class="flex shrink-0 items-center">
                <!-- 清冷却 -->
                <button
                  type="button"
                  class="shrink-0 rounded border border-orange-200 px-1 py-0.5 text-[10px] text-orange-600 hover:bg-orange-50 dark:border-orange-900/50 dark:text-orange-300"
                  :title="t('admin.scheduler.clearTempUnsched')"
                  @click="clearTempUnsched(entry)"
                >
                  {{ t('admin.scheduler.clearTempUnsched') }}
                </button>
              </div>
            </div>
          </VueDraggable>
        </section>
      </div>

    </div>
  </AppLayout>

  <BaseDialog
    :show="showSchedulerConfigDialog"
    :title="schedulerConfigTitle"
    width="normal"
    @close="closeSchedulerConfig"
  >
    <form class="space-y-4" data-testid="scheduler-account-config-dialog" @submit.prevent="saveSchedulerConfig">
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.scheduler.capacityLimit') }}
          </span>
          <input
            v-model.number="schedulerConfigForm.concurrency"
            data-testid="scheduler-config-capacity"
            type="number"
            min="0"
            step="1"
            class="input"
          />
        </label>
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.scheduler.loadFactor') }}
          </span>
          <input
            v-model.number="schedulerConfigForm.load_factor"
            data-testid="scheduler-config-load-factor"
            type="number"
            min="0"
            step="1"
            class="input"
            :placeholder="t('admin.scheduler.useCapacityDefault')"
          />
        </label>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.scheduler.billingRateMultiplier') }}
          </span>
          <input
            v-model.number="schedulerConfigForm.rate_multiplier"
            data-testid="scheduler-config-billing-rate"
            type="number"
            min="0"
            step="0.0001"
            class="input"
            :placeholder="t('admin.scheduler.emptyAuto')"
          />
        </label>
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.scheduler.monitorModel') }}
          </span>
          <input
            v-model.trim="schedulerConfigForm.monitor_model"
            data-testid="scheduler-config-monitor-model"
            type="text"
            class="input"
            :disabled="!schedulerConfigCanMonitor"
            :placeholder="DEFAULT_MONITOR_MODEL"
          />
        </label>
      </div>

      <div class="rounded-md border border-gray-200 p-3 dark:border-dark-600">
        <div class="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduler.rateDisplayConfig') }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="block">
            <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.scheduler.manualRate') }}
            </span>
            <input
              v-model.number="schedulerConfigForm.manual_rate"
              data-testid="scheduler-config-manual-rate"
              type="number"
              min="0"
              step="0.0001"
              class="input"
              :placeholder="t('admin.scheduler.emptyAuto')"
            />
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.scheduler.rateScale') }}
            </span>
            <input
              v-model.number="schedulerConfigForm.rate_scale"
              data-testid="scheduler-config-rate-scale"
              type="number"
              min="0"
              step="0.0001"
              class="input"
              :placeholder="t('admin.scheduler.emptyAuto')"
            />
          </label>
        </div>
      </div>

      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" :disabled="schedulerConfigSaving" @click="closeSchedulerConfig">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" class="btn btn-primary" :disabled="schedulerConfigSaving">
          <Icon v-if="schedulerConfigSaving" name="refresh" size="xs" class="mr-1 animate-spin" />
          {{ schedulerConfigSaving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </form>
  </BaseDialog>

  <BaseDialog
    :show="showBulkMonitorModelDialog"
    :title="t('admin.scheduler.bulkMonitorModel')"
    width="narrow"
    @close="showBulkMonitorModelDialog = false"
  >
    <div class="space-y-4" data-testid="scheduler-bulk-monitor-model-dialog">
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduler.monitorModel') }}
        </label>
        <input
          v-model.trim="bulkMonitorModel"
          data-testid="scheduler-bulk-monitor-model-input"
          type="text"
          class="input"
          :placeholder="DEFAULT_MONITOR_MODEL"
          @keyup.enter="applyBulkMonitorModel"
        />
      </div>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="showBulkMonitorModelDialog = false">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="bulkMonitorModelSaving || !bulkMonitorModel.trim()"
          @click="applyBulkMonitorModel"
        >
          {{ bulkMonitorModelSaving ? t('common.saving') : t('common.confirm') }}
        </button>
      </div>
    </div>
  </BaseDialog>

  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="detailEntry"
        class="fixed inset-0 z-50 bg-black/35"
        @click.self="closeSchedulerDetail"
      >
        <aside class="ml-auto flex h-full w-full max-w-[34rem] flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900">
          <div class="flex items-start justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <div class="min-w-0">
              <p class="text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
                {{ t('admin.scheduler.detailTitle') }}
              </p>
              <h3 class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white">
                {{ accountName(detailEntry) }}
              </h3>
              <p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                {{ detailEntry.account?.name || '-' }} · {{ schedulerStateLabel(detailEntry) }}
              </p>
            </div>
            <button type="button" class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-800" @click="closeSchedulerDetail">
              <Icon name="x" size="sm" />
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.stateLabel') }}</div>
                <div class="mt-1 flex items-center gap-2">
                  <span :class="statusDotClass(schedulerState(detailEntry))" class="h-2.5 w-2.5 rounded-full"></span>
                  <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ schedulerStateLabel(detailEntry) }}</span>
                </div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ entryTooltip(detailEntry) }}</div>
              </div>
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.monitorLatest') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ monitorLatestLatencyText(detailEntry) }}</div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.scheduler.monitorPing') }}: {{ monitorPingLatencyText(detailEntry) }}
                </div>
              </div>
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.recentFirstToken5m') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ recentFirstTokenValueText(detailEntry) }}</div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ recentFirstTokenTitle(detailEntry) }}</div>
              </div>
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.monitorAvailability1h') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ monitorAvailabilityText(detailEntry.account_id) }}</div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduler.monitorAvgLatency1h') }}: {{ monitorAvgLatencyText(detailEntry) }}</div>
              </div>
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.timeline') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ recentCheckedText(detailEntry) }}</div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduler.recentFailed') }}: {{ recentFailedText(detailEntry) }}</div>
              </div>
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.modelCooldownTitle') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ modelCooldownSummary(detailEntry) }}</div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ modelCooldownDetail(detailEntry) }}</div>
              </div>
              <div class="rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[10px] font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.groupReserveTitle') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ groupReserveSummary(detailEntry) }}</div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ groupReserveDetail(detailEntry) }}</div>
              </div>
            </div>

            <div class="mt-4 rounded-md border border-gray-200 p-3 dark:border-dark-700">
              <div class="mb-2 text-xs font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.totals') }}</div>
              <div class="grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
                <div class="rounded border border-gray-200 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-800/60">
                  <div class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.priorityLabel') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ detailEntry.account?.priority ?? '-' }}</div>
                </div>
                <div class="rounded border border-gray-200 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-800/60">
                  <div class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.capacityLimit') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ capacityText(detailEntry) }}</div>
                </div>
                <div class="rounded border border-gray-200 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-800/60">
                  <div class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.currentConcurrency') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ currentConcurrencyText(detailEntry) }}</div>
                </div>
                <div class="rounded border border-gray-200 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-800/60">
                  <div class="text-[10px] text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.recentErrorHold') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ activeDurationText(detailEntry) }}</div>
                </div>
              </div>
            </div>

            <div class="mt-4">
              <div class="mb-2 text-xs font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.monitorTimeline') }}</div>
              <div class="grid h-24 items-end gap-px overflow-hidden rounded-md border border-gray-200 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-800/60" :style="{ gridTemplateColumns: `repeat(${miniBars(detailEntry.account_id).length}, minmax(0, 1fr))` }">
                <div
                  v-for="(bar, bi) in miniBars(detailEntry.account_id)"
                  :key="bi"
                  class="min-w-0 rounded-sm"
                  :class="bar.cls"
                  :style="{ height: bar.h + '%' }"
                  :title="bar.title"
                ></div>
              </div>
            </div>

            <div class="mt-4 rounded-md border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <div class="text-xs font-semibold uppercase text-gray-400 dark:text-gray-500">{{ t('admin.scheduler.historyTitle') }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduler.historyHint') }}</div>
                </div>
                <div class="flex flex-wrap items-center gap-2">
                  <input v-model.trim="historyQuery" type="search" class="input h-8 w-40 text-xs" :placeholder="t('admin.scheduler.historySearch')" />
                  <Select v-model="historyEventFilter" :options="historyEventOptions" class="w-44" />
                </div>
              </div>
              <div v-if="historyLoading" class="py-6 text-center text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.scheduler.historyLoading') }}
              </div>
              <div v-else-if="historyVisibleItems.length === 0" class="py-6 text-center text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.scheduler.historyEmpty') }}
              </div>
              <div v-else class="mt-3 space-y-2">
                <div
                  v-for="item in historyVisibleItems"
                  :key="item.id"
                  class="rounded border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-800/60"
                >
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="badge badge-gray text-[10px]">{{ historyItemTypeLabel(item) }}</span>
                        <span class="text-[11px] text-gray-500 dark:text-gray-400">{{ formatHistoryTime(item.created_at) }}</span>
                      </div>
                      <div class="mt-1 text-xs text-gray-900 dark:text-white">
                        {{ historyItemCompactLabel(item) }}
                      </div>
                    </div>
                    <span class="shrink-0 text-[10px] text-gray-400 dark:text-gray-500">{{ historyItemCountLabel(item) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/custom/api/admin'
import { accountMonitorAPI } from '@/custom/api/admin/accountMonitor'
import type { UpstreamSub2APIAccountStatus } from '@/custom/api/admin/accounts'
import type {
  AccountSchedulingEntry,
  AdminGroup,
  AccountPlatform,
  Account,
} from '@/types'
import {
  getAccountScheduling,
  getAccountSchedulingHistory,
  updateAccountScheduling,
  type GroupSchedulerHistoryItem
} from '@/custom/api/admin/groups'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { VueDraggable } from 'vue-draggable-plus'
import type { AccountMonitor, AccountMonitorStatus } from '@/custom/api/admin/accountMonitor'

const DEFAULT_MONITOR_MODEL = 'gpt-5.4-mini'
const DEFAULT_MONITOR_INTERVAL = 60
const MONITOR_POLL_MS = 30000
const UPSTREAM_STATUS_POLL_MS = 30000
const AUTO_SORT_REFRESH_MS = 65000
type SchedulerStateFilter =
  | 'all'
  | 'active'
  | 'stopped'
  | 'error'
  | 'rate_limited'
  | 'overloaded'
  | 'temp_unschedulable'
  | 'account_cooldown'
  | 'model_unavailable'
  | 'quota_exceeded'
  | 'unknown'
type SchedulerStateInput = AccountSchedulingEntry | string | null | undefined
type SchedulerHistoryViewItem =
  | { kind: 'outbox'; id: string; created_at: string; item: GroupSchedulerHistoryItem }
  | { kind: 'monitor'; id: string; created_at: string; status: string; latency_ms: number | null; message?: string }
interface ActiveModelCooldown {
  model: string
  resetAt: Date
  reason?: string
}
const { t } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()

const MONITORABLE_PLATFORMS: AccountPlatform[] = ['openai', 'anthropic', 'gemini']

const groups = ref<AdminGroup[]>([])
const selectedGroupId = ref<number | null>(null)
const entries = ref<AccountSchedulingEntry[]>([])
const orderDirty = ref(false)
const loading = ref(false)
const saving = ref(false)
const schedulableBusyIds = ref<Set<number>>(new Set())
const showBulkMonitorModelDialog = ref(false)
const bulkMonitorModel = ref(DEFAULT_MONITOR_MODEL)
const bulkMonitorModelSaving = ref(false)
const stateFilter = ref<SchedulerStateFilter>('all')
const selectedAccountIds = ref<Set<number>>(new Set())
const bulkActionBusy = ref(false)
const detailEntry = ref<AccountSchedulingEntry | null>(null)
const historyLoading = ref(false)
const historyItems = ref<GroupSchedulerHistoryItem[]>([])
const historyQuery = ref('')
type HistoryEventFilter = 'all' | 'account_changed' | 'account_groups_changed' | 'account_bulk_changed' | 'account_last_used' | 'scheduling_blocked' | 'scheduling_block_skipped' | 'group_changed' | 'full_rebuild' | 'other'
const historyEventFilter = ref<HistoryEventFilter | 'account_monitor'>('all')

interface SchedulerConfigForm {
  concurrency: number
  load_factor: number | null
  rate_multiplier: number | null
  manual_rate: number | null
  rate_scale: number | null
  monitor_model: string
}

const showSchedulerConfigDialog = ref(false)
const schedulerConfigEntry = ref<AccountSchedulingEntry | null>(null)
const schedulerConfigSaving = ref(false)
const schedulerConfigForm = ref<SchedulerConfigForm>({
  concurrency: 0,
  load_factor: null,
  rate_multiplier: null,
  manual_rate: null,
  rate_scale: null,
  monitor_model: DEFAULT_MONITOR_MODEL,
})

const upstreamStatusMap = ref<Map<number, UpstreamSub2APIAccountStatus>>(new Map())
const upstreamLoading = ref(false)
let upstreamReqSeq = 0
let upstreamInteractiveReqSeq = 0
let upstreamStatusPollTimer: ReturnType<typeof setInterval> | null = null

// 账号监控：account_id -> 配置 monitor / 聚合状态 status / 操作中
const monitorMap = ref<Map<number, AccountMonitor>>(new Map())
const monitorStatus = ref<Map<number, AccountMonitorStatus>>(new Map())
const monitorBusyIds = ref<Set<number>>(new Set())
let monitorPollTimer: ReturnType<typeof setInterval> | null = null

const groupOptions = computed(() =>
  groups.value.map((g) => ({ label: g.name, value: g.id }))
)

// ---- 分组级持续自动排序（每分钟后端定时任务）----
const autoSortSaving = ref(false)
const selectedGroup = computed(() => groups.value.find((g) => g.id === selectedGroupId.value) || null)
const insufficientBalanceCount = computed(() => entries.value.filter(hasInsufficientBalance).length)
const autoSortEnabled = computed(() => selectedGroup.value?.auto_sort_config?.enabled === true)
// Every group uses the same backend policy: recent stability and model success
// first, with latency/load/cost only as bounded tie-breakers. Keep this value
// local so legacy group configs cannot make the page offer a second ordering
// policy that the scheduler does not actually use anymore.
const AUTO_SORT_BASIS = 'experience' as const
let autoSortPollTimer: ReturnType<typeof setInterval> | null = null

function syncAutoSortPolling() {
  if (autoSortPollTimer) {
    clearInterval(autoSortPollTimer)
    autoSortPollTimer = null
  }
  if (!selectedGroupId.value || !autoSortEnabled.value) return
  autoSortPollTimer = setInterval(() => {
    if (
      document.hidden ||
      loading.value ||
      saving.value ||
      orderDirty.value ||
      stateFilter.value !== 'all'
    ) return
    void loadEntries()
  }, AUTO_SORT_REFRESH_MS)
}

watch([selectedGroupId, autoSortEnabled], syncAutoSortPolling)
const stateCounts = computed(() => {
  const counts: Record<SchedulerStateFilter, number> = {
    all: entries.value.length,
    active: 0,
    stopped: 0,
    error: 0,
    rate_limited: 0,
    overloaded: 0,
    temp_unschedulable: 0,
    account_cooldown: 0,
    model_unavailable: 0,
    quota_exceeded: 0,
    unknown: 0,
  }
  for (const entry of entries.value) {
    const state = schedulerState(entry) as SchedulerStateFilter
    if (state in counts && state !== 'all') counts[state] += 1
    else counts.unknown += 1
  }
  return counts
})
const schedulerStateFilters = computed(() => {
  const states: SchedulerStateFilter[] = [
    'all',
    'active',
    'temp_unschedulable',
    'account_cooldown',
    'model_unavailable',
    'rate_limited',
    'overloaded',
    'quota_exceeded',
    'stopped',
    'error',
    'unknown',
  ]
  return states.map((value) => ({
    value,
    label: value === 'all' ? t('common.all') : schedulerStateLabelByState(value),
    count: stateCounts.value[value] ?? 0,
    dotClass: stateDotClass(value),
  }))
})
const displayedEntries = computed<AccountSchedulingEntry[]>({
  get() {
    if (stateFilter.value === 'all') return entries.value
    return entries.value.filter((entry) => schedulerState(entry) === stateFilter.value)
  },
  set(next) {
    if (stateFilter.value === 'all') {
      entries.value = next
    }
  },
})
const selectedEntries = computed(() =>
  entries.value.filter((entry) => selectedAccountIds.value.has(entry.account_id))
)
const selectedAccountCooldownEntries = computed(() =>
  selectedEntries.value.filter((entry) => accountTempUnschedActive(entry))
)
const selectedMonitorableEntries = computed(() =>
  selectedEntries.value.filter((entry) => canMonitor(entry))
)
const recentGroupFirstTokenStats = computed(() => {
  let weightedMs = 0
  let sampleCount = 0
  for (const entry of entries.value) {
    const avg = entry.recent_user_avg_first_token_ms
    const count = entry.recent_user_first_token_sample_count || 0
    if (typeof avg === 'number' && Number.isFinite(avg) && count > 0) {
      weightedMs += avg * count
      sampleCount += count
    }
  }
  return {
    avgMs: sampleCount > 0 ? weightedMs / sampleCount : null,
    sampleCount,
  }
})
const recentGroupFirstTokenText = computed(() => {
  const stats = recentGroupFirstTokenStats.value
  if (stats.avgMs == null) return t('admin.scheduler.recentFirstTokenGroupNoSamples')
  return t('admin.scheduler.recentFirstTokenGroupText', {
    value: formatLatencyCompact(stats.avgMs),
    count: stats.sampleCount,
  })
})
const recentGroupFirstTokenTitle = computed(() => {
  const stats = recentGroupFirstTokenStats.value
  if (stats.avgMs == null) return t('admin.scheduler.recentFirstTokenNoSamples')
  return t('admin.scheduler.recentFirstTokenGroupTitle', {
    value: formatLatencyCompact(stats.avgMs),
    count: stats.sampleCount,
  })
})
const allVisibleSelected = computed(() =>
  displayedEntries.value.length > 0 &&
  displayedEntries.value.every((entry) => selectedAccountIds.value.has(entry.account_id))
)
const historyEventOptions = computed(() => [
  { label: t('common.all'), value: 'all' },
  { label: t('admin.scheduler.historyType.scheduling_blocked'), value: 'scheduling_blocked' },
  { label: t('admin.scheduler.historyType.scheduling_block_skipped'), value: 'scheduling_block_skipped' },
  { label: t('admin.scheduler.historyType.account_monitor'), value: 'account_monitor' },
  { label: t('admin.scheduler.historyType.account_changed'), value: 'account_changed' },
  { label: t('admin.scheduler.historyType.account_groups_changed'), value: 'account_groups_changed' },
  { label: t('admin.scheduler.historyType.group_changed'), value: 'group_changed' },
  { label: t('admin.scheduler.historyType.full_rebuild'), value: 'full_rebuild' },
  { label: t('admin.scheduler.historyType.other'), value: 'other' },
])

const historyCombinedItems = computed<SchedulerHistoryViewItem[]>(() => {
  const items: SchedulerHistoryViewItem[] = historyItems.value.map((item) => ({
    kind: 'outbox',
    id: `outbox-${item.id}`,
    created_at: item.created_at,
    item,
  }))
  const entry = detailEntry.value
  if (entry && canMonitor(entry) && monitorEnabled(entry.account_id)) {
    const timeline = statusFor(entry.account_id)?.timeline || []
    for (let i = 0; i < timeline.length; i += 1) {
      const point = timeline[i]
      items.push({
        kind: 'monitor',
        id: `monitor-${entry.account_id}-${point.checked_at}-${i}`,
        created_at: point.checked_at,
        status: point.status,
        latency_ms: point.latency_ms,
        message: monitorHistoryMessage(point.status, point.latency_ms),
      })
    }
  }
  return items.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
})

const historyVisibleItems = computed(() => {
  const q = historyQuery.value.trim().toLowerCase()
  return historyCombinedItems.value.filter((item) => {
    const type = item.kind === 'monitor' ? 'account_monitor' : normalizeHistoryType(item.item.event_type)
    if (historyEventFilter.value !== 'all' && type !== historyEventFilter.value) return false
    if (!q) return true
    const blob = historySearchText(item)
    return blob.includes(q)
  })
})

// 持久化当前分组的自动排序配置，并就地更新本地 groups 缓存。
async function saveAutoSortConfig(enabled: boolean) {
  const gid = selectedGroupId.value
  if (!gid) return
  const group = groups.value.find((item) => item.id === gid)
  const previous = group?.auto_sort_config
  if (group) group.auto_sort_config = { enabled, basis: AUTO_SORT_BASIS }
  autoSortSaving.value = true
  try {
    const updated = await adminAPI.groups.update(gid, {
      auto_sort_config: { enabled, basis: AUTO_SORT_BASIS },
    })
    if (group) group.auto_sort_config = updated.auto_sort_config || { enabled, basis: AUTO_SORT_BASIS }
    appStore.showSuccess(
      enabled ? t('admin.scheduler.autoSortOn') : t('admin.scheduler.autoSortOff')
    )
  } catch (err: any) {
    if (group) group.auto_sort_config = previous
    appStore.showError(err?.message || t('admin.scheduler.autoSortSaveFailed'))
  } finally {
    autoSortSaving.value = false
  }
}

function onToggleAutoSort(enabled: boolean) {
  void saveAutoSortConfig(enabled)
}

// ---- 加载分组 ----
// autoSelectFirst=true 时加载完默认选中第一个分组并加载其账号。
function sortGroupsByUserOrder(items: AdminGroup[]): AdminGroup[] {
  return [...items].sort((a, b) => {
    const sortOrderDelta = (Number(a.sort_order) || 0) - (Number(b.sort_order) || 0)
    if (sortOrderDelta !== 0) return sortOrderDelta
    return a.id - b.id
  })
}

async function loadGroups(autoSelectFirst = false) {
  try {
    groups.value = sortGroupsByUserOrder(await adminAPI.groups.getAll())
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.loadGroupsFailed'))
    groups.value = []
  }
  if (selectedGroupId.value && !groups.value.some((g) => g.id === selectedGroupId.value)) {
    selectedGroupId.value = null
    entries.value = []
    upstreamStatusMap.value = new Map()
  }
  if (autoSelectFirst && !selectedGroupId.value && groups.value.length > 0) {
    selectedGroupId.value = groups.value[0].id
    await onGroupChange()
  }
}

async function onGroupChange() {
  if (!selectedGroupId.value) {
    entries.value = []
    return
  }
  router.replace({ query: { ...route.query, group: String(selectedGroupId.value) } })
  await loadEntries()
}

function normalizeWeight(value: number | string | null | undefined): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 1) return 1
  return Math.round(parsed)
}

async function loadEntries() {
  if (!selectedGroupId.value) return
  loading.value = true
  orderDirty.value = false
  entries.value = []
  upstreamStatusMap.value = new Map()
  selectedAccountIds.value = new Set()
  stateFilter.value = 'all'
  try {
    const data = await getAccountScheduling(selectedGroupId.value)
    const list = (data.accounts || []).map((e) => ({
      ...e,
      role: e.role === 'backup' ? 'backup' as const : 'primary' as const,
      weight: normalizeWeight(e.weight),
      sort_order: Number(e.sort_order) || 0,
    }))
    // account_groups 返回顺序就是后端的 canonical 稳定性/模型成功率顺序。
    // 不在前端按倍率、探测状态或账号角色二次重排，避免页面与实际调度顺序分叉。
    entries.value = list
    void refreshUpstreamStatuses(false, undefined, true)
    void loadMonitors()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.loadFailed'))
  } finally {
    loading.value = false
  }
}

// ---- 账号监控（独立的 account-monitor 一套）----
async function loadMonitors() {
  const ids = new Set(entries.value.map((e) => e.account_id))
  if (ids.size === 0) {
    monitorMap.value = new Map()
    monitorStatus.value = new Map()
    return
  }
  try {
    const all = await accountMonitorAPI.list()
    const map = new Map<number, AccountMonitor>()
    for (const m of all) {
      if (ids.has(m.account_id)) map.set(m.account_id, m)
    }
    monitorMap.value = map
    await refreshMonitorStatus()
  } catch {
    monitorMap.value = new Map()
    monitorStatus.value = new Map()
  }
}

async function refreshMonitorStatus() {
  if (monitorMap.value.size === 0) {
    monitorStatus.value = new Map()
    return
  }
  try {
    monitorStatus.value = await accountMonitorAPI.status()
  } catch {
    // 保留旧状态，避免轮询抖动清空。
  }
}

function monitorFor(accountId: number): AccountMonitor | undefined {
  return monitorMap.value.get(accountId)
}
// 监控是否「启用中」：记录存在且 enabled=true。停用的记录仍在库里（保留历史），
// 关掉再打开可用率延续，所以这里看 enabled 而非记录是否存在。
function monitorEnabled(accountId: number): boolean {
  return monitorMap.value.get(accountId)?.enabled === true
}
function statusFor(accountId: number): AccountMonitorStatus | undefined {
  return monitorStatus.value.get(accountId)
}
function monitorModel(accountId: number): string {
  return monitorFor(accountId)?.model || DEFAULT_MONITOR_MODEL
}

function setMonitorBusy(accountId: number, busy: boolean) {
  const next = new Set(monitorBusyIds.value)
  if (busy) next.add(accountId)
  else next.delete(accountId)
  monitorBusyIds.value = next
}

async function toggleMonitor(entry: AccountSchedulingEntry, enabled: boolean) {
  if (!entry.account) return
  const accountId = entry.account_id
  if (monitorBusyIds.value.has(accountId)) return
  const existing = monitorFor(accountId)
  if (existing?.enabled === enabled || (!existing && !enabled)) return
  setMonitorBusy(accountId, true)
  try {
    if (!enabled && existing) {
      // 关闭 → 停用（enabled=false），保留记录与历史，下次打开可用率延续。
      const updated = await accountMonitorAPI.update(existing.id, { enabled: false })
      const map = new Map(monitorMap.value)
      map.set(accountId, updated)
      monitorMap.value = map
      appStore.showSuccess(t('admin.scheduler.monitorStopped'))
    } else if (existing) {
      // 已有停用记录 → 重新启用（保留历史），并立即探测一次。
      const updated = await accountMonitorAPI.update(existing.id, { enabled: true })
      const map = new Map(monitorMap.value)
      map.set(accountId, updated)
      monitorMap.value = map
      appStore.showSuccess(t('admin.scheduler.monitorStarted'))
      accountMonitorAPI.runNow(updated.id).then(() => refreshMonitorStatus()).catch(() => {})
    } else {
      // 开启 → 首次创建账号监控（仅 api_key 类账号；provider 后端会按账号推断）
      const provider = entry.account.platform as AccountPlatform
      if (!MONITORABLE_PLATFORMS.includes(provider)) {
        appStore.showError(t('admin.scheduler.monitorPlatformUnsupported'))
        return
      }
      const model = monitorModel(accountId).trim() || DEFAULT_MONITOR_MODEL
      const created = await accountMonitorAPI.create({
        account_id: accountId,
        provider: provider as 'openai' | 'anthropic' | 'gemini',
        model,
        interval_seconds: DEFAULT_MONITOR_INTERVAL,
        enabled: true,
      })
      const map = new Map(monitorMap.value)
      map.set(accountId, created)
      monitorMap.value = map
      appStore.showSuccess(t('admin.scheduler.monitorStarted'))
      // 立即触发一次探测，尽快出首个状态。
      accountMonitorAPI
        .runNow(created.id)
        .then(() => refreshMonitorStatus())
        .catch(() => {})
    }
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.monitorToggleFailed'))
  } finally {
    setMonitorBusy(accountId, false)
  }
}

async function applyBulkMonitorModel() {
  const model = bulkMonitorModel.value.trim()
  if (!model) return
  const targets = entries.value.filter((entry) => canMonitor(entry))
  if (targets.length === 0) {
    appStore.showError(t('admin.scheduler.monitorPlatformUnsupported'))
    return
  }
  bulkMonitorModelSaving.value = true
  try {
    await Promise.all(targets.map(async (entry) => {
      const existing = monitorFor(entry.account_id)
      if (existing) {
        const updated = await accountMonitorAPI.update(existing.id, { model })
        const map = new Map(monitorMap.value)
        map.set(entry.account_id, updated)
        monitorMap.value = map
        return
      }
      const provider = entry.account?.platform as AccountPlatform | undefined
      if (!provider || !MONITORABLE_PLATFORMS.includes(provider)) return
      const created = await accountMonitorAPI.create({
        account_id: entry.account_id,
        provider: provider as 'openai' | 'anthropic' | 'gemini',
        model,
        interval_seconds: DEFAULT_MONITOR_INTERVAL,
        enabled: false,
      })
      const map = new Map(monitorMap.value)
      map.set(entry.account_id, created)
      monitorMap.value = map
    }))
    showBulkMonitorModelDialog.value = false
    appStore.showSuccess(t('admin.scheduler.monitorModelUpdated'))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.monitorModelUpdateFailed'))
  } finally {
    bulkMonitorModelSaving.value = false
  }
}

// The backend auto-sort worker is the sole owner of ranking. Reloading here
// only displays the persisted canonical order; it never derives a client-side
// order from rates or probe snapshots (which may be stale or incomplete).
async function refreshSchedulerOrder() {
  if (!selectedGroupId.value || orderDirty.value) return
  await loadEntries()
  appStore.showSuccess(t('admin.scheduler.refreshOrderSuccess'))
}

// ---- 保存当前分组的调度顺序与 weight/role ----
async function saveOrder() {
  if (!selectedGroupId.value) return
  saving.value = true
  try {
    // 顺序只落到 account_groups，避免共用账号覆盖其他分组的顺序。
    const payload = {
      accounts: entries.value.map((entry, index) => ({
        account_id: entry.account_id,
        role: entry.role === 'backup' ? 'backup' as const : 'primary' as const,
        weight: normalizeWeight(entry.weight),
        sort_order: (index + 1) * 10,
        scheduling_configured: true,
      })),
    }
    await updateAccountScheduling(selectedGroupId.value, payload)
    orderDirty.value = false
    appStore.showSuccess(t('admin.scheduler.saved'))
    await loadEntries()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.saveFailed'))
  } finally {
    saving.value = false
  }
}

// ---- schedulable 开关 ----
function setSchedulableBusy(accountId: number, busy: boolean) {
  const next = new Set(schedulableBusyIds.value)
  if (busy) next.add(accountId)
  else next.delete(accountId)
  schedulableBusyIds.value = next
}

async function toggleSchedulable(entry: AccountSchedulingEntry, next: boolean) {
  if (!entry.account) return
  const accountId = entry.account_id
  if (schedulableBusyIds.value.has(accountId)) return
  const previous = isSchedulable(entry)
  if (next === previous) return

  entry.account.schedulable = next
  setSchedulableBusy(accountId, true)
  try {
    const updated = await adminAPI.accounts.setSchedulable(accountId, next)
    if (entry.account) entry.account = { ...entry.account, ...updated }
    entry.scheduling_configured = true
  } catch (err: any) {
    if (entry.account) entry.account.schedulable = previous
    appStore.showError(err?.message || t('admin.scheduler.toggleFailed'))
  } finally {
    setSchedulableBusy(accountId, false)
  }
}

// ---- 清除临时不可调度 ----
async function clearTempUnsched(entry: AccountSchedulingEntry) {
  try {
    await adminAPI.accounts.resetTempUnschedulable(entry.account_id)
    if (entry.account) {
      entry.account.temp_unschedulable_until = null
      entry.account.temp_unschedulable_reason = null
    }
    appStore.showSuccess(t('admin.scheduler.tempUnschedCleared'))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.clearTempUnschedFailed'))
  }
}

// ---- 上游状态 ----
async function refreshUpstreamStatuses(force = false, accountIds?: number[], silent = false): Promise<boolean> {
  const ids = accountIds ?? entries.value.map((e) => e.account_id)
  const seq = ++upstreamReqSeq
  const interactiveSeq = silent ? 0 : ++upstreamInteractiveReqSeq
  if (ids.length === 0) {
    upstreamStatusMap.value = new Map()
    upstreamLoading.value = false
    return true
  }
  if (!silent) upstreamLoading.value = true
  try {
    const statuses = force
      ? await adminAPI.upstreams.refreshManagedAccountStatuses(ids)
      : await adminAPI.upstreams.getManagedAccountStatuses(ids)
    if (seq !== upstreamReqSeq) return false
    const next = new Map(upstreamStatusMap.value)
    for (const id of ids) next.delete(id)
    for (const status of statuses) next.set(status.account_id, status)
    upstreamStatusMap.value = next
    return true
  } catch (error) {
    if (seq !== upstreamReqSeq) return false
    if (force) throw error
    return false
  } finally {
    if (!silent && interactiveSeq === upstreamInteractiveReqSeq) upstreamLoading.value = false
  }
}

async function loadSchedulerHistory() {
  const gid = selectedGroupId.value
  if (!gid) {
    historyItems.value = []
    return
  }
  historyLoading.value = true
  try {
    historyItems.value = await getAccountSchedulingHistory(gid, 50)
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.historyLoadFailed'))
    historyItems.value = []
  } finally {
    historyLoading.value = false
  }
}

// ---- 展示 helper ----
function accountName(entry: AccountSchedulingEntry): string {
  return entry.account?.name || t('admin.accounts.unnamed')
}
function schedulerState(entryOrState: SchedulerStateInput): string {
  if (typeof entryOrState === 'string') return entryOrState
  if (!entryOrState) return 'unknown'
  // Probe health is shown by the leading status dot. A single failed or
  // degraded probe is not itself a scheduler block; only persisted runtime
  // state determines the scheduler status and its filters.
  return baseSchedulerState(entryOrState)
}
function baseSchedulerState(entryOrState: SchedulerStateInput): string {
  if (typeof entryOrState === 'string') return entryOrState
  if (!entryOrState) return 'unknown'
  // The API state can lag behind runtime cooldown fields. Prefer runtime facts
  // whenever they show that an otherwise active account is currently excluded.
  const inferred = inferSchedulerState(entryOrState)
  if (inferred !== 'active') return inferred
  return entryOrState.block_reason || entryOrState.state || inferred
}
function schedulerStateLabel(entryOrState: SchedulerStateInput): string {
  const state = typeof entryOrState === 'string' ? entryOrState : schedulerState(entryOrState)
  return schedulerStateLabelByState(state)
}
function schedulerStateLabelByState(state: string): string {
  const map: Record<string, string> = {
    active: t('admin.scheduler.stateActive'),
    stopped: t('admin.scheduler.stateStopped'),
    manual_unschedulable: t('admin.scheduler.stateStopped'),
    inactive: t('admin.scheduler.stateError'),
    expired: t('admin.scheduler.stateExpired'),
    error: t('admin.scheduler.stateError'),
    rate_limited: t('admin.scheduler.stateRateLimited'),
    overloaded: t('admin.scheduler.stateOverloaded'),
    temp_unschedulable: t('admin.scheduler.stateTempUnschedulable'),
    account_cooldown: t('admin.scheduler.stateAccountCooldown'),
    model_unavailable: t('admin.scheduler.stateModelUnavailable'),
    quota_exceeded: t('admin.scheduler.stateQuotaExceeded'),
    unknown: t('admin.scheduler.stateUnknown'),
  }
  return map[state] || t('admin.scheduler.stateUnknown')
}
function stateDotClass(state: string): string {
  switch (state) {
    case 'active':
      return 'bg-emerald-500'
    case 'stopped':
      return 'bg-gray-400'
    case 'error':
      return 'bg-red-500'
    case 'rate_limited':
      return 'bg-amber-400'
    case 'overloaded':
      return 'bg-rose-500'
    case 'temp_unschedulable':
    case 'account_cooldown':
      return 'bg-orange-500'
    case 'model_unavailable':
      return 'bg-amber-400'
    case 'quota_exceeded':
      return 'bg-purple-500'
    default:
      return 'bg-gray-300 dark:bg-dark-500'
  }
}
function inferSchedulerState(entry: AccountSchedulingEntry): string {
  const a = entry.account
  if (!a) return 'unknown'
  const now = new Date()
  if (a.status !== 'active') return 'error'
  if (!a.schedulable) return 'stopped'
  if (a.temp_unschedulable_until && new Date(a.temp_unschedulable_until) > now) return 'account_cooldown'
  if (a.rate_limit_reset_at && new Date(a.rate_limit_reset_at) > now) return 'rate_limited'
  if (a.overload_until && new Date(a.overload_until) > now) return 'overloaded'
  const quotaExceeded = (used?: number | null, limit?: number | null) => typeof limit === 'number' && limit > 0 && typeof used === 'number' && used >= limit
  if (
    quotaExceeded(a.quota_used, a.quota_limit) ||
    quotaExceeded(a.quota_daily_used, a.quota_daily_limit) ||
    quotaExceeded(a.quota_weekly_used, a.quota_weekly_limit)
  ) {
    return 'quota_exceeded'
  }
  if (activeModelCooldowns(entry).length > 0) return 'model_unavailable'
  return 'active'
}
function isSchedulable(entry: AccountSchedulingEntry): boolean {
  return entry.account?.schedulable !== false
}
function tempUnschedActive(entry: AccountSchedulingEntry): boolean {
  return ['temp_unschedulable', 'account_cooldown'].includes(baseSchedulerState(entry))
}
function accountTempUnschedActive(entry: AccountSchedulingEntry): boolean {
  const until = entry.account?.temp_unschedulable_until
  if (!until) return false
  const end = new Date(until).getTime()
  return Number.isFinite(end) && end > Date.now()
}
function accountCooldownTooltip(entry: AccountSchedulingEntry): string {
  const until = entry.account?.temp_unschedulable_until
  const reason = entry.account?.temp_unschedulable_reason?.trim()
  const parts = [t('admin.scheduler.accountCooldownTitle')]
  if (until) parts.push(`${t('admin.scheduler.historyUntil')}: ${new Date(until).toLocaleString()}`)
  if (reason) parts.push(reason)
  return parts.join('\n')
}
function isInsufficientBalanceReason(reason: unknown): boolean {
  if (typeof reason !== 'string') return false
  const normalized = reason.toLowerCase()
  return (
    normalized.includes('account_monitor_insufficient_balance') ||
    normalized.includes('insufficient balance') ||
    normalized.includes('insufficient_balance') ||
    normalized.includes('余额不足') ||
    normalized.includes('额度不足')
  )
}
function insufficientBalanceReason(entry: AccountSchedulingEntry): string {
  const accountReason = entry.account?.temp_unschedulable_reason
  if (isInsufficientBalanceReason(accountReason)) return accountReason || ''
  return ''
}
function hasInsufficientBalance(entry: AccountSchedulingEntry): boolean {
  return insufficientBalanceReason(entry) !== ''
}
function insufficientBalanceTitle(entry: AccountSchedulingEntry): string {
  const until = entry.account?.temp_unschedulable_until
  if (!until) return t('admin.scheduler.insufficientBalanceTitle')
  return `${t('admin.scheduler.insufficientBalanceTitle')} · ${t('admin.scheduler.historyUntil')}: ${new Date(until).toLocaleString()}`
}
function groupReserveUntil(entry: AccountSchedulingEntry | null | undefined): Date | null {
  const raw = entry?.group_reserve_until
  if (typeof raw !== 'string' || !raw.trim()) return null
  const until = new Date(raw)
  if (!Number.isFinite(until.getTime()) || until.getTime() <= Date.now()) return null
  return until
}
function groupReserveActive(entry: AccountSchedulingEntry | null | undefined): boolean {
  return entry?.group_reserve === true && groupReserveUntil(entry) !== null
}
function groupReserveSummary(entry: AccountSchedulingEntry | null | undefined): string {
  return groupReserveActive(entry)
    ? t('admin.scheduler.groupReserveActive')
    : t('admin.scheduler.groupReserveNone')
}
function groupReserveDetail(entry: AccountSchedulingEntry | null | undefined): string {
  const until = groupReserveUntil(entry)
  if (!until) return t('admin.scheduler.groupReserveNoneDetail')
  const parts = [
    `${t('admin.scheduler.historyUntil')}: ${until.toLocaleString()}`,
    `${t('admin.scheduler.remaining')}: ${modelCooldownRemainingText(until)}`,
  ]
  const reason = entry?.group_reserve_reason?.trim()
  if (reason) parts.push(reason)
  return parts.join(' · ')
}
function groupReserveTooltip(entry: AccountSchedulingEntry): string {
  const parts = [t('admin.scheduler.groupReserveTitle'), groupReserveDetail(entry)]
  return parts.filter(Boolean).join('\n')
}
function activeModelCooldowns(entry: AccountSchedulingEntry | null | undefined): ActiveModelCooldown[] {
  const raw = entry?.account?.extra?.model_rate_limits
  if (!raw || typeof raw !== 'object') return []
  const now = Date.now()
  const items: ActiveModelCooldown[] = []
  for (const [model, value] of Object.entries(raw)) {
    if (!value || typeof value !== 'object') continue
    const resetRaw = (value as { rate_limit_reset_at?: unknown }).rate_limit_reset_at
    if (typeof resetRaw !== 'string' || !resetRaw.trim()) continue
    const resetAt = new Date(resetRaw)
    if (!Number.isFinite(resetAt.getTime()) || resetAt.getTime() <= now) continue
    const reasonRaw = (value as { reason?: unknown }).reason
    items.push({
      model,
      resetAt,
      reason: typeof reasonRaw === 'string' ? reasonRaw : undefined,
    })
  }
  return items.sort((a, b) => a.resetAt.getTime() - b.resetAt.getTime() || a.model.localeCompare(b.model))
}
function modelCooldownRemainingText(resetAt: Date): string {
  const diff = Math.max(0, resetAt.getTime() - Date.now())
  const mins = Math.ceil(diff / 60000)
  if (mins >= 60) {
    const hrs = Math.floor(mins / 60)
    const rest = mins % 60
    return rest > 0 ? `${hrs}h ${rest}m` : `${hrs}h`
  }
  return `${mins}m`
}
function modelCooldownBadge(entry: AccountSchedulingEntry): string {
  const items = activeModelCooldowns(entry)
  if (items.length === 0) return ''
  if (items.length === 1) return t('admin.scheduler.modelCooldownOne', { model: items[0].model })
  return t('admin.scheduler.modelCooldownMany', { count: items.length })
}
function modelCooldownSummary(entry: AccountSchedulingEntry | null | undefined): string {
  const items = activeModelCooldowns(entry)
  if (items.length === 0) return t('admin.scheduler.modelCooldownNone')
  return t('admin.scheduler.modelCooldownSummary', { count: items.length })
}
function modelCooldownDetail(entry: AccountSchedulingEntry | null | undefined): string {
  const items = activeModelCooldowns(entry)
  if (items.length === 0) return t('admin.scheduler.modelCooldownNoActive')
  return items
    .slice(0, 4)
    .map((item) => `${item.model} ${modelCooldownRemainingText(item.resetAt)}`)
    .join(' · ')
}
function modelCooldownTooltip(entry: AccountSchedulingEntry): string {
  const items = activeModelCooldowns(entry)
  if (items.length === 0) return ''
  return items
    .map((item) => {
      const parts = [
        item.model,
        `${t('admin.scheduler.historyUntil')}: ${item.resetAt.toLocaleString()}`,
        `${t('admin.scheduler.remaining')}: ${modelCooldownRemainingText(item.resetAt)}`,
      ]
      if (item.reason) parts.push(item.reason)
      return parts.join(' · ')
    })
    .join('\n')
}
function canMonitor(entry: AccountSchedulingEntry): boolean {
  const platform = entry.account?.platform as AccountPlatform | undefined
  return (
    entry.account?.type === 'apikey' &&
    !!platform &&
    MONITORABLE_PLATFORMS.includes(platform)
  )
}
function statusLabel(entry: AccountSchedulingEntry): string {
  const a = entry.account
  if (!a) return t('admin.scheduler.statusUnknown')
  if (!a.schedulable) return t('admin.scheduler.statusStopped')
  if (a.status === 'active') return t('admin.scheduler.statusActive')
  if (a.status === 'inactive') return t('admin.scheduler.statusInactive')
  if (a.status === 'error') return t('admin.scheduler.statusError')
  return a.status
}
function statusDotClass(entryOrState: AccountSchedulingEntry | string): string {
  switch (baseSchedulerState(entryOrState)) {
    case 'active':
      return 'bg-emerald-500'
    case 'stopped':
      return 'bg-gray-400'
    case 'error':
      return 'bg-red-500'
    case 'rate_limited':
      return 'bg-amber-400'
    case 'overloaded':
      return 'bg-rose-500'
    case 'temp_unschedulable':
    case 'account_cooldown':
      return 'bg-orange-500'
    case 'model_unavailable':
      return 'bg-amber-400'
    case 'quota_exceeded':
      return 'bg-purple-500'
    default:
      return 'bg-gray-300 dark:bg-dark-500'
  }
}
// 行首小点：监控开启时反映监控探测状态（绿=正常/黄=缓慢/红=异常/灰=无数据），
// 未开监控时回退到账号自身状态。
function headDotClass(entry: AccountSchedulingEntry): string {
  if (canMonitor(entry) && monitorEnabled(entry.account_id)) {
    const st = statusFor(entry.account_id)
    switch (st?.latest_status) {
      case 'operational':
        return 'bg-emerald-500'
      case 'degraded':
        return 'bg-amber-400'
      case 'failed':
      case 'error':
        return 'bg-red-500'
      default:
        return 'bg-gray-300 dark:bg-dark-500'
    }
  }
  return statusDotClass(entry)
}
function entryTooltip(entry: AccountSchedulingEntry): string {
  const a = entry.account
  const lines = [
    `${accountName(entry)}`,
    `${t('admin.scheduler.stateLabel')}: ${schedulerStateLabel(entry)}`,
    `${t('admin.scheduler.currentConcurrency')}: ${currentConcurrencyText(entry)}`,
    `${t('admin.scheduler.statusActive')}: ${statusLabel(entry)}`,
    `${t('admin.scheduler.priorityLabel')}: ${a?.priority ?? '-'}`,
  ]
  const up = upstreamStatus(entry)
  if (up) {
    const balance = balanceInfo(entry)
    if (balance) {
      lines.push(`${t('admin.scheduler.walletBalance')}: ${balance.text}`)
    }
    if (!up.stale && typeof up.key_remaining === 'number') {
      lines.push(`${t('admin.scheduler.keyRemaining')}: ${formatMoney(up.key_remaining, up.balance_unit)}`)
    }
    if (!up.stale && up.usage_mode === 'unlimited') {
      lines.push(t('admin.scheduler.keyQuotaUnlimited'))
    }
  }
  if (!isSchedulable(entry)) lines.push(t('admin.scheduler.notSchedulable'))
  if (hasInsufficientBalance(entry)) lines.push(t('admin.scheduler.insufficientBalanceTitle'))
  if (tempUnschedActive(entry)) lines.push(t('admin.scheduler.tempUnsched'))
  if (groupReserveActive(entry)) lines.push(`${t('admin.scheduler.groupReserveTitle')}: ${groupReserveDetail(entry)}`)
  const cooldowns = activeModelCooldowns(entry)
  if (cooldowns.length > 0) lines.push(`${t('admin.scheduler.modelCooldownTitle')}: ${modelCooldownDetail(entry)}`)
  return lines.join('\n')
}
function monitorAvailabilityText(accountId: number): string {
  const st = statusFor(accountId)
  if (st?.availability_1h != null && Number.isFinite(st.availability_1h)) {
    return `${st.availability_1h.toFixed(1)}%`
  }
  return '—'
}
// 彩虹状态条：把 timeline（newest-first）转成颜色高度数组（最旧在左，最新在右）。
// 与用户视图渠道监控的彩虹条一致：高=好+绿，中=黄(降级)，短=红(失败/错误)，很短灰=未测试。
const MONITOR_BAR_LEN = 40
function miniBars(accountId: number): Array<{ cls: string; h: number; title: string }> {
  const tl = statusFor(accountId)?.timeline ?? []
  const pts = [...tl].slice(0, MONITOR_BAR_LEN).reverse()
  const bars: Array<{ cls: string; h: number; title: string }> = []
  const pad = Math.max(0, MONITOR_BAR_LEN - pts.length)
  for (let i = 0; i < pad; i++) {
    bars.push({ cls: 'bg-gray-300/40 dark:bg-dark-500/40', h: 15, title: '' })
  }
  for (const p of pts) {
    let cls = 'bg-gray-300/40 dark:bg-dark-500/40'
    let h = 15
    if (p.status === 'operational') { cls = 'bg-emerald-500'; h = 100 }
    else if (p.status === 'degraded') { cls = 'bg-amber-400'; h = 65 }
    else if (p.status === 'failed' || p.status === 'error') { cls = 'bg-red-500'; h = 35 }
    const when = new Date(p.checked_at).toLocaleString()
    const latency = p.latency_ms != null ? `${p.latency_ms}ms` : '-'
    bars.push({ cls, h, title: `${when} · ${p.status} · ${latency}` })
  }
  return bars
}
function hasMonitorTimeline(accountId: number): boolean {
  return (statusFor(accountId)?.timeline?.length ?? 0) > 0
}
// 监控状态徽章样式：红/绿/黄底色块（第二行用）。
function currentConcurrencyText(entry: AccountSchedulingEntry): string {
  const current = entry.account?.current_concurrency
  if (typeof current === 'number' && Number.isFinite(current) && current >= 0) {
    return String(current)
  }
  return '0'
}
function capacityText(entry: AccountSchedulingEntry): string {
  const capacity = entry.account?.concurrency
  if (typeof capacity === 'number' && Number.isFinite(capacity) && capacity > 0) {
    return String(Math.trunc(capacity))
  }
  return '∞'
}
function concurrencyUsageText(entry: AccountSchedulingEntry): string {
  return `${currentConcurrencyText(entry)}/${capacityText(entry)}`
}
function formatLatencyCompact(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '-'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${parseFloat((ms / 1000).toFixed(1))}s`
}
function recentFirstTokenValueText(entry: AccountSchedulingEntry): string {
  const avg = entry.recent_user_avg_first_token_ms
  if (typeof avg !== 'number' || !Number.isFinite(avg)) return '-'
  return formatLatencyCompact(avg)
}
function recentFirstTokenText(entry: AccountSchedulingEntry): string {
  const count = entry.recent_user_first_token_sample_count || 0
  const value = recentFirstTokenValueText(entry)
  if (value === '-') return `${t('admin.scheduler.recentFirstToken5m')} -`
  return `${t('admin.scheduler.recentFirstToken5m')} ${value} · ${count}`
}
function recentFirstTokenTitle(entry: AccountSchedulingEntry): string {
  const count = entry.recent_user_first_token_sample_count || 0
  if (count <= 0) return t('admin.scheduler.recentFirstTokenNoSamples')
  return t('admin.scheduler.recentFirstToken5mTitle', { count })
}
function upstreamStatus(entry: AccountSchedulingEntry): UpstreamSub2APIAccountStatus | undefined {
  return upstreamStatusMap.value.get(entry.account_id)
}
function formatMoney(value: number | null | undefined, unit?: string): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  const formatted = Math.abs(value) >= 100 ? value.toFixed(1) : value.toFixed(2)
  if (unit && unit.toUpperCase() !== 'USD') return `${formatted} ${unit}`
  return `$${formatted}`
}
type RateSource = 'manual' | 'upstream' | 'account'

interface FinalRateInfo {
  value: number
  source: RateSource
}

// finalRateInfo 计算账号「最终换算后倍率」，优先级：
//   1) 手动倍率 manual_rate（账号 extra 手填，黄色）
//   2) 本次上游探测倍率 × rate_scale（普通色）
//   3) 明确配置的非 1 账号计费倍率（黄色）
//   4) 默认 1 不展示，避免把“检测不到”误显示为 1
function finalRateInfo(entry: AccountSchedulingEntry): FinalRateInfo | null {
  const ex = (entry.account?.extra as Record<string, unknown> | undefined) || {}
  const manual = ex.manual_rate
  if (typeof manual === 'number' && Number.isFinite(manual)) {
    return { value: manual, source: 'manual' }
  }
  const scale = typeof ex.rate_scale === 'number' && Number.isFinite(ex.rate_scale) ? ex.rate_scale : 1
  const s = upstreamStatus(entry)
  const upstream = s?.upstream_group_effective_rate_multiplier ?? s?.upstream_group_default_rate_multiplier
  if (!s?.stale && typeof upstream === 'number' && Number.isFinite(upstream)) {
    return { value: upstream * scale, source: 'upstream' }
  }
  const billingRate = entry.account?.rate_multiplier
  if (typeof billingRate === 'number' && Number.isFinite(billingRate) && billingRate !== 1) {
    return { value: billingRate, source: 'account' }
  }
  return null
}
// 倍率显示：最终换算后倍率。取不到返回 null（不显示）。
function rateMultiplierText(entry: AccountSchedulingEntry): string | null {
  const r = finalRateInfo(entry)?.value
  if (r == null) return null
  // 保留至多 4 位小数，去掉多余尾零。
  return `×${parseFloat(r.toFixed(4))}`
}
function rateMultiplierClass(entry: AccountSchedulingEntry): string {
  const source = finalRateInfo(entry)?.source
  if (source === 'manual' || source === 'account') {
    return 'text-amber-600 dark:text-amber-300'
  }
  return 'text-cyan-600 dark:text-cyan-300'
}
function rateMultiplierTitle(entry: AccountSchedulingEntry): string {
  const source = finalRateInfo(entry)?.source
  if (source === 'manual') return t('admin.scheduler.manualRate')
  if (source === 'account') return t('admin.scheduler.billingRateMultiplier')
  return t('admin.scheduler.rateMultiplier')
}
interface UpstreamBalanceInfo {
  text: string
}

// A failed probe means the current wallet is unavailable; historical snapshots
// must not be presented as current values.
function balanceInfo(entry: AccountSchedulingEntry): UpstreamBalanceInfo | null {
  const s = upstreamStatus(entry)
  if (!s?.stale && typeof s?.user_balance === 'number' && Number.isFinite(s.user_balance)) {
    return { text: formatMoney(s.user_balance, s.balance_unit) }
  }
  return null
}
function balanceText(entry: AccountSchedulingEntry): string | null {
  return balanceInfo(entry)?.text ?? null
}
function balanceClass(entry: AccountSchedulingEntry): string {
	return balanceInfo(entry) ? 'text-gray-500 dark:text-gray-400' : ''
}
function balanceTitle(entry: AccountSchedulingEntry): string {
	return balanceInfo(entry) ? t('admin.scheduler.balanceWalletKey') : ''
}
function activeDurationText(entry: AccountSchedulingEntry): string {
  const a = entry.account
  const until = a?.temp_unschedulable_until
  if (!until) return '—'
  const end = new Date(until).getTime()
  if (!Number.isFinite(end)) return '—'
  const diff = Math.max(0, end - Date.now())
  const mins = Math.floor(diff / 60000)
  const hrs = Math.floor(mins / 60)
  if (hrs > 0) return `${hrs}h ${mins % 60}m`
  return `${mins}m`
}
function recentCheckedText(entry: AccountSchedulingEntry): string {
  const st = statusFor(entry.account_id)
  if (st?.last_checked_at) return new Date(st.last_checked_at).toLocaleString()
  return entry.account?.updated_at ? new Date(entry.account.updated_at).toLocaleString() : '—'
}
function recentFailedText(entry: AccountSchedulingEntry): string {
  const st = statusFor(entry.account_id)
  const latest = st?.timeline?.find((p) => p.status === 'failed' || p.status === 'error')
  return latest?.checked_at ? new Date(latest.checked_at).toLocaleString() : '—'
}
function monitorAvgLatencyText(entry: AccountSchedulingEntry): string {
  const st = statusFor(entry.account_id)
  if (st?.avg_latency_1h != null && Number.isFinite(st.avg_latency_1h)) {
    return `${st.avg_latency_1h.toFixed(1)}ms`
  }
  return '—'
}
function monitorLatestLatencyText(entry: AccountSchedulingEntry): string {
  const st = statusFor(entry.account_id)
  if (st?.latest_latency_ms != null) return `${st.latest_latency_ms}ms`
  return '—'
}
function monitorPingLatencyText(entry: AccountSchedulingEntry): string {
  const st = statusFor(entry.account_id)
  if (st?.ping_latency_ms != null) return `${st.ping_latency_ms}ms`
  return '—'
}

function normalizeHistoryType(value: string): string {
  const known = new Set(['account_changed', 'account_groups_changed', 'account_bulk_changed', 'account_last_used', 'scheduling_blocked', 'scheduling_block_skipped', 'group_changed', 'full_rebuild'])
  return known.has(value) ? value : 'other'
}

function historyTypeLabel(eventType: string): string {
  const type = normalizeHistoryType(eventType)
  return t(`admin.scheduler.historyType.${type}`)
}

function historyItemTypeLabel(item: SchedulerHistoryViewItem): string {
  if (item.kind === 'monitor') return t('admin.scheduler.historyType.account_monitor')
  return historyTypeLabel(item.item.event_type)
}

function formatHistoryTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}

function historyFailureCategoryLabel(category: string): string {
  const map: Record<string, string> = {
    account_monitor_consecutive_failures: t('admin.scheduler.historyFailureTransient'),
    account_monitor_insufficient_balance: t('admin.scheduler.historyFailureInsufficientBalance'),
    account_monitor_auth_failed: t('admin.scheduler.historyFailureAuthFailed'),
    account_monitor_upstream_group_unavailable: t('admin.scheduler.historyFailureUpstreamGroupUnavailable'),
    account_monitor_model_unsupported: t('admin.scheduler.historyFailureModelUnsupported'),
    sticky_escape_ttft: t('admin.scheduler.historyFailureStickyTTFT'),
    sticky_escape_error_rate: t('admin.scheduler.historyFailureStickyErrorRate'),
    sticky_escape_concurrency_full: t('admin.scheduler.historyFailureStickyConcurrencyFull'),
  }
  return map[category] || category
}

function historyCooldownText(minutes: number): string {
  if (!Number.isFinite(minutes) || minutes <= 0) return ''
  if (minutes >= 60 && minutes % 60 === 0) {
    return t('admin.scheduler.historyCooldownHours', { hours: minutes / 60 })
  }
  if (minutes >= 60) {
    const hours = Math.floor(minutes / 60)
    const rest = minutes % 60
    return t('admin.scheduler.historyCooldownHoursMinutes', { hours, minutes: rest })
  }
  return t('admin.scheduler.historyCooldownMinutes', { minutes })
}

function historyRepeatCount(payload: Record<string, unknown> | null | undefined): number {
  const raw = payload?.history_count
  const count = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : 0
  return Number.isFinite(count) && count > 1 ? Math.trunc(count) : 0
}

function historyRepeatRange(payload: Record<string, unknown> | null | undefined): string {
  const count = historyRepeatCount(payload)
  if (count <= 1) return ''
  const first = typeof payload?.history_first_at === 'string' ? payload.history_first_at : ''
  const last = typeof payload?.history_last_at === 'string' ? payload.history_last_at : ''
  const parts = [t('admin.scheduler.historyRepeated', { count })]
  if (first) parts.push(`${t('admin.scheduler.historyFirstAt')}: ${formatHistoryTime(first)}`)
  if (last) parts.push(`${t('admin.scheduler.historyLastAt')}: ${formatHistoryTime(last)}`)
  return parts.join(' · ')
}

function historySummary(item: GroupSchedulerHistoryItem): string {
  if (item.event_type === 'scheduling_blocked') {
    const payload = item.payload || {}
    const source = typeof payload.source === 'string' ? payload.source : ''
    const reason = typeof payload.reason === 'string' ? payload.reason : ''
    const granularity = typeof payload.block_granularity === 'string' ? payload.block_granularity : ''
    const sourceText = granularity === 'model'
      ? t('admin.scheduler.historyModelCooldown')
      : granularity === 'runtime'
      ? t('admin.scheduler.historyRuntimeCooldown')
      : source === 'account_monitor'
      ? t('admin.scheduler.historyBlockedByMonitor')
      : t('admin.scheduler.historyBlockedByPolicy')
    const parts = [sourceText]
    if (item.account_id != null) parts.push(`${t('admin.scheduler.historyAccount')}: #${item.account_id}`)
    if (item.group_id != null) parts.push(`${t('admin.scheduler.historyGroup')}: #${item.group_id}`)
    const modelRateLimit = typeof payload.model_rate_limit === 'string' ? payload.model_rate_limit : ''
    if (modelRateLimit) parts.push(`${t('admin.scheduler.model')}: ${modelRateLimit}`)
    const failureCategory = typeof payload.failure_category === 'string' ? payload.failure_category : ''
    if (failureCategory) parts.push(historyFailureCategoryLabel(failureCategory))
    const threshold = Number(payload.failure_threshold)
    if (Number.isFinite(threshold) && threshold > 0) parts.push(t('admin.scheduler.historyFailureThreshold', { count: threshold }))
    const cooldown = Number(payload.cooldown_minutes)
    const cooldownText = historyCooldownText(cooldown)
    if (cooldownText) parts.push(cooldownText)
    const ttftMs = Number(payload.ttft_ms)
    if (Number.isFinite(ttftMs) && ttftMs > 0) parts.push(`TTFT ${Math.round(ttftMs)}ms`)
    const errorRate = Number(payload.error_rate)
    if (Number.isFinite(errorRate) && errorRate > 0) parts.push(`${t('admin.scheduler.historyErrorRate')}: ${(errorRate * 100).toFixed(1)}%`)
    const until = typeof payload.until === 'string' ? payload.until : ''
    if (until) parts.push(`${t('admin.scheduler.historyUntil')}: ${formatHistoryTime(until)}`)
    if (reason && source !== 'account_monitor') parts.push(reason)
    const latestMessage = typeof payload.latest_message === 'string' ? payload.latest_message : ''
    if (latestMessage) parts.push(latestMessage)
    const repeatRange = historyRepeatRange(payload)
    if (repeatRange) parts.push(repeatRange)
    return parts.join(' · ')
  }
  if (item.event_type === 'scheduling_block_skipped') {
    const payload = item.payload || {}
    const reason = typeof payload.reason === 'string' ? payload.reason : ''
    const reasonText = reason === 'single_candidate_retry'
      ? t('admin.scheduler.historyBlockSkippedSingleCandidate')
      : t('admin.scheduler.historyBlockSkippedLastAccount')
    const parts = [reasonText]
    if (item.account_id != null) parts.push(`${t('admin.scheduler.historyAccount')}: #${item.account_id}`)
    if (item.group_id != null) parts.push(`${t('admin.scheduler.historyGroup')}: #${item.group_id}`)
    const statusCode = Number(payload.status_code)
    if (Number.isFinite(statusCode) && statusCode > 0) parts.push(`HTTP ${statusCode}`)
    const source = typeof payload.source === 'string' ? payload.source : ''
    if (source) parts.push(source)
    const repeatRange = historyRepeatRange(payload)
    if (repeatRange) parts.push(repeatRange)
    return parts.join(' · ')
  }
  const parts: string[] = []
  if (item.account_id != null) parts.push(`${t('admin.scheduler.historyAccount')}: #${item.account_id}`)
  if (item.group_id != null) parts.push(`${t('admin.scheduler.historyGroup')}: #${item.group_id}`)
  const payload = item.payload || {}
  if (payload && Object.keys(payload).length > 0) {
    const keys = Object.keys(payload).slice(0, 3)
    const summary = keys
      .map((key) => {
        const raw = payload[key]
        const text = typeof raw === 'string' ? raw : Array.isArray(raw) ? `[${raw.length}]` : raw && typeof raw === 'object' ? '{...}' : String(raw)
        return `${key}=${text}`
      })
      .join(' · ')
    if (summary) parts.push(summary)
  }
  if (parts.length === 0) return item.event_type
  return parts.join(' · ')
}
function compressAccountList(ids: number[]): string {
  if (ids.length === 0) return ''
  const shown = ids.slice(0, 4).map((id) => `#${id}`)
  if (ids.length <= 4) return shown.join(' ')
  return `${shown.join(' ')} +${ids.length - 4}`
}

function historyItemSummary(item: SchedulerHistoryViewItem): string {
  if (item.kind === 'outbox') return historySummary(item.item)
  const latency = item.latency_ms != null ? `${item.latency_ms}ms` : '-'
  const status = monitorStatusLabel(item.status)
  return `${status} · ${latency}${item.message ? ` · ${item.message}` : ''}`
}

function historyItemMeta(item: SchedulerHistoryViewItem): string {
  if (item.kind === 'outbox') return `#${item.item.id}`
  return t('admin.scheduler.historyMonitorMeta')
}

function historySearchText(item: SchedulerHistoryViewItem): string {
  if (item.kind === 'outbox') {
    return [
      item.item.event_type,
      item.item.account_id ?? '',
      item.item.group_id ?? '',
      JSON.stringify(item.item.payload || {}),
    ].join(' ').toLowerCase()
  }
  return [item.status, item.latency_ms ?? '', item.message || '', item.created_at].join(' ').toLowerCase()
}

function extractNumericList(value: unknown): number[] | null {
  if (!Array.isArray(value)) return null
  const ids = value.map((v) => Number(v)).filter((v) => Number.isFinite(v) && v > 0)
  return ids.length > 0 ? ids : null
}

function historyItemCountLabel(item: SchedulerHistoryViewItem): string {
  if (item.kind === 'monitor') return t('admin.scheduler.historyMonitorMeta')
  const it = item.item
  const repeated = historyRepeatCount(it.payload)
  if (repeated > 1) return t('admin.scheduler.historyRepeatedShort', { count: repeated })
  if (it.event_type !== 'account_changed') return historyItemMeta(item)
  const ids = extractNumericList(it.payload?.account_ids)
  if (!ids || ids.length <= 1) return historyItemMeta(item)
  return `${t('admin.scheduler.historyBatch')}: ${ids.length}`
}

function historyItemCompactLabel(item: SchedulerHistoryViewItem): string {
  if (item.kind === 'monitor') return historyItemSummary(item)
  const it = item.item
  if (it.event_type !== 'account_changed') return historyItemSummary(item)
  const ids = extractNumericList(it.payload?.account_ids)
  if (!ids || ids.length <= 1) return historyItemSummary(item)
  const prefix = compressAccountList(ids)
  return `${historyItemTypeLabel(item)} · ${prefix}`
}

function monitorStatusLabel(status: string): string {
  const map: Record<string, string> = {
    operational: t('admin.scheduler.monitorStatusOperational'),
    degraded: t('admin.scheduler.monitorStatusDegraded'),
    failed: t('admin.scheduler.monitorStatusFailed'),
    error: t('admin.scheduler.monitorStatusError'),
  }
  return map[status] || status || t('admin.scheduler.stateUnknown')
}

function monitorHistoryMessage(status: string, latency: number | null): string {
  if (status === 'degraded' && latency != null) return t('admin.scheduler.monitorSlowResponse', { ms: latency })
  if (status === 'failed' || status === 'error') return t('admin.scheduler.monitorFailedProbe')
  return ''
}

const schedulerConfigTitle = computed(() => {
  const entry = schedulerConfigEntry.value
  return entry ? `${t('admin.scheduler.accountConfig')}: ${accountName(entry)}` : t('admin.scheduler.accountConfig')
})

const schedulerConfigCanMonitor = computed(() => {
  const entry = schedulerConfigEntry.value
  return !!entry && canMonitor(entry)
})

function numberOrNull(value: unknown): number | null {
  if (value === '' || value === null || value === undefined) return null
  const n = Number(value)
  return Number.isFinite(n) ? n : null
}

function nonNegativeNumberOrNull(value: unknown): number | null {
  const n = numberOrNull(value)
  if (n == null) return null
  return n >= 0 ? n : null
}

function nonNegativeInt(value: unknown, fallback = 0): number {
  const n = nonNegativeNumberOrNull(value)
  if (n == null) return fallback
  return Math.trunc(n)
}

function openSchedulerConfig(entry: AccountSchedulingEntry) {
  if (!entry.account) return
  const extra = (entry.account.extra as Record<string, unknown> | undefined) || {}
  schedulerConfigEntry.value = entry
  schedulerConfigForm.value = {
    concurrency: nonNegativeInt(entry.account.concurrency, 0),
    load_factor: typeof entry.account.load_factor === 'number' ? entry.account.load_factor : null,
    rate_multiplier:
      typeof entry.account.rate_multiplier === 'number' && entry.account.rate_multiplier !== 1
        ? entry.account.rate_multiplier
        : null,
    manual_rate: typeof extra.manual_rate === 'number' ? extra.manual_rate : null,
    rate_scale: typeof extra.rate_scale === 'number' ? extra.rate_scale : null,
    monitor_model: monitorModel(entry.account_id),
  }
  showSchedulerConfigDialog.value = true
}

function closeSchedulerConfig() {
  if (schedulerConfigSaving.value) return
  showSchedulerConfigDialog.value = false
  schedulerConfigEntry.value = null
}

function openSchedulerDetail(entry: AccountSchedulingEntry) {
  detailEntry.value = entry
  void loadSchedulerHistory()
}

function closeSchedulerDetail() {
  detailEntry.value = null
  historyItems.value = []
  historyQuery.value = ''
  historyEventFilter.value = 'all'
}

function toggleEntrySelected(entry: AccountSchedulingEntry) {
  const next = new Set(selectedAccountIds.value)
  if (next.has(entry.account_id)) next.delete(entry.account_id)
  else next.add(entry.account_id)
  selectedAccountIds.value = next
}

function toggleSelectAllVisible() {
  const next = new Set(selectedAccountIds.value)
  if (allVisibleSelected.value) {
    for (const entry of displayedEntries.value) next.delete(entry.account_id)
  } else {
    for (const entry of displayedEntries.value) next.add(entry.account_id)
  }
  selectedAccountIds.value = next
}

async function bulkClearTempUnsched() {
  const targets = selectedAccountCooldownEntries.value
  if (targets.length === 0) return
  bulkActionBusy.value = true
  try {
    await Promise.all(targets.map((entry) => adminAPI.accounts.resetTempUnschedulable(entry.account_id)))
    for (const entry of targets) {
      if (entry.account) {
        entry.account.temp_unschedulable_until = null
        entry.account.temp_unschedulable_reason = null
      }
    }
    appStore.showSuccess(t('admin.scheduler.bulkClearTempUnschedDone', { count: targets.length }))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.bulkActionFailed'))
  } finally {
    bulkActionBusy.value = false
  }
}

async function bulkEnableMonitor() {
  const targets = selectedMonitorableEntries.value
  if (targets.length === 0) return
  bulkActionBusy.value = true
  try {
    await Promise.all(targets.map(async (entry) => {
      const existing = monitorFor(entry.account_id)
      const model = monitorModel(entry.account_id).trim() || DEFAULT_MONITOR_MODEL
      if (existing) {
        const updated = await accountMonitorAPI.update(existing.id, { enabled: true, model })
        const map = new Map(monitorMap.value)
        map.set(entry.account_id, updated)
        monitorMap.value = map
        return
      }
      const provider = entry.account?.platform as AccountPlatform | undefined
      if (!provider || !MONITORABLE_PLATFORMS.includes(provider)) return
      const created = await accountMonitorAPI.create({
        account_id: entry.account_id,
        provider: provider as 'openai' | 'anthropic' | 'gemini',
        model,
        interval_seconds: DEFAULT_MONITOR_INTERVAL,
        enabled: true,
      })
      const map = new Map(monitorMap.value)
      map.set(entry.account_id, created)
      monitorMap.value = map
    }))
    appStore.showSuccess(t('admin.scheduler.bulkMonitorEnabled', { count: targets.length }))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.bulkActionFailed'))
  } finally {
    bulkActionBusy.value = false
  }
}

async function bulkRefreshUpstream() {
  const targets = selectedEntries.value.map((entry) => entry.account_id)
  if (targets.length === 0) return
  bulkActionBusy.value = true
  try {
    const refreshed = await refreshUpstreamStatuses(true, targets)
    if (refreshed) appStore.showSuccess(t('admin.scheduler.bulkRefreshDone', { count: targets.length }))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.bulkActionFailed'))
  } finally {
    bulkActionBusy.value = false
  }
}

function updateLocalAccount(account: Account) {
  const idx = entries.value.findIndex((entry) => entry.account_id === account.id)
  if (idx >= 0) {
    entries.value[idx] = {
      ...entries.value[idx],
      account,
    }
  }
}

async function ensureMonitorModel(entry: AccountSchedulingEntry, model: string) {
  if (!canMonitor(entry) || !model) return
  const existing = monitorFor(entry.account_id)
  if (existing) {
    if (existing.model === model) return
    const updated = await accountMonitorAPI.update(existing.id, { model })
    const map = new Map(monitorMap.value)
    map.set(entry.account_id, updated)
    monitorMap.value = map
    return
  }
  const provider = entry.account?.platform as AccountPlatform | undefined
  if (!provider || !MONITORABLE_PLATFORMS.includes(provider)) return
  const created = await accountMonitorAPI.create({
    account_id: entry.account_id,
    provider: provider as 'openai' | 'anthropic' | 'gemini',
    model,
    interval_seconds: DEFAULT_MONITOR_INTERVAL,
    enabled: false,
  })
  const map = new Map(monitorMap.value)
  map.set(entry.account_id, created)
  monitorMap.value = map
}

async function saveSchedulerConfig() {
  const entry = schedulerConfigEntry.value
  if (!entry?.account) return

  const form = schedulerConfigForm.value
  const concurrency = nonNegativeInt(form.concurrency, 0)
  const loadFactor = nonNegativeInt(form.load_factor, 0)
  const rateMultiplier = nonNegativeNumberOrNull(form.rate_multiplier)
  const manualRate = nonNegativeNumberOrNull(form.manual_rate)
  const rateScale = nonNegativeNumberOrNull(form.rate_scale)

  schedulerConfigSaving.value = true
  try {
    const accountPayload: {
      concurrency: number
      load_factor: number
      rate_multiplier?: number
      manual_rate: number | null
      rate_scale: number | null
    } = {
      concurrency,
      load_factor: loadFactor,
      manual_rate: manualRate,
      rate_scale: rateScale,
    }
    if (rateMultiplier != null) {
      accountPayload.rate_multiplier = rateMultiplier
    }
    const account = await adminAPI.accounts.updateSchedulerConfig(entry.account_id, accountPayload)
    updateLocalAccount(account)

    const model = form.monitor_model.trim() || DEFAULT_MONITOR_MODEL
    await ensureMonitorModel(entry, model)
    appStore.showSuccess(t('admin.scheduler.accountConfigSaved'))
    showSchedulerConfigDialog.value = false
    schedulerConfigEntry.value = null
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.scheduler.saveAccountConfigFailed'))
  } finally {
    schedulerConfigSaving.value = false
  }
}

onMounted(async () => {
  await loadGroups()
  const q = route.query.group
  const gid = q ? Number(q) : NaN
  if (Number.isFinite(gid) && groups.value.some((g) => g.id === gid)) {
    // URL 带 group 参数：恢复该分组。
    selectedGroupId.value = gid
    await loadEntries()
  } else if (groups.value.length > 0) {
    // 否则默认选中第一个分组。
    selectedGroupId.value = groups.value[0].id
    await onGroupChange()
  }
  // 定时刷新监控状态
  monitorPollTimer = setInterval(() => {
    if (monitorMap.value.size > 0) void refreshMonitorStatus()
  }, MONITOR_POLL_MS)
  upstreamStatusPollTimer = setInterval(() => {
    if (document.hidden || upstreamLoading.value || entries.value.length === 0) return
    void refreshUpstreamStatuses(false, undefined, true)
  }, UPSTREAM_STATUS_POLL_MS)
})

onUnmounted(() => {
  if (monitorPollTimer) clearInterval(monitorPollTimer)
  if (upstreamStatusPollTimer) clearInterval(upstreamStatusPollTimer)
  if (autoSortPollTimer) clearInterval(autoSortPollTimer)
})
</script>
