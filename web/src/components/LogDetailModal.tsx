import { useState, useEffect, useRef, useCallback } from 'react'
import { getAdminRequestLogDetail, RequestLogDetail } from '@/api/amp'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { motion, AnimatePresence } from '@/lib/motion'

interface LogDetailModalProps {
  logId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

const TAB_ITEMS = [
  { value: 'request-headers', label: '请求头' },
  { value: 'request-body', label: '请求体' },
  { value: 'translated-request', label: '转换请求' },
  { value: 'response-headers', label: '响应头' },
  { value: 'response-body', label: '响应体' },
  { value: 'translated-response', label: '转换响应' },
] as const

type TabValue = typeof TAB_ITEMS[number]['value']

function getBadgeInfo(detail: RequestLogDetail, value: TabValue): string | null {
  switch (value) {
    case 'request-headers':
      return detail.requestHeaders && Object.keys(detail.requestHeaders).length > 0
        ? String(Object.keys(detail.requestHeaders).length)
        : null
    case 'request-body':
      return detail.requestBody ? `${(detail.requestBody.length / 1024).toFixed(1)}KB` : null
    case 'translated-request':
      return detail.translatedRequestBody ? `${(detail.translatedRequestBody.length / 1024).toFixed(1)}KB` : null
    case 'response-headers':
      return detail.responseHeaders && Object.keys(detail.responseHeaders).length > 0
        ? String(Object.keys(detail.responseHeaders).length)
        : null
    case 'response-body':
      return detail.responseBody ? `${(detail.responseBody.length / 1024).toFixed(1)}KB` : null
    case 'translated-response':
      return detail.translatedResponseBody ? `${(detail.translatedResponseBody.length / 1024).toFixed(1)}KB` : null
  }
}

const HEIGHT_SPRING = { type: 'spring' as const, stiffness: 300, damping: 30, mass: 0.8 }

export function LogDetailModal({ logId, open, onOpenChange }: LogDetailModalProps) {
  const [detail, setDetail] = useState<RequestLogDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [activeTab, setActiveTab] = useState<TabValue>('request-headers')
  const [contentHeight, setContentHeight] = useState<number | 'auto'>('auto')
  const innerRef = useRef<HTMLDivElement>(null)

  const measureHeight = useCallback(() => {
    if (innerRef.current) {
      setContentHeight(innerRef.current.scrollHeight)
    }
  }, [])

  useEffect(() => {
    measureHeight()
  }, [activeTab, detail, measureHeight])

  useEffect(() => {
    if (!open || !logId) {
      setDetail(null)
      setError('')
      setActiveTab('request-headers')
      setContentHeight('auto')
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
      <motion.div
        className="space-y-1"
        initial="hidden"
        animate="visible"
        variants={{
          hidden: { opacity: 0 },
          visible: { opacity: 1, transition: { staggerChildren: 0.03 } },
        }}
      >
        {Object.entries(headers).map(([key, value]) => (
          <motion.div
            key={key}
            className="flex gap-2 text-sm font-mono"
            variants={{
              hidden: { opacity: 0, x: -12 },
              visible: { opacity: 1, x: 0, transition: { type: 'spring', bounce: 0.2, duration: 0.4 } },
            }}
          >
            <span className="font-semibold text-primary min-w-32">{key}:</span>
            <span className="text-muted-foreground break-all">{value}</span>
          </motion.div>
        ))}
      </motion.div>
    )
  }

  const formatBody = (body: string | undefined) => {
    if (!body) {
      return <p className="text-muted-foreground text-sm">无数据</p>
    }

    let content: string
    try {
      const parsed = JSON.parse(body)
      content = JSON.stringify(parsed, null, 2)
    } catch {
      content = body
    }

    return (
      <pre className="text-xs font-mono bg-muted p-4 rounded-lg overflow-auto max-h-96 whitespace-pre-wrap">
        {content}
      </pre>
    )
  }

  const getTabContent = (tab: TabValue): React.ReactNode => {
    if (!detail) return null
    switch (tab) {
      case 'request-headers': return formatHeaders(detail.requestHeaders)
      case 'request-body': return formatBody(detail.requestBody)
      case 'translated-request':
        return detail.translatedRequestBody
          ? formatBody(detail.translatedRequestBody)
          : <p className="text-muted-foreground text-sm">无转换数据（请求未经过格式转换）</p>
      case 'response-headers': return formatHeaders(detail.responseHeaders)
      case 'response-body': return formatBody(detail.responseBody)
      case 'translated-response':
        return detail.translatedResponseBody
          ? (
            <pre className="text-xs font-mono bg-muted p-4 rounded-lg overflow-auto max-h-96 whitespace-pre-wrap">
              {detail.translatedResponseBody}
            </pre>
          )
          : <p className="text-muted-foreground text-sm">无翻译数据（非翻译请求或流式响应未记录）</p>
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
            <motion.div
              className="flex items-center justify-center py-8"
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ type: 'spring', bounce: 0.3, duration: 0.5 }}
            >
              <div className="flex items-center gap-3 text-muted-foreground">
                <motion.div
                  className="h-5 w-5 rounded-full border-2 border-primary/30 border-t-primary"
                  animate={{ rotate: 360 }}
                  transition={{ duration: 0.8, repeat: Infinity, ease: 'linear' }}
                />
                <span className="text-sm font-medium">加载中...</span>
              </div>
            </motion.div>
          )}

