<template>
  <UserPageHero
    :kicker="t('profile.gateway.kicker')"
    :title="t('profile.title')"
    :description="t('profile.description')"
  >
    <template #body>
      <UserSummaryStats
        class="mt-5"
        :items="profileSummaryItems"
        grid-class="grid-cols-1 sm:grid-cols-3"
      />
    </template>

    <template #aside>
      <aside class="profile-security-panel rounded-lg border p-4">
        <p class="text-xs font-semibold uppercase text-[var(--apple-muted-2)]">
          {{ t('profile.gateway.securityScope') }}
        </p>
        <p class="mt-1 text-base font-semibold text-[var(--apple-text)]">
          {{ t('profile.gateway.panelTitle') }}
        </p>
        <div class="mt-4 space-y-3">
          <div
            v-for="item in profileTrustItems"
            :key="item.title"
            class="flex gap-3 border-t border-[color:var(--apple-border-soft)] pt-3"
          >
            <Icon
              :name="item.icon"
              size="sm"
              class="mt-0.5 shrink-0 text-[var(--apple-blue)]"
            />
            <div class="min-w-0">
              <p class="text-sm font-semibold text-[var(--apple-text)]">
                {{ item.title }}
              </p>
              <p class="mt-0.5 text-xs leading-5 text-[var(--apple-muted)]">
                {{ item.description }}
              </p>
            </div>
          </div>
        </div>
      </aside>
    </template>
  </UserPageHero>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import UserSummaryStats from '@/custom/user/UserSummaryStats.vue'
import type { User } from '@/types'

defineOptions({ name: 'WegooProfileHero' })

const props = defineProps<{
  user: User | null
}>()

const { t } = useI18n()
type IconName = InstanceType<typeof Icon>['$props']['name']

const profileTrustItems = computed<Array<{
  icon: 'shield' | 'link' | 'lock'
  title: string
  description: string
}>>(() => [
  {
    icon: 'shield',
    title: t('profile.trust.privacyBoundary'),
    description: t('profile.trust.privacyBoundaryDesc'),
  },
  {
    icon: 'link',
    title: t('profile.trust.auditableBindings'),
    description: t('profile.trust.auditableBindingsDesc'),
  },
  {
    icon: 'lock',
    title: t('profile.trust.accountSecurity'),
    description: t('profile.trust.accountSecurityDesc'),
  },
])

const profileRoleLabel = computed(() => {
  if (props.user?.role === 'admin') return t('profile.administrator')
  if (props.user?.role === 'support') return t('profile.gateway.supportRole')
  return t('profile.user')
})

const profileStatusLabel = computed(() => (
  props.user?.status === 'active' ? t('common.active') : t('common.disabled')
))

const profileSummaryItems = computed<Array<{
  icon: IconName
  iconClass: string
  label: string
  value: string
  meta: string
}>>(() => [
  {
    icon: 'userCircle',
    iconClass: 'text-[var(--apple-blue)]',
    label: t('profile.gateway.accountId'),
    value: props.user ? `#${props.user.id}` : '-',
    meta: props.user?.username || '-',
  },
  {
    icon: 'shield',
    iconClass: props.user?.status === 'active'
      ? 'text-[var(--apple-success)]'
      : 'text-[var(--apple-danger)]',
    label: t('profile.status'),
    value: profileStatusLabel.value,
    meta: props.user?.email || '-',
  },
  {
    icon: 'lock',
    iconClass: 'text-amber-300',
    label: t('profile.role'),
    value: profileRoleLabel.value,
    meta: t('profile.gateway.securityMeta'),
  },
])
</script>

<style scoped>
.profile-security-panel {
  background:
    linear-gradient(180deg, rgb(255 255 255 / 0.055), transparent),
    #070a10;
  border-color: var(--apple-border-soft);
  box-shadow: var(--apple-shadow-sm);
}
</style>
