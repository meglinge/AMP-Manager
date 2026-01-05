const API_BASE = '/api'

export interface AvailableModel {
  modelId: string
  displayName: string
  channelType: 'openai' | 'claude' | 'gemini'
  channelName: string
}

export interface FetchModelsResult {
  message: string
  results: Record<string, number>
}

export async function listAvailableModels(): Promise<AvailableModel[]> {
  const token = localStorage.getItem('token')
  const res = await fetch(`${API_BASE}/models`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.error || '获取模型列表失败')
  }
  const data = await res.json()
  return data.models || []
}

export async function fetchChannelModels(channelId: string): Promise<{ message: string; count: number }> {
  const token = localStorage.getItem('token')
  const res = await fetch(`${API_BASE}/admin/channels/${channelId}/fetch-models`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.error || '获取模型失败')
  }
  return res.json()
}

export async function fetchAllModels(): Promise<FetchModelsResult> {
  const token = localStorage.getItem('token')
  const res = await fetch(`${API_BASE}/admin/models/fetch-all`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.error || '获取模型失败')
  }
  return res.json()
}
