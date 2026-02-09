import { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import { Calendar, ChevronLeft, ChevronRight, Clock, X } from 'lucide-react'

interface DateTimePickerProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
}

const MONTHS = ['一月', '二月', '三月', '四月', '五月', '六月', '七月', '八月', '九月', '十月', '十一月', '十二月']
const WEEKDAYS = ['日', '一', '二', '三', '四', '五', '六']

function pad(n: number) {
  return n.toString().padStart(2, '0')
}

function formatDisplay(value: string): string {
  if (!value) return ''
  const d = new Date(value)
  if (isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

function getFirstDayOfMonth(year: number, month: number): number {
  return new Date(year, month, 1).getDay()
}

export function DateTimePicker({ value, onChange, placeholder = '选择时间', className }: DateTimePickerProps) {
  const [open, setOpen] = useState(false)
  const [viewYear, setViewYear] = useState(() => {
    if (value) return new Date(value).getFullYear()
    return new Date().getFullYear()
  })
  const [viewMonth, setViewMonth] = useState(() => {
    if (value) return new Date(value).getMonth()
    return new Date().getMonth()
  })

  const parsed = value ? new Date(value) : null
  const selectedDay = parsed ? parsed.getDate() : null
  const selectedMonth = parsed ? parsed.getMonth() : null
  const selectedYear = parsed ? parsed.getFullYear() : null

  const [hour, setHour] = useState(parsed ? pad(parsed.getHours()) : '00')
  const [minute, setMinute] = useState(parsed ? pad(parsed.getMinutes()) : '00')

  const hourRef = useRef<HTMLInputElement>(null)
  const minuteRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (value) {
      const d = new Date(value)
      if (!isNaN(d.getTime())) {
        setHour(pad(d.getHours()))
        setMinute(pad(d.getMinutes()))
        setViewYear(d.getFullYear())
        setViewMonth(d.getMonth())
      }
    }
  }, [value])

  const daysInMonth = getDaysInMonth(viewYear, viewMonth)
  const firstDay = getFirstDayOfMonth(viewYear, viewMonth)
  const today = new Date()

  const buildValue = (year: number, month: number, day: number, h: string, m: string) => {
    return `${year}-${pad(month + 1)}-${pad(day)}T${h}:${m}`
  }

  const handleDayClick = (day: number) => {
    const newValue = buildValue(viewYear, viewMonth, day, hour, minute)
    onChange(newValue)
  }

  const handleTimeChange = (newHour: string, newMinute: string) => {
    if (parsed && selectedYear !== null && selectedMonth !== null && selectedDay !== null) {
      onChange(buildValue(selectedYear, selectedMonth, selectedDay, newHour, newMinute))
    }
  }

  const handleHourChange = (val: string) => {
    const num = val.replace(/\D/g, '')
    if (num === '') {
      setHour('')
      return
    }
    const n = Math.min(23, Math.max(0, parseInt(num)))
    const padded = pad(n)
    setHour(padded)
    handleTimeChange(padded, minute)
  }

  const handleMinuteChange = (val: string) => {
    const num = val.replace(/\D/g, '')
    if (num === '') {
      setMinute('')
      return
    }
    const n = Math.min(59, Math.max(0, parseInt(num)))
    const padded = pad(n)
    setMinute(padded)
    handleTimeChange(hour, padded)
  }

  const prevMonth = () => {
    if (viewMonth === 0) {
      setViewMonth(11)
      setViewYear(y => y - 1)
    } else {
      setViewMonth(m => m - 1)
    }
  }

  const nextMonth = () => {
    if (viewMonth === 11) {
      setViewMonth(0)
      setViewYear(y => y + 1)
    } else {
      setViewMonth(m => m + 1)
    }
  }

  const setNow = () => {
    const now = new Date()
    const h = pad(now.getHours())
    const m = pad(now.getMinutes())
    setHour(h)
    setMinute(m)
    setViewYear(now.getFullYear())
    setViewMonth(now.getMonth())
    onChange(buildValue(now.getFullYear(), now.getMonth(), now.getDate(), h, m))
  }

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation()
    onChange('')
    setHour('00')
    setMinute('00')
  }

  const days: (number | null)[] = []
  for (let i = 0; i < firstDay; i++) days.push(null)
  for (let i = 1; i <= daysInMonth; i++) days.push(i)

  const displayValue = formatDisplay(value)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            'w-[200px] h-9 justify-start text-left font-normal gap-2',
            !value && 'text-muted-foreground',
            className
          )}
        >
          <Calendar className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          <span className="truncate text-xs">
            {displayValue || placeholder}
          </span>
          {value && (
            <X
              className="h-3 w-3 ml-auto text-muted-foreground hover:text-foreground shrink-0 cursor-pointer"
              onClick={handleClear}
            />
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="p-3">
          {/* Month/Year Header */}
          <div className="flex items-center justify-between mb-2">
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={prevMonth}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm font-medium">
              {MONTHS[viewMonth]} {viewYear}
            </span>
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={nextMonth}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>

          {/* Weekday Headers */}
          <div className="grid grid-cols-7 gap-0 mb-1">
            {WEEKDAYS.map(d => (
              <div key={d} className="text-center text-xs text-muted-foreground py-1 font-medium">
                {d}
              </div>
            ))}
          </div>

          {/* Days Grid */}
          <div className="grid grid-cols-7 gap-0">
            {days.map((day, idx) => {
              if (day === null) {
                return <div key={`empty-${idx}`} className="h-8 w-8" />
              }
              const isSelected = day === selectedDay && viewMonth === selectedMonth && viewYear === selectedYear
              const isToday = day === today.getDate() && viewMonth === today.getMonth() && viewYear === today.getFullYear()
              return (
                <button
                  key={day}
                  onClick={() => handleDayClick(day)}
                  className={cn(
                    'h-8 w-8 rounded-md text-xs font-medium transition-colors',
                    'hover:bg-accent hover:text-accent-foreground',
                    'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring',
                    isSelected && 'bg-primary text-primary-foreground hover:bg-primary/90',
                    isToday && !isSelected && 'border border-primary/50 text-primary',
                  )}
                >
                  {day}
                </button>
              )
            })}
          </div>

          {/* Time Selector */}
          <div className="border-t mt-3 pt-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  ref={hourRef}
                  value={hour}
                  onChange={(e) => handleHourChange(e.target.value)}
                  className="w-11 h-7 text-center text-xs px-1"
                  maxLength={2}
                />
                <span className="text-sm text-muted-foreground font-medium">:</span>
                <Input
                  ref={minuteRef}
                  value={minute}
                  onChange={(e) => handleMinuteChange(e.target.value)}
                  className="w-11 h-7 text-center text-xs px-1"
                  maxLength={2}
                />
              </div>
              <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={setNow}>
                现在
              </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
