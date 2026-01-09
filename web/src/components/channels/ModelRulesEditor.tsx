import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ChannelModel } from '@/api/channels'

interface Props {
  models: ChannelModel[] | undefined
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, field: keyof ChannelModel, value: string) => void
}

export default function ModelRulesEditor({ models, onAdd, onRemove, onChange }: Props) {
  return (
    <div className="col-span-2 space-y-2">
      <div className="flex items-center justify-between">
        <Label>模型规则 (留空则按类型自动匹配)</Label>
        <Button type="button" variant="link" size="sm" onClick={onAdd}>
          + 添加规则
        </Button>
      </div>
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
