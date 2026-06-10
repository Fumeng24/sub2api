<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <section class="relative overflow-hidden rounded-3xl border border-emerald-200 bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.20),_transparent_34%),linear-gradient(135deg,_#f8fafc_0%,_#ecfdf5_42%,_#fff7ed_100%)] p-5 shadow-sm dark:border-emerald-900/60 dark:bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.22),_transparent_34%),linear-gradient(135deg,_#08111f_0%,_#052e26_48%,_#1c1308_100%)] sm:p-7">
          <div class="pointer-events-none absolute -right-16 -top-20 h-52 w-52 rounded-full bg-amber-300/30 blur-3xl dark:bg-amber-500/10"></div>
          <div class="pointer-events-none absolute -bottom-24 left-1/3 h-56 w-56 rounded-full bg-emerald-400/20 blur-3xl dark:bg-emerald-400/10"></div>

          <div class="relative grid gap-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.65fr)] lg:items-stretch">
            <div class="min-w-0">
              <div class="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-white/70 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-emerald-700 shadow-sm backdrop-blur dark:border-emerald-800/70 dark:bg-white/10 dark:text-emerald-200">
                <Icon name="sparkles" size="xs" :stroke-width="2" />
                {{ t('affiliate.hero.kicker') }}
              </div>
              <h1 class="mt-4 max-w-3xl text-3xl font-black tracking-tight text-gray-950 dark:text-white sm:text-4xl lg:text-5xl">
                {{ t('affiliate.hero.title') }}
              </h1>
              <p class="mt-4 max-w-2xl text-base leading-7 text-gray-700 dark:text-gray-300">
                {{ t('affiliate.hero.description') }}
              </p>

              <div class="mt-5 grid gap-2 sm:grid-cols-3">
                <div
                  v-for="item in heroPills"
                  :key="item"
                  class="rounded-2xl border border-white/70 bg-white/75 px-3 py-2 text-sm font-semibold text-gray-800 shadow-sm backdrop-blur dark:border-white/10 dark:bg-white/10 dark:text-gray-100"
                >
                  {{ item }}
                </div>
              </div>

              <div class="mt-6 flex flex-col gap-3 sm:flex-row">
                <button class="btn btn-primary w-full sm:w-auto" @click="copyPromoText">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.promo.copyButton') }}</span>
                </button>
                <button class="btn btn-secondary w-full border-white/70 bg-white/75 sm:w-auto dark:border-white/10 dark:bg-white/10" @click="copyInviteLink">
                  <Icon name="link" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
              <p class="mt-3 text-sm text-emerald-800/80 dark:text-emerald-200/80">
                {{ t('affiliate.hero.shareHint') }}
              </p>
            </div>

            <div class="grid min-w-0 gap-3 sm:grid-cols-2 lg:grid-cols-1">
              <div
                v-for="item in affiliateStats"
                :key="item.label"
                class="rounded-2xl border border-white/70 bg-white/80 p-4 shadow-sm backdrop-blur dark:border-white/10 dark:bg-white/10"
              >
                <p class="text-sm text-gray-500 dark:text-dark-300">{{ item.label }}</p>
                <p class="mt-2 break-words text-2xl font-black text-gray-950 dark:text-white">
                  {{ item.value }}
                </p>
                <p v-if="item.hint" class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-300">{{ item.hint }}</p>
              </div>
            </div>
          </div>
        </section>

        <div class="grid gap-5 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]">
          <section class="card min-w-0 p-5 sm:p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="text-xs font-semibold uppercase tracking-[0.18em] text-emerald-600 dark:text-emerald-300">
                  {{ t('affiliate.sharePanel.kicker') }}
                </p>
                <h2 class="mt-2 text-xl font-black tracking-tight text-gray-950 dark:text-white">
                  {{ t('affiliate.sharePanel.title') }}
                </h2>
                <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">
                  {{ t('affiliate.sharePanel.description') }}
                </p>
              </div>
              <button class="btn btn-primary w-full sm:w-auto sm:shrink-0" @click="copyPromoText">
                <Icon name="copy" size="sm" />
                <span>{{ t('affiliate.promo.copyButton') }}</span>
              </button>
            </div>

            <div class="mt-5 space-y-4">
              <div class="min-w-0 space-y-2">
                <p class="text-sm font-semibold text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
                <div class="flex min-w-0 flex-col gap-2 rounded-2xl border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                  <code class="block min-w-0 flex-1 break-all text-sm font-bold tracking-wide text-gray-950 dark:text-white sm:truncate sm:break-normal" v-text="detail.aff_code"></code>
                  <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyCode') }}</span>
                  </button>
                </div>
              </div>

              <div class="min-w-0 space-y-2">
                <p class="text-sm font-semibold text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
                <div class="flex min-w-0 flex-col gap-2 rounded-2xl border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                  <code class="block min-w-0 flex-1 break-all text-sm text-gray-700 dark:text-gray-300 sm:truncate sm:break-normal" v-text="inviteLink"></code>
                  <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyLink') }}</span>
                  </button>
                </div>
              </div>
            </div>

          <div
            v-if="detail.inviter_id == null && detail.can_bind_inviter"
            class="mt-5 rounded-2xl border border-amber-200 bg-amber-50/80 p-4 dark:border-amber-900/50 dark:bg-amber-950/20"
          >
            <form
              class="flex flex-col gap-4 lg:flex-row lg:items-end"
              @submit.prevent="bindInviter"
            >
              <div class="min-w-0 flex-1">
                <h4 class="text-sm font-semibold text-amber-900 dark:text-amber-100">
                  {{ t('affiliate.bind.title') }}
                </h4>
                <p class="mt-1 text-sm text-amber-800/80 dark:text-amber-200/80">
                  {{ t('affiliate.bind.description') }}
                </p>
                <p
                  v-if="bindBonusAmount > 0"
                  class="mt-2 inline-flex rounded-full bg-white/70 px-3 py-1 text-xs font-semibold text-amber-700 shadow-sm dark:bg-white/10 dark:text-amber-200"
                >
                  {{ t('affiliate.bind.bonusHint', { amount: formatCurrency(bindBonusAmount) }) }}
                </p>
              </div>
              <div class="flex min-w-0 flex-col gap-2 sm:flex-row lg:w-[420px]">
                <input
                  v-model="bindCode"
                  type="text"
                  class="input min-w-0 flex-1 bg-white/90 uppercase dark:bg-dark-950/70"
                  :placeholder="t('affiliate.bind.codePlaceholder')"
                  :disabled="binding"
                  autocomplete="off"
                />
                <button
                  type="submit"
                  class="btn btn-primary shrink-0"
                  :disabled="binding || !bindCode.trim()"
                >
                  <Icon v-if="binding" name="refresh" size="sm" class="animate-spin" />
                  <Icon v-else name="users" size="sm" />
                  <span>{{ binding ? t('affiliate.bind.binding') : t('affiliate.bind.button') }}</span>
                </button>
              </div>
            </form>
          </div>

          <div
            v-if="detail.can_claim_bind_bonus && bindBonusAmount > 0"
            class="mt-5 rounded-2xl border border-emerald-200 bg-emerald-50/85 p-4 dark:border-emerald-900/50 dark:bg-emerald-950/20"
          >
            <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <h4 class="text-sm font-semibold text-emerald-900 dark:text-emerald-100">
                  {{ t('affiliate.bind.claimTitle') }}
                </h4>
                <p class="mt-1 text-sm leading-6 text-emerald-800/80 dark:text-emerald-200/80">
                  {{ t('affiliate.bind.claimDescription', { amount: formatCurrency(bindBonusAmount) }) }}
                </p>
              </div>
              <button
                type="button"
                class="btn btn-primary w-full sm:w-auto sm:shrink-0"
                :disabled="claimingBindBonus"
                @click="claimBindBonus"
              >
                <Icon v-if="claimingBindBonus" name="refresh" size="sm" class="animate-spin" />
                <Icon v-else name="gift" size="sm" />
                <span>{{ claimingBindBonus ? t('affiliate.bind.claiming') : t('affiliate.bind.claimButton', { amount: formatCurrency(bindBonusAmount) }) }}</span>
              </button>
            </div>
          </div>

            <div class="mt-5 overflow-hidden rounded-2xl border border-emerald-200 bg-emerald-50/80 p-4 dark:border-emerald-900/50 dark:bg-emerald-950/20">
              <p class="text-sm font-semibold text-emerald-900 dark:text-emerald-100">
                {{ t('affiliate.promo.previewTitle') }}
              </p>
              <div class="mt-3 break-words rounded-xl border border-white/80 bg-white/85 p-4 text-sm leading-6 text-gray-700 shadow-sm dark:border-white/10 dark:bg-dark-950/60 dark:text-gray-300">
                {{ promoShareText }}
              </div>
            </div>
          </section>

          <section class="card min-w-0 p-5 sm:p-6">
            <p class="text-xs font-semibold uppercase tracking-[0.18em] text-amber-600 dark:text-amber-300">
              {{ t('affiliate.friendBenefits.kicker') }}
            </p>
            <h2 class="mt-2 text-xl font-black tracking-tight text-gray-950 dark:text-white">
              {{ t('affiliate.friendBenefits.title') }}
            </h2>
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">
              {{ t('affiliate.friendBenefits.description') }}
            </p>

            <div class="mt-5 grid gap-3 sm:grid-cols-2">
              <div
                v-for="item in proofCards"
                :key="item.title"
                class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900"
              >
                <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-white text-emerald-600 shadow-sm dark:bg-white/10 dark:text-emerald-300">
                  <Icon :name="item.icon" size="sm" :stroke-width="2" />
                </div>
                <h3 class="mt-3 text-sm font-bold text-gray-950 dark:text-white">{{ item.title }}</h3>
                <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ item.description }}</p>
              </div>
            </div>
          </section>
        </div>

        <section class="card min-w-0 p-5 sm:p-6">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-300">
                {{ t('affiliate.audiences.kicker') }}
              </p>
              <h2 class="mt-2 text-xl font-black tracking-tight text-gray-950 dark:text-white">
                {{ t('affiliate.audiences.title') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">
                {{ t('affiliate.audiences.description') }}
              </p>
            </div>
          </div>

          <div class="mt-5 grid gap-4 lg:grid-cols-2">
            <article
              v-for="item in audienceCards"
              :key="item.title"
              class="min-w-0 rounded-2xl border border-gray-200 bg-gradient-to-br from-white to-gray-50 p-4 dark:border-dark-700 dark:from-dark-900 dark:to-dark-950"
            >
              <div class="flex items-start gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                  <Icon :name="item.icon" size="sm" :stroke-width="2" />
                </div>
                <div class="min-w-0">
                  <h3 class="text-base font-bold text-gray-950 dark:text-white">{{ item.title }}</h3>
                  <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ item.description }}</p>
                </div>
              </div>
              <div class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white/70 p-3 text-sm leading-6 text-gray-700 dark:border-dark-700 dark:bg-white/5 dark:text-gray-300">
                {{ item.copy }}
              </div>
              <button class="btn btn-secondary btn-sm mt-3 w-full sm:w-auto" @click="copyAudienceText(item.copy)">
                <Icon name="copy" size="sm" />
                <span>{{ t('affiliate.audiences.copyButton') }}</span>
              </button>
            </article>
          </div>
        </section>

        <section class="grid gap-5 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
          <div class="card min-w-0 p-5 sm:p-6">
            <p class="text-sm font-semibold text-primary-800 dark:text-primary-200">{{ t('affiliate.rules.title') }}</p>
            <div class="mt-4 grid gap-3">
              <div
                v-for="(item, index) in howItWorks"
                :key="item"
                class="flex gap-3 rounded-2xl border border-primary-100 bg-primary-50/70 p-3 dark:border-primary-900/40 dark:bg-primary-900/15"
              >
                <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-600 text-sm font-bold text-white">
                  {{ index + 1 }}
                </span>
                <p class="min-w-0 text-sm leading-6 text-primary-800 dark:text-primary-200">{{ item }}</p>
              </div>
            </div>
          </div>

          <div class="card min-w-0 p-5 sm:p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
          </div>
        </section>

        <div class="card min-w-0 p-5 sm:p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { useSettlementCurrency } from '@/composables/useSettlementCurrency'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()
