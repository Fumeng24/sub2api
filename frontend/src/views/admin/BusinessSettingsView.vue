<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <header class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ localText('业务设置', 'Business Settings') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ localText('管理工单规则与分组折扣。', 'Manage ticket rules and group discounts.') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <router-link to="/admin/settings" class="btn btn-secondary">
            <Icon name="cog" size="sm" />
            {{ localText('系统设置', 'System settings') }}
          </router-link>
          <button type="button" class="btn btn-secondary" :disabled="loading || saving" @click="loadSettings">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </div>
      </header>

      <div class="border-b border-gray-200 dark:border-dark-700">
        <div class="flex gap-6" role="tablist" :aria-label="localText('业务设置', 'Business settings')">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            role="tab"
            class="relative flex h-11 items-center gap-2 px-1 text-sm font-medium transition-colors"
            :class="activeTab === tab.key
              ? 'text-primary-700 dark:text-primary-300'
              : 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
            :aria-selected="activeTab === tab.key"
            @click="activeTab = tab.key"
          >
            <Icon :name="tab.icon" size="sm" />
            {{ tab.label }}
            <span
              v-if="activeTab === tab.key"
              class="absolute inset-x-0 bottom-0 h-0.5 rounded-full bg-primary-600"
            ></span>
          </button>
        </div>
      </div>

      <div v-if="loading" class="flex min-h-[360px] items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-primary-600 dark:border-dark-600 dark:border-t-primary-400"></div>
      </div>

      <form v-else class="space-y-6" @submit.prevent="saveSettings">
        <template v-if="activeTab === 'tickets'">
          <section class="card">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ localText('客服权限', 'Support permissions') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ localText('超级管理员始终拥有全部权限。', 'Super admins always retain full access.') }}
              </p>
            </div>
            <div class="grid grid-cols-1 gap-px bg-gray-100 dark:bg-dark-700 md:grid-cols-2 xl:grid-cols-3">
              <label
                v-for="permission in ticketPermissionOptions"
                :key="permission.key"
                class="flex min-h-[92px] items-start justify-between gap-4 bg-white px-5 py-4 dark:bg-dark-800"
              >
                <span class="min-w-0">
                  <span class="block text-sm font-medium text-gray-800 dark:text-gray-100">{{ permission.label }}</span>
                  <span class="mt-1 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ permission.hint }}</span>
                </span>
                <Toggle v-model="form.ticketSystem.support_permissions[permission.key]" />
              </label>
            </div>
          </section>

          <section class="card">
            <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between sm:px-6">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">SLA</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ localText('首次响应、催办、升级与自动关闭。', 'First response, reminders, escalation, and auto-close.') }}
                </p>
              </div>
              <Toggle v-model="form.ticketSystem.sla.enabled" />
            </div>
            <div class="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2 xl:grid-cols-4 sm:p-6">
              <div>
                <label class="input-label">{{ localText('首次响应分钟', 'First response minutes') }}</label>
                <input v-model.number="form.ticketSystem.sla.first_response_minutes" type="number" min="1" max="43200" class="input" />
              </div>
              <div>
                <label class="input-label">{{ localText('提前催办分钟', 'Reminder minutes before due') }}</label>
                <input v-model.number="form.ticketSystem.sla.reminder_before_minutes" type="number" min="0" max="43200" class="input" />
              </div>
              <div>
                <label class="input-label">{{ localText('超时后升级分钟', 'Escalate minutes after due') }}</label>
                <input v-model.number="form.ticketSystem.sla.auto_escalate_after_minutes" type="number" min="0" max="43200" class="input" />
              </div>
              <div>
                <label class="input-label">{{ localText('扫描间隔秒', 'Worker interval seconds') }}</label>
                <input v-model.number="form.ticketSystem.sla.worker_interval_seconds" type="number" min="30" max="86400" class="input" />
              </div>
              <div>
                <label class="input-label">{{ localText('自动关闭已解决天数', 'Auto-close resolved days') }}</label>
                <input v-model.number="form.ticketSystem.sla.auto_close_resolved_days" type="number" min="0" max="365" class="input" />
              </div>
              <label class="flex min-h-11 items-center justify-between gap-3 rounded-lg border border-gray-200 px-4 dark:border-dark-600">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ localText('发送催办通知', 'Send reminder notifications') }}</span>
                <Toggle v-model="form.ticketSystem.sla.reminder_notifications" />
              </label>
              <label class="flex min-h-11 items-center justify-between gap-3 rounded-lg border border-gray-200 px-4 dark:border-dark-600">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ localText('升级时通知超管', 'Notify super admins on escalation') }}</span>
                <Toggle v-model="form.ticketSystem.sla.auto_escalate_notifications" />
              </label>
            </div>
          </section>

          <section class="space-y-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ localText('工单类型', 'Ticket types') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ localText('配置分类、优先级、路由和用户必填字段。', 'Configure category, priority, routing, and required fields.') }}
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button type="button" class="btn btn-secondary" @click="resetTicketSettings">
                  <Icon name="refresh" size="sm" />
                  {{ localText('恢复默认', 'Reset defaults') }}
                </button>
                <button type="button" class="btn btn-primary" @click="addTicketTemplate">
                  <Icon name="plus" size="sm" />
                  {{ localText('新增类型', 'Add type') }}
                </button>
              </div>
            </div>

            <article
              v-for="(template, templateIndex) in form.ticketSystem.templates"
              :key="`${template.key}-${templateIndex}`"
              class="card overflow-hidden"
            >
              <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
                <div class="min-w-0">
                  <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                    {{ template.name || localText('未命名类型', 'Unnamed type') }}
                  </h3>
                  <p class="mt-0.5 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ template.key }}</p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="duplicateTicketTemplate(templateIndex)">
                    <Icon name="copy" size="sm" />
                    {{ localText('复制', 'Copy') }}
                  </button>
                  <button type="button" class="btn btn-danger btn-sm" @click="removeTicketTemplate(templateIndex)">
                    <Icon name="trash" size="sm" />
                    {{ t('common.delete') }}
                  </button>
                </div>
              </div>

              <div class="space-y-5 p-5 sm:p-6">
                <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                  <div>
                    <label class="input-label">{{ localText('标识', 'Key') }}</label>
                    <input v-model="template.key" type="text" class="input font-mono" placeholder="group_connection_issue" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('名称', 'Name') }}</label>
                    <input v-model="template.name" type="text" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('分类', 'Category') }}</label>
                    <Select v-model="template.category" :options="ticketCategoryOptions" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('优先级', 'Priority') }}</label>
                    <Select v-model="template.priority" :options="ticketPriorityOptions" />
                  </div>
                  <div class="md:col-span-2">
                    <label class="input-label">{{ localText('说明', 'Description') }}</label>
                    <input v-model="template.description" type="text" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('默认标题', 'Subject template') }}</label>
                    <input v-model="template.subject_template" type="text" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('最少描述字数', 'Body min length') }}</label>
                    <input v-model.number="template.body_min_length" type="number" min="0" max="2000" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('上下文类型', 'Context type') }}</label>
                    <input v-model="template.context_type" type="text" class="input" placeholder="group / order / api_key" />
                  </div>
                  <label class="flex min-h-11 items-center justify-between gap-3 rounded-lg border border-gray-200 px-4 dark:border-dark-600">
                    <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ localText('需要超级管理员', 'Requires super admin') }}</span>
                    <Toggle v-model="template.requires_super_admin" />
                  </label>
                  <label class="flex min-h-11 items-center justify-between gap-3 rounded-lg border border-gray-200 px-4 dark:border-dark-600">
                    <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ localText('自动分配给超管', 'Auto-assign super admin') }}</span>
                    <Toggle v-model="template.auto_assign_super_admin" />
                  </label>
                </div>

                <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ localText('字段要求', 'Field requirements') }}</h4>
                    <button type="button" class="btn btn-secondary btn-sm" @click="addTicketTemplateField(template)">
                      <Icon name="plus" size="sm" />
                      {{ localText('新增字段', 'Add field') }}
                    </button>
                  </div>

                  <p v-if="!template.fields?.length" class="rounded-lg border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
                    {{ localText('没有额外字段。', 'No additional fields.') }}
                  </p>

                  <div v-else class="space-y-3">
                    <div
                      v-for="(field, fieldIndex) in template.fields"
                      :key="`${field.key}-${fieldIndex}`"
                      class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700/40"
                    >
                      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
                        <div>
                          <label class="input-label">{{ localText('字段标识', 'Field key') }}</label>
                          <input v-model="field.key" type="text" class="input font-mono" />
                        </div>
                        <div>
                          <label class="input-label">{{ localText('字段名称', 'Field label') }}</label>
                          <input v-model="field.label" type="text" class="input" />
                        </div>
                        <div>
                          <label class="input-label">{{ localText('类型', 'Type') }}</label>
                          <Select v-model="field.type" :options="ticketFieldTypeOptions" />
                        </div>
                        <div>
                          <label class="input-label">{{ localText('最小长度', 'Min length') }}</label>
                          <input v-model.number="field.min_length" type="number" min="0" class="input" />
                        </div>
                        <div>
                          <label class="input-label">{{ localText('最大长度', 'Max length') }}</label>
                          <input v-model.number="field.max_length" type="number" min="0" class="input" />
                        </div>
                        <div v-if="field.type === 'amount'">
                          <label class="input-label">{{ localText('最小金额', 'Minimum amount') }}</label>
                          <input v-model.number="field.min_value" type="number" min="0" step="0.01" class="input" />
                        </div>
                        <div class="md:col-span-2">
                          <label class="input-label">{{ localText('占位提示', 'Placeholder') }}</label>
                          <input v-model="field.placeholder" type="text" class="input" />
                        </div>
                        <div class="md:col-span-2 xl:col-span-3">
                          <label class="input-label">{{ localText('说明', 'Description') }}</label>
                          <input v-model="field.description" type="text" class="input" />
                        </div>
                        <div class="flex items-end justify-between gap-3">
                          <label class="flex h-10 items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                            <input v-model="field.required" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                            {{ localText('必填', 'Required') }}
                          </label>
                          <button type="button" class="btn btn-ghost btn-sm px-2 text-red-600" :title="t('common.delete')" @click="removeTicketTemplateField(template, fieldIndex)">
                            <Icon name="trash" size="sm" />
                          </button>
                        </div>
                      </div>

                      <div v-if="field.type === 'select'" class="mt-4 space-y-2 border-t border-gray-200 pt-3 dark:border-dark-600">
                        <div class="flex items-center justify-between gap-3">
                          <span class="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">{{ localText('下拉选项', 'Select options') }}</span>
                          <button type="button" class="btn btn-secondary btn-sm" @click="addTicketFieldOption(field)">
                            <Icon name="plus" size="sm" />
                            {{ localText('新增选项', 'Add option') }}
                          </button>
                        </div>
                        <div
                          v-for="(option, optionIndex) in field.options || []"
                          :key="optionIndex"
                          class="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
                        >
                          <input v-model="option.value" type="text" class="input font-mono" placeholder="value" />
                          <input v-model="option.label" type="text" class="input" :placeholder="localText('显示名称', 'Label')" />
                          <button type="button" class="btn btn-ghost px-2 text-red-600" :title="t('common.remove')" @click="removeTicketFieldOption(field, optionIndex)">
                            <Icon name="x" size="sm" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </article>
          </section>
        </template>

        <template v-else>
          <section class="card">
            <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between sm:px-6">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ localText('分组折扣', 'Group discount') }}</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ localText('为指定分组设置统一折扣倍率和生效时间。', 'Set a shared discount multiplier and schedule for selected groups.') }}</p>
              </div>
              <Toggle v-model="form.groupDiscount.enabled" />
            </div>

            <div class="space-y-6 p-5 sm:p-6">
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div>
                  <label class="input-label">{{ localText('活动名称', 'Campaign name') }}</label>
                  <input v-model="form.groupDiscount.name" type="text" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ localText('折扣倍率', 'Discount multiplier') }}</label>
                  <input v-model.number="form.groupDiscount.discount_multiplier" type="number" min="0.01" max="0.99" step="0.01" class="input" />
                  <p class="input-hint">{{ localText('0.8 表示按原倍率的 80% 计费。', '0.8 bills at 80% of the original rate.') }}</p>
                </div>
              </div>

              <div>
                <label class="input-label">{{ localText('计划类型', 'Schedule type') }}</label>
                <div class="inline-flex w-full rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-600 dark:bg-dark-700 sm:w-auto">
                  <button
                    v-for="mode in scheduleModes"
                    :key="mode.value"
                    type="button"
                    class="min-w-0 flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors sm:flex-none"
                    :class="form.groupDiscount.schedule_mode === mode.value
                      ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                      : 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
                    @click="form.groupDiscount.schedule_mode = mode.value"
                  >
                    {{ mode.label }}
                  </button>
                </div>
              </div>

              <template v-if="form.groupDiscount.schedule_mode === 'weekly'">
                <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div>
                    <label class="input-label">{{ localText('每日开始时间', 'Daily start time') }}</label>
                    <input v-model="form.groupDiscount.daily_start_time" type="time" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ localText('每日结束时间', 'Daily end time') }}</label>
                    <input v-model="form.groupDiscount.daily_end_time" type="time" class="input" />
                    <p class="input-hint">{{ localText('结束时间早于开始时间时跨到次日。', 'An earlier end time continues into the next day.') }}</p>
                  </div>
                </div>
                <div>
                  <div class="mb-2 flex items-center justify-between gap-3">
                    <label class="input-label mb-0">{{ localText('生效日期', 'Active weekdays') }}</label>
                    <button type="button" class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="selectAllDiscountWeekdays">
                      {{ localText('每天', 'Every day') }}
                    </button>
                  </div>
                  <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7">
                    <label
                      v-for="day in weekdayOptions"
                      :key="day.value"
                      class="flex h-10 cursor-pointer items-center justify-center gap-2 rounded-lg border px-3 text-sm font-medium transition-colors"
                      :class="isDiscountWeekdaySelected(day.value)
                        ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
                        : 'border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700'"
                    >
                      <input
                        type="checkbox"
                        class="sr-only"
                        :checked="isDiscountWeekdaySelected(day.value)"
                        @change="toggleDiscountWeekday(day.value, ($event.target as HTMLInputElement).checked)"
                      />
                      {{ day.label }}
                    </label>
                  </div>
                </div>
              </template>

              <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div>
                  <label class="input-label">{{ localText('开始时间', 'Start time') }}</label>
                  <input v-model="form.groupDiscount.start_at" type="datetime-local" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ localText('结束时间', 'End time') }}</label>
                  <input v-model="form.groupDiscount.end_at" type="datetime-local" class="input" />
                </div>
              </div>

              <div>
                <div class="mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                  <label class="input-label mb-0">{{ localText('参与分组', 'Discount groups') }}</label>
                  <div class="flex items-center gap-3">
                    <span class="text-xs text-gray-500 dark:text-gray-400">{{ localText('已选', 'Selected') }} {{ form.groupDiscount.group_ids.length }}</span>
                    <label class="relative block w-48">
                      <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                      <input v-model.trim="groupSearch" type="search" class="input h-9 pl-9" :placeholder="t('common.search')" />
                    </label>
                  </div>
                </div>
                <div v-if="visibleGroups.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
                  {{ localText('暂无匹配分组。', 'No matching groups.') }}
                </div>
                <div v-else class="grid max-h-80 grid-cols-1 gap-2 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-700/40 md:grid-cols-2">
                  <label
                    v-for="group in visibleGroups"
                    :key="group.id"
                    class="flex min-h-11 cursor-pointer items-center gap-3 rounded-lg bg-white px-3 py-2 hover:bg-gray-100 dark:bg-dark-800 dark:hover:bg-dark-700"
                  >
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="isDiscountGroupSelected(group.id)"
                      @change="toggleDiscountGroup(group.id, ($event.target as HTMLInputElement).checked)"
                    />
                    <GroupBadge
                      :name="group.name"
                      :platform="group.platform"
                      :subscription-type="group.subscription_type"
                      :group-id="group.id"
                      :rate-multiplier="group.rate_multiplier"
                      class="min-w-0"
                    />
                  </label>
                </div>
              </div>

              <div v-if="discountPreview.length" class="rounded-lg border border-primary-200 bg-primary-50 p-4 dark:border-primary-500/30 dark:bg-primary-500/10">
                <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                  <div class="text-sm font-semibold text-primary-800 dark:text-primary-200">
                    {{ form.groupDiscount.name || localText('限时折扣', 'Limited-time discount') }}
                    <span class="ml-1 rounded bg-white/80 px-1.5 py-0.5 text-xs dark:bg-dark-800/80">{{ discountPercentLabel }}</span>
                  </div>
                  <div class="text-xs text-primary-700 dark:text-primary-300">{{ discountWindowLabel }}</div>
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <span v-for="item in discountPreview" :key="item.group.id" class="inline-flex items-center gap-1.5 rounded-md bg-white px-2 py-1 text-xs font-medium text-gray-800 shadow-sm dark:bg-dark-800 dark:text-gray-200">
                    {{ item.group.name }}
                    <span class="text-gray-400 line-through">{{ formatRateMultiplier(item.group.rate_multiplier) }}x</span>
                    <span class="font-bold text-primary-700 dark:text-primary-300">{{ formatRateMultiplier(item.discountedRate) }}x</span>
                  </span>
                </div>
              </div>
            </div>
          </section>
        </template>

        <div class="flex justify-end border-t border-gray-200 pt-4 dark:border-dark-700">
          <button type="submit" class="btn btn-primary min-w-32" :disabled="saving">
            <span v-if="saving" class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"></span>
            <Icon v-else name="check" size="sm" />
            {{ saving ? localText('保存中...', 'Saving...') : t('common.save') }}
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import AppLayout from '@/components/layout/AppLayout.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/custom/api/admin'
import type { GroupRateDiscountSettings } from '@/custom/api/admin/settings'
import { formatDiscountLabel, formatRateMultiplier, roundRateMultiplier } from '@/custom/utils/groupRateDiscount'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type {
  AdminGroup,
  TicketCategory,
  TicketPriority,
  TicketSupportPermissions,
  TicketSystemSettings,
  TicketTemplate,
  TicketTemplateField,
  TicketTemplateFieldType,
} from '@/types'

