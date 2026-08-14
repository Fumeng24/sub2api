<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-5">
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-[color:var(--apple-border)] border-t-[color:var(--apple-blue)]"></div>
      </div>
      <template v-else>
        <UserPageHero :kicker="t('payment.gateway.kicker')" :title="paymentPageTitle">
          <template #below>
              <div
                v-if="paymentGatewayDescription || showNativeRechargeLockHint"
                class="mt-5"
              >
                <div class="min-w-0">
                  <p v-if="paymentGatewayDescription" class="max-w-3xl text-sm leading-6 text-[var(--apple-muted)]">
                    {{ paymentGatewayDescription }}
                  </p>
                  <p v-if="showNativeRechargeLockHint" class="max-w-3xl text-sm leading-6 text-[var(--apple-muted)]">
                    {{ nativeRechargeUnlockHint }}
                  </p>
                </div>
              </div>
          </template>
        </UserPageHero>
        <section
          v-if="showRechargeModePicker"
          class="rounded-[var(--apple-radius)] border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-3 shadow-sm sm:p-4"
          aria-labelledby="recharge-method-title"
        >
          <div class="mb-3 flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h2 id="recharge-method-title" class="text-sm font-semibold text-[var(--apple-text)]">
                {{ t('payment.rechargeMethod.title') }}
              </h2>
              <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
                {{ selectedRechargeMode === 'native' ? t('payment.rechargeMethod.nativeDescription') : t('payment.rechargeMethod.cardCodeDescription') }}
              </p>
            </div>
            <Icon name="creditCard" size="md" class="shrink-0 text-[var(--apple-muted)]" />
          </div>
          <div class="grid grid-cols-2 gap-2 rounded-[var(--apple-radius)] bg-[var(--apple-surface-elevated)] p-1">
            <button
              type="button"
              class="flex min-h-16 items-center justify-center gap-2 rounded-[calc(var(--apple-radius)-4px)] px-3 py-2 text-center text-sm transition-colors"
              :class="selectedRechargeMode === 'card_code' ? 'bg-[var(--apple-surface)] text-[var(--apple-text)] shadow-sm' : 'text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'"
              @click="selectCardCodeRechargeMode"
            >
              <Icon name="gift" size="sm" />
              <span class="min-w-0">
                <span class="block font-medium">{{ t('payment.rechargeMethod.cardCodeTitle') }}</span>
                <span class="mt-0.5 block truncate text-[11px] text-[var(--apple-muted-2)]">{{ cardCodeRechargeFeeLabel }}</span>
              </span>
            </button>
            <button
              type="button"
              class="flex min-h-16 items-center justify-center gap-2 rounded-[calc(var(--apple-radius)-4px)] px-3 py-2 text-center text-sm transition-colors"
              :class="[
                selectedRechargeMode === 'native' ? 'bg-[var(--apple-surface)] text-[var(--apple-text)] shadow-sm' : 'text-[var(--apple-muted)]',
                canUseNativeRecharge ? 'hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]' : 'cursor-not-allowed opacity-60'
              ]"
              @click="selectNativeRechargeMode"
            >
              <Icon name="creditCard" size="sm" />
              <span class="min-w-0">
                <span class="block font-medium">{{ t('payment.rechargeMethod.nativeTitle') }}</span>
                <span class="mt-0.5 block truncate text-[11px] text-[var(--apple-muted-2)]">{{ nativeRechargeFeeLabel }}</span>
              </span>
            </button>
          </div>
        </section>
        <!-- Tab Switcher (hide during payment and subscription confirm) -->
        <div v-if="tabs.length > 1 && paymentPhase === 'select' && !selectedPlan" class="flex space-x-1 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface-elevated)] p-1">
          <button v-for="tab in tabs" :key="tab.key"
            class="flex-1 rounded-md px-4 py-2.5 text-sm font-medium transition-all"
            :class="activeTab === tab.key ? 'bg-[var(--apple-surface)] text-[var(--apple-text)] shadow-sm' : 'text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'"
            @click="activeTab = tab.key">{{ tab.label }}</button>
        </div>
        <div v-if="errorMessage && paymentPhase === 'select'" class="rounded-lg border border-[color:color-mix(in_srgb,var(--apple-danger)_24%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-danger)_8%,var(--apple-surface))] px-4 py-3" role="alert">
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="md" :stroke-width="2" class="mt-0.5 shrink-0 text-[var(--apple-danger)]" />
            <div class="min-w-0 flex-1">
              <p class="text-sm font-semibold text-[var(--apple-danger)]">{{ errorMessage }}</p>
              <p class="mt-1 text-xs leading-relaxed text-[var(--apple-muted)]">{{ orderErrorHint }}</p>
            </div>
          </div>
        </div>
        <!-- Payment in progress (shared by recharge and subscription) -->
        <template v-if="paymentPhase === 'paying'">
          <PaymentStatusPanel
            :order-id="paymentState.orderId"
            :qr-code="paymentState.qrCode"
            :expires-at="paymentState.expiresAt"
            :payment-type="paymentState.paymentType"
            :pay-url="paymentState.payUrl"
            :order-type="paymentState.orderType"
            :currency="paymentState.currency || selectedCurrency"
            @done="onPaymentDone"
            @success="onPaymentSuccess"
            @settled="onPaymentSettled"
          />
        </template>
        <!-- Tab content (select phase) -->
        <template v-else>
          <div v-if="tabs.length === 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-6 py-16 text-center shadow-sm">
            <Icon name="gift" size="xl" class="mx-auto mb-3 text-[var(--apple-muted-2)]" />
            <p class="text-[var(--apple-muted)]">{{ t('payment.notAvailable') }}</p>
          </div>
          <!-- Top-up Tab -->
          <template v-else-if="activeTab === 'recharge'">
            <section class="overflow-hidden rounded-[var(--apple-radius)] border border-[color:var(--apple-border)] bg-[var(--apple-surface)] shadow-sm">
              <div v-if="selectedRechargeMode === 'card_code'" class="p-5">
                <div class="space-y-5">
                  <div>
                    <p class="text-sm font-semibold text-[var(--apple-text)]">{{ t('payment.cardCodePurchase.title') }}</p>
                    <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">{{ t('payment.cardCodePurchase.description') }}</p>
                    <button type="button" class="btn btn-primary mt-4 w-full justify-center sm:w-auto" @click="openCardCodePurchase">
                      <Icon name="externalLink" size="sm" />
                      {{ t('payment.cardCodePurchase.action') }}
                    </button>
                  </div>
                  <div class="border-t border-[color:var(--apple-border-soft)] pt-5">
                    <p class="text-sm font-semibold text-[var(--apple-text)]">{{ t('payment.cardCodePurchase.redeemTitle') }}</p>
                    <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">{{ t('payment.cardCodePurchase.redeemDescription') }}</p>
                    <form class="mt-4 space-y-3" @submit.prevent="handleCardCodeRedeem">
                      <div>
                        <label for="payment-redeem-code" class="sr-only">{{ t('payment.cardCodePurchase.redeemLabel') }}</label>
                        <div class="relative">
                          <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                            <Icon name="gift" size="sm" class="text-[var(--apple-muted)]" />
                          </div>
                          <input
                            id="payment-redeem-code"
                            v-model="cardCodeRedeemCode"
                            type="text"
                            class="input py-3 pl-10"
                            :placeholder="t('payment.cardCodePurchase.redeemPlaceholder')"
                            :disabled="cardCodeRedeemSubmitting"
                          />
                        </div>
                      </div>
                      <button
                        type="submit"
                        class="btn btn-secondary w-full justify-center"
                        :disabled="!cardCodeRedeemCode.trim() || cardCodeRedeemSubmitting"
                      >
                        <Icon
                          :name="cardCodeRedeemSubmitting ? 'refresh' : 'checkCircle'"
                          size="sm"
                          :class="cardCodeRedeemSubmitting ? 'animate-spin' : ''"
                        />
                        {{ cardCodeRedeemSubmitting ? t('redeem.redeeming') : t('payment.cardCodePurchase.redeemAction') }}
                      </button>
                    </form>
                    <div v-if="cardCodeRedeemResult" class="mt-3 rounded-[var(--apple-radius)] border border-[color:color-mix(in_srgb,var(--apple-success)_26%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-success)_8%,var(--apple-surface))] px-3 py-2 text-sm text-[var(--apple-text)]">
                      <p class="font-semibold text-[var(--apple-success)]">{{ t('redeem.redeemSuccess') }}</p>
                      <p v-if="cardCodeRedeemResult.type === 'balance'" class="mt-1 text-[var(--apple-muted)]">
                        {{ t('redeem.added') }}: {{ formatCreditedBalance(cardCodeRedeemResult.value) }}
                      </p>
                      <p v-else-if="cardCodeRedeemResult.type === 'concurrency'" class="mt-1 text-[var(--apple-muted)]">
                        {{ t('redeem.added') }}: {{ cardCodeRedeemResult.value }} {{ t('redeem.concurrentRequests') }}
                      </p>
                      <p v-else-if="cardCodeRedeemResult.type === 'subscription'" class="mt-1 text-[var(--apple-muted)]">
                        {{ t('redeem.subscriptionAssigned') }}
                      </p>
                      <p v-if="cardCodeRedeemResult.new_balance !== undefined" class="mt-1 text-[var(--apple-muted)]">
                        {{ t('redeem.newBalance') }}:
                        <span class="font-semibold text-[var(--apple-text)]">{{ formatCreditedBalance(cardCodeRedeemResult.new_balance) }}</span>
                      </p>
                    </div>
                    <p v-if="cardCodeRedeemError" class="mt-3 text-sm leading-5 text-[var(--apple-danger)]">{{ cardCodeRedeemError }}</p>
                  </div>
                </div>
              </div>
              <template v-else-if="selectedRechargeMode === 'native'">
                <div v-if="!canUseNativeRecharge" class="p-5">
                  <div class="rounded-[var(--apple-radius)] border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-4 py-5">
                    <p class="text-sm font-semibold text-[var(--apple-text)]">{{ t('payment.rechargeMethod.nativeUnavailableTitle') }}</p>
                    <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">{{ nativeRechargeUnlockHint }}</p>
                  </div>
                </div>
                <template v-else>
                <div class="p-5">
                  <AmountInput
                    v-model="amount"
                    :amounts="quickRechargeAmounts"
                    :amount-label="formatQuickRechargeAmountLabel"
                    :amount-description="formatQuickRechargeAmountDescription"
                    :disabled-reason="quickRechargeDisabledReason"
                    :min="rechargeMinAmount"
                    :max="rechargeMaxAmount"
                    :prefix="selectedPaymentInputPrefix"
                  />
                  <p v-if="amountError" class="mt-2 text-xs text-[var(--apple-warning)]">{{ amountError }}</p>
                </div>
                <div v-if="methodOptions.length >= 1" class="border-t border-[color:var(--apple-border-soft)] p-5">
                  <PaymentMethodSelector
                    :methods="methodOptions"
                    :selected="selectedMethod"
                    @select="selectedMethod = $event"
                  />
                </div>
                <div v-if="validAmount > 0" class="border-t border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-5">
                  <div class="space-y-2 text-sm">
                    <p class="text-xs font-medium text-[var(--apple-muted-2)]">
                      {{ t('payment.orderPreview') }}
                    </p>
                    <div class="flex justify-between gap-4">
                      <span class="text-[var(--apple-muted)]">{{ t('payment.paymentAmount') }}</span>
                      <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(validAmount) }}</span>
                    </div>
                    <div v-if="feeRate > 0" class="flex justify-between gap-4">
                      <span class="text-[var(--apple-muted)]">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                      <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(feeAmount) }}</span>
                    </div>
                    <div v-if="feeRate > 0" class="flex justify-between gap-4 border-t border-[color:var(--apple-border-soft)] pt-2">
                      <span class="font-medium text-[var(--apple-text)]">{{ t('payment.actualPay') }}</span>
                      <span class="text-lg font-semibold text-[var(--apple-blue)]">{{ formatSelectedPaymentAmount(totalAmount) }}</span>
                    </div>
                    <div class="flex justify-between gap-4" :class="{ 'border-t border-[color:var(--apple-border-soft)] pt-2': feeRate <= 0 }">
                      <span class="text-[var(--apple-muted)]">{{ t('payment.creditedBalance') }}</span>
                      <span class="text-[var(--apple-text)]">{{ formatBalanceCreditAmount(creditedAmount, localeCode) }}</span>
                    </div>
                    <p v-if="showBalanceRechargeRate" class="border-t border-[color:var(--apple-border-soft)] pt-2 text-xs leading-5 text-[var(--apple-muted)]">
                      {{ t('payment.rechargeRatePreview', { cny: balanceRechargeCnyPerCredit.toFixed(2) }) }}
                    </p>
                  </div>
                </div>
                <div class="border-t border-[color:var(--apple-border-soft)] p-5">
                  <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmit || submitting" @click="handleSubmitRecharge">
                    <span v-if="submitting" class="flex items-center justify-center gap-2">
                      <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                      {{ t('common.processing') }}
                    </span>
                    <span v-else>{{ rechargeSubmitLabel }}</span>
                  </button>
                </div>
                </template>
              </template>
            </section>
          </template>
          <!-- Subscribe Tab -->
          <template v-else-if="activeTab === 'subscription'">
            <!-- Subscription confirm (inline, replaces plan list) -->
            <template v-if="selectedPlan">
              <div class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
                <!-- Header: platform badge + plan name -->
                <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <p class="text-xs font-medium text-[var(--apple-muted-2)]">
                      {{ t('payment.selectedPlan') }}
                    </p>
                    <div class="mt-2 flex flex-wrap items-center gap-2">
                      <span class="rounded-md border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2 py-0.5 text-xs font-medium text-[var(--apple-muted)]">
                        {{ platformLabel(selectedPlan.group_platform || '') }}
                      </span>
                      <h3 class="text-lg font-semibold text-[var(--apple-text)]">{{ selectedPlan.name }}</h3>
                    </div>
                  </div>
                  <button class="btn btn-secondary btn-sm w-full shrink-0 sm:w-auto" @click="selectedPlan = null">{{ t('payment.changePlan') }}</button>
                </div>
                <!-- Price -->
                <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                  <span v-if="selectedPlan.original_price" class="text-sm text-[var(--apple-muted-2)] line-through">
                    {{ formatSelectedSubscriptionPaymentAmount(selectedPlan.original_price) }}
                  </span>
                  <span class="text-3xl font-semibold text-[var(--apple-text)]">{{ formatSelectedSubscriptionPaymentAmount(selectedPlan.price) }}</span>
                  <span class="text-sm text-[var(--apple-muted)]">/ {{ planValiditySuffix }}</span>
                </div>
                <!-- Description -->
                <p v-if="selectedPlan.description" class="mt-2 text-sm leading-relaxed text-[var(--apple-muted)]">
                  {{ selectedPlan.description }}
                </p>
                <!-- Rate + Limits grid -->
                <div class="mt-4 grid grid-cols-1 gap-3 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-3 sm:grid-cols-2">
                  <div class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.rate') }}</span>
                    <div class="flex items-baseline">
                      <span class="text-lg font-semibold text-[var(--apple-text)]">×{{ selectedPlan.rate_multiplier ?? 1 }}</span>
                    </div>
                  </div>
                  <div v-if="planHasPeakRate(selectedPlan)" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.peakRate') }}</span>
                    <div class="text-sm font-semibold text-amber-700 dark:text-amber-300">
                      {{ planPeakRateLabel(selectedPlan) }}
                    </div>
                  </div>
                  <div v-if="selectedPlan.daily_limit_usd != null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.dailyLimit') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ formatSettlementAmount(selectedPlan.daily_limit_usd, 2) }}</div>
                  </div>
                  <div v-if="selectedPlan.weekly_limit_usd != null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.weeklyLimit') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ formatSettlementAmount(selectedPlan.weekly_limit_usd, 2) }}</div>
                  </div>
                  <div v-if="selectedPlan.monthly_limit_usd != null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.monthlyLimit') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ formatSettlementAmount(selectedPlan.monthly_limit_usd, 2) }}</div>
                  </div>
                  <div v-if="selectedPlan.daily_limit_usd == null && selectedPlan.weekly_limit_usd == null && selectedPlan.monthly_limit_usd == null" class="min-w-0">
                    <span class="text-xs text-[var(--apple-muted-2)]">{{ t('payment.planCard.quota') }}</span>
                    <div class="text-lg font-semibold text-[var(--apple-text)]">{{ t('payment.planCard.unlimited') }}</div>
                  </div>
                </div>
              </div>
              <div v-if="enabledMethods.length >= 1" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
                <PaymentMethodSelector
                  :methods="subMethodOptions"
                  :selected="selectedMethod"
                  @select="selectedMethod = $event"
                />
              </div>
              <div v-if="selectedPlan.price > 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-5 shadow-sm">
                <div class="space-y-2 text-sm">
                  <p class="text-xs font-medium text-[var(--apple-muted-2)]">
                    {{ t('payment.orderPreview') }}
                  </p>
                  <div class="flex justify-between gap-4">
                    <span class="text-[var(--apple-muted)]">{{ t('payment.amountLabel') }}</span>
                    <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(subPaymentAmount) }}</span>
                  </div>
                  <div v-if="feeRate > 0" class="flex justify-between gap-4">
                    <span class="text-[var(--apple-muted)]">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                    <span class="text-[var(--apple-text)]">{{ formatSelectedPaymentAmount(subFeeAmount) }}</span>
                  </div>
                  <div class="flex justify-between gap-4 border-t border-[color:var(--apple-border-soft)] pt-2">
                    <span class="font-medium text-[var(--apple-text)]">{{ t('payment.actualPay') }}</span>
                    <span class="text-lg font-semibold text-[var(--apple-blue)]">{{ formatSelectedPaymentAmount(subTotalAmount) }}</span>
                  </div>
                </div>
              </div>
              <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmitSubscription || submitting" @click="confirmSubscribe">
                <span v-if="submitting" class="flex items-center justify-center gap-2">
                  <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                  {{ t('common.processing') }}
                </span>
                <span v-else>{{ t('payment.confirmPayment') }} {{ formatSelectedPaymentAmount(subTotalAmount) }}</span>
              </button>
            </template>
            <!-- Plan list -->
            <template v-else>
              <div v-if="checkout.plans.length === 0" class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-6 py-16 text-center shadow-sm">
                <Icon name="gift" size="xl" class="mx-auto mb-3 text-[var(--apple-muted-2)]" />
                <p class="text-[var(--apple-muted)]">{{ t('payment.noPlans') }}</p>
                <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-[var(--apple-muted)]">{{ t('payment.noPlansDesc') }}</p>
              </div>
              <div v-else :class="planGridClass">
                <SubscriptionPlanCard v-for="plan in checkout.plans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlan" />
              </div>
              <!-- Active subscriptions (compact, below plan list) -->
              <div v-if="activeSubscriptions.length > 0">
                <p class="mb-2 text-xs font-medium text-[var(--apple-muted-2)]">{{ t('payment.activeSubscription') }}</p>
                <div class="space-y-2">
                  <div v-for="sub in activeSubscriptions" :key="sub.id"
                    class="flex items-center gap-3 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 py-2 shadow-sm">
                    <div class="h-6 w-1 shrink-0 rounded-full bg-[var(--apple-blue)]" />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <span class="truncate text-xs font-semibold text-[var(--apple-text)]">{{ sub.group?.name || t('payment.groupFallback', { id: sub.group_id }) }}</span>
                        <span class="shrink-0 rounded-full border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-1.5 py-0.5 text-[9px] font-medium text-[var(--apple-muted)]">{{ platformLabel(sub.group?.platform || '') }}</span>
                      </div>
                      <div class="flex flex-wrap gap-x-3 text-[11px] text-[var(--apple-muted-2)]">
                        <span>{{ t('payment.planCard.rate') }}: ×{{ sub.group?.rate_multiplier ?? 1 }}</span>
                        <span v-if="subscriptionHasPeakRate(sub)">{{ t('payment.planCard.peakRate') }}: {{ subscriptionPeakRateLabel(sub) }}</span>
                        <span v-if="sub.group?.daily_limit_usd == null && sub.group?.weekly_limit_usd == null && sub.group?.monthly_limit_usd == null">{{ t('payment.planCard.quota') }}: {{ t('payment.planCard.unlimited') }}</span>
                        <span v-if="sub.expires_at">{{ t('userSubscriptions.daysRemaining', { days: getDaysRemaining(sub.expires_at) }) }}</span>
                        <span v-else>{{ t('userSubscriptions.noExpiration') }}</span>
                      </div>
                    </div>
                    <span class="badge badge-success shrink-0 text-[10px]">{{ t('userSubscriptions.status.active') }}</span>
                  </div>
                </div>
              </div>
            </template>
          </template>
        </template>
        <div v-if="showNativeRechargeHelp" class="rounded-[var(--apple-radius)] border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-4 shadow-sm">
          <div class="flex flex-col items-center gap-3">
            <img v-if="checkout.help_image_url" :src="checkout.help_image_url" alt=""
              class="h-40 max-w-full cursor-pointer rounded-lg object-contain transition-opacity hover:opacity-80"
              @click="previewImage = checkout.help_image_url" />
            <p v-if="checkout.help_text" class="text-center text-sm text-[var(--apple-muted)]">{{ checkout.help_text }}</p>
          </div>
        </div>
      </template>
    </div>
    <!-- Renewal Plan Selection Modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showRenewalModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" @click.self="closeRenewalModal">
          <div class="relative w-full max-w-lg rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-6 shadow-sm">
            <!-- Close button -->
            <button class="absolute right-4 top-4 rounded-lg p-1 text-[var(--apple-muted-2)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]" @click="closeRenewalModal">
              <Icon name="x" size="md" :stroke-width="2" />
            </button>
            <h3 class="mb-4 text-lg font-semibold text-[var(--apple-text)]">{{ t('payment.selectPlan') }}</h3>
            <div class="space-y-4">
              <SubscriptionPlanCard v-for="plan in renewalPlans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlanFromModal" />
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
    <!-- Image Preview Overlay -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="previewImage" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/70" @click="previewImage = ''">
          <img :src="previewImage" alt="" class="max-h-[85vh] max-w-[90vw] rounded-lg object-contain" />
        </div>
      </Transition>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { redeemAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { usePaymentStore } from '@/stores/payment'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAppStore } from '@/stores'
