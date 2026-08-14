import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAmountPrefix } from '@/custom/payment/currency'

export const DEFAULT_SETTLEMENT_CURRENCY = 'CNY'
export const DEFAULT_CNY_PER_CREDIT = 6.8
export const SETTLEMENT_CURRENCIES = ['CNY', 'USD'] as const
export type SettlementCurrency = typeof SETTLEMENT_CURRENCIES[number]

const SETTLEMENT_CURRENCY_STORAGE_KEY = 'sub2api_settlement_currency'

type FractionDigitsOptions = number | {
  minimumFractionDigits?: number
  maximumFractionDigits?: number
}

function readStoredSettlementCurrency(): SettlementCurrency {
  if (typeof localStorage === 'undefined') return DEFAULT_SETTLEMENT_CURRENCY
  try {
    return normalizeSettlementCurrency(localStorage.getItem(SETTLEMENT_CURRENCY_STORAGE_KEY))
  } catch {
    return DEFAULT_SETTLEMENT_CURRENCY
  }
}

function writeStoredSettlementCurrency(currency: SettlementCurrency): void {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(SETTLEMENT_CURRENCY_STORAGE_KEY, currency)
  } catch {
    // Ignore storage failures; the selected currency still applies in memory.
  }
}

const settlementCurrency = ref<SettlementCurrency>(readStoredSettlementCurrency())
const settlementCnyPerCredit = ref(DEFAULT_CNY_PER_CREDIT)

if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event) => {
    if (event.key === SETTLEMENT_CURRENCY_STORAGE_KEY) {
      settlementCurrency.value = normalizeSettlementCurrency(event.newValue)
    }
  })
}

export function normalizeSettlementCurrency(value: unknown): SettlementCurrency {
  const normalized = String(value || '').trim().toUpperCase()
  return normalized === 'USD' ? 'USD' : DEFAULT_SETTLEMENT_CURRENCY
}

export function resolveCnyPerCredit(value: unknown): number {
  const numeric = Number(value)
  return Number.isFinite(numeric) && numeric > 0 ? numeric : DEFAULT_CNY_PER_CREDIT
}

export function convertSettlementAmount(
  amount: number | null | undefined,
  currency: SettlementCurrency,
  cnyPerCredit = DEFAULT_CNY_PER_CREDIT,
): number {
  const value = Number(amount)
  const safeAmount = Number.isFinite(value) ? value : 0
  return currency === 'CNY' ? safeAmount * resolveCnyPerCredit(cnyPerCredit) : safeAmount
}

export function convertSettlementAmountToCredits(
  amount: number | null | undefined,
  currency: SettlementCurrency,
  cnyPerCredit = DEFAULT_CNY_PER_CREDIT,
): number {
  const value = Number(amount)
  const safeAmount = Number.isFinite(value) ? value : 0
  return currency === 'CNY' ? safeAmount / resolveCnyPerCredit(cnyPerCredit) : safeAmount
}

function resolveFractionDigits(options?: FractionDigitsOptions): { minimum: number; maximum: number } {
  if (typeof options === 'number') {
    return { minimum: options, maximum: options }
  }

  const maximum = options?.maximumFractionDigits ?? 2
  const minimum = options?.minimumFractionDigits ?? maximum
  return {
    minimum: Math.max(0, minimum),
    maximum: Math.max(0, Math.max(maximum, minimum)),
  }
}

export function formatSettlementCurrencyAmount(
  amount: number | null | undefined,
  currency: SettlementCurrency,
  cnyPerCredit = DEFAULT_CNY_PER_CREDIT,
  locale?: string,
  fractionDigits?: FractionDigitsOptions,
): string {
  const converted = convertSettlementAmount(amount, currency, cnyPerCredit)
  const digits = resolveFractionDigits(fractionDigits)
  try {
    return new Intl.NumberFormat(locale || undefined, {
      style: 'currency',
      currency,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: digits.minimum,
      maximumFractionDigits: digits.maximum,
    }).format(converted)
  } catch {
    return `${paymentAmountPrefix(currency)}${converted.toFixed(digits.maximum)}`
  }
}

export function setSettlementCnyPerCredit(value: unknown): void {
  settlementCnyPerCredit.value = resolveCnyPerCredit(value)
}

export function setCurrentSettlementCurrency(value: unknown): void {
  const normalized = normalizeSettlementCurrency(value)
  settlementCurrency.value = normalized
  writeStoredSettlementCurrency(normalized)
}

export function getCurrentSettlementCurrency(): SettlementCurrency {
  return settlementCurrency.value
}

export function getCurrentSettlementCnyPerCredit(): number {
  return settlementCnyPerCredit.value
}

export function useSettlementCurrency() {
  const { t, locale } = useI18n()

  const cnyPerCredit = computed(() => settlementCnyPerCredit.value)
  const settlementCurrencyOptions = computed(() => [
    { value: 'CNY', label: t('settlementCurrency.cny') },
    { value: 'USD', label: t('settlementCurrency.usd') },
  ])
  const settlementAmountPrefix = computed(() => paymentAmountPrefix(settlementCurrency.value))

  const localeCode = computed(() => {
    const raw = locale as unknown
    if (typeof raw === 'string') return raw
    if (raw && typeof raw === 'object' && 'value' in raw) {
      return String((raw as { value?: string }).value || '')
    }
    return undefined
  })

  function setSettlementCurrency(value: unknown): void {
    setCurrentSettlementCurrency(value)
  }

  function formatSettlementAmount(
    amount: number | null | undefined,
    fractionDigits?: FractionDigitsOptions,
  ): string {
    return formatSettlementCurrencyAmount(
      amount,
      settlementCurrency.value,
      cnyPerCredit.value,
      localeCode.value,
      fractionDigits,
    )
  }

  function formatSettlementAmountPair(
    used: number | null | undefined,
    limit: number | null | undefined,
    fractionDigits?: FractionDigitsOptions,
  ): string {
    return `${formatSettlementAmount(used, fractionDigits)} / ${formatSettlementAmount(limit, fractionDigits)}`
  }

  function toBalanceCreditAmount(amount: number | null | undefined): number {
    return convertSettlementAmountToCredits(amount, settlementCurrency.value, cnyPerCredit.value)
  }

  return {
    settlementCurrency,
    settlementCurrencyOptions,
    settlementAmountPrefix,
    cnyPerCredit,
    setSettlementCurrency,
    formatSettlementAmount,
    formatSettlementAmountPair,
    toBalanceCreditAmount,
  }
}
