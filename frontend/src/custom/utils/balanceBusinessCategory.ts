import type { BalanceBusinessCategory } from '@/types'

export const balanceBusinessCategoryOptions: Array<{ value: BalanceBusinessCategory; label: string }> = [
  { value: 'manual_collection', label: '人工收款' },
  { value: 'manual_refund', label: '人工退款' },
  { value: 'gift_compensation', label: '赠送补偿' },
  { value: 'gift_reversal', label: '赠送冲正' },
  { value: 'system_service_fee', label: '系统服务费' },
  { value: '', label: '未分类' }
]

export const redeemBusinessCategoryOptions: Array<{ value: BalanceBusinessCategory; label: string }> = [
  { value: '', label: '未分类' },
  { value: 'recharge', label: '真实充值' }
]

const balanceBusinessCategoryLabelOptions = [
  ...redeemBusinessCategoryOptions,
  ...balanceBusinessCategoryOptions.filter((option) => option.value !== ''),
  { value: 'affiliate_reward' as BalanceBusinessCategory, label: '邀请奖励' }
]

export function defaultBalanceBusinessCategory(
  operation: 'set' | 'add' | 'subtract'
): BalanceBusinessCategory {
  return operation === 'subtract' ? 'manual_refund' : 'manual_collection'
}

export function balanceBusinessCategoryLabel(category?: string | null): string {
  return (
    balanceBusinessCategoryLabelOptions.find((option) => option.value === (category || ''))?.label ||
    category ||
    '未分类'
  )
}
