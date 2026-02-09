import { useState, useEffect } from 'react'
import { UserInfo } from '@/api/users'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { DateTimePicker } from '@/components/ui/datetime-picker'
import { motion } from '@/lib/motion'

export interface FilterValues {
  userId: string
  model: string
  from: string
  to: string
}

interface LogFilterBarProps {
  isAdmin: boolean
  users: UserInfo[]
  models: string[]
  values: FilterValues
  onChange: (values: FilterValues) => void
}

type PresetKey = 'custom' | '1h' | '6h' | '24h' | '7d' | '30d'

const presets: { key: PresetKey; label: string }[] = [
  { key: '1h', label: '最近 1 小时' },
  { key: '6h', label: '最近 6 小时' },
  { key: '24h', label: '最近 24 小时' },
  { key: '7d', label: '最近 7 天' },
  { key: '30d', label: '最近 30 天' },
]

function toLocalDatetimeString(date: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function getPresetRange(key: PresetKey): { from: string; to: string } {
  const now = new Date()
  const to = toLocalDatetimeString(now)
  let fromDate: Date

  switch (key) {
    case '1h':
      fromDate = new Date(now.getTime() - 60 * 60 * 1000)
      break
    case '6h':
      fromDate = new Date(now.getTime() - 6 * 60 * 60 * 1000)
      break
    case '24h':
      fromDate = new Date(now.getTime() - 24 * 60 * 60 * 1000)
      break
    case '7d':
      fromDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
      break
    case '30d':
      fromDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
      break
    default:
      return { from: '', to: '' }
  }

  return { from: toLocalDatetimeString(fromDate), to }
}

function localToISO(localStr: string): string {
  if (!localStr) return ''
  return new Date(localStr).toISOString()
}

export function LogFilterBar({ isAdmin, users, models, values, onChange }: LogFilterBarProps) {
  const [activePreset, setActivePreset] = useState<PresetKey>('custom')

  useEffect(() => {
    if (values.from || values.to) {
      setActivePreset('custom')
    }
  }, [])

  const update = (partial: Partial<FilterValues>) => {
    onChange({ ...values, ...partial })
  }

  const handlePreset = (key: PresetKey) => {
    setActivePreset(key)
    if (key === 'custom') {
      update({ from: '', to: '' })
    } else {
      const range = getPresetRange(key)
      update(range)
    }
  }

  const handleFromChange = (from: string) => {
    setActivePreset('custom')
    update({ from })
  }

  const handleToChange = (to: string) => {
    setActivePreset('custom')
    update({ to })
  }

  const handleClearAll = () => {
    setActivePreset('custom')
    onChange({ userId: '', model: '', from: '', to: '' })
  }

  const hasAnyFilter = values.userId || values.model || values.from || values.to

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: 'spring', bounce: 0.2, duration: 0.5, delay: 0.05 }}
    >
      <Card>
        <CardContent className="pt-4 pb-4 space-y-3">
          <div className="flex items-center gap-3 flex-wrap">
            {isAdmin && (
              <div className="flex items-center gap-2">
                <Label className="text-sm text-muted-foreground whitespace-nowrap">用户</Label>
                <Select
                  value={values.userId || 'all'}
                  onValueChange={(v) => update({ userId: v === 'all' ? '' : v })}
                >
                  <SelectTrigger className="w-36">
                    <SelectValue placeholder="所有用户" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">所有用户</SelectItem>
                    {users.map(u => (
                      <SelectItem key={u.id} value={u.id}>{u.username}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div className="flex items-center gap-2">
              <Label className="text-sm text-muted-foreground whitespace-nowrap">模型</Label>
              {isAdmin && models.length > 0 ? (
                <Select
                  value={values.model || 'all'}
                  onValueChange={(v) => update({ model: v === 'all' ? '' : v })}
                >
                  <SelectTrigger className="w-48">
                    <SelectValue placeholder="所有模型" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">所有模型</SelectItem>
                    {models.map(m => (
                      <SelectItem key={m} value={m}>{m}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  type="text"
                  placeholder="按模型筛选..."
                  value={values.model}
                  onChange={(e) => update({ model: e.target.value })}
                  className="w-48 h-9"
                />
              )}
            </div>

            <div className="h-6 w-px bg-border mx-1" />

            <div className="flex items-center gap-2">
              <Label className="text-sm text-muted-foreground whitespace-nowrap">从</Label>
              <DateTimePicker
                value={values.from}
                onChange={handleFromChange}
                placeholder="开始时间"
              />
            </div>

            <div className="flex items-center gap-2">
              <Label className="text-sm text-muted-foreground whitespace-nowrap">到</Label>
              <DateTimePicker
                value={values.to}
                onChange={handleToChange}
                placeholder="结束时间"
              />
            </div>

            {hasAnyFilter && (
              <Button variant="ghost" size="sm" onClick={handleClearAll} className="text-muted-foreground">
                清除筛选
              </Button>
            )}
          </div>

          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-xs text-muted-foreground mr-1">快捷时间:</span>
            {presets.map(p => (
              <Button
                key={p.key}
                variant={activePreset === p.key ? 'default' : 'outline'}
                size="sm"
                className="h-6 text-xs px-2"
                onClick={() => handlePreset(p.key)}
              >
                {p.label}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>
    </motion.div>
  )
}

export { localToISO }