import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'
import { setSettlementCnyPerCredit, useSettlementCurrency } from '@/custom/composables/useSettlementCurrency'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'
import { isMobileDevice } from '@/utils/device'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel, type PeakRateFields } from '@/utils/peak-rate'
import type { SubscriptionPlan, CheckoutInfoResponse, CreateOrderResult, OrderType } from '@/types/payment'
import AppLayout from '@/custom/layout/WegooAppLayout.vue'
import AmountInput from '@/custom/payment/WegooAmountInput.vue'
import PaymentMethodSelector from '@/custom/payment/WegooPaymentMethodSelector.vue'
import { CARD_CODE_PURCHASE_URL, METHOD_ORDER, getPaymentPopupFeatures } from '@/custom/payment/providerConfig'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  buildCreateOrderPayload,
  clearPaymentRecoverySnapshot,
  decidePaymentLaunch,
  getVisibleMethods,
  normalizeVisibleMethod,
  readPaymentRecoverySnapshot,
  type PaymentRecoverySnapshot,
  writePaymentRecoverySnapshot,
} from '@/custom/payment/paymentFlow'
import { platformLabel } from '@/utils/platformColors'
import SubscriptionPlanCard from '@/custom/payment/WegooSubscriptionPlanCard.vue'
import PaymentStatusPanel from '@/custom/payment/WegooPaymentStatusPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import UserPageHero from '@/custom/user/UserPageHero.vue'
import { DEFAULT_PAYMENT_CURRENCY, formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import { paymentAmountPrefix } from '@/custom/payment/currency'
import type { PaymentMethodOption } from '@/custom/payment/WegooPaymentMethodSelector.vue'
import { buildPaymentErrorToastMessage, describePaymentScenarioError } from '@/views/user/paymentUx'
import { formatBalanceCreditAmount, formatCreditedBalance } from '@/custom/payment/orderAmounts'
import { hasWechatResumeQuery, parseWechatResumeRoute, stripWechatResumeQuery } from '@/views/user/paymentWechatResume'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const paymentStore = usePaymentStore()
const paymentCheckoutStore = usePaymentCheckoutStore()
const subscriptionStore = useSubscriptionStore()
const subscriptionCapabilityStore = useSubscriptionCapabilityStore()
const appStore = useAppStore()
const { formatSettlementAmount } = useSettlementCurrency()

const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)

function getDaysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

function subscriptionHasPeakRate(sub: { group?: PeakRateFields | null }): boolean {
  return hasPeakRate(sub.group)
}

function subscriptionPeakRateLabel(sub: { group?: PeakRateFields | null }): string {
  return formatPeakRateWindow(sub.group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

const loading = ref(true)
const submitting = ref(false)
const errorMessage = ref('')
const errorHintMessage = ref('')
const orderErrorHint = computed(() => errorHintMessage.value || t('payment.errors.createOrderHint'))
const activeTab = ref<'recharge' | 'subscription'>('recharge')
const amount = ref<number | null>(null)
const selectedMethod = ref('')
const selectedPlan = ref<SubscriptionPlan | null>(null)
const previewImage = ref('')
const selectedRechargeMode = ref<'card_code' | 'native'>('card_code')
const nativeRechargeLockHintVisible = ref(false)
const cardCodeRedeemCode = ref('')
const cardCodeRedeemSubmitting = ref(false)
const cardCodeRedeemError = ref('')
const cardCodeRedeemResult = ref<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
} | null>(null)

const hasSubscriptionPlans = computed(() => checkout.value.plans.length > 0)
const canRechargeBalance = computed(() => CARD_CODE_PURCHASE_URL.trim() !== '')
const showRechargeModePicker = computed(() =>
  paymentPhase.value === 'select'
    && activeTab.value === 'recharge'
    && !selectedPlan.value
)
const paymentPageTitle = computed(() => hasSubscriptionPlans.value ? t('payment.title') : t('payment.rechargeTitle'))
const paymentGatewayDescription = computed(() =>
  hasSubscriptionPlans.value ? t('payment.gateway.description') : '',
)
const paymentPhase = ref<'select' | 'paying'>('select')

interface CreateOrderOptions {
  openid?: string
  wechatResumeToken?: string
  paymentType?: string
  isResume?: boolean
  mobileQrFallbackAttempted?: boolean
}

interface WeixinJSBridgeLike {
  invoke(
    action: string,
    payload: Record<string, unknown>,
    callback: (result: Record<string, unknown>) => void,
  ): void
}

function emptyPaymentState(): PaymentRecoverySnapshot {
  return {
    orderId: 0,
    amount: 0,
    qrCode: '',
    expiresAt: '',
    paymentType: '',
    payUrl: '',
    outTradeNo: '',
    clientSecret: '',
    intentId: '',
    currency: '',
    countryCode: '',
    paymentEnv: '',
    payAmount: 0,
    orderType: '',
    paymentMode: '',
    resumeToken: '',
    createdAt: 0,
  }
}

function getWeixinJSBridge(): WeixinJSBridgeLike | undefined {
  return (window as Window & { WeixinJSBridge?: WeixinJSBridgeLike }).WeixinJSBridge
}

function waitForWeixinJSBridge(timeoutMs = 4000): Promise<WeixinJSBridgeLike | null> {
  const existing = getWeixinJSBridge()
  if (existing) return Promise.resolve(existing)

  return new Promise((resolve) => {
    let settled = false
    const finish = (bridge: WeixinJSBridgeLike | null) => {
      if (settled) return
      settled = true
      document.removeEventListener('WeixinJSBridgeReady', handleReady)
      document.removeEventListener('onWeixinJSBridgeReady', handleReady)
      window.clearTimeout(timer)
      resolve(bridge)
    }
    const handleReady = () => finish(getWeixinJSBridge() ?? null)
    const timer = window.setTimeout(() => finish(getWeixinJSBridge() ?? null), timeoutMs)
    document.addEventListener('WeixinJSBridgeReady', handleReady, false)
    document.addEventListener('onWeixinJSBridgeReady', handleReady, false)
  })
}

async function invokeWechatJsapiPayment(payload: Record<string, unknown>): Promise<Record<string, unknown>> {
  const bridge = await waitForWeixinJSBridge()
  if (!bridge) {
    throw new Error('WECHAT_JSAPI_UNAVAILABLE')
  }
  return new Promise((resolve) => {
    bridge.invoke('getBrandWCPayRequest', payload, (result) => resolve(result || {}))
  })
}

const paymentState = ref<PaymentRecoverySnapshot>(emptyPaymentState())

function persistRecoverySnapshot(snapshot: PaymentRecoverySnapshot) {
  if (typeof window === 'undefined' || !snapshot.orderId) return
  writePaymentRecoverySnapshot(window.localStorage, snapshot, PAYMENT_RECOVERY_STORAGE_KEY)
}

function removeRecoverySnapshot() {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function resetPayment() {
  paymentPhase.value = 'select'
  paymentState.value = emptyPaymentState()
  removeRecoverySnapshot()
}

async function redirectToPaymentResult(state: PaymentRecoverySnapshot): Promise<void> {
  const query: Record<string, string | undefined> = {}
  if (state.orderId > 0) {
    query.order_id = String(state.orderId)
  }
  if (state.outTradeNo) {
    query.out_trade_no = state.outTradeNo
  }
  if (state.resumeToken) {
    query.resume_token = state.resumeToken
  }
  await router.push({
    path: '/payment/result',
    query,
  })
}

function buildWechatOAuthAuthorizeUrl(
  authorizeUrl: string,
  context: { paymentType: string; orderType: OrderType; planId?: number; orderAmount: number },
): string {
  const normalizedUrl = authorizeUrl.trim()
  if (!normalizedUrl || typeof window === 'undefined') {
    return normalizedUrl
  }

  try {
    const targetUrl = new URL(normalizedUrl, window.location.origin)
    const redirectPath = targetUrl.searchParams.get('redirect') || '/purchase'
    const redirectUrl = new URL(redirectPath, window.location.origin)
    const paymentType = normalizeVisibleMethod(context.paymentType) || context.paymentType.trim() || 'wxpay'

    redirectUrl.searchParams.set('payment_type', paymentType)
    redirectUrl.searchParams.set('order_type', context.orderType)

    if (context.planId) {
      redirectUrl.searchParams.set('plan_id', String(context.planId))
    } else {
      redirectUrl.searchParams.delete('plan_id')
    }

    if (context.orderAmount > 0) {
      redirectUrl.searchParams.set('amount', String(context.orderAmount))
    } else {
      redirectUrl.searchParams.delete('amount')
    }

    targetUrl.searchParams.set('redirect', `${redirectUrl.pathname}${redirectUrl.search}`)
    return targetUrl.toString()
  } catch {
    return normalizedUrl
  }
}

function onPaymentDone() {
  const wasSubscription = paymentState.value.orderType === 'subscription'
  resetPayment()
  selectedPlan.value = null
  if (wasSubscription) {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function onPaymentSuccess() {
  removeRecoverySnapshot()
  authStore.refreshUser()
  if (paymentState.value.orderType === 'subscription') {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
}

function onPaymentSettled() {
  removeRecoverySnapshot()
}

// All checkout data from single API call
const checkout = ref<CheckoutInfoResponse>({
  methods: {}, global_min: 0, global_max: 0, min_amount: 0, max_amount: 0,
  plans: [], balance_disabled: false, balance_recharge_multiplier: 6.8, subscription_usd_to_cny_rate: 0, recharge_fee_rate: 0, help_text: '', help_image_url: '', stripe_publishable_key: '',
})

const tabs = computed(() => {
  const result: { key: 'recharge' | 'subscription'; label: string }[] = []
  if (canRechargeBalance.value) result.push({ key: 'recharge', label: t('payment.tabTopUp') })
  if (hasSubscriptionPlans.value) result.push({ key: 'subscription', label: t('payment.tabSubscribe') })
  return result
})

const visibleMethods = computed(() => getVisibleMethods(checkout.value.methods))
const enabledMethods = computed(() => Object.keys(visibleMethods.value))
const validAmount = computed(() => amount.value ?? 0)
const nativeRechargeMethods = computed(() => enabledMethods.value)
const showNativeRechargeMethods = computed(() => checkout.value.balance_recharge_available === true)
const canUseNativeRecharge = computed(() => showNativeRechargeMethods.value && nativeRechargeMethods.value.length > 0)
const nativeRechargeUnlockHint = computed(() =>
  canUseNativeRecharge.value
    ? t('payment.rechargeMethod.unlockedHint')
    : t('payment.rechargeMethod.unlockHint')
)
const showNativeRechargeLockHint = computed(() =>
  showRechargeModePicker.value
    && nativeRechargeLockHintVisible.value
    && !canUseNativeRecharge.value
)
const showNativeRechargeHelp = computed(() =>
  (checkout.value.help_text || checkout.value.help_image_url)
    && paymentPhase.value === 'select'
    && activeTab.value === 'recharge'
    && selectedRechargeMode.value === 'native'
    && canUseNativeRecharge.value
    && !selectedPlan.value
)
const quickRechargeBalanceCredits = [5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000]
const defaultQuickRechargeBalanceCredit = 20
const balanceRechargeCnyPerCredit = computed(() => {
  const multiplier = checkout.value.balance_recharge_multiplier
  return multiplier > 0 ? multiplier : 6.8
})
// 订阅 CNY 换算汇率（1 USD = X CNY）。0 = 未配置，订阅保持 price 直付（与后端 opt-in 条件严格镜像）。
const subscriptionUsdToCnyRate = computed(() => {
  const rate = checkout.value.subscription_usd_to_cny_rate
  return Number.isFinite(rate) && rate > 0 ? rate : 0
})

// Adaptive grid: center single card, 2-col for 2 plans, 3-col for 3+
const planGridClass = computed(() => {
  const n = checkout.value.plans.length
  if (n <= 2) return 'grid grid-cols-1 gap-5 sm:grid-cols-2'
  return 'grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3'
})

function syncActivePaymentTab(preferred?: 'recharge' | 'subscription') {
  const availableTabs = tabs.value
  if (availableTabs.length === 0) {
    selectedPlan.value = null
    activeTab.value = 'recharge'
    return
  }
  const preferredTab = preferred && availableTabs.some(tab => tab.key === preferred)
    ? preferred
    : undefined
  if (preferredTab) {
    activeTab.value = preferredTab
  } else if (!availableTabs.some(tab => tab.key === activeTab.value)) {
    activeTab.value = availableTabs[0].key
  }
  if (activeTab.value !== 'subscription') {
    selectedPlan.value = null
    showRenewalModal.value = false
    renewGroupId.value = null
  }
}

// Check if an amount fits a method's [min, max]. 0 = no limit.
function amountFitsMethod(amt: number, methodType: string): boolean {
  if (amt <= 0) return true
  const ml = visibleMethods.value[methodType]
  if (!ml) return false
  if (ml.single_min > 0 && amt < ml.single_min) return false
  if (ml.single_max > 0 && amt > ml.single_max) return false
  return true
}

// Visible methods decide the amount range shown to users.
const globalMinAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_min <= 0)) return 0
  return Math.min(...limits.map(limit => limit.single_min))
})
const globalMaxAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_max <= 0)) return 0
  return Math.max(...limits.map(limit => limit.single_max))
})
const configuredRechargeMinAmount = computed(() => Math.max(checkout.value.min_amount ?? 0, 0))
const configuredRechargeMaxAmount = computed(() => Math.max(checkout.value.max_amount ?? 0, 0))
const rechargeMinAmount = computed(() => Math.max(globalMinAmount.value, configuredRechargeMinAmount.value))
const rechargeMaxAmount = computed(() => {
  const methodMax = globalMaxAmount.value
  const configMax = configuredRechargeMaxAmount.value
  if (methodMax > 0 && configMax > 0) return Math.min(methodMax, configMax)
  return methodMax > 0 ? methodMax : configMax
})

