<template>
  <AppLayout>
    <div class="mx-auto max-w-lg space-y-6 py-8">
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
      </div>

      <div v-else-if="errorMessage" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-8 text-center shadow-sm">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-lg border border-[color:color-mix(in_srgb,var(--apple-danger)_30%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_10%,var(--apple-surface))]">
          <Icon name="exclamationCircle" size="xl" class="text-[var(--apple-danger)]" />
        </div>
        <h3 class="text-lg font-semibold text-[var(--apple-text)]">{{ t('payment.airwallexLoadFailed') }}</h3>
        <p class="mt-2 text-sm text-[var(--apple-muted)]">{{ errorMessage }}</p>
        <button class="btn btn-primary mt-6" @click="router.push('/purchase')">{{ t('payment.result.backToRecharge') }}</button>
      </div>

      <div v-else class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6 shadow-sm">
        <div class="flex flex-col items-center space-y-4 py-4">
          <div class="h-10 w-10 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
          <p class="text-center text-sm leading-6 text-[var(--apple-muted)]">{{ t('payment.airwallexRedirecting') }}</p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  readPaymentRecoverySnapshot,
  type PaymentRecoverySnapshot,
} from '@/custom/payment/paymentFlow'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const loading = ref(true)
const errorMessage = ref('')

const AIRWALLEX_SUPPORTED_LOCALES = [
  'en',
  'zh',
  'id',
  'ms',
  'ja',
  'ko',
  'ar',
  'fr',
  'es',
  'nl',
  'nl-NL',
  'de',
  'it',
  'zh-HK',
  'pl',
  'fi',
  'ru',
  'da',
  'sv',
  'pt',
  'ro',
] as const

type AirwallexCheckoutLocale = (typeof AIRWALLEX_SUPPORTED_LOCALES)[number]

function normalizeAirwallexLocale(value: unknown): AirwallexCheckoutLocale | undefined {
  const localeCode = String(value || '').trim()
  if (AIRWALLEX_SUPPORTED_LOCALES.includes(localeCode as AirwallexCheckoutLocale)) {
    return localeCode as AirwallexCheckoutLocale
  }

  const languageCode = localeCode.split('-')[0]
  if (AIRWALLEX_SUPPORTED_LOCALES.includes(languageCode as AirwallexCheckoutLocale)) {
    return languageCode as AirwallexCheckoutLocale
  }

  return undefined
}

function queryString(key: string): string {
  const value = route.query[key]
  if (Array.isArray(value)) return value[0] || ''
  return typeof value === 'string' ? value : ''
}

function buildSuccessUrl(snapshot: PaymentRecoverySnapshot): string {
  const url = new URL('/payment/result', window.location.origin)
  const orderId = queryString('order_id')
  const outTradeNo = queryString('out_trade_no')
  const resumeToken = queryString('resume_token')

  if (orderId || snapshot.orderId > 0) url.searchParams.set('order_id', orderId || String(snapshot.orderId))
  if (outTradeNo || snapshot.outTradeNo) url.searchParams.set('out_trade_no', outTradeNo || snapshot.outTradeNo)
  if (resumeToken || snapshot.resumeToken) url.searchParams.set('resume_token', resumeToken || snapshot.resumeToken)
  return url.toString()
}

function restoreAirwallexSnapshot(): PaymentRecoverySnapshot | null {
  if (typeof window === 'undefined') {
    return null
  }

  const orderId = Number(queryString('order_id')) || 0
  const outTradeNo = queryString('out_trade_no')
  const resumeToken = queryString('resume_token')
  const snapshot = readPaymentRecoverySnapshot(
    window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY),
    resumeToken ? { resumeToken } : {},
  )

  if (!snapshot || snapshot.paymentType !== 'airwallex') {
    return null
  }
  if (orderId > 0 && snapshot.orderId !== orderId) {
    return null
  }
  if (outTradeNo && snapshot.outTradeNo !== outTradeNo) {
    return null
  }
  if (!snapshot.intentId || !snapshot.clientSecret) {
    return null
  }
  return snapshot
}

onMounted(async () => {
  const snapshot = restoreAirwallexSnapshot()
  const checkoutLocale = normalizeAirwallexLocale(locale.value)

  if (!snapshot) {
    loading.value = false
    errorMessage.value = t('payment.airwallexMissingParams')
    return
  }

  try {
    const airwallex = await import('@airwallex/components-sdk')
    const result = await airwallex.init({
      env: snapshot.paymentEnv === 'prod' ? 'prod' : 'demo',
      enabledElements: ['payments'],
      locale: checkoutLocale,
    })

    loading.value = false
    const checkoutOptions = {
      intent_id: snapshot.intentId,
      client_secret: snapshot.clientSecret,
      currency: snapshot.currency || 'CNY',
      country_code: snapshot.countryCode || 'CN',
      successUrl: buildSuccessUrl(snapshot),
    }
    if (!result.payments) {
      throw new Error(t('payment.airwallexLoadFailed'))
    }
    const redirectResult = result.payments.redirectToCheckout(checkoutOptions)

    if (typeof redirectResult === 'string' && redirectResult) {
      window.location.assign(redirectResult)
    }
  } catch (err: unknown) {
    loading.value = false
    errorMessage.value = err instanceof Error && err.message
      ? err.message
      : t('payment.airwallexLoadFailed')
  }
})
</script>
