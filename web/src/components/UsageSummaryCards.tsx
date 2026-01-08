import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { formatNumber } from '@/lib/formatters'
import { UsageSummary } from '@/api/amp'

interface UsageSummaryCardsProps {
  summary: UsageSummary[]
}

export function UsageSummaryCards({ summary }: UsageSummaryCardsProps) {
  const totalRequests = summary.reduce((acc, s) => acc + s.requestCount, 0)
  const totalInputTokens = summary.reduce((acc, s) => acc + s.inputTokensSum, 0)
  const totalOutputTokens = summary.reduce((acc, s) => acc + s.outputTokensSum, 0)
  const totalCacheRead = summary.reduce((acc, s) => acc + s.cacheReadInputTokensSum, 0)
  const totalCacheWrite = summary.reduce((acc, s) => acc + s.cacheCreationInputTokensSum, 0)
  const totalCostMicros = summary.reduce((acc, s) => acc + (s.costMicrosSum || 0), 0)
  const totalCostUsd = (totalCostMicros / 1_000_000).toFixed(6)

  return (
    <div className="grid gap-4 md:grid-cols-6">
      <Card>
        <CardHeader className="pb-2">
          <CardDescription>总请求数</CardDescription>
          <CardTitle className="text-2xl">{formatNumber(totalRequests)}</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardDescription>输入 Tokens</CardDescription>
          <CardTitle className="text-2xl">{formatNumber(totalInputTokens)}</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardDescription>输出 Tokens</CardDescription>
          <CardTitle className="text-2xl">{formatNumber(totalOutputTokens)}</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardDescription>缓存读取</CardDescription>
          <CardTitle className="text-2xl">{formatNumber(totalCacheRead)}</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardDescription>缓存写入</CardDescription>
          <CardTitle className="text-2xl">{formatNumber(totalCacheWrite)}</CardTitle>
        </CardHeader>
      </Card>
      <Card className="bg-gradient-to-br from-green-500/10 to-emerald-500/10 border-green-500/20">
        <CardHeader className="pb-2">
          <CardDescription>总成本</CardDescription>
          <CardTitle className="text-2xl text-green-600 dark:text-green-400">
            ${totalCostUsd}
          </CardTitle>
        </CardHeader>
      </Card>
    </div>
  )
}