type BusinessTab = 'tickets' | 'discount'

interface BusinessSettingsForm {
  ticketSystem: TicketSystemSettings
  groupDiscount: GroupRateDiscountSettings
}

const { t, locale } = useI18n()
const appStore = useAppStore()
const activeTab = ref<BusinessTab>('tickets')
const loading = ref(true)
const saving = ref(false)
const groups = ref<AdminGroup[]>([])
const groupSearch = ref('')

const form = reactive<BusinessSettingsForm>({
  ticketSystem: defaultTicketSystemSettings(),
  groupDiscount: defaultGroupRateDiscountSettings(),
})

const isChinese = computed(() => locale.value.startsWith('zh'))

function localText(zh: string, en: string): string {
  return isChinese.value ? zh : en
}

const tabs = computed(() => [
  { key: 'tickets' as const, icon: 'chatBubble' as const, label: localText('工单规则', 'Ticket rules') },
  { key: 'discount' as const, icon: 'calculator' as const, label: localText('分组折扣', 'Group discounts') },
])

const scheduleModes = computed(() => [
  { value: 'weekly', label: localText('每周重复', 'Weekly') },
  { value: 'once', label: localText('单次时段', 'One-time') },
])

const ticketCategoryOptions = computed(() => [
  { value: 'general', label: localText('通用', 'General') },
  { value: 'billing', label: localText('充值账务', 'Billing') },
  { value: 'usage', label: localText('使用问题', 'Usage') },
  { value: 'technical', label: localText('技术故障', 'Technical') },
  { value: 'account', label: localText('账号问题', 'Account') },
])

