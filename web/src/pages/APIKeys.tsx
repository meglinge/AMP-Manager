import { useState, useEffect } from 'react'
import {
  getAPIKeys,
  createAPIKey,
  revokeAPIKey,
  APIKey,
  CreateAPIKeyResponse,
} from '../api/amp'

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
        <div className="text-gray-500">加载中...</div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {/* 新创建的 API Key 显示 */}
      {newKey && (
        <div className="rounded-lg border-2 border-green-500 bg-green-50 p-6">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-lg font-bold text-green-800">API Key 创建成功</h3>
            <button
              onClick={() => setNewKey(null)}
              className="text-green-600 hover:text-green-800"
            >
              关闭
            </button>
          </div>
          <p className="mb-4 text-sm text-yellow-700 bg-yellow-100 rounded p-2">
            ⚠️ 请立即复制以下信息，API Key 明文只显示一次！
          </p>

          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700">API Key</label>
              <div className="mt-1 flex items-center gap-2">
                <code className="flex-1 rounded bg-white p-2 text-sm font-mono break-all">
                  {newKey.apiKey}
                </code>
                <button
                  onClick={() => copyToClipboard(newKey.apiKey, 'apiKey')}
                  className="rounded bg-blue-600 px-3 py-2 text-sm text-white hover:bg-blue-700"
                >
                  {copied === 'apiKey' ? '已复制' : '复制'}
                </button>
              </div>
            </div>

            {newKey && (
              <div>
                <label className="block text-sm font-medium text-gray-700">使用方法 (Linux/macOS)</label>
                <div className="mt-1 rounded bg-gray-800 p-3 text-sm font-mono text-green-400">
                  <div>export AMP_URL="{window.location.origin}"</div>
                  <div>export AMP_API_KEY="{newKey.apiKey}"</div>
                </div>
                <button
                  onClick={() => copyToClipboard(`export AMP_URL="${window.location.origin}"\nexport AMP_API_KEY="${newKey.apiKey}"`, 'env')}
                  className="mt-2 rounded bg-gray-600 px-3 py-1 text-sm text-white hover:bg-gray-700"
                >
                  {copied === 'env' ? '已复制' : '复制环境变量'}
                </button>

                <label className="mt-4 block text-sm font-medium text-gray-700">Windows PowerShell (永久)</label>
                <div className="mt-1 rounded bg-gray-800 p-3 text-sm font-mono text-green-400">
                  <div>[Environment]::SetEnvironmentVariable("AMP_URL", "{window.location.origin}", "User")</div>
                  <div>[Environment]::SetEnvironmentVariable("AMP_API_KEY", "{newKey.apiKey}", "User")</div>
                </div>
                <button
                  onClick={() => copyToClipboard(`[Environment]::SetEnvironmentVariable("AMP_URL", "${window.location.origin}", "User")\n[Environment]::SetEnvironmentVariable("AMP_API_KEY", "${newKey.apiKey}", "User")`, 'ps')}
                  className="mt-2 rounded bg-gray-600 px-3 py-1 text-sm text-white hover:bg-gray-700"
                >
                  {copied === 'ps' ? '已复制' : '复制 PowerShell 命令'}
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* API Key 管理卡片 */}
      <div className="rounded-lg bg-white p-6 shadow-md">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-xl font-bold text-gray-800">API Key 管理</h2>
          <button
            onClick={() => setShowCreate(true)}
            className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
          >
            创建 API Key
          </button>
        </div>

        {error && (
          <div className="mb-4 rounded bg-red-100 p-3 text-red-700">{error}</div>
        )}

        {/* 创建弹窗 */}
        {showCreate && (
          <div className="mb-6 rounded-lg border border-blue-200 bg-blue-50 p-4">
            <h3 className="mb-3 font-medium text-gray-800">创建新 API Key</h3>
            <div className="flex items-center gap-3">
              <input
                type="text"
                value={createName}
                onChange={(e) => setCreateName(e.target.value)}
                placeholder="输入 API Key 名称（如：工作电脑）"
                className="flex-1 rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
              />
              <button
                onClick={handleCreate}
                disabled={creating || !createName.trim()}
                className="rounded-md bg-green-600 px-4 py-2 text-white hover:bg-green-700 disabled:bg-gray-300"
              >
                {creating ? '创建中...' : '创建'}
              </button>
              <button
                onClick={() => {
                  setShowCreate(false)
                  setCreateName('')
                }}
                className="rounded-md border border-gray-300 px-4 py-2 text-gray-700 hover:bg-gray-100"
              >
                取消
              </button>
            </div>
          </div>
        )}

        {/* API Key 列表 */}
        {keys.length === 0 ? (
          <div className="rounded-md border border-dashed border-gray-300 p-8 text-center text-gray-500">
            暂无 API Key，点击上方按钮创建
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    名称
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Prefix
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    状态
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    最后使用
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    创建时间
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">
                    操作
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {keys.map((key) => (
                  <tr key={key.id}>
                    <td className="whitespace-nowrap px-4 py-3 text-sm font-medium text-gray-900">
                      {key.name}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500 font-mono">
                      {key.prefix}...
                    </td>
                    <td className="whitespace-nowrap px-4 py-3">
                      <span
                        className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                          key.isActive
                            ? 'bg-green-100 text-green-800'
                            : 'bg-red-100 text-red-800'
                        }`}
                      >
                        {key.isActive ? '活跃' : '已撤销'}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {formatDate(key.lastUsedAt)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                      {formatDate(key.createdAt)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-right text-sm">
                      {key.isActive && (
                        <button
                          onClick={() => handleRevoke(key.id, key.name)}
                          className="text-red-600 hover:text-red-800"
                        >
                          撤销
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* 使用说明 */}
      <div className="rounded-lg bg-white p-6 shadow-md">
        <h3 className="mb-4 text-lg font-bold text-gray-800">使用说明</h3>
        <div className="space-y-3 text-sm text-gray-600">
          <p>1. 创建一个 API Key 用于 Amp CLI 认证</p>
          <p>2. 在终端配置环境变量：</p>
          <div className="space-y-4">
            <div>
              <p className="mb-1 font-medium text-gray-700">Linux/macOS:</p>
              <div className="rounded bg-gray-800 p-3 font-mono text-green-400">
                <div>export AMP_URL="{window.location.origin}"</div>
                <div>export AMP_API_KEY="your-api-key-here"</div>
              </div>
            </div>
            <div>
              <p className="mb-1 font-medium text-gray-700">Windows PowerShell (永久):</p>
              <div className="rounded bg-gray-800 p-3 font-mono text-green-400">
                <div>[Environment]::SetEnvironmentVariable("AMP_URL", "{window.location.origin}", "User")</div>
                <div>[Environment]::SetEnvironmentVariable("AMP_API_KEY", "your-api-key-here", "User")</div>
              </div>
            </div>
          </div>
          <p>3. Amp CLI 会自动使用这些环境变量连接到反代服务</p>
        </div>
      </div>
    </div>
  )
}
