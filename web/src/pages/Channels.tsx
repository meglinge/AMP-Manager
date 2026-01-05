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

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-gray-500">加载中...</div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      {error && (
        <div className="rounded bg-red-100 p-3 text-red-700">{error}</div>
      )}

      <div className="rounded-lg bg-white p-6 shadow-md">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-xl font-bold text-gray-800">渠道管理</h2>
          <button
            onClick={handleCreate}
            className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
          >
            添加渠道
          </button>
        </div>

        {showForm && (
          <div className="mb-6 rounded-lg border border-blue-200 bg-blue-50 p-4">
            <h3 className="mb-4 font-medium text-gray-800">
              {editingChannel ? '编辑渠道' : '添加渠道'}
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">类型 *</label>
                <select
                  value={formData.type}
                  onChange={(e) => handleTypeChange(e.target.value as ChannelType)}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                >
                  {CHANNEL_TYPES.map(t => (
                    <option key={t.value} value={t.value}>{t.label}</option>
                  ))}
                </select>
              </div>
              {formData.type === 'openai' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700">API 端点</label>
                  <select
                    value={formData.endpoint || 'chat_completions'}
                    onChange={(e) => setFormData(prev => ({ ...prev, endpoint: e.target.value as ChannelEndpoint }))}
                    className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                  >
                    {OPENAI_ENDPOINTS.map(e => (
                      <option key={e.value} value={e.value}>{e.label}</option>
                    ))}
                  </select>
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700">名称 *</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="例如：OpenAI-主力"
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Base URL *</label>
                <input
                  type="text"
                  value={formData.baseUrl}
                  onChange={(e) => setFormData(prev => ({ ...prev, baseUrl: e.target.value }))}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  API Key {editingChannel ? '(留空不更新)' : '*'}
                </label>
                <input
                  type="password"
                  value={formData.apiKey}
                  onChange={(e) => setFormData(prev => ({ ...prev, apiKey: e.target.value }))}
                  placeholder={editingChannel ? '留空保持原值' : ''}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">权重</label>
                <input
                  type="number"
                  min="1"
                  value={formData.weight}
                  onChange={(e) => setFormData(prev => ({ ...prev, weight: parseInt(e.target.value) || 1 }))}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">优先级 (越小越优先)</label>
                <input
                  type="number"
                  min="1"
                  value={formData.priority}
                  onChange={(e) => setFormData(prev => ({ ...prev, priority: parseInt(e.target.value) || 100 }))}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
              </div>
              <div className="col-span-2 flex items-center gap-2">
                <input
                  type="checkbox"
                  id="enabled"
                  checked={formData.enabled}
                  onChange={(e) => setFormData(prev => ({ ...prev, enabled: e.target.checked }))}
                  className="h-4 w-4 rounded border-gray-300"
                />
                <label htmlFor="enabled" className="text-sm text-gray-700">启用</label>
              </div>

              <div className="col-span-2">
                <div className="flex items-center justify-between">
                  <label className="block text-sm font-medium text-gray-700">模型规则 (留空则按类型自动匹配)</label>
                  <button
                    type="button"
                    onClick={handleAddModel}
                    className="text-sm text-blue-600 hover:text-blue-800"
                  >
                    + 添加规则
                  </button>
                </div>
                {formData.models && formData.models.length > 0 && (
                  <div className="mt-2 space-y-2">
                    {formData.models.map((model, index) => (
                      <div key={index} className="flex items-center gap-2">
                        <input
                          type="text"
                          value={model.name}
                          onChange={(e) => handleModelChange(index, 'name', e.target.value)}
                          placeholder="模型名 (支持 * 通配符)"
                          className="flex-1 rounded-md border border-gray-300 px-3 py-1 text-sm"
                        />
                        <input
                          type="text"
                          value={model.alias || ''}
                          onChange={(e) => handleModelChange(index, 'alias', e.target.value)}
                          placeholder="别名 (可选)"
                          className="flex-1 rounded-md border border-gray-300 px-3 py-1 text-sm"
                        />
                        <button
                          type="button"
                          onClick={() => handleRemoveModel(index)}
                          className="text-red-500 hover:text-red-700"
                        >
                          删除
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div className="mt-4 flex gap-2">
              <button
                onClick={handleSubmit}
                disabled={saving}
                className="rounded-md bg-green-600 px-4 py-2 text-white hover:bg-green-700 disabled:bg-gray-300"
              >
                {saving ? '保存中...' : '保存'}
              </button>
              <button
                onClick={() => setShowForm(false)}
                className="rounded-md border border-gray-300 px-4 py-2 text-gray-700 hover:bg-gray-100"
              >
                取消
              </button>
            </div>
          </div>
        )}

        {channels.length === 0 ? (
          <div className="rounded-md border border-dashed border-gray-300 p-8 text-center text-gray-500">
            暂无渠道，点击上方按钮添加
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">名称</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">类型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Base URL</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">状态</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">优先级/权重</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">更新时间</th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {channels.map((channel) => (
                  <tr key={channel.id}>
                    <td className="whitespace-nowrap px-4 py-3 text-sm font-medium text-gray-900">
                      {channel.name}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      <span className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                        channel.type === 'gemini' ? 'bg-blue-100 text-blue-800' :
                        channel.type === 'claude' ? 'bg-orange-100 text-orange-800' :
                        'bg-green-100 text-green-800'
                      }`}>
                        {channel.type.toUpperCase()}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 max-w-xs truncate" title={channel.baseUrl}>
                      {channel.baseUrl}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <button
                        onClick={() => handleToggleEnabled(channel.id, !channel.enabled)}
                        className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                          channel.enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}
                      >
                        {channel.enabled ? '启用' : '禁用'}
                      </button>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {channel.priority} / {channel.weight}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {formatDate(channel.updatedAt)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-right text-sm space-x-2">
                      <button
                        onClick={() => handleTest(channel.id)}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        测试
                      </button>
                      <button
                        onClick={() => handleFetchModels(channel.id)}
                        disabled={fetchingModels[channel.id]}
                        className="text-purple-600 hover:text-purple-800 disabled:text-gray-400"
                      >
                        {fetchingModels[channel.id] ? '获取中...' : '获取模型'}
                        {modelCounts[channel.id] !== undefined && (
                          <span className="ml-1 text-xs text-gray-500">({modelCounts[channel.id]})</span>
                        )}
                      </button>
                      <button
                        onClick={() => handleEdit(channel)}
                        className="text-gray-600 hover:text-gray-800"
                      >
                        编辑
                      </button>
                      <button
                        onClick={() => handleDelete(channel.id, channel.name)}
                        className="text-red-600 hover:text-red-800"
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {Object.keys(testResults).length > 0 && (
          <div className="mt-4 space-y-2">
            {Object.entries(testResults).map(([id, result]) => {
              const channel = channels.find(c => c.id === id)
              return (
                <div
                  key={id}
                  className={`rounded p-2 text-sm ${
                    result.success ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  }`}
                >
                  <strong>{channel?.name}:</strong> {result.message}
                  {result.latencyMs && ` (${result.latencyMs}ms)`}
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">渠道路由说明</h3>
        <div className="space-y-2 text-sm text-gray-600">
          <p>• <strong>模型匹配</strong>：根据请求中的 model 字段自动选择合适的渠道</p>
          <p>• <strong>默认匹配</strong>：如果未配置模型规则，将根据类型自动匹配（gemini-* → Gemini, claude-* → Claude, gpt-* → OpenAI）</p>
          <p>• <strong>负载均衡</strong>：多个相同类型渠道时，按优先级和权重进行轮询分配</p>
          <p>• <strong>回退机制</strong>：如果没有匹配的渠道，请求将转发到用户配置的上游地址</p>
        </div>
      </div>
    </div>
  )
}