const ticketPriorityOptions = computed(() => [
  { value: 'low', label: localText('低', 'Low') },
  { value: 'normal', label: localText('普通', 'Normal') },
  { value: 'high', label: localText('高', 'High') },
  { value: 'urgent', label: localText('紧急', 'Urgent') },
])

const ticketFieldTypeOptions = computed(() => [
  { value: 'text', label: localText('短文本', 'Text') },
  { value: 'textarea', label: localText('长文本', 'Textarea') },
  { value: 'select', label: localText('下拉选项', 'Select') },
  { value: 'group_select', label: localText('用户分组选择', 'Group select') },
  { value: 'recent_orders', label: localText('最近充值记录', 'Recent orders') },
  { value: 'amount', label: localText('金额', 'Amount') },
  { value: 'image', label: localText('图片', 'Image') },
  { value: 'attachments', label: localText('附件', 'Attachments') },
])

const ticketPermissionOptions = computed<Array<{
  key: keyof TicketSupportPermissions
  label: string
  hint: string
}>>(() => [
  { key: 'can_view_all', label: localText('查看全部工单', 'View all tickets'), hint: localText('否则仅查看未分配和自己的工单。', 'Otherwise agents see unassigned and their own tickets.') },
  { key: 'can_view_escalated', label: localText('查看已升级工单', 'View escalated tickets'), hint: localText('升级后的处理仍由超级管理员完成。', 'Escalated tickets remain owned by super admins.') },
  { key: 'can_internal_note', label: localText('写内部备注', 'Write internal notes'), hint: localText('内部备注不会展示给用户。', 'Internal notes are hidden from users.') },
  { key: 'can_close', label: localText('解决或关闭', 'Resolve or close'), hint: localText('允许处理普通工单状态。', 'Allows status changes on normal tickets.') },
  { key: 'can_transfer', label: localText('转派或认领', 'Transfer or claim'), hint: localText('允许修改处理人。', 'Allows changing the assignee.') },
  { key: 'can_batch_update', label: localText('批量处理', 'Batch update'), hint: localText('允许在列表中批量修改。', 'Allows bulk changes in the ticket list.') },
  { key: 'can_update_priority', label: localText('修改优先级', 'Update priority'), hint: localText('允许调整普通工单优先级。', 'Allows priority changes on normal tickets.') },
  { key: 'can_update_category', label: localText('修改分类', 'Update category'), hint: localText('允许调整普通工单分类。', 'Allows category changes on normal tickets.') },
  { key: 'can_reply_unassigned', label: localText('回复未分配工单', 'Reply to unassigned'), hint: localText('关闭后需先认领。', 'When disabled, an agent must claim first.') },
  { key: 'can_reply_assigned_to_self', label: localText('回复自己的工单', 'Reply to own tickets'), hint: localText('允许回复分配给自己的工单。', 'Allows replies to tickets assigned to the agent.') },
  { key: 'can_escalate', label: localText('升级给超级管理员', 'Escalate to super admin'), hint: localText('升级时需要填写原因。', 'An escalation reason is required.') },
])

