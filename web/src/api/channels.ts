const API_BASE = '/api/admin/channels'

function getAuthHeader(): HeadersInit {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '请求失败' }))
    throw new Error(error.error || '请求失败')
  }
  return response.json()
}

export type ChannelType = 'gemini' | 'claude' | 'openai'
export type ChannelEndpoint = 'chat_completions' | 'responses' | 'messages' | 'generate_content'

export interface ChannelModel {
  name: string
  alias?: string
}

export interface Channel {
  id: string
  type: ChannelType
  endpoint: ChannelEndpoint
  name: string
  baseUrl: string
  apiKeySet: boolean
  enabled: boolean
  weight: number
  priority: number
  groupIds: string[]
  groupNames: string[]
  models: ChannelModel[]
  headers: Record<string, string>
  createdAt: string
  updatedAt: string
}

export interface ChannelRequest {
  type: ChannelType
  endpoint?: ChannelEndpoint
  name: string
  baseUrl: string
  apiKey?: string
  enabled: boolean
  weight: number
  priority: number
  groupIds?: string[]
  models?: ChannelModel[]
  headers?: Record<string, string>
}

export interface TestChannelResult {
  success: boolean
  message: string
  latencyMs?: number
}

export async function listChannels(): Promise<Channel[]> {
  const response = await fetch(API_BASE, {
    headers: getAuthHeader(),
  })
  const data = await handleResponse<{ channels: Channel[] }>(response)
  return data.channels || []
}

export async function getChannel(id: string): Promise<Channel> {
  const response = await fetch(`${API_BASE}/${id}`, {
    headers: getAuthHeader(),
  })
  return handleResponse<Channel>(response)
}

export async function createChannel(data: ChannelRequest): Promise<Channel> {
  const response = await fetch(API_BASE, {
    method: 'POST',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<Channel>(response)
}

export async function updateChannel(id: string, data: ChannelRequest): Promise<Channel> {
  const response = await fetch(`${API_BASE}/${id}`, {
    method: 'PUT',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<Channel>(response)
}

export async function deleteChannel(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/${id}`, {
    method: 'DELETE',
    headers: getAuthHeader(),
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '删除失败' }))
    throw new Error(error.error || '删除失败')
  }
}

export async function setChannelEnabled(id: string, enabled: boolean): Promise<void> {
  const response = await fetch(`${API_BASE}/${id}/enabled`, {
    method: 'PATCH',
    headers: getAuthHeader(),
    body: JSON.stringify({ enabled }),
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '更新失败' }))
    throw new Error(error.error || '更新失败')
  }
}

export async function testChannel(id: string): Promise<TestChannelResult> {
  const response = await fetch(`${API_BASE}/${id}/test`, {
    method: 'POST',
    headers: getAuthHeader(),
  })
  return handleResponse<TestChannelResult>(response)
}
