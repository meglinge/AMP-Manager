import { useState, useEffect, FormEvent } from 'react'
import { motion } from '@/lib/motion'
import {
    getAmpSettings,
    updateAmpSettings,
    testAmpConnection,
    AmpSettings as AmpSettingsType,
    ModelMapping,
    WebSearchMode,
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
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'

export default function AmpSettings() {
    const [settings, setSettings] = useState<AmpSettingsType | null>(null)
    const [enabled, setEnabled] = useState(true)
    const [upstreamUrl, setUpstreamUrl] = useState('')
    const [upstreamApiKey, setUpstreamApiKey] = useState('')
    const [forceModelMappings, setForceModelMappings] = useState(false)
    const [modelMappings, setModelMappings] = useState<ModelMapping[]>([])
    const [nativeMode, setNativeMode] = useState(false)
    const [webSearchMode, setWebSearchMode] = useState<WebSearchMode>('upstream')
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
            setNativeMode(data.nativeMode)
            setWebSearchMode(data.webSearchMode || 'upstream')
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
                nativeMode,
                upstreamUrl,
                ...(upstreamApiKey ? { upstreamApiKey } : {}),
                forceModelMappings,
                modelMappings,
                webSearchMode,
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
        <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ type: 'spring', bounce: 0.2, duration: 0.7 }} className="mx-auto max-w-2xl">
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
                        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.1, type: 'spring', bounce: 0.2, duration: 0.5 }}>
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
                        </motion.div>

                        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.2, type: 'spring', bounce: 0.2, duration: 0.5 }}>
                        <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                                <Label htmlFor="nativeMode">原生模式</Label>
                                <p className="text-sm text-muted-foreground">开启后所有请求直接转发到上游，不进行任何拦截、模型映射或渠道路由</p>
                            </div>
                            <Switch
                                id="nativeMode"
                                checked={nativeMode}
                                onCheckedChange={setNativeMode}
                            />
                        </div>

                        {nativeMode && (
                            <Alert className="border-amber-200 bg-amber-50 text-amber-800">
                                <AlertDescription>原生模式已开启，以下设置将不生效。所有请求将直接转发到上游服务器。</AlertDescription>
                            </Alert>
                        )}
                        </motion.div>

                        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.25, type: 'spring', bounce: 0.2, duration: 0.5 }}>
                        <Separator />
                        </motion.div>

                        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3, type: 'spring', bounce: 0.2, duration: 0.6 }}>
                        <div className={nativeMode ? 'space-y-6 opacity-50 pointer-events-none' : 'space-y-6'}>
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

                            <Separator />

                            <div className="space-y-3">
                                <div className="space-y-0.5">
                                    <Label>网页搜索模式</Label>
                                    <p className="text-sm text-muted-foreground">选择网页搜索和网页内容获取的处理方式</p>
                                </div>
                                <RadioGroup
                                    value={webSearchMode}
                                    onValueChange={(value) => setWebSearchMode(value as WebSearchMode)}
                                    className="space-y-2"
                                >
                                    <div className="flex items-center space-x-2">
                                        <RadioGroupItem value="upstream" id="upstream" />
                                        <Label htmlFor="upstream" className="font-normal cursor-pointer">
                                            <span className="font-medium">上游代理</span>
                                            <span className="text-muted-foreground ml-2">- 直接转发到上游，不做任何修改</span>
                                        </Label>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <RadioGroupItem value="builtin_free" id="builtin_free" />
                                        <Label htmlFor="builtin_free" className="font-normal cursor-pointer">
                                            <span className="font-medium">内置免费搜索</span>
                                            <span className="text-muted-foreground ml-2">- 使用 Amp 内置的免费搜索功能</span>
                                        </Label>
                                    </div>
                                    <div className="flex items-center space-x-2">
                                        <RadioGroupItem value="local_duckduckgo" id="local_duckduckgo" />
                                        <Label htmlFor="local_duckduckgo" className="font-normal cursor-pointer">
                                            <span className="font-medium">本地搜索</span>
                                            <span className="text-muted-foreground ml-2">- 使用本地 DuckDuckGo 搜索（完全免费）</span>
                                        </Label>
                                    </div>
                                </RadioGroup>
                            </div>
                        </div>
                        </motion.div>

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
        </motion.div>
    )
}