const weekdayOptions = computed(() => {
  const labels = isChinese.value
    ? ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
    : ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
  return labels.map((label, index) => ({ label, value: index + 1 }))
})

const visibleGroups = computed(() => {
  const query = groupSearch.value.toLocaleLowerCase()
  return groups.value.filter((group) => {
    if (group.status !== 'active') return false
    if (!query) return true
    return `${group.name}\n${group.platform}`.toLocaleLowerCase().includes(query)
  })
})

const selectedGroups = computed(() => {
  const ids = new Set(form.groupDiscount.group_ids)
  return groups.value.filter((group) => ids.has(group.id))
})

const discountPreview = computed(() => {
  const multiplier = Number(form.groupDiscount.discount_multiplier)
  if (!Number.isFinite(multiplier) || multiplier <= 0) return []
  return selectedGroups.value.map((group) => ({
    group,
    discountedRate: roundRateMultiplier(group.rate_multiplier * multiplier),
  }))
})

const discountPercentLabel = computed(() => formatDiscountLabel(Number(form.groupDiscount.discount_multiplier)))

const discountWindowLabel = computed(() => {
  if (form.groupDiscount.schedule_mode === 'once') {
    if (!form.groupDiscount.start_at || !form.groupDiscount.end_at) return ''
    return `${formatLocalDateTime(form.groupDiscount.start_at)} - ${formatLocalDateTime(form.groupDiscount.end_at)}`
  }
  const days = formatWeekdays(form.groupDiscount.weekdays)
  if (!days) return ''
  return `${days} ${form.groupDiscount.daily_start_time}-${form.groupDiscount.daily_end_time}`
})

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function defaultGroupRateDiscountSettings(): GroupRateDiscountSettings {
  return {
    enabled: false,
    name: '限时折扣',
    discount_multiplier: 1,
    schedule_mode: 'weekly',
    start_at: '',
    end_at: '',
    weekdays: [1, 2, 3, 4, 5, 6, 7],
    daily_start_time: '00:00',
    daily_end_time: '23:59',
    group_ids: [],
  }
}

