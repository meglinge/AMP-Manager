import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import {
  getRequestLogs, getAdminRequestLogs, getAdminDistinctModels,
  RequestLog,
} from '@/api/amp'
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
import { formatDate } from '@/lib/formatters'
import { Num } from '@/components/Num'
import { StatusBadge } from '@/components/StatusBadge'
import { LogDetailModal } from '@/components/LogDetailModal'
import { LogFilterBar, FilterValues, localToISO } from '@/components/LogFilterBar'
import { motion } from '@/lib/motion'
import { PageSizeSlider } from '@/components/PageSizeSlider'

interface Props {
  isAdmin: boolean
}

type RefreshInterval = 5 | 10 | 30 | 60

export default function RequestLogs({ isAdmin }: Props) {
  const [logs, setLogs] = useState<RequestLog[]>([])
  const [loading, setLoading] = useState(true)
  const [fetching, setFetching] = useState(false)
  const hasLoadedRef = useRef(false)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [pageSize, setPageSize] = useState(20)

  const [filters, setFilters] = useState<FilterValues>({ userId: '', model: '', from: '', to: '' })

  const [users, setUsers] = useState<UserInfo[]>([])
  const [models, setModels] = useState<string[]>([])

  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>(10)
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const abortControllerRef = useRef<AbortController | null>(null)

  const [selectedLogId, setSelectedLogId] = useState<string | null>(null)
  const [detailModalOpen, setDetailModalOpen] = useState(false)

  const userIdToUsername = useMemo(() => {
    const map = new Map<string, string>()
    users.forEach(u => map.set(u.id, u.username))
    return map
  }, [users])

  useEffect(() => {
    if (isAdmin) {
      Promise.all([listUsers(), getAdminDistinctModels()])
        .then(([usersRes, modelsRes]) => {
          setUsers(usersRes || [])
          setModels(modelsRes.models || [])
        })
        .catch(console.error)
    }
  }, [isAdmin])

  useEffect(() => {
    loadData()
  }, [page, pageSize, filters])

  useEffect(() => {
    if (autoRefresh) {
      refreshTimerRef.current = setInterval(loadData, refreshInterval * 1000)
    } else {
      if (refreshTimerRef.current) {
        clearInterval(refreshTimerRef.current)
        refreshTimerRef.current = null
      }
    }
    return () => {
      if (refreshTimerRef.current) clearInterval(refreshTimerRef.current)
    }
  }, [autoRefresh, refreshInterval, page, pageSize, filters])

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) abortControllerRef.current.abort()
    }
  }, [])

  const loadData = useCallback(async () => {
    if (abortControllerRef.current) abortControllerRef.current.abort()
    const controller = new AbortController()
    abortControllerRef.current = controller

    if (!hasLoadedRef.current) setLoading(true)
    setFetching(true)
    setError('')
    try {
      const params = {
        page,
        pageSize,
        model: filters.model || undefined,
        from: filters.from ? localToISO(filters.from) : undefined,
        to: filters.to ? localToISO(filters.to) : undefined,
      }
      const result = isAdmin
        ? await getAdminRequestLogs({ ...params, userId: filters.userId || undefined }, controller.signal)
        : await getRequestLogs(params, controller.signal)
      if (controller.signal.aborted) return
      setLogs(result.items || [])
      setTotal(result.total)
    } catch (err) {
      if (controller.signal.aborted) return
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false)
        setFetching(false)
        hasLoadedRef.current = true
      }
    }
  }, [isAdmin, page, pageSize, filters])

  const handleFilterChange = (newFilters: FilterValues) => {
    setFilters(newFilters)
    setPage(1)
  }

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize)
    setPage(1)
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <motion.div initial={{ opacity: 0, y: -20 }} animate={{ opacity: 1, y: 0 }} transition={{ type: 'spring', bounce: 0.2, duration: 0.6 }}>
        <h2 className="text-2xl font-bold tracking-tight">请求日志</h2>
        <p className="text-muted-foreground">
          {isAdmin ? '管理员视图：查看所有用户的 API 请求历史' : '查看 API 请求历史'}
        </p>
      </motion.div>

      <LogFilterBar
        isAdmin={isAdmin}
        users={users}
        models={models}
        values={filters}
        onChange={handleFilterChange}
      />

      {error && (
        <div className="rounded-md bg-red-50 p-4 text-red-700">{error}</div>
      )}

      <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ type: 'spring', bounce: 0.2, duration: 0.6, delay: 0.1 }}>
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>请求记录</CardTitle>
                <CardDescription>共 {total} 条记录</CardDescription>
              </div>
              <div className="flex items-center gap-3">
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
                <Button variant="outline" onClick={loadData}>刷新</Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-center text-muted-foreground py-8">加载中...</p>
            ) : logs.length === 0 && !fetching ? (
              <p className="text-center text-muted-foreground py-8">暂无请求记录</p>
            ) : (
              <>
                <div className={`relative overflow-auto max-h-[calc(100vh-320px)] min-h-[400px] rounded-md border transition-opacity ${fetching ? 'opacity-50 pointer-events-none' : ''}`}>
                <Table>
                  <TableHeader className="sticky top-0 z-10 bg-background">
                    <TableRow>
                      <TableHead>时间</TableHead>
                      {isAdmin && <TableHead>用户</TableHead>}
                      <TableHead>模型</TableHead>
                      <TableHead>思维等级</TableHead>
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
                        {isAdmin && (
                          <TableCell className="text-xs truncate max-w-24" title={log.username || userIdToUsername.get(log.userId) || log.userId}>
                            {log.username || userIdToUsername.get(log.userId) || (log.userId.slice(0, 8) + '...')}
                          </TableCell>
                        )}
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
                          {log.thinkingLevel ? (
                            <Badge variant="secondary" className="text-xs">{log.thinkingLevel}</Badge>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{log.method}</Badge>
                        </TableCell>
                        <TableCell>
                          {isAdmin ? (
                            <button
                              onClick={() => { setSelectedLogId(log.id); setDetailModalOpen(true) }}
                              className="cursor-pointer hover:opacity-80 transition-opacity"
                              title="点击查看请求详情"
                            >
                              <StatusBadge status={log.statusCode} />
                            </button>
                          ) : (
                            <StatusBadge status={log.statusCode} />
                          )}
                        </TableCell>
                        <TableCell className="text-right text-muted-foreground">
                          {log.latencyMs}ms
                        </TableCell>
                        <TableCell className="text-right"><Num value={log.inputTokens} /></TableCell>
                        <TableCell className="text-right"><Num value={log.outputTokens} /></TableCell>
                        <TableCell className="text-right"><Num value={log.cacheReadInputTokens} /></TableCell>
                        <TableCell className="text-right"><Num value={log.cacheCreationInputTokens} /></TableCell>
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
                </div>

                <div className="flex items-center justify-between mt-4">
                  <div className="flex items-center gap-4">
                    <p className="text-sm text-muted-foreground">
                      第 {page} 页，共 {totalPages} 页
                    </p>
                    <PageSizeSlider value={pageSize} onChange={handlePageSizeChange} />
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>
                      上一页
                    </Button>
                    <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>
                      下一页
                    </Button>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </motion.div>

      {isAdmin && (
        <LogDetailModal
          logId={selectedLogId}
          open={detailModalOpen}
          onOpenChange={setDetailModalOpen}
        />
      )}
    </motion.div>
  )
}