          {error && (
            <motion.div
              className="rounded-md bg-destructive/10 border border-destructive/20 p-4 text-destructive"
              initial={{ opacity: 0, y: -8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ type: 'spring', bounce: 0.2, duration: 0.4 }}
            >
              {error}
            </motion.div>
          )}

          {!loading && !error && detail && (
            <Tabs
              value={activeTab}
              onValueChange={(v) => setActiveTab(v as TabValue)}
              className="h-full flex flex-col"
            >
              <motion.div
                initial="hidden"
                animate="visible"
                variants={{
                  hidden: { opacity: 0 },
                  visible: { opacity: 1, transition: { staggerChildren: 0.04, delayChildren: 0.1 } },
                }}
              >
                <TabsList className="grid w-full grid-cols-6">
                  {TAB_ITEMS.map((tab) => {
                    const badgeText = getBadgeInfo(detail, tab.value)
                    return (
                      <motion.div
                        key={tab.value}
                        variants={{
                          hidden: { opacity: 0, y: -8, scale: 0.9 },
                          visible: { opacity: 1, y: 0, scale: 1, transition: { type: 'spring', bounce: 0.3, duration: 0.45 } },
                        }}
                        whileHover={{ scale: 1.04 }}
                        whileTap={{ scale: 0.96 }}
                      >
                        <TabsTrigger value={tab.value} className="w-full relative">
                          {tab.label}
                          {badgeText && (
                            <Badge variant="secondary" className="ml-1 text-xs">
                              {badgeText}
                            </Badge>
                          )}
                        </TabsTrigger>
                      </motion.div>
                    )
                  })}
                </TabsList>
              </motion.div>

              {/* Outer wrapper: animates height explicitly with spring */}
              <motion.div
                className="mt-4 overflow-hidden rounded-lg"
                animate={{ height: contentHeight }}
                transition={HEIGHT_SPRING}
              >
                {/* Inner measurer: always auto height, ref measures it */}
                <AnimatePresence mode="wait" initial={false}>
                  <motion.div
                    key={activeTab}
                    ref={innerRef}
                    initial={{ opacity: 0, y: 16, filter: 'blur(4px)' }}
                    animate={{ opacity: 1, y: 0, filter: 'blur(0px)' }}
                    exit={{ opacity: 0, y: -12, filter: 'blur(4px)' }}
                    transition={{ type: 'spring', bounce: 0.2, duration: 0.45 }}
                    onAnimationComplete={measureHeight}
                    className="overflow-auto max-h-[55vh]"
                  >
                    {getTabContent(activeTab)}
                  </motion.div>
                </AnimatePresence>
              </motion.div>
            </Tabs>
          )}

          {!loading && !error && !detail && (
            <motion.div
              className="flex items-center justify-center py-8"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.2 }}
            >
              <p className="text-muted-foreground">日志详情不存在或已过期（5分钟后自动清理）</p>
            </motion.div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
