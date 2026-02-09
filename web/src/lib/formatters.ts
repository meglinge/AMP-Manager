export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('zh-CN')
}

export function formatNumber(num: number | undefined): string {
  if (num === undefined || num === null) return '-'
  return num.toLocaleString()
}

export function formatCompact(num: number | undefined): string {
  if (num === undefined || num === null) return '-'
  const abs = Math.abs(num)
  if (abs >= 1_000_000_000) return (num / 1_000_000_000).toFixed(1).replace(/\.0$/, '') + 'B'
  if (abs >= 1_000_000) return (num / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  if (abs >= 1_000) return (num / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  return num.toString()
}

export function formatExact(num: number | undefined): string {
  if (num === undefined || num === null) return '-'
  return num.toLocaleString()
}
