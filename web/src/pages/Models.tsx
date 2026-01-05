import { useState, useEffect } from 'react'
import { listAvailableModels, fetchAllModels, AvailableModel, FetchModelsResult } from '../api/models'

interface Props {
  isAdmin: boolean
}

export default function Models({ isAdmin }: Props) {
  const [models, setModels] = useState<AvailableModel[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [fetching, setFetching] = useState(false)
  const [fetchResult, setFetchResult] = useState<FetchModelsResult | null>(null)
  const [filter, setFilter] = useState<'all' | 'openai' | 'claude' | 'gemini'>('all')

  useEffect(() => {
    loadModels()
  }, [])

  const loadModels = async () => {
    try {
      setLoading(true)
      const data = await listAvailableModels()
      setModels(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleFetchAll = async () => {
    try {
      setFetching(true)
      setFetchResult(null)
      setError('')
      const result = await fetchAllModels()
      setFetchResult(result)
      loadModels()
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取失败')
    } finally {
      setFetching(false)
    }
  }

  const filteredModels = filter === 'all' 
    ? models 
    : models.filter(m => m.channelType === filter)

  const groupedModels = filteredModels.reduce((acc, model) => {
    const key = model.channelType
    if (!acc[key]) acc[key] = []
    acc[key].push(model)
    return acc
  }, {} as Record<string, AvailableModel[]>)

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'gemini': return 'bg-blue-100 text-blue-800'
      case 'claude': return 'bg-orange-100 text-orange-800'
      case 'openai': return 'bg-green-100 text-green-800'
      default: return 'bg-gray-100 text-gray-800'
    }
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

      {fetchResult && (
        <div className="rounded bg-green-100 p-3 text-green-800">
          <p className="font-medium">{fetchResult.message}</p>
          <div className="mt-2 text-sm">
            {Object.entries(fetchResult.results).map(([name, count]) => (
              <div key={name}>
                {name}: {count === -1 ? '获取失败' : `${count} 个模型`}
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="rounded-lg bg-white p-6 shadow-md">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-xl font-bold text-gray-800">可用模型</h2>
          <div className="flex items-center gap-4">
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value as typeof filter)}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm"
            >
              <option value="all">全部类型</option>
              <option value="openai">OpenAI</option>
              <option value="claude">Claude</option>
              <option value="gemini">Gemini</option>
            </select>
            {isAdmin && (
              <button
                onClick={handleFetchAll}
                disabled={fetching}
                className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:bg-gray-300"
              >
                {fetching ? '获取中...' : '刷新所有模型'}
              </button>
            )}
          </div>
        </div>

        {models.length === 0 ? (
          <div className="rounded-md border border-dashed border-gray-300 p-8 text-center text-gray-500">
            {isAdmin ? (
              <div>
                <p>暂无模型数据</p>
                <p className="mt-2 text-sm">请先在渠道管理中添加渠道，然后点击"刷新所有模型"获取模型列表</p>
              </div>
            ) : (
              <p>暂无可用模型，请联系管理员配置</p>
            )}
          </div>
        ) : (
          <div className="space-y-6">
            {Object.entries(groupedModels).map(([type, typeModels]) => (
              <div key={type}>
                <h3 className="mb-3 flex items-center gap-2 text-lg font-medium text-gray-800">
                  <span className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${getTypeColor(type)}`}>
                    {type.toUpperCase()}
                  </span>
                  <span className="text-sm text-gray-500">({typeModels.length} 个模型)</span>
                </h3>
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {typeModels.map((model) => (
                    <div
                      key={`${model.channelName}-${model.modelId}`}
                      className="rounded-lg border border-gray-200 p-4 hover:bg-gray-50"
                    >
                      <div className="font-mono text-sm font-medium text-gray-900">
                        {model.modelId}
                      </div>
                      {model.displayName !== model.modelId && (
                        <div className="mt-1 text-xs text-gray-500">
                          {model.displayName}
                        </div>
                      )}
                      <div className="mt-2 text-xs text-gray-400">
                        渠道: {model.channelName}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="mt-6 rounded-lg bg-gray-50 p-4 text-sm text-gray-600">
          <p className="font-medium">说明</p>
          <ul className="mt-2 list-inside list-disc space-y-1">
            <li>模型列表来自各渠道的 API，已按类型过滤</li>
            <li>OpenAI 渠道只显示 gpt/o1/o3/o4 开头的模型</li>
            <li>Claude 渠道只显示 claude 开头的模型</li>
            <li>Gemini 渠道只显示 gemini 开头的模型</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
