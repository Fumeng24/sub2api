<template>
  <AppLayout>
    <div class="admin-apple-page admin-table-page">
      <TablePageLayout>
        <template #filters>
          <div class="flex min-w-0 flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
            <div class="relative min-w-0 flex-1 sm:max-w-md">
              <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input v-model.trim="search" class="input pl-10" :placeholder="t('admin.upstreams.search')" @input="scheduleSearch" />
            </div>
            <div class="flex w-full flex-wrap items-center justify-end gap-2 xl:w-auto">
              <label class="inline-flex h-9 items-center gap-2 rounded border border-gray-200 bg-white px-2.5 text-xs text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300" :title="t('common.autoRefresh.title')">
                <input type="checkbox" class="h-3.5 w-3.5" :checked="upstreamAutoRefresh.enabled.value" @change="onAutoRefreshToggle" />
                <span class="hidden sm:inline">{{ t('common.autoRefresh.title') }}</span>
                <span v-if="upstreamAutoRefresh.enabled.value" class="tabular-nums text-gray-400">{{ upstreamAutoRefresh.countdown.value }}s</span>
              </label>
              <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadUpstreams()">
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
                <span class="ml-2 hidden sm:inline">{{ t('common.refresh') }}</span>
              </button>
              <button class="btn btn-secondary" :disabled="renamePreviewLoading" @click="openRenamePreview">
                <Icon name="edit" size="md" :class="renamePreviewLoading ? 'animate-pulse' : ''" />
                <span class="ml-2 hidden md:inline">{{ t('admin.upstreams.renameAccounts') }}</span>
              </button>
              <button class="btn btn-primary" @click="openCreate">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.upstreams.add') }}
              </button>
            </div>
          </div>
        </template>

        <template #table>
          <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
            <div class="hidden grid-cols-[minmax(240px,1.7fr)_110px_150px_170px_120px_160px] gap-4 border-b border-gray-200 px-5 py-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:border-dark-600 dark:text-gray-400 lg:grid">
              <span>{{ t('admin.upstreams.columns.upstream') }}</span>
              <span>{{ t('admin.upstreams.columns.kind') }}</span>
              <span>{{ t('admin.upstreams.columns.wallet') }}</span>
              <span>{{ t('admin.upstreams.columns.localGroups') }}</span>
              <span>{{ t('admin.upstreams.columns.accounts') }}</span>
              <span class="text-right">{{ t('admin.upstreams.columns.actions') }}</span>
            </div>

            <div v-if="loading" class="flex flex-1 items-center justify-center py-20 text-sm text-gray-500">
              {{ t('common.loading') }}
            </div>
            <div v-else-if="upstreams.length === 0" class="flex flex-1 items-center justify-center py-20 text-sm text-gray-500">
              {{ t('admin.upstreams.empty') }}
            </div>
            <div v-else class="min-h-0 flex-1 divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
              <div
                v-for="item in upstreams"
                :key="item.id"
                class="grid gap-4 px-5 py-4 transition-colors hover:bg-gray-50 dark:hover:bg-dark-800 lg:grid-cols-[minmax(240px,1.7fr)_110px_150px_170px_120px_160px] lg:items-center"
              >
                <button class="min-w-0 text-left" @click="openDetail(item)">
                  <div class="flex min-w-0 items-center gap-3">
                    <span :class="['h-2.5 w-2.5 shrink-0 rounded-full', statusDotClass(item.status)]" />
                    <div class="min-w-0">
                      <div class="truncate font-semibold text-gray-900 dark:text-white">{{ item.name }}</div>
                      <div class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ item.base_url }}</div>
                      <div :class="['mt-1 truncate text-[11px]', item.metadata?.refresh?.stale ? 'text-amber-600 dark:text-amber-400' : 'text-gray-400 dark:text-gray-500']">
                        {{ refreshStatusText(item) }}
                      </div>
                      <div
                        v-if="upstreamFailureReasons(item).length"
                        :class="['mt-1 line-clamp-2 break-words text-[11px] leading-4', failureTextClass(item)]"
                        :title="failureReasonsTitle(item)"
                      >
                        <span class="font-semibold">{{ t('admin.upstreams.failureReasonsTitle') }}：</span>
                        {{ failureSummaryText(item) }}
                      </div>
                    </div>
                  </div>
                </button>

                <div class="flex items-center gap-2 text-sm lg:block">
                  <span class="text-xs text-gray-500 lg:hidden">{{ t('admin.upstreams.columns.kind') }}</span>
                  <span class="badge badge-gray">{{ kindLabel(item.kind) }}</span>
                </div>

                <div class="flex items-center gap-2 text-sm lg:block">
                  <span class="text-xs text-gray-500 lg:hidden">{{ t('admin.upstreams.columns.wallet') }}</span>
                  <span v-if="walletBalance(item) != null" class="font-medium text-gray-800 dark:text-gray-200">
                    {{ formatMoney(walletBalance(item)!, walletUnit(item)) }}
                  </span>
                  <span v-else class="text-gray-400">{{ t('admin.upstreams.notAvailable') }}</span>
                </div>

                <div class="flex min-w-0 flex-wrap items-center gap-1.5 text-xs" :title="localGroupNames(item).join(' / ')">
                  <span v-for="group in item.local_groups || []" :key="group.id" class="badge badge-gray max-w-full">
                    <span class="truncate">{{ group.name }}</span>
                  </span>
                  <span v-if="!item.local_groups?.length" class="text-gray-400">{{ t('admin.upstreams.notBound') }}</span>
                </div>

                <div class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                  <Icon name="link" size="sm" class="text-gray-400" />
                  {{ item.account_count }}
                </div>

                <div class="flex items-center justify-start gap-1 lg:justify-end">
                  <button class="icon-button" :title="t('admin.upstreams.openDetail')" @click="openDetail(item)">
                    <Icon name="externalLink" size="md" />
                  </button>
                  <button class="icon-button" :title="t('admin.upstreams.probe')" :disabled="probingIds.has(item.id)" @click="probe(item)">
                    <Icon name="refresh" size="md" :class="probingIds.has(item.id) ? 'animate-spin' : ''" />
                  </button>
                  <button class="icon-button" :title="t('common.edit')" @click="openEdit(item)">
                    <Icon name="edit" size="md" />
                  </button>
                  <button class="icon-button text-red-600 hover:text-red-700" :title="t('common.delete')" @click="remove(item)">
                    <Icon name="trash" size="md" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>

        <template #pagination>
          <Pagination
            v-if="pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="changePage"
            @update:pageSize="changePageSize"
          />
        </template>
      </TablePageLayout>
    </div>

    <BaseDialog :show="showEditor" :title="editing ? t('admin.upstreams.edit') : t('admin.upstreams.add')" width="wide" @close="closeEditor">
      <form class="space-y-5" @submit.prevent="saveEditor">
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.upstreams.form.name') }}</label>
            <input v-model.trim="editor.name" class="input" required :placeholder="t('admin.upstreams.form.namePlaceholder')" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreams.form.kind') }}</label>
            <Select v-model="editor.kind" :options="kindOptions" />
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.form.baseUrl') }}</label>
          <input v-model.trim="editor.base_url" class="input font-mono" required placeholder="https://api.example.com" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.form.proxy') }}</label>
          <Select v-model="editor.proxy_id" :options="proxyOptions" :placeholder="t('admin.upstreams.form.direct')" />
        </div>

        <div class="border-t border-gray-200 pt-5 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreams.form.credentials') }}</h4>
            <span class="text-xs text-gray-500">{{ t('admin.upstreams.form.blankPreserves') }}</span>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <CredentialInput v-model="editor.credentials.api_key" :configured="credentialConfigured('has_api_key')" :clear="isCredentialCleared('api_key')" :label="t('admin.upstreams.form.apiKey')" @clear="toggleCredentialClear('api_key')" />
            <CredentialInput v-model="editor.credentials.management_access_token" :configured="credentialConfigured('has_management_access_token')" :clear="isCredentialCleared('management_access_token')" :label="t('admin.upstreams.form.managementToken')" @clear="toggleCredentialClear('management_access_token')" />
            <CredentialInput v-model="editor.credentials.management_user_id" :configured="credentialConfigured('has_management_user_id')" :clear="isCredentialCleared('management_user_id')" :label="t('admin.upstreams.form.managementUserId')" :secret="false" @clear="toggleCredentialClear('management_user_id')" />
            <CredentialInput v-model="editor.credentials.username" :configured="credentialConfigured('has_username')" :clear="isCredentialCleared('username')" :label="t('admin.upstreams.form.username')" :secret="false" @clear="toggleCredentialClear('username')" />
            <CredentialInput v-model="editor.credentials.password" :configured="credentialConfigured('has_password')" :clear="isCredentialCleared('password')" :label="t('admin.upstreams.form.password')" @clear="toggleCredentialClear('password')" />
          </div>
          <details class="mt-5">
            <summary class="cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.upstreams.form.protocolKeys') }}</summary>
            <div class="mt-4 grid gap-4 md:grid-cols-2">
              <CredentialInput v-model="editor.credentials.openai_api_key" :configured="credentialConfigured('has_openai_api_key')" :clear="isCredentialCleared('openai_api_key')" :label="t('admin.upstreams.form.openaiKey')" @clear="toggleCredentialClear('openai_api_key')" />
              <CredentialInput v-model="editor.credentials.anthropic_api_key" :configured="credentialConfigured('has_anthropic_api_key')" :clear="isCredentialCleared('anthropic_api_key')" :label="t('admin.upstreams.form.anthropicKey')" @clear="toggleCredentialClear('anthropic_api_key')" />
              <CredentialInput v-model="editor.credentials.gemini_api_key" :configured="credentialConfigured('has_gemini_api_key')" :clear="isCredentialCleared('gemini_api_key')" :label="t('admin.upstreams.form.geminiKey')" @clear="toggleCredentialClear('gemini_api_key')" />
              <CredentialInput v-model="editor.credentials.grok_api_key" :configured="credentialConfigured('has_grok_api_key')" :clear="isCredentialCleared('grok_api_key')" :label="t('admin.upstreams.form.grokKey')" @clear="toggleCredentialClear('grok_api_key')" />
            </div>
          </details>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" @click="closeEditor">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" type="button" :disabled="saving" @click="saveEditor">
            <Icon v-if="saving" name="refresh" size="md" class="mr-2 animate-spin" />
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="showRenamePreview" :title="t('admin.upstreams.renameDialog.title')" width="wide" @close="closeRenamePreview">
      <div v-if="renamePreviewLoading" class="flex min-h-48 items-center justify-center text-sm text-gray-500">
        <Icon name="refresh" size="md" class="mr-2 animate-spin" />{{ t('common.loading') }}
      </div>
      <div v-else-if="renamePreview" class="space-y-4">
        <div class="grid grid-cols-2 gap-4 border-b border-gray-200 pb-4 dark:border-dark-600">
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.upstreams.renameDialog.willRename') }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ renamePreview.renames }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.upstreams.renameDialog.willSkip') }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ renamePreview.skips }}</div>
          </div>
        </div>
        <div class="max-h-[50vh] divide-y divide-gray-100 overflow-y-auto border border-gray-200 dark:divide-dark-700 dark:border-dark-600">
          <div v-for="item in renamePreview.items" :key="item.account_id" class="grid gap-1 px-4 py-3 text-sm sm:grid-cols-[minmax(0,1fr)_auto] sm:gap-4">
            <div class="min-w-0">
              <div class="truncate text-gray-500 line-through" v-if="item.action === 'rename'">{{ item.current_name }}</div>
              <div :class="['break-words', item.action === 'rename' ? 'font-medium text-gray-900 dark:text-white' : 'text-gray-700 dark:text-gray-300']">
                {{ item.proposed_name || item.current_name }}
              </div>
            </div>
            <span :class="['self-center text-xs', item.action === 'rename' ? 'text-emerald-600' : 'text-gray-500']">
              {{ item.action === 'rename' ? t('admin.upstreams.renameDialog.rename') : renameSkipReason(item.reason) }}
            </span>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" :disabled="renameApplying" @click="closeRenamePreview">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" type="button" :disabled="renameApplying || !renamePreview?.renames" @click="applyRenamePreview">
            <Icon :name="renameApplying ? 'refresh' : 'check'" size="md" :class="renameApplying ? 'mr-2 animate-spin' : 'mr-2'" />
            {{ renameApplying ? t('admin.upstreams.renameDialog.applying') : t('admin.upstreams.renameDialog.apply') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog v-if="selectedUpstream" :show="showDetail" :title="selectedUpstream.name" width="full" @close="closeDetail">
      <div class="space-y-6">
        <div class="grid gap-4 border-b border-gray-200 pb-5 sm:grid-cols-2 lg:grid-cols-5 dark:border-dark-600">
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.upstreams.detail.status') }}</div>
            <div class="mt-1 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
              <span :class="['h-2.5 w-2.5 rounded-full', statusDotClass(selectedUpstream.status)]" />
              {{ statusLabel(selectedUpstream.status) }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.upstreams.detail.wallet') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
              {{ walletBalance(selectedUpstream) != null ? formatMoney(walletBalance(selectedUpstream)!, walletUnit(selectedUpstream)) : t('admin.upstreams.notAvailable') }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.upstreams.detail.groups') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ selectedUpstream.metadata?.groups?.length || 0 }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.upstreams.detail.accounts') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ selectedUpstream.account_count }}</div>
          </div>
          <div class="flex items-end justify-start gap-2 lg:justify-end">
            <button class="btn btn-secondary" :disabled="detailProbing" @click="probeSelected">
              <Icon name="refresh" size="md" :class="detailProbing ? 'mr-2 animate-spin' : 'mr-2'" />
              {{ t('admin.upstreams.probe') }}
            </button>
          </div>
        </div>

        <div
          v-if="selectedUpstream.duplicate_base_url_count > 0"
          class="flex items-start gap-3 border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200"
        >
          <Icon name="infoCircle" size="md" class="mt-0.5 shrink-0" />
          <span>{{ t('admin.upstreams.detail.duplicateBaseUrl', { count: selectedUpstream.duplicate_base_url_count }) }}</span>
        </div>

        <div
          v-if="upstreamFailureReasons(selectedUpstream).length"
          :class="['border px-4 py-3', failurePanelClass(selectedUpstream)]"
        >
          <div class="flex items-center gap-2 text-sm font-semibold">
            <Icon name="exclamationTriangle" size="md" class="shrink-0" />
            {{ t('admin.upstreams.failureReasonsTitle') }}
          </div>
          <div class="mt-3 divide-y divide-gray-200 dark:divide-white/15">
            <div
              v-for="reason in upstreamFailureReasons(selectedUpstream)"
              :key="reason.key"
              class="grid gap-1 py-2 text-xs first:pt-0 last:pb-0 sm:grid-cols-[minmax(150px,auto)_1fr] sm:gap-3"
            >
              <div class="flex flex-wrap items-center gap-1.5 self-start">
                <span class="font-semibold">{{ failureScopeLabel(selectedUpstream, reason) }}</span>
                <span class="rounded border border-gray-400/50 px-1.5 py-0.5 font-medium dark:border-white/25">{{ failureCodeLabel(reason.code) }}</span>
              </div>
              <p class="break-words leading-5">{{ failureReasonMessage(reason) }}</p>
            </div>
          </div>
        </div>

        <div class="flex gap-5 border-b border-gray-200 dark:border-dark-600">
          <button v-for="tab in detailTabs" :key="tab" :class="['-mb-px border-b-2 px-1 pb-3 text-sm font-medium', detailTab === tab ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 hover:text-gray-800 dark:hover:text-gray-200']" @click="detailTab = tab">
            {{ tab === 'capabilities' ? t('admin.upstreams.tabs.capabilities') : tab === 'binding' ? t('admin.upstreams.tabs.binding') : t('admin.upstreams.tabs.accounts') }}
          </button>
        </div>

        <section v-if="detailTab === 'capabilities'" class="space-y-5">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreams.detail.generateTitle') }}</h3>
            <p class="mt-1 text-xs text-gray-500">{{ t('admin.upstreams.detail.generateDescription') }}</p>
          </div>
          <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(300px,0.8fr)]">
            <div class="space-y-4">
              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.upstreams.detail.protocol') }}</label>
                  <Select v-model="generationForm.platform" :options="platformOptions" @change="onPlatformChange" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.upstreams.detail.upstreamGroup') }}</label>
                  <Select
                    v-model="generationForm.groupName"
                    :options="generationGroupOptions"
                    :placeholder="t('admin.upstreams.detail.selectGroup')"
                    :disabled="!generationGroupOptions.length"
                    @change="onUpstreamGroupChange"
                  />
                </div>
              </div>
              <div>
                <div class="mb-2 flex items-center justify-between gap-3">
                  <label class="input-label mb-0">{{ t('admin.upstreams.detail.models') }}</label>
                  <div class="flex flex-wrap items-center justify-end gap-2">
                    <span v-if="currentModels.length" class="text-xs text-gray-500">
                      {{ generationForm.models.length }}/{{ currentModels.length }}
                    </span>
                    <button class="text-xs font-medium text-primary-600 hover:text-primary-700 disabled:cursor-not-allowed disabled:text-gray-400" type="button" :disabled="!verifiedModels.length || groupModelsLoading || modelsBatchTesting" @click="toggleAllModels">
                      {{ allModelsSelected ? t('common.clear') : t('common.selectAll') }}
                    </button>
                    <button
                      class="btn btn-secondary btn-sm"
                      type="button"
                      :disabled="!currentModels.length || groupModelsLoading || modelsBatchTesting"
                      @click="testCandidateModels"
                    >
                      <Icon :name="modelsBatchTesting ? 'refresh' : 'play'" size="sm" :class="modelsBatchTesting ? 'mr-1 animate-spin' : 'mr-1'" />
                      {{ modelsBatchTesting ? t('admin.upstreams.detail.testingModels') : t('admin.upstreams.detail.testSelectedModels') }}
                    </button>
                  </div>
                </div>
                <div v-if="groupModelsLoading" class="border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600">
                  <Icon name="refresh" size="md" class="mr-2 inline-block animate-spin" />{{ t('admin.upstreams.detail.loadingModels') }}
                </div>
                <div v-else-if="modelProbeError" class="border border-red-200 bg-red-50 px-4 py-4 text-sm text-red-800 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-200">
                  {{ modelProbeError }}
                </div>
                <div v-else-if="currentModels.length" class="upstream-model-list grid max-h-56 gap-2 overflow-y-auto border border-gray-200 p-3 sm:grid-cols-2 dark:border-dark-600">
                  <div v-for="model in currentModels" :key="model" class="flex min-w-0 items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                    <label :class="['flex min-h-11 min-w-0 flex-1 items-center gap-2 py-1', modelProbeResults[model]?.success ? 'cursor-pointer' : 'cursor-not-allowed']">
                      <input v-model="generationForm.models" type="checkbox" :value="model" :disabled="!modelProbeResults[model]?.success" class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-40" />
                      <span class="truncate font-mono text-xs">{{ model }}</span>
                    </label>
                    <span
                      :class="['flex h-7 w-7 shrink-0 items-center justify-center rounded-full', modelProbeResults[model] ? modelProbeStatusClass(modelProbeResults[model]) : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400']"
                      :title="modelProbeResults[model]?.message || t('admin.upstreams.detail.modelNotTested')"
                    >
                      <Icon :name="modelProbeResults[model] ? modelProbeStatusIcon(modelProbeResults[model]) : 'clock'" size="sm" />
                    </span>
                    <button
                      type="button"
                      class="icon-button min-h-10 min-w-10 shrink-0"
                      :title="t('admin.upstreams.detail.testModel')"
                      :aria-label="t('admin.upstreams.detail.testModel')"
                      :disabled="!generationForm.groupName || modelTestingKey === modelTestKey(model) || modelsBatchTesting"
                      @click="testModel(model)"
                    >
                      <Icon :name="modelTestingKey === modelTestKey(model) ? 'refresh' : 'play'" size="sm" :class="modelTestingKey === modelTestKey(model) ? 'animate-spin' : ''" />
                    </button>
                  </div>
                </div>
                <div v-else class="border border-dashed border-gray-300 px-4 py-6 text-center text-sm text-gray-500 dark:border-dark-600">
                  {{ generationForm.groupName ? t('admin.upstreams.detail.noModels') : t('admin.upstreams.detail.selectGroupForModels') }}
                </div>
              </div>
              <div>
                <label class="input-label">{{ t('admin.upstreams.detail.localGroups') }}</label>
                <div v-if="localGroups.length" class="grid max-h-48 gap-2 overflow-y-auto border border-gray-200 p-3 sm:grid-cols-2 dark:border-dark-600">
                  <label v-for="group in localGroups" :key="group.id" class="flex min-w-0 items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                    <input v-model="generationForm.localGroupIds" type="checkbox" :value="group.id" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                    <span class="truncate">{{ group.name }}</span>
                  </label>
                </div>
                <div v-else class="border border-dashed border-gray-300 px-4 py-6 text-center text-sm text-gray-500 dark:border-dark-600">{{ t('admin.upstreams.detail.noLocalGroups') }}</div>
              </div>
            </div>
            <div class="space-y-4">
              <div>
                <label class="input-label">{{ t('admin.upstreams.detail.accountName') }}</label>
                <input v-model.trim="generationForm.name" class="input" :placeholder="t('admin.upstreams.detail.accountNamePlaceholder')" />
              </div>
              <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.upstreams.detail.concurrency') }}</label>
                  <input v-model.number="generationForm.concurrency" type="number" min="1" max="10000" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.upstreams.detail.priority') }}</label>
                  <input v-model.number="generationForm.priority" type="number" min="0" class="input" />
                </div>
              </div>
              <div>
                <label class="input-label">{{ t('admin.upstreams.detail.keyOverride') }}</label>
                <input v-model.trim="generationForm.apiKey" type="password" class="input font-mono" :placeholder="t('admin.upstreams.detail.keyOverridePlaceholder')" autocomplete="new-password" />
              </div>
              <button class="btn btn-secondary w-full" :disabled="!canAddSpec" @click="addGenerationSpec">
                <Icon name="plus" size="md" class="mr-2" />
                {{ t('admin.upstreams.detail.addToPlan') }}
              </button>
            </div>
          </div>

          <div v-if="generationSpecs.length" class="border-t border-gray-200 pt-5 dark:border-dark-600">
            <div class="mb-3 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreams.detail.plan') }}</h3>
              <button class="text-xs text-gray-500 hover:text-red-600" @click="generationSpecs = []">{{ t('common.clear') }}</button>
            </div>
            <div class="divide-y divide-gray-100 border border-gray-200 dark:divide-dark-700 dark:border-dark-600">
              <div v-for="(spec, index) in generationSpecs" :key="`${spec.platform}-${spec.upstream_group_name}-${index}`" class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
                <div class="min-w-0">
                  <span class="font-semibold text-gray-900 dark:text-white">{{ spec.name || defaultName(spec) }}</span>
                  <span class="ml-2 text-gray-500">{{ protocolLabel(spec.platform) }} · {{ spec.upstream_group_name }}</span>
                </div>
                <div class="flex items-center gap-3 text-xs text-gray-500">
                  <span v-if="spec.rate_multiplier != null">{{ spec.rate_multiplier }}x</span>
                  <span>{{ spec.models.length }} {{ t('admin.upstreams.detail.modelsShort') }}</span>
                  <span>{{ spec.local_group_ids.length }} {{ t('admin.upstreams.detail.localGroupsShort') }}</span>
                  <button class="text-red-600 hover:text-red-700" :title="t('common.remove')" @click="generationSpecs.splice(index, 1)"><Icon name="x" size="sm" /></button>
                </div>
              </div>
            </div>
            <div class="mt-4 flex justify-end">
              <button class="btn btn-primary" :disabled="previewing || !generationSpecs.length" @click="previewGeneration">
                <Icon name="check" size="md" class="mr-2" />
                {{ previewing ? t('admin.upstreams.detail.previewing') : t('admin.upstreams.detail.preview') }}
              </button>
            </div>
          </div>

          <div v-if="generationPreview" class="border-t border-gray-200 pt-5 dark:border-dark-600">
            <div class="mb-3 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreams.detail.previewTitle') }}</h3>
              <span :class="generationPreview.valid ? 'text-emerald-600' : 'text-red-600'">{{ generationPreview.valid ? t('admin.upstreams.detail.previewValid') : t('admin.upstreams.detail.previewInvalid') }}</span>
            </div>
            <div class="space-y-2">
              <div v-for="entry in generationPreview.items" :key="entry.index" class="border px-4 py-3 text-sm" :class="entry.errors.length ? 'border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-950/20' : 'border-gray-200 dark:border-dark-600'">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ entry.name }}</span>
                  <span class="text-xs text-gray-500">{{ entry.action === 'skip' ? t('admin.upstreams.detail.previewSkip') : entry.will_create_upstream_key ? t('admin.upstreams.detail.previewCreateKey') : t('admin.upstreams.detail.previewCreate') }}</span>
                </div>
                <div v-if="entry.errors.length" class="mt-2 text-xs text-red-700 dark:text-red-300">{{ entry.errors.join('; ') }}</div>
                <div v-if="entry.warnings.length" class="mt-2 text-xs text-amber-700 dark:text-amber-300">{{ entry.warnings.join('; ') }}</div>
              </div>
            </div>
            <div class="mt-4 flex justify-end">
              <button class="btn btn-primary" :disabled="generating || !generationPreview.valid" @click="generatePlannedAccounts">
                <Icon name="server" size="md" class="mr-2" />
                {{ generating ? t('admin.upstreams.detail.generating') : t('admin.upstreams.detail.confirmGenerate') }}
              </button>
            </div>
          </div>
        </section>

        <section v-else-if="detailTab === 'binding'" class="space-y-5">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreams.detail.bindTitle') }}</h3>
              <p class="mt-1 text-xs text-gray-500">{{ t('admin.upstreams.detail.bindOnlyApiKeys') }}</p>
              <p class="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300">{{ t('admin.upstreams.detail.bindPreservesAccount') }}</p>
            </div>
            <button class="btn btn-primary" :disabled="!selectedCandidateIds.size || binding" @click="bindSelected">
              <Icon name="link" size="md" class="mr-2" />
              {{ binding ? t('admin.upstreams.detail.binding') : t('admin.upstreams.detail.bindSelected', { count: selectedCandidateIds.size }) }}
            </button>
          </div>
          <div class="flex items-center gap-3">
            <div class="relative min-w-0 flex-1 sm:max-w-sm">
              <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input v-model.trim="candidateSearch" class="input pl-9" :placeholder="t('admin.upstreams.detail.searchAccounts')" @input="loadCandidates" />
            </div>
            <button class="icon-button" :title="t('common.refresh')" @click="loadCandidates"><Icon name="refresh" size="md" :class="candidatesLoading ? 'animate-spin' : ''" /></button>
          </div>
          <div class="max-h-96 overflow-y-auto border border-gray-200 dark:border-dark-600">
            <div v-if="candidatesLoading" class="px-4 py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
            <div v-else-if="candidates.length === 0" class="px-4 py-8 text-center text-sm text-gray-500">{{ t('admin.upstreams.detail.noCandidates') }}</div>
            <label v-for="account in candidates" :key="account.id" class="flex cursor-pointer items-center gap-3 border-b border-gray-100 px-4 py-3 last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-800">
              <input v-model="selectedCandidateIdsArray" type="checkbox" :value="account.id" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span :class="['h-2 w-2 rounded-full', account.status === 'active' ? 'bg-emerald-500' : 'bg-gray-400']" />
              <span class="min-w-0 flex-1 truncate text-sm text-gray-800 dark:text-gray-200">{{ account.name }}</span>
              <span class="text-xs text-gray-500">{{ protocolLabel(account.platform) }}</span>
            </label>
          </div>
        </section>

        <section v-else class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreams.detail.boundAccounts') }}</h3>
            <div class="flex flex-wrap items-center justify-end gap-3">
              <label class="inline-flex min-h-10 items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
                <input v-model="deleteAccountsOnUnbind" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ t('admin.upstreams.detail.deleteOnUnbind') }}
              </label>
              <button class="btn btn-secondary" type="button" @click="detailTab = 'binding'"><Icon name="link" size="md" class="mr-2" />{{ t('admin.upstreams.detail.manageBinding') }}</button>
            </div>
          </div>
          <div v-if="selectedUpstream.accounts?.length" class="border border-gray-200 dark:border-dark-600">
            <div class="hidden grid-cols-[minmax(220px,1.2fr)_minmax(240px,1fr)_120px_110px_44px] gap-4 border-b border-gray-200 bg-gray-50 px-4 py-3 text-xs font-semibold uppercase text-gray-500 md:grid dark:border-dark-600 dark:bg-dark-800">
              <span>{{ t('admin.upstreams.detail.account') }}</span>
              <span>{{ t('admin.upstreams.detail.upstreamGroup') }}</span>
              <span>{{ t('admin.upstreams.detail.upstreamRate') }}</span>
              <span>{{ t('admin.upstreams.detail.accountStatus') }}</span>
              <span class="text-right">{{ t('admin.upstreams.columns.actions') }}</span>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <div
                v-for="account in selectedUpstream.accounts"
                :key="account.id"
                class="grid gap-4 px-4 py-4 md:grid-cols-[minmax(220px,1.2fr)_minmax(240px,1fr)_120px_110px_44px] md:items-center"
              >
                <div class="min-w-0">
                  <div class="truncate font-medium text-gray-900 dark:text-white">{{ account.name }}</div>
                  <div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500">
                    <span>{{ protocolLabel(account.platform) }}</span>
                    <span>{{ account.generated ? t('admin.upstreams.detail.generated') : t('admin.upstreams.detail.manual') }}</span>
                  </div>
                </div>
                <div class="min-w-0">
                  <div class="flex min-w-0 items-center gap-2">
                    <Select
                      :model-value="accountGroupSelection(account)"
                      :options="accountGroupOptions(account)"
                      :disabled="accountGroupChanging.has(account.id) || !account.upstream_group_change_supported || groupCatalogueIsStale()"
                      :placeholder="t('admin.upstreams.detail.selectGroup')"
                      class="min-w-0 flex-1"
                      searchable
                      @update:model-value="setAccountGroupSelection(account.id, $event)"
                    />
                    <button
                      class="icon-button shrink-0 text-primary-600"
                      :title="t('admin.upstreams.detail.applyGroupChange')"
                      :disabled="!canApplyAccountGroup(account) || accountGroupChanging.has(account.id)"
                      @click="applyAccountGroupChange(account)"
                    >
                      <Icon :name="accountGroupChanging.has(account.id) ? 'refresh' : 'check'" size="md" :class="accountGroupChanging.has(account.id) ? 'animate-spin' : ''" />
                    </button>
                  </div>
                  <p v-if="!account.upstream_group_change_supported" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
                    {{ accountGroupChangeReason(account) }}
                  </p>
                  <p v-else-if="groupCatalogueIsStale()" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
                    {{ t('admin.upstreams.detail.groupCatalogueStale') }}
                  </p>
                  <p v-else-if="account.upstream_group_stale" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
                    {{ t('admin.upstreams.detail.groupDataStale') }}
                  </p>
                </div>
                <div class="flex items-center gap-2 text-sm md:block">
                  <span class="text-xs text-gray-500 md:hidden">{{ t('admin.upstreams.detail.upstreamRate') }}</span>
                  <span v-if="account.upstream_group_rate_multiplier != null" class="font-medium text-gray-800 dark:text-gray-200">{{ formatRate(account.upstream_group_rate_multiplier) }}x</span>
                  <span v-else class="text-gray-400">{{ t('admin.upstreams.notAvailable') }}</span>
                </div>
                <div class="flex items-center gap-2">
                  <span :class="['h-2 w-2 shrink-0 rounded-full', account.status === 'active' ? 'bg-emerald-500' : 'bg-gray-400']" />
                  <span class="text-sm text-gray-700 dark:text-gray-300">{{ account.status }}</span>
                </div>
                <div class="flex justify-end">
                  <button class="icon-button text-red-600" :title="t('admin.upstreams.detail.unbind')" :disabled="accountGroupChanging.has(account.id)" @click="unbind(account.id)"><Icon name="x" size="md" /></button>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-600">{{ t('admin.upstreams.detail.noBoundAccounts') }}</div>
        </section>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/custom/api/admin'
