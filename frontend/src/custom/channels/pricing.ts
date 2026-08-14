export function formatScaledCny(value: number | null, scale: number): string {
  if (value == null) return '-'
  return `¥${(value * scale).toPrecision(10).replace(/\.?0+$/, '')}`
}
