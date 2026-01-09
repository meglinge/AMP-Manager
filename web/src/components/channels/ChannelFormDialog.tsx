import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Channel,
  ChannelRequest,
  ChannelType,
  ChannelEndpoint,
  ChannelModel,
} from '@/api/channels'
import ModelRulesEditor from './ModelRulesEditor'

const CHANNEL_TYPES: { value: ChannelType; label: string; defaultUrl: string; defaultEndpoint: ChannelEndpoint }[] = [
  { value: 'gemini', label: 'Gemini', defaultUrl: 'https://generativelanguage.googleapis.com', defaultEndpoint: 'generate_content' },
  { value: 'claude', label: 'Claude', defaultUrl: 'https://api.anthropic.com', defaultEndpoint: 'messages' },
  { value: 'openai', label: 'OpenAI', defaultUrl: 'https://api.openai.com', defaultEndpoint: 'chat_completions' },
]

const OPENAI_ENDPOINTS: { value: ChannelEndpoint; label: string }[] = [
  { value: 'chat_completions', label: '/v1/chat/completions' },
  { value: 'responses', label: '/v1/responses' },
]

interface ChannelFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editingChannel: Channel | null
  formData: ChannelRequest
  setFormData: React.Dispatch<React.SetStateAction<ChannelRequest>>
  onSubmit: () => void
  saving: boolean
}

export function ChannelFormDialog({
  open,
  onOpenChange,
  editingChannel,
  formData,
  setFormData,
  onSubmit,
  saving,
}: ChannelFormDialogProps) {
  const handleTypeChange = (type: ChannelType) => {
    const channelType = CHANNEL_TYPES.find(t => t.value === type)
    if (channelType) {
      setFormData(prev => ({
        ...prev,
        type,
        baseUrl: prev.baseUrl || channelType.defaultUrl,
        endpoint: channelType.defaultEndpoint,
      }))
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
      models: (prev.models || []).filter((_, i) => i !== index),
    }))
  }

  const handleModelChange = (index: number, field: keyof ChannelModel, value: string) => {
    setFormData(prev => {
      const newModels = [...(prev.models || [])]
      newModels[index] = { ...newModels[index], [field]: value }
      return { ...prev, models: newModels }
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{editingChannel ? '编辑渠道' : '添加渠道'}</DialogTitle>
          <DialogDescription>
            配置 API 代理渠道的基本信息和模型规则
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-4 py-4">
          {/* 类型选择 */}
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

          {/* OpenAI Endpoint 选择 */}
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

          {/* 非 OpenAI 类型时的占位 */}
          {formData.type !== 'openai' && <div />}

          {/* 名称 */}
          <div className="space-y-2">
            <Label>名称 *</Label>
            <Input
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="渠道名称"
            />
          </div>

          {/* Base URL */}
          <div className="space-y-2">
            <Label>Base URL *</Label>
            <Input
              value={formData.baseUrl}
              onChange={(e) => setFormData(prev => ({ ...prev, baseUrl: e.target.value }))}
              placeholder="https://api.example.com"
            />
          </div>

          {/* API Key */}
          <div className="space-y-2 col-span-2">
            <Label>API Key {editingChannel ? '(留空保持不变)' : '*'}</Label>
            <Input
              type="password"
              value={formData.apiKey || ''}
              onChange={(e) => setFormData(prev => ({ ...prev, apiKey: e.target.value }))}
              placeholder={editingChannel ? '留空保持原有密钥' : '输入 API Key'}
            />
          </div>

          {/* 权重 */}
          <div className="space-y-2">
            <Label>权重</Label>
            <Input
              type="number"
              min={1}
              value={formData.weight}
              onChange={(e) => setFormData(prev => ({ ...prev, weight: parseInt(e.target.value) || 1 }))}
            />
          </div>

          {/* 优先级 */}
          <div className="space-y-2">
            <Label>优先级</Label>
            <Input
              type="number"
              min={0}
              value={formData.priority}
              onChange={(e) => setFormData(prev => ({ ...prev, priority: parseInt(e.target.value) || 0 }))}
            />
          </div>

          {/* 启用开关 */}
          <div className="col-span-2 flex items-center justify-between rounded-lg border p-4">
            <div className="space-y-0.5">
              <Label>启用渠道</Label>
              <p className="text-sm text-muted-foreground">
                启用后此渠道将参与负载均衡
              </p>
            </div>
            <Switch
              checked={formData.enabled}
              onCheckedChange={(checked) => setFormData(prev => ({ ...prev, enabled: checked }))}
            />
          </div>

          {/* 模型规则编辑器 */}
          <ModelRulesEditor
            models={formData.models}
            onAdd={handleAddModel}
            onRemove={handleRemoveModel}
            onChange={handleModelChange}
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={onSubmit} disabled={saving}>
            {saving ? '保存中...' : '保存'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