import type { AdminGroup, GroupPlatform, Proxy } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { formatMultiplier } from '@/custom/utils/formatters'
import type {
  ManagedUpstreamPlatform,
  Upstream,
  UpstreamAccountBillingMetadata,
  UpstreamAccountGenerationPreview,
  UpstreamAccountGenerationSpec,
  UpstreamAccountRenamePreview,
  UpstreamAccountSummary,
  UpstreamGroupMetadata,
  UpstreamKind,
  UpstreamModelProbeResult,
  UpstreamMutationRequest
} from '@/custom/api/admin/upstreams'

interface UpstreamFailureReason {
  key: string
  scope: 'management' | 'protocol' | 'account' | 'general'
  code: string
  message: string
  accountId?: number
  platform?: string
}

const { t } = useI18n()
const appStore = useAppStore()

const CredentialInput = defineComponent({
  props: {
    modelValue: { type: String, default: '' },
    configured: { type: Boolean, default: false },
    clear: { type: Boolean, default: false },
    label: { type: String, required: true },
    secret: { type: Boolean, default: true }
  },
  emits: ['update:modelValue', 'clear'],
  setup(props, { emit }) {
    return () => h('div', [
      h('label', { class: 'input-label' }, props.label),
      h('div', { class: 'flex items-center gap-2' }, [
        h('input', {
          class: 'input min-w-0 flex-1 font-mono',
          type: props.secret ? 'password' : 'text',
          value: props.modelValue,
          autocomplete: 'new-password',
          placeholder: props.configured ? t('admin.upstreams.form.configuredPlaceholder') : t('admin.upstreams.form.emptyPlaceholder'),
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value)
        }),
        props.configured
          ? h('button', { type: 'button', class: ['icon-button', props.clear ? 'text-red-600' : ''], title: props.clear ? t('common.cancel') : t('admin.upstreams.form.clearCredential'), onClick: () => emit('clear') }, [h(Icon, { name: props.clear ? 'x' : 'trash', size: 'sm' })])
          : null
      ])
    ])
  }
})