// Selected method's limits (for validation and error messages)
const selectedLimit = computed(() => visibleMethods.value[selectedMethod.value])
const selectedCurrency = computed(() => normalizePaymentCurrency(selectedLimit.value?.currency))
const appliesBalanceRechargeRate = computed(() => selectedCurrency.value === DEFAULT_PAYMENT_CURRENCY)
const quickRechargeAmounts = computed(() => {
  const min = rechargeMinAmount.value
  const max = rechargeMaxAmount.value
  const amounts = quickRechargeBalanceCredits
    .filter((value) => value >= min && (max <= 0 || value <= max))
    .map((value) => value)
  if (min > 0 && (max <= 0 || min <= max) && !amounts.includes(min)) {
    amounts.unshift(Math.round(min * 100) / 100)
  }
  return amounts.slice(0, 6)
})
const showBalanceRechargeRate = computed(() => appliesBalanceRechargeRate.value && balanceRechargeCnyPerCredit.value !== 1)
const creditedAmount = computed(() => {
  return Math.round(validAmount.value * 100) / 100
})
const selectedPaymentInputPrefix = computed(() => paymentAmountPrefix(selectedCurrency.value))
const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

function formatSelectedPaymentAmount(value: number): string {
  return formatPaymentAmount(value, selectedCurrency.value, localeCode.value)
}

