/**
 * 格式化缓存 token 数量（1K/1M 缩写）
 */
export function formatCacheTokens(tokens: number): string {
  if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(1)}M`
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}K`
  return tokens.toLocaleString()
}

/**
 * 自适应精度格式化倍率（确保小数值如 0.001 不被截断）
 */
export function formatMultiplier(val: number): string {
  if (!Number.isFinite(val)) return '1'
  if (val >= 0.01) {
    const [whole, decimal = ''] = val.toFixed(4).replace(/0+$/, '').replace(/\.$/, '').split('.')
    return `${whole}.${decimal.padEnd(2, '0')}`
  }
  if (val >= 0.001) return val.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')
  if (val >= 0.0001) return val.toFixed(4)
  return val.toPrecision(2)
}