const upstreams = ref<Upstream[]>([])
const loading = ref(false)
const saving = ref(false)
const search = ref('')
const page = ref(1)
const pageSize = ref(20)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const probingIds = reactive(new Set<number>())
let searchTimer: number | undefined

const proxies = ref<Proxy[]>([])
const proxyOptions = computed(() => [{ value: null, label: t('admin.upstreams.form.direct') }, ...proxies.value.map(proxy => ({ value: proxy.id, label: proxy.name }))])
const kindOptions = computed(() => [
  { value: 'auto', label: t('admin.upstreams.kind.auto') },
  { value: 'newapi', label: 'NewAPI' },
  { value: 'sub2api', label: 'Sub2API' }
])

const showEditor = ref(false)
const editing = ref<Upstream | null>(null)
const editor = reactive({
  name: '',
  base_url: '',
  kind: 'auto' as UpstreamKind,
  proxy_id: null as number | null,
  credentials: {
    api_key: '',
    management_access_token: '',
    management_user_id: '',
    username: '',
    password: '',
    openai_api_key: '',
    anthropic_api_key: '',
    gemini_api_key: '',
    grok_api_key: ''
  },
  clearCredentials: [] as string[]
})

const selectedUpstream = ref<Upstream | null>(null)
const showDetail = ref(false)
const detailProbing = ref(false)
const detailTab = ref<'capabilities' | 'binding' | 'accounts'>('capabilities')
const detailTabs = ['capabilities', 'binding', 'accounts'] as const

