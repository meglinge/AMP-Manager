import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { getAdminRequestLogs, getAdminUsageSummary, getAdminDistinctModels, RequestLog, UsageSummary } from '@/api/amp'
import { listUsers, UserInfo } from '@/api/users'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
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
import { formatDate, formatNumber } from '@/lib/formatters'
import { StatusBadge } from '@/components/StatusBadge'
import { UsageSummaryCards } from '@/components/UsageSummaryCards'

type SummaryGroupBy = 'day' | 'model' | 'user'
type RefreshInterval = 5 | 10 | 30 | 60

export default function AdminRequestLogs() {
  const [logs, setLogs] = useState<RequestLog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [pageSize] = useState(20)
  
  // Filters
  const [modelFilter, setModelFilter] = useState('')
  const [userIdFilter, setUserIdFilter] = useState('')
  
  // Filter options
  const [users, setUsers] = useState<UserInfo[]>([])
  const [models, setModels] = useState<string[]>([])
  
  // Summary
  const [summary, setSummary] = useState<UsageSummary[]>([])
  const [summaryGroupBy, setSummaryGroupBy] = useState<SummaryGroupBy>('day')
  
  // Auto refresh
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>(10)
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const abortControllerRef = useRef<AbortController | null>(null)
  const summaryAbortControllerRef = useRef<AbortController | null>(null)

  // userId -> username 映射
  const userIdToUsername = useMemo(() => {
    const map = new Map<string, string>()
    users.forEach(u => map.set(u.id, u.username))
    return map
  }, [users])

  useEffect(() => {
    loadFilterOptions()
  }, [])

  useEffect(() => {
    loadData()
  }, [page, modelFilter, userIdFilter])

  useEffect(() => {
    loadSummary()
  }, [summaryGroupBy, userIdFilter])

  useEffect(() => {
    if (autoRefresh) {
      refreshTimerRef.current = setInterval(() => {
        loadData()
        loadSummary()
      }, refreshInterval * 1000)
    } else {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current)
        refreshTimerRef.current = null
      }
    }
    return () => {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current)
      }
    }
  }, [autoRefresh, refreshInterval, page, modelFilter, userIdFilter, summaryGroupBy])

  // 组件卸载时取消请求
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

  const loadFilterOptions = async () => {
    try {
      const [usersRes, modelsRes] = await Promise.all([
        listUsers(),
        getAdminDistinctModels(),
      ])
      setUsers(usersRes || [])
      setModels(modelsRes.models || [])
    } catch (err) {
      console.error('Failed to load filter options:', err)
    }
  }

  const loadData = useCallback(async () => {
    // 取消之前的请求
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }
    const controller = new AbortController()
    abortControllerRef.current = controller

    setLoading(true)
    setError('')
    try {
      const result = await getAdminRequestLogs({
        page,
        pageSize,
        model: modelFilter || undefined,
        userId: userIdFilter || undefined,
      }, controller.signal)
      // 检查请求是否已被取消
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
  }, [page, pageSize, modelFilter, userIdFilter])

  const loadSummary = useCallback(async () => {
    // 取消之前的请求
    if (summaryAbortControllerRef.current) {
      summaryAbortControllerRef.current.abort()
    }
    const controller = new AbortController()
    summaryAbortControllerRef.current = controller

    setError('')
    try {
      const result = await getAdminUsageSummary({ 
        groupBy: summaryGroupBy,
        userId: userIdFilter || undefined,
      }, controller.signal)
      // 检查请求是否已被取消
      if (controller.signal.aborted) return
      setSummary(result.items || [])
    } catch (err) {
      if (controller.signal.aborted) return
      console.error('Failed to load summary:', err)
    }
  }, [summaryGroupBy, userIdFilter])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 获取 summary 表格中的显示名称
  const getSummaryDisplayName = (groupKey: string): string => {
    if (summaryGroupBy === 'user') {
      return userIdToUsername.get(groupKey) || groupKey
    }
    return groupKey
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">全局请求日志</h2>
        <p className="text-muted-foreground">管理员视图：查看所有用户的 API 请求历史和 Token 使用统计</p>
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
              <CardDescription>按时间、模型或用户分组的 Token 使用量</CardDescription>
            </div>
            <Select value={summaryGroupBy} onValueChange={(v) => setSummaryGroupBy(v as SummaryGroupBy)}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="day">按日期</SelectItem>
                <SelectItem value="model">按模型</SelectItem>
                <SelectItem value="user">按用户</SelectItem>
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
                  <TableHead>{summaryGroupBy === 'day' ? '日期' : summaryGroupBy === 'model' ? '模型' : '用户'}</TableHead>
                  <TableHead className="text-right">请求数</TableHead>
                  <TableHead className="text-right">输入</TableHead>
                  <TableHead className="text-right">输出</TableHead>
                  <TableHead className="text-right">缓存读</TableHead>
                  <TableHead className="text-right">缓存写</TableHead>
                  <TableHead className="text-right">错误</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {summary.slice(0, 20).map((s, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-medium">{getSummaryDisplayName(s.groupKey)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.requestCount)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.inputTokensSum)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.outputTokensSum)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.cacheReadInputTokensSum)}</TableCell>
                    <TableCell className="text-right">{formatNumber(s.cacheCreationInputTokensSum)}</TableCell>
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
            <div className="flex items-center gap-3">
              <Select value={userIdFilter || 'all'} onValueChange={(v) => { setUserIdFilter(v === 'all' ? '' : v); setPage(1) }}>
                <SelectTrigger className="w-36" aria-label="按用户筛选">
                  <SelectValue placeholder="所有用户" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有用户</SelectItem>
                  {users.map(u => (
                    <SelectItem key={u.id} value={u.id}>{u.username}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={modelFilter || 'all'} onValueChange={(v) => { setModelFilter(v === 'all' ? '' : v); setPage(1) }}>
                <SelectTrigger className="w-48" aria-label="按模型筛选">
                  <SelectValue placeholder="所有模型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">所有模型</SelectItem>
                  {models.map(m => (
                    <SelectItem key={m} value={m}>{m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <div className="flex items-center gap-2 border-l pl-3">
                <Switch id="auto-refresh" checked={autoRefresh} onCheckedChange={setAutoRefresh} />
                <Label htmlFor="auto-refresh" className="text-sm">自动刷新</Label>
                {autoRefresh && (
                  <Select value={String(refreshInterval)} onValueChange={(v) => setRefreshInterval(Number(v) as RefreshInterval)}>
                    <SelectTrigger className="w-20">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="5">5秒</SelectItem>
                      <SelectItem value="10">10秒</SelectItem>
                      <SelectItem value="30">30秒</SelectItem>
                      <SelectItem value="60">60秒</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              </div>
              <Button variant="outline" onClick={() => { loadData(); loadSummary() }}>刷新</Button>
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
                    <TableHead>用户</TableHead>
                    <TableHead>模型</TableHead>
                    <TableHead>方法</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="text-right">延迟</TableHead>
                    <TableHead className="text-right">输入</TableHead>
                    <TableHead className="text-right">输出</TableHead>
                    <TableHead className="text-right">缓存读</TableHead>
                    <TableHead className="text-right">缓存写</TableHead>
                    <TableHead>流式</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                        {formatDate(log.createdAt)}
                      </TableCell>
                      <TableCell className="text-xs truncate max-w-24" title={log.username || userIdToUsername.get(log.userId) || log.userId}>
                        {log.username || userIdToUsername.get(log.userId) || (log.userId.slice(0, 8) + '...')}
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