function defaultTicketSupportPermissions(): TicketSupportPermissions {
  return {
    can_view_all: false,
    can_view_escalated: false,
    can_internal_note: true,
    can_close: true,
    can_transfer: false,
    can_batch_update: false,
    can_update_priority: false,
    can_update_category: false,
    can_reply_unassigned: false,
    can_reply_assigned_to_self: true,
    can_escalate: true,
  }
}

function defaultTicketSystemSettings(): TicketSystemSettings {
  return {
    templates: [
      { key: 'general', name: '其他问题', description: '没有匹配分类时使用', category: 'general', priority: 'normal', subject_template: '其他问题', body_min_length: 10, requires_super_admin: false, auto_assign_super_admin: false, context_type: 'general', fields: [] },
      { key: 'group_connection_issue', name: '分组连接不上', description: '选择正在使用的分组并提供报错截图', category: 'technical', priority: 'high', subject_template: '分组连接问题', body_min_length: 15, requires_super_admin: false, auto_assign_super_admin: false, context_type: 'group', fields: [
        { key: 'group_id', label: '正在使用的分组', type: 'group_select', required: true },
        { key: 'error_screenshot', label: '报错截图', type: 'image', required: true },
      ] },
      { key: 'billing_missing_payment', name: '充值未到账', description: '选择充值记录并由超级管理员处理', category: 'billing', priority: 'urgent', subject_template: '充值未到账', body_min_length: 15, requires_super_admin: true, auto_assign_super_admin: true, context_type: 'order', fields: [
        { key: 'recent_order_ids', label: '最近充值记录', type: 'recent_orders', required: true },
        { key: 'missing_amount', label: '未到账金额', type: 'amount', required: true, min_value: 0 },
        { key: 'payment_screenshot', label: '支付截图', type: 'image', required: true },
      ] },
      { key: 'api_key_issue', name: 'API Key 有问题', description: '提供 Key 和错误信息', category: 'usage', priority: 'normal', subject_template: 'API Key 使用问题', body_min_length: 15, requires_super_admin: false, auto_assign_super_admin: false, context_type: 'api_key', fields: [
        { key: 'api_key_id', label: 'API Key ID', type: 'text', required: false },
        { key: 'error_message', label: '错误信息', type: 'textarea', required: false, min_length: 5 },
      ] },
    ],
    support_permissions: defaultTicketSupportPermissions(),
    sla: {
      enabled: true,
      first_response_minutes: 1440,
      reminder_before_minutes: 60,
      auto_escalate_after_minutes: 0,
      reminder_notifications: true,
      auto_escalate_notifications: true,
      auto_close_resolved_days: 0,
      worker_interval_seconds: 300,
    },
  }
}

