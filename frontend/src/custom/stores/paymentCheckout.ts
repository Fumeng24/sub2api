import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { paymentAPI } from '@/api/payment'
import { CARD_CODE_PURCHASE_URL } from '@/custom/payment/providerConfig'
import { usePaymentStore } from '@/stores/payment'
import type { CheckoutInfoResponse } from '@/types/payment'

export const usePaymentCheckoutStore = defineStore('paymentCheckout', () => {
  const paymentStore = usePaymentStore()
  const checkoutInfo = ref<CheckoutInfoResponse | null>(null)
  const checkoutInfoLoading = ref(false)
  const checkoutInfoLoaded = ref(false)

  let requestGeneration = 0
  let checkoutInfoRequest: Promise<CheckoutInfoResponse | null> | null = null

  const checkoutPlans = computed(() => checkoutInfo.value?.plans ?? [])
  const hasCheckoutSubscriptionPlans = computed(() => checkoutPlans.value.length > 0)
  const canRechargeBalance = computed(() => {
    if (!checkoutInfo.value) return false
    if (checkoutInfo.value.balance_recharge_available !== undefined) {
      return checkoutInfo.value.balance_recharge_available === true
    }
    return checkoutInfo.value.balance_disabled !== true
  })
  const hasCardCodePurchase = computed(() => CARD_CODE_PURCHASE_URL.trim() !== '')
  const canAccessPurchase = computed(
    () => hasCardCodePurchase.value || canRechargeBalance.value || hasCheckoutSubscriptionPlans.value,
  )

  async function fetchCheckoutInfo(force = false): Promise<CheckoutInfoResponse | null> {
    if (checkoutInfoLoaded.value && !force) return checkoutInfo.value
    if (checkoutInfoRequest && !force) return checkoutInfoRequest

    const currentGeneration = ++requestGeneration
    checkoutInfoLoading.value = true
    const requestPromise = paymentAPI
      .getCheckoutInfo()
      .then((response) => {
        const checkout = response.data
        if (currentGeneration === requestGeneration) {
          checkoutInfo.value = checkout
          checkoutInfoLoaded.value = true
          paymentStore.plans = checkout.plans || []
          if (paymentStore.config) {
            paymentStore.config = {
              ...paymentStore.config,
              balance_disabled: checkout.balance_disabled,
              balance_recharge_unlock_threshold:
                checkout.balance_recharge_unlock_threshold
                ?? paymentStore.config.balance_recharge_unlock_threshold,
              balance_recharge_multiplier: checkout.balance_recharge_multiplier,
              help_text: checkout.help_text,
              help_image_url: checkout.help_image_url,
              stripe_publishable_key: checkout.stripe_publishable_key,
            }
          }
        }
        return checkout
      })
      .catch((error: unknown) => {
        console.error('[payment] Failed to fetch checkout info:', error)
        return null
      })
      .finally(() => {
        if (checkoutInfoRequest === requestPromise) {
          checkoutInfoLoading.value = false
          checkoutInfoRequest = null
        }
      })

    checkoutInfoRequest = requestPromise
    return requestPromise
  }

  function clear() {
    requestGeneration++
    checkoutInfoRequest = null
    checkoutInfo.value = null
    checkoutInfoLoaded.value = false
    checkoutInfoLoading.value = false
    paymentStore.clearCurrentOrder()
  }

  return {
    checkoutInfo,
    checkoutInfoLoading,
    checkoutInfoLoaded,
    checkoutPlans,
    hasCheckoutSubscriptionPlans,
    canRechargeBalance,
    hasCardCodePurchase,
    canAccessPurchase,
    fetchCheckoutInfo,
    clear,
  }
})
