import { useState, useEffect, useRef, useCallback } from 'react'
import { getRequestLogs, getUsageSummary, RequestLog, UsageSummary } from '@/api/amp'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatDate, formatNumber } from '@/lib/formatters'
import { StatusBadge } from '@/components/StatusBadge'
import { UsageSummaryCards } from '@/components/UsageSummaryCards'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export default function RequestLogs() {
  const [logs, setLogs] = useState<RequestLog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [pageSize] = useState(20)
  
  // Filters
  const [modelFilter, setModelFilter] = useState('')
  
  // Summary
  const [summary, setSummary] = useState<UsageSummary[]>([])
  const [summaryGroupBy, setSummaryGroupBy] = useState('day')
  
  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null)
  const summaryAbortControllerRef = useRef<AbortController | null>(null)

  useEffect(() => {
    loadData()
  }, [page, modelFilter])

  useEffect(() => {
    loadSummary()
  }, [summaryGroupBy])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      if (summaryAbortControllerRef.current) {
        summaryAbortControllerRef.current.abort()
      }
    }
  }, [])

  const loadData = useCallback(async () => {
    // Cancel previous request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }
    const controller = new AbortController()
    abortControllerRef.current = controller

    setError('')
    setLoading(true)
    try {
      const result = await getRequestLogs({
        page,
        pageSize,
        model: modelFilter || undefined,
      }, controller.signal)
      if (controller.signal.aborted) return
      setLogs(result.items || [])
      setTotal(result.total)
    } catch (err) {
      if (controller.signal.aborted) return
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false)
      }
    }
  }, [page, pageSize, modelFilter])

  const loadSummary = useCallback(async () => {
    // Cancel previous request
    if (summaryAbortControllerRef.current) {
      summaryAbortControllerRef.current.abort()
    }
    const controller = new AbortController()
    summaryAbortControllerRef.current = controller

    try {
      const result = await getUsageSummary({ groupBy: summaryGroupBy }, controller.signal)
      if (controller.signal.aborted) return
      setSummary(result.items || [])
    } catch (err) {
      if (controller.signal.aborted) return
      console.error('Failed to load summary:', err)
    }
  }, [summaryGroupBy])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">请求日志</h2>
        <p className="text-muted-foreground">查看 API 请求历史和 Token 使用统计</p>
      </div>

      {error && (
        <div className="rounded-md bg-red-50 p-4 text-red-700">{error}</div>
      )}

      {/* 统计卡片 */}
      <UsageSummaryCards summary={summary} />

      {/* 使用量统计 */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>使用量统计</CardTitle>
              <CardDescription>按时间或模型分组的 Token 使用量</CardDescription>
            </div>
            <Select value={summaryGroupBy} onValueChange={setSummaryGroupBy}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="day">按日期</SelectItem>
                <SelectItem value="model">按模型</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          {summary.length === 0 ? (
            <p className="text-center text-muted-foreground py-4">暂无数据</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{summaryGroupBy === 'day' ? '日期' : '模型'}</TableHead>
                  <TableHead className="text-right">请求数</TableHead>
                  <TableHead className="text-right">输入</TableHead>
                  <TableHead className="text-right">输出</TableHead>
                  <TableHead className="text-right">缓存读</TableHead>
                  <TableHead className="text-right">缓存写</TableHead>
                  <TableHead className="text-right">成本</TableHead>
                  <TableHead className="text-right">错误</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {summary.slice(0, 10).map((s, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-medium">{s.groupKey}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.requestCount)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.inputTokensSum)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.outputTokensSum)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.cacheReadInputTokensSum)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.cacheCreationInputTokensSum)}</TableCell>
                    <TableCell className="text-right text-green-600 dark:text-green-400">
                      {s.costUsdSum ? `$${s.costUsdSum}` : '-'}
                    </TableCell>
                    <TableCell className="text-right">
                      {s.errorCount > 0 ? (
                        <Badge variant="destructive">{s.errorCount}</Badge>
                      ) : (
                        <span className="text-muted-foreground">0</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* 请求日志列表 */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>请求记录</CardTitle>
              <CardDescription>共 {total} 条记录</CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Input
                placeholder="按模型筛选..."
                value={modelFilter}
                onChange={(e) => {
                  setModelFilter(e.target.value)
                  setPage(1)
                }}
                className="w-48"
              />
              <Button variant="outline" onClick={loadData}>刷新</Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-center text-muted-foreground py-8">加载中...</p>
          ) : logs.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">暂无请求记录</p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>模型</TableHead>
                    <TableHead>方法</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="text-right">延迟</TableHead>
                    <TableHead className="text-right">输入</TableHead>
                    <TableHead className="text-right">输出</TableHead>
                    <TableHead className="text-right">缓存读</TableHead>
                    <TableHead className="text-right">缓存写</TableHead>
                    <TableHead className="text-right">成本</TableHead>
                    <TableHead>流式</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                        {formatDate(log.createdAt)}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium text-sm truncate max-w-32" title={log.mappedModel || log.originalModel}>
                            {log.mappedModel || log.originalModel || '-'}
                          </span>
                          {log.mappedModel && log.originalModel && log.mappedModel !== log.originalModel && (
                            <span className="text-xs text-muted-foreground truncate max-w-32" title={log.originalModel}>
                              ← {log.originalModel}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{log.method}</Badge>
                      </TableCell>
                      <TableCell><StatusBadge status={log.statusCode} /></TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {log.latencyMs}ms
                      </TableCell>
                      <TableCell className="text-right">{formatNumber(log.inputTokens)}</TableCell>
                      <TableCell className="text-right">{formatNumber(log.outputTokens)}</TableCell>
                      <TableCell className="text-right">{formatNumber(log.cacheReadInputTokens)}</TableCell>
                      <TableCell className="text-right">{formatNumber(log.cacheCreationInputTokens)}</TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {log.costUsd ? `$${log.costUsd}` : '-'}
                      </TableCell>
                      <TableCell>
                        {log.isStreaming ? (
                          <Badge variant="secondary">流式</Badge>
                        ) : null}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* 分页 */}
              <div className="flex items-center justify-between mt-4">
                <p className="text-sm text-muted-foreground">
                  第 {page} 页，共 {totalPages} 页
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage(p => p - 1)}
                  >
                    上一页
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= totalPages}
                    onClick={() => setPage(p => p + 1)}
                  >
                    下一页
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