function normalizeGroupDiscount(source?: Partial<GroupRateDiscountSettings> | null): GroupRateDiscountSettings {
  const defaults = defaultGroupRateDiscountSettings()
  const mode = source?.schedule_mode === 'once' ? 'once' : 'weekly'
  return {
    ...defaults,
    ...(source || {}),
    schedule_mode: mode,
    start_at: mode === 'once' ? toDateTimeLocal(source?.start_at || '') : '',
    end_at: mode === 'once' ? toDateTimeLocal(source?.end_at || '') : '',
    weekdays: normalizeWeekdays(source?.weekdays, defaults.weekdays),
    daily_start_time: normalizeTime(source?.daily_start_time, defaults.daily_start_time),
    daily_end_time: normalizeTime(source?.daily_end_time, defaults.daily_end_time),
    group_ids: Array.from(new Set((source?.group_ids || []).map(Number).filter((id) => Number.isInteger(id) && id > 0))).sort((a, b) => a - b),
  }
}

function normalizeTicketSystem(source?: Partial<TicketSystemSettings> | null): TicketSystemSettings {
  const defaults = defaultTicketSystemSettings()
  const templates = Array.isArray(source?.templates)
    ? source.templates.map((item, index) => normalizeTicketTemplate(item, index))
    : defaults.templates
  return {
    templates: templates.length ? templates : defaults.templates,
    support_permissions: { ...defaults.support_permissions, ...(source?.support_permissions || {}) },
    sla: { ...defaults.sla, ...(source?.sla || {}) },
  }
}

function normalizeTicketTemplate(source: Partial<TicketTemplate>, index: number): TicketTemplate {
  const fallback = defaultTicketSystemSettings().templates[index]
  return {
    key: normalizeKey(source.key || fallback?.key || '', `template_${index + 1}`),
    name: String(source.name || fallback?.name || `Template ${index + 1}`).trim(),
    description: String(source.description || '').trim(),
    category: normalizeCategory(source.category),
    priority: normalizePriority(source.priority),
    subject_template: String(source.subject_template || '').trim(),
    body_min_length: Math.max(0, Math.floor(Number(source.body_min_length) || 0)),
    requires_super_admin: Boolean(source.requires_super_admin),
    auto_assign_super_admin: Boolean(source.auto_assign_super_admin),
    context_type: String(source.context_type || '').trim(),
    fields: Array.isArray(source.fields)
      ? source.fields.map((field, fieldIndex) => normalizeTicketField(field, fieldIndex))
      : [],
  }
}

function normalizeTicketField(source: Partial<TicketTemplateField>, index: number): TicketTemplateField {
  return {
    key: normalizeKey(source.key || '', `field_${index + 1}`),
    label: String(source.label || `Field ${index + 1}`).trim(),
    type: normalizeFieldType(source.type),
    required: Boolean(source.required),
    min_length: Math.max(0, Math.floor(Number(source.min_length) || 0)),
    max_length: Math.max(0, Math.floor(Number(source.max_length) || 0)),
    min_value: source.min_value == null ? undefined : Number(source.min_value),
    options: Array.isArray(source.options)
      ? source.options.map((option) => ({ value: String(option.value || '').trim(), label: String(option.label || '').trim() })).filter((option) => option.value && option.label)
      : [],
    description: String(source.description || '').trim(),
    placeholder: String(source.placeholder || '').trim(),
  }
}

