import { normalizePaymentCurrency } from '@/components/payment/currency'

export function paymentAmountPrefix(currency?: string | null): string {
  switch (normalizePaymentCurrency(currency)) {
    case 'CNY':
    case 'JPY':
      return '¥'
    case 'USD':
    case 'HKD':
    case 'AUD':
    case 'CAD':
    case 'SGD':
    case 'NZD':
      return '$'
    case 'EUR':
      return '€'
    case 'GBP':
      return '£'
    default:
      return normalizePaymentCurrency(currency)
  }
}