const platformOptions = computed(() => [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'grok', label: 'Grok' }
])
const generationForm = reactive({
  platform: 'openai' as ManagedUpstreamPlatform,
  groupName: '',
  models: [] as string[],
  localGroupIds: [] as number[],
  name: '',
  concurrency: 3,
  priority: 50,
  apiKey: ''
})
const localGroups = ref<AdminGroup[]>([])
const generationSpecs = ref<UpstreamAccountGenerationSpec[]>([])
const generationPreview = ref<UpstreamAccountGenerationPreview | null>(null)
const previewing = ref(false)
const generating = ref(false)
const modelTestingKey = ref<string | null>(null)
const modelsBatchTesting = ref(false)
const groupModelsLoading = ref(false)
const modelProbeError = ref('')
const currentModels = ref<string[]>([])
const modelProbeResults = reactive<Record<string, UpstreamModelProbeResult>>({})
const modelProbeBatchSize = 100
let groupModelsRequestID = 0
let modelProbeRequestID = 0
const deleteAccountsOnUnbind = ref(true)

const candidates = ref<UpstreamAccountSummary[]>([])
const candidatesLoading = ref(false)
const candidateSearch = ref('')
const selectedCandidateIdsArray = ref<number[]>([])
const selectedCandidateIds = computed(() => new Set(selectedCandidateIdsArray.value))
const binding = ref(false)
const accountGroupChanging = reactive(new Set<number>())
const accountGroupSelections = reactive<Record<number, string | null>>({})