const { formatSettlementAmount } = useSettlementCurrency()

const loading = ref(true)
const transferring = ref(false)
const binding = ref(false)
const claimingBindBonus = ref(false)
const bindCode = ref('')
const detail = ref<UserAffiliateDetail | null>(null)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

type AffiliateIconName = 'gift' | 'dollar' | 'shield' | 'sparkles' | 'terminal' | 'cpu' | 'lightbulb' | 'chat'

const heroPills = computed(() => [
  t('affiliate.hero.pillTrial'),
  t('affiliate.hero.pillCheap'),
  t('affiliate.hero.pillStable'),
])

const affiliateStats = computed(() => [
  {
    label: t('affiliate.stats.rebateRate'),
    value: `${formattedRebateRate.value}%`,
    hint: t('affiliate.stats.rebateRateHint'),
  },
  {
    label: t('affiliate.stats.invitedUsers'),
    value: formatCount(detail.value?.aff_count ?? 0),
    hint: t('affiliate.stats.invitedUsersHint'),
  },
  {
    label: t('affiliate.stats.availableQuota'),
    value: formatCurrency(detail.value?.aff_quota ?? 0),
    hint: detail.value && detail.value.aff_frozen_quota > 0
      ? t('affiliate.stats.frozenQuotaLine', { amount: formatCurrency(detail.value.aff_frozen_quota) })
      : t('affiliate.stats.availableQuotaHint'),
  },
  {
    label: t('affiliate.stats.totalQuota'),
    value: formatCurrency(detail.value?.aff_history_quota ?? 0),
    hint: t('affiliate.stats.totalQuotaHint'),
  },
])