function formatQuickRechargeAmountLabel(value: number): string {
  return formatBalanceCreditAmount(value, localeCode.value)
}

function formatQuickRechargeAmountDescription(value: number): string {
  return t('payment.quickAmountPayDescription', {
    amount: formatSelectedPaymentAmount(value),
  })
}

function quickRechargeDisabledReason(value: number): string {
  if (rechargeMinAmount.value > 0 && value < rechargeMinAmount.value) {
    return t('payment.quickAmountBelowRechargeMin')
  }
  if (rechargeMaxAmount.value > 0 && value > rechargeMaxAmount.value) {
    return t('payment.quickAmountAboveLimit')
  }
  if (!selectedMethod.value || amountFitsMethod(value, selectedMethod.value)) return ''
  const limit = selectedLimit.value
  if (limit?.single_min && limit.single_min > 0 && value < limit.single_min) {
    return t('payment.quickAmountBelowLimit')
  }
  if (limit?.single_max && limit.single_max > 0 && value > limit.single_max) {
    return t('payment.quickAmountAboveLimit')
  }
  return t('payment.quickAmountUnavailable')
}

function paymentAmountForBalanceCredit(credit: number): number {
  return Math.round(credit * 100) / 100
}

function amountFitsRechargeRules(amt: number, methodType: string): boolean {
  if (amt <= 0) return amountFitsMethod(amt, methodType)
  if (!amountFitsMethod(amt, methodType)) return false
  if (rechargeMinAmount.value > 0 && amt < rechargeMinAmount.value) return false
  if (rechargeMaxAmount.value > 0 && amt > rechargeMaxAmount.value) return false
  return true
}