const renamePreviewLoading = ref(false)
const showRenamePreview = ref(false)
const renamePreview = ref<UpstreamAccountRenamePreview | null>(null)
const renameApplying = ref(false)

const upstreamAutoRefresh = useAutoRefresh({
  storageKey: 'admin-upstreams-auto-refresh',
  intervals: [30] as const,
  defaultInterval: 30,
  defaultEnabled: true,
  onRefresh: () => loadUpstreams(true),
  shouldPause: () => document.hidden || showEditor.value || showDetail.value || saving.value || probingIds.size > 0
})

const verifiedModels = computed(() => currentModels.value.filter(model => modelProbeResults[model]?.success))
const allModelsSelected = computed(() => verifiedModels.value.length > 0 && verifiedModels.value.every(model => generationForm.models.includes(model)))
const generationGroupOptions = computed(() => {
  const groups = selectedUpstream.value?.metadata?.groups || []
  return groups
    .filter(group => !group.platform || group.platform === generationForm.platform)
    .map(group => ({ value: group.name, label: group.rate_multiplier != null ? `${group.name} · ${group.rate_multiplier}x` : group.name }))
})
const canAddSpec = computed(() => Boolean(
  generationForm.groupName
  && generationForm.models.length
  && generationForm.models.every(model => modelProbeResults[model]?.success)
  && generationForm.localGroupIds.length
))

watch(() => generationForm.apiKey, (value, previous) => {
  if (value === previous) return
  clearModelVerificationState()
})

function errorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error) return String((error as { message?: unknown }).message || '')
  return t('common.operationFailed')
}

async function loadUpstreams(silent = false) {
  if (!silent) loading.value = true
  try {
    const result = await adminAPI.upstreams.list(page.value, pageSize.value, search.value)
    upstreams.value = result.items || []
    pagination.page = result.page
    pagination.page_size = result.page_size
    pagination.total = result.total
  } catch (error) {
    if (!silent) appStore.showError(errorMessage(error))
  } finally {
    if (!silent) loading.value = false
  }
}

function onAutoRefreshToggle(event: Event) {
  upstreamAutoRefresh.setEnabled((event.target as HTMLInputElement).checked)
}

async function loadProxies() {
  try {
    proxies.value = await adminAPI.proxies.getAll()
  } catch {
    proxies.value = []
  }
}

function scheduleSearch() {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadUpstreams()
  }, 250)
}

function changePage(next: number) {
  page.value = next
  void loadUpstreams()
}

function changePageSize(next: number) {
  pageSize.value = next
  page.value = 1
  void loadUpstreams()
}

function resetEditor() {
  editor.name = ''
  editor.base_url = ''
  editor.kind = 'auto'
  editor.proxy_id = null
  editor.clearCredentials = []
  for (const key of Object.keys(editor.credentials) as Array<keyof typeof editor.credentials>) editor.credentials[key] = ''
}

function openCreate() {
  editing.value = null
  resetEditor()
  showEditor.value = true
}

function openEdit(item: Upstream) {
  editing.value = item
  resetEditor()
  editor.name = item.name
  editor.base_url = item.base_url
  editor.kind = item.kind
  editor.proxy_id = item.proxy_id ?? null
  showEditor.value = true
}

function closeEditor() {
  if (!saving.value) showEditor.value = false
}

function credentialConfigured(key: keyof Upstream['credential_status']): boolean {
  return Boolean(editing.value?.credential_status?.[key])
}

function isCredentialCleared(key: string): boolean {
  return editor.clearCredentials.includes(key)
}

function toggleCredentialClear(key: string) {
  if (editor.clearCredentials.includes(key)) editor.clearCredentials = editor.clearCredentials.filter(item => item !== key)
  else editor.clearCredentials.push(key)
}

async function saveEditor() {
  if (!editor.name.trim() || !editor.base_url.trim()) return
  saving.value = true
  try {
    const creating = editing.value == null
    const credentials: Record<string, string> = {}
    for (const [key, value] of Object.entries(editor.credentials)) if (value.trim()) credentials[key] = value.trim()
    const payload: UpstreamMutationRequest = {
      name: editor.name.trim(),
      base_url: editor.base_url.trim(),
      kind: editor.kind,
      proxy_id: editor.proxy_id,
      clear_proxy: editor.proxy_id == null && Boolean(editing.value?.proxy_id),
      credentials,
      clear_credentials: editor.clearCredentials
    }
    let result = editing.value ? await adminAPI.upstreams.update(editing.value.id, payload) : await adminAPI.upstreams.create(payload)
    showEditor.value = false
    await loadUpstreams()
    appStore.showSuccess(t('admin.upstreams.saved'))
    if (creating) {
      try {
        result = await adminAPI.upstreams.probe(result.id)
        await loadUpstreams()
      } catch (error) {
        appStore.showError(t('admin.upstreams.savedProbeFailed', { error: errorMessage(error) }))
      }
    }
    await openDetail(result)
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    saving.value = false
  }
}

async function probe(item: Upstream) {
  probingIds.add(item.id)
  try {
    const result = await adminAPI.upstreams.probe(item.id)
    const index = upstreams.value.findIndex(current => current.id === item.id)
    if (index >= 0) upstreams.value[index] = result
    if (selectedUpstream.value?.id === item.id) selectedUpstream.value = result
    appStore.showSuccess(t('admin.upstreams.probeDone'))
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    probingIds.delete(item.id)
  }
}

async function remove(item: Upstream) {
  if (!window.confirm(t('admin.upstreams.deleteConfirm', { name: item.name }))) return
  try {
    await adminAPI.upstreams.delete(item.id)
    await loadUpstreams()
    appStore.showSuccess(t('admin.upstreams.deleted'))
  } catch (error) {
    appStore.showError(errorMessage(error))
  }
}

