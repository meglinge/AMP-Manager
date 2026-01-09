import { Alert, AlertDescription } from '@/components/ui/alert'
import { Channel, TestChannelResult } from '@/api/channels'

interface TestResultsDisplayProps {
  testResults: Record<string, TestChannelResult>
  channels: Channel[]
}

export function TestResultsDisplay({ testResults, channels }: TestResultsDisplayProps) {
  if (Object.keys(testResults).length === 0) {
    return null
  }

  return (
    <div className="mt-4 space-y-2">
      {Object.entries(testResults).map(([id, result]) => {
        const channel = channels.find(c => c.id === id)
        return (
          <Alert key={id} variant={result.success ? 'default' : 'destructive'}>
            <AlertDescription>
              <strong>{channel?.name}:</strong> {result.message}
              {result.latencyMs && ` (${result.latencyMs}ms)`}
            </AlertDescription>
          </Alert>
        )
      })}
    </div>
  )
}