function applyDefaultRechargeAmount() {
  if (amount.value !== null || checkout.value.balance_disabled || activeTab.value !== 'recharge' || paymentPhase.value !== 'select') {
    return
  }
  const preferredAmount = paymentAmountForBalanceCredit(defaultQuickRechargeBalanceCredit)
  if (amountFitsRechargeRules(preferredAmount, selectedMethod.value)) {
    amount.value = preferredAmount
    return
  }
  const fallbackAmount = quickRechargeAmounts.value.find((value) => amountFitsRechargeRules(value, selectedMethod.value))
  if (fallbackAmount) {
    amount.value = fallbackAmount
  }
}

function currencyFractionDigits(currency: string): number {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency,
    }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function roundPaymentAmount(value: number, currency: string): number {
  if (!Number.isFinite(value)) return 0
  const factor = 10 ** currencyFractionDigits(currency)
  return Math.round(value * factor) / factor
}

function ceilPaymentAmount(value: number, currency: string): number {
  if (!Number.isFinite(value)) return 0
  const factor = 10 ** currencyFractionDigits(currency)
  return Math.ceil(value * factor) / factor
}

function subscriptionPaymentAmountForCurrency(value: number, currency: string): number {
  const rate = subscriptionUsdToCnyRate.value
  if (rate <= 0 || currency !== DEFAULT_PAYMENT_CURRENCY) return roundPaymentAmount(value, currency)
  return roundPaymentAmount(value * rate, currency)
}

function formatSelectedSubscriptionPaymentAmount(value: number): string {
  return formatSelectedPaymentAmount(subscriptionPaymentAmountForCurrency(value, selectedCurrency.value))
}

const methodOptions = computed<PaymentMethodOption[]>(() =>
  nativeRechargeMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      display_name: ml?.display_name,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsRechargeRules(validAmount.value, type),
    }
  })
)

const feeRate = computed(() => checkout.value?.recharge_fee_rate ?? 0)
const cardCodeRechargeFeeLabel = computed(() => t('payment.rechargeMethod.feeRate', { rate: '1.8' }))
const nativeRechargeFeeLabel = computed(() =>
  feeRate.value > 0
    ? t('payment.rechargeMethod.feeRate', { rate: formatFeeRate(feeRate.value) })
    : t('payment.rechargeMethod.noFee')
)
const feeAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.ceil(((validAmount.value * feeRate.value) / 100) * 100) / 100
    : 0
)

function formatFeeRate(rate: number): string {
  return Number.isInteger(rate) ? String(rate) : String(Number(rate.toFixed(2)))
}
const totalAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.round((validAmount.value + feeAmount.value) * 100) / 100
    : validAmount.value
)

const amountError = computed(() => {
  if (validAmount.value <= 0) return ''
  if (rechargeMinAmount.value > 0 && validAmount.value < rechargeMinAmount.value) {
    return t('payment.amountTooLowCardCode', { min: formatSelectedPaymentAmount(rechargeMinAmount.value) })
  }
  if (rechargeMaxAmount.value > 0 && validAmount.value > rechargeMaxAmount.value) {
    return t('payment.amountTooHigh', { max: formatSelectedPaymentAmount(rechargeMaxAmount.value) })
  }
  // No method can handle this amount
  if (!nativeRechargeMethods.value.some((m) => amountFitsMethod(validAmount.value, m))) {
    return t('payment.amountNoMethod')
  }
  // Selected method can't handle this amount (but others can)
  const ml = selectedLimit.value
  if (ml) {
    if (ml.single_min > 0 && validAmount.value < ml.single_min) return t('payment.amountTooLow', { min: formatSelectedPaymentAmount(ml.single_min) })
    if (ml.single_max > 0 && validAmount.value > ml.single_max) return t('payment.amountTooHigh', { max: formatSelectedPaymentAmount(ml.single_max) })
  }
  return ''
})

const canSubmit = computed(() =>
  selectedRechargeMode.value === 'native'
    && canUseNativeRecharge.value
    && validAmount.value > 0
    && amountFitsRechargeRules(validAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

const rechargeSubmitLabel = computed(() =>
  `${t('payment.confirmPayment')} ${formatSelectedPaymentAmount(totalAmount.value)}`
)

const subPaymentAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  return subscriptionPaymentAmountForCurrency(price, selectedCurrency.value)
})

const subFeeAmount = computed(() => {
  if (feeRate.value <= 0 || subPaymentAmount.value <= 0) return 0
  return ceilPaymentAmount((subPaymentAmount.value * feeRate.value) / 100, selectedCurrency.value)
})

const subTotalAmount = computed(() => {
  if (feeRate.value <= 0 || subPaymentAmount.value <= 0) return subPaymentAmount.value
  return roundPaymentAmount(subPaymentAmount.value + subFeeAmount.value, selectedCurrency.value)
})

function subscriptionTotalAmountForCurrency(value: number, currency: string): number {
  const paymentAmount = subscriptionPaymentAmountForCurrency(value, currency)
  if (feeRate.value <= 0 || paymentAmount <= 0) return paymentAmount
  const fee = ceilPaymentAmount((paymentAmount * feeRate.value) / 100, currency)
  return roundPaymentAmount(paymentAmount + fee, currency)
}

// Subscription-specific: method options based on gateway pay amount
const subMethodOptions = computed<PaymentMethodOption[]>(() => {
  const planPrice = selectedPlan.value?.price ?? 0
  const gatewayAmount = subscriptionTotalAmountForCurrency(planPrice, selectedCurrency.value)
  return enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      display_name: ml?.display_name,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(gatewayAmount, type),
    }
  })
})