async function openDetail(item: Upstream) {
  try {
    selectedUpstream.value = await adminAPI.upstreams.get(item.id)
    clearAccountGroupSelections()
    showDetail.value = true
    detailTab.value = 'capabilities'
    generationPreview.value = null
    generationSpecs.value = []
    resetGenerationForm()
    await loadLocalGroups()
    await loadCandidates()
  } catch (error) {
    appStore.showError(errorMessage(error))
  }
}

function closeDetail() {
  if (!generating.value && !binding.value) showDetail.value = false
}

async function probeSelected() {
  if (!selectedUpstream.value) return
  detailProbing.value = true
  try {
    selectedUpstream.value = await adminAPI.upstreams.probe(selectedUpstream.value.id)
    await loadUpstreams()
    await onPlatformChange(generationForm.platform)
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    detailProbing.value = false
  }
}

function resetGenerationForm() {
  generationForm.platform = 'openai'
  generationForm.groupName = ''
  generationForm.models = []
  generationForm.localGroupIds = []
  generationForm.name = ''
  generationForm.concurrency = 3
  generationForm.priority = 50
  generationForm.apiKey = ''
  resetGroupModels()
  deleteAccountsOnUnbind.value = true
}

async function loadLocalGroups() {
  if (!selectedUpstream.value) return
  try {
    localGroups.value = await adminAPI.groups.getByPlatform(generationForm.platform as GroupPlatform)
  } catch {
    localGroups.value = []
  }
}

async function onPlatformChange(value: string | number | boolean | null) {
  if (typeof value === 'string') generationForm.platform = value as ManagedUpstreamPlatform
  generationForm.groupName = ''
  generationForm.localGroupIds = []
  resetGroupModels()
  await loadLocalGroups()
}

function resetGroupModels() {
  groupModelsRequestID += 1
  groupModelsLoading.value = false
  currentModels.value = []
  clearModelVerificationState()
}

function clearModelVerificationState() {
  modelProbeRequestID += 1
  generationForm.models = []
  modelProbeError.value = ''
  modelTestingKey.value = null
  modelsBatchTesting.value = false
  for (const model of Object.keys(modelProbeResults)) delete modelProbeResults[model]
}

async function onUpstreamGroupChange(value: string | number | boolean | null) {
  generationForm.groupName = typeof value === 'string' ? value : ''
  resetGroupModels()
  if (!generationForm.groupName) return
  await discoverSelectedGroupModels()
}

async function discoverSelectedGroupModels() {
  if (!selectedUpstream.value || !generationForm.groupName) return
  const requestID = ++groupModelsRequestID
  groupModelsLoading.value = true
  modelProbeError.value = ''
  try {
    const result = await adminAPI.upstreams.probeModels(selectedUpstream.value.id, {
      platform: generationForm.platform,
      group_name: generationForm.groupName,
      models: []
    })
    if (requestID !== groupModelsRequestID) return
    if (result.status === 'error') {
      currentModels.value = []
      generationForm.models = []
      modelProbeError.value = result.message || t('admin.upstreams.detail.modelsLoadFailed')
      return
    }
    currentModels.value = [...result.available_models]
    generationForm.models = []
  } catch (error) {
    if (requestID !== groupModelsRequestID) return
    currentModels.value = []
    generationForm.models = []
    modelProbeError.value = errorMessage(error)
  } finally {
    if (requestID === groupModelsRequestID) groupModelsLoading.value = false
  }
}

function toggleAllModels() {
  generationForm.models = allModelsSelected.value ? [] : [...verifiedModels.value]
}

function modelTestKey(model: string) {
  return `${generationForm.platform}:${generationForm.groupName}:${model}`
}

async function testModel(model: string) {
  if (!selectedUpstream.value || !generationForm.groupName || modelTestingKey.value || modelsBatchTesting.value) return
  const key = modelTestKey(model)
  const requestID = ++modelProbeRequestID
  modelTestingKey.value = key
  try {
    const batch = await adminAPI.upstreams.probeModels(selectedUpstream.value.id, {
      platform: generationForm.platform,
      group_name: generationForm.groupName,
      models: [model],
      api_key: generationForm.apiKey.trim() || undefined
    })
    if (requestID !== modelProbeRequestID) return
    const result = batch.results.find(entry => entry.model === model)
    if (!result) throw new Error(batch.message || t('admin.upstreams.detail.modelsLoadFailed'))
    modelProbeResults[model] = result
    if (result.success) {
      if (!generationForm.models.includes(model)) generationForm.models.push(model)
      appStore.showSuccess(t('admin.upstreams.detail.testSuccess', { model, latency: result.latency_ms }))
    } else {
      generationForm.models = generationForm.models.filter(item => item !== model)
      appStore.showError(t('admin.upstreams.detail.testFailed', { model, message: result.message || t('common.operationFailed') }))
    }
  } catch (error) {
    if (requestID !== modelProbeRequestID) return
    appStore.showError(t('admin.upstreams.detail.testFailed', { model, message: errorMessage(error) }))
  } finally {
    if (requestID === modelProbeRequestID) modelTestingKey.value = null
  }
}

async function testCandidateModels() {
  if (!selectedUpstream.value || !generationForm.groupName || !currentModels.value.length || modelsBatchTesting.value) return
  const upstreamID = selectedUpstream.value.id
  const requestID = ++modelProbeRequestID
  modelsBatchTesting.value = true
  modelTestingKey.value = null
  for (const model of Object.keys(modelProbeResults)) delete modelProbeResults[model]
  try {
    const results: UpstreamModelProbeResult[] = []
    let latencyMs = 0
    const messages: string[] = []
    for (let offset = 0; offset < currentModels.value.length; offset += modelProbeBatchSize) {
      const batch = await adminAPI.upstreams.probeModels(upstreamID, {
        platform: generationForm.platform,
        group_name: generationForm.groupName,
        models: currentModels.value.slice(offset, offset + modelProbeBatchSize),
        api_key: generationForm.apiKey.trim() || undefined
      })
      if (requestID !== modelProbeRequestID) return
      latencyMs += batch.latency_ms
      if (batch.message && !messages.includes(batch.message)) messages.push(batch.message)
      for (const entry of batch.results) {
        results.push(entry)
        modelProbeResults[entry.model] = entry
      }
    }
    generationForm.models = currentModels.value.filter(model => modelProbeResults[model]?.success)
    const failed = results.filter(entry => !entry.success).length
    if (failed > 0) {
      appStore.showError(t('admin.upstreams.detail.batchTestFailed', { failed, total: results.length, message: messages.join('; ') || t('common.operationFailed') }))
    } else {
      appStore.showSuccess(t('admin.upstreams.detail.batchTestSuccess', { count: results.length, latency: latencyMs }))
    }
  } catch (error) {
    if (requestID !== modelProbeRequestID) return
    appStore.showError(t('admin.upstreams.detail.batchTestRequestFailed', { message: errorMessage(error) }))
  } finally {
    if (requestID === modelProbeRequestID) modelsBatchTesting.value = false
  }
}

function modelProbeStatusClass(result: UpstreamModelProbeResult): string {
  if (result.success) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  if (result.status === 'unsupported') return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
}

function modelProbeStatusIcon(result: UpstreamModelProbeResult): 'check' | 'exclamationTriangle' | 'x' {
  if (result.success) return 'check'
  return result.status === 'unsupported' ? 'x' : 'exclamationTriangle'
}

function addGenerationSpec() {
  if (!selectedUpstream.value || !canAddSpec.value) return
  const group = selectedUpstream.value.metadata?.groups?.find(item => item.name === generationForm.groupName)
  generationSpecs.value.push({
    name: generationForm.name.trim() || undefined,
    platform: generationForm.platform,
    upstream_group_name: generationForm.groupName,
    upstream_group_id: group?.id,
    models: [...generationForm.models],
    local_group_ids: [...generationForm.localGroupIds],
    concurrency: generationForm.concurrency,
    priority: generationForm.priority,
    rate_multiplier: group?.rate_multiplier,
    api_key: generationForm.apiKey.trim() || undefined
  })
  generationForm.name = ''
  generationForm.apiKey = ''
  generationPreview.value = null
}

async function previewGeneration() {
  if (!selectedUpstream.value) return
  previewing.value = true
  try {
    generationPreview.value = await adminAPI.upstreams.previewGeneratedAccounts(selectedUpstream.value.id, generationSpecs.value)
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    previewing.value = false
  }
}

async function generatePlannedAccounts() {
  if (!selectedUpstream.value || !generationPreview.value?.valid) return
  generating.value = true
  try {
    const results = await adminAPI.upstreams.generateAccounts(selectedUpstream.value.id, generationSpecs.value)
    const failures = results.filter(result => !result.success)
    if (failures.length) appStore.showError(t('admin.upstreams.detail.generatePartial', { count: failures.length }))
    else appStore.showSuccess(t('admin.upstreams.detail.generateDone'))
    selectedUpstream.value = await adminAPI.upstreams.get(selectedUpstream.value.id)
    generationPreview.value = null
    generationSpecs.value = []
    detailTab.value = 'accounts'
    await loadUpstreams()
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    generating.value = false
  }
}

