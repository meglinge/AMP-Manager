import { useState, useEffect } from 'react'
import { getDashboard, DashboardData, DashboardCacheHitRate } from '@/api/dashboard'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Num } from '@/components/Num'
import { motion, staggerContainer, staggerItem } from '@/lib/motion'
import {
  Wallet,
  TrendingUp,
  Zap,
  ArrowUpRight,
  ArrowDownRight,
  RefreshCw,
  Activity,
  DollarSign,
  Hash,
  AlertTriangle,
  DatabaseZap,
} from 'lucide-react'

export default function Overview() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadDashboard = async () => {
    setLoading(true)
    setError('')
    try {
      const result = await getDashboard()
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadDashboard()
  }, [])

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error && !data) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-4">
        <p className="text-destructive">{error}</p>
        <Button variant="outline" onClick={loadDashboard}>重试</Button>
      </div>
    )
  }

  if (!data) return null

  const balanceUsd = parseFloat(data.balance.balanceUsd)
  const todayCost = parseFloat(data.today.costUsd)
  const weekCost = parseFloat(data.week.costUsd)

  const maxTrendCost = Math.max(...(data.dailyTrend || []).map(d => d.costMicros), 1)
  const maxTrendReqs = Math.max(...(data.dailyTrend || []).map(d => d.requests), 1)

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: 'spring', bounce: 0.2, duration: 0.6 }}
        className="flex items-center justify-between"
      >
        <div>
          <h2 className="text-2xl font-bold tracking-tight">概览</h2>
          <p className="text-muted-foreground">账户用量和费用一览</p>
        </div>
        <Button variant="outline" size="sm" onClick={loadDashboard} disabled={loading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </motion.div>

      {/* Balance + Today/Week/Month stats */}
      <motion.div
        variants={staggerContainer}
        initial="hidden"
        animate="visible"
        className="grid gap-4 md:grid-cols-2 lg:grid-cols-4"
      >
        {/* Balance Card */}
        <motion.div variants={staggerItem} whileHover={{ scale: 1.03, y: -4 }} whileTap={{ scale: 0.98 }}>
          <Card className="bg-gradient-to-br from-blue-500/10 to-cyan-500/10 border-blue-500/20">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardDescription className="flex items-center gap-1.5">
                  <Wallet className="h-4 w-4" />
                  账户余额
                </CardDescription>
              </div>
              <CardTitle className="text-3xl text-blue-600 dark:text-blue-400">
                ${balanceUsd.toFixed(2)}
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-3">
              <p className="text-xs text-muted-foreground">
                {data.balance.balanceMicros.toLocaleString()} 微美元
              </p>
            </CardContent>
          </Card>
        </motion.div>

        {/* Today */}
        <motion.div variants={staggerItem} whileHover={{ scale: 1.03, y: -4 }} whileTap={{ scale: 0.98 }}>
          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardDescription className="flex items-center gap-1.5">
                  <Zap className="h-4 w-4" />
                  今日用量
                </CardDescription>
                {data.today.errorCount > 0 && (
                  <Badge variant="destructive" className="text-[10px]">
                    {data.today.errorCount} 错误
                  </Badge>
                )}
              </div>
              <CardTitle className="text-2xl"><Num value={data.today.requestCount} /></CardTitle>
            </CardHeader>
            <CardContent className="pb-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <DollarSign className="h-3 w-3" />
                ${todayCost.toFixed(4)}
              </p>
            </CardContent>
          </Card>
        </motion.div>

        {/* 7 days */}
        <motion.div variants={staggerItem} whileHover={{ scale: 1.03, y: -4 }} whileTap={{ scale: 0.98 }}>
          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardDescription className="flex items-center gap-1.5">
                  <TrendingUp className="h-4 w-4" />
                  近 7 天
                </CardDescription>
              </div>
              <CardTitle className="text-2xl"><Num value={data.week.requestCount} /></CardTitle>
            </CardHeader>
            <CardContent className="pb-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <DollarSign className="h-3 w-3" />
                ${weekCost.toFixed(4)}
              </p>
            </CardContent>
          </Card>
        </motion.div>

        {/* 30 days */}
        <motion.div variants={staggerItem} whileHover={{ scale: 1.03, y: -4 }} whileTap={{ scale: 0.98 }}>
          <Card className="bg-gradient-to-br from-green-500/10 to-emerald-500/10 border-green-500/20">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardDescription className="flex items-center gap-1.5">
                  <Activity className="h-4 w-4" />
                  近 30 天
                </CardDescription>
              </div>
              <CardTitle className="text-2xl"><Num value={data.month.requestCount} /></CardTitle>
            </CardHeader>
            <CardContent className="pb-3">
              <p className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1 font-medium">
                <DollarSign className="h-3 w-3" />
                ${parseFloat(data.month.costUsd).toFixed(4)}
              </p>
            </CardContent>
          </Card>
        </motion.div>
      </motion.div>

      {/* Token summary for month */}
      <motion.div
        variants={staggerContainer}
        initial="hidden"
        animate="visible"
        className="grid gap-4 md:grid-cols-4"
      >
        {[
          { label: '输入 Tokens (30天)', value: data.month.inputTokensSum, icon: ArrowUpRight, color: 'text-orange-500' },
          { label: '输出 Tokens (30天)', value: data.month.outputTokensSum, icon: ArrowDownRight, color: 'text-purple-500' },
          { label: '总请求 (30天)', value: data.month.requestCount, icon: Hash, color: 'text-blue-500' },
          { label: '错误数 (30天)', value: data.month.errorCount, icon: AlertTriangle, color: data.month.errorCount > 0 ? 'text-red-500' : 'text-muted-foreground' },
        ].map((item) => (
          <motion.div key={item.label} variants={staggerItem} whileHover={{ scale: 1.03, y: -4 }}>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription className="flex items-center gap-1.5">
                  <item.icon className={`h-4 w-4 ${item.color}`} />
                  {item.label}
                </CardDescription>
                <CardTitle className="text-xl"><Num value={item.value} /></CardTitle>
              </CardHeader>
            </Card>
          </motion.div>
        ))}
      </motion.div>

      {/* Charts and Top Models */}
      <div className="grid gap-6 lg:grid-cols-5">
        {/* Daily Trend Chart */}
        <motion.div
          className="lg:col-span-3"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ type: 'spring', bounce: 0.2, duration: 0.6, delay: 0.2 }}
        >
          <Card>
            <CardHeader>
              <CardTitle className="text-base">每日趋势（近 14 天）</CardTitle>
              <CardDescription>每日请求数和成本变化</CardDescription>
            </CardHeader>
            <CardContent>
              {(!data.dailyTrend || data.dailyTrend.length === 0) ? (
                <p className="text-center text-muted-foreground py-8">暂无数据</p>
              ) : (
                <div className="space-y-4">
                  {/* Cost bars */}
                  <div>
                    <p className="text-xs text-muted-foreground mb-2 flex items-center gap-1">
                      <DollarSign className="h-3 w-3" /> 成本 (USD)
                    </p>
                    <div className="flex items-end gap-1 h-32">
                      {data.dailyTrend.map((d, i) => {
                        const height = maxTrendCost > 0 ? (d.costMicros / maxTrendCost) * 100 : 0
                        return (
                          <motion.div
                            key={d.date}
                            className="flex-1 flex flex-col items-center gap-1 group relative"
                            initial={{ scaleY: 0 }}
                            animate={{ scaleY: 1 }}
                            transition={{ delay: i * 0.03, type: 'spring', bounce: 0.2 }}
                            style={{ originY: 1 }}
                          >
                            <div
                              className="w-full rounded-t bg-gradient-to-t from-green-500 to-emerald-400 dark:from-green-600 dark:to-emerald-500 transition-all hover:opacity-80 cursor-pointer min-h-[2px]"
                              style={{ height: `${Math.max(height, 2)}%` }}
                              title={`${d.date}: $${parseFloat(d.costUsd).toFixed(4)}`}
                            />
                            <span className="text-[9px] text-muted-foreground truncate w-full text-center">
                              {d.date.slice(5)}
                            </span>
                          </motion.div>
                        )
                      })}
                    </div>
                  </div>

                  {/* Request bars */}
                  <div>
                    <p className="text-xs text-muted-foreground mb-2 flex items-center gap-1">
                      <Activity className="h-3 w-3" /> 请求数
                    </p>
                    <div className="flex items-end gap-1 h-24">
                      {data.dailyTrend.map((d, i) => {
                        const height = maxTrendReqs > 0 ? (d.requests / maxTrendReqs) * 100 : 0
                        return (
                          <motion.div
                            key={d.date}
                            className="flex-1 flex flex-col items-center gap-1"
                            initial={{ scaleY: 0 }}
                            animate={{ scaleY: 1 }}
                            transition={{ delay: i * 0.03 + 0.1, type: 'spring', bounce: 0.2 }}
                            style={{ originY: 1 }}
                          >
                            <div
                              className="w-full rounded-t bg-gradient-to-t from-blue-500 to-cyan-400 dark:from-blue-600 dark:to-cyan-500 transition-all hover:opacity-80 cursor-pointer min-h-[2px]"
                              style={{ height: `${Math.max(height, 2)}%` }}
                              title={`${d.date}: ${d.requests} 请求`}
                            />
                            <span className="text-[9px] text-muted-foreground truncate w-full text-center">
                              {d.date.slice(5)}
                            </span>
                          </motion.div>
                        )
                      })}
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>

        {/* Top Models */}
        <motion.div
          className="lg:col-span-2"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ type: 'spring', bounce: 0.2, duration: 0.6, delay: 0.3 }}
        >
          <Card className="h-full">
            <CardHeader>
              <CardTitle className="text-base">热门模型（30天）</CardTitle>
              <CardDescription>按请求次数排名</CardDescription>
            </CardHeader>
            <CardContent>
              {(!data.topModels || data.topModels.length === 0) ? (
                <p className="text-center text-muted-foreground py-8">暂无数据</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>模型</TableHead>
                      <TableHead className="text-right">请求数</TableHead>
                      <TableHead className="text-right">成本</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.topModels.map((m, i) => (
                      <motion.tr
                        key={m.model}
                        className="border-b transition-colors hover:bg-muted/50"
                        initial={{ opacity: 0, x: 10 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: i * 0.05 }}
                      >
                        <TableCell className="font-medium">
                          <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-[10px] px-1.5 py-0 font-mono shrink-0">
                              #{i + 1}
                            </Badge>
                            <span className="font-mono text-xs break-all">
                              {m.model}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="text-right"><Num value={m.requestCount} /></TableCell>
                        <TableCell className="text-right text-green-600 dark:text-green-400">
                          ${parseFloat(m.costUsd).toFixed(4)}
                        </TableCell>
                      </motion.tr>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>

      {/* Cache Hit Rates by Provider */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: 'spring', bounce: 0.2, duration: 0.6, delay: 0.35 }}
      >
        <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <DatabaseZap className="h-4 w-4" />
                缓存命中率（30天）
              </CardTitle>
              <CardDescription>按模型提供商分类的缓存使用情况</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-3">
                {(['Claude', 'OpenAI', 'Gemini'] as const).map((providerName) => {
                  const rate = (data.cacheHitRates || []).find((r: DashboardCacheHitRate) => r.provider === providerName)
                  const hitRate = rate ? parseFloat(rate.hitRate) : 0
                  const hasData = !!rate

                  const providerColors: Record<string, { bg: string; bar: string; text: string; border: string }> = {
                    'Claude': {
                      bg: 'from-orange-500/10 to-amber-500/10',
                      bar: 'bg-gradient-to-r from-orange-500 to-amber-400',
                      text: 'text-orange-600 dark:text-orange-400',
                      border: 'border-orange-500/20',
                    },
                    'OpenAI': {
                      bg: 'from-emerald-500/10 to-green-500/10',
                      bar: 'bg-gradient-to-r from-emerald-500 to-green-400',
                      text: 'text-emerald-600 dark:text-emerald-400',
                      border: 'border-emerald-500/20',
                    },
                    'Gemini': {
                      bg: 'from-blue-500/10 to-indigo-500/10',
                      bar: 'bg-gradient-to-r from-blue-500 to-indigo-400',
                      text: 'text-blue-600 dark:text-blue-400',
                      border: 'border-blue-500/20',
                    },
                  }

                  const colors = providerColors[providerName]

                  return (
                    <motion.div
                      key={providerName}
                      initial={{ opacity: 0, scale: 0.95 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{ delay: 0.1 }}
                      whileHover={{ scale: 1.02, y: -2 }}
                    >
                      <Card className={`bg-gradient-to-br ${colors.bg} ${colors.border}`}>
                        <CardHeader className="pb-3">
                          <div className="flex items-center justify-between">
                            <CardTitle className={`text-sm font-semibold ${colors.text}`}>
                              {providerName}
                            </CardTitle>
                            {hasData && (
                              <Badge variant="outline" className="text-[10px] font-mono">
                                <Num value={rate!.requestCount} /> 请求
                              </Badge>
                            )}
                          </div>
                        </CardHeader>
                        <CardContent className="space-y-3 pb-4">
                          {!hasData ? (
                            <p className="text-sm text-muted-foreground">暂无数据</p>
                          ) : (
                            <>
                              <div className="flex items-baseline gap-2">
                                <span className={`text-2xl font-bold ${colors.text}`}>
                                  {rate!.hitRate}%
                                </span>
                                <span className="text-xs text-muted-foreground">命中率</span>
                              </div>

                              <div className="w-full h-2.5 bg-muted rounded-full overflow-hidden">
                                <motion.div
                                  className={`h-full rounded-full ${colors.bar}`}
                                  initial={{ width: 0 }}
                                  animate={{ width: `${Math.min(hitRate, 100)}%` }}
                                  transition={{ delay: 0.3, type: 'spring', bounce: 0.2, duration: 0.8 }}
                                />
                              </div>

                              <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-muted-foreground">
                                <span>缓存读取</span>
                                <span className="text-right font-mono"><Num value={rate!.cacheReadTokens} /></span>
                                <span>缓存写入</span>
                                <span className="text-right font-mono"><Num value={rate!.cacheCreationTokens} /></span>
                                <span>输入 Tokens</span>
                                <span className="text-right font-mono"><Num value={rate!.totalInputTokens} /></span>
                              </div>
                            </>
                          )}
                        </CardContent>
                      </Card>
                    </motion.div>
                  )
                })}
              </div>
            </CardContent>
          </Card>
        </motion.div>
    </motion.div>
  )
}
