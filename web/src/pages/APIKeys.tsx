import { useState, useEffect } from 'react'
import {
  getAPIKeys,
  createAPIKey,
  revokeAPIKey,
  APIKey,
  CreateAPIKeyResponse,
} from '../api/amp'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

export default function APIKeys() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [createName, setCreateName] = useState('')
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState<CreateAPIKeyResponse | null>(null)
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const keysData = await getAPIKeys()
      setKeys(keysData)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async () => {
    if (!createName.trim()) return

    setCreating(true)
    setError('')

    try {
      const result = await createAPIKey(createName.trim())
      setNewKey(result)
      setCreateName('')
      setShowCreate(false)
      loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setCreating(false)
    }
  }

  const handleRevoke = async (id: string, name: string) => {
    if (!confirm(`确定要撤销 API Key "${name}" 吗？此操作不可恢复。`)) return

    try {
      await revokeAPIKey(id)
      loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : '撤销失败')
    }
  }

  const copyToClipboard = async (text: string, type: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(type)
    setTimeout(() => setCopied(null), 2000)
  }

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-muted-foreground">加载中...</div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {newKey && (
        <Card className="border-2 border-green-500 bg-green-50 dark:bg-green-950">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
            <CardTitle className="text-green-800 dark:text-green-200">API Key 创建成功</CardTitle>
            <Button variant="ghost" size="sm" onClick={() => setNewKey(null)}>
              关闭
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            <Alert className="bg-yellow-100 border-yellow-300 dark:bg-yellow-900 dark:border-yellow-700">
              <AlertDescription className="text-yellow-800 dark:text-yellow-200">
                ⚠️ 请立即复制以下信息，API Key 明文只显示一次！
              </AlertDescription>
            </Alert>

            <div className="space-y-3">
              <div className="space-y-2">
                <Label>API Key</Label>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded bg-white dark:bg-gray-800 p-2 text-sm font-mono break-all border">
                    {newKey.apiKey}
                  </code>
                  <Button
                    size="sm"
                    onClick={() => copyToClipboard(newKey.apiKey, 'apiKey')}
                  >
                    {copied === 'apiKey' ? '已复制' : '复制'}
                  </Button>
                </div>
              </div>

              <div className="space-y-2">
                <Label>使用方法 (Linux/macOS)</Label>
                <div className="rounded bg-gray-800 p-3 text-sm font-mono text-green-400">
                  <div>export AMP_URL="{window.location.origin}"</div>
                  <div>export AMP_API_KEY="{newKey.apiKey}"</div>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => copyToClipboard(`export AMP_URL="${window.location.origin}"\nexport AMP_API_KEY="${newKey.apiKey}"`, 'env')}
                >
                  {copied === 'env' ? '已复制' : '复制环境变量'}
                </Button>
              </div>

              <div className="space-y-2">
                <Label>Windows PowerShell (永久)</Label>
                <div className="rounded bg-gray-800 p-3 text-sm font-mono text-green-400">
                  <div>[Environment]::SetEnvironmentVariable("AMP_URL", "{window.location.origin}", "User")</div>
                  <div>[Environment]::SetEnvironmentVariable("AMP_API_KEY", "{newKey.apiKey}", "User")</div>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => copyToClipboard(`[Environment]::SetEnvironmentVariable("AMP_URL", "${window.location.origin}", "User")\n[Environment]::SetEnvironmentVariable("AMP_API_KEY", "${newKey.apiKey}", "User")`, 'ps')}
                >
                  {copied === 'ps' ? '已复制' : '复制 PowerShell 命令'}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <div>
            <CardTitle>API Key 管理</CardTitle>
            <CardDescription>管理用于 Amp CLI 认证的 API Key</CardDescription>
          </div>
          <Button onClick={() => setShowCreate(true)}>创建 API Key</Button>
        </CardHeader>
        <CardContent>
          {error && (
            <Alert variant="destructive" className="mb-4">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {keys.length === 0 ? (
            <div className="rounded-md border border-dashed p-8 text-center text-muted-foreground">
              暂无 API Key，点击上方按钮创建
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>Prefix</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>最后使用</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {keys.map((key) => (
                    <TableRow key={key.id}>
                      <TableCell className="font-medium">{key.name}</TableCell>
                      <TableCell className="font-mono text-muted-foreground">
                        {key.prefix}...
                      </TableCell>
                      <TableCell>
                        <Badge variant={key.isActive ? 'default' : 'destructive'}>
                          {key.isActive ? '活跃' : '已撤销'}
                        </Badge>
                      </TableCell>
                      <TableCell>{formatDate(key.lastUsedAt)}</TableCell>
                      <TableCell>{formatDate(key.createdAt)}</TableCell>
                      <TableCell className="text-right">
                        {key.isActive && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            onClick={() => handleRevoke(key.id, key.name)}
                          >
                            撤销
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>创建新 API Key</DialogTitle>
            <DialogDescription>
              为新设备或应用创建一个 API Key
            </DialogDescription>
          </DialogHeader>
          <div className="py-4 space-y-4">
            <div className="space-y-2">
              <Label htmlFor="keyName">API Key 名称</Label>
              <Input
                id="keyName"
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
                placeholder="输入 API Key 名称（如：工作电脑）"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowCreate(false)
                setCreateName('')
              }}
            >
              取消
            </Button>
            <Button
              onClick={handleCreate}
              disabled={creating || !createName.trim()}
            >
              {creating ? '创建中...' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card>
        <CardHeader>
          <CardTitle>使用说明</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <p>1. 创建一个 API Key 用于 Amp CLI 认证</p>
          <p>2. 在终端配置环境变量：</p>
          <div className="space-y-4">
            <div>
              <p className="mb-1 font-medium text-foreground">Linux/macOS:</p>
              <div className="rounded bg-gray-800 p-3 font-mono text-green-400">
                <div>export AMP_URL="{window.location.origin}"</div>
                <div>export AMP_API_KEY="your-api-key-here"</div>
              </div>
            </div>
            <div>
              <p className="mb-1 font-medium text-foreground">Windows PowerShell (永久):</p>
              <div className="rounded bg-gray-800 p-3 font-mono text-green-400">
                <div>[Environment]::SetEnvironmentVariable("AMP_URL", "{window.location.origin}", "User")</div>
                <div>[Environment]::SetEnvironmentVariable("AMP_API_KEY", "your-api-key-here", "User")</div>
              </div>
            </div>
          </div>
          <p>3. Amp CLI 会自动使用这些环境变量连接到反代服务</p>
        </CardContent>
      </Card>
    </div>
  )
}