async function loadCandidates() {
  if (!selectedUpstream.value || !showDetail.value) return
  candidatesLoading.value = true
  try {
    const result = await adminAPI.upstreams.listBindCandidates(selectedUpstream.value.id, 1, 100, candidateSearch.value)
    candidates.value = result.items || []
    selectedCandidateIdsArray.value = []
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    candidatesLoading.value = false
  }
}

async function bindSelected() {
  if (!selectedUpstream.value || !selectedCandidateIdsArray.value.length) return
  binding.value = true
  try {
    await adminAPI.upstreams.bindAccounts(selectedUpstream.value.id, selectedCandidateIdsArray.value)
    appStore.showSuccess(t('admin.upstreams.detail.bindDone'))
    selectedUpstream.value = await adminAPI.upstreams.get(selectedUpstream.value.id)
    await loadCandidates()
    await loadUpstreams()
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    binding.value = false
  }
}

async function unbind(accountId: number) {
  if (!selectedUpstream.value) return
  const confirmKey = deleteAccountsOnUnbind.value
    ? 'admin.upstreams.detail.unbindDeleteConfirm'
    : 'admin.upstreams.detail.unbindPreserveConfirm'
  if (!window.confirm(t(confirmKey))) return
  try {
    await adminAPI.upstreams.unbindAccounts(selectedUpstream.value.id, [accountId], deleteAccountsOnUnbind.value)
    selectedUpstream.value = await adminAPI.upstreams.get(selectedUpstream.value.id)
    await loadCandidates()
    await loadUpstreams()
    appStore.showSuccess(deleteAccountsOnUnbind.value ? t('admin.upstreams.detail.unbindDeleted') : t('admin.upstreams.detail.unbindPreserved'))
  } catch (error) {
    appStore.showError(errorMessage(error))
  }
}

function clearAccountGroupSelections() {
  for (const accountID of Object.keys(accountGroupSelections)) delete accountGroupSelections[Number(accountID)]
}

function upstreamGroupOptionValue(group: UpstreamGroupMetadata): string {
  return `${group.id ?? ''}:${group.platform || ''}:${group.name}`
}

function compatibleAccountGroups(account: UpstreamAccountSummary): UpstreamGroupMetadata[] {
  return [...(selectedUpstream.value?.metadata?.groups || [])]
    .filter(group => !group.platform || group.platform.toLowerCase() === account.platform.toLowerCase())
    .sort((left, right) => left.name.localeCompare(right.name, undefined, { sensitivity: 'base' }) || (left.id || 0) - (right.id || 0))
}

function accountGroupOptions(account: UpstreamAccountSummary) {
  const requiresGroupID = selectedUpstream.value?.kind === 'sub2api'
  return compatibleAccountGroups(account).map(group => ({
    value: upstreamGroupOptionValue(group),
    label: group.rate_multiplier != null ? `${group.name} · ${group.rate_multiplier}x` : group.name,
    disabled: group.rate_multiplier == null || (requiresGroupID && group.id == null)
  }))
}

function currentAccountGroup(account: UpstreamAccountSummary): UpstreamGroupMetadata | undefined {
  const groups = compatibleAccountGroups(account)
  if (account.upstream_group_id != null) {
    const byID = groups.find(group => group.id === account.upstream_group_id)
    if (byID) return byID
  }
  const name = account.upstream_group?.trim().toLowerCase()
  return name ? groups.find(group => group.name.trim().toLowerCase() === name) : undefined
}

function accountGroupSelection(account: UpstreamAccountSummary): string | null {
  if (Object.prototype.hasOwnProperty.call(accountGroupSelections, account.id)) {
    return accountGroupSelections[account.id]
  }
  const current = currentAccountGroup(account)
  return current ? upstreamGroupOptionValue(current) : null
}

function setAccountGroupSelection(accountID: number, value: string | number | boolean | null) {
  accountGroupSelections[accountID] = typeof value === 'string' ? value : null
}

function selectedAccountGroup(account: UpstreamAccountSummary): UpstreamGroupMetadata | undefined {
  const selected = accountGroupSelection(account)
  return selected ? compatibleAccountGroups(account).find(group => upstreamGroupOptionValue(group) === selected) : undefined
}

function groupCatalogueIsStale(): boolean {
  const metadata = selectedUpstream.value?.metadata
  return metadata?.management_status !== 'ok' || !metadata.groups?.length
}

function canApplyAccountGroup(account: UpstreamAccountSummary): boolean {
  if (!account.upstream_group_change_supported || accountGroupChanging.has(account.id) || groupCatalogueIsStale()) return false
  const target = selectedAccountGroup(account)
  if (!target || target.rate_multiplier == null) return false
  if (selectedUpstream.value?.kind === 'sub2api' && target.id == null) return false
  if (account.upstream_group_id != null && target.id != null) return account.upstream_group_id !== target.id
  return account.upstream_group?.trim().toLowerCase() !== target.name.trim().toLowerCase()
}

function accountGroupChangeReason(account: UpstreamAccountSummary): string {
  const reasonKeys: Record<string, string> = {
    'upstream management context is unavailable': 'contextUnavailable',
    'only API key accounts can change upstream groups': 'apiKeyOnly',
    'the bound account has no API key': 'missingApiKey',
    'a valid NewAPI management user ID is required': 'missingManagementUserId',
    'NewAPI management credentials are required': 'missingNewAPICredentials',
    'Sub2API panel credentials are required': 'missingSub2APICredentials',
    'probe and identify the upstream type first': 'unknownKind'
  }
  const key = reasonKeys[account.upstream_group_change_reason || ''] || 'unavailable'
  return t(`admin.upstreams.detail.groupChangeReasons.${key}`)
}

async function applyAccountGroupChange(account: UpstreamAccountSummary) {
  if (!selectedUpstream.value || !canApplyAccountGroup(account)) return
  const upstreamID = selectedUpstream.value.id
  const target = selectedAccountGroup(account)
  if (!target) return
  accountGroupChanging.add(account.id)
  try {
    const result = await adminAPI.upstreams.changeAccountUpstreamGroup(upstreamID, account.id, {
      group_name: target.name,
      group_id: target.id
    })
    const index = selectedUpstream.value.accounts?.findIndex(item => item.id === account.id) ?? -1
    if (index >= 0 && selectedUpstream.value.accounts) selectedUpstream.value.accounts[index] = result.account
    delete accountGroupSelections[account.id]
    appStore.showSuccess(t('admin.upstreams.detail.groupChangeDone', { group: target.name }))
    if (result.warning) appStore.showWarning(t('admin.upstreams.detail.groupChangeSnapshotWarning'))
    try {
      selectedUpstream.value = await adminAPI.upstreams.get(upstreamID)
    } catch {
      // The mutation response already contains the committed account state.
    }
    await loadUpstreams(true)
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    accountGroupChanging.delete(account.id)
  }
}

async function openRenamePreview() {
  showRenamePreview.value = true
  renamePreviewLoading.value = true
  renamePreview.value = null
  try {
    renamePreview.value = await adminAPI.upstreams.previewAccountRenames()
  } catch (error) {
    showRenamePreview.value = false
    appStore.showError(errorMessage(error))
  } finally {
    renamePreviewLoading.value = false
  }
}

function closeRenamePreview() {
  if (renameApplying.value) return
  showRenamePreview.value = false
  renamePreview.value = null
}

function renameSkipReason(reason?: string): string {
  const reasonKeys: Record<string, string> = {
    'upstream group is not currently verified': 'groupNotVerified',
    'already uses the automatic name': 'alreadyNamed',
    'local account update failed': 'updateFailed'
  }
  return t(`admin.upstreams.renameDialog.reasons.${reasonKeys[reason || ''] || 'unknown'}`)
}

