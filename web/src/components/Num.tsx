import { formatCompact, formatExact } from '@/lib/formatters'
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip'

interface NumProps {
  value: number | undefined
  className?: string
}

export function Num({ value, className }: NumProps) {
  const compact = formatCompact(value)
  const exact = formatExact(value)

  if (value === undefined || value === null || compact === exact) {
    return <span className={className}>{compact}</span>
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`cursor-default ${className || ''}`}>{compact}</span>
      </TooltipTrigger>
      <TooltipContent>
        <span className="font-mono">{exact}</span>
      </TooltipContent>
    </Tooltip>
  )
}
