<template>
  <section class="space-y-3">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between lg:flex-col lg:items-start xl:flex-row xl:items-end">
      <div class="min-w-0 space-y-1">
        <p class="text-xs font-semibold text-[var(--apple-blue)]">
          {{ t('dashboard.quickActionsKicker') }}
        </p>
        <h2 class="text-base font-semibold text-[var(--apple-text)]">
          {{ t('dashboard.quickActions') }}
        </h2>
        <p class="text-sm leading-6 text-[var(--apple-muted)]">
          {{ t('dashboard.quickActionsDescription') }}
        </p>
      </div>
      <span class="badge badge-gray w-fit">{{ t('dashboard.quickActionsBadge') }}</span>
    </div>

    <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
      <router-link
        v-for="item in quickLinks"
        :key="item.to"
        :to="item.to"
        class="card group flex items-start gap-3 p-3 transition-colors hover:border-[color:var(--apple-border)] hover:bg-[color-mix(in_srgb,var(--apple-blue)_5%,var(--apple-surface))]"
      >
        <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--apple-radius)] bg-[var(--apple-surface-elevated)] text-[var(--apple-blue)] ring-1 ring-[color:var(--apple-border)]">
          <Icon :name="item.icon" size="sm" :stroke-width="2" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-start justify-between gap-2">
            <p class="truncate text-sm font-semibold text-[var(--apple-text)] transition-colors group-hover:text-[var(--apple-blue)]">
              {{ item.title }}
            </p>
            <Icon
              name="arrowRight"
              size="xs"
              class="mt-0.5 shrink-0 text-[var(--apple-muted-2)] transition-transform group-hover:translate-x-0.5 group-hover:text-[var(--apple-blue)]"
            />
          </div>
          <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
            {{ item.description }}
          </p>
        </div>
      </router-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

type IconName = InstanceType<typeof Icon>['$props']['name']

const quickLinks = computed<Array<{
  icon: IconName
  title: string
  description: string
  to: string
}>>(() => [
  { icon: 'key', title: t('dashboard.quickLinks.keys.title'), description: t('dashboard.quickLinks.keys.description'), to: '/keys' },
  { icon: 'chart', title: t('dashboard.quickLinks.usage.title'), description: t('dashboard.quickLinks.usage.description'), to: '/usage' },
  { icon: 'creditCard', title: t('dashboard.quickLinks.plans.title'), description: t('dashboard.quickLinks.plans.description'), to: '/subscriptions' },
  { icon: 'clipboard', title: t('dashboard.quickLinks.orders.title'), description: t('dashboard.quickLinks.orders.description'), to: '/orders' },
  { icon: 'user', title: t('dashboard.quickLinks.profile.title'), description: t('dashboard.quickLinks.profile.description'), to: '/profile' },
  { icon: 'chatBubble', title: t('dashboard.quickLinks.support.title'), description: t('dashboard.quickLinks.support.description'), to: '/tickets' },
  { icon: 'gift', title: t('dashboard.quickLinks.redeem.title'), description: t('dashboard.quickLinks.redeem.description'), to: '/redeem' },
  { icon: 'users', title: t('dashboard.quickLinks.affiliate.title'), description: t('dashboard.quickLinks.affiliate.description'), to: '/affiliate' },
])

</script>