async function applyRenamePreview() {
  if (renameApplying.value || !renamePreview.value?.renames) return
  renameApplying.value = true
  const selectedUpstreamID = selectedUpstream.value?.id
  try {
    const result = await adminAPI.upstreams.applyAccountRenames()
    if (result.failed > 0) {
      appStore.showWarning(t('admin.upstreams.renameDialog.resultPartial', { renamed: result.renamed, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.upstreams.renameDialog.resultDone', { count: result.renamed }))
    }
    showRenamePreview.value = false
    renamePreview.value = null
    await loadUpstreams()
    if (selectedUpstreamID) {
      try {
        selectedUpstream.value = await adminAPI.upstreams.get(selectedUpstreamID)
      } catch {
        // The main list has already been refreshed; keep the existing dialog state.
      }
    }
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    renameApplying.value = false
  }
}

function statusDotClass(status: string) {
  return status === 'healthy' ? 'bg-emerald-500' : status === 'degraded' ? 'bg-amber-500' : status === 'error' ? 'bg-red-500' : 'bg-gray-400'
}

function statusLabel(status: string) {
  return t(`admin.upstreams.status.${status}`, status)
}

function kindLabel(kind: string) {
  return kind === 'newapi' ? 'NewAPI' : kind === 'sub2api' ? 'Sub2API' : t('admin.upstreams.kind.auto')
}

function protocolLabel(platform: string) {
  return platform === 'openai' ? 'OpenAI' : platform === 'anthropic' ? 'Anthropic' : platform === 'gemini' ? 'Gemini' : platform === 'grok' ? 'Grok' : platform
}

function localGroupNames(item: Upstream): string[] {
  return (item.local_groups || []).map(group => group.name)
}

function walletBalance(item: Upstream): number | undefined {
  return item.metadata?.wallet?.balance
}

function walletUnit(item: Upstream): string {
  return item.metadata?.wallet?.unit || 'USD'
}

function formatMoney(value: number, unit: string) {
  const currency = unit.toUpperCase() === 'CNY' ? 'CNY' : 'USD'
  return new Intl.NumberFormat(undefined, { style: 'currency', currency, maximumFractionDigits: 2 }).format(value)
}

function defaultName(spec: UpstreamAccountGenerationSpec) {
  return `${selectedUpstream.value?.name || ''} / ${spec.upstream_group_name} / ${spec.platform.toUpperCase()}`
}

function formatRate(value: number): string {
  return formatMultiplier(value)
}

function refreshStatusText(item: Upstream): string {
  const refresh = item.metadata?.refresh
  if (!refresh?.last_attempt_at) return t('admin.upstreams.refreshPending')
  const timestamp = new Date(refresh.last_attempt_at)
  const formatted = Number.isFinite(timestamp.getTime()) ? timestamp.toLocaleString() : '-'
  if (refresh.status === 'partial') return t('admin.upstreams.refreshPartial', { time: formatted })
  if (!refresh.stale) return t('admin.upstreams.refreshFresh', { time: formatted })
  return refresh.last_success_at
    ? t('admin.upstreams.refreshStale', { time: formatted })
    : t('admin.upstreams.refreshFailed', { time: formatted })
}

function upstreamFailureReasons(item: Upstream): UpstreamFailureReason[] {
  const reasons: UpstreamFailureReason[] = []
  const managementStatus = item.metadata?.management_status
  if (managementStatus && managementStatus !== 'ok') {
    reasons.push({
      key: `management-${managementStatus}`,
      scope: 'management',
      code: managementStatus,
      message: item.metadata?.management_hint || t('admin.upstreams.failureMessages.missingManagement')
    })
  }

  if (item.status !== 'healthy') {
    for (const protocol of item.metadata?.protocols || []) {
      if (protocol.status === 'ok' || protocol.status === 'missing_api_key') continue
      reasons.push({
        key: `protocol-${protocol.platform}-${protocol.status}`,
        scope: 'protocol',
        code: protocol.status || 'unknown',
        platform: protocol.platform,
        message: protocol.message || failureFallbackMessage(protocol.status)
      })
    }
  }

  const billingEntries = Object.values(item.metadata?.account_billing || {})
    .filter((billing): billing is UpstreamAccountBillingMetadata => billing.status !== 'unsupported')
    .sort((left, right) => left.account_id - right.account_id)
  for (const billing of billingEntries) {
    const hasRate = billing.group_effective_rate_multiplier != null || billing.group_default_rate_multiplier != null
    if (!billing.stale && billing.status === 'ok' && hasRate) continue
    let code = billing.status || 'unknown'
    if (code === 'ok' && !hasRate) code = 'rate_unavailable'
    reasons.push({
      key: `account-${billing.account_id}-${code}`,
      scope: 'account',
      code,
      accountId: billing.account_id,
      message: billing.message || failureFallbackMessage(code)
    })
  }

  if (reasons.length === 0 && item.last_probe_error && (item.status === 'error' || item.status === 'degraded')) {
    reasons.push({
      key: 'general-probe-error',
      scope: 'general',
      code: 'error',
      message: item.last_probe_error
    })
  }
  return reasons
}

function failureFallbackMessage(code: string): string {
  if (code === 'missing_management_user_id') return t('admin.upstreams.failureMessages.missingManagementUserId')
  if (code === 'missing_login') return t('admin.upstreams.failureMessages.missingLogin')
  if (code === 'cloudflare_blocked') return t('admin.upstreams.failureMessages.cloudflareBlocked')
  if (code === 'key_not_found') return t('admin.upstreams.failureMessages.keyNotFound')
  if (code === 'rate_unavailable') return t('admin.upstreams.failureMessages.rateUnavailable')
  return t('admin.upstreams.failureCodes.unknown')
}

function failureScopeLabel(item: Upstream, reason: UpstreamFailureReason): string {
  if (reason.scope === 'management') return t('admin.upstreams.failureScopeManagement')
  if (reason.scope === 'protocol') return t('admin.upstreams.failureScopeProtocol', { platform: protocolLabel(reason.platform || '') })
  if (reason.scope === 'general') return t('admin.upstreams.failureScopeGeneral')
  const account = item.accounts?.find(candidate => candidate.id === reason.accountId)
  const label = t('admin.upstreams.failureScopeAccount', { id: reason.accountId || '-' })
  return account?.name ? `${label} · ${account.name}` : label
}

function failureCodeLabel(code: string): string {
  const knownCodes = new Set([
    'missing_management_credentials',
    'missing_management_user_id',
    'missing_login',
    'cloudflare_blocked',
    'public_api_unavailable',
    'request_failed',
    'key_not_found',
    'rate_unavailable',
    'error',
    'unavailable',
    'unknown'
  ])
  return knownCodes.has(code) ? t(`admin.upstreams.failureCodes.${code}`) : code
}

function failureReasonMessage(reason: UpstreamFailureReason): string {
  if (reason.code === 'missing_management_credentials') return t('admin.upstreams.failureMessages.missingManagement')
  if (reason.code === 'missing_management_user_id') return t('admin.upstreams.failureMessages.missingManagementUserId')
  if (reason.code === 'missing_login') return t('admin.upstreams.failureMessages.missingLogin')
  if (reason.code === 'cloudflare_blocked') return t('admin.upstreams.failureMessages.cloudflareBlocked')
  if (reason.code === 'key_not_found') return t('admin.upstreams.failureMessages.keyNotFound')
  if (reason.code === 'rate_unavailable') return t('admin.upstreams.failureMessages.rateUnavailable')
  return reason.message
}

function failureSummaryText(item: Upstream): string {
  const reasons = upstreamFailureReasons(item)
  if (reasons.length === 0) return ''
  const first = reasons[0]
  const summary = `${failureScopeLabel(item, first)} · ${failureCodeLabel(first.code)}：${failureReasonMessage(first)}`
  if (reasons.length === 1) return summary
  return `${summary} ${t('admin.upstreams.failureSummaryMore', { count: reasons.length - 1 })}`
}

function failureReasonsTitle(item: Upstream): string {
  return upstreamFailureReasons(item)
    .map(reason => `${failureScopeLabel(item, reason)} · ${failureCodeLabel(reason.code)}：${failureReasonMessage(reason)}`)
    .join('\n')
}

function failureTextClass(item: Upstream): string {
  return item.metadata?.refresh?.status === 'partial' ? 'text-amber-700 dark:text-amber-300' : 'text-red-700 dark:text-red-300'
}

function failurePanelClass(item: Upstream): string {
  return item.metadata?.refresh?.status === 'partial'
    ? 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200'
    : 'border-red-200 bg-red-50 text-red-900 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200'
}

onMounted(() => {
  void Promise.all([loadUpstreams(), loadProxies()])
  if (upstreamAutoRefresh.enabled.value) {
    upstreamAutoRefresh.resetCountdown()
    upstreamAutoRefresh.start()
  }
})
</script>

<style scoped>
.upstream-model-list {
  -webkit-overflow-scrolling: touch;
  overscroll-behavior: contain;
  touch-action: pan-y;
}

.icon-button {
  display: inline-flex;
  height: 2rem;
  width: 2rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.375rem;
  color: var(--apple-muted);
  transition: background-color 0.15s ease, color 0.15s ease;
}

.icon-button:hover:not(:disabled) {
  background: var(--apple-hover);
  color: var(--apple-text);
}

.icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

@media (max-width: 639px) and (pointer: coarse) {
  .upstream-model-list .icon-button {
    height: 2.75rem;
    width: 2.75rem;
  }
}
</style>
