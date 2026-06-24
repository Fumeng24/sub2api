<template>
  <AppLayout>
    <div
      data-testid="profile-shell"
      class="mx-auto max-w-[950px] space-y-5"
    >
      <div class="page-header mb-0">
        <h1 class="page-title">
          {{ t('profile.title') }}
        </h1>
        <p class="page-description max-w-2xl leading-6">
          {{ t('profile.description') }}
        </p>
      </div>

      <div class="grid gap-2 sm:grid-cols-3">
        <div
          v-for="item in profileTrustItems"
          :key="item.title"
          class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-4 py-3"
        >
          <div class="flex items-center gap-2 text-sm font-semibold text-[var(--apple-text)]">
            <Icon :name="item.icon" size="sm" class="text-[var(--apple-muted)]" />
            <span>{{ item.title }}</span>
          </div>
          <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
            {{ item.description }}
          </p>
        </div>
      </div>

      <ProfileInfoCard
        :user="user"
        :linuxdo-enabled="linuxdoOAuthEnabled"
        :dingtalk-enabled="dingtalkOAuthEnabled"
        :oidc-enabled="oidcOAuthEnabled"
        :oidc-provider-name="oidcOAuthProviderName"
        :wechat-enabled="wechatOAuthEnabled"
        :wechat-open-enabled="wechatOAuthOpenEnabled"
        :wechat-mp-enabled="wechatOAuthMPEnabled"
      />

      <div
        v-if="contactInfo"
        class="card p-5"
      >
        <div class="flex min-w-0 items-center gap-4">
          <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-3 text-[var(--apple-blue)]">
            <Icon name="chat" size="lg" />
          </div>
          <div class="min-w-0">
            <h3 class="font-semibold text-[var(--apple-text)]">
              {{ t('common.contactSupport') }}
            </h3>
            <p class="break-words text-sm font-medium text-[var(--apple-muted)]">{{ contactInfo }}</p>
          </div>
        </div>
      </div>

      <ProfilePasswordForm />

      <ProfileBalanceNotifyCard
        v-if="user && balanceLowNotifyEnabled"
        :enabled="user.balance_notify_enabled ?? true"
        :threshold="user.balance_notify_threshold"
        :extra-emails="user.balance_notify_extra_emails ?? []"
        :system-default-threshold="systemDefaultThreshold"
        :user-email="user.email"
      />

      <ProfileTotpCard />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import { isWeChatWebOAuthEnabled } from '@/api/auth'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { setSettlementCnyPerCredit } from '@/composables/useSettlementCurrency'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const profileTrustItems = computed<Array<{ icon: 'shield' | 'link' | 'lock'; title: string; description: string }>>(() => [
  {
    icon: 'shield',
    title: t('profile.trust.privacyBoundary'),
    description: t('profile.trust.privacyBoundaryDesc')
  },
  {
    icon: 'link',
    title: t('profile.trust.auditableBindings'),
    description: t('profile.trust.auditableBindingsDesc')
  },
  {
    icon: 'lock',
    title: t('profile.trust.accountSecurity'),
    description: t('profile.trust.accountSecurityDesc')
  }
])

const contactInfo = ref('')
const balanceLowNotifyEnabled = ref(false)
const systemDefaultThreshold = ref(0)
const linuxdoOAuthEnabled = ref(false)
const dingtalkOAuthEnabled = ref(false)
const wechatOAuthEnabled = ref(false)
const wechatOAuthOpenEnabled = ref<boolean | undefined>(undefined)
const wechatOAuthMPEnabled = ref<boolean | undefined>(undefined)
const oidcOAuthEnabled = ref(false)
const oidcOAuthProviderName = ref('OIDC')

onMounted(async () => {
  const profileRefresh = authStore.refreshUser().catch((error) => {
    console.error('Failed to refresh profile:', error)
  })

  const settingsLoad = appStore.fetchPublicSettings()
    .then((settings) => {
      if (!settings) {
        return
      }
      contactInfo.value = settings.contact_info || ''
      setSettlementCnyPerCredit(settings.payment_balance_recharge_multiplier)
      balanceLowNotifyEnabled.value = settings.balance_low_notify_enabled ?? false
      systemDefaultThreshold.value = settings.balance_low_notify_threshold ?? 0
      linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled ?? false
      dingtalkOAuthEnabled.value = settings.dingtalk_oauth_enabled ?? false
      wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
      wechatOAuthOpenEnabled.value = typeof settings.wechat_oauth_open_enabled === 'boolean'
        ? settings.wechat_oauth_open_enabled
        : undefined
      wechatOAuthMPEnabled.value = typeof settings.wechat_oauth_mp_enabled === 'boolean'
        ? settings.wechat_oauth_mp_enabled
        : undefined
      oidcOAuthEnabled.value = settings.oidc_oauth_enabled ?? false
      oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
    })
    .catch((error) => {
      console.error('Failed to load settings:', error)
    })

  await Promise.all([profileRefresh, settingsLoad])
})
</script>