function normalizeKey(value: string, fallback: string): string {
  return String(value || '').trim().toLowerCase().replace(/[^a-z0-9_-]+/g, '_').replace(/_{2,}/g, '_').replace(/^_+|_+$/g, '') || fallback
}

function normalizeCategory(value: unknown): TicketCategory {
  const allowed: TicketCategory[] = ['general', 'billing', 'usage', 'technical', 'account']
  return allowed.includes(value as TicketCategory) ? value as TicketCategory : 'general'
}

function normalizePriority(value: unknown): TicketPriority {
  const allowed: TicketPriority[] = ['low', 'normal', 'high', 'urgent']
  return allowed.includes(value as TicketPriority) ? value as TicketPriority : 'normal'
}

function normalizeFieldType(value: unknown): TicketTemplateFieldType {
  const allowed: TicketTemplateFieldType[] = ['text', 'textarea', 'select', 'group_select', 'recent_orders', 'amount', 'image', 'attachments']
  return allowed.includes(value as TicketTemplateFieldType) ? value as TicketTemplateFieldType : 'text'
}

function normalizeWeekdays(value: unknown, fallback: number[] = []): number[] {
  const source = Array.isArray(value) ? value : fallback
  return Array.from(new Set(source.map(Number).map(Math.floor).filter((day) => day >= 1 && day <= 7))).sort((a, b) => a - b)
}

function normalizeTime(value: unknown, fallback: string): string {
  const raw = String(value || '')
  return /^([01]\d|2[0-3]):[0-5]\d$/.test(raw) ? raw : fallback
}

function toDateTimeLocal(value: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value.slice(0, 16)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function toRFC3339(value: string): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toISOString()
}

function formatLocalDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(isChinese.value ? 'zh-CN' : 'en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  }).format(date)
}

function formatWeekdays(days: number[]): string {
  const normalized = normalizeWeekdays(days)
  if (normalized.length === 7) return localText('每天', 'Every day')
  const labels = new Map(weekdayOptions.value.map((item) => [item.value, item.label]))
  return normalized.map((day) => labels.get(day)).filter(Boolean).join(isChinese.value ? '、' : ', ')
}

function isDiscountGroupSelected(groupID: number): boolean {
  return form.groupDiscount.group_ids.includes(groupID)
}

function toggleDiscountGroup(groupID: number, checked: boolean) {
  const ids = new Set(form.groupDiscount.group_ids)
  checked ? ids.add(groupID) : ids.delete(groupID)
  form.groupDiscount.group_ids = Array.from(ids).sort((a, b) => a - b)
}

function isDiscountWeekdaySelected(day: number): boolean {
  return form.groupDiscount.weekdays.includes(day)
}

function toggleDiscountWeekday(day: number, checked: boolean) {
  const days = new Set(form.groupDiscount.weekdays)
  checked ? days.add(day) : days.delete(day)
  form.groupDiscount.weekdays = Array.from(days).sort((a, b) => a - b)
}

function selectAllDiscountWeekdays() {
  form.groupDiscount.weekdays = [1, 2, 3, 4, 5, 6, 7]
}

function addTicketTemplate() {
  form.ticketSystem.templates.push({
    key: `custom_${Date.now()}`,
    name: localText('自定义问题', 'Custom issue'),
    description: '',
    category: 'general',
    priority: 'normal',
    subject_template: '',
    body_min_length: 15,
    requires_super_admin: false,
    auto_assign_super_admin: false,
    context_type: '',
    fields: [],
  })
}

function duplicateTicketTemplate(index: number) {
  const current = form.ticketSystem.templates[index]
  if (!current) return
  const duplicated = clone(current)
  duplicated.key = `${current.key}_copy_${Date.now()}`
  duplicated.name = `${current.name} Copy`
  form.ticketSystem.templates.splice(index + 1, 0, duplicated)
}

function removeTicketTemplate(index: number) {
  if (form.ticketSystem.templates.length <= 1) {
    appStore.showError(localText('至少保留一个工单类型。', 'Keep at least one ticket type.'))
    return
  }
  form.ticketSystem.templates.splice(index, 1)
}

function resetTicketSettings() {
  form.ticketSystem = defaultTicketSystemSettings()
}

function addTicketTemplateField(template: TicketTemplate) {
  if (!template.fields) template.fields = []
  template.fields.push({ key: `field_${template.fields.length + 1}`, label: localText('字段名称', 'Field label'), type: 'text', required: false, min_length: 0, max_length: 0, options: [] })
}

function removeTicketTemplateField(template: TicketTemplate, index: number) {
  template.fields?.splice(index, 1)
}

function addTicketFieldOption(field: TicketTemplateField) {
  if (!field.options) field.options = []
  field.options.push({ value: `option_${field.options.length + 1}`, label: localText('选项', 'Option') })
}

function removeTicketFieldOption(field: TicketTemplateField, index: number) {
  field.options?.splice(index, 1)
}

