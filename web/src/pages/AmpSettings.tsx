import { useState, useEffect, FormEvent } from 'react'
import {
  getAmpSettings,
  updateAmpSettings,
  testAmpConnection,
  AmpSettings as AmpSettingsType,
  ModelMapping,
} from '../api/amp'
import ModelMappingEditor from '../components/ModelMappingEditor'

export default function AmpSettings() {
  const [settings, setSettings] = useState<AmpSettingsType | null>(null)
  const [enabled, setEnabled] = useState(true)
  const [upstreamUrl, setUpstreamUrl] = useState('')
  const [upstreamApiKey, setUpstreamApiKey] = useState('')
  const [forceModelMappings, setForceModelMappings] = useState(false)
  const [modelMappings, setModelMappings] = useState<ModelMapping[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null)

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async () => {
    try {
      const data = await getAmpSettings()
      setSettings(data)
      setEnabled(data.enabled)
      setUpstreamUrl(data.upstreamUrl)
      setForceModelMappings(data.forceModelMappings)
      setModelMappings(data.modelMappings || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载设置失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    setSaving(true)

    try {
      const data = await updateAmpSettings({
        enabled,
        upstreamUrl,
        ...(upstreamApiKey ? { upstreamApiKey } : {}),
        forceModelMappings,
        modelMappings,
      })
      setSettings(data)
      setUpstreamApiKey('')
      setSuccess('设置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    setTestResult(null)
    setTesting(true)

    try {
      const result = await testAmpConnection()
      setTestResult(result)
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : '测试失败',
      })
    } finally {
      setTesting(false)
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
    <div className="mx-auto max-w-2xl">
      <div className="rounded-lg bg-white p-6 shadow-md">
        <h2 className="mb-6 text-xl font-bold text-gray-800">Amp 设置</h2>

        {error && (
          <div className="mb-4 rounded bg-red-100 p-3 text-red-700">{error}</div>
        )}
        {success && (
          <div className="mb-4 rounded bg-green-100 p-3 text-green-700">{success}</div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="enabled"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="enabled" className="text-sm font-medium text-gray-700">
              启用代理
            </label>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">
              Upstream URL
            </label>
            <input
              type="url"
              value={upstreamUrl}
              onChange={(e) => setUpstreamUrl(e.target.value)}
              placeholder="https://ampcode.com"
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">
              Upstream API Key
            </label>
            <div className="mt-1 flex items-center gap-3">
              <input
                type="password"
                value={upstreamApiKey}
                onChange={(e) => setUpstreamApiKey(e.target.value)}
                placeholder={settings?.apiKeySet ? '••••••••（已设置，留空保持不变）' : '请输入 API Key'}
                className="flex-1 rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <span
                className={`rounded-full px-3 py-1 text-xs ${
                  settings?.apiKeySet
                    ? 'bg-green-100 text-green-700'
                    : 'bg-yellow-100 text-yellow-700'
                }`}
              >
                {settings?.apiKeySet ? '已设置' : '未设置'}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="forceModelMappings"
              checked={forceModelMappings}
              onChange={(e) => setForceModelMappings(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="forceModelMappings" className="text-sm font-medium text-gray-700">
              Force Model Mappings
            </label>
          </div>

          <ModelMappingEditor mappings={modelMappings} onChange={setModelMappings} />

          <div className="flex items-center gap-4 border-t pt-6">
            <button
              type="button"
              onClick={handleTest}
              disabled={testing}
              className="rounded-md border border-gray-300 bg-white px-4 py-2 text-gray-700 hover:bg-gray-50 disabled:bg-gray-100"
            >
              {testing ? '测试中...' : '测试连接'}
            </button>
            <button
              type="submit"
              disabled={saving}
              className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:bg-blue-300"
            >
              {saving ? '保存中...' : '保存设置'}
            </button>
          </div>

          {testResult && (
            <div
              className={`rounded p-3 ${
                testResult.success
                  ? 'bg-green-100 text-green-700'
                  : 'bg-red-100 text-red-700'
              }`}
            >
              {testResult.message}
            </div>
          )}
        </form>
      </div>
    </div>
  )
}
