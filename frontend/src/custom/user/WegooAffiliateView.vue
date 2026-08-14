<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <UserPageHero
          :kicker="t('affiliate.hero.kicker')"
          :title="t('affiliate.hero.title')"
          :description="t('affiliate.hero.description')"
        >
          <template #body>
              <div class="mt-6 flex flex-col gap-3 sm:flex-row">
                <button class="btn btn-primary w-full sm:w-auto" @click="copyPromoText">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.promo.copyButton') }}</span>
                </button>
                <button class="btn btn-secondary w-full sm:w-auto" @click="copyInviteLink">
                  <Icon name="link" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
              <p class="mt-3 text-sm text-[var(--apple-muted)]">
                {{ t('affiliate.hero.shareHint') }}
              </p>
          </template>

          <template #aside>
            <UserSummaryStats :items="affiliateStats" grid-class="grid-cols-1 sm:grid-cols-2 lg:grid-cols-1" />
          </template>
        </UserPageHero>

        <div class="grid gap-5 2xl:grid-cols-[minmax(0,1.08fr)_minmax(340px,0.92fr)]">
          <section class="card min-w-0 p-5 sm:p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <p class="text-xs font-semibold uppercase tracking-normal text-[var(--apple-muted)]">
                  {{ t('affiliate.sharePanel.kicker') }}
                </p>
                <h2 class="mt-2 text-xl font-semibold tracking-normal text-[var(--apple-text)]">
                  {{ t('affiliate.sharePanel.title') }}
                </h2>
                <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
                  {{ t('affiliate.sharePanel.description') }}
                </p>
              </div>
              <button class="btn btn-primary w-full sm:w-auto sm:shrink-0" @click="copyPromoText">
                <Icon name="copy" size="sm" />
                <span>{{ t('affiliate.promo.copyButton') }}</span>
              </button>
            </div>

            <div class="mt-5 grid gap-3 sm:grid-cols-3">
              <dl
                v-for="item in shareSummaryStats"
                :key="item.label"
                class="min-w-0 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-3"
              >
                <dt class="truncate text-xs font-medium text-[var(--apple-muted)]">
                  {{ item.label }}
                </dt>
                <dd class="mt-1 truncate text-base font-semibold text-[var(--apple-text)]">
                  {{ item.value }}
                </dd>
              </dl>
            </div>

            <div class="mt-5 space-y-4">
              <div class="min-w-0 space-y-2">
                <p class="text-sm font-semibold text-[var(--apple-text)]">{{ t('affiliate.yourCode') }}</p>
                <div class="flex min-w-0 flex-col gap-2 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] px-3 py-3 md:flex-row md:items-center">
                  <code class="block min-w-0 flex-1 select-all break-all font-mono text-sm font-semibold tracking-normal text-[var(--apple-text)] md:truncate md:break-normal" :title="detail.aff_code" v-text="detail.aff_code"></code>
                  <button class="btn btn-secondary btn-sm w-full md:w-auto md:shrink-0" @click="copyCode">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyCode') }}</span>
                  </button>
                </div>
              </div>

              <div class="min-w-0 space-y-2">
                <p class="text-sm font-semibold text-[var(--apple-text)]">{{ t('affiliate.inviteLink') }}</p>
                <div class="flex min-w-0 flex-col gap-2 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] px-3 py-3 md:flex-row md:items-center">
                  <code class="block min-w-0 flex-1 select-all break-all font-mono text-sm text-[var(--apple-muted)] md:truncate md:break-normal" :title="inviteLink" v-text="inviteLink"></code>
                  <button class="btn btn-secondary btn-sm w-full md:w-auto md:shrink-0" @click="copyInviteLink">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyLink') }}</span>
                  </button>
                </div>
              </div>
            </div>

          <div
            v-if="detail.inviter_id == null && detail.can_bind_inviter"
            class="mt-5 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4"
          >
            <form
              class="grid gap-4 xl:grid-cols-[minmax(280px,1fr)_minmax(320px,420px)] xl:items-end"
              @submit.prevent="bindInviter"
            >
              <div class="min-w-0">
                <h4 class="text-sm font-semibold text-[var(--apple-text)]">
                  {{ t('affiliate.bind.title') }}
                </h4>
                <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
                  {{ t('affiliate.bind.description') }}
                </p>
                <p
                  v-if="bindBonusAmount > 0"
                  class="mt-2 inline-flex rounded-full border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 py-1 text-xs font-semibold text-[var(--apple-blue)]"
                >
                  {{ t('affiliate.bind.bonusHint', { amount: formatCurrency(bindBonusAmount) }) }}
                </p>
              </div>
              <div class="flex min-w-0 flex-col gap-2 sm:flex-row xl:w-full">
                <input
                  v-model="bindCode"
                  type="text"
                  class="input min-w-0 flex-1 uppercase"
                  :placeholder="t('affiliate.bind.codePlaceholder')"
                  :disabled="binding"
                  autocomplete="off"
                />
                <button
                  type="submit"
                  class="btn btn-primary w-full sm:w-auto sm:shrink-0"
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
            class="mt-5 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4"
          >
            <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <h4 class="text-sm font-semibold text-[var(--apple-text)]">
                  {{ t('affiliate.bind.claimTitle') }}
                </h4>
                <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
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

            <div class="mt-5 overflow-hidden rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4">
              <p class="text-sm font-semibold text-[var(--apple-text)]">
                {{ t('affiliate.promo.previewTitle') }}
              </p>
              <div class="mt-3 break-words rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 text-sm leading-6 text-[var(--apple-muted)]">
                {{ promoShareText }}
              </div>
            </div>
          </section>

          <section class="card min-w-0 p-5 sm:p-6">
            <p class="text-xs font-semibold uppercase tracking-normal text-[var(--apple-muted)]">
              {{ t('affiliate.friendBenefits.kicker') }}
            </p>
            <h2 class="mt-2 text-xl font-semibold tracking-normal text-[var(--apple-text)]">
              {{ t('affiliate.friendBenefits.title') }}
            </h2>
            <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
              {{ t('affiliate.friendBenefits.description') }}
            </p>

            <div class="mt-5 grid gap-3 sm:grid-cols-2">
              <div
                v-for="item in proofCards"
                :key="item.title"
                class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4"
              >
                <div class="flex h-9 w-9 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-blue)]">
                  <Icon :name="item.icon" size="sm" :stroke-width="2" />
                </div>
                <h3 class="mt-3 text-sm font-semibold text-[var(--apple-text)]">{{ item.title }}</h3>
                <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">{{ item.description }}</p>
              </div>
            </div>
          </section>
        </div>

        <section class="card min-w-0 p-5 sm:p-6">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p class="text-xs font-semibold uppercase tracking-normal text-[var(--apple-muted)]">
                {{ t('affiliate.audiences.kicker') }}
              </p>
              <h2 class="mt-2 text-xl font-semibold tracking-normal text-[var(--apple-text)]">
                {{ t('affiliate.audiences.title') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
                {{ t('affiliate.audiences.description') }}
              </p>
            </div>
          </div>

          <div class="mt-5 grid gap-4 lg:grid-cols-2">
            <article
              v-for="item in audienceCards"
              :key="item.title"
              class="min-w-0 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-4"
            >
              <div class="flex items-start gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-blue)]">
                  <Icon :name="item.icon" size="sm" :stroke-width="2" />
                </div>
                <div class="min-w-0">
                  <h3 class="text-base font-semibold text-[var(--apple-text)]">{{ item.title }}</h3>
                  <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">{{ item.description }}</p>
                </div>
              </div>
              <div class="mt-4 rounded-lg border border-dashed border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-3 text-sm leading-6 text-[var(--apple-muted)]">
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
            <p class="text-sm font-semibold text-[var(--apple-text)]">{{ t('affiliate.rules.title') }}</p>
            <div class="mt-4 grid gap-3">
              <div
                v-for="(item, index) in howItWorks"
                :key="item"
                class="flex gap-3 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-3"
              >
                <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[var(--apple-blue)] text-sm font-semibold text-white">
                  {{ index + 1 }}
                </span>
                <p class="min-w-0 text-sm leading-6 text-[var(--apple-muted)]">{{ item }}</p>
              </div>
            </div>
          </div>

          <div class="card min-w-0 p-5 sm:p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-[var(--apple-text)]">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-[var(--apple-muted)]">{{ t('affiliate.transfer.description') }}</p>
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
          <h3 class="text-base font-semibold text-[var(--apple-text)]">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-lg border border-dashed border-[color:var(--apple-border)] p-6 text-center text-sm text-[var(--apple-muted)]">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="table w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr>
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
                >
                  <td class="px-3 py-3">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3">{{ formatDateTime(item.created_at) || '-' }}</td>
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
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import UserSummaryStats from '@/custom/user/UserSummaryStats.vue'
import userAPI from '@/api/user'
import { bindAffiliateInviter, claimAffiliateBindBonus } from '@/custom/user/affiliateApi'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
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

const shareSummaryStats = computed(() => affiliateStats.value.slice(0, 3))

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
    detail.value = await bindAffiliateInviter(code)
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
    const resp = await claimAffiliateBindBonus()
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
