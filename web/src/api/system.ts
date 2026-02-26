import { authFetch } from './client'

const API_BASE = '/api'

export async function uploadDatabase(file: File): Promise<{ message: string; backupFile: string }> {
  const formData = new FormData()
  formData.append('database', file)

  const res = await authFetch(`${API_BASE}/admin/system/database/upload`, {
    method: 'POST',
    body: formData,
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '上传失败')
  }

  return res.json()
}

export async function downloadDatabase(): Promise<void> {
  const res = await authFetch(`${API_BASE}/admin/system/database/download`)

  if (!res.ok) {
    throw new Error('下载失败')
  }

  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `ampmanager_${new Date().toISOString().slice(0, 10)}.db`
  document.body.appendChild(a)
  a.click()
  window.URL.revokeObjectURL(url)
  document.body.removeChild(a)
}

export interface Backup {
  filename: string
  size: number
  modTime: string
}

export async function listBackups(): Promise<Backup[]> {
  const res = await authFetch(`${API_BASE}/admin/system/database/backups`)

  if (!res.ok) {
    throw new Error('获取备份列表失败')
  }

  return res.json()
}

export async function restoreBackup(filename: string): Promise<{ message: string }> {
  const res = await authFetch(`${API_BASE}/admin/system/database/restore`, {
    method: 'POST',
    body: JSON.stringify({ filename }),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '恢复失败')
  }

  return res.json()
}

export async function deleteBackup(filename: string): Promise<{ message: string }> {
  const res = await authFetch(`${API_BASE}/admin/system/database/backups/${encodeURIComponent(filename)}`, {
    method: 'DELETE',
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '删除失败')
  }

  return res.json()
}

// 重试配置接口
export interface RetryConfig {
  enabled: boolean
  maxAttempts: number
  gateTimeoutMs: number
  maxBodyBytes: number
  backoffBaseMs: number
  backoffMaxMs: number
  retryOn429: boolean
  retryOn5xx: boolean
  respectRetryAfter: boolean
}

// 获取重试配置
export async function getRetryConfig(): Promise<RetryConfig> {
  const res = await authFetch(`${API_BASE}/admin/system/retry-config`)

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '获取配置失败')
  }

  return res.json()
}

// 更新重试配置
export async function updateRetryConfig(config: RetryConfig): Promise<{ message: string; config: RetryConfig }> {
  const res = await authFetch(`${API_BASE}/admin/system/retry-config`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新配置失败')
  }

  return res.json()
}

// 获取请求详情监控状态
export async function getRequestDetailEnabled(): Promise<{ enabled: boolean }> {
  const res = await authFetch(`${API_BASE}/admin/system/request-detail-enabled`)

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '获取配置失败')
  }

  return res.json()
}

// 更新请求详情监控状态
export async function updateRequestDetailEnabled(enabled: boolean): Promise<{ message: string; enabled: boolean }> {
  const res = await authFetch(`${API_BASE}/admin/system/request-detail-enabled`, {
    method: 'PUT',
    body: JSON.stringify({ enabled }),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新配置失败')
  }

  return res.json()
}

// 超时配置接口
export interface TimeoutConfig {
  idleConnTimeoutSec: number
  readIdleTimeoutSec: number
  keepAliveIntervalSec: number
  dialTimeoutSec: number
  tlsHandshakeTimeoutSec: number
}

// 获取超时配置
export async function getTimeoutConfig(): Promise<TimeoutConfig> {
  const res = await authFetch(`${API_BASE}/admin/system/timeout-config`)

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '获取配置失败')
  }

  return res.json()
}

// 更新超时配置
export async function updateTimeoutConfig(config: TimeoutConfig): Promise<{ message: string; config: TimeoutConfig }> {
  const res = await authFetch(`${API_BASE}/admin/system/timeout-config`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新配置失败')
  }

  return res.json()
}

// 缓存 TTL 配置
export async function getCacheTTLConfig(): Promise<{ cacheTTL: string }> {
  const res = await authFetch(`${API_BASE}/admin/system/cache-ttl`)

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '获取配置失败')
  }

  return res.json()
}

export async function updateCacheTTLConfig(cacheTTL: string): Promise<{ message: string; cacheTTL: string }> {
  const res = await authFetch(`${API_BASE}/admin/system/cache-ttl`, {
    method: 'PUT',
    body: JSON.stringify({ cacheTTL }),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '更新配置失败')
  }

  return res.json()
}
