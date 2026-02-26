import { RequestLog } from './amp'

export function connectRequestLogsWS(
  onLog: (log: RequestLog) => void,
  onError?: () => void,
): () => void {
  const token = localStorage.getItem('token')
  if (!token) {
    onError?.()
    return () => {}
  }

  const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${scheme}://${window.location.host}/api/admin/request-logs/ws?token=${encodeURIComponent(token)}`

  let ws: WebSocket | null = new WebSocket(url)
  let closed = false

  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data)
      if (msg.type === 'request_log_completed' && msg.data) {
        onLog(msg.data)
      }
    } catch {
      // ignore parse errors
    }
  }

  ws.onerror = () => {
    if (!closed) onError?.()
  }

  ws.onclose = () => {
    ws = null
  }

  return () => {
    closed = true
    if (ws) {
      ws.close()
      ws = null
    }
  }
}