const proofCards = computed<Array<{ icon: AffiliateIconName; title: string; description: string }>>(() => [
  {
    icon: 'gift',
    title: t('affiliate.friendBenefits.trialTitle'),
    description: t('affiliate.friendBenefits.trialDescription'),
  },
  {
    icon: 'dollar',
    title: t('affiliate.friendBenefits.valueTitle'),
    description: t('affiliate.friendBenefits.valueDescription'),
  },
  {
    icon: 'shield',
    title: t('affiliate.friendBenefits.stabilityTitle'),
    description: t('affiliate.friendBenefits.stabilityDescription'),
  },
  {
    icon: 'sparkles',
    title: t('affiliate.friendBenefits.rangeTitle'),
    description: t('affiliate.friendBenefits.rangeDescription'),
  },
])

const audienceCards = computed<Array<{ icon: AffiliateIconName; title: string; description: string; copy: string }>>(() => [
  {
    icon: 'terminal',
    title: t('affiliate.audiences.developerTitle'),
    description: t('affiliate.audiences.developerDescription'),
    copy: t('affiliate.audiences.developerCopy', { link: inviteLink.value }),
  },
  {
    icon: 'cpu',
    title: t('affiliate.audiences.heavyUserTitle'),
    description: t('affiliate.audiences.heavyUserDescription'),
    copy: t('affiliate.audiences.heavyUserCopy', { link: inviteLink.value }),
  },
  {
    icon: 'lightbulb',
    title: t('affiliate.audiences.newcomerTitle'),
    description: t('affiliate.audiences.newcomerDescription'),
    copy: t('affiliate.audiences.newcomerCopy', { link: inviteLink.value }),
  },
  {
    icon: 'chat',
    title: t('affiliate.audiences.groupTitle'),
    description: t('affiliate.audiences.groupDescription'),
    copy: t('affiliate.audiences.groupCopy', { link: inviteLink.value }),
  },
])

