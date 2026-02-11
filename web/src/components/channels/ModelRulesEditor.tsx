import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { ChannelModel } from '@/api/channels'
import { getChannelModels, ChannelModel2 } from '@/api/models'

interface Props {
  models: ChannelModel[] | undefined
  channelId?: string
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, field: keyof ChannelModel, value: string) => void
  onSetModels: (models: ChannelModel[]) => void
}

export default function ModelRulesEditor({ models, channelId, onAdd, onRemove, onChange, onSetModels }: Props) {
  const [fetchedModels, setFetchedModels] = useState<ChannelModel2[]>([])
  const [showSelector, setShowSelector] = useState(false)
  const [loadingModels, setLoadingModels] = useState(false)
  const [fetchError, setFetchError] = useState('')

  const handleLoadModels = async () => {
    if (!channelId) return
    setLoadingModels(true)
    setFetchError('')
    try {
      const data = await getChannelModels(channelId)
      setFetchedModels(data)
      setShowSelector(true)
    } catch (err) {
      setFetchError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoadingModels(false)
    }
  }

  const isModelSelected = (modelId: string) => {
    return (models || []).some(m => m.name === modelId)
  }

  const handleToggleModel = (modelId: string, checked: boolean) => {
    const currentModels = models || []
    if (checked) {
      onSetModels([...currentModels, { name: modelId }])
    } else {
      onSetModels(currentModels.filter(m => m.name !== modelId))
    }
  }

  const handleSelectAll = () => {
    const allModels = fetchedModels.map(m => ({ name: m.modelId }))
    onSetModels(allModels)
  }

  const handleDeselectAll = () => {
    onSetModels([])
  }

  return (
    <div className="col-span-2 space-y-2">
      <div className="flex items-center justify-between">
        <Label>模型规则 (留空则按类型自动匹配)</Label>
        <div className="flex items-center gap-2">
          {channelId && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleLoadModels}
              disabled={loadingModels}
            >
              {loadingModels ? '加载中...' : '从已获取的模型中选择'}
            </Button>
          )}
          <Button type="button" variant="link" size="sm" onClick={onAdd}>
            + 添加规则
          </Button>
        </div>
      </div>

      {fetchError && (
        <p className="text-sm text-destructive">{fetchError}</p>
      )}

      {showSelector && fetchedModels.length > 0 && (
        <div className="rounded-md border p-3 space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">已获取的模型 ({fetchedModels.length})</span>
            <div className="flex items-center gap-2">
              <Button type="button" variant="ghost" size="sm" onClick={handleSelectAll}>
                全选
              </Button>
              <Button type="button" variant="ghost" size="sm" onClick={handleDeselectAll}>
                全不选
              </Button>
              <Button type="button" variant="ghost" size="sm" onClick={() => setShowSelector(false)}>
                收起
              </Button>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-1 max-h-60 overflow-y-auto">
            {fetchedModels.map((m) => (
              <label key={m.id} className="flex items-center gap-2 rounded px-2 py-1.5 hover:bg-muted cursor-pointer text-sm">
                <Checkbox
                  checked={isModelSelected(m.modelId)}
                  onCheckedChange={(checked) => handleToggleModel(m.modelId, checked === true)}
                />
                <span className="truncate" title={m.displayName !== m.modelId ? `${m.modelId} (${m.displayName})` : m.modelId}>
                  {m.modelId}
                </span>
              </label>
            ))}
          </div>
        </div>
      )}

      {showSelector && fetchedModels.length === 0 && !loadingModels && (
        <div className="rounded-md border border-dashed p-3">
          <p className="text-sm text-muted-foreground text-center">
            暂无已获取的模型，请先在渠道列表中点击"获取模型"
          </p>
        </div>
      )}

      {models && models.length > 0 && (
        <div className="space-y-2">
          {models.map((model, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                value={model.name}
                onChange={(e) => onChange(index, 'name', e.target.value)}
                placeholder="模型名 (支持 * 通配符)"
                className="flex-1"
              />
              <Input
                value={model.alias || ''}
                onChange={(e) => onChange(index, 'alias', e.target.value)}
                placeholder="别名 (可选)"
                className="flex-1"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="text-destructive hover:text-destructive"
                onClick={() => onRemove(index)}
              >
                删除
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
