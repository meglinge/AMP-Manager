import { useState, useEffect } from 'react'
import {
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  setChannelEnabled,
  testChannel,
  Channel,
  ChannelRequest,
  ChannelType,
  ChannelEndpoint,
  ChannelModel,
  TestChannelResult,
} from '../api/channels'
import { fetchChannelModels } from '../api/models'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
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

const CHANNEL_TYPES: { value: ChannelType; label: string; defaultUrl: string; defaultEndpoint: ChannelEndpoint }[] = [
  { value: 'gemini', label: 'Gemini', defaultUrl: 'https://generativelanguage.googleapis.com', defaultEndpoint: 'generate_content' },
  { value: 'claude', label: 'Claude', defaultUrl: 'https://api.anthropic.com', defaultEndpoint: 'messages' },
  { value: 'openai', label: 'OpenAI', defaultUrl: 'https://api.openai.com', defaultEndpoint: 'chat_completions' },
]

const OPENAI_ENDPOINTS: { value: ChannelEndpoint; label: string }[] = [
  { value: 'chat_completions', label: '/v1/chat/completions' },
  { value: 'responses', label: '/v1/responses' },
]

export default function Channels() {
  const [channels, setChannels] = useState<Channel[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null)
  const [testResults, setTestResults] = useState<Record<string, TestChannelResult>>({})
  const [fetchingModels, setFetchingModels] = useState<Record<string, boolean>>({})
  const [modelCounts, setModelCounts] = useState<Record<string, number>>({})

  const [formData, setFormData] = useState<ChannelRequest>({
    type: 'openai',
    endpoint: 'chat_completions',
    name: '',
    baseUrl: 'https://api.openai.com',
    apiKey: '',
    enabled: true,
    weight: 1,
    priority: 100,
    models: [],
    headers: {},
  })
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    loadChannels()
  }, [])

  const loadChannels = async () => {
    try {
      const data = await listChannels()
      setChannels(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleTypeChange = (type: ChannelType) => {
    const typeInfo = CHANNEL_TYPES.find(t => t.value === type)
    setFormData(prev => ({
      ...prev,
      type,
      baseUrl: typeInfo?.defaultUrl || prev.baseUrl,
      endpoint: typeInfo?.defaultEndpoint || 'chat_completions',
    }))
  }

  const handleCreate = () => {
    setEditingChannel(null)
    setFormData({
      type: 'openai',
      endpoint: 'chat_completions',
      name: '',
      baseUrl: 'https://api.openai.com',
      apiKey: '',
      enabled: true,
      weight: 1,
      priority: 100,
      models: [],
      headers: {},
    })
    setShowForm(true)
  }

  const handleEdit = (channel: Channel) => {
    setEditingChannel(channel)
    setFormData({
      type: channel.type,
      endpoint: channel.endpoint,
      name: channel.name,
      baseUrl: channel.baseUrl,
      apiKey: '',
      enabled: channel.enabled,
      weight: channel.weight,
      priority: channel.priority,
      models: channel.models,
      headers: channel.headers,
    })
    setShowForm(true)
  }

  const handleSubmit = async () => {
    if (!formData.name.trim() || !formData.baseUrl.trim()) {
      setError('请填写必填字段')
      return
    }

    setSaving(true)
    setError('')

    try {
      if (editingChannel) {
        await updateChannel(editingChannel.id, formData)
      } else {
        if (!formData.apiKey) {
          setError('创建渠道时 API Key 为必填')
          setSaving(false)
          return
        }
        await createChannel(formData)
      }
      setShowForm(false)
      loadChannels()
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`确定要删除渠道 "${name}" 吗？`)) return

    try {
      await deleteChannel(id)
      loadChannels()
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除失败')
    }
  }

  const handleToggleEnabled = async (id: string, enabled: boolean) => {
    try {
      await setChannelEnabled(id, enabled)
      loadChannels()
    } catch (err) {
      setError(err instanceof Error ? err.message : '更新失败')
    }
  }

  const handleTest = async (id: string) => {
    try {
      const result = await testChannel(id)
      setTestResults(prev => ({ ...prev, [id]: result }))
    } catch (err) {
      setTestResults(prev => ({
        ...prev,
        [id]: { success: false, message: err instanceof Error ? err.message : '测试失败' },
      }))
    }
  }

  const handleFetchModels = async (id: string) => {
    try {
      setFetchingModels(prev => ({ ...prev, [id]: true }))
      const result = await fetchChannelModels(id)
      setModelCounts(prev => ({ ...prev, [id]: result.count }))
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取模型失败')
    } finally {
      setFetchingModels(prev => ({ ...prev, [id]: false }))
    }
  }

  const handleAddModel = () => {
    setFormData(prev => ({
      ...prev,
      models: [...(prev.models || []), { name: '', alias: '' }],
    }))
  }

  const handleRemoveModel = (index: number) => {
    setFormData(prev => ({
      ...prev,
      models: prev.models?.filter((_, i) => i !== index) || [],
    }))
  }

  const handleModelChange = (index: number, field: keyof ChannelModel, value: string) => {
    setFormData(prev => ({
      ...prev,
      models: prev.models?.map((m, i) => (i === index ? { ...m, [field]: value } : m)) || [],
    }))
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  const getTypeBadgeVariant = (type: ChannelType) => {
    switch (type) {
      case 'gemini':
        return 'default'
      case 'claude':
        return 'secondary'
      case 'openai':
        return 'outline'
      default:
        return 'default'
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-muted-foreground">加载中...</div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <div>
            <CardTitle>渠道管理</CardTitle>
            <CardDescription>管理 API 代理渠道配置</CardDescription>
          </div>
          <Button onClick={handleCreate}>添加渠道</Button>
        </CardHeader>
        <CardContent>
          {channels.length === 0 ? (
            <div className="rounded-md border border-dashed p-8 text-center text-muted-foreground">
              暂无渠道，点击上方按钮添加
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>Base URL</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>优先级/权重</TableHead>
                    <TableHead>更新时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {channels.map((channel) => (
                    <TableRow key={channel.id}>
                      <TableCell className="font-medium">{channel.name}</TableCell>
                      <TableCell>
                        <Badge variant={getTypeBadgeVariant(channel.type)}>
                          {channel.type.toUpperCase()}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-xs truncate" title={channel.baseUrl}>
                        {channel.baseUrl}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={channel.enabled}
                            onCheckedChange={(checked) => handleToggleEnabled(channel.id, checked)}
                          />
                          <span className="text-sm text-muted-foreground">
                            {channel.enabled ? '启用' : '禁用'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>{channel.priority} / {channel.weight}</TableCell>
                      <TableCell>{formatDate(channel.updatedAt)}</TableCell>
                      <TableCell className="text-right space-x-2">
                        <Button variant="ghost" size="sm" onClick={() => handleTest(channel.id)}>
                          测试
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleFetchModels(channel.id)}
                          disabled={fetchingModels[channel.id]}
                        >
                          {fetchingModels[channel.id] ? '获取中...' : '获取模型'}
                          {modelCounts[channel.id] !== undefined && (
                            <span className="ml-1 text-xs text-muted-foreground">
                              ({modelCounts[channel.id]})
                            </span>
                          )}
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleEdit(channel)}>
                          编辑
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDelete(channel.id, channel.name)}
                        >
                          删除
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {Object.keys(testResults).length > 0 && (
            <div className="mt-4 space-y-2">
              {Object.entries(testResults).map(([id, result]) => {
                const channel = channels.find(c => c.id === id)
                return (
                  <Alert key={id} variant={result.success ? 'default' : 'destructive'}>
                    <AlertDescription>
                      <strong>{channel?.name}:</strong> {result.message}
                      {result.latencyMs && ` (${result.latencyMs}ms)`}
                    </AlertDescription>
                  </Alert>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingChannel ? '编辑渠道' : '添加渠道'}</DialogTitle>
            <DialogDescription>
              配置 API 代理渠道的基本信息和模型规则
            </DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-2 gap-4 py-4">
            <div className="space-y-2">
              <Label>类型 *</Label>
              <select
                value={formData.type}
                onChange={(e) => handleTypeChange(e.target.value as ChannelType)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {CHANNEL_TYPES.map(t => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </div>

            {formData.type === 'openai' && (
              <div className="space-y-2">
                <Label>Endpoint</Label>
                <select
                  value={formData.endpoint}
                  onChange={(e) => setFormData(prev => ({ ...prev, endpoint: e.target.value as ChannelEndpoint }))}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {OPENAI_ENDPOINTS.map(e => (
                    <option key={e.value} value={e.value}>{e.label}</option>
                  ))}
                </select>
              </div>
            )}

            <div className="space-y-2">
              <Label>名称 *</Label>
              <Input
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                placeholder="例如：OpenAI-主力"
              />
            </div>

            <div className="space-y-2">
              <Label>Base URL *</Label>
              <Input
                value={formData.baseUrl}
                onChange={(e) => setFormData(prev => ({ ...prev, baseUrl: e.target.value }))}
              />
            </div>

            <div className="space-y-2">
              <Label>API Key {editingChannel ? '(留空不更新)' : '*'}</Label>
              <Input
                type="password"
                value={formData.apiKey}
                onChange={(e) => setFormData(prev => ({ ...prev, apiKey: e.target.value }))}
                placeholder={editingChannel ? '留空保持原值' : ''}
              />
            </div>

            <div className="space-y-2">
              <Label>权重</Label>
              <Input
                type="number"
                min="1"
                value={formData.weight}
                onChange={(e) => setFormData(prev => ({ ...prev, weight: parseInt(e.target.value) || 1 }))}
              />
            </div>

            <div className="space-y-2">
              <Label>优先级 (越小越优先)</Label>
              <Input
                type="number"
                min="1"
                value={formData.priority}
                onChange={(e) => setFormData(prev => ({ ...prev, priority: parseInt(e.target.value) || 100 }))}
              />
            </div>

            <div className="flex items-center gap-2 col-span-2">
              <Switch
                id="enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) => setFormData(prev => ({ ...prev, enabled: checked }))}
              />
              <Label htmlFor="enabled">启用</Label>
            </div>

            <div className="col-span-2 space-y-2">
              <div className="flex items-center justify-between">
                <Label>模型规则 (留空则按类型自动匹配)</Label>
                <Button type="button" variant="link" size="sm" onClick={handleAddModel}>
                  + 添加规则
                </Button>
              </div>
              {formData.models && formData.models.length > 0 && (
                <div className="space-y-2">
                  {formData.models.map((model, index) => (
                    <div key={index} className="flex items-center gap-2">
                      <Input
                        value={model.name}
                        onChange={(e) => handleModelChange(index, 'name', e.target.value)}
                        placeholder="模型名 (支持 * 通配符)"
                        className="flex-1"
                      />
                      <Input
                        value={model.alias || ''}
                        onChange={(e) => handleModelChange(index, 'alias', e.target.value)}
                        placeholder="别名 (可选)"
                        className="flex-1"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        onClick={() => handleRemoveModel(index)}
                      >
                        删除
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowForm(false)}>
              取消
            </Button>
            <Button onClick={handleSubmit} disabled={saving}>
              {saving ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card>
        <CardHeader>
          <CardTitle>渠道路由说明</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>• <strong>模型匹配</strong>：根据请求中的 model 字段自动选择合适的渠道</p>
          <p>• <strong>默认匹配</strong>：如果未配置模型规则，将根据类型自动匹配（gemini-* → Gemini, claude-* → Claude, gpt-* → OpenAI）</p>
          <p>• <strong>负载均衡</strong>：多个相同类型渠道时，按优先级和权重进行轮询分配</p>
          <p>• <strong>回退机制</strong>：如果没有匹配的渠道，请求将转发到用户配置的上游地址</p>
        </CardContent>
      </Card>
    </div>
  )
}