const canSubmitSubscription = computed(() =>
  selectedPlan.value !== null
    && amountFitsMethod(subTotalAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

// Auto-switch to first available method when current selection can't handle the amount
watch(() => [validAmount.value, selectedMethod.value] as const, ([amt, method]) => {
  if (amt <= 0 || amountFitsMethod(amt, method)) return
  const available = nativeRechargeMethods.value.find((m) => amountFitsMethod(amt, m))
  if (available) selectedMethod.value = available
})

// Payment buttons stay on the product primary color; provider identity is shown by icons.
const paymentButtonClass = computed(() => {
  return 'btn-primary'
})

// Renewal modal state
const showRenewalModal = ref(false)
const renewGroupId = ref<number | null>(null)
const renewalPlans = computed(() => {
  if (renewGroupId.value == null) return []
  return checkout.value.plans.filter(p => p.group_id === renewGroupId.value)
})

const planValiditySuffix = computed(() => {
  if (!selectedPlan.value) return ''
  const u = selectedPlan.value.validity_unit || 'day'
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${selectedPlan.value.validity_days}${t('payment.days')}`
})

function planHasPeakRate(plan: SubscriptionPlan): boolean {
  return hasPeakRate(plan)
}

function planPeakRateLabel(plan: SubscriptionPlan): string {
  return formatPeakRateWindow(plan, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

function selectPlan(plan: SubscriptionPlan) {
  selectedPlan.value = plan
  errorMessage.value = ''
}

function selectPlanFromModal(plan: SubscriptionPlan) {
  showRenewalModal.value = false
  renewGroupId.value = null
  selectedPlan.value = plan
  errorMessage.value = ''
}

function closeRenewalModal() {
  showRenewalModal.value = false
  renewGroupId.value = null
}

async function handleSubmitRecharge() {
  if (!canSubmit.value || submitting.value) return
  await createOrder(validAmount.value, 'balance')
}

function openCardCodePurchase() {
  const opened = window.open(CARD_CODE_PURCHASE_URL, '_blank')
  if (opened) {
    opened.opener = null
  }
}

function selectCardCodeRechargeMode() {
  nativeRechargeLockHintVisible.value = false
  selectedRechargeMode.value = 'card_code'
}

function selectNativeRechargeMode() {
  selectedRechargeMode.value = 'native'
  if (!canUseNativeRecharge.value) {
    nativeRechargeLockHintVisible.value = true
    return
  }
  nativeRechargeLockHintVisible.value = false
}

async function handleCardCodeRedeem() {
  const code = cardCodeRedeemCode.value.trim()
  if (!code) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  cardCodeRedeemSubmitting.value = true
  cardCodeRedeemError.value = ''
  cardCodeRedeemResult.value = null
  try {
    const result = await redeemAPI.redeem(code)
    cardCodeRedeemResult.value = result
    cardCodeRedeemCode.value = ''
    await authStore.refreshUser()
    if (result.type === 'subscription') {
      await Promise.allSettled([
        subscriptionStore.fetchActiveSubscriptions(true),
        subscriptionCapabilityStore.fetchSubscriptionCapability(true),
      ])
    }
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (err: unknown) {
    cardCodeRedeemError.value = extractApiErrorMessage(err, t('redeem.failedToRedeem'))
    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    cardCodeRedeemSubmitting.value = false
  }
}

async function confirmSubscribe() {
  if (!hasSubscriptionPlans.value || !selectedPlan.value || submitting.value) return
  await createOrder(selectedPlan.value.price, 'subscription', selectedPlan.value.id)
}

async function createOrder(orderAmount: number, orderType: OrderType, planId?: number, options: CreateOrderOptions = {}) {
  submitting.value = true
  errorMessage.value = ''
  errorHintMessage.value = ''
  const requestType = normalizeVisibleMethod(options.paymentType || selectedMethod.value) || options.paymentType || selectedMethod.value
  try {
    const payload = buildCreateOrderPayload({
      amount: orderAmount,
      paymentType: requestType,
      orderType,
      planId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && normalizeVisibleMethod(requestType) === 'alipay'),
    })
    if (options.openid) {
      payload.openid = options.openid
    }
    if (options.wechatResumeToken) {
      payload.wechat_resume_token = options.wechatResumeToken
    }

    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const openWindow = (url: string) => {
      const win = window.open(url, 'paymentPopup', getPaymentPopupFeatures())
      if (!win || win.closed) {
        window.location.href = url
      }
    }
    const visibleMethod = normalizeVisibleMethod(requestType) || requestType
    // When user clicks the dedicated Stripe button, leave method blank so the
    // landing page renders Stripe's full Payment Element (card/link/alipay/wxpay).
    const stripeMethod = visibleMethod === 'stripe'
      ? ''
      : visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret && visibleMethod !== 'airwallex'
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const airwallexRouteUrl = result.client_secret && result.intent_id
      ? router.resolve({
        path: '/payment/airwallex',
        query: {
          order_id: String(result.order_id),
          out_trade_no: result.out_trade_no || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType,
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && visibleMethod === 'alipay'),
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
      airwallexRouteUrl,
    })

    if (decision.kind === 'wechat_oauth' && decision.oauth?.authorize_url) {
      window.location.href = buildWechatOAuthAuthorizeUrl(decision.oauth.authorize_url, {
        paymentType: visibleMethod,
        orderType,
        planId,
        orderAmount,
      })
      return
    }

    if (decision.kind === 'unhandled') {
      applyScenarioError({ reason: 'UNHANDLED_PAYMENT_SCENARIO' }, visibleMethod)
      return
    }

    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)

    if (decision.kind === 'stripe_popup') {
      openWindow(decision.paymentState.payUrl)
      return
    }
    if (decision.kind === 'stripe_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'airwallex_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'wechat_jsapi' && decision.jsapi) {
      try {
        const jsapiResult = await invokeWechatJsapiPayment(decision.jsapi as Record<string, unknown>)
        const errMsg = String(jsapiResult.err_msg || '').toLowerCase()
        if (errMsg.includes('cancel')) {
          appStore.showInfo(t('payment.qr.cancelled'))
          resetPayment()
        } else if (errMsg && !errMsg.includes('ok')) {
          resetPayment()
          const fallbackApplied = await attemptMobileQrFallback(
            { reason: 'WECHAT_JSAPI_FAILED', message: errMsg },
            {
              orderAmount,
              orderType,
              planId,
              paymentType: visibleMethod,
              attempted: options.mobileQrFallbackAttempted === true,
            },
          )
          if (!fallbackApplied) {
            applyScenarioError({ reason: 'WECHAT_JSAPI_FAILED', message: errMsg }, visibleMethod)
          }
        } else {
          const resultState = { ...decision.paymentState }
          resetPayment()
          await redirectToPaymentResult(resultState)
        }
      } catch (err: unknown) {
        resetPayment()
        const fallbackApplied = await attemptMobileQrFallback(err, {
          orderAmount,
          orderType,
          planId,
          paymentType: visibleMethod,
          attempted: options.mobileQrFallbackAttempted === true,
        })
        if (!fallbackApplied) {
          throw err
        }
      }
      return
    }
    if (decision.kind === 'redirect_waiting' && decision.paymentState.payUrl) {
      if (isMobileDevice()) {
        window.location.href = decision.paymentState.payUrl
        return
      }
      openWindow(decision.paymentState.payUrl)
    }
  } catch (err: unknown) {
    const apiErr = err as Record<string, unknown>
    if (apiErr.reason === 'TOO_MANY_PENDING') {
      const metadata = apiErr.metadata as Record<string, unknown> | undefined
      errorMessage.value = t('payment.errors.tooManyPending', { max: metadata?.max || '' })
      errorHintMessage.value = ''
    } else if (apiErr.reason === 'CANCEL_RATE_LIMITED') {
      errorMessage.value = t('payment.errors.cancelRateLimited')
      errorHintMessage.value = ''
    } else if (await attemptMobileQrFallback(err, {
      orderAmount,
      orderType,
      planId,
      paymentType: requestType,
      attempted: options.mobileQrFallbackAttempted === true,
    })) {
      return
    } else {
      const handled = applyScenarioError(
        err,
        normalizeVisibleMethod(options.paymentType || selectedMethod.value) || selectedMethod.value,
      )
      if (!handled) {
        errorMessage.value = extractI18nErrorMessage(err, t, 'payment.errors', extractApiErrorMessage(err, t('payment.result.failed')))
        errorHintMessage.value = ''
      }
      if (handled) {
        return
      }
    }
    appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  } finally {
    submitting.value = false
  }
}

interface MobileQrFallbackContext {
  orderAmount: number
  orderType: OrderType
  planId?: number
  paymentType: string
  attempted: boolean
}

function shouldFallbackToDesktopQr(err: unknown, paymentMethod: string, attempted: boolean): boolean {
  if (attempted || !isMobileDevice()) {
    return false
  }

  const normalizedMethod = normalizeVisibleMethod(paymentMethod) || paymentMethod
  const reason = typeof err === 'object' && err && 'reason' in err && typeof err.reason === 'string'
    ? err.reason
    : ''
  const message = err instanceof Error
    ? err.message
    : (typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
      ? err.message
      : '')
  const normalizedMessage = message.toLowerCase()

  if (normalizedMethod === 'wxpay') {
    return reason === 'WECHAT_H5_NOT_AUTHORIZED'
      || reason === 'WECHAT_PAYMENT_MP_NOT_CONFIGURED'
      || reason === 'WECHAT_JSAPI_FAILED'
      || reason === 'PAYMENT_GATEWAY_ERROR'
      || reason === 'UNHANDLED_PAYMENT_SCENARIO'
      || normalizedMessage.includes('weixinjsbridge is unavailable')
      || normalizedMessage.includes('wechat_jsapi_unavailable')
  }

  if (normalizedMethod === 'alipay') {
    return reason === 'PAYMENT_GATEWAY_ERROR' || reason === 'UNHANDLED_PAYMENT_SCENARIO'
  }

  return false
}

async function attemptMobileQrFallback(err: unknown, context: MobileQrFallbackContext): Promise<boolean> {
  if (!shouldFallbackToDesktopQr(err, context.paymentType, context.attempted)) {
    return false
  }

  try {
    const visibleMethod = normalizeVisibleMethod(context.paymentType) || context.paymentType
    const payload = buildCreateOrderPayload({
      amount: context.orderAmount,
      paymentType: visibleMethod,
      orderType: context.orderType,
      planId: context.planId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: false,
      isWechatBrowser: false,
    })
    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const stripeMethod = visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType: context.orderType,
      isMobile: false,
      isWechatBrowser: false,
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
    })

    if (decision.kind !== 'qr_waiting' || !decision.paymentState.qrCode) {
      return false
    }

    errorMessage.value = ''
    errorHintMessage.value = ''
    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)
    appStore.showWarning(t('payment.errors.mobilePaymentFallbackToQr'))
    return true
  } catch {
    return false
  }
}

function applyScenarioError(err: unknown, paymentMethod: string): boolean {
  const descriptor = describePaymentScenarioError(err, {
    paymentMethod,
    isMobile: isMobileDevice(),
    isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
  })
  if (!descriptor) {
    errorMessage.value = ''
    errorHintMessage.value = ''
    return false
  }
  errorMessage.value = t(descriptor.messageKey)
  errorHintMessage.value = descriptor.hintKey ? t(descriptor.hintKey) : ''
  appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  return true
}

async function resumeWechatPaymentFromQuery() {
  const resume = parseWechatResumeRoute(route.query, checkout.value.plans, validAmount.value)
  if (!resume) {
    return
  }

  selectedMethod.value = resume.paymentType
  if (resume.orderType === 'balance' && resume.orderAmount > 0) {
    amount.value = resume.orderAmount
  }
  if (resume.orderType === 'subscription' && resume.planId) {
    selectedPlan.value = checkout.value.plans.find(plan => plan.id === resume.planId) ?? null
    if (!selectedPlan.value) {
      return
    }
  }

  await router.replace({ path: route.path, query: stripWechatResumeQuery(route.query) })

  if (resume.wechatResumeToken) {
    await createOrder(0, resume.orderType, resume.planId, {
      wechatResumeToken: resume.wechatResumeToken,
      paymentType: resume.paymentType,
      isResume: true,
    })
    return
  }

  if (resume.orderAmount > 0 && resume.openid) {
    await createOrder(resume.orderAmount, resume.orderType, resume.planId, {
      openid: resume.openid,
      paymentType: resume.paymentType,
      isResume: true,
    })
  }
}

onMounted(async () => {
  try {
    const res = await paymentCheckoutStore.fetchCheckoutInfo(true)
    if (!res) {
      appStore.showError(t('common.error'))
      return
    }
    checkout.value = res
    setSettlementCnyPerCredit(checkout.value.balance_recharge_multiplier)
    if (enabledMethods.value.length) {
      const order: readonly string[] = METHOD_ORDER
      const sorted = [...enabledMethods.value].sort((a, b) => {
        const ai = order.indexOf(a)
        const bi = order.indexOf(b)
        return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
      })
      selectedMethod.value = sorted[0]
    }
    if (typeof window !== 'undefined') {
      if (hasWechatResumeQuery(route.query)) {
        removeRecoverySnapshot()
      }
      const routeResumeToken = typeof route.query.resume_token === 'string'
        ? route.query.resume_token
        : typeof route.query.wechat_resume_token === 'string'
          ? route.query.wechat_resume_token
          : undefined
      const restored = readPaymentRecoverySnapshot(
        window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY),
        { resumeToken: routeResumeToken },
      )
      if (restored) {
        paymentState.value = restored
        paymentPhase.value = 'paying'
        const restoredMethod = normalizeVisibleMethod(restored.paymentType)
          || (visibleMethods.value[restored.paymentType] ? restored.paymentType : '')
        if (restoredMethod) {
          selectedMethod.value = restoredMethod
        }
      } else {
        removeRecoverySnapshot()
      }
    }
    await resumeWechatPaymentFromQuery()
    syncActivePaymentTab()
    // Handle renewal navigation: ?tab=subscription&group=123
    if (route.query.tab === 'subscription' && hasSubscriptionPlans.value) {
      syncActivePaymentTab('subscription')
      if (route.query.group) {
        const groupId = Number(route.query.group)
        const groupPlans = checkout.value.plans.filter(p => p.group_id === groupId)
        if (groupPlans.length === 1) {
          selectedPlan.value = groupPlans[0]
        } else if (groupPlans.length > 1) {
          renewGroupId.value = groupId
          showRenewalModal.value = true
        }
      }
    }
    selectedRechargeMode.value = 'card_code'
    applyDefaultRechargeAmount()
  } catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { loading.value = false }
  // Fetch active subscriptions (uses cache, non-blocking)
  subscriptionStore.fetchActiveSubscriptions().catch(() => {})
})
</script>
