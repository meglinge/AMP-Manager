export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('zh-CN')
}

export function formatNumber(num: number | undefined): string {
  if (num === undefined || num === null) return '-'
  return num.toLocaleString()
}
