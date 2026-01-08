import { useState, useEffect } from 'react'
import { getAdminRequestLogDetail, RequestLogDetail } from '@/api/amp'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'

interface LogDetailModalProps {
  logId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function LogDetailModal({ logId, open, onOpenChange }: LogDetailModalProps) {
  const [detail, setDetail] = useState<RequestLogDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open || !logId) {
      setDetail(null)
      setError('')
      return
    }

    const controller = new AbortController()
    setLoading(true)
    setError('')

    getAdminRequestLogDetail(logId, controller.signal)
      .then(setDetail)
      .catch((err) => {
        if (err.name !== 'AbortError') {
          setError(err.message || '加载详情失败')
        }
      })
      .finally(() => setLoading(false))

    return () => controller.abort()
  }, [open, logId])

  const formatHeaders = (headers: Record<string, string> | undefined) => {
    if (!headers || Object.keys(headers).length === 0) {
      return <p className="text-muted-foreground text-sm">无数据</p>
    }
    return (
      <div className="space-y-1">
        {Object.entries(headers).map(([key, value]) => (
          <div key={key} className="flex gap-2 text-sm font-mono">
            <span className="font-semibold text-primary min-w-32">{key}:</span>
            <span className="text-muted-foreground break-all">{value}</span>
          </div>
        ))}
      </div>
    )
  }

  const formatBody = (body: string | undefined) => {
    if (!body) {
      return <p className="text-muted-foreground text-sm">无数据</p>
    }
    
    // Try to parse and format JSON
    try {
      const parsed = JSON.parse(body)
      return (
        <pre className="text-xs font-mono bg-muted p-4 rounded overflow-auto max-h-96 whitespace-pre-wrap">
          {JSON.stringify(parsed, null, 2)}
        </pre>
      )
    } catch {
      // Not JSON, show as plain text
      return (
        <pre className="text-xs font-mono bg-muted p-4 rounded overflow-auto max-h-96 whitespace-pre-wrap">
          {body}
        </pre>
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>请求详情</DialogTitle>
          <DialogDescription>
            {logId && (
              <span className="font-mono text-xs">
                Request ID: {logId}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-hidden">
          {loading && (
            <div className="flex items-center justify-center py-8">
              <p className="text-muted-foreground">加载中...</p>
            </div>
          )}

          {error && (
            <div className="rounded-md bg-red-50 p-4 text-red-700">
              {error}
            </div>
          )}

          {!loading && !error && detail && (
            <Tabs defaultValue="request-headers" className="h-full flex flex-col">
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="request-headers">
                  请求头
                  {detail.requestHeaders && Object.keys(detail.requestHeaders).length > 0 && (
                    <Badge variant="secondary" className="ml-1 text-xs">
                      {Object.keys(detail.requestHeaders).length}
                    </Badge>
                  )}
                </TabsTrigger>
                <TabsTrigger value="request-body">
                  请求体
                  {detail.requestBody && (
                    <Badge variant="secondary" className="ml-1 text-xs">
                      {(detail.requestBody.length / 1024).toFixed(1)}KB
                    </Badge>
                  )}
                </TabsTrigger>
                <TabsTrigger value="response-headers">
                  响应头
                  {detail.responseHeaders && Object.keys(detail.responseHeaders).length > 0 && (
                    <Badge variant="secondary" className="ml-1 text-xs">
                      {Object.keys(detail.responseHeaders).length}
                    </Badge>
                  )}
                </TabsTrigger>
                <TabsTrigger value="response-body">
                  响应体
                  {detail.responseBody && (
                    <Badge variant="secondary" className="ml-1 text-xs">
                      {(detail.responseBody.length / 1024).toFixed(1)}KB
                    </Badge>
                  )}
                </TabsTrigger>
              </TabsList>
              
              <div className="flex-1 overflow-auto mt-4">
                <TabsContent value="request-headers" className="m-0">
                  {formatHeaders(detail.requestHeaders)}
                </TabsContent>
                <TabsContent value="request-body" className="m-0">
                  {formatBody(detail.requestBody)}
                </TabsContent>
                <TabsContent value="response-headers" className="m-0">
                  {formatHeaders(detail.responseHeaders)}
                </TabsContent>
                <TabsContent value="response-body" className="m-0">
                  {formatBody(detail.responseBody)}
                </TabsContent>
              </div>
            </Tabs>
          )}

          {!loading && !error && !detail && (
            <div className="flex items-center justify-center py-8">
              <p className="text-muted-foreground">日志详情不存在或已过期（5分钟后自动清理）</p>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