function validateTicketSettings(): TicketSystemSettings | null {
  const normalized = normalizeTicketSystem(form.ticketSystem)
  const templateKeys = new Set<string>()
  for (const template of normalized.templates) {
    if (templateKeys.has(template.key)) {
      appStore.showError(localText(`工单类型标识重复：${template.key}`, `Duplicate ticket type key: ${template.key}`))
      return null
    }
    templateKeys.add(template.key)
    const fieldKeys = new Set<string>()
    for (const field of template.fields || []) {
      if (fieldKeys.has(field.key)) {
        appStore.showError(localText(`${template.name} 字段标识重复：${field.key}`, `${template.name} has duplicate field key: ${field.key}`))
        return null
      }
      fieldKeys.add(field.key)
      if (field.type === 'select' && !field.options?.length) {
        appStore.showError(localText(`${template.name} 的下拉字段至少需要一个选项。`, `${template.name} select fields need at least one option.`))
        return null
      }
      if ((field.max_length || 0) > 0 && (field.min_length || 0) > (field.max_length || 0)) {
        appStore.showError(localText(`${template.name} 字段最小长度不能大于最大长度。`, `${template.name} field min length cannot exceed max length.`))
        return null
      }
    }
  }
  normalized.sla.first_response_minutes = Math.max(1, Math.floor(Number(normalized.sla.first_response_minutes) || 1440))
  normalized.sla.reminder_before_minutes = Math.max(0, Math.floor(Number(normalized.sla.reminder_before_minutes) || 0))
  normalized.sla.auto_escalate_after_minutes = Math.max(0, Math.floor(Number(normalized.sla.auto_escalate_after_minutes) || 0))
  normalized.sla.auto_close_resolved_days = Math.max(0, Math.floor(Number(normalized.sla.auto_close_resolved_days) || 0))
  normalized.sla.worker_interval_seconds = Math.max(30, Math.floor(Number(normalized.sla.worker_interval_seconds) || 300))
  normalized.sla.reminder_before_minutes = Math.min(normalized.sla.reminder_before_minutes, normalized.sla.first_response_minutes)
  return normalized
}

function buildDiscountPayload(): GroupRateDiscountSettings | null {
  const normalized = normalizeGroupDiscount(form.groupDiscount)
  normalized.name = String(normalized.name || '').trim() || localText('限时折扣', 'Limited-time discount')
  normalized.discount_multiplier = Number(normalized.discount_multiplier) || 1
  if (normalized.schedule_mode === 'once') {
    normalized.start_at = toRFC3339(form.groupDiscount.start_at)
    normalized.end_at = toRFC3339(form.groupDiscount.end_at)
  } else {
    normalized.start_at = ''
    normalized.end_at = ''
  }
  if (!normalized.enabled) return normalized
  if (normalized.discount_multiplier <= 0 || normalized.discount_multiplier >= 1) {
    appStore.showError(localText('折扣倍率必须大于 0 且小于 1。', 'Discount multiplier must be greater than 0 and less than 1.'))
    return null
  }
  if (!normalized.group_ids.length) {
    appStore.showError(localText('请至少选择一个参与分组。', 'Select at least one discount group.'))
    return null
  }
  if (normalized.schedule_mode === 'weekly') {
    if (!normalized.weekdays.length) {
      appStore.showError(localText('请至少选择一个生效日期。', 'Select at least one active weekday.'))
      return null
    }
    if (normalized.daily_start_time === normalized.daily_end_time) {
      appStore.showError(localText('每日结束时间不能等于开始时间。', 'Daily end time must differ from the start time.'))
      return null
    }
  } else {
    const start = new Date(normalized.start_at)
    const end = new Date(normalized.end_at)
    if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end <= start) {
      appStore.showError(localText('单次折扣的结束时间必须晚于开始时间。', 'The one-time discount must end after it starts.'))
      return null
    }
  }
  return normalized
}

async function loadSettings() {
  loading.value = true
  try {
    const [settings, groupItems] = await Promise.all([
      adminAPI.settings.getSettings(),
      adminAPI.groups.getAll(),
    ])
    form.ticketSystem = normalizeTicketSystem(settings.ticket_system_config)
    form.groupDiscount = normalizeGroupDiscount(settings.group_rate_discount_settings)
    groups.value = groupItems
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, localText('业务设置加载失败。', 'Failed to load business settings.')))
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  const ticketSystem = validateTicketSettings()
  if (!ticketSystem) return
  const groupDiscount = buildDiscountPayload()
  if (!groupDiscount) return

  saving.value = true
  try {
    const updated = await adminAPI.settings.updateSettings({
      ticket_system_config: ticketSystem,
      group_rate_discount_settings: groupDiscount,
    })
    form.ticketSystem = normalizeTicketSystem(updated.ticket_system_config || ticketSystem)
    form.groupDiscount = normalizeGroupDiscount(updated.group_rate_discount_settings || groupDiscount)
    void appStore.fetchPublicSettings(true)
    appStore.showSuccess(localText('业务设置已保存。', 'Business settings saved.'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, localText('业务设置保存失败。', 'Failed to save business settings.')))
  } finally {
    saving.value = false
  }
}

onMounted(loadSettings)
</script>