const howItWorks = computed(() => {
  const items = [
    t('affiliate.rules.line1'),
    t('affiliate.rules.line2', { rate: `${formattedRebateRate.value}%` }),
    rebateDurationRule.value,
    t('affiliate.rules.line4'),
  ]
  if (detail.value && detail.value.aff_frozen_quota > 0) {
    items.push(t('affiliate.rules.line5'))
  }
  return items
})

const promoShareText = computed(() => t('affiliate.promo.shareText', {
  link: inviteLink.value,
}))

const bindBonusAmount = computed(() => Math.max(0, detail.value?.bind_bonus_amount ?? 0))
const rebateDurationDays = computed(() => Math.max(0, Math.trunc(detail.value?.rebate_duration_days ?? 0)))
const rebateDurationRule = computed(() => {
  if (rebateDurationDays.value <= 0) {
    return t('affiliate.rules.durationPermanent')
  }
  return t('affiliate.rules.durationLimited', { days: rebateDurationDays.value })
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

function formatCurrency(value: number): string {
  return formatSettlementAmount(value, 2)
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function copyPromoText(): Promise<void> {
  if (!promoShareText.value) return
  await copyToClipboard(promoShareText.value, t('affiliate.promo.copied'))
}

async function copyAudienceText(text: string): Promise<void> {
  if (!text) return
  await copyToClipboard(text, t('affiliate.audiences.copied'))
}

async function bindInviter(): Promise<void> {
  const code = bindCode.value.trim()
  if (!code || binding.value) return

  binding.value = true
  const bonusAmount = bindBonusAmount.value
  try {
    detail.value = await userAPI.bindAffiliateInviter(code)
    bindCode.value = ''
    appStore.showSuccess(
      bonusAmount > 0
        ? t('affiliate.bind.successWithBonus', { amount: formatCurrency(bonusAmount) })
        : t('affiliate.bind.success')
    )
    await authStore.refreshUser().catch(() => undefined)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.bindFailed'), {
      AFFILIATE_BIND_WINDOW_EXPIRED: t('affiliate.bind.errors.AFFILIATE_BIND_WINDOW_EXPIRED'),
      AFFILIATE_CODE_INVALID: t('affiliate.bind.errors.AFFILIATE_CODE_INVALID'),
      AFFILIATE_ALREADY_BOUND: t('affiliate.bind.errors.AFFILIATE_ALREADY_BOUND'),
    }))
  } finally {
    binding.value = false
  }
}

async function claimBindBonus(): Promise<void> {
  if (!detail.value?.can_claim_bind_bonus || bindBonusAmount.value <= 0 || claimingBindBonus.value) return

  claimingBindBonus.value = true
  const amount = bindBonusAmount.value
  try {
    const resp = await userAPI.claimAffiliateBindBonus()
    detail.value = resp.detail
    appStore.showSuccess(t('affiliate.bind.claimSuccess', { amount: formatCurrency(amount) }))
    await authStore.refreshUser().catch(() => undefined)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.bindFailed'), {
      AFFILIATE_BIND_WINDOW_EXPIRED: t('affiliate.bind.errors.AFFILIATE_BIND_WINDOW_EXPIRED'),
      AFFILIATE_BIND_BONUS_UNAVAILABLE: t('affiliate.bind.errors.AFFILIATE_BIND_BONUS_UNAVAILABLE'),
      AFFILIATE_BIND_BONUS_ALREADY_CLAIMED: t('affiliate.bind.errors.AFFILIATE_BIND_BONUS_ALREADY_CLAIMED'),
    }))
  } finally {
    claimingBindBonus.value = false
  }
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
