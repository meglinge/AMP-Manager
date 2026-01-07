import { Badge } from '@/components/ui/badge'

interface StatusBadgeProps {
  status: number
}

export function StatusBadge({ status }: StatusBadgeProps) {
  if (status >= 200 && status < 300) {
    return <Badge variant="default" className="bg-green-500">{status}</Badge>
  } else if (status >= 400 && status < 500) {
    return <Badge variant="destructive">{status}</Badge>
  } else if (status >= 500) {
    return <Badge variant="destructive" className="bg-red-700">{status}</Badge>
  }
  return <Badge variant="secondary">{status}</Badge>
}
