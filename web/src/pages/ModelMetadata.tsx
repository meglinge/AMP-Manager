import { useState, useEffect } from 'react'
import {
  listModelMetadata,
  createModelMetadata,
  updateModelMetadata,
  deleteModelMetadata,
  ModelMetadata,
  ModelMetadataRequest,
} from '../api/modelMetadata'

const PROVIDERS = ['anthropic', 'openai', 'google', 'deepseek', 'alibaba', 'other']

function formatTokenCount(count: number): string {
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(count % 1000000 === 0 ? 0 : 1)}M`
  }
  if (count >= 1000) {
    return `${(count / 1000).toFixed(count % 1000 === 0 ? 0 : 1)}k`
  }
  return count.toString()
}

export default function ModelMetadataPage() {
  const [metadata, setMetadata] = useState<ModelMetadata[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editingItem, setEditingItem] = useState<ModelMetadata | null>(null)

  const [formData, setFormData] = useState<ModelMetadataRequest>({
    modelPattern: '',
    displayName: '',
    contextLength: 200000,
    maxCompletionTokens: 8192,
    provider: 'anthropic',
  })
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    loadMetadata()
  }, [])

  const loadMetadata = async () => {
    try {
      const data = await listModelMetadata()
      setMetadata(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = () => {
    setEditingItem(null)
    setFormData({
      modelPattern: '',
      displayName: '',
      contextLength: 200000,
      maxCompletionTokens: 8192,
      provider: 'anthropic',
    })
    setShowForm(true)
  }

  const handleEdit = (item: ModelMetadata) => {
    setEditingItem(item)
    setFormData({
      modelPattern: item.modelPattern,
      displayName: item.displayName,
      contextLength: item.contextLength,
      maxCompletionTokens: item.maxCompletionTokens,
      provider: item.provider,
    })
    setShowForm(true)
  }

  const handleSubmit = async () => {
    if (!formData.modelPattern.trim() || !formData.displayName.trim()) {
      setError('请填写必填字段')
      return
    }

    setSaving(true)
    setError('')

    try {
      if (editingItem) {
        await updateModelMetadata(editingItem.id, formData)
      } else {
        await createModelMetadata(formData)
      }
      setShowForm(false)
      loadMetadata()
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string, pattern: string) => {
    if (!confirm(`确定要删除模型元数据 "${pattern}" 吗？`)) return

    try {
      await deleteModelMetadata(id)
      loadMetadata()
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除失败')
    }
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
          <h2 className="text-xl font-bold text-gray-800">模型元数据管理</h2>
          <button
            onClick={handleCreate}
            className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
          >
            添加模型元数据
          </button>
        </div>

        {showForm && (
          <div className="mb-6 rounded-lg border border-blue-200 bg-blue-50 p-4">
            <h3 className="mb-4 font-medium text-gray-800">
              {editingItem ? '编辑模型元数据' : '添加模型元数据'}
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">模型模式 *</label>
                <input
                  type="text"
                  value={formData.modelPattern}
                  onChange={(e) => setFormData(prev => ({ ...prev, modelPattern: e.target.value }))}
                  placeholder="例如：claude-sonnet 或精确模型名"
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
                <p className="mt-1 text-xs text-gray-500">前缀匹配，如 claude-sonnet 匹配 claude-sonnet-4-5-xxx</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">显示名称 *</label>
                <input
                  type="text"
                  value={formData.displayName}
                  onChange={(e) => setFormData(prev => ({ ...prev, displayName: e.target.value }))}
                  placeholder="例如：Claude Sonnet 4"
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">上下文长度 (tokens)</label>
                <input
                  type="number"
                  min="1"
                  value={formData.contextLength}
                  onChange={(e) => setFormData(prev => ({ ...prev, contextLength: parseInt(e.target.value) || 0 }))}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
                <p className="mt-1 text-xs text-gray-500">当前: {formatTokenCount(formData.contextLength)}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">最大输出 (tokens)</label>
                <input
                  type="number"
                  min="1"
                  value={formData.maxCompletionTokens}
                  onChange={(e) => setFormData(prev => ({ ...prev, maxCompletionTokens: parseInt(e.target.value) || 0 }))}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                />
                <p className="mt-1 text-xs text-gray-500">当前: {formatTokenCount(formData.maxCompletionTokens)}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">提供商</label>
                <select
                  value={formData.provider}
                  onChange={(e) => setFormData(prev => ({ ...prev, provider: e.target.value }))}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2"
                >
                  {PROVIDERS.map(p => (
                    <option key={p} value={p}>{p}</option>
                  ))}
                </select>
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

        {metadata.length === 0 ? (
          <div className="rounded-md border border-dashed border-gray-300 p-8 text-center text-gray-500">
            暂无模型元数据，点击上方按钮添加
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">模型模式</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">显示名称</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">上下文</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">最大输出</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">提供商</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">更新时间</th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {metadata.map((item) => (
                  <tr key={item.id}>
                    <td className="whitespace-nowrap px-4 py-3 text-sm font-mono text-gray-900">
                      {item.modelPattern}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-700">
                      {item.displayName}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {formatTokenCount(item.contextLength)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {formatTokenCount(item.maxCompletionTokens)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm">
                      <span className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                        item.provider === 'anthropic' ? 'bg-orange-100 text-orange-800' :
                        item.provider === 'openai' ? 'bg-green-100 text-green-800' :
                        item.provider === 'google' ? 'bg-blue-100 text-blue-800' :
                        item.provider === 'deepseek' ? 'bg-purple-100 text-purple-800' :
                        item.provider === 'alibaba' ? 'bg-yellow-100 text-yellow-800' :
                        'bg-gray-100 text-gray-800'
                      }`}>
                        {item.provider}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {formatDate(item.updatedAt)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-right text-sm space-x-2">
                      <button
                        onClick={() => handleEdit(item)}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        编辑
                      </button>
                      <button
                        onClick={() => handleDelete(item.id, item.modelPattern)}
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
      </div>

      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">模型元数据说明</h3>
        <div className="space-y-2 text-sm text-gray-600">
          <p>• <strong>模型模式</strong>：使用前缀匹配，例如 claude-sonnet 可匹配所有 claude-sonnet-4-5-xxx 系列模型</p>
          <p>• <strong>上下文长度</strong>：模型支持的最大上下文窗口大小，用于 bootstrap 响应重写</p>
          <p>• <strong>最大输出</strong>：单次请求允许的最大输出 token 数</p>
          <p>• 如果没有配置匹配的模型，将使用代码内置的默认值 (200k)</p>
        </div>
      </div>
    </div>
  )
}
