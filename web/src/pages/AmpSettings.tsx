import { useState, useEffect, FormEvent } from 'react'
import {
  getAmpSettings,
  updateAmpSettings,
  testAmpConnection,
  AmpSettings as AmpSettingsType,
  ModelMapping,
} from '../api/amp'
import ModelMappingEditor from '../components/ModelMappingEditor'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'

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
        <div className="text-muted-foreground">加载中...</div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle>Amp 设置</CardTitle>
          <CardDescription>配置 Amp 代理服务的上游连接和模型映射</CardDescription>
        </CardHeader>

        <CardContent>
          {error && (
            <Alert variant="destructive" className="mb-4">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {success && (
            <Alert className="mb-4 border-green-200 bg-green-50 text-green-800">
              <AlertDescription>{success}</AlertDescription>
            </Alert>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="enabled">启用代理</Label>
                <p className="text-sm text-muted-foreground">开启后将启用 Amp 代理服务</p>
              </div>
              <Switch
                id="enabled"
                checked={enabled}
                onCheckedChange={setEnabled}
              />
            </div>

            <Separator />

            <div className="space-y-2">
              <Label htmlFor="upstreamUrl">Upstream URL</Label>
              <Input
                id="upstreamUrl"
                type="url"
                value={upstreamUrl}
                onChange={(e) => setUpstreamUrl(e.target.value)}
                placeholder="https://ampcode.com"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="upstreamApiKey">Upstream API Key</Label>
              <div className="flex items-center gap-3">
                <Input
                  id="upstreamApiKey"
                  type="password"
                  value={upstreamApiKey}
                  onChange={(e) => setUpstreamApiKey(e.target.value)}
                  placeholder={settings?.apiKeySet ? '••••••••（已设置，留空保持不变）' : '请输入 API Key'}
                  className="flex-1"
                />
                <Badge variant={settings?.apiKeySet ? 'default' : 'secondary'}>
                  {settings?.apiKeySet ? '已设置' : '未设置'}
                </Badge>
              </div>
            </div>

            <Separator />

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="forceModelMappings">Force Model Mappings</Label>
                <p className="text-sm text-muted-foreground">强制使用模型映射规则</p>
              </div>
              <Switch
                id="forceModelMappings"
                checked={forceModelMappings}
                onCheckedChange={setForceModelMappings}
              />
            </div>

            <ModelMappingEditor mappings={modelMappings} onChange={setModelMappings} />

            {testResult && (
              <Alert
                variant={testResult.success ? 'default' : 'destructive'}
                className={testResult.success ? 'border-green-200 bg-green-50 text-green-800' : ''}
              >
                <AlertDescription>{testResult.message}</AlertDescription>
              </Alert>
            )}
          </form>
        </CardContent>

        <CardFooter className="flex justify-between border-t pt-6">
          <Button
            type="button"
            variant="outline"
            onClick={handleTest}
            disabled={testing}
          >
            {testing ? '测试中...' : '测试连接'}
          </Button>
          <Button
            type="submit"
            disabled={saving}
            onClick={handleSubmit}
          >
            {saving ? '保存中...' : '保存设置'}
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
