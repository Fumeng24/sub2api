<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <UserPageHero
        :kicker="t('invoice.page.gateway.kicker')"
        :title="t('invoice.page.title')"
      >
        <template #actions>
          <button class="btn btn-secondary w-full justify-center sm:w-auto" :disabled="loading" @click="reload">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </template>

        <template #below>
          <UserSummaryStats
            class="mt-5"
            :items="invoiceSummaryItems"
            grid-class="grid-cols-2 sm:grid-cols-3 xl:grid-cols-5"
          />

        </template>
      </UserPageHero>

      <div class="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_390px]">
        <section class="overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm">
          <div class="border-b border-[color:var(--apple-border-soft)] p-4">
            <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-[var(--apple-text)]">
                  {{ t('invoice.page.orders.title') }}
                </h2>
              </div>
              <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:min-w-[520px] lg:grid-cols-4">
                <div class="lg:col-span-2">
                  <label class="input-label">{{ t('invoice.page.orders.keywordLabel') }}</label>
                  <input
                    v-model.trim="orderFilters.keyword"
                    class="input"
                    :placeholder="t('invoice.page.orders.keywordPlaceholder')"
                  />
                </div>
                <div>
                  <label class="input-label">{{ t('invoice.page.orders.statusLabel') }}</label>
                  <Select v-model="orderFilters.status" :options="orderStatusOptions" @change="handleOrderServerFilterChange" />
                </div>
                <div>
                  <label class="input-label">{{ t('invoice.page.orders.invoiceabilityLabel') }}</label>
                  <Select v-model="orderFilters.invoiceability" :options="invoiceabilityOptions" @change="handleOrderServerFilterChange" />
                </div>
              </div>
            </div>

            <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-[1fr_auto_1fr] sm:items-end lg:max-w-[520px]">
              <div>
                <label class="input-label">{{ t('invoice.page.orders.startDate') }}</label>
                <input v-model="orderFilters.start_date" class="input" type="date" />
              </div>
              <span class="hidden pb-2 text-center text-sm text-[var(--apple-muted-2)] sm:block">~</span>
              <div>
                <label class="input-label">{{ t('invoice.page.orders.endDate') }}</label>
                <input v-model="orderFilters.end_date" class="input" type="date" />
              </div>
            </div>

            <div class="mt-3 flex items-center justify-between gap-3 text-sm lg:hidden">
              <label class="inline-flex items-center gap-2 text-[var(--apple-muted)]">
                <input
                  type="checkbox"
                  class="h-4 w-4 rounded border-[color:var(--apple-border)] text-[var(--apple-blue)] focus:ring-[color:var(--apple-focus-ring)]"
                  :checked="allVisibleInvoiceableSelected"
                  :disabled="visibleInvoiceableRows.length === 0"
                  @change="toggleAllVisibleOrders"
                />
                <span>{{ t('common.selectAll') }}</span>
              </label>
              <span class="text-xs text-[var(--apple-muted)]">
                {{ visibleInvoiceableRows.length }} {{ t('invoice.page.orders.invoiceability.available') }}
              </span>
            </div>
          </div>

          <div v-if="ordersLoading" class="flex justify-center py-16">
            <div class="h-8 w-8 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
          </div>

          <div v-else-if="visibleOrderRows.length === 0" class="py-16 text-center">
            <Icon name="inbox" size="xl" class="mx-auto mb-3 text-[var(--apple-muted-2)]" />
            <p class="font-medium text-[var(--apple-text)]">{{ t('invoice.page.orders.emptyTitle') }}</p>
            <p class="mt-1 text-sm text-[var(--apple-muted)]">{{ t('invoice.page.orders.emptyDescription') }}</p>
          </div>

          <template v-else>
            <div class="hidden lg:block">
              <div class="overflow-x-auto">
                <table class="min-w-[900px] w-full text-left text-sm">
                  <thead class="bg-[var(--apple-surface-elevated)] text-xs text-[var(--apple-muted)]">
                    <tr>
                      <th class="w-10 px-4 py-3">
                        <label class="inline-flex items-center">
                          <input
                            type="checkbox"
                            class="h-4 w-4 rounded border-[color:var(--apple-border)] text-[var(--apple-blue)] focus:ring-[color:var(--apple-focus-ring)]"
                            :checked="allVisibleInvoiceableSelected"
                            :disabled="visibleInvoiceableRows.length === 0"
                            @change="toggleAllVisibleOrders"
                          />
                        </label>
                      </th>
                      <th class="px-4 py-3">{{ t('invoice.page.orders.columns.order') }}</th>
                      <th class="px-4 py-3 text-right">{{ t('invoice.page.orders.columns.amount') }}</th>
                      <th class="px-4 py-3">{{ t('invoice.page.orders.columns.method') }}</th>
                      <th class="px-4 py-3">{{ t('invoice.page.orders.columns.status') }}</th>
                      <th class="px-4 py-3">{{ t('invoice.page.orders.columns.paidAt') }}</th>
                      <th class="px-4 py-3">{{ t('invoice.page.orders.columns.invoiceability') }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-[color:var(--apple-border-soft)]">
                    <tr
                      v-for="row in visibleOrderRows"
                      :key="row.order.id"
                      :class="[
                        row.invoiceable ? 'bg-[color-mix(in_srgb,var(--apple-blue)_3%,var(--apple-surface))]' : 'bg-[var(--apple-surface)]',
                        'transition-colors hover:bg-[var(--apple-hover)]'
                      ]"
                    >
                      <td class="px-4 py-3 align-top">
                        <input
                          type="checkbox"
                          class="h-4 w-4 rounded border-[color:var(--apple-border)] text-[var(--apple-blue)] focus:ring-[color:var(--apple-focus-ring)] disabled:cursor-not-allowed disabled:opacity-40"
                          :checked="isOrderSelected(row)"
                          :disabled="!row.invoiceable"
                          @change="toggleOrderSelection(row)"
                        />
                      </td>
                      <td class="max-w-[260px] px-4 py-3 align-top">
                        <div class="font-mono text-xs text-[var(--apple-text)]">#{{ row.order.id }}</div>
                        <div class="mt-1 truncate font-mono text-xs text-[var(--apple-muted)]" :title="row.order.out_trade_no">
                          {{ row.order.out_trade_no || t('common.notAvailable') }}
                        </div>
                      </td>
                      <td class="px-4 py-3 text-right align-top">
                        <div class="font-semibold text-[var(--apple-text)]">{{ formatMoney(row.invoiceAmount) }}</div>
                      </td>
                      <td class="px-4 py-3 align-top text-[var(--apple-muted)]">{{ paymentTypeLabel(row.order.payment_type) }}</td>
                      <td class="px-4 py-3 align-top">
                        <span :class="['badge', orderStatusBadgeClass(row.order.status)]">{{ orderStatusLabel(row.order.status) }}</span>
                      </td>
                      <td class="whitespace-nowrap px-4 py-3 align-top text-[var(--apple-muted)]">{{ formatDateTime(row.paidAt) }}</td>
                      <td class="px-4 py-3 align-top">
                        <span :class="['badge', row.invoiceable ? 'badge-success' : 'badge-gray']">
                          {{ row.invoiceable ? t('invoice.page.orders.invoiceability.available') : t('invoice.page.orders.invoiceability.unavailable') }}
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <div class="space-y-3 p-4 lg:hidden">
              <article
                v-for="row in visibleOrderRows"
                :key="row.order.id"
                class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm"
              >
                <div class="flex items-start justify-between gap-3">
                  <label class="flex min-w-0 items-start gap-3">
                    <input
                      type="checkbox"
                      class="mt-1 h-4 w-4 rounded border-[color:var(--apple-border)] text-[var(--apple-blue)] focus:ring-[color:var(--apple-focus-ring)] disabled:cursor-not-allowed disabled:opacity-40"
                      :checked="isOrderSelected(row)"
                      :disabled="!row.invoiceable"
                      @change="toggleOrderSelection(row)"
                    />
                    <div class="min-w-0">
                      <p class="font-mono text-xs text-[var(--apple-muted)]">#{{ row.order.id }}</p>
                      <p class="mt-0.5 truncate text-sm font-medium text-[var(--apple-text)]" :title="row.order.out_trade_no">
                        {{ row.order.out_trade_no || t('common.notAvailable') }}
                      </p>
                    </div>
                  </label>
                  <span :class="['badge', row.invoiceable ? 'badge-success' : 'badge-gray']">
                    {{ row.invoiceable ? t('invoice.page.orders.invoiceability.available') : t('invoice.page.orders.invoiceability.unavailable') }}
                  </span>
                </div>

                <dl class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <div>
                    <dt class="text-xs text-[var(--apple-muted)]">{{ t('invoice.page.orders.columns.amount') }}</dt>
                    <dd class="mt-1 font-semibold text-[var(--apple-text)]">{{ formatMoney(row.invoiceAmount) }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-[var(--apple-muted)]">{{ t('invoice.page.orders.columns.method') }}</dt>
                    <dd class="mt-1 text-[var(--apple-text)]">{{ paymentTypeLabel(row.order.payment_type) }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-[var(--apple-muted)]">{{ t('invoice.page.orders.columns.paidAt') }}</dt>
                    <dd class="mt-1 text-[var(--apple-text)]">{{ formatDateTime(row.paidAt) }}</dd>
                  </div>
                </dl>

                <div class="mt-4 flex flex-wrap items-center gap-2">
                  <span :class="['badge', orderStatusBadgeClass(row.order.status)]">{{ orderStatusLabel(row.order.status) }}</span>
                </div>
              </article>
            </div>
          </template>

          <div class="border-t border-[color:var(--apple-border-soft)]">
            <Pagination
              v-if="orderPagination.total > 0"
              :page="orderPagination.page"
              :total="orderPagination.total"
              :page-size="orderPagination.page_size"
              @update:page="handleOrderPageChange"
              @update:pageSize="handleOrderPageSizeChange"
            />
          </div>
        </section>

        <aside class="space-y-5">
          <section class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm xl:sticky xl:top-5">
            <div class="mb-4 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-[var(--apple-text)]">{{ t('invoice.page.form.title') }}</h2>
                <p class="mt-1 text-sm text-[var(--apple-muted)]">
                  {{ t('invoice.page.form.selectedSummary', { count: selectedOrderRows.length, amount: formatMoney(selectedInvoiceAmount) }) }}
                </p>
              </div>
            </div>

            <div class="mb-4 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-3 text-xs leading-5 text-[var(--apple-muted)]">
              <p>{{ t('invoice.page.form.limitHint', { min: formatMoney(minInvoiceAmount), available: formatMoney(summary?.available_amount) }) }}</p>
              <p class="mt-1">
                {{ t('invoice.page.form.taxHint', {
                  tax: formatMoney(taxFeePreview),
                  rate: summary?.tax_rate_percent ?? 3,
                }) }}
              </p>
              <p v-if="balanceInsufficient" class="mt-1 text-[var(--apple-danger)]">
                {{ t('invoice.page.form.balanceInsufficientHint', { balance: formatMoney(summary?.current_balance), tax: formatMoney(taxFeePreview) }) }}
              </p>
            </div>

            <form class="space-y-4" @submit.prevent="submit">
              <div>
                <div class="flex items-center justify-between gap-3">
                  <label class="input-label">{{ t('invoice.page.form.templateLabel') }}</label>
                  <button type="button" class="text-xs font-medium text-[var(--apple-blue)] hover:text-[var(--apple-blue-hover)]" @click="openCreateTemplateDialog">
                    {{ t('invoice.page.form.saveTemplate') }}
                  </button>
                </div>
                <Select v-model="selectedTemplateId" :options="templateOptions" @change="applySelectedTemplate" />
                <div v-if="activeTemplate" class="mt-2 flex flex-wrap items-center gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="openUpdateTemplateDialog">
                    {{ t('invoice.page.form.updateTemplate') }}
                  </button>
                  <button
                    v-if="!activeTemplate.is_default"
                    type="button"
                    class="btn btn-secondary btn-sm"
                    :disabled="templateSaving"
                    @click="setDefaultTemplate"
                  >
                    {{ t('invoice.page.form.setDefault') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm text-[var(--apple-danger)] hover:text-[var(--apple-danger)]"
                    :disabled="templateSaving"
                    @click="deleteSelectedTemplate"
                  >
                    {{ t('invoice.page.form.deleteTemplate') }}
                  </button>
                </div>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('invoice.page.form.typeLabel') }}</label>
                  <Select v-model="form.invoice_type" :options="invoiceTypeOptions" />
                </div>
                <div>
                  <label class="input-label">{{ t('invoice.page.form.amountLabel') }}</label>
                  <input v-model.number="form.amount" class="input" type="number" min="0" step="0.01" required :placeholder="t('invoice.page.form.amountPlaceholder')" />
                </div>
              </div>

              <div>
                <label class="input-label">{{ t('invoice.page.form.titleLabel') }}</label>
                <input v-model.trim="form.title" class="input" maxlength="255" required :placeholder="t('invoice.page.form.titlePlaceholder')" />
              </div>

              <div v-if="form.invoice_type !== 'personal'">
                <label class="input-label">{{ t('invoice.page.form.taxIdLabel') }}</label>
                <input v-model.trim="form.tax_id" class="input uppercase" maxlength="100" required :placeholder="t('invoice.page.form.taxIdPlaceholder')" />
              </div>

              <div>
                <label class="input-label">{{ t('invoice.page.form.itemNameLabel') }}</label>
                <input v-model.trim="form.item_name" class="input" maxlength="100" required :placeholder="t('invoice.page.form.itemNamePlaceholder')" />
              </div>

              <div>
                <label class="input-label">{{ t('invoice.page.form.receiverEmailLabel') }}</label>
                <input v-model.trim="form.receiver_email" class="input" type="email" maxlength="255" required :placeholder="t('invoice.page.form.receiverEmailPlaceholder')" />
              </div>

              <div>
                <div class="flex items-center justify-between gap-3">
                  <label class="input-label">{{ t('invoice.page.form.noteLabel') }}</label>
                  <span class="text-xs text-[var(--apple-muted-2)]">{{ form.note.length }}/1000</span>
                </div>
                <textarea
                  v-model.trim="form.note"
                  class="input"
                  rows="3"
                  maxlength="1000"
                  :placeholder="t('invoice.page.form.notePlaceholder')"
                ></textarea>
              </div>

              <button class="btn btn-primary w-full py-3" :disabled="!canSubmit || submitting">
                <span v-if="submitting" class="inline-flex items-center gap-2">
                  <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                  {{ t('invoice.page.form.submitting') }}
                </span>
                <span v-else>{{ t('invoice.page.form.submit') }}</span>
              </button>
              <p v-if="!summary?.can_apply" class="text-center text-xs text-[var(--apple-warning)]">
                {{ t('invoice.page.form.minimumNotMet') }}
              </p>
            </form>
          </section>
        </aside>
      </div>

      <section class="overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm">
        <div class="border-b border-[color:var(--apple-border-soft)] p-5">
          <h2 class="text-base font-semibold text-[var(--apple-text)]">{{ t('invoice.page.records.title') }}</h2>
        </div>

        <div v-if="loading" class="flex justify-center py-16">
          <div class="h-8 w-8 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
        </div>

        <div v-else-if="invoices.length === 0" class="py-16 text-center">
          <Icon name="document" size="xl" class="mx-auto mb-3 text-[var(--apple-muted-2)]" />
          <p class="font-medium text-[var(--apple-text)]">{{ t('invoice.page.records.emptyTitle') }}</p>
          <p class="mt-1 text-sm text-[var(--apple-muted)]">{{ t('invoice.page.records.emptyDescription') }}</p>
        </div>

        <template v-else>
          <div class="hidden md:block overflow-x-auto">
            <table class="min-w-[900px] w-full text-left text-sm">
              <thead class="bg-[var(--apple-surface-elevated)] text-xs text-[var(--apple-muted)]">
                <tr>
                  <th class="px-4 py-3">{{ t('invoice.page.records.columns.title') }}</th>
                  <th class="px-4 py-3">{{ t('invoice.page.records.columns.type') }}</th>
                  <th class="px-4 py-3 text-right">{{ t('invoice.page.records.columns.amount') }}</th>
                  <th class="px-4 py-3">{{ t('invoice.page.records.columns.status') }}</th>
                  <th class="px-4 py-3">{{ t('invoice.page.records.columns.number') }}</th>
                  <th class="px-4 py-3">{{ t('invoice.page.records.columns.submittedAt') }}</th>
                  <th class="px-4 py-3 text-right">{{ t('invoice.page.records.columns.actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-[color:var(--apple-border-soft)]">
                <tr v-for="item in invoices" :key="item.id" class="transition-colors hover:bg-[var(--apple-hover)]">
                  <td class="max-w-[220px] truncate px-4 py-3 font-medium text-[var(--apple-text)]" :title="item.title">{{ item.title }}</td>
                  <td class="px-4 py-3 text-[var(--apple-muted)]">{{ invoiceTypeLabel(item.invoice_type) }}</td>
                  <td class="px-4 py-3 text-right font-semibold text-[var(--apple-text)]">{{ formatMoney(item.amount) }}</td>
                  <td class="px-4 py-3">
                    <span :class="['badge', statusBadgeClass(item.status)]">{{ statusLabel(item.status) }}</span>
                  </td>
                  <td class="px-4 py-3 text-[var(--apple-muted)]">{{ item.invoice_no || t('common.notAvailable') }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-[var(--apple-muted)]">{{ formatDateTime(item.created_at) }}</td>
                  <td class="px-4 py-3 text-right">
                    <button
                      v-if="item.status === 'pending'"
                      class="text-sm font-medium text-[var(--apple-muted)] hover:text-[var(--apple-text)]"
                      @click="cancel(item)"
                    >
                      {{ t('invoice.page.records.cancel') }}
                    </button>
                    <span v-else class="text-sm text-[var(--apple-muted-2)]">{{ t('common.notAvailable') }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="space-y-3 p-4 md:hidden">
            <article
              v-for="item in invoices"
              :key="item.id"
              class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate text-base font-medium text-[var(--apple-text)]" :title="item.title">
                    {{ item.title }}
                  </p>
                  <p class="mt-0.5 text-xs text-[var(--apple-muted)]">{{ invoiceTypeLabel(item.invoice_type) }}</p>
                </div>
                <span :class="['badge', statusBadgeClass(item.status)]">{{ statusLabel(item.status) }}</span>
              </div>

              <dl class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <dt class="text-xs text-[var(--apple-muted)]">{{ t('invoice.page.records.columns.amount') }}</dt>
                  <dd class="mt-1 font-semibold text-[var(--apple-text)]">{{ formatMoney(item.amount) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-[var(--apple-muted)]">{{ t('invoice.page.records.columns.number') }}</dt>
                  <dd class="mt-1 text-[var(--apple-text)]">{{ item.invoice_no || t('common.notAvailable') }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-[var(--apple-muted)]">{{ t('invoice.page.records.columns.submittedAt') }}</dt>
                  <dd class="mt-1 text-[var(--apple-text)]">{{ formatDateTime(item.created_at) }}</dd>
                </div>
              </dl>

              <div v-if="item.status === 'pending'" class="mt-4 flex justify-end">
                <button
                  class="text-sm font-medium text-[var(--apple-muted)] hover:text-[var(--apple-text)]"
                  @click="cancel(item)"
                >
                  {{ t('invoice.page.records.cancel') }}
                </button>
              </div>
            </article>
          </div>
        </template>

        <div class="border-t border-[color:var(--apple-border-soft)]">
          <Pagination
            v-if="invoicePagination.total > 0"
            :page="invoicePagination.page"
            :total="invoicePagination.total"
            :page-size="invoicePagination.page_size"
            @update:page="handleInvoicePageChange"
            @update:pageSize="handleInvoicePageSizeChange"
          />
        </div>
      </section>
    </div>

    <BaseDialog :show="templateDialog.open" :title="templateDialogTitle" width="narrow" @close="templateDialog.open = false">
      <div class="space-y-4">
        <div>
          <label class="input-label">{{ t('invoice.page.dialog.templateName') }}</label>
          <input v-model.trim="templateDialog.name" class="input" maxlength="80" :placeholder="t('invoice.page.dialog.templateNamePlaceholder')" />
        </div>
        <label class="flex items-start gap-3 rounded-lg border border-[color:var(--apple-border)] p-3 text-sm">
          <input v-model="templateDialog.is_default" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-[color:var(--apple-border)] text-[var(--apple-blue)] focus:ring-[color:var(--apple-focus-ring)]" />
          <span class="text-[var(--apple-muted)]">{{ t('invoice.page.dialog.defaultTemplate') }}</span>
        </label>
      </div>
      <template #footer>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button class="btn btn-secondary w-full sm:w-auto" @click="templateDialog.open = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary w-full sm:w-auto" :disabled="!canSaveTemplate || templateSaving" @click="saveTemplate">
            {{ templateSaving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import UserSummaryStats from '@/custom/user/UserSummaryStats.vue'
import invoicesAPI, { type InvoiceRequest, type InvoiceSummary, type InvoiceTemplate, type InvoiceType } from '@/custom/api/invoices'
import { paymentAPI } from '@/custom/api/payment'
import type { PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import {
  invoiceUnavailableReasonKey,
  orderNetInvoiceAmount,
} from '@/custom/utils/paymentRecordSemantics'
import { useI18n } from 'vue-i18n'

type InvoiceabilityFilter = 'all' | 'available' | 'unavailable'

interface OrderRow {
  order: PaymentOrder
  invoiceable: boolean
  reason: string
  paidAt: string | null
  invoiceAmount: number
}

const { t, locale } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const ordersLoading = ref(false)
const submitting = ref(false)
const templateSaving = ref(false)
const summary = ref<InvoiceSummary | null>(null)
const invoices = ref<InvoiceRequest[]>([])
const templates = ref<InvoiceTemplate[]>([])
const orders = ref<PaymentOrder[]>([])
const selectedOrderIds = ref<Set<number>>(new Set())
const selectedTemplateId = ref<number | ''>('')
const invoiceSummaryItems = computed(() => [
  {
    label: t('invoice.page.stats.totalOrders'),
    value: String(orderPagination.total),
    valueClass: '',
  },
  {
    label: t('invoice.page.stats.availableOnPage'),
    value: String(invoiceableOrderRows.value.length),
    valueClass: '',
  },
  {
    label: t('invoice.page.stats.availableAmount'),
    value: formatMoney(summary.value?.available_amount),
    valueClass: '',
  },
  {
    label: t('invoice.page.stats.lockedAmount'),
    value: formatMoney(lockedInProgress.value),
    valueClass: 'text-[var(--apple-warning)]',
  },
  {
    label: t('invoice.page.stats.unavailableOnPage'),
    value: String(unavailableOrderRows.value.length),
    valueClass: '',
  },
])

const invoicePagination = reactive({ page: 1, page_size: 10, total: 0 })
const orderPagination = reactive({ page: 1, page_size: 20, total: 0 })

const orderFilters = reactive({
  keyword: '',
  status: '',
  invoiceability: 'available' as InvoiceabilityFilter,
  start_date: '',
  end_date: '',
})

const form = reactive({
  invoice_type: 'company_vat_general' as InvoiceType,
  title: '',
  tax_id: '',
  item_name: t('invoice.page.form.defaultItemName'),
  amount: undefined as number | undefined,
  receiver_email: '',
  note: '',
})

const templateDialog = reactive({
  open: false,
  mode: 'create' as 'create' | 'update',
  id: 0,
  name: '',
  is_default: false,
})

const invoiceTypeOptions = computed(() => [
  { value: 'company_vat_general', label: t('invoice.page.types.company_vat_general') },
  { value: 'company_vat_special', label: t('invoice.page.types.company_vat_special') },
  { value: 'personal', label: t('invoice.page.types.personal') },
])

const orderStatusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'PARTIALLY_REFUNDED', label: t('payment.status.partially_refunded') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'CANCELLED', label: t('payment.status.cancelled') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

const invoiceabilityOptions = computed(() => [
  { value: 'all', label: t('invoice.page.invoiceability.all') },
  { value: 'available', label: t('invoice.page.invoiceability.available') },
  { value: 'unavailable', label: t('invoice.page.invoiceability.unavailable') },
])

const templateOptions = computed(() => [
  { value: '', label: templates.value.length > 0 ? t('invoice.page.form.noTemplate') : t('invoice.page.form.noTemplates') },
  ...templates.value.map((item) => ({
    value: item.id,
    label: `${item.name}${item.is_default ? t('invoice.page.form.defaultSuffix') : ''}`,
  })),
])

const lockedInProgress = computed(() => Math.max((summary.value?.locked_amount || 0) - (summary.value?.invoiced_amount || 0), 0))
const minInvoiceAmount = computed(() => roundMoney(summary.value?.min_amount || 100))
const taxFeePreview = computed(() => {
  const amount = Number(form.amount) || 0
  if (amount <= 0) return 0
  const rate = Number(summary.value?.tax_rate) || 0.03
  return roundMoney(amount * rate)
})
const balanceInsufficient = computed(() => {
  if (!summary.value) return false
  return taxFeePreview.value > (Number(summary.value.current_balance) || 0) + 0.000001
})

const orderRows = computed<OrderRow[]>(() => orders.value.map((order) => {
  const invoiceAmount = orderNetInvoiceAmount(order)
  const paidAt = order.completed_at || order.paid_at || order.created_at || null
  const reason = getUnavailableReason(order, invoiceAmount)
  return {
    order,
    invoiceable: reason === '',
    reason,
    paidAt,
    invoiceAmount,
  }
}))

const visibleOrderRows = computed(() => {
  const keyword = orderFilters.keyword.trim().toLowerCase()
  return orderRows.value.filter((row) => {
    if (orderFilters.invoiceability === 'available' && !row.invoiceable) return false
    if (orderFilters.invoiceability === 'unavailable' && row.invoiceable) return false
    if (keyword) {
      const haystack = `${row.order.id} ${row.order.out_trade_no || ''}`.toLowerCase()
      if (!haystack.includes(keyword)) return false
    }
    if (!isWithinDateRange(row.paidAt)) return false
    return true
  })
})

const invoiceableOrderRows = computed(() => orderRows.value.filter((row) => row.invoiceable))
const unavailableOrderRows = computed(() => orderRows.value.filter((row) => !row.invoiceable))
const visibleInvoiceableRows = computed(() => visibleOrderRows.value.filter((row) => row.invoiceable))
const allVisibleInvoiceableSelected = computed(() => {
  if (visibleInvoiceableRows.value.length === 0) return false
  return visibleInvoiceableRows.value.every((row) => selectedOrderIds.value.has(row.order.id))
})
const selectedOrderRows = computed(() => orderRows.value.filter((row) => row.invoiceable && selectedOrderIds.value.has(row.order.id)))
const selectedInvoiceAmount = computed(() => roundMoney(selectedOrderRows.value.reduce((sum, row) => sum + row.invoiceAmount, 0)))

const activeTemplate = computed(() => {
  const id = Number(selectedTemplateId.value)
  if (!id) return null
  return templates.value.find((item) => item.id === id) || null
})

const templateDialogTitle = computed(() => (
  templateDialog.mode === 'update'
    ? t('invoice.page.dialog.updateTitle')
    : t('invoice.page.dialog.createTitle')
))

const canSubmit = computed(() => {
  const amount = Number(form.amount) || 0
  if (!summary.value?.can_apply) return false
  if (amount < minInvoiceAmount.value) return false
  if (amount > (summary.value.available_amount || 0)) return false
  if (balanceInsufficient.value) return false
  if (!form.title.trim() || !form.item_name.trim() || !form.receiver_email.trim()) return false
  if (form.invoice_type !== 'personal' && !form.tax_id.trim()) return false
  return true
})

const canSaveTemplate = computed(() => {
  if (!templateDialog.name.trim()) return false
  if (!form.title.trim() || !form.item_name.trim() || !form.receiver_email.trim()) return false
  if (form.invoice_type !== 'personal' && !form.tax_id.trim()) return false
  return true
})

async function reload() {
  loading.value = true
  try {
    await Promise.all([
      loadSummary(),
      loadInvoices(),
      loadOrders(),
      loadTemplates(),
    ])
  } finally {
    loading.value = false
  }
}

async function loadSummary() {
  try {
    const res = await invoicesAPI.getSummary()
    summary.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.loadSummaryFailed')))
  }
}

async function loadInvoices() {
  try {
    const res = await invoicesAPI.list({ page: invoicePagination.page, page_size: invoicePagination.page_size })
    invoices.value = res.data.items || []
    invoicePagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.loadInvoicesFailed')))
  }
}

async function loadOrders() {
  ordersLoading.value = true
  try {
    const invoiceable =
      orderFilters.invoiceability === 'available'
        ? true
        : orderFilters.invoiceability === 'unavailable'
          ? false
          : undefined
    const res = await paymentAPI.getMyOrders({
      page: orderPagination.page,
      page_size: orderPagination.page_size,
      status: orderFilters.status || undefined,
      invoiceable,
    })
    orders.value = res.data.items || []
    orderPagination.total = res.data.total || 0
    selectedOrderIds.value = new Set([...selectedOrderIds.value].filter((id) => orders.value.some((order) => order.id === id)))
    syncAmountFromSelection()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('invoice.page.messages.loadOrdersFailed')))
  } finally {
    ordersLoading.value = false
  }
}

async function loadTemplates() {
  try {
    const res = await invoicesAPI.listTemplates()
    templates.value = res.data || []
    if (!selectedTemplateId.value) {
      const defaultTemplate = templates.value.find((item) => item.is_default)
      if (defaultTemplate) {
        selectedTemplateId.value = defaultTemplate.id
        copyTemplateToForm(defaultTemplate)
      }
    }
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.loadTemplatesFailed')))
  }
}

async function submit() {
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  try {
    await invoicesAPI.create({
      invoice_type: form.invoice_type,
      title: form.title,
      tax_id: form.invoice_type === 'personal' ? '' : form.tax_id,
      item_name: form.item_name,
      amount: Number(form.amount),
      receiver_email: form.receiver_email,
      note: form.note,
      source_order_ids: selectedOrderRows.value.map((row) => row.order.id),
    })
    appStore.showSuccess(t('invoice.page.messages.submitSuccess'))
    selectedOrderIds.value = new Set()
    form.amount = undefined
    form.note = ''
    await Promise.all([loadSummary(), loadInvoices()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.submitFailed')))
  } finally {
    submitting.value = false
  }
}

async function cancel(item: InvoiceRequest) {
  try {
    await invoicesAPI.cancel(item.id)
    appStore.showSuccess(t('invoice.page.messages.cancelSuccess'))
    await Promise.all([loadSummary(), loadInvoices()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.cancelFailed')))
  }
}

function handleInvoicePageChange(page: number) {
  invoicePagination.page = page
  loadInvoices()
}

function handleInvoicePageSizeChange(size: number) {
  invoicePagination.page_size = size
  invoicePagination.page = 1
  loadInvoices()
}

function handleOrderPageChange(page: number) {
  orderPagination.page = page
  loadOrders()
}

function handleOrderPageSizeChange(size: number) {
  orderPagination.page_size = size
  orderPagination.page = 1
  loadOrders()
}

function handleOrderServerFilterChange() {
  orderPagination.page = 1
  loadOrders()
}

function isOrderSelected(row: OrderRow) {
  return selectedOrderIds.value.has(row.order.id)
}

function toggleOrderSelection(row: OrderRow) {
  if (!row.invoiceable) return
  const next = new Set(selectedOrderIds.value)
  if (next.has(row.order.id)) {
    next.delete(row.order.id)
  } else {
    next.add(row.order.id)
  }
  selectedOrderIds.value = next
  syncAmountFromSelection()
}

function toggleAllVisibleOrders() {
  const next = new Set(selectedOrderIds.value)
  if (allVisibleInvoiceableSelected.value) {
    visibleInvoiceableRows.value.forEach((row) => next.delete(row.order.id))
  } else {
    visibleInvoiceableRows.value.forEach((row) => next.add(row.order.id))
  }
  selectedOrderIds.value = next
  syncAmountFromSelection()
}

function syncAmountFromSelection() {
  if (selectedInvoiceAmount.value <= 0) return
  const available = summary.value?.available_amount || selectedInvoiceAmount.value
  form.amount = roundMoney(Math.min(selectedInvoiceAmount.value, available))
}

function applySelectedTemplate() {
  if (activeTemplate.value) {
    copyTemplateToForm(activeTemplate.value)
  }
}

function copyTemplateToForm(template: InvoiceTemplate) {
  form.invoice_type = template.invoice_type
  form.title = template.title
  form.tax_id = template.tax_id || ''
  form.item_name = template.item_name || t('invoice.page.form.defaultItemName')
  form.receiver_email = template.receiver_email || ''
  form.note = template.note || ''
}

function openCreateTemplateDialog() {
  templateDialog.mode = 'create'
  templateDialog.id = 0
  templateDialog.name = form.title.trim() || t('invoice.page.dialog.defaultTemplateName')
  templateDialog.is_default = templates.value.length === 0
  templateDialog.open = true
}

function openUpdateTemplateDialog() {
  if (!activeTemplate.value) return
  templateDialog.mode = 'update'
  templateDialog.id = activeTemplate.value.id
  templateDialog.name = activeTemplate.value.name
  templateDialog.is_default = activeTemplate.value.is_default
  templateDialog.open = true
}

async function saveTemplate() {
  if (!canSaveTemplate.value || templateSaving.value) return
  templateSaving.value = true
  try {
    const payload = {
      name: templateDialog.name,
      invoice_type: form.invoice_type,
      title: form.title,
      tax_id: form.invoice_type === 'personal' ? '' : form.tax_id,
      item_name: form.item_name,
      receiver_email: form.receiver_email,
      note: form.note,
      is_default: templateDialog.is_default,
    }
    const res = templateDialog.mode === 'update' && templateDialog.id > 0
      ? await invoicesAPI.updateTemplate(templateDialog.id, payload)
      : await invoicesAPI.createTemplate(payload)
    const successMessage = templateDialog.mode === 'update'
      ? t('invoice.page.messages.templateUpdated')
      : t('invoice.page.messages.templateSaved')
    templateDialog.open = false
    appStore.showSuccess(successMessage)
    await loadTemplates()
    selectedTemplateId.value = res.data.id
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.saveTemplateFailed')))
  } finally {
    templateSaving.value = false
  }
}

async function setDefaultTemplate() {
  if (!activeTemplate.value || templateSaving.value) return
  templateSaving.value = true
  try {
    const res = await invoicesAPI.setDefaultTemplate(activeTemplate.value.id)
    appStore.showSuccess(t('invoice.page.messages.defaultTemplateUpdated'))
    await loadTemplates()
    selectedTemplateId.value = res.data.id
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.defaultTemplateFailed')))
  } finally {
    templateSaving.value = false
  }
}

async function deleteSelectedTemplate() {
  if (!activeTemplate.value || templateSaving.value) return
  if (!window.confirm(t('invoice.page.messages.deleteTemplateConfirm'))) return
  templateSaving.value = true
  try {
    await invoicesAPI.deleteTemplate(activeTemplate.value.id)
    appStore.showSuccess(t('invoice.page.messages.templateDeleted'))
    selectedTemplateId.value = ''
    await loadTemplates()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.page.messages.deleteTemplateFailed')))
  } finally {
    templateSaving.value = false
  }
}

function isWithinDateRange(value?: string | null) {
  if (!value) return true
  const time = new Date(value).getTime()
  if (Number.isNaN(time)) return true
  if (orderFilters.start_date) {
    const start = new Date(`${orderFilters.start_date}T00:00:00`).getTime()
    if (time < start) return false
  }
  if (orderFilters.end_date) {
    const end = new Date(`${orderFilters.end_date}T23:59:59`).getTime()
    if (time > end) return false
  }
  return true
}

function localeCode(): string | undefined {
  const raw = locale.value as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
}

function formatMoney(value?: number | null) {
  return `¥${(Number(value) || 0).toFixed(2)}`
}

function getUnavailableReason(order: PaymentOrder, amount: number) {
  const reasonKey = invoiceUnavailableReasonKey(order, amount)
  return reasonKey ? t(`invoice.page.reasons.${reasonKey}`) : ''
}

function roundMoney(value: number) {
  return Math.round(value * 100) / 100
}

function formatDateTime(value?: string | null) {
  if (!value) return t('common.notAvailable')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('common.notAvailable')
  return date.toLocaleString(localeCode())
}

function invoiceTypeLabel(type: InvoiceType | string) {
  return t(`invoice.page.types.${type}`, type)
}

function statusLabel(status: string) {
  return t(`invoice.page.status.${status}`, status)
}

function statusBadgeClass(status: string) {
  const map: Record<string, string> = {
    pending: 'badge-warning',
    approved: 'badge-primary',
    rejected: 'badge-danger',
    completed: 'badge-success',
    cancelled: 'badge-gray',
  }
  return map[status] || 'badge-gray'
}

function orderStatusLabel(status: string) {
  return t(`payment.status.${status.toLowerCase()}`, status)
}

function orderStatusBadgeClass(status: string) {
  const map: Record<string, string> = {
    PENDING: 'badge-warning',
    PAID: 'badge-primary',
    RECHARGING: 'badge-warning',
    COMPLETED: 'badge-success',
    EXPIRED: 'badge-gray',
    CANCELLED: 'badge-gray',
    FAILED: 'badge-danger',
    REFUND_REQUESTED: 'badge-warning',
    REFUNDING: 'badge-warning',
    PARTIALLY_REFUNDED: 'badge-warning',
    REFUNDED: 'badge-gray',
    REFUND_FAILED: 'badge-danger',
  }
  return map[status] || 'badge-gray'
}

function paymentTypeLabel(type: string) {
  return t(`payment.methods.${type}`, type || t('common.notAvailable'))
}

onMounted(reload)
</script>
