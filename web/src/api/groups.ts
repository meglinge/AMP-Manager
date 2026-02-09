const API_BASE = '/api/admin/groups'

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

export interface Group {
  id: string
  name: string
  description: string
  rateMultiplier: number
  userCount: number
  channelCount: number
  createdAt: string
  updatedAt: string
}

export interface GroupRequest {
  name: string
  description: string
  rateMultiplier: number
}

export async function listGroups(): Promise<Group[]> {
  const response = await fetch(API_BASE, { headers: getAuthHeader() })
  const data = await handleResponse<{ groups: Group[] }>(response)
  return data.groups || []
}

export async function createGroup(data: GroupRequest): Promise<Group> {
  const response = await fetch(API_BASE, {
    method: 'POST',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<Group>(response)
}

export async function updateGroup(id: string, data: GroupRequest): Promise<Group> {
  const response = await fetch(`${API_BASE}/${id}`, {
    method: 'PUT',
    headers: getAuthHeader(),
    body: JSON.stringify(data),
  })
  return handleResponse<Group>(response)
}

export async function deleteGroup(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/${id}`, {
    method: 'DELETE',
    headers: getAuthHeader(),
  })
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: '删除失败' }))
    throw new Error(error.error || '删除失败')
  }
}
