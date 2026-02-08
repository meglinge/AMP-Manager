import { useState, useEffect, useRef } from 'react'
import {
  uploadDatabase,
  downloadDatabase,
  listBackups,
  restoreBackup,
  deleteBackup,
  Backup,
  getRetryConfig,
  updateRetryConfig,
  RetryConfig,
  getRequestDetailEnabled,
  updateRequestDetailEnabled,
  getTimeoutConfig,
  updateTimeoutConfig,
  TimeoutConfig,
  getCacheTTLConfig,
  updateCacheTTLConfig,
} from '../api/system'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export default function SystemSettings() {
  const [backups, setBackups] = useState<Backup[]>([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [retryConfig, setRetryConfig] = useState<RetryConfig | null>(null)
  const [retryLoading, setRetryLoading] = useState(false)
  const [requestDetailEnabled, setRequestDetailEnabled] = useState(true)
  const [requestDetailLoading, setRequestDetailLoading] = useState(false)
  const [timeoutConfig, setTimeoutConfig] = useState<TimeoutConfig | null>(null)
  const [timeoutLoading, setTimeoutLoading] = useState(false)
  const [cacheTTL, setCacheTTL] = useState<string>('1h')
  const [cacheTTLLoading, setCacheTTLLoading] = useState(false)

  useEffect(() => {
    fetchBackups()
    fetchRetryConfig()
    fetchRequestDetailEnabled()
    fetchTimeoutConfig()
    fetchCacheTTLConfig()
  }, [])

  const fetchBackups = async () => {
    try {
      const data = await listBackups()
      setBackups(data)
    } catch (err) {
      console.error('获取备份列表失败:', err)
    }
  }

  const fetchRetryConfig = async () => {
    try {
      const data = await getRetryConfig()
      setRetryConfig(data)
    } catch (err) {
      console.error('获取重试配置失败:', err)
    }
  }

  const fetchRequestDetailEnabled = async () => {
    try {
      const data = await getRequestDetailEnabled()
      setRequestDetailEnabled(data.enabled)
    } catch (err) {
      console.error('获取请求详情监控配置失败:', err)
    }
  }

  const fetchTimeoutConfig = async () => {
    try {
      const data = await getTimeoutConfig()
      setTimeoutConfig(data)
    } catch (err) {
      console.error('获取超时配置失败:', err)
    }
  }

  const fetchCacheTTLConfig = async () => {
    try {
      const data = await getCacheTTLConfig()
      setCacheTTL(data.cacheTTL)
    } catch (err) {
      console.error('获取缓存TTL配置失败:', err)
    }
  }

  const handleCacheTTLChange = async (value: string) => {
    setCacheTTLLoading(true)
    try {
      await updateCacheTTLConfig(value)
      setCacheTTL(value)
      showMessage('success', '缓存 TTL 配置已更新')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '保存失败')
    } finally {
      setCacheTTLLoading(false)
    }
  }

  const handleTimeoutConfigChange = (key: keyof TimeoutConfig, value: number) => {
    if (timeoutConfig) {
      setTimeoutConfig({ ...timeoutConfig, [key]: value })
    }
  }

  const handleSaveTimeoutConfig = async () => {
    if (!timeoutConfig) return
    
    setTimeoutLoading(true)
    try {
      await updateTimeoutConfig(timeoutConfig)
      showMessage('success', '超时配置已保存')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '保存失败')
    } finally {
      setTimeoutLoading(false)
    }
  }

  const handleRequestDetailToggle = async (enabled: boolean) => {
    setRequestDetailLoading(true)
    try {
      await updateRequestDetailEnabled(enabled)
      setRequestDetailEnabled(enabled)
      showMessage('success', enabled ? '请求详情监控已启用' : '请求详情监控已停止')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '操作失败')
    } finally {
      setRequestDetailLoading(false)
    }
  }

  const handleRetryConfigChange = (key: keyof RetryConfig, value: boolean | number) => {
    if (retryConfig) {
      setRetryConfig({ ...retryConfig, [key]: value })
    }
  }

  const handleSaveRetryConfig = async () => {
    if (!retryConfig) return
    
    setRetryLoading(true)
    try {
      await updateRetryConfig(retryConfig)
      showMessage('success', '重试配置已保存')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '保存失败')
    } finally {
      setRetryLoading(false)
    }
  }

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 5000)
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (!file.name.endsWith('.db')) {
      showMessage('error', '请选择 .db 数据库文件')
      return
    }

    if (!confirm('上传新数据库将覆盖现有数据，确定继续吗？')) {
      if (fileInputRef.current) fileInputRef.current.value = ''
      return
    }

    setLoading(true)
    try {
      const result = await uploadDatabase(file)
      showMessage('success', result.message)
      fetchBackups()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '上传失败')
    } finally {
      setLoading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleDownload = async () => {
    setLoading(true)
    try {
      await downloadDatabase()
      showMessage('success', '数据库下载成功')
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '下载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRestore = async (filename: string) => {
    if (!confirm(`确定要恢复备份 ${filename} 吗？当前数据将被备份。`)) {
      return
    }

    setLoading(true)
    try {
      const result = await restoreBackup(filename)
      showMessage('success', result.message)
      fetchBackups()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '恢复失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (filename: string) => {
    if (!confirm(`确定要删除备份 ${filename} 吗？`)) {
      return
    }

    setLoading(true)
    try {
      await deleteBackup(filename)
      showMessage('success', '备份已删除')
      fetchBackups()
    } catch (err) {
      showMessage('error', err instanceof Error ? err.message : '删除失败')
    } finally {
      setLoading(false)
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  return (
    <div className="space-y-6">
      {message && (
        <Alert variant={message.type === 'error' ? 'destructive' : 'default'}>
          <AlertDescription>{message.text}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>数据库管理</CardTitle>
          <CardDescription>上传、下载和管理系统数据库</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-4">
            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".db"
                onChange={handleUpload}
                className="hidden"
                id="db-upload"
                disabled={loading}
              />
              <Button asChild disabled={loading}>
                <label htmlFor="db-upload" className="cursor-pointer">
                  上传数据库
                </label>
              </Button>
            </div>

            <Button variant="secondary" onClick={handleDownload} disabled={loading}>
              下载当前数据库
            </Button>
          </div>

          <p className="text-sm text-muted-foreground">
            上传新数据库将自动备份当前数据库。更改生效需要重启服务。
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>备份列表</CardTitle>
          <CardDescription>查看和管理数据库备份</CardDescription>
        </CardHeader>
        <CardContent>
          {backups.length === 0 ? (
            <div className="rounded-md border border-dashed p-8 text-center text-muted-foreground">
              暂无备份
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>文件名</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>备份时间</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {backups.map((backup) => (
                    <TableRow key={backup.filename}>
                      <TableCell className="font-mono text-xs">{backup.filename}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{formatSize(backup.size)}</Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(backup.modTime)}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleRestore(backup.filename)}
                            disabled={loading}
                          >
                            恢复
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => handleDelete(backup.filename)}
                            disabled={loading}
                          >
                            删除
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>重试配置</CardTitle>
          <CardDescription>配置请求失败时的自动重试策略（首包门控）</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {retryConfig ? (
            <>
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>启用重试</Label>
                  <p className="text-sm text-muted-foreground">在首包到达前自动重试失败的请求</p>
                </div>
                <Switch
                  checked={retryConfig.enabled}
                  onCheckedChange={(checked) => handleRetryConfigChange('enabled', checked)}
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>最大重试次数</Label>
                  <Input
                    type="number"
                    min={1}
                    value={retryConfig.maxAttempts}
                    onChange={(e) => handleRetryConfigChange('maxAttempts', parseInt(e.target.value) || 1)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>首包超时 (毫秒)</Label>
                  <Input
                    type="number"
                    min={1000}
                    value={retryConfig.gateTimeoutMs}
                    onChange={(e) => handleRetryConfigChange('gateTimeoutMs', parseInt(e.target.value) || 10000)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>退避基数 (毫秒)</Label>
                  <Input
                    type="number"
                    min={50}
                    value={retryConfig.backoffBaseMs}
                    onChange={(e) => handleRetryConfigChange('backoffBaseMs', parseInt(e.target.value) || 100)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>退避上限 (毫秒)</Label>
                  <Input
                    type="number"
                    min={500}
                    value={retryConfig.backoffMaxMs}
                    onChange={(e) => handleRetryConfigChange('backoffMaxMs', parseInt(e.target.value) || 2000)}
                  />
                </div>
              </div>

              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>429 时重试</Label>
                    <p className="text-sm text-muted-foreground">请求被限流时自动重试</p>
                  </div>
                  <Switch
                    checked={retryConfig.retryOn429}
                    onCheckedChange={(checked) => handleRetryConfigChange('retryOn429', checked)}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>5xx 时重试</Label>
                    <p className="text-sm text-muted-foreground">服务端错误时自动重试</p>
                  </div>
                  <Switch
                    checked={retryConfig.retryOn5xx}
                    onCheckedChange={(checked) => handleRetryConfigChange('retryOn5xx', checked)}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>尊重 Retry-After</Label>
                    <p className="text-sm text-muted-foreground">按服务器返回的等待时间退避</p>
                  </div>
                  <Switch
                    checked={retryConfig.respectRetryAfter}
                    onCheckedChange={(checked) => handleRetryConfigChange('respectRetryAfter', checked)}
                  />
                </div>
              </div>

              <Button onClick={handleSaveRetryConfig} disabled={retryLoading}>
                {retryLoading ? '保存中...' : '保存配置'}
              </Button>
            </>
          ) : (
            <div className="text-center text-muted-foreground py-4">加载中...</div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>请求详情监控</CardTitle>
          <CardDescription>控制是否记录请求和响应的详细信息（头部和正文）</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>启用详情监控</Label>
              <p className="text-sm text-muted-foreground">
                启用后可在日志页面点击状态列查看请求/响应详情。关闭可减少内存使用。
              </p>
            </div>
            <Switch
              checked={requestDetailEnabled}
              onCheckedChange={handleRequestDetailToggle}
              disabled={requestDetailLoading}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>缓存 TTL 覆盖</CardTitle>
          <CardDescription>控制发送给 Claude API 的 cache_control TTL 值</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-3">
            <Label>TTL 策略</Label>
            <div className="flex gap-2">
              <Button
                variant={cacheTTL === '1h' ? 'default' : 'outline'}
                size="sm"
                onClick={() => handleCacheTTLChange('1h')}
                disabled={cacheTTLLoading}
              >
                1 小时
              </Button>
              <Button
                variant={cacheTTL === '5m' ? 'default' : 'outline'}
                size="sm"
                onClick={() => handleCacheTTLChange('5m')}
                disabled={cacheTTLLoading}
              >
                5 分钟
              </Button>
              <Button
                variant={cacheTTL === '' ? 'default' : 'outline'}
                size="sm"
                onClick={() => handleCacheTTLChange('')}
                disabled={cacheTTLLoading}
              >
                不覆盖
              </Button>
            </div>
            <p className="text-sm text-muted-foreground">
              选择"1小时"将强制所有 cache_control TTL 为 1h（省钱），"5分钟"为原始值，"不覆盖"保留请求原始 TTL 不做修改。
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>超时配置</CardTitle>
          <CardDescription>配置代理连接和流式响应的超时时间（单位：秒）</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {timeoutConfig ? (
            <>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>空闲连接超时</Label>
                  <Input
                    type="number"
                    min={30}
                    value={timeoutConfig.idleConnTimeoutSec}
                    onChange={(e) => handleTimeoutConfigChange('idleConnTimeoutSec', parseInt(e.target.value) || 300)}
                  />
                  <p className="text-xs text-muted-foreground">连接池中空闲连接的最大存活时间（&gt;=30秒）</p>
                </div>
                <div className="space-y-2">
                  <Label>读取空闲超时</Label>
                  <Input
                    type="number"
                    min={60}
                    value={timeoutConfig.readIdleTimeoutSec}
                    onChange={(e) => handleTimeoutConfigChange('readIdleTimeoutSec', parseInt(e.target.value) || 300)}
                  />
                  <p className="text-xs text-muted-foreground">AI 思考时无数据的最大等待时间（&gt;=60秒）</p>
                </div>
                <div className="space-y-2">
                  <Label>心跳间隔</Label>
                  <Input
                    type="number"
                    min={5}
                    value={timeoutConfig.keepAliveIntervalSec}
                    onChange={(e) => handleTimeoutConfigChange('keepAliveIntervalSec', parseInt(e.target.value) || 15)}
                  />
                  <p className="text-xs text-muted-foreground">SSE 流心跳发送间隔（&gt;=5秒）</p>
                </div>
                <div className="space-y-2">
                  <Label>连接超时</Label>
                  <Input
                    type="number"
                    min={5}
                    value={timeoutConfig.dialTimeoutSec}
                    onChange={(e) => handleTimeoutConfigChange('dialTimeoutSec', parseInt(e.target.value) || 30)}
                  />
                  <p className="text-xs text-muted-foreground">建立 TCP 连接的超时时间（&gt;=5秒）</p>
                </div>
                <div className="space-y-2">
                  <Label>TLS 握手超时</Label>
                  <Input
                    type="number"
                    min={5}
                    value={timeoutConfig.tlsHandshakeTimeoutSec}
                    onChange={(e) => handleTimeoutConfigChange('tlsHandshakeTimeoutSec', parseInt(e.target.value) || 15)}
                  />
                  <p className="text-xs text-muted-foreground">TLS 握手的超时时间（&gt;=5秒）</p>
                </div>
              </div>

              <Button onClick={handleSaveTimeoutConfig} disabled={timeoutLoading}>
                {timeoutLoading ? '保存中...' : '保存配置'}
              </Button>
            </>
          ) : (
            <div className="text-center text-muted-foreground py-4">加载中...</div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
