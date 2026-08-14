import { describe, expect, it } from 'vitest'
import {
  PROVIDER_CONFIG_FIELDS,
  isBuiltInAlipayMethod,
  isBuiltInWxpayMethod,
} from '@/custom/payment/providerConfig'

function findField(providerKey: string, key: string) {
  return (PROVIDER_CONFIG_FIELDS[providerKey] || []).find((field) => field.key === key)
}

describe('Wegoo payment provider config', () => {
  it('configures GM Pay USDT checkout defaults', () => {
    expect(findField('gmpay', 'currency')?.defaultValue).toBe('CNY')
    expect(findField('gmpay', 'token')?.defaultValue).toBe('USDT')
    expect(findField('gmpay', 'network')?.defaultValue).toBe('tron')
    expect(findField('gmpay', 'secretKey')?.sensitive).toBe(true)
  })

  it('does not classify custom Alipay aliases as built-in methods', () => {
    expect(isBuiltInAlipayMethod('easypay_alipay')).toBe(false)
    expect(isBuiltInAlipayMethod('custom_alipay')).toBe(false)
  })

  it('does not classify custom WeChat aliases as built-in methods', () => {
    expect(isBuiltInWxpayMethod('easypay_wxpay')).toBe(false)
    expect(isBuiltInWxpayMethod('custom_wxpay')).toBe(false)
  })
})
