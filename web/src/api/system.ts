const API_BASE = '/api'

function getAuthHeaders() {
  const token = localStorage.getItem('token')
  return {
    Authorization: `Bearer ${token}`,
  }
}

export async function uploadDatabase(file: File): Promise<{ message: string; backupFile: string }> {
  const formData = new FormData()
  formData.append('database', file)

  const res = await fetch(`${API_BASE}/admin/system/database/upload`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: formData,
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '上传失败')
  }

  return res.json()
}

export async function downloadDatabase(): Promise<void> {
  const res = await fetch(`${API_BASE}/admin/system/database/download`, {
    headers: getAuthHeaders(),
  })

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
  const res = await fetch(`${API_BASE}/admin/system/database/backups`, {
    headers: getAuthHeaders(),
  })

  if (!res.ok) {
    throw new Error('获取备份列表失败')
  }

  return res.json()
}

export async function restoreBackup(filename: string): Promise<{ message: string }> {
  const res = await fetch(`${API_BASE}/admin/system/database/restore`, {
    method: 'POST',
    headers: {
      ...getAuthHeaders(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ filename }),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '恢复失败')
  }

  return res.json()
}

export async function deleteBackup(filename: string): Promise<{ message: string }> {
  const res = await fetch(`${API_BASE}/admin/system/database/backups/${encodeURIComponent(filename)}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })

  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || '删除失败')
  }

  return res.json()
}
